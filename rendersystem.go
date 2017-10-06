// Copyright 2017, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package main

import (
	glfw "github.com/go-gl/glfw/v3.1/glfw"
	mgl "github.com/go-gl/mathgl/mgl32"
	forward "github.com/tbogdala/fizzle/renderer/forward"
)

var (
	nearView  = float32(0.1)
	farView   = float32(300.0)
	fovyRads  = mgl.DegToRad(60.0)
	glSamples = 4
)

// RenderSystem is a common interface between VR and non-VR
// render systems.
type RenderSystem interface {
	GetRenderer() *forward.ForwardRenderer
	GetMainWindow() *glfw.Window
}
