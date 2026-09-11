package render

import (
	"testing"
)

func TestCamera_BijectiveMapping(t *testing.T) {
	cam := NewCamera(800, 600)
	// Base scale
	verifyBijectiveMapping(t, cam)

	// Zoomed in
	cam.Scale = 2.0
	verifyBijectiveMapping(t, cam)

	// Zoomed out
	cam.Scale = 0.5
	verifyBijectiveMapping(t, cam)
}

func verifyBijectiveMapping(t *testing.T, cam *Camera) {
	// Test the bounds of the grid
	for x := -10; x <= 10; x++ {
		for y := -10; y <= 10; y++ {
			sx, sy := cam.GridToScreen(x, y)
			rx, ry := cam.ScreenToGrid(sx, sy)

			if rx != x || ry != y {
				t.Errorf("Bijective mapping failed at scale %f: (%d, %d) -> (%f, %f) -> (%d, %d)", 
					cam.Scale, x, y, sx, sy, rx, ry)
			}
		}
	}
}

func TestCamera_ClickTolerance(t *testing.T) {
	cam := NewCamera(800, 600)
	
	// Origin is (400, 300)
	// Grid (0, 0) is at (400, 300)
	
	// Clicking slightly off center should still snap to (0,0)
	rx, ry := cam.ScreenToGrid(405.0, 305.0)
	if rx != 0 || ry != 0 {
		t.Errorf("Expected click near origin to map to (0,0), got (%d, %d)", rx, ry)
	}

	// Grid (1, 1) is at (400 + 32, 300 - 32) = (432, 268)
	rx, ry = cam.ScreenToGrid(430.0, 270.0)
	if rx != 1 || ry != 1 {
		t.Errorf("Expected click near (1,1) to map to (1,1), got (%d, %d)", rx, ry)
	}
}
