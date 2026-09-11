package e2e

import (
	"testing"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
)

func TestTier4_RapidValidCommands(t *testing.T) {
	h, err := NewHarness()
	if err != nil {
		t.Fatalf("failed to create harness: %v", err)
	}

	// Inject 100 valid EndActivation commands in a loop without ticking
	const numCommands = 100
	for i := 0; i < numCommands; i++ {
		h.InjectCommand(commands.NewEndActivationCommand())
	}

	// Run Tick() in a loop 100 times
	for i := 0; i < numCommands; i++ {
		if err := h.Tick(); err != nil {
			t.Fatalf("tick %d failed: %v", i+1, err)
		}
	}

	// Assert no errors and ActivePlayer toggled appropriately
	// 100 EndActivations starting from PlayerOne toggles 100 times, ending back at PlayerOne.
	expectedPlayer := domain.PlayerOne
	if h.State.ActivePlayer != expectedPlayer {
		t.Errorf("expected active player %s after %d end activations, got %s", expectedPlayer, numCommands, h.State.ActivePlayer)
	}

	if h.State.Phase != domain.PhaseSelectUnit {
		t.Errorf("expected phase %s, got %s", domain.PhaseSelectUnit, h.State.Phase)
	}
}
