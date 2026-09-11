package app

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var _ Scene = (*DebuggerScene)(nil)

// DebuggerScene provides an interactive environment for executing and
// visually observing render and control tests.
type DebuggerScene struct {
	backBtnX    int
	backBtnY    int
	backBtnW    int
	backBtnH    int
	backHovered bool
}

// NewDebuggerScene initializes a new DebuggerScene instance.
func NewDebuggerScene() *DebuggerScene {
	return &DebuggerScene{
		backBtnX: 20,
		backBtnY: 50,
		backBtnW: 80,
		backBtnH: 30,
	}
}

// Update ticks scene logic, detecting mouse hover and click events on buttons.
func (s *DebuggerScene) Update() (Transition, error) {
	mx, my := ebiten.CursorPosition()
	s.backHovered = s.inBackBtn(mx, my)

	if s.backHovered && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return Transition{NextScene: NewStartMenuScene()}, nil
	}

	return Transition{}, nil
}

// Draw renders the debugger view to the screen.
func (s *DebuggerScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 20, G: 20, B: 24, A: 255})

	// Render title
	ebitenutil.DebugPrintAt(screen, "Debugger: Render/Control Tests", 20, 20)

	// Render "Back" button background
	btnColor := color.RGBA{R: 60, G: 60, B: 70, A: 255}
	if s.backHovered {
		btnColor = color.RGBA{R: 90, G: 110, B: 150, A: 255}
	}
	vector.DrawFilledRect(
		screen,
		float32(s.backBtnX),
		float32(s.backBtnY),
		float32(s.backBtnW),
		float32(s.backBtnH),
		btnColor,
		false,
	)

	// Render "Back" button label centered inside the button
	btnText := "Back"
	textX := s.backBtnX + (s.backBtnW-len(btnText)*6)/2
	textY := s.backBtnY + (s.backBtnH-16)/2
	ebitenutil.DebugPrintAt(screen, btnText, textX, textY)
}

func (s *DebuggerScene) inBackBtn(x, y int) bool {
	return x >= s.backBtnX && x <= s.backBtnX+s.backBtnW &&
		y >= s.backBtnY && y <= s.backBtnY+s.backBtnH
}
