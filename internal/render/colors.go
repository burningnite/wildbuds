package render

import (
	"image/color"
)

// Palette defines the standard colors used in the Wildbuds renderer.
var (
	// Board Tiles
	ColorTileLight = color.RGBA{R: 200, G: 200, B: 200, A: 255}
	ColorTileDark  = color.RGBA{R: 150, G: 150, B: 150, A: 255}

	// Players
	ColorPlayerOne = color.RGBA{R: 50, G: 150, B: 250, A: 255} // Blue
	ColorPlayerTwo = color.RGBA{R: 250, G: 50, B: 50, A: 255}  // Red
	ColorPlayerNone = color.RGBA{R: 100, G: 100, B: 100, A: 255} // Gray

	// UI & Cursors
	ColorCursor    = color.RGBA{R: 255, G: 215, B: 0, A: 128} // Gold, semi-transparent
	ColorSelection = color.RGBA{R: 0, G: 255, B: 0, A: 255}   // Green outline
	ColorAttack    = color.RGBA{R: 255, G: 100, B: 0, A: 255} // Orange outline
)
