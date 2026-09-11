package resolver_test

import (
	"math"
	"math/rand"
	"reflect"
	"testing"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
	"wildbuds/internal/resolver"
)

// ============================================================================
// 1. Poison-Pill Injection: Comprehensive Interleaved Game Script
// ============================================================================

func TestAdversarial_PoisonPill_ComprehensiveInterleavedScript(t *testing.T) {
	state := domain.NewInitialGameState()
	r := resolver.NewResolver()
	q := commands.NewCommandQueue()

	// Initial State:
	// - ActivePlayer: PlayerOne
	// - Phase: PhaseSelectUnit
	// - ActiveUnitID: nil
	// - Unit 1: P1 at (0, -3), HP 100
	// - Unit 2: P2 at (0, 3), HP 100

	type step struct {
		name         string
		cmd          commands.NetworkCommand
		expectReject bool
		rejectReason string
	}

	steps := []step{
		// --- Phase 1: SelectUnit Stage ---
		// Poison 1: Sender is PlayerNone
		{
			name:         "Poison_SenderPlayerNone",
			cmd:          commands.NewNetworkCommand(domain.PlayerNone, commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3})),
			expectReject: true,
			rejectReason: resolver.ReasonNotActivePlayer,
		},
		// Poison 2: Sender is PlayerTwo (wrong player)
		{
			name:         "Poison_WrongPlayer_SelectUnit",
			cmd:          commands.NewNetworkCommand(domain.PlayerTwo, commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: 3})),
			expectReject: true,
			rejectReason: resolver.ReasonNotActivePlayer,
		},
		// Poison 3: Sender is PlayerTwo trying to end activation
		{
			name:         "Poison_WrongPlayer_EndActivation",
			cmd:          commands.NewNetworkCommand(domain.PlayerTwo, commands.NewEndActivationCommand()),
			expectReject: true,
			rejectReason: resolver.ReasonNotActivePlayer,
		},
		// Poison 4: Active player selects out-of-bounds coordinate
		{
			name:         "Poison_ActivePlayer_SelectOutOfBounds_MaxInt",
			cmd:          commands.NewNetworkCommand(domain.PlayerOne, commands.NewSelectUnitCommand(domain.GridPosition{X: math.MaxInt32, Y: 0})),
			expectReject: true,
			rejectReason: resolver.ReasonTargetOutOfBounds,
		},
		// Poison 5: Active player selects out-of-bounds coordinate (-6, 0)
		{
			name:         "Poison_ActivePlayer_SelectOutOfBounds_Negative",
			cmd:          commands.NewNetworkCommand(domain.PlayerOne, commands.NewSelectUnitCommand(domain.GridPosition{X: -6, Y: 0})),
			expectReject: true,
			rejectReason: resolver.ReasonTargetOutOfBounds,
		},
		// Poison 6: Active player selects empty board tile (0, 0)
		{
			name:         "Poison_ActivePlayer_SelectEmptyTile",
			cmd:          commands.NewNetworkCommand(domain.PlayerOne, commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: 0})),
			expectReject: true,
			rejectReason: resolver.ReasonNoUnitAtTarget,
		},
		// Poison 7: Active player selects opponent unit at (0, 3)
		{
			name:         "Poison_ActivePlayer_SelectOpponentUnit",
			cmd:          commands.NewNetworkCommand(domain.PlayerOne, commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: 3})),
			expectReject: true,
			rejectReason: resolver.ReasonUnitNotOwned,
		},
		// Poison 8: Unknown command type from active player
		{
			name: "Poison_ActivePlayer_UnknownCommandType",
			cmd: commands.NewNetworkCommand(domain.PlayerOne, commands.GameCommand{
				Type:   "HACK_SUPER_KILL",
				Target: domain.GridPosition{X: 0, Y: 3},
			}),
			expectReject: true,
			rejectReason: resolver.ReasonUnknownCommand,
		},

		// Valid Step 1: PlayerOne selects own unit 1 at (0, -3)
		{
			name:         "Valid_SelectOwnUnit",
			cmd:          commands.NewNetworkCommand(domain.PlayerOne, commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3})),
			expectReject: false,
		},

		// --- Phase 2: Action Stage (Unit 1 active, PhaseChooseAction) ---
		// Poison 9: Wrong player tries to move unit
		{
			name:         "Poison_WrongPlayer_MoveDuringActionPhase",
			cmd:          commands.NewNetworkCommand(domain.PlayerTwo, commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: 2})),
			expectReject: true,
			rejectReason: resolver.ReasonNotActivePlayer,
		},
		// Poison 10: Active player moves out of bounds (0, -6)
		{
			name:         "Poison_MoveOutOfBounds",
			cmd:          commands.NewNetworkCommand(domain.PlayerOne, commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: -6})),
			expectReject: true,
			rejectReason: resolver.ReasonDestOutOfBounds,
		},
		// Poison 11: Active player moves into occupied tile (Unit 2 at 0, 3)
		{
			name:         "Poison_MoveToOccupiedTile",
			cmd:          commands.NewNetworkCommand(domain.PlayerOne, commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: 3})),
			expectReject: true,
			rejectReason: resolver.ReasonDestinationOccupied,
		},
		// Poison 12: Active player attacks out of bounds (10, 10)
		{
			name:         "Poison_AttackOutOfBounds",
			cmd:          commands.NewNetworkCommand(domain.PlayerOne, commands.NewAttackCommand(domain.GridPosition{X: 10, Y: 10})),
			expectReject: true,
			rejectReason: resolver.ReasonTargetOutOfBounds,
		},
		// Poison 13: Active player attacks empty tile (1, 1)
		{
			name:         "Poison_AttackEmptyTile",
			cmd:          commands.NewNetworkCommand(domain.PlayerOne, commands.NewAttackCommand(domain.GridPosition{X: 1, Y: 1})),
			expectReject: true,
			rejectReason: resolver.ReasonNoUnitAtTarget,
		},

		// Valid Step 2: Active player moves Unit 1 from (0, -3) to (0, -2)
		{
			name:         "Valid_MoveUnit1",
			cmd:          commands.NewNetworkCommand(domain.PlayerOne, commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: -2})),
			expectReject: false,
		},

		// Poison 14: Move to occupied tile again (0, 3)
		{
			name:         "Poison_MoveToOccupiedAgain",
			cmd:          commands.NewNetworkCommand(domain.PlayerOne, commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: 3})),
			expectReject: true,
			rejectReason: resolver.ReasonDestinationOccupied,
		},

		// Valid Step 3: Attack Unit 2 at (0, 3)
		{
			name:         "Valid_AttackUnit2",
			cmd:          commands.NewNetworkCommand(domain.PlayerOne, commands.NewAttackCommand(domain.GridPosition{X: 0, Y: 3})),
			expectReject: false,
		},

		// Poison 15: PlayerTwo tries to end activation while it's still PlayerOne's turn
		{
			name:         "Poison_WrongPlayer_PrematureEndActivation",
			cmd:          commands.NewNetworkCommand(domain.PlayerTwo, commands.NewEndActivationCommand()),
			expectReject: true,
			rejectReason: resolver.ReasonNotActivePlayer,
		},

		// Valid Step 4: PlayerOne ends activation -> turn passes to PlayerTwo
		{
			name:         "Valid_PlayerOne_EndActivation",
			cmd:          commands.NewNetworkCommand(domain.PlayerOne, commands.NewEndActivationCommand()),
			expectReject: false,
		},

		// --- Phase 3: PlayerTwo Turn ---
		// Poison 16: PlayerOne tries to act immediately after ending turn
		{
			name:         "Poison_PlayerOne_ActAfterTurnEnded",
			cmd:          commands.NewNetworkCommand(domain.PlayerOne, commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -2})),
			expectReject: true,
			rejectReason: resolver.ReasonNotActivePlayer,
		},
		// Poison 17: PlayerTwo selects PlayerOne's unit at (0, -2)
		{
			name:         "Poison_PlayerTwo_SelectOpponentUnit",
			cmd:          commands.NewNetworkCommand(domain.PlayerTwo, commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -2})),
			expectReject: true,
			rejectReason: resolver.ReasonUnitNotOwned,
		},

		// Valid Step 5: PlayerTwo selects own Unit 2 at (0, 3)
		{
			name:         "Valid_PlayerTwo_SelectUnit2",
			cmd:          commands.NewNetworkCommand(domain.PlayerTwo, commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: 3})),
			expectReject: false,
		},

		// Valid Step 6: PlayerTwo moves Unit 2 to (0, 2)
		{
			name:         "Valid_PlayerTwo_MoveUnit2",
			cmd:          commands.NewNetworkCommand(domain.PlayerTwo, commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: 2})),
			expectReject: false,
		},

		// Valid Step 7: PlayerTwo ends activation -> turn passes back to PlayerOne
		{
			name:         "Valid_PlayerTwo_EndActivation",
			cmd:          commands.NewNetworkCommand(domain.PlayerTwo, commands.NewEndActivationCommand()),
			expectReject: false,
		},
	}

	// Enqueue all commands into the queue
	for _, s := range steps {
		q.PushIncoming(s.cmd)
	}

	if q.LenIncoming() != len(steps) {
		t.Fatalf("queue length mismatch: expected %d, got %d", len(steps), q.LenIncoming())
	}

	// Execute through the authoritative resolver loop
	events := r.Resolve(state, q)

	if !q.IsEmpty() {
		t.Errorf("queue not empty after resolution: %d remaining", q.LenIncoming())
	}

	// Verify events match expectations
	evIdx := 0
	for stepIdx, s := range steps {
		if evIdx >= len(events) {
			t.Fatalf("step %d (%s): ran out of events (got %d total)", stepIdx, s.name, len(events))
		}

		ev := events[evIdx]
		if s.expectReject {
			if ev.Type != resolver.EventCommandRejected {
				t.Fatalf("step %d (%s): expected rejection event, got %s", stepIdx, s.name, ev.Type)
			}
			if ev.Reason != s.rejectReason {
				t.Errorf("step %d (%s): reason mismatch: got %q, want %q", stepIdx, s.name, ev.Reason, s.rejectReason)
			}
			evIdx++
		} else {
			if ev.Type == resolver.EventCommandRejected {
				t.Fatalf("step %d (%s): valid command unexpectedly rejected: reason=%s", stepIdx, s.name, ev.Reason)
			}
			evIdx++
		}
	}

	// Verify final state integrity
	if state.ActivePlayer != domain.PlayerOne {
		t.Errorf("final ActivePlayer mismatch: expected PlayerOne, got %s", state.ActivePlayer)
	}
	if state.Phase != domain.PhaseSelectUnit {
		t.Errorf("final Phase mismatch: expected PhaseSelectUnit, got %s", state.Phase)
	}
	if state.ActiveUnitID != nil {
		t.Errorf("final ActiveUnitID should be nil, got %v", state.ActiveUnitID)
	}

	// Unit 1 should be at (0, -2)
	u1 := state.GetUnit(1)
	if !u1.Position.Equals(domain.GridPosition{X: 0, Y: -2}) {
		t.Errorf("Unit 1 final position mismatch: expected (0, -2), got %s", u1.Position)
	}

	// Unit 2 should be at (0, 2) and have taken 20 damage (HP 80)
	u2 := state.GetUnit(2)
	if !u2.Position.Equals(domain.GridPosition{X: 0, Y: 2}) {
		t.Errorf("Unit 2 final position mismatch: expected (0, 2), got %s", u2.Position)
	}
	if u2.Stats.HP != 80 {
		t.Errorf("Unit 2 HP mismatch: expected 80, got %d", u2.Stats.HP)
	}

	if err := state.Validate(); err != nil {
		t.Fatalf("final state failed Validate(): %v", err)
	}
}

