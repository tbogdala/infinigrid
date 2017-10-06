// Copyright 2017, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package main

import (
	"fmt"
	"math"

	glfw "github.com/go-gl/glfw/v3.1/glfw"
	mgl "github.com/go-gl/mathgl/mgl32"

	fizzle "github.com/tbogdala/fizzle"
	graphics "github.com/tbogdala/fizzle/graphicsprovider"
	opengl "github.com/tbogdala/fizzle/graphicsprovider/opengl"
	forward "github.com/tbogdala/fizzle/renderer/forward"
	"github.com/tbogdala/fizzle/scene"
)

const (
	forwardRenderSystemPriority = 100.0
	forwardRenderSystemName     = "RenderSystem"
)

// ForwardRenderSystem implements fizzle/scene/System interface and handles the rendering
// of entities in the scene in a forward renderer.
type ForwardRenderSystem struct {
	Renderer   *forward.ForwardRenderer
	MainWindow *glfw.Window
	Camera     *fizzle.OrbitCamera
	gfx        graphics.GraphicsProvider

	visibleEntities []scene.Entity

	// cachedPlayerEntity is the player entity that was added to the scene.
	cachedPlayerEntity *VisibleEntity

	// cachedPlayerShipEntity is the player's ship entity that was added to the scene.
	cachedPlayerShipEntity *ShipEntity
}

// NewForwardRenderSystem allocates a new ForwardRenderSystem object.
func NewForwardRenderSystem() *ForwardRenderSystem {
	rs := new(ForwardRenderSystem)
	rs.visibleEntities = []scene.Entity{}
	return rs
}

// Initialize will create the main window using glfw and then create the underyling
// renderer.
func (rs *ForwardRenderSystem) Initialize(windowName string, w int, h int) error {
	// create the window and iniitialize opengl
	err := rs.initGraphics(windowName, w, h)
	if err != nil {
		return err
	}

	// setup the forward renderer
	rs.Renderer = forward.NewForwardRenderer(rs.gfx)
	rs.Renderer.ChangeResolution(int32(w), int32(h))

	// setup the camera to look at the ship
	rs.Camera = fizzle.NewOrbitCamera(mgl.Vec3{0, 0, 0}, math.Pi/2.5, 0.5, 1.5*math.Pi)

	// set some OpenGL flags
	rs.gfx.Enable(graphics.CULL_FACE)
	rs.gfx.Enable(graphics.DEPTH_TEST)
	rs.gfx.Enable(graphics.MIPMAP)
	rs.gfx.Enable(graphics.BLEND)
	rs.gfx.BlendFunc(graphics.SRC_ALPHA, graphics.ONE_MINUS_SRC_ALPHA)
	return nil
}

// initGraphics creates an OpenGL window and initializes the required graphics libraries.
// It will either succeed or panic.
func (rs *ForwardRenderSystem) initGraphics(title string, w int, h int) error {
	// GLFW must be initialized before it's called
	err := glfw.Init()
	if err != nil {
		return fmt.Errorf("Failed to initialize GLFW. %v", err)
	}

	// request a OpenGL 3.3 core context
	glfw.WindowHint(glfw.Samples, glSamples)
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)

	// do the actual window creation
	rs.MainWindow, err = glfw.CreateWindow(w, h, title, nil, nil)
	if err != nil {
		return fmt.Errorf("Failed to create the main window. %v", err)
	}

	// set a function to update the renderer on window resize
	rs.MainWindow.SetSizeCallback(func(w *glfw.Window, width int, height int) {
		rs.Renderer.ChangeResolution(int32(width), int32(height))
	})

	rs.MainWindow.MakeContextCurrent()

	// disable v-sync for max draw rate
	glfw.SwapInterval(0)

	// initialize OpenGL
	rs.gfx, err = opengl.InitOpenGL()
	if err != nil {
		return fmt.Errorf("Failed to initialize OpenGL. %v", err)
	}
	fizzle.SetGraphics(rs.gfx)

	return nil
}

// GetRenderer returns the internal renderer being used.
func (rs *ForwardRenderSystem) GetRenderer() *forward.ForwardRenderer {
	return rs.Renderer
}

// GetMainWindow returns the internal renderer being used.
func (rs *ForwardRenderSystem) GetMainWindow() *glfw.Window {
	return rs.MainWindow
}

// SetLight puts a light in the specified slot for the renderer.
func (rs *ForwardRenderSystem) SetLight(i int, l *forward.Light) {
	rs.Renderer.ActiveLights[i] = l
}

// GetRequestedPriority returns the requested priority level for the System
// which may be of significance to a Manager if they want to order Update() calls.
func (rs *ForwardRenderSystem) GetRequestedPriority() float32 {
	return forwardRenderSystemPriority
}

// GetName returns the name of the system that can be used to identify
// the System within Manager.
func (rs *ForwardRenderSystem) GetName() string {
	return forwardRenderSystemName
}

// OnAddEntity should get called by the scene Manager each time a new entity
// has been added to the scene.
func (rs *ForwardRenderSystem) OnAddEntity(newEntity scene.Entity) {
	_, okay := newEntity.(RenderableEntity)
	if okay {
		rs.visibleEntities = append(rs.visibleEntities, newEntity)

		if newEntity.GetName() == playerEntityName {
			rs.cachedPlayerEntity = newEntity.(*VisibleEntity)
		} else if newEntity.GetName() == playerShipEntityName {
			rs.cachedPlayerShipEntity = newEntity.(*ShipEntity)
		}
	}
}

// OnRemoveEntity should get called by the scene Manager each time an entity
// has been removed from the scene.
func (rs *ForwardRenderSystem) OnRemoveEntity(oldEntity scene.Entity) {
	surviving := rs.visibleEntities[:0]
	for _, e := range rs.visibleEntities {
		if e.GetID() != oldEntity.GetID() {
			surviving = append(surviving, e)
		}
	}
	rs.visibleEntities = surviving

	if oldEntity.GetName() == playerEntityName {
		rs.cachedPlayerEntity = nil
	} else if oldEntity.GetName() == playerShipEntityName {
		rs.cachedPlayerShipEntity = nil
	}
}

// Update renderers the known entities.
func (rs *ForwardRenderSystem) Update(frameDelta float32) {
	// clear the screen
	width, height := rs.Renderer.GetResolution()
	rs.gfx.Viewport(0, 0, int32(width), int32(height))
	rs.gfx.ClearColor(0.15, 0.15, 0.18, 1.0) // nice background color, but not black
	rs.gfx.Clear(graphics.COLOR_BUFFER_BIT | graphics.DEPTH_BUFFER_BIT)

	// make the projection and view matrixes
	projection := mgl.Perspective(fovyRads, float32(width)/float32(height), nearView, farView)
	if rs.cachedPlayerShipEntity != nil {
		rs.Camera.SetTarget(rs.cachedPlayerShipEntity.GetLocation())
	}
	view := rs.Camera.GetViewMatrix()

	// draw stuff the visible entities
	for _, e := range rs.visibleEntities {
		visibleEntity, okay := e.(RenderableEntity)
		if okay {
			if r := visibleEntity.GetRenderable(); r != nil {
				rs.Renderer.DrawRenderable(r, nil, projection, view, rs.Camera)
			}
		}
	}

	// draw the screen
	rs.MainWindow.SwapBuffers()
}
