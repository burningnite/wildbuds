package e2e

import (
	"testing"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
)

func TestTier3_MoveAndEnd(t *testing.T) {
	h, err := NewHarness()
	if err != nil {
		t.Fatalf("failed to create harness: %v", err)
	}

	// 1. Select P1 unit at (0, -3)
	h.InjectCommand(commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3}))
	if err := h.Tick(); err != nil {
		t.Fatalf("tick after SelectUnit failed: %v", err)
	}

	// 2. Move P1 unit to (0, -2)
	h.InjectCommand(commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: -2}))
	if err := h.Tick(); err != nil {
		t.Fatalf("tick after MoveUnit failed: %v", err)
	}

	// 3. End activation
	h.InjectCommand(commands.NewEndActivationCommand())
	if err := h.Tick(); err != nil {
		t.Fatalf("tick after EndActivation failed: %v", err)
	}

	// Assert state is PhaseSelectUnit and ActivePlayer is PlayerTwo
	if h.State.Phase != domain.PhaseSelectUnit {
		t.Errorf("expected phase %s, got %s", domain.PhaseSelectUnit, h.State.Phase)
	}
	if h.State.ActivePlayer != domain.PlayerTwo {
		t.Errorf("expected active player %s, got %s", domain.PlayerTwo, h.State.ActivePlayer)
	}
}
