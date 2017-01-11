// Copyright 2016, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package main

import (
	"fmt"
	"math"

	mgl "github.com/go-gl/mathgl/mgl32"

	fizzle "github.com/tbogdala/fizzle"
	component "github.com/tbogdala/fizzle/component"
	forward "github.com/tbogdala/fizzle/renderer/forward"
	scene "github.com/tbogdala/fizzle/scene"
	vr "github.com/tbogdala/openvr-go"
)

const (
	playerEntityName = "Player"

	maxRollRads  = math.Pi / 4.0 // 45 deg
	maxPitchRads = math.Pi / 8.0 // 22.5 deg

	floorSizeLength = 300.0
	floorSizeWidth  = 10.0
)

// GameScene is the main game scene that plays the current level.
type GameScene struct {
	// embed the basic scene manager
	*scene.BasicSceneManager

	// playerEntity is the cached reference to the player entity. will be added to
	// BasicSceneManager side as well.
	playerEntity *VisibleEntity

	// shipEntity is the cached reference to the ship pawn.
	shipEntity *VisibleEntity

	components         *component.Manager
	textureMan         *fizzle.TextureManager
	shaders            map[string]*fizzle.RenderShader
	cachedRenderSystem *VRRenderSystem
	currentFrameDelta  float32

	// currentShipRoll is the current Roll rotation for the ship in radians.
	currentShipRoll float32

	// currentShipPitch is the current Pitch rotation for the ship in radians.
	currentShipPitch float32

	currentShipSpeed mgl.Vec3 // m/s

	timeSinceLastSpawn float32
}

// NewGameScene creates a new game scene object
func NewGameScene() *GameScene {
	gs := new(GameScene)
	gs.BasicSceneManager = scene.NewBasicSceneManager()
	gs.shaders = make(map[string]*fizzle.RenderShader)
	return gs
}

// Update should be called each frame to update the scene manager.
func (s *GameScene) Update(frameDelta float32) {
	// store the framedelta value before running the system updates.
	// this will allow for callback from the input system to see the
	// current frame delta.
	s.currentFrameDelta = frameDelta

	// call the base version which will update the systems
	s.BasicSceneManager.Update(frameDelta)

	// HACK: spawn walls here
	s.SpawnNewWalls()

	// HACK: and rotate the ship
	qRoll := mgl.QuatRotate(s.currentShipRoll, mgl.Vec3{0.0, 0.0, 1.0})
	qPitch := mgl.QuatRotate(s.currentShipPitch, mgl.Vec3{1.0, 0.0, 0.0})
	s.shipEntity.SetOrientation(qRoll.Mul(qPitch))

	// HACK: move the ship in the world x/y axis at a speed determined
	// by the proportion of current roll/pitch to the maximum values.
	rollRatio := s.currentShipRoll / maxRollRads
	pitchRatio := s.currentShipPitch / maxPitchRads
	const moveSpeed = 1.0 // 1 m/s
	shipLoc := s.shipEntity.GetLocation()
	shipLoc[0] -= moveSpeed * rollRatio * frameDelta
	shipLoc[0] = mgl.Clamp(shipLoc[0], -floorSizeWidth/2.0, floorSizeWidth/2.0)
	shipLoc[1] -= moveSpeed * pitchRatio * frameDelta
	shipLoc[1] = mgl.Clamp(shipLoc[1], 0.1, 2.0)
	s.shipEntity.SetLocation(shipLoc)

	// HACK: go through all entities and update positions of everything
	// that's not the player
	wallsToRemove := []scene.Entity{}
	backwardSpeed := s.currentShipSpeed.Mul(-s.currentFrameDelta)
	s.BasicSceneManager.MapEntities(func(id uint64, e scene.Entity) {
		// skip the ship and the player entities
		if id == s.shipEntity.ID || id == s.playerEntity.ID {
			return
		}

		// move the floor grid in a special way. we only move it a fraction of a meter.
		// once it's deviated more than a meter away we recenter it on world origin so that
		// we never run out of grid plane.
		if e.GetName() == "GridFloor" {
			loc := e.GetLocation().Add(backwardSpeed)
			// only handles movement on z axis right now
			if loc[2] < -1.0 {
				loc[2] += 1.0
			}
			e.SetLocation(loc)
			return
		}

		// move everything else back the current speed of the ship
		loc := e.GetLocation().Add(backwardSpeed)
		e.SetLocation(loc)

		// HACK: bad test
		if loc[2] < -200.0 {
			wallsToRemove = append(wallsToRemove, e)
		}
	})

	for _, toRemove := range wallsToRemove {
		s.RemoveEntity(toRemove)
	}
}