// ============================================================================
// 2. High-Volume Fuzz Stream & State Determinism
// ============================================================================

func TestAdversarial_PoisonPill_HighVolumeStreamDeterminism(t *testing.T) {
	const streamLength = 2000

	// Initialize two independent GameStates
	stateA := domain.NewInitialGameState()
	stateB := domain.NewInitialGameState()

	// Initialize two independent Resolvers
	rA := resolver.NewResolver()
	rB := resolver.NewResolver()

	// Initialize two independent CommandQueues
	qA := commands.NewCommandQueue()
	qB := commands.NewCommandQueue()

	// Deterministic PRNG seed
	rng := rand.New(rand.NewSource(42))

	allTypes := []commands.CommandType{
		commands.CommandSelectUnit,
		commands.CommandMoveUnit,
		commands.CommandAttack,
		commands.CommandEndActivation,
		commands.CommandType("BOGUS_CMD_1"),
		commands.CommandType(""),
	}

	allPlayers := []domain.Player{
		domain.PlayerOne,
		domain.PlayerTwo,
		domain.PlayerNone,
		domain.Player(99),
		domain.Player(-1),
	}

	// Generate stream
	for i := 0; i < streamLength; i++ {
		sender := allPlayers[rng.Intn(len(allPlayers))]
		cmdType := allTypes[rng.Intn(len(allTypes))]
		target := domain.GridPosition{
			X: rng.Intn(15) - 7, // Range [-7, 7], includes out of bounds
			Y: rng.Intn(15) - 7,
		}
		dest := domain.GridPosition{
			X: rng.Intn(15) - 7,
			Y: rng.Intn(15) - 7,
		}

		cmd := commands.NetworkCommand{
			Sender: sender,
			Command: commands.GameCommand{
				Type:        cmdType,
				Target:      target,
				Destination: dest,
			},
		}

		qA.PushIncoming(cmd)
		qB.PushIncoming(cmd)
	}

	// Execute through both resolvers
	eventsA := rA.Resolve(stateA, qA)
	eventsB := rB.Resolve(stateB, qB)

	// Invariant 1: Queue must be fully drained
	if !qA.IsEmpty() || !qB.IsEmpty() {
		t.Fatalf("queues were not completely drained after high volume stream")
	}

	// Invariant 2: Identical event streams
	if len(eventsA) != len(eventsB) {
		t.Fatalf("determinism violation: event count mismatch (A=%d, B=%d)", len(eventsA), len(eventsB))
	}

	for i := range eventsA {
		evA := eventsA[i]
		evB := eventsB[i]
		if evA.Type != evB.Type || evA.Reason != evB.Reason {
			t.Fatalf("event %d mismatch:\nA: %v\nB: %v", i, evA, evB)
		}
	}

	// Invariant 3: Identical final states
	if stateA.ActivePlayer != stateB.ActivePlayer {
		t.Fatalf("ActivePlayer determinism mismatch: %s vs %s", stateA.ActivePlayer, stateB.ActivePlayer)
	}
	if stateA.Phase != stateB.Phase {
		t.Fatalf("Phase determinism mismatch: %s vs %s", stateA.Phase, stateB.Phase)
	}
	if !reflect.DeepEqual(stateA.ActiveUnitID, stateB.ActiveUnitID) {
		t.Fatalf("ActiveUnitID determinism mismatch")
	}
	for i := range stateA.Units {
		uA := stateA.Units[i]
		uB := stateB.Units[i]
		if !reflect.DeepEqual(uA, uB) {
			t.Fatalf("Unit %d determinism mismatch:\nA: %+v\nB: %+v", i, uA, uB)
		}
	}

	// Invariant 4: State validation passes
	if err := stateA.Validate(); err != nil {
		t.Fatalf("state A invalid after high volume stream: %v", err)
	}
	if err := stateB.Validate(); err != nil {
		t.Fatalf("state B invalid after high volume stream: %v", err)
	}
}

