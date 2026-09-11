package app

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"wildbuds/internal/assets"
	"wildbuds/internal/i18n"
)

var _ Scene = (*LanguageScene)(nil)

type langButton struct {
	label string
	x     int
	y     int
	w     int
	h     int
}

func (b *langButton) contains(x, y int) bool {
	return x >= b.x && x <= b.x+b.w && y >= b.y && y <= b.y+b.h
}

// LanguageScene renders the Minecraft-style language selection menu.
type LanguageScene struct {
	buttons    []langButton
	hoveredIdx int
	backBtn    langButton
	backHover  bool
}

// NewLanguageScene initializes the language scene with two columns of language buttons centered on 800x600.
func NewLanguageScene() *LanguageScene {
	languages := []string{
		"English",
		"Español",
		"Français",
		"Deutsch",
		"Italiano",
		"Português",
	}

	const (
		screenWidth  = 800
		screenHeight = 600
		btnW         = 240
		btnH         = 36
		colGap       = 20
		rowGap       = 12
	)

	numCols := 2
	numRows := (len(languages) + numCols - 1) / numCols

	totalW := numCols*btnW + (numCols-1)*colGap
	totalH := numRows*btnH + (numRows-1)*rowGap

	startX := (screenWidth - totalW) / 2
	startY := (screenHeight - totalH) / 2

	buttons := make([]langButton, len(languages))
	for i, lang := range languages {
		col := i % numCols
		row := i / numCols
		buttons[i] = langButton{
			label: lang,
			x:     startX + col*(btnW+colGap),
			y:     startY + row*(btnH+rowGap),
			w:     btnW,
			h:     btnH,
		}
	}

	return &LanguageScene{
		buttons:    buttons,
		hoveredIdx: -1,
		backBtn: langButton{
			label: i18n.Get("back_symbol"),
			x:     20,
			y:     540,
			w:     40,
			h:     40,
		},
	}
}

// Update ticks the language scene, detecting hover and clicks on language buttons.
func (s *LanguageScene) Update() (Transition, error) {
	mx, my := ebiten.CursorPosition()

	s.hoveredIdx = -1
	for i := range s.buttons {
		if s.buttons[i].contains(mx, my) {
			s.hoveredIdx = i
			break
		}
	}

	s.backHover = s.backBtn.contains(mx, my)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if s.hoveredIdx >= 0 {
			clicked := s.buttons[s.hoveredIdx].label
			i18n.CurrentLanguage = clicked
			return Transition{NextScene: NewStartMenuScene()}, nil
		}
		if s.backHover {
			return Transition{NextScene: NewStartMenuScene()}, nil
		}
	}

	return Transition{}, nil
}

// Draw renders the background and language selection buttons with text.Draw.
func (s *LanguageScene) Draw(screen *ebiten.Image) {
	// Background fill
	screen.Fill(color.RGBA{R: 24, G: 28, B: 36, A: 255})

	for i, btn := range s.buttons {
		isHovered := i == s.hoveredIdx
		s.drawButton(screen, btn, isHovered)
	}

	// Draw back button
	s.drawButton(screen, s.backBtn, s.backHover)
}

func (s *LanguageScene) drawButton(screen *ebiten.Image, btn langButton, isHovered bool) {
	bgColor := color.RGBA{R: 45, G: 52, B: 68, A: 255}
	borderColor := color.RGBA{R: 70, G: 80, B: 100, A: 255}
	if isHovered {
		bgColor = color.RGBA{R: 70, G: 95, B: 130, A: 255}
		borderColor = color.RGBA{R: 120, G: 160, B: 220, A: 255}
	}

	vector.DrawFilledRect(screen, float32(btn.x), float32(btn.y), float32(btn.w), float32(btn.h), bgColor, false)
	vector.StrokeRect(screen, float32(btn.x), float32(btn.y), float32(btn.w), float32(btn.h), 1.5, borderColor, false)

	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(btn.x)+float64(btn.w)/2, float64(btn.y)+float64(btn.h)/2)
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter
	op.ColorScale.ScaleWithColor(color.White)

	text.Draw(screen, btn.label, assets.GetFont(16), op)
}