// SetupScene initializes the scene's assets and sets up the initial entities.
// NOTE: A render System implementation will need to be added before this
// method is called.
func (s *GameScene) SetupScene() error {
	// pull a reference to the render system
	system := s.BasicSceneManager.GetSystemByName(vrRenderSystemName)
	if system == nil {
		return fmt.Errorf("Need to add a render System implementation first")
	}
	renderSystem := system.(*VRRenderSystem)
	s.cachedRenderSystem = renderSystem

	// load the shaders necessary
	err := s.createShaders()
	if err != nil {
		return err
	}

	// load some textures
	s.textureMan = fizzle.NewTextureManager()
	texturePath := "assets/textures/gridpattern.png"
	gridPatternTex, err := s.textureMan.LoadTexture("gridpattern", texturePath)
	if err != nil {
		return fmt.Errorf("Failed to load the grid pattern texture at %s!\n%v", texturePath, err)
	}
	fizzle.GenerateMipmaps(gridPatternTex)

	// create the component manager
	s.components = component.NewManager(s.textureMan, s.shaders)

	// TODO: don't hardcode the component references here
	s.components.LoadComponentFromFile("assets/components/ship.json", "entity/ship")
	s.components.LoadComponentFromFile("assets/components/wall_8mx4m.json", "geom/wall_8mx4m")

	// put a light in there
	light := renderSystem.Renderer.NewDirectionalLight(mgl.Vec3{1.0, -0.5, -1.0})
	light.DiffuseIntensity = 0.20
	light.SpecularIntensity = 0.10
	light.AmbientIntensity = 1.0
	renderSystem.Renderer.ActiveLights[0] = light

	// create the 'infinite' grid plane
	gridFloor := fizzle.CreatePlaneXZ(-floorSizeWidth/2.0, floorSizeLength/2.0, floorSizeWidth/2.0, -floorSizeLength/2.0)
	gridFloor.Material = fizzle.NewMaterial()
	gridFloor.Material.Shader = s.shaders["GridFloor"]
	gridFloor.Material.DiffuseColor = mgl.Vec4{float32(0x66) / 255.0, float32(0xA8) / 255.0, 0.00, 1.0}
	gridFloor.Material.Shininess = 0.0
	gridFloor.Material.CustomTex[0] = gridPatternTex
	gridFloorEntity := NewVisibleEntity()
	gridFloorEntity.ID = s.GetNextID()
	gridFloorEntity.Name = "GridFloor"
	gridFloorEntity.Renderable = gridFloor
	s.AddEntity(gridFloorEntity)

	// FIXME: quick test to make sure I can add the ship in
	shipComponent, _ := s.components.GetComponent("entity/ship")
	shipRenderable := s.components.GetRenderableInstance(shipComponent)
	s.shipEntity = NewVisibleEntity()
	s.shipEntity.ID = s.GetNextID()
	s.shipEntity.Renderable = shipRenderable
	s.shipEntity.SetLocation(mgl.Vec3{0.0, 1.0, 0.0})
	s.AddEntity(s.shipEntity)
	s.currentShipSpeed = mgl.Vec3{0.0, 0.0, 25.0}

	// create the player entity
	s.playerEntity = NewVisibleEntity()
	s.playerEntity.ID = s.GetNextID()
	s.playerEntity.Name = playerEntityName
	s.playerEntity.SetLocation(s.shipEntity.GetLocation().Add(mgl.Vec3{0.0, 0.2, -0.25}))
	//s.playerEntity.SetLocation(mgl.Vec3{0.0, 1.0, 0.0})
	s.AddEntity(s.playerEntity)

	return nil
}

