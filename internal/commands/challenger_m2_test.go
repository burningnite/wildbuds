package commands_test

import (
	"encoding/json"
	"math"
	"sync"
	"testing"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
)

func TestChallenger_Commands_ExtremeCoordinatesValidation(t *testing.T) {
	extremeCoords := []domain.GridPosition{
		{X: math.MinInt32, Y: math.MaxInt32},
		{X: -6, Y: 0},
		{X: 6, Y: 0},
		{X: 0, Y: -6},
		{X: 0, Y: 6},
		{X: 100, Y: -100},
	}

	for _, pos := range extremeCoords {
		selBad := commands.NewSelectUnitCommand(pos)
		if err := selBad.Validate(); err != commands.ErrTargetOutOfBounds {
			t.Errorf("expected ErrTargetOutOfBounds for %v, got %v", pos, err)
		}

		moveBad := commands.NewMoveUnitCommand(pos)
		if err := moveBad.Validate(); err != commands.ErrDestOutOfBounds {
			t.Errorf("expected ErrDestOutOfBounds for %v, got %v", pos, err)
		}

		atkBad := commands.NewAttackCommand(pos)
		if err := atkBad.Validate(); err != commands.ErrTargetOutOfBounds {
			t.Errorf("expected ErrTargetOutOfBounds for %v, got %v", pos, err)
		}
	}
}

func TestChallenger_Commands_JSON_AdversarialInputs(t *testing.T) {
	adversarialPayloads := []struct {
		name    string
		payload string
	}{
		{name: "Corrupted_JSON_Truncated", payload: `{"type":"SelectUnit"`},
		{name: "Corrupted_JSON_Array", payload: `[1, 2, 3]`},
		{name: "Corrupted_JSON_Number", payload: `12345`},
		{name: "Corrupted_JSON_Null", payload: `null`},
		{name: "Corrupted_JSON_EmptyObject", payload: `{}`},
		{name: "Unknown_Command_Type", payload: `{"type":"NukeBoard"}`},
		{name: "MultiKey_ExternallyTagged", payload: `{"SelectUnit":{},"MoveUnit":{}}`},
		{name: "Unknown_ExternallyTagged", payload: `{"UnknownAction":{"x":1}}`},
		{name: "Invalid_Target_Format", payload: `{"type":"SelectUnit","target":"invalid"}`},
		{name: "Invalid_Destination_Format", payload: `{"type":"MoveUnit","destination":"not_a_pos"}`},
	}

	for _, tc := range adversarialPayloads {
		t.Run(tc.name, func(t *testing.T) {
			var cmd commands.GameCommand
			err := json.Unmarshal([]byte(tc.payload), &cmd)
			// Must return an error or produce an invalid command that fails Validate()
			if err == nil {
				if vErr := cmd.Validate(); vErr == nil {
					t.Errorf("expected invalid payload %s to fail unmarshal or validation, but passed: %+v", tc.payload, cmd)
				}
			}
		})
	}
}

func TestChallenger_NetworkCommand_ValidationEdgeCases(t *testing.T) {
	// Invalid sender PlayerNone
	ncNone := commands.NewNetworkCommand(domain.PlayerNone, commands.NewEndActivationCommand())
	if err := ncNone.Validate(); err != commands.ErrInvalidSender {
		t.Errorf("expected ErrInvalidSender, got %v", err)
	}

	// Invalid sender int out of range
	ncBogus := commands.NewNetworkCommand(domain.Player(42), commands.NewEndActivationCommand())
	if err := ncBogus.Validate(); err != commands.ErrInvalidSender {
		t.Errorf("expected ErrInvalidSender for Player(42), got %v", err)
	}

	// Valid sender, invalid inner command
	ncBadInner := commands.NewNetworkCommand(domain.PlayerOne, commands.NewMoveUnitCommand(domain.GridPosition{X: 6, Y: 0}))
	if err := ncBadInner.Validate(); err != commands.ErrDestOutOfBounds {
		t.Errorf("expected ErrDestOutOfBounds, got %v", err)
	}
}

func TestChallenger_Queue_ThreadSafetyStress(t *testing.T) {
	q := commands.NewCommandQueue()
	const numGoroutines = 20
	const opsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Launch concurrent writers to outgoing and incoming
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				q.PushOutgoing(commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: id % 5}))
			}
		}(i)

		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				q.PushIncoming(commands.NewNetworkCommand(domain.PlayerOne, commands.NewEndActivationCommand()))
			}
		}(i)
	}

	wg.Wait()

	if q.LenOutgoing() != numGoroutines*opsPerGoroutine {
		t.Errorf("expected %d outgoing, got %d", numGoroutines*opsPerGoroutine, q.LenOutgoing())
	}
	if q.LenIncoming() != numGoroutines*opsPerGoroutine {
		t.Errorf("expected %d incoming, got %d", numGoroutines*opsPerGoroutine, q.LenIncoming())
	}

	// Route outgoing to incoming concurrently
	routed := q.RouteOutgoingToIncoming(domain.PlayerTwo)
	if routed != numGoroutines*opsPerGoroutine {
		t.Errorf("expected %d routed, got %d", numGoroutines*opsPerGoroutine, routed)
	}
	if !q.IsOutgoingEmpty() {
		t.Errorf("outgoing should be empty after routing")
	}
	if q.LenIncoming() != 2*numGoroutines*opsPerGoroutine {
		t.Errorf("expected %d incoming after loopback, got %d", 2*numGoroutines*opsPerGoroutine, q.LenIncoming())
	}

	// Drain incoming
	drained := q.DrainIncoming()
	if len(drained) != 2*numGoroutines*opsPerGoroutine {
		t.Errorf("expected %d drained, got %d", 2*numGoroutines*opsPerGoroutine, len(drained))
	}
	if !q.IsEmpty() {
		t.Errorf("queue should be empty after draining all")
	}
}
