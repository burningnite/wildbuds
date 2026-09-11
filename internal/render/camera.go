package render

import (
	"math"
	"wildbuds/internal/domain"
)

// Camera transforms between mathematical grid coordinates and 2D screen pixels.
// It assumes the origin (0, 0) of the grid is rendered at (OriginX, OriginY) on screen,
// with +Y pointing UP in grid space (unlike screen space where +Y points down).
type Camera struct {
	OriginX float64
	OriginY float64
	Scale   float64
}

// NewCamera initializes a Camera centered on the given screen dimensions.
func NewCamera(screenWidth, screenHeight int) *Camera {
	return &Camera{
		OriginX: float64(screenWidth) / 2.0,
		OriginY: float64(screenHeight) / 2.0,
		Scale:   1.0,
	}
}

// GridToScreen converts a discrete grid coordinate into an absolute pixel coordinate.
// It applies the TilePitch to separate tiles and flips the Y-axis so +Y goes up.
func (c *Camera) GridToScreen(gx, gy int) (float64, float64) {
	// Screen +X is Right, Grid +X is Right
	screenX := c.OriginX + float64(gx)*domain.TilePitch*c.Scale

	// Screen +Y is Down, Grid +Y is Up
	screenY := c.OriginY - float64(gy)*domain.TilePitch*c.Scale

	return screenX, screenY
}

// ScreenToGrid converts a screen pixel coordinate into the closest discrete grid coordinate.
func (c *Camera) ScreenToGrid(sx, sy float64) (int, int) {
	// Reverse the GridToScreen transform
	// sx = OriginX + gx * Pitch * Scale => gx = (sx - OriginX) / (Pitch * Scale)
	gxFloat := (sx - c.OriginX) / (domain.TilePitch * c.Scale)
	
	// sy = OriginY - gy * Pitch * Scale => gy = (OriginY - sy) / (Pitch * Scale)
	gyFloat := (c.OriginY - sy) / (domain.TilePitch * c.Scale)

	// Round to nearest integer (math.Round handles negative properly)
	gx := int(math.Round(gxFloat))
	gy := int(math.Round(gyFloat))

	return gx, gy
}

// BoundsRect returns the top-left (x, y) and dimensions (w, h) for drawing a tile
// centered at the given GridPosition.
func (c *Camera) BoundsRect(gx, gy int, visualSize float64) (float64, float64, float64, float64) {
	cx, cy := c.GridToScreen(gx, gy)
	scaledSize := visualSize * c.Scale
	
	// Ebiten rectangles are drawn from top-left, so we offset by half the size
	topLeftX := cx - scaledSize/2.0
	topLeftY := cy - scaledSize/2.0
	
	return topLeftX, topLeftY, scaledSize, scaledSize
}
