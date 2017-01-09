// Copyright 2016, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package main

import (
	"fmt"

	glfw "github.com/go-gl/glfw/v3.1/glfw"
	mgl "github.com/go-gl/mathgl/mgl32"

	fizzle "github.com/tbogdala/fizzle"
	graphics "github.com/tbogdala/fizzle/graphicsprovider"
	opengl "github.com/tbogdala/fizzle/graphicsprovider/opengl"
	forward "github.com/tbogdala/fizzle/renderer/forward"
	scene "github.com/tbogdala/fizzle/scene"
	vr "github.com/tbogdala/openvr-go"
	fizzlevr "github.com/tbogdala/openvr-go/util/fizzlevr"
)

const (
	vrRenderSystemPriority = 100.0
	vrRenderSystemName     = "VRRenderSystem"
	nearView               = 0.1
	farView                = 30.0
)

// VRRenderSystem implements fizzle/scene/System interface and handles the rendering
// of entities in the scene.
type VRRenderSystem struct {
	Renderer   *forward.ForwardRenderer
	MainWindow *glfw.Window

	gfx graphics.GraphicsProvider

	currentWindowWidth  int
	currentWindowHeight int

	// interfaces for openvr
	vrSystem       *vr.System
	vrCompositor   *vr.Compositor
	distortionLens *fizzlevr.DistortionLens

	// render surfaces and transforms
	renderWidth         uint32
	renderHeight        uint32
	eyeTransforms       *vr.EyeTransforms
	eyeFramebufferLeft  *fizzlevr.EyeFramebuffer
	eyeFramebufferRight *fizzlevr.EyeFramebuffer
	hmdPose             mgl.Mat4
	hmdLoc              mgl.Vec3

	visibleEntities []scene.Entity

	// cachedPlayerEntity is the player entity that was added to the scene.
	cachedPlayerEntity *VisibleEntity
}

// NewVRRenderSystem allocates a new VRRenderSystem object.
func NewVRRenderSystem() *VRRenderSystem {
	rs := new(VRRenderSystem)
	rs.visibleEntities = []scene.Entity{}
	return rs
}

// GetVRSystem returns the vr.System interface that was obtained during Initialize().
func (rs *VRRenderSystem) GetVRSystem() *vr.System {
	return rs.vrSystem
}

// GetHMDLocation returns a vector describing the location of the HMD in the 'room'.
func (rs *VRRenderSystem) GetHMDLocation() mgl.Vec3 {
	return rs.hmdLoc
}

