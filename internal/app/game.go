package app

import (
	"errors"
	"github.com/hajimehoshi/ebiten/v2"
)

// Game is the main application struct implementing ebiten.Game.
type Game struct {
	activeScene Scene
}

// NewGame initializes the application and sets the initial scene.
func NewGame(initialScene Scene) (*Game, error) {
	return &Game{
		activeScene: initialScene,
	}, nil
}

// Update proceeds the game state by one tick.
func (g *Game) Update() error {
	if g.activeScene == nil {
		return errors.New("no active scene")
	}

	transition, err := g.activeScene.Update()
	if err != nil {
		return err
	}

	if transition.Quit {
		return ebiten.Termination
	}

	if transition.NextScene != nil {
		g.activeScene = transition.NextScene
	}

	return nil
}

// Draw renders the active scene to the screen.
func (g *Game) Draw(screen *ebiten.Image) {
	if g.activeScene != nil {
		g.activeScene.Draw(screen)
	}
}

// Layout takes the outside size and returns the logical screen size.
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 800, 600
}
