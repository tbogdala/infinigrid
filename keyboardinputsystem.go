// Copyright 2016, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package main

import (
	glfw "github.com/go-gl/glfw/v3.1/glfw"
	mgl "github.com/go-gl/mathgl/mgl32"

	input "github.com/tbogdala/fizzle/input/glfwinput"
	"github.com/tbogdala/fizzle/scene"
)

const (
	keyboardInputSystemPriority = -100.0
	keyboardInputSystemName     = "KeyboardInputSystem"
)

var (
	// scales the roll and pitch movement for keypresses
	kbRollPitchSpeedF = float32(1.5)
)

// KeyboardInputSystem implements the System interface and handles the
// player input via keyboard.
type KeyboardInputSystem struct {
	kbModel    *input.KeyboardModel
	mainWindow *glfw.Window

	frameDelta float32

	// playerShipEntity is the cached reference to the player ship pawn.
	playerShipEntity *ShipEntity
}

// NewKeyboardInputSystem creates a new KeyboardInputSystem object
func NewKeyboardInputSystem() *KeyboardInputSystem {
	system := new(KeyboardInputSystem)
	return system
}

// Initialize sets up the input models for the scene.
func (s *KeyboardInputSystem) Initialize(w *glfw.Window) {
	s.mainWindow = w

	// set the callback functions for key input
	s.kbModel = input.NewKeyboardModel(s.mainWindow)
	s.kbModel.Bind(glfw.KeyA, s.handleRollLeft)
	s.kbModel.Bind(glfw.KeyD, s.handleRollRight)
	s.kbModel.Bind(glfw.KeyW, s.handlePitchUp)
	s.kbModel.Bind(glfw.KeyS, s.handlePitchDown)
	s.kbModel.SetupCallbacks()

}

// Update should get called to run updates for the system every frame
// by the owning Manager object.
func (s *KeyboardInputSystem) Update(frameDelta float32) {
	// cache this in the system object so the keyboard handlers can reference it
	s.frameDelta = frameDelta

	// advise GLFW to poll for input. without this the window appears to hang.
	glfw.PollEvents()

	// handle any keyboard input
	s.kbModel.CheckKeyPresses()

	// modify ship rotation based on input state
	qRoll := mgl.QuatRotate(s.playerShipEntity.currentShipRoll, mgl.Vec3{0.0, 0.0, 1.0})
	qPitch := mgl.QuatRotate(s.playerShipEntity.currentShipPitch, mgl.Vec3{1.0, 0.0, 0.0})
	s.playerShipEntity.SetOrientation(qRoll.Mul(qPitch))

	// HACK: move the ship around
	rollRatio := s.playerShipEntity.currentShipRoll / maxRollRads
	pitchRatio := s.playerShipEntity.currentShipPitch / maxPitchRads
	shipLoc := s.playerShipEntity.GetLocation()
	shipLoc[0] -= shipMoveSpeed * rollRatio * frameDelta
	shipLoc[1] -= shipMoveSpeed * pitchRatio * frameDelta
	s.playerShipEntity.SetLocation(shipLoc)
}

// OnAddEntity should get called by the scene Manager each time a new entity
// has been added to the scene.
func (s *KeyboardInputSystem) OnAddEntity(newEntity scene.Entity) {
	if newEntity.GetName() == playerShipEntityName {
		s.playerShipEntity = newEntity.(*ShipEntity)
	}
}

// OnRemoveEntity should get called by the scene Manager each time an entity
// has been removed from the scene.
func (s *KeyboardInputSystem) OnRemoveEntity(oldEntity scene.Entity) {
	if oldEntity.GetName() == playerShipEntityName {
		s.playerShipEntity = nil
	}
}

// GetRequestedPriority returns the requested priority level for the System
// which may be of significance to a Manager if they want to order Update() calls.
func (s *KeyboardInputSystem) GetRequestedPriority() float32 {
	return keyboardInputSystemPriority
}

// GetName returns the name of the system that can be used to identify
// the System within Manager.
func (s *KeyboardInputSystem) GetName() string {
	return keyboardInputSystemName
}

func (s *KeyboardInputSystem) handleRollLeft() {
	s.handleRollLeftV(1.0)
}

func (s *KeyboardInputSystem) handleRollRight() {
	s.handleRollRightV(1.0)
}

func (s *KeyboardInputSystem) handlePitchDown() {
	s.handlePitchDownV(1.0)
}

func (s *KeyboardInputSystem) handlePitchUp() {
	s.handlePitchUpV(1.0)
}

func (s *KeyboardInputSystem) handleRollLeftV(v float32) {
	s.playerShipEntity.currentShipRoll += -v * maxRollRads * s.frameDelta * kbRollPitchSpeedF
	s.playerShipEntity.currentShipRoll = mgl.Clamp(s.playerShipEntity.currentShipRoll, -maxRollRads, maxRollRads)
}

func (s *KeyboardInputSystem) handleRollRightV(v float32) {
	s.playerShipEntity.currentShipRoll += v * maxRollRads * s.frameDelta * kbRollPitchSpeedF
	s.playerShipEntity.currentShipRoll = mgl.Clamp(s.playerShipEntity.currentShipRoll, -maxRollRads, maxRollRads)
}

func (s *KeyboardInputSystem) handlePitchDownV(v float32) {
	s.playerShipEntity.currentShipPitch += v * maxPitchRads * s.frameDelta * kbRollPitchSpeedF
	s.playerShipEntity.currentShipPitch = mgl.Clamp(s.playerShipEntity.currentShipPitch, -maxPitchRads, maxPitchRads)
}

func (s *KeyboardInputSystem) handlePitchUpV(v float32) {
	s.playerShipEntity.currentShipPitch += -v * maxPitchRads * s.frameDelta * kbRollPitchSpeedF
	s.playerShipEntity.currentShipPitch = mgl.Clamp(s.playerShipEntity.currentShipPitch, -maxPitchRads, maxPitchRads)
}
