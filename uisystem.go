// Copyright 2017, Timothy Bogdala <tdb@animal-machine.com>
// See the LICENSE file for more details.

package main

import (
	"fmt"

	gui "github.com/tbogdala/eweygewey"
	fonts "github.com/tbogdala/eweygewey/embeddedfonts"
	glfwinput "github.com/tbogdala/eweygewey/glfwinput"
	"github.com/tbogdala/fizzle"
	"github.com/tbogdala/fizzle/scene"
)

const (
	uiSystemPriority = 100.0
	uiSystemName     = "UserInterface"
)

var (
	fontScale  = 14
	fontGlyphs = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890., :[]{}\\|<>;\"'~`?/-+_=()*&^%$#@!"
)

// UISystem implements fizzle/scene/System interface and handles the rendering
// of the user interface.
type UISystem struct {
	uiman       *gui.Manager
	mainMenuWnd *gui.Window
	visible     bool
}

// NewUISystem allocates a new UISystem object.
func NewUISystem() *UISystem {
	s := new(UISystem)
	return s
}

// Initialize creates the user interface manager and gets it ready for rendering the scene
func (s *UISystem) Initialize(rs RenderSystem) error {
	// create the UI manager
	s.uiman = gui.NewManager(fizzle.GetGraphics())
	mainWin := rs.GetMainWindow()
	w, h := mainWin.GetSize()

	err := s.uiman.Initialize(gui.VertShader330, gui.FragShader330, int32(w), int32(h), int32(h))
	if err != nil {
		return fmt.Errorf("Failed to initialize the user interface! " + err.Error())
	}
	glfwinput.SetInputHandlers(s.uiman, mainWin)

	// load a font
	fontBytes, err := fonts.OswaldHeavyTtfBytes()
	if err != nil {
		return fmt.Errorf("Failed to load the embedded font: %v", err)
	}
	_, err = s.uiman.NewFontBytes("Default", fontBytes, fontScale, fontGlyphs)
	if err != nil {
		panic("Failed to load the font file! " + err.Error())
	}

	return nil
}

// SetVisible will control whether or not the user interface will draw on frame update.
func (s *UISystem) SetVisible(vis bool) {
	s.visible = vis
}

// ShowQuitMenu will render a window with a message prompting the user to replay or quit.
func (s *UISystem) ShowQuitMenu() {
	s.mainMenuWnd = s.uiman.NewWindow("Menu", 0.4, 0.6, 0.2, 0.25, func(wnd *gui.Window) {
		wnd.Text("GAME OVER")

		wnd.StartRow()
		wnd.Separator()

		wnd.StartRow()
		wnd.RequestItemWidthMin(.5)
		onQuit, _ := wnd.Button("QuitButton", "Quit")
		wnd.RequestItemWidthMin(.5)
		onReplay, _ := wnd.Button("ReplayButton", "Play Again")

		_ = onQuit
		_ = onReplay
	})

	s.mainMenuWnd.Title = "Menu"
	s.mainMenuWnd.ShowTitleBar = false
	s.mainMenuWnd.IsMoveable = false
	s.mainMenuWnd.AutoAdjustHeight = false
	s.mainMenuWnd.ShowScrollBar = false
	s.mainMenuWnd.IsScrollable = false
}

// Update should get called to run updates for the system every frame
// by the owning Manager object.
func (s *UISystem) Update(frameDelta float32) {
	// draw the user interface if visible
	if s.visible {
		gfx := fizzle.GetGraphics()
		width, height := s.uiman.GetResolution()
		gfx.Viewport(0, 0, int32(width), int32(height))

		s.uiman.Construct(float64(frameDelta))
		s.uiman.Draw()
	}
}

// OnAddEntity should get called by the scene Manager each time a new entity
// has been added to the scene.
func (s *UISystem) OnAddEntity(newEntity scene.Entity) {}

// OnRemoveEntity should get called by the scene Manager each time an entity
// has been removed from the scene.
func (s *UISystem) OnRemoveEntity(oldEntity scene.Entity) {}

// GetRequestedPriority returns the requested priority level for the System
// which may be of significance to a Manager if they want to order Update() calls.
func (s *UISystem) GetRequestedPriority() float32 { return uiSystemPriority }

// GetName returns the name of the system that can be used to identify
// the System within Manager.
func (s *UISystem) GetName() string { return uiSystemName }
