package render

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"wildbuds/internal/domain"
)

// DrawBoard renders the 11x11 checkerboard grid onto the screen.
func DrawBoard(screen *ebiten.Image, cam *Camera) {
	for x := domain.GridMin; x <= domain.GridMax; x++ {
		for y := domain.GridMin; y <= domain.GridMax; y++ {
			gp := domain.GridPosition{X: x, Y: y}
			
			// Determine tile color based on parity
			tileColor := ColorTileLight
			if gp.IsDarkTile() {
				tileColor = ColorTileDark
			}

			// Calculate bounds
			px, py, w, h := cam.BoundsRect(x, y, domain.TileVisualSize)
			
			// Draw filled rect
			vector.DrawFilledRect(screen, float32(px), float32(py), float32(w), float32(h), tileColor, false)
		}
	}
}

// DrawUnits renders all units in the game state.
func DrawUnits(screen *ebiten.Image, cam *Camera, state *domain.GameState) {
	if state == nil {
		return
	}
	
	for _, unit := range state.Units {
		// Determine unit color based on owner
		unitColor := ColorPlayerNone
		if unit.Owner == domain.PlayerOne {
			unitColor = ColorPlayerOne
		} else if unit.Owner == domain.PlayerTwo {
			unitColor = ColorPlayerTwo
		}

		px, py, w, h := cam.BoundsRect(unit.Position.X, unit.Position.Y, domain.UnitVisualSize)
		vector.DrawFilledRect(screen, float32(px), float32(py), float32(w), float32(h), unitColor, false)
	}
}

// DrawCursor renders the active player's cursor or hover highlight.
func DrawCursor(screen *ebiten.Image, cam *Camera, cursorX, cursorY int) {
	px, py, w, h := cam.BoundsRect(cursorX, cursorY, domain.CursorVisualSize)
	
	// Draw a slightly larger, semi-transparent highlight behind/around the tile
	vector.DrawFilledRect(screen, float32(px), float32(py), float32(w), float32(h), ColorCursor, false)
}
