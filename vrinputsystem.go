// Copyright 2017, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package main

import (
	"fmt"

	mgl "github.com/go-gl/mathgl/mgl32"
	scene "github.com/tbogdala/fizzle/scene"
	vr "github.com/tbogdala/openvr-go"
)

/* Notes on controller state:

Axis mapping
============
Axis 0 = Trackpad {X,Y} where vector <0,1> is up and <1,0> is right
Axis 1 = Trigger {X} where trigger is 0..1

*/

const (
	vrInputSystemPriority = -100.0
	vrInputSystemName     = "VRInputSystem"
)

// VRInputSystem implements the fizzle/scene/System interface and handles the
// player input.
//
// NOTE: for purposes of this API, the left controller is the controller that
// is detected first and is probably the controller that was first powered on.
type VRInputSystem struct {
	// OnAppMenuButtonL is a function that will get called when the application
	// menu button on the left controller gets pressed.
	OnAppMenuButtonL func()

	// OnAppMenuButtonR is a function that will get called when the application
	// menu button on the right controller gets pressed.
	OnAppMenuButtonR func()

	// OnGripButtonL is a function that will get called win the grip side button
	// on the left controller gets pressed.
	OnGripButtonL func()

	// OnGripButtonR is a function that will get called win the grip side button
	// on the right controller gets pressed.
	OnGripButtonR func()

	// OnControllerAxisUpdateL is a function that will get called each frame and
	// has a slice of ControllerAxis data passed to it for the left controller
	OnControllerAxisUpdateL func([vr.ControllerStateAxisCount]vr.ControllerAxis)

	// OnControllerAxisUpdateR is a function that will get called each frame and
	// has a slice of ControllerAxis data passed to it for the right controller
	OnControllerAxisUpdateR func([vr.ControllerStateAxisCount]vr.ControllerAxis)

	// vrSystem is the IVRSystem interface for OpenVR set on Initialize().
	vrSystem *vr.System

	// vrCompositor is the IVRCompositor interface for OpenVR set on Initialize().
	vrCompositor *vr.Compositor

	// vrRenderSystem is the cached vrrender system object for the game
	vrRenderSystem *VRRenderSystem

	// playerEntity is the cached reference to the player entity.
	playerEntity *VisibleEntity

	// playerShipEntity is the cached reference to the player ship pawn.
	playerShipEntity *ShipEntity
}

// NewVRInputSystem creates a new InputSystem object
func NewVRInputSystem() *VRInputSystem {
	system := new(VRInputSystem)
	return system
}

// Initialize sets up the input models for the scene.
func (s *VRInputSystem) Initialize(vrRenderSystem *VRRenderSystem) {
	s.vrRenderSystem = vrRenderSystem
	s.vrSystem = s.vrRenderSystem.GetVRSystem()
	s.vrCompositor = s.vrRenderSystem.GetVRCompositor()
}

// Update should get called to run updates for the system every frame
// by the owning Manager object.
func (s *VRInputSystem) Update(frameDelta float32) {
	var controllerState vr.ControllerState

	var foundLeft bool
	// find the first controller connected and check its buttons
	for i := vr.TrackedDeviceIndexHmd + 1; i < vr.MaxTrackedDeviceCount; i++ {
		deviceClass := s.vrSystem.GetTrackedDeviceClass(int(i))
		if deviceClass != vr.TrackedDeviceClassController {
			continue
		}

		// we don't track controllers that are powered off
		if !s.vrSystem.IsTrackedDeviceConnected(uint32(i)) {
			continue
		}

		// get the controller button state
		s.vrSystem.GetControllerState(int(i), &controllerState)

		// do we have any buttons down?
		if controllerState.ButtonPressed != 0 {
			// check for an app menu button press
			const menuMask uint64 = 1 << vr.ButtonApplicationMenu
			if menuMask&controllerState.ButtonPressed > 0 {
				if foundLeft && s.OnAppMenuButtonR != nil {
					s.OnAppMenuButtonR()
				} else if s.OnAppMenuButtonL != nil {
					s.OnAppMenuButtonL()
				}
			}

			// check for the grip button press
			const gripMask uint64 = 1 << vr.ButtonGrip
			if gripMask&controllerState.ButtonPressed > 0 {
				if foundLeft && s.OnGripButtonR != nil {
					s.OnGripButtonR()
				} else if s.OnGripButtonL != nil {
					s.OnGripButtonL()
				}
			}
		}

		// are we sending axis data out to a callback event?
		if foundLeft && s.OnControllerAxisUpdateR != nil {
			s.OnControllerAxisUpdateR(controllerState.Axis)
		} else if s.OnControllerAxisUpdateL != nil {
			s.OnControllerAxisUpdateL(controllerState.Axis)
		}

		// switch processing to the 'right' controller
		if foundLeft == true {
			break
		}
		foundLeft = true
	}

	// if the game state is in the player died state do not move the player
	if gameScene.gameState == gameStatePlayerDied {
		return
	}

	// after updating the controller state, adjust the player position
	// based on the input.
	s.movePlayer(frameDelta)
}

