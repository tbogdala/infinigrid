// Copyright 2017, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package main

import (
	"fmt"
	"math/rand"

	mgl "github.com/go-gl/mathgl/mgl32"

	fizzle "github.com/tbogdala/fizzle"
	component "github.com/tbogdala/fizzle/component"
	forward "github.com/tbogdala/fizzle/renderer/forward"
	scene "github.com/tbogdala/fizzle/scene"
	"github.com/tbogdala/glider"
)

const (
	playerEntityName     = "Player"
	playerShipEntityName = "PlayerShip"

	floorSizeWidth = 20.0
)

const (
	gameStatePlaying    = 1
	gameStatePlayerDied = 2
)

// GameScene is the main game scene that plays the current level.
type GameScene struct {
	// embed the basic scene manager
	*scene.BasicSceneManager

	// playerEntity is the cached reference to the player entity. will be added to
	// BasicSceneManager side as well.
	playerEntity *VisibleEntity

	// shipEntity is the cached reference to the ship pawn.
	shipEntity *ShipEntity

	components        *component.Manager
	textureMan        *fizzle.TextureManager
	shaders           map[string]*fizzle.RenderShader
	currentFrameDelta float32

	currentGameTime   float64
	lastGridSpawn     float64
	lastBombSpawn     float64
	distanceTravelled float64

	spawnIntervalSec float64
	maxToSpawn       int

	gameState int
}

// ScrollableEntity is an entity that scrolls past the player while the game plays.
// This should include things like wall sets and bombs.
type ScrollableEntity interface {
	// ScrollPastPlayer should move the entity with relation to the inverse
	// of the player speed, adjusted for frame delta.
	ScrollPastPlayer(backwardSpeed mgl.Vec3, frameDelta float32)
}

// CollisionEntity should be implemented for all entities that can collide with other objects.
type CollisionEntity interface {
	// GetColliders should return all of the coarse colliders for an entity.
	GetColliders() []glider.Collider
}

// NewGameScene creates a new game scene object
func NewGameScene() *GameScene {
	gs := new(GameScene)
	gs.BasicSceneManager = scene.NewBasicSceneManager()
	gs.shaders = make(map[string]*fizzle.RenderShader)

	// starting difficulty
	gs.spawnIntervalSec = 2.0
	gs.maxToSpawn = 12

	return gs
}

// Update should be called each frame to update the scene manager.
func (s *GameScene) Update(frameDelta float32) {
	// store the framedelta value before running the system updates.
	// this will allow for callback from the input system to see the
	// current frame delta.
	s.currentFrameDelta = frameDelta
	s.currentGameTime += float64(frameDelta)

	// call the base version which will update the systems
	s.BasicSceneManager.Update(frameDelta)

	// if the player is dead we don't do anything on update
	if s.gameState == gameStatePlayerDied {
		return
	}

	// ======================================================================
	// test to see if we need to spawn walls
	s.SpawnNewWalls()

	// ======================================================================
	// test to see if we need to spawn some bombs
	s.SpawnNewBombs()

	// ======================================================================
	// HACK: check colliders vs ship to see if we have a hit
	collisionFound := false
	s.BasicSceneManager.MapEntities(func(id uint64, e scene.Entity) {
		// skip the ship and the player entities
		if id == s.shipEntity.ID || id == s.playerEntity.ID {
			return
		}

		collisionEntity, okay := e.(CollisionEntity)
		if okay {
			for _, colObject := range collisionEntity.GetColliders() {
				for _, shipColObject := range s.shipEntity.GetColliders() {
					if glider.Collide(colObject, shipColObject) != glider.NoIntersect {
						collisionFound = true
						break
					}
				}
				if collisionFound {
					break
				}
			}
		}
	})

	// if the player hits another entity it's considered the end of the road!
	if collisionFound && s.gameState != gameStatePlayerDied {
		s.gameState = gameStatePlayerDied

		system := s.BasicSceneManager.GetSystemByName(uiSystemName)
		uisys := system.(*UISystem)
		uisys.SetVisible(true)
		uisys.ShowQuitMenu()
	}

	// calculate the distance the ship has travelled so far
	dist := float64(s.shipEntity.currentShipSpeed.Mul(s.currentFrameDelta)[2])
	s.lastGridSpawn += dist
	s.distanceTravelled += dist

	// ======================================================================
	// go through all entities and update positions of everything
	// that's not the player
	toRemove := []scene.Entity{}
	backwardSpeed := s.shipEntity.currentShipSpeed.Mul(-s.currentFrameDelta)
	s.BasicSceneManager.MapEntities(func(id uint64, e scene.Entity) {
		// skip the ship and the player entities
		if id == s.shipEntity.ID || id == s.playerEntity.ID {
			return
		}

		// see if it implements the ScrollableEntity interface
		scrollableEntity, scrollable := e.(ScrollableEntity)
		if scrollable {
			scrollableEntity.ScrollPastPlayer(backwardSpeed, frameDelta)

			// if it's far away, list it for removal
			if e.GetLocation()[2] < -200.0 {
				fmt.Printf("DEBUG removing entity: %s at %v\n", e.GetName(), e.GetLocation())
				toRemove = append(toRemove, e)
			}
		}

	})
	for _, e := range toRemove {
		s.RemoveEntity(e)
	}
}

