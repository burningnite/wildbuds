package app

import (
	"github.com/hajimehoshi/ebiten/v2"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
	"wildbuds/internal/input"
	"wildbuds/internal/render"
	"wildbuds/internal/resolver"
	"wildbuds/internal/transport"
)

// BattleScene represents the main gameplay state.
type BattleScene struct {
	state     *domain.GameState
	queue     *commands.CommandQueue
	transport transport.Transport
	camera    *render.Camera
	input     *input.InputHandler
	resolver  *resolver.Resolver
}

// NewBattleScene initializes the game mechanics and systems.
func NewBattleScene() (*BattleScene, error) {
	state := domain.NewInitialGameState()
	queue := commands.NewCommandQueue()
	
	trans, err := transport.NewLoopback(
		domain.PlayerOne,
		transport.WithAutoPlayerToggle(true),
	)
	if err != nil {
		return nil, err
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
	}, nil
}

// Update ticks the battle logic.
func (s *BattleScene) Update() (Transition, error) {
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
}