// movePlayer moves the ship in the world x/y axis at a speed determined
// by the proportion of current roll/pitch to the maximum values.
func (s *VRInputSystem) movePlayer(frameDelta float32) {
	var orientation mgl.Vec3
	for i := vr.TrackedDeviceIndexHmd + 1; i < vr.MaxTrackedDeviceCount; i++ {
		deviceClass := s.vrSystem.GetTrackedDeviceClass(int(i))
		if deviceClass != vr.TrackedDeviceClassController {
			continue
		}

		// we don't track controllers that are powered off
		if !s.vrSystem.IsTrackedDeviceConnected(uint32(i)) {
			continue
		}

		controllerPose := s.vrCompositor.GetRenderPose(i)
		forward := mgl.Vec4{0.0, 0.0, -1.0, 0.0}
		orientation = controllerPose.DeviceToAbsoluteTracking.Mul4x1(forward) //vec3 return
		break
	}
	rollInput := orientation[0] // axisData[0].X
	s.playerShipEntity.currentShipRoll = -rollInput * maxRollRads
	pitchInput := orientation[2] // axisData[0].Y
	s.playerShipEntity.currentShipPitch = pitchInput * maxPitchRads

	// rotate the ship
	qRoll := mgl.QuatRotate(s.playerShipEntity.currentShipRoll, mgl.Vec3{0.0, 0.0, 1.0})
	qPitch := mgl.QuatRotate(s.playerShipEntity.currentShipPitch, mgl.Vec3{1.0, 0.0, 0.0})
	s.playerShipEntity.SetOrientation(qRoll.Mul(qPitch))

	// move the ship around
	rollRatio := s.playerShipEntity.currentShipRoll / maxRollRads
	pitchRatio := s.playerShipEntity.currentShipPitch / maxPitchRads
	const moveSpeed = 20.0 // 1 m/s
	shipLoc := s.playerShipEntity.GetLocation()
	shipLoc[0] -= moveSpeed * rollRatio * frameDelta
	shipLoc[1] -= moveSpeed * pitchRatio * frameDelta
	s.playerShipEntity.SetLocation(shipLoc)

	// glue the HMD to the ship
	hmdLoc := s.vrRenderSystem.GetHMDLocation()
	s.playerEntity.SetLocation(s.playerShipEntity.GetLocation().Add(mgl.Vec3{
		0.0 - hmdLoc[0],
		0.2 - hmdLoc[1],
		-0.5 - hmdLoc[2]}))
}

// HandleMenuButtonInput should be invoked when the top menu button on the
// vive controller is pressed.
func (s *VRInputSystem) HandleMenuButtonInput() {
	// if the game is in the PlayerDied state, this button will reset the game.
	if gameScene.gameState == gameStatePlayerDied {
		err := gameScene.ResetScene()
		if err != nil {
			fmt.Printf("Could not reset the game: %v\n", err)
			gameScene.ShouldClose = true
		}
		return
	}

	// otherwise, use this button to auto level the HMD
	s.HandleHeadAutoLevel()
}

// HandleHeadAutoLevel should be called to set the auto-level 'head' position. This allows
// for the HMD to move around and affect the camera, but be centered appropriately for a
// sitting position.
func (s *VRInputSystem) HandleHeadAutoLevel() {
	// update the playerEntity's height to account for a new calibration of
	// the HMD head position
	hmdLoc := s.vrRenderSystem.GetHMDLocation()
	s.playerEntity.SetLocation(s.playerShipEntity.GetLocation().Add(mgl.Vec3{
		0.0 - hmdLoc[0],
		0.2 - hmdLoc[1],
		-0.5 - hmdLoc[2]}))
}

// OnAddEntity should get called by the scene Manager each time a new entity
// has been added to the scene.
func (s *VRInputSystem) OnAddEntity(newEntity scene.Entity) {
	// we cache the player and their ship for moving around based on input
	name := newEntity.GetName()
	if name == playerEntityName {
		s.playerEntity = newEntity.(*VisibleEntity)
	} else if name == playerShipEntityName {
		s.playerShipEntity = newEntity.(*ShipEntity)
	}
}

// OnRemoveEntity should get called by the scene Manager each time an entity
// has been removed from the scene.
func (s *VRInputSystem) OnRemoveEntity(oldEntity scene.Entity) {
	name := oldEntity.GetName()
	if name == playerEntityName {
		s.playerEntity = nil
	} else if name == playerShipEntityName {
		s.playerShipEntity = nil
	}
}

// GetRequestedPriority returns the requested priority level for the System
// which may be of significance to a Manager if they want to order Update() calls.
func (s *VRInputSystem) GetRequestedPriority() float32 {
	return vrInputSystemPriority
}

// GetName returns the name of the system that can be used to identify
// the System within Manager.
func (s *VRInputSystem) GetName() string {
	return vrInputSystemName
}