// SetupScene initializes the scene's assets and sets up the initial entities.
// NOTE: A render System implementation will need to be added before this
// method is called.
func (s *GameScene) SetupScene() error {
	// pull a reference to the render system
	var renderSystem RenderSystem
	system := s.BasicSceneManager.GetSystemByName(vrRenderSystemName)
	if system == nil {
		system = s.BasicSceneManager.GetSystemByName(forwardRenderSystemName)
		if system == nil {
			return fmt.Errorf("Need to add a render System implementation first")
		}
		forwardRenderSystem := system.(*ForwardRenderSystem)
		renderSystem = forwardRenderSystem
	} else {
		vrRenderSystem := system.(*VRRenderSystem)
		renderSystem = vrRenderSystem
	}

	// load the shaders necessary
	err := s.createShaders()
	if err != nil {
		return err
	}

	// setup the texture manager for use in the component manager
	s.textureMan = fizzle.NewTextureManager()

	// create the component manager
	s.components = component.NewManager(s.textureMan, s.shaders)

	// TODO: don't hardcode the component references here
	_, err = s.components.LoadComponentFromFile("assets/components/grid_ship.json", "entity/ship")
	if err != nil {
		return fmt.Errorf("failed to load the entity/ship component: %v", err)
	}
	_, err = s.components.LoadComponentFromFile("assets/components/grid_bomb.json", "entity/bomb")
	if err != nil {
		return fmt.Errorf("failed to load the entity/bomb component: %v", err)
	}
	_, err = s.components.LoadComponentFromFile("assets/components/level_prototype.json", "grid/proto")
	if err != nil {
		return fmt.Errorf("failed to load the grid/proto component: %v", err)
	}

	// put a light in there
	renderer := renderSystem.GetRenderer()
	light := renderer.NewDirectionalLight(mgl.Vec3{1.0, -0.5, -1.0})
	light.DiffuseIntensity = 0.20
	light.SpecularIntensity = 0.10
	light.AmbientIntensity = 1.0
	renderer.ActiveLights[0] = light

	// create the grid
	gridProtoComponent, _ := s.components.GetComponent("grid/proto")
	var gridProtoRenderable *fizzle.Renderable
	var gridProtoEntity *WallSetEntity
	for z := float32(12.5); z <= 312.5; z += 25.0 {
		gridProtoRenderable = s.components.GetRenderableInstance(gridProtoComponent)
		gridProtoEntity = NewWallSetEntity()
		gridProtoEntity.CreateCollidersFromComponent(gridProtoComponent)
		gridProtoEntity.ID = s.GetNextID()
		gridProtoEntity.Name = fmt.Sprintf("GridProto_pre%d", int(z))
		gridProtoEntity.Renderable = gridProtoRenderable
		gridProtoEntity.SetLocation(mgl.Vec3{0, 0, z})
		s.AddEntity(gridProtoEntity)
	}

	// add the ship in
	shipComponent, _ := s.components.GetComponent("entity/ship")
	shipRenderable := s.components.GetRenderableInstance(shipComponent)
	s.shipEntity = NewShipEntity()
	s.shipEntity.CreateCollidersFromComponent(shipComponent)
	s.shipEntity.ID = s.GetNextID()
	s.shipEntity.Renderable = shipRenderable
	s.shipEntity.SetLocation(mgl.Vec3{0.0, 2.0, 0.0})
	s.shipEntity.Name = playerShipEntityName
	s.AddEntity(s.shipEntity)
	s.shipEntity.currentShipSpeed = mgl.Vec3{0.0, 0.0, 25.0}

	// create the player entity
	// FIXME: Is this really a visible entity??
	s.playerEntity = NewVisibleEntity()
	s.playerEntity.ID = s.GetNextID()
	s.playerEntity.Name = playerEntityName
	s.playerEntity.SetLocation(s.shipEntity.GetLocation().Add(mgl.Vec3{0.0, 0.2, -0.25}))
	s.AddEntity(s.playerEntity)

	// set the state to playing
	s.gameState = gameStatePlaying

	return nil
}

