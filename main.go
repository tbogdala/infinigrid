// Copyright 2017, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package main

import (
	"flag"
	"fmt"
	"runtime"
	"time"

	"github.com/tbogdala/fizzle/scene"

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

	flagUseVR = flag.Bool("vr", false, "run the game in VR mode")
)

func init() {
	runtime.LockOSThread()
}

func main() {
	var err error
	flag.Parse()

	var renderSystem RenderSystem
	var renderSceneSystem scene.System
	var inputSceneSystem scene.System

	// setup vr mode if indicated via command line flag
	if *flagUseVR {
		// create the render system and initialize it
		vrRenderSystem := NewVRRenderSystem()
		err = vrRenderSystem.Initialize("GRID", windowWidth, windowHeight)
		if err != nil {
			fmt.Printf("Failed to initialize the VR render system! %v", err)
			return
		}

		// create the vr input system to handle the vr controllers
		vrInputSystem := NewVRInputSystem()
		vrInputSystem.Initialize(vrRenderSystem)

		// wire some inputs for the vive wands
		vrInputSystem.OnAppMenuButtonL = vrInputSystem.HandleHeadAutoLevel

		renderSystem = vrRenderSystem
		renderSceneSystem = vrRenderSystem
		inputSceneSystem = vrInputSystem
	} else {
		// no vr flag was specified so construct a normal renderer
		forwardRenderSystem := NewForwardRenderSystem()
		err = forwardRenderSystem.Initialize("GRID", windowWidth, windowHeight)
		if err != nil {
			fmt.Printf("Failed to initialize the VR render system! %v", err)
			return
		}

		// create the keyboard interface to the game
		kbInputSystem := NewKeyboardInputSystem()
		kbInputSystem.Initialize(forwardRenderSystem.GetMainWindow())

		renderSystem = forwardRenderSystem
		renderSceneSystem = forwardRenderSystem
		inputSceneSystem = kbInputSystem
	}

	////////////////////////////////////////////////////////////////////////////
	// create a scene manager
	gameScene := NewGameScene()
	gameScene.AddSystem(renderSceneSystem)
	gameScene.AddSystem(inputSceneSystem)

	// create some objects and lights
	gameScene.SetupScene()

	////////////////////////////////////////////////////////////////////////////
	// set the callback functions for key input common to all input systems
	mainWindow := renderSystem.GetMainWindow()
	kbModel = input.NewKeyboardModel(mainWindow)
	kbModel.BindTrigger(glfw.KeyEscape, func() {
		mainWindow.SetShouldClose(true)
	})
	kbModel.SetupCallbacks()

	////////////////////////////////////////////////////////////////////////////
	// the main application loop
	lastFrame := time.Now()
	for !mainWindow.ShouldClose() {
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
