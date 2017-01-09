// Copyright 2016, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package main

import (
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
}

// NewVRInputSystem creates a new InputSystem object
func NewVRInputSystem() *VRInputSystem {
	system := new(VRInputSystem)
	return system
}

// Initialize sets up the input models for the scene.
func (s *VRInputSystem) Initialize(vrSystem *vr.System) {
	s.vrSystem = vrSystem
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
}

// OnAddEntity should get called by the scene Manager each time a new entity
// has been added to the scene.
func (s *VRInputSystem) OnAddEntity(newEntity scene.Entity) {
	// NOP
}

// OnRemoveEntity should get called by the scene Manager each time an entity
// has been removed from the scene.
func (s *VRInputSystem) OnRemoveEntity(oldEntity scene.Entity) {
	// NOP
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