// createShaders will load the shaders necessary for the game scene.
func (s *GameScene) createShaders() error {
	gridFloorShaderV := `#version 330
      precision highp float;

      uniform mat4 MVP_MATRIX;

      in vec3 VERTEX_POSITION;
      in vec2 VERTEX_UV_0;

      out vec3 vs_pos;
      out vec2 vs_tex0_uv;

      void main(void) {
        vs_pos = VERTEX_POSITION;
        vs_tex0_uv = VERTEX_UV_0;
        gl_Position = MVP_MATRIX * vec4(VERTEX_POSITION, 1.0);
      }
      `

	gridFloorShaderF := `#version 330
      precision highp float;

      uniform sampler2D MATERIAL_TEX_0;
      uniform vec4 MATERIAL_DIFFUSE;

      in vec3 vs_pos;
      in vec2 vs_tex0_uv;

      out vec4 frag_color;

      void main (void) {
        vec2 uv = vec2(fract(vs_pos.x), fract(vs_pos.z));
        vec4 mask = texture(MATERIAL_TEX_0, uv);
        if (mask.a < 0.1) {
          discard;
        }
        frag_color.rgb = MATERIAL_DIFFUSE.rgb * mask.rgb;
        frag_color.a = mask.a;
      }
      `

	// load the diffuse shader for the cube
	var err error
	basicShader, err := forward.CreateBasicShader()
	if err != nil {
		return fmt.Errorf("Failed to compile and link the diffuse shader program!\n%v", err)
	}
	s.shaders["Basic"] = basicShader

	// load the shader used to render the framebuffers to a window for viewing
	gridFloorShader, err := fizzle.LoadShaderProgram(gridFloorShaderV, gridFloorShaderF, nil)
	if err != nil {
		return fmt.Errorf("Failed to compile and link the grid floor shader program!\n%v", err)
	}
	s.shaders["GridFloor"] = gridFloorShader

	return nil
}

// HandleHeadAutoLevel should be called to set the auto-level 'head' position. This allows
// for the HMD to move around and affect the camera, but be centered appropriately for a
// sitting position.
func (s *GameScene) HandleHeadAutoLevel() {
	// update the playerEntity's height to account for a new calibration of
	// the HMD head position
	hmdLoc := s.cachedRenderSystem.GetHMDLocation()
	s.playerEntity.SetLocation(s.shipEntity.GetLocation().Add(mgl.Vec3{
		0.0 - hmdLoc[0],
		0.2 - hmdLoc[1],
		-0.25 - hmdLoc[2]}))
}