// Initialize will create the main window using glfw and then create the underyling
// renderer.
func (rs *VRRenderSystem) Initialize(windowName string, w int, h int) error {
	var err error
	rs.vrSystem, err = vr.Init()
	if err != nil || rs.vrSystem == nil {
		return fmt.Errorf("vr.Init() returned an error: %v", err)
	}

	// print out some information about the headset as a good smoke test
	driver, errInt := rs.vrSystem.GetStringTrackedDeviceProperty(int(vr.TrackedDeviceIndexHmd), vr.PropTrackingSystemNameString)
	if errInt != vr.TrackedPropSuccess {
		return fmt.Errorf("error getting VR driver name")
	}
	displaySerial, errInt := rs.vrSystem.GetStringTrackedDeviceProperty(int(vr.TrackedDeviceIndexHmd), vr.PropSerialNumberString)
	if errInt != vr.TrackedPropSuccess {
		return fmt.Errorf("error getting VR display name")
	}
	fmt.Printf("Connected to %s %s\n", driver, displaySerial)

	// get the size of the render targets to make
	rs.renderWidth, rs.renderHeight = rs.vrSystem.GetRecommendedRenderTargetSize()
	fmt.Printf("rec size: %d, %d\n", rs.renderWidth, rs.renderHeight)

	// create the window and iniitialize opengl
	err = rs.initGraphics(windowName, w, h, int(rs.renderWidth), int(rs.renderHeight))
	if err != nil {
		return err
	}

	// create a new renderer
	rs.Renderer = forward.NewForwardRenderer(rs.gfx)
	rs.Renderer.ChangeResolution(int32(rs.renderWidth), int32(rs.renderHeight))

	// get the eye transforms necessary for the VR HMD
	rs.eyeTransforms = rs.vrSystem.GetEyeTransforms(nearView, farView)

	// setup the framebuffers for the eyes
	rs.eyeFramebufferLeft, rs.eyeFramebufferRight = fizzlevr.CreateStereoRenderTargets(rs.renderWidth, rs.renderHeight)

	// load the shader used to render the framebuffers to a window for viewing
	lensShader, err := fizzle.LoadShaderProgram(vr.ShaderLensDistortionV, vr.ShaderLensDistortionF, nil)
	if err != nil {
		return fmt.Errorf("Failed to compile and link the lens distortion shader program!\n%v", err)
	}

	// create the lens distortion object which will be used to render the
	// eye framebuffers to the GLFW window.
	rs.distortionLens = fizzlevr.CreateDistortionLens(rs.vrSystem, lensShader, rs.eyeFramebufferLeft, rs.eyeFramebufferRight)

	// pull an interface to the compositor
	rs.vrCompositor, err = vr.GetCompositor()
	if err != nil {
		return fmt.Errorf("Failed to get the compositor interface: %v", err)
	}

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
func (rs *VRRenderSystem) initGraphics(title string, w int, h int, rw int, rh int) error {
	// GLFW must be initialized before it's called
	err := glfw.Init()
	if err != nil {
		return fmt.Errorf("Failed to initialize GLFW. %v", err)
	}

	// request a OpenGL 3.3 core context
	glfw.WindowHint(glfw.Samples, 0)
	glfw.WindowHint(glfw.ContextVersionMajor, 3)
	glfw.WindowHint(glfw.ContextVersionMinor, 3)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)

	// do the actual window creation
	rs.MainWindow, err = glfw.CreateWindow(w, h, title, nil, nil)
	if err != nil {
		return fmt.Errorf("Failed to create the main window. %v", err)
	}

	rs.currentWindowWidth = w
	rs.currentWindowHeight = h

	// set a function to update the renderer on window resize
	rs.MainWindow.SetSizeCallback(func(w *glfw.Window, width int, height int) {
		rs.currentWindowWidth = width
		rs.currentWindowHeight = height
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

// SetLight puts a light in the specified slot for the renderer.
func (rs *VRRenderSystem) SetLight(i int, l *forward.Light) {
	rs.Renderer.ActiveLights[i] = l
}

// GetRequestedPriority returns the requested priority level for the System
// which may be of significance to a Manager if they want to order Update() calls.
func (rs *VRRenderSystem) GetRequestedPriority() float32 {
	return vrRenderSystemPriority
}

// GetName returns the name of the system that can be used to identify
// the System within Manager.
func (rs *VRRenderSystem) GetName() string {
	return vrRenderSystemName
}

// OnAddEntity should get called by the scene Manager each time a new entity
// has been added to the scene.
func (rs *VRRenderSystem) OnAddEntity(newEntity scene.Entity) {
	_, okay := newEntity.(RenderableEntity)
	if okay {
		rs.visibleEntities = append(rs.visibleEntities, newEntity)
		// check to see if it's the player entity; if so, cache the reference
		if newEntity.GetName() == playerEntityName {
			rs.cachedPlayerEntity, _ = newEntity.(*VisibleEntity)
		}
	}
}

// OnRemoveEntity should get called by the scene Manager each time an entity
// has been removed from the scene.
func (rs *VRRenderSystem) OnRemoveEntity(oldEntity scene.Entity) {
	surviving := rs.visibleEntities[:0]
	for _, e := range rs.visibleEntities {
		if e.GetID() != oldEntity.GetID() {
			surviving = append(surviving, e)
		}
	}
	rs.visibleEntities = surviving
}

// Update renderers the known entities.
func (rs *VRRenderSystem) Update(frameDelta float32) {
	// draw the framebuffers
	rs.renderStereoTargets()

	// draw the framebuffers to the window
	rs.distortionLens.Render(int32(rs.currentWindowWidth), int32(rs.currentWindowHeight))

	// send the framebuffer textures out to the compositor for rendering to the HMD
	rs.vrCompositor.Submit(vr.EyeLeft, uint32(rs.eyeFramebufferLeft.ResolveTexture))
	rs.vrCompositor.Submit(vr.EyeRight, uint32(rs.eyeFramebufferRight.ResolveTexture))

	// draw the screen
	rs.MainWindow.SwapBuffers()

	// update the HMD pose, which causes a wait to vsync the HMD
	rs.updateHMDPose()
}

func (rs *VRRenderSystem) updateHMDPose() {
	// WaitGetPoses is used as a sync point in the OpenVR API. This is on a timer to keep 90fps, so
	// the OpenVR gives you that much time to draw a frame. By calling WaitGetPoses() you wait the
	// remaining amount of time. If you only used 1ms it will wait 10ms here. If you used 5ms it will wait 6ms.
	// (approx.)

	// Side note: I believe you orient the room space to have 'forward' be toward
	// the monitor during calibration. So as I face my monitor, +Z is forward,
	// +X is to my right and +Y is toward the celing.
	rs.vrCompositor.WaitGetPoses(false)
	if rs.vrCompositor.IsPoseValid(vr.TrackedDeviceIndexHmd) {
		pose := rs.vrCompositor.GetRenderPose(vr.TrackedDeviceIndexHmd)
		rs.hmdPose = mgl.Mat4(vr.Mat34ToMat4(&pose.DeviceToAbsoluteTracking)).Inv()

		rs.hmdLoc[0] = pose.DeviceToAbsoluteTracking[9]
		rs.hmdLoc[1] = pose.DeviceToAbsoluteTracking[10]
		rs.hmdLoc[2] = pose.DeviceToAbsoluteTracking[11]
	}
}

// renderStereoTargets renders each of the left and right eye framebuffers
// calling renderScene to do the rendering for the scene.
func (rs *VRRenderSystem) renderStereoTargets() {
	gfx := rs.gfx
	gfx.Enable(graphics.CULL_FACE)
	gfx.ClearColor(0.15, 0.15, 0.18, 1.0) // nice background color, but not black

	// left eye
	gfx.Enable(graphics.MULTISAMPLE)
	gfx.BindFramebuffer(graphics.FRAMEBUFFER, rs.eyeFramebufferLeft.RenderFramebuffer)
	gfx.Viewport(0, 0, int32(rs.renderWidth), int32(rs.renderHeight))
	rs.renderScene(vr.EyeLeft)
	gfx.BindFramebuffer(graphics.FRAMEBUFFER, 0)
	gfx.Disable(graphics.MULTISAMPLE)

	gfx.BindFramebuffer(graphics.READ_FRAMEBUFFER, rs.eyeFramebufferLeft.RenderFramebuffer)
	gfx.BindFramebuffer(graphics.DRAW_FRAMEBUFFER, rs.eyeFramebufferLeft.ResolveFramebuffer)
	gfx.BlitFramebuffer(0, 0, int32(rs.renderWidth), int32(rs.renderHeight), 0, 0, int32(rs.renderWidth), int32(rs.renderHeight), graphics.COLOR_BUFFER_BIT, graphics.LINEAR)
	gfx.BindFramebuffer(graphics.READ_FRAMEBUFFER, 0)
	gfx.BindFramebuffer(graphics.DRAW_FRAMEBUFFER, 0)

	// right eye
	gfx.Enable(graphics.MULTISAMPLE)
	gfx.BindFramebuffer(graphics.FRAMEBUFFER, rs.eyeFramebufferRight.RenderFramebuffer)
	gfx.Viewport(0, 0, int32(rs.renderWidth), int32(rs.renderHeight))
	rs.renderScene(vr.EyeRight)
	gfx.BindFramebuffer(graphics.FRAMEBUFFER, 0)
	gfx.Disable(graphics.MULTISAMPLE)

	gfx.BindFramebuffer(graphics.READ_FRAMEBUFFER, rs.eyeFramebufferRight.RenderFramebuffer)
	gfx.BindFramebuffer(graphics.DRAW_FRAMEBUFFER, rs.eyeFramebufferRight.ResolveFramebuffer)
	gfx.BlitFramebuffer(0, 0, int32(rs.renderWidth), int32(rs.renderHeight), 0, 0, int32(rs.renderWidth), int32(rs.renderHeight), graphics.COLOR_BUFFER_BIT, graphics.LINEAR)
	gfx.BindFramebuffer(graphics.READ_FRAMEBUFFER, 0)
	gfx.BindFramebuffer(graphics.DRAW_FRAMEBUFFER, 0)

}

// FixedCamera is a fake camera for the VR HMD
type FixedCamera struct {
	View     mgl.Mat4
	Position mgl.Vec3
}

// GetViewMatrix returns the hmd's view
func (c FixedCamera) GetViewMatrix() mgl.Mat4 {
	return c.View
}

// GetPosition returns the hmd's position
func (c FixedCamera) GetPosition() mgl.Vec3 {
	return c.Position
}

// renderScene gets called for each eye and is responsible for
// rendering the entire scene.
func (rs *VRRenderSystem) renderScene(eye int) {
	rs.gfx.Clear(graphics.COLOR_BUFFER_BIT | graphics.DEPTH_BUFFER_BIT)
	rs.gfx.Enable(graphics.DEPTH_TEST)

	var perspective, worldView mgl.Mat4
	var camera FixedCamera

	// construct the worldHmdPose translation matrix that will place the HMD pose
	// into world space.
	var playerTranslation mgl.Mat4
	var playerPosition mgl.Vec3
	if rs.cachedPlayerEntity != nil {
		playerPosition = rs.cachedPlayerEntity.GetLocation()
		playerCameraTranslation := playerPosition.Mul(-1) //.Add(rs.hmdPose.Col(3).Vec3())
		playerTranslation = mgl.Translate3D(playerCameraTranslation[0], playerCameraTranslation[1], playerCameraTranslation[2])
	} else {
		playerPosition = mgl.Vec3{0, 0, 0}
		playerTranslation = mgl.Translate3D(0, 0, 0)
	}

	//worldHmdPose := mgl.Mat4FromCols(rs.hmdPose.Col(0), rs.hmdPose.Col(1), rs.hmdPose.Col(2), mgl.Vec4{0, 0, 0, 1}).Mul4(playerTranslation)
	worldHmdPose := rs.hmdPose.Mul4(playerTranslation)

	// Note: hmdLocalView is not being generated ATM. Only necessary to show
	// renderables local to the HMD space like controllers or maybe HUD.
	if eye == vr.EyeLeft {
		worldView = rs.eyeTransforms.PositionLeft.Mul4(worldHmdPose)
		//hmdLocalView = rs.eyeTransforms.PositionLeft.Mul4(rs.hmdPose)
		perspective = rs.eyeTransforms.ProjectionLeft
		camera.View = worldView
		camera.Position = playerPosition
	} else {
		worldView = rs.eyeTransforms.PositionRight.Mul4(worldHmdPose)
		//hmdLocalView = rs.eyeTransforms.PositionRight.Mul4(rs.hmdPose)
		perspective = rs.eyeTransforms.ProjectionRight
		camera.View = worldView
		camera.Position = playerPosition
	}

	// draw stuff the visible entities
	for _, e := range rs.visibleEntities {
		visibleEntity, okay := e.(RenderableEntity)
		if okay {
			if r := visibleEntity.GetRenderable(); r != nil {
				rs.Renderer.DrawRenderable(r, nil, perspective, worldView, camera)
			}
		}
	}
}
