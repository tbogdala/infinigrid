// Copyright 2016, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	vr "github.com/tbogdala/openvr-go"

	glfw "github.com/go-gl/glfw/v3.1/glfw"

	input "github.com/tbogdala/fizzle/input/glfwinput"
)

const (
	windowWidth  = int(1280)
	windowHeight = int(720)
)

var (
	kbModel *input.KeyboardModel
)

func init() {
	runtime.LockOSThread()
}

func main() {
	var err error

	// create the render system and initialize it
	renderSystem := NewVRRenderSystem()
	err = renderSystem.Initialize("GRID", windowWidth, windowHeight)
	if err != nil {
		fmt.Printf("Failed to initialize the VR render system! %v", err)
		os.Exit(1)
	}

	// create the vr input system to handle the vr controllers
	inputSystem := NewVRInputSystem()
	inputSystem.Initialize(renderSystem.GetVRSystem())

	// create a scene manager
	gameScene := NewGameScene()
	gameScene.AddSystem(renderSystem)
	gameScene.AddSystem(inputSystem)

	// create some objects and lights
	gameScene.SetupScene()

	////////////////////////////////////////////////////////////////////////////
	// wire some inputs for the vive wands
	inputSystem.OnAppMenuButtonL = gameScene.HandleHeadAutoLevel

	////////////////////////////////////////////////////////////////////////////
	// set the callback functions for key input
	kbModel = input.NewKeyboardModel(renderSystem.MainWindow)
	kbModel.BindTrigger(glfw.KeyEscape, func() {
		renderSystem.MainWindow.SetShouldClose(true)
	})
	kbModel.SetupCallbacks()

	////////////////////////////////////////////////////////////////////////////
	// the main application loop
	lastFrame := time.Now()
	for !renderSystem.MainWindow.ShouldClose() {
		// calculate the difference in time to control rotation speed
		thisFrame := time.Now()
		frameDelta := float32(thisFrame.Sub(lastFrame).Seconds())

		handleInput()

		// update the game scene
		gameScene.Update(frameDelta)

		// update our last frame time
		lastFrame = thisFrame
	}

	vr.Shutdown()
}

func handleInput() {
	// advise GLFW to poll for input. without this the window appears to hang.
	glfw.PollEvents()

	// handle any keyboard input
	kbModel.CheckKeyPresses()
}
