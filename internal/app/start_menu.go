package app

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var _ Scene = (*StartMenuScene)(nil)

// menuButton represents a clickable UI button in the start menu.
type menuButton struct {
	label string
	x     int
	y     int
	w     int
	h     int
}

func (b *menuButton) contains(x, y int) bool {
	return x >= b.x && x <= b.x+b.w && y >= b.y && y <= b.y+b.h
}

// StartMenuScene is the main entry scene displaying the menu options.
type StartMenuScene struct {
	buttons    []menuButton
	hoveredIdx int
}

// NewStartMenuScene initializes the start menu scene and positions the buttons.
func NewStartMenuScene() *StartMenuScene {
	buttonLabels := []string{
		"Spar",
		"Duel",
		"Team Building",
		"Profile",
		"Options",
		"Debugger",
		"Quit",
	}

	const (
		screenWidth  = 800
		screenHeight = 600
		btnW         = 220
		btnH         = 36
		btnGap       = 12
	)

	totalH := len(buttonLabels)*btnH + (len(buttonLabels)-1)*btnGap
	startY := (screenHeight - totalH) / 2
	startX := (screenWidth - btnW) / 2

	buttons := make([]menuButton, len(buttonLabels))
	for i, label := range buttonLabels {
		buttons[i] = menuButton{
			label: label,
			x:     startX,
			y:     startY + i*(btnH+btnGap),
			w:     btnW,
			h:     btnH,
		}
	}

	return &StartMenuScene{
		buttons:    buttons,
		hoveredIdx: -1,
	}
}

// Update ticks the start menu scene, checking for mouse hover and click events.
func (s *StartMenuScene) Update() (Transition, error) {
	mx, my := ebiten.CursorPosition()

	s.hoveredIdx = -1
	for i := range s.buttons {
		if s.buttons[i].contains(mx, my) {
			s.hoveredIdx = i
			break
		}
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && s.hoveredIdx >= 0 {
		clicked := s.buttons[s.hoveredIdx].label
		log.Printf("[StartMenu] Button clicked: %s\n", clicked)
		return s.handleAction(clicked)
	}

	return Transition{}, nil
}

// handleAction processes actions triggered by menu button selection.
func (s *StartMenuScene) handleAction(label string) (Transition, error) {
	switch label {
	case "Quit":
		return Transition{Quit: true}, nil
	case "Spar", "Duel":
		battle, _ := NewBattleScene()
		return Transition{NextScene: battle}, nil
	case "Debugger":
		debug := NewDebuggerScene()
		return Transition{NextScene: debug}, nil
	default:
		// "Team Building", "Profile", "Options" do nothing
		return Transition{}, nil
	}
}

// Draw renders the start menu background, title, and buttons.
func (s *StartMenuScene) Draw(screen *ebiten.Image) {
	// Background fill
	screen.Fill(color.RGBA{R: 24, G: 28, B: 36, A: 255})

	// Title text
	title := "WILDBUDS"
	titleX := (800 - len(title)*6) / 2
	ebitenutil.DebugPrintAt(screen, title, titleX, 90)

	// Buttons
	for i, btn := range s.buttons {
		isHovered := i == s.hoveredIdx

		// Background rect
		bgColor := color.RGBA{R: 45, G: 52, B: 68, A: 255}
		borderColor := color.RGBA{R: 70, G: 80, B: 100, A: 255}
		if isHovered {
			bgColor = color.RGBA{R: 70, G: 95, B: 130, A: 255}
			borderColor = color.RGBA{R: 120, G: 160, B: 220, A: 255}
		}

		vector.DrawFilledRect(screen, float32(btn.x), float32(btn.y), float32(btn.w), float32(btn.h), bgColor, false)
		vector.StrokeRect(screen, float32(btn.x), float32(btn.y), float32(btn.w), float32(btn.h), 1.5, borderColor, false)

		// Center text inside button (debug font is ~6px wide per char, 16px high)
		textX := btn.x + (btn.w-len(btn.label)*6)/2
		textY := btn.y + (btn.h-16)/2
		ebitenutil.DebugPrintAt(screen, btn.label, textX, textY)
	}
}
