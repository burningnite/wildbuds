package app

import (
	"image/color"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
	"wildbuds/internal/input"
	"wildbuds/internal/render"
	"wildbuds/internal/resolver"
	"wildbuds/internal/transport"
	"wildbuds/internal/assets"
	"wildbuds/internal/i18n"
)

// BattleScene represents the main gameplay state.
type BattleScene struct {
	state     *domain.GameState
	queue     *commands.CommandQueue
	transport transport.Transport
	camera    *render.Camera
	input     *input.InputHandler
	resolver  *resolver.Resolver
	backBtnX  int
	backBtnY  int
	backBtnW  int
	backBtnH  int
	backHover bool
}

// NewBattleScene initializes the game mechanics and systems.
func NewBattleScene(t transport.Transport) (*BattleScene, error) {
	state := domain.NewInitialGameState()
	queue := commands.NewCommandQueue()

	var trans transport.Transport
	var err error
	if t != nil {
		trans = t
	} else {
		trans, err = transport.NewLoopback(
			domain.PlayerOne,
			transport.WithAutoPlayerToggle(true),
		)
		if err != nil {
			return nil, err
		}
	}

	camera := render.NewCamera(800, 600)
	inputHandler := input.NewInputHandler(queue, camera)
	gameResolver := resolver.NewResolver()

	return &BattleScene{
		state:     state,
		queue:     queue,
		transport: trans,
		camera:    camera,
		input:     inputHandler,
		resolver:  gameResolver,
		backBtnX:  20,
		backBtnY:  540,
		backBtnW:  40,
		backBtnH:  40,
	}, nil
}

// Update ticks the battle logic.
func (s *BattleScene) Update() (Transition, error) {
	mx, my := ebiten.CursorPosition()
	s.backHover = s.inBackBtn(mx, my)

	// Listen for ESC key or back button click
	if ebiten.IsKeyPressed(ebiten.KeyEscape) || (s.backHover && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)) {
		if s.transport != nil {
			s.transport.Close()
		}
		return Transition{NextScene: NewStartMenuScene()}, nil
	}

	s.input.Update(s.state)

	_, _, err := transport.Pump(s.transport, s.queue)
	if err != nil {
		return Transition{}, err
	}

	s.resolver.Resolve(s.state, s.queue)

	return Transition{}, nil // Stay in this scene
}

// Draw renders the battle state.
func (s *BattleScene) Draw(screen *ebiten.Image) {
	render.DrawBoard(screen, s.camera)
	render.DrawUnits(screen, s.camera, s.state)
	render.DrawCursor(screen, s.camera, s.input.CursorX, s.input.CursorY)
	s.drawBackButton(screen)
}

func (s *BattleScene) drawBackButton(screen *ebiten.Image) {
	importColor := color.RGBA{R: 60, G: 60, B: 70, A: 255}
	if s.backHover {
		importColor = color.RGBA{R: 90, G: 110, B: 150, A: 255}
	}
	vector.DrawFilledRect(
		screen,
		float32(s.backBtnX),
		float32(s.backBtnY),
		float32(s.backBtnW),
		float32(s.backBtnH),
		importColor,
		false,
	)

	font20 := assets.GetFont(20)
	btnText := i18n.Get("back_symbol")
	w, h := text.Measure(btnText, font20, 0)
	textX := float64(s.backBtnX) + (float64(s.backBtnW)-w)/2
	textY := float64(s.backBtnY) + (float64(s.backBtnH)-h)/2
	btnOpts := &text.DrawOptions{}
	btnOpts.GeoM.Translate(textX, textY)
	btnOpts.ColorScale.ScaleWithColor(color.White)
	text.Draw(screen, btnText, font20, btnOpts)
}

func (s *BattleScene) inBackBtn(x, y int) bool {
	return x >= s.backBtnX && x <= s.backBtnX+s.backBtnW &&
		y >= s.backBtnY && y <= s.backBtnY+s.backBtnH
}
