package e2e

import (
	"testing"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
)

func TestTier2_OutOfBoundsMove(t *testing.T) {
	h, err := NewHarness()
	if err != nil {
		t.Fatalf("failed to create harness: %v", err)
	}

	// Select P1 unit at (0, -3)
	initialPos := domain.GridPosition{X: 0, Y: -3}
	h.InjectCommand(commands.NewSelectUnitCommand(initialPos))
	if err := h.Tick(); err != nil {
		t.Fatalf("tick failed: %v", err)
	}

	if h.State.ActiveUnitID == nil {
		t.Fatalf("expected unit to be selected")
	}

	unitID := *h.State.ActiveUnitID

	// Try moving unit to (0, 6) which is out of bounds
	outOfBoundsPos := domain.GridPosition{X: 0, Y: 6}
	h.InjectCommand(commands.NewMoveUnitCommand(outOfBoundsPos))
	if err := h.Tick(); err != nil {
		t.Fatalf("tick failed: %v", err)
	}

	// Assert unit is still at original position (command rejected)
	u := h.State.GetUnit(unitID)
	if u == nil {
		t.Fatalf("unit not found")
	}
	if !u.Position.Equals(initialPos) {
		t.Errorf("expected unit position to remain %s (command rejected), but got %s", initialPos, u.Position)
	}
}
