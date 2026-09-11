package app

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"wildbuds/internal/assets"
	"wildbuds/internal/i18n"
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
	buttons     []menuButton
	langButton  menuButton
	hoveredIdx  int
	langHovered bool
}

// NewStartMenuScene initializes the start menu scene and positions the buttons.
func NewStartMenuScene() *StartMenuScene {
	buttonLabels := []string{
		i18n.Get("spar"),
		i18n.Get("duel"),
		i18n.Get("team_building"),
		i18n.Get("profile"),
		i18n.Get("options"),
		i18n.Get("debugger"),
		i18n.Get("quit"),
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

	langButton := menuButton{
		label: i18n.Get("lang_menu"),
		x:     740,
		y:     540,
		w:     40,
		h:     40,
	}

	return &StartMenuScene{
		buttons:     buttons,
		langButton:  langButton,
		hoveredIdx:  -1,
		langHovered: false,
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

	s.langHovered = s.langButton.contains(mx, my)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if s.langHovered {
			log.Println("[StartMenu] Language button clicked")
			return Transition{NextScene: NewLanguageScene()}, nil
		}
		if s.hoveredIdx >= 0 {
			clicked := s.buttons[s.hoveredIdx].label
			log.Printf("[StartMenu] Button clicked: %s\n", clicked)
			return s.handleAction(clicked)
		}
	}

	return Transition{}, nil
}

// handleAction processes actions triggered by menu button selection.
func (s *StartMenuScene) handleAction(label string) (Transition, error) {
	switch label {
	case i18n.Get("quit"), "Quit", "quit":
		return Transition{Quit: true}, nil
	case i18n.Get("spar"), "Spar":
		battle, err := NewBattleScene(nil)
		if err != nil {
			return Transition{}, err
		}
		return Transition{NextScene: battle}, nil
	case i18n.Get("duel"), "Duel":
		lobby, err := NewLobbyScene()
		if err != nil {
			return Transition{}, err
		}
		return Transition{NextScene: lobby}, nil
	case i18n.Get("debugger"), "Debugger", "debugger":
		debug := NewDebuggerScene()
		return Transition{NextScene: debug}, nil
	case i18n.Get("lang_menu"), "lang_menu", "L":
		return Transition{NextScene: NewLanguageScene()}, nil
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
	face := assets.GetFont(20)
	w, _ := text.Measure(title, face, 0)
	titleX := (800 - w) / 2
	op := &text.DrawOptions{}
	op.GeoM.Translate(titleX, 90)
	text.Draw(screen, title, face, op)

	// Buttons
	for i, btn := range s.buttons {
		drawButton(screen, btn, i == s.hoveredIdx)
	}

	// Language menu button
	drawButton(screen, s.langButton, s.langHovered)
}

func drawButton(screen *ebiten.Image, btn menuButton, isHovered bool) {
	bgColor := color.RGBA{R: 45, G: 52, B: 68, A: 255}
	borderColor := color.RGBA{R: 70, G: 80, B: 100, A: 255}
	if isHovered {
		bgColor = color.RGBA{R: 70, G: 95, B: 130, A: 255}
		borderColor = color.RGBA{R: 120, G: 160, B: 220, A: 255}
	}

	vector.DrawFilledRect(screen, float32(btn.x), float32(btn.y), float32(btn.w), float32(btn.h), bgColor, false)
	vector.StrokeRect(screen, float32(btn.x), float32(btn.y), float32(btn.w), float32(btn.h), 1.5, borderColor, false)

	face := assets.GetFont(20)
	w, h := text.Measure(btn.label, face, 0)
	textX := float64(btn.x) + (float64(btn.w)-w)/2
	textY := float64(btn.y) + (float64(btn.h)-h)/2
	op := &text.DrawOptions{}
	op.GeoM.Translate(textX, textY)
	text.Draw(screen, btn.label, face, op)
}
