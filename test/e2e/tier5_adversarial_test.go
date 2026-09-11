package e2e

import (
	"testing"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
)

func TestTier5_OutOfTurnInjection(t *testing.T) {
	h, err := NewHarness()
	if err != nil {
		t.Fatalf("failed to create harness: %v", err)
	}

	// P1 is active initially
	if h.State.ActivePlayer != domain.PlayerOne {
		t.Fatalf("expected ActivePlayer to be PlayerOne, got %s", h.State.ActivePlayer)
	}

	// Inject an adversarial payload for P2 using domain.NetworkCommand
	h.InjectAdversarialPayload(domain.NetworkCommand{
		Sender:  domain.PlayerTwo,
		Payload: commands.NewEndActivationCommand(),
	})

	// Tick the harness
	if err := h.Tick(); err != nil {
		t.Fatalf("tick failed: %v", err)
	}

	// Assert ActivePlayer is STILL PlayerOne (the command should be rejected)
	if h.State.ActivePlayer != domain.PlayerOne {
		t.Errorf("expected ActivePlayer to remain PlayerOne, got %s", h.State.ActivePlayer)
	}

	if err := h.State.Validate(); err != nil {
		t.Errorf("state validation failed: %v", err)
	}
}

func TestTier5_ControlEnemyUnit(t *testing.T) {
	h, err := NewHarness()
	if err != nil {
		t.Fatalf("failed to create harness: %v", err)
	}

	// P1 is active. Inject an adversarial payload for P1 trying to select P2's unit at (0, 3).
	h.InjectAdversarialPayload(domain.NetworkCommand{
		Sender:  domain.PlayerOne,
		Payload: commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: 3}),
	})

	// Tick the harness
	if err := h.Tick(); err != nil {
		t.Fatalf("tick failed: %v", err)
	}

	// Assert ActiveUnitID is nil (command rejected)
	if h.State.ActiveUnitID != nil {
		t.Errorf("expected ActiveUnitID to be nil, got %v", *h.State.ActiveUnitID)
	}

	if err := h.State.Validate(); err != nil {
		t.Errorf("state validation failed: %v", err)
	}
}