// createShaders will load the shaders necessary for the game scene.
func (s *GameScene) createShaders() error {
	// load the diffuse shader for the cube
	var err error
	basicShader, err := forward.CreateBasicShader()
	if err != nil {
		return fmt.Errorf("Failed to compile and link the diffuse shader program!\n%v", err)
	}
	s.shaders["Basic"] = basicShader

	return nil
}

// SpawnNewWalls will spawn new walls for the player to fly around if
// the time is right.
func (s *GameScene) SpawnNewWalls() {
	const gridSegmentLength = 25.0
	const spawnDistance = 300.0 + (gridSegmentLength / 2.0)

	if s.lastGridSpawn > gridSegmentLength {
		gridProtoComponent, _ := s.components.GetComponent("grid/proto")
		gridProtoRenderable := s.components.GetRenderableInstance(gridProtoComponent)
		gridProtoEntity := NewWallSetEntity()
		gridProtoEntity.CreateCollidersFromComponent(gridProtoComponent)
		gridProtoEntity.ID = s.GetNextID()
		gridProtoEntity.Name = fmt.Sprintf("GridProto_%d", int(s.distanceTravelled))
		gridProtoEntity.Renderable = gridProtoRenderable
		gridProtoEntity.SetLocation(mgl.Vec3{0, 0, spawnDistance})
		s.AddEntity(gridProtoEntity)

		//fmt.Printf("Created grid proto: %s\n", gridProtoEntity.Name)
		s.lastGridSpawn = 0.0
	}
}

// SpawnNewBombs will spawn new bombs for the player to avoid.
func (s *GameScene) SpawnNewBombs() {
	const spawnDistance = 250.0
	const minToSpawn = 1
	const minX = -10
	const maxX = 10
	const minY = 1
	const maxY = 14
	const minZDelta = -10
	const maxZDelta = 10

	// return now if we haven't hit the spawn timer
	if s.currentGameTime-s.lastBombSpawn <= s.spawnIntervalSec {
		return
	}

	spawnCount := rand.Intn(s.maxToSpawn-minToSpawn) + minToSpawn

	// spawn new bombs
	bombComponent, _ := s.components.GetComponent("entity/bomb")
	for i := 0; i < spawnCount; i++ {
		bombRenderable := s.components.GetRenderableInstance(bombComponent)
		bombEntity := NewBombEntity()
		bombEntity.CreateCollidersFromComponent(bombComponent)
		bombEntity.ID = s.GetNextID()
		bombEntity.Name = fmt.Sprintf("Bomb_%d_%d", i, int(s.distanceTravelled))
		bombEntity.Renderable = bombRenderable

		x := rand.Intn(maxX-minX) + minX
		y := rand.Intn(maxY-minY) + minY
		z := rand.Intn(maxZDelta-minZDelta) + minZDelta

		bombEntity.SetLocation(mgl.Vec3{float32(x), float32(y), float32(spawnDistance + z)})
		bombEntity.SetMaxSpeed()
		s.AddEntity(bombEntity)
	}
	// reset the timer
	s.lastBombSpawn = s.currentGameTime
}