// ============================================================================
// 3. Resolver Defensive Guards & Edge Cases
// ============================================================================

func TestAdversarial_Resolver_DefensiveGuards(t *testing.T) {
	r := resolver.NewResolver()

	t.Run("NilState_ResolveReturnsNil", func(t *testing.T) {
		q := commands.NewCommandQueue()
		q.PushIncoming(commands.NewNetworkCommand(domain.PlayerOne, commands.NewEndActivationCommand()))
		events := r.Resolve(nil, q)
		if events != nil {
			t.Errorf("expected nil events on nil state, got %v", events)
		}
	})

	t.Run("NilQueue_ResolveReturnsNil", func(t *testing.T) {
		state := domain.NewInitialGameState()
		events := r.Resolve(state, nil)
		if events != nil {
			t.Errorf("expected nil events on nil queue, got %v", events)
		}
	})

	t.Run("NilState_ResolveCommandReturnsNil", func(t *testing.T) {
		cmd := commands.NewNetworkCommand(domain.PlayerOne, commands.NewEndActivationCommand())
		events := r.ResolveCommand(nil, cmd)
		if events != nil {
			t.Errorf("expected nil events on nil state ResolveCommand, got %v", events)
		}
	})

	t.Run("ZeroValueNetworkCommand_RejectionWithoutPanic", func(t *testing.T) {
		state := domain.NewInitialGameState()
		zeroCmd := commands.NetworkCommand{}
		events := r.ResolveCommand(state, zeroCmd)
		if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
			t.Fatalf("expected rejection for zero-value NetworkCommand, got %v", events)
		}
		if events[0].Reason != resolver.ReasonNotActivePlayer {
			t.Errorf("reason mismatch: got %q, want %q", events[0].Reason, resolver.ReasonNotActivePlayer)
		}
	})

	t.Run("AttackOverkill_NoUintUnderflow", func(t *testing.T) {
		state := domain.NewInitialGameState()
		u1 := state.GetUnit(1)
		u2 := state.GetUnit(2)

		// Set Unit 2 HP to 0 already
		u2.Stats.HP = 0

		// Select Unit 1
		r.ResolveCommand(state, commands.NewNetworkCommand(domain.PlayerOne, commands.NewSelectUnitCommand(u1.Position)))

		// Attack Unit 2 (which already has 0 HP)
		events := r.ResolveCommand(state, commands.NewNetworkCommand(domain.PlayerOne, commands.NewAttackCommand(u2.Position)))

		if len(events) != 2 {
			t.Fatalf("expected 2 events (UnitAttacked and UnitDefeated), got %d", len(events))
		}
		if events[0].Type != resolver.EventUnitAttacked || !events[0].Defeated {
			t.Errorf("expected UnitAttacked with Defeated=true")
		}
		if u2.Stats.HP != 0 {
			t.Errorf("HP underflow occurred: got %d", u2.Stats.HP)
		}
	})

	t.Run("MoveToSameTile_PermittedHoldAction", func(t *testing.T) {
		state := domain.NewInitialGameState()
		u1 := state.GetUnit(1)

		// Select Unit 1 at (0, -3)
		r.ResolveCommand(state, commands.NewNetworkCommand(domain.PlayerOne, commands.NewSelectUnitCommand(u1.Position)))

		// Move Unit 1 to its own coordinate (0, -3)
		events := r.ResolveCommand(state, commands.NewNetworkCommand(domain.PlayerOne, commands.NewMoveUnitCommand(u1.Position)))

		if len(events) != 1 || events[0].Type != resolver.EventUnitMoved {
			t.Fatalf("expected legal move to own tile, got %v", events)
		}
		if !u1.Position.Equals(domain.GridPosition{X: 0, Y: -3}) {
			t.Errorf("unit position altered: %s", u1.Position)
		}
	})
}
