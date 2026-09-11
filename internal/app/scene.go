package app

import "github.com/hajimehoshi/ebiten/v2"

// Transition handles passing control between scenes.
// If NextScene is non-nil, the Game will switch to it.
type Transition struct {
	NextScene Scene
	Quit      bool
}

// Scene represents a distinct application state (e.g., Main Menu, Battle, Debugger).
type Scene interface {
	// Update ticks the scene logic. It returns a Transition to indicate state changes.
	Update() (Transition, error)
	// Draw renders the scene to the screen.
	Draw(screen *ebiten.Image)
}