// HandleAxisLUpdate is a DEBUG / TEST input callback
// FIXME: rename if finalized
func (s *GameScene) HandleAxisLUpdate(axisData [vr.ControllerStateAxisCount]vr.ControllerAxis) {
	// Ship roll / pitch will not be a direct value from the trackpad [0..1].
	// Instead the trackpad value will scale a speed variable and accumulate
	// towards a maximum value. If no input on the corresponding axis is
	// detected, then the accumulator should decay at a separate speed towards 0.

	const rollAccumulatorFactor = 0.2  // sec until max roll
	const rollDecayFactor = 0.4        // sec until roll decays to 0 from max
	const pitchAccumulatorFactor = 0.1 // sec until max pitch
	const pitchDecayFactor = 0.2       // sec until pitch decays to 0 from max

	var accDelta float32
	var decayDelta float32

	// Check to see if there's any movement on the X axis to determine roll.
	rollInput := axisData[0].X
	if rollInput != 0.0 {
		// we are in a roll so accumulate a value. this will automatically
		// account for direction based on the sign of the axis value.
		accDelta = rollInput * (s.currentFrameDelta / rollAccumulatorFactor) * maxRollRads
		s.currentShipRoll += accDelta
		s.currentShipRoll = mgl.Clamp(s.currentShipRoll, -maxRollRads, maxRollRads)
	} else {
		// are we currently in a roll with no further input from user
		if s.currentShipRoll != 0.0 {
			// decay the roll accumulator down to 0.0, but this can be from
			// a negative rotation or a positive rotation.
			decayDelta = (s.currentFrameDelta / rollDecayFactor) * maxRollRads
			if s.currentShipRoll > 0.0 {
				s.currentShipRoll -= decayDelta
				s.currentShipRoll = mgl.Clamp(s.currentShipRoll, 0.0, s.currentShipRoll)
			} else if s.currentShipRoll < 0.0 {
				s.currentShipRoll += decayDelta
				s.currentShipRoll = mgl.Clamp(s.currentShipRoll, s.currentShipRoll, 0.0)
			}
		}
	}

	pitchInput := axisData[0].Y
	if pitchInput != 0.0 {
		// we are in a roll so accumulate a value. this will automatically
		// account for direction based on the sign of the axis value.
		accDelta = pitchInput * (s.currentFrameDelta / pitchAccumulatorFactor) * maxPitchRads
		s.currentShipPitch += accDelta
		s.currentShipPitch = mgl.Clamp(s.currentShipPitch, -maxPitchRads, maxPitchRads)
	} else {
		// are we currently in a roll with no further input from user
		if s.currentShipPitch != 0.0 {
			// decay the roll accumulator down to 0.0, but this can be from
			// a negative rotation or a positive rotation.
			decayDelta = (s.currentFrameDelta / pitchDecayFactor) * maxPitchRads
			if s.currentShipPitch > 0.0 {
				s.currentShipPitch -= decayDelta
				s.currentShipPitch = mgl.Clamp(s.currentShipPitch, 0.0, s.currentShipPitch)
			} else if s.currentShipPitch < 0.0 {
				s.currentShipPitch += decayDelta
				s.currentShipPitch = mgl.Clamp(s.currentShipPitch, s.currentShipPitch, 0.0)
			}
		}
	}

	// DEBUG **
	// check to see if Axis0 (touchpad) has a non-zero X or Y
	//if axisData[0].X != 0.0 || axisData[0].Y != 0.0 {
	//fmt.Printf("LTouchpad X:%f Y:%f Acum:%f Accel:%f Decay:%f\n",
	//	axisData[0].X, axisData[0].Y, s.currentShipPitch, accDelta, decayDelta)
	//}
}

// SpawnNewWalls will spawn new walls for the player to fly around if
// the time is right.
func (s *GameScene) SpawnNewWalls() {
	const spawnTime = 4.0 // a new wall every x seconds

	// update our timer for spawning walls
	s.timeSinceLastSpawn += s.currentFrameDelta
	if s.timeSinceLastSpawn < spawnTime {
		return
	}

	const spawnDistance = 100.0 // spawn x meters away

	// get the wall component to spawn
	wallComponent, _ := s.components.GetComponent("geom/wall_8mx4m")
	wallRenderable := s.components.GetRenderableInstance(wallComponent)
	wallEntity := NewVisibleEntity()
	wallEntity.ID = s.GetNextID()
	wallEntity.Name = "Wall_8mx4m"
	wallEntity.Renderable = wallRenderable
	wallEntity.SetLocation(mgl.Vec3{0.0, 0.0, 1.0 * spawnDistance})
	s.AddEntity(wallEntity)

	s.timeSinceLastSpawn = 0.0

	// DEBUG
	fmt.Printf("Spawned a wall at %f\n", wallEntity.GetLocation())
}
