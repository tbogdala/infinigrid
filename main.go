// Copyright 2017, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"runtime/pprof"
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
	kbModel   *input.KeyboardModel
	gameScene *GameScene

	flagUseVR      = flag.Bool("vr", false, "run the game in VR mode")
	flagCPUProfile = flag.String("cpuprofile", "", "provide a filename for the output pprof file")
)

func init() {
	runtime.LockOSThread()
}

func main() {
	var err error
	flag.Parse()

	// potentially enable cpu profiling
	if *flagCPUProfile != "" {
		fmt.Printf("Enabling CPU Profiling!\n")
		cpuPprofF, err := os.Create(*flagCPUProfile)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		pprof.StartCPUProfile(cpuPprofF)
		defer func() {
			pprof.StopCPUProfile()
			cpuPprofF.Close()
		}()
	}

	// seed the RNG
	rand.Seed(time.Now().UnixNano())

	var renderSystem RenderSystem
	var renderSceneSystem scene.System
	var inputSceneSystem scene.System
	var uiSceneSystem scene.System

	// setup vr mode if indicated via command line flag
	if *flagUseVR {
		// create the render system and initialize it
		vrRenderSystem := NewVRRenderSystem()
		err = vrRenderSystem.Initialize("Infinigrid", windowWidth, windowHeight)
		if err != nil {
			fmt.Printf("Failed to initialize the VR render system! %v", err)
			return
		}

		// create the vr input system to handle the vr controllers
		vrInputSystem := NewVRInputSystem()
		vrInputSystem.Initialize(vrRenderSystem)

		// wire some inputs for the vive wands
		vrInputSystem.OnAppMenuButtonL = vrInputSystem.HandleMenuButtonInput

		renderSystem = vrRenderSystem
		renderSceneSystem = vrRenderSystem
		inputSceneSystem = vrInputSystem
	} else {
		// no vr flag was specified so construct a normal renderer
		forwardRenderSystem := NewForwardRenderSystem()
		err = forwardRenderSystem.Initialize("Infinigrid", windowWidth, windowHeight)
		if err != nil {
			fmt.Printf("Failed to initialize the VR render system! %v", err)
			return
		}

		// create the keyboard interface to the game
		kbInputSystem := NewKeyboardInputSystem()
		kbInputSystem.Initialize(forwardRenderSystem.GetMainWindow())

		// use a 'traditional' user interface system for the game UI
		uisys := NewUISystem()
		err = uisys.Initialize(forwardRenderSystem)
		if err != nil {
			fmt.Printf("Failed to initialize the user interface! %v", err)
			return
		}

		renderSystem = forwardRenderSystem
		renderSceneSystem = forwardRenderSystem
		inputSceneSystem = kbInputSystem
		uiSceneSystem = uisys
	}

	////////////////////////////////////////////////////////////////////////////
	// create a scene manager
	gameScene = NewGameScene()
	gameScene.AddSystem(renderSceneSystem)
	gameScene.AddSystem(inputSceneSystem)
	gameScene.AddSystem(uiSceneSystem)

	// create some objects and lights
	err = gameScene.SetupScene()
	if err != nil {
		fmt.Printf("Failed to setup the game scene. %v\n", err)
		return
	}

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
	for !mainWindow.ShouldClose() && !gameScene.ShouldClose {
		// calculate the difference in time to control rotation speed
		thisFrame := time.Now()
		frameDelta := float32(thisFrame.Sub(lastFrame).Seconds())

		handleInput()

		// update the game scene
		gameScene.Update(frameDelta)

		// update our last frame time
		lastFrame = thisFrame

		// draw the screen
		mainWindow.SwapBuffers()
	}

	vr.Shutdown()
}

func handleInput() {
	// advise GLFW to poll for input. without this the window appears to hang.
	glfw.PollEvents()

	// handle any keyboard input
	kbModel.CheckKeyPresses()
}
