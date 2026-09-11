package e2e

import (
	"testing"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
)

func TestTier1_HappyPath(t *testing.T) {
	h, err := NewHarness()
	if err != nil {
		t.Fatalf("failed to create harness: %v", err)
	}

	// 1. Select P1 unit at (0, -3)
	h.InjectCommand(commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3}))
	if err := h.Tick(); err != nil {
		t.Fatalf("tick failed: %v", err)
	}

	// Verify unit selected
	if h.State.ActiveUnitID == nil {
		t.Fatalf("expected ActiveUnitID to be set after selection")
	}

	// 2. Move unit to (0, -2)
	h.InjectCommand(commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: -2}))
	if err := h.Tick(); err != nil {
		t.Fatalf("tick failed: %v", err)
	}

	// Assert unit moved to (0, -2)
	u := h.State.GetUnit(*h.State.ActiveUnitID)
	if u == nil {
		t.Fatalf("active unit not found")
	}
	expectedPos := domain.GridPosition{X: 0, Y: -2}
	if !u.Position.Equals(expectedPos) {
		t.Errorf("expected unit position to be %s, got %s", expectedPos, u.Position)
	}

	// 3. End activation
	h.InjectCommand(commands.NewEndActivationCommand())
	if err := h.Tick(); err != nil {
		t.Fatalf("tick failed: %v", err)
	}

	// Assert ActivePlayer is now PlayerTwo
	if h.State.ActivePlayer != domain.PlayerTwo {
		t.Errorf("expected ActivePlayer to be PlayerTwo, got %s", h.State.ActivePlayer)
	}
}
