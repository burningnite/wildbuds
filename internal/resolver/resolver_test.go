package resolver_test

import (
	"testing"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
	"wildbuds/internal/resolver"
)

// Helper to construct a standard NetworkCommand
func makeCmd(sender domain.Player, cmdType commands.CommandType, target, dest domain.GridPosition) commands.NetworkCommand {
	return commands.NetworkCommand{
		Sender: sender,
		Command: commands.GameCommand{
			Type:        cmdType,
			Target:      target,
			Destination: dest,
		},
	}
}

// Helper to set up a standard test state with initial units:
// - Unit 1: PlayerOne at (0, -3), Water element, HP: 100, Attack: 10
// - Unit 2: PlayerTwo at (0, 3), Fire element, HP: 100, Attack: 10
// ActivePlayer: PlayerOne, ActiveUnitID: nil, Phase: PhaseSelectUnit
func setupInitialState() *domain.GameState {
	return domain.NewInitialGameState()
}

// ============================================================================
// Group 1: Reference Rust Unit Test Parity (src/game.rs:100-125)
// ============================================================================

func TestRustParity_SelectUnitValid(t *testing.T) {
	state := setupInitialState()
	r := resolver.NewResolver()
	q := commands.NewCommandQueue()

	cmd := commands.NetworkCommand{
		Sender: domain.PlayerOne,
		Command: commands.GameCommand{
			Type:   commands.CommandSelectUnit,
			Target: domain.GridPosition{X: 0, Y: -3},
		},
	}
	q.PushIncoming(cmd)

	events := r.Resolve(state, q)

	// Verify state mutation
	if state.ActiveUnitID == nil {
		t.Fatalf("expected ActiveUnitID to be set, got nil")
	}
	if *state.ActiveUnitID != 1 {
		t.Errorf("expected ActiveUnitID 1, got %d", *state.ActiveUnitID)
	}
	if state.Phase != domain.PhaseChooseAction {
		t.Errorf("expected phase %s, got %s", domain.PhaseChooseAction, state.Phase)
	}
	if state.ActivePlayer != domain.PlayerOne {
		t.Errorf("expected ActivePlayer PlayerOne, got %s", state.ActivePlayer)
	}

	// Verify invariant integrity
	if err := state.Validate(); err != nil {
		t.Fatalf("state validation failed: %v", err)
	}

	// Verify queue is drained
	if !q.IsEmpty() {
		t.Errorf("expected command queue to be empty, len=%d", q.LenIncoming())
	}

	// Verify event emission
	if len(events) != 1 || events[0].Type != resolver.EventUnitSelected {
		t.Errorf("expected 1 UnitSelected event, got %v", events)
	}
}

func TestRustParity_SelectUnitWrongPlayer(t *testing.T) {
	state := setupInitialState()
	r := resolver.NewResolver()
	q := commands.NewCommandQueue()

	cmd := commands.NetworkCommand{
		Sender: domain.PlayerTwo,
		Command: commands.GameCommand{
			Type:   commands.CommandSelectUnit,
			Target: domain.GridPosition{X: 0, Y: 3}, // PlayerTwo's own unit
		},
	}
	q.PushIncoming(cmd)

	events := r.Resolve(state, q)

	// Verify ZERO state mutations
	if state.ActiveUnitID != nil {
		t.Errorf("expected ActiveUnitID nil, got %v", *state.ActiveUnitID)
	}
	if state.Phase != domain.PhaseSelectUnit {
		t.Errorf("expected phase %s, got %s", domain.PhaseSelectUnit, state.Phase)
	}
	if state.ActivePlayer != domain.PlayerOne {
		t.Errorf("expected ActivePlayer PlayerOne, got %s", state.ActivePlayer)
	}

	// Verify units untouched
	u1 := state.GetUnit(1)
	if !u1.Position.Equals(domain.GridPosition{X: 0, Y: -3}) {
		t.Errorf("expected Unit 1 position unchanged")
	}

	if err := state.Validate(); err != nil {
		t.Fatalf("state validation failed: %v", err)
	}
	if !q.IsEmpty() {
		t.Errorf("expected command queue to be empty")
	}

	// Verify rejection event was emitted
	if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
		t.Fatalf("expected EventCommandRejected, got %v", events)
	}
	if events[0].Reason != resolver.ReasonNotActivePlayer {
		t.Errorf("expected reason %q, got %q", resolver.ReasonNotActivePlayer, events[0].Reason)
	}
}

// ============================================================================
// Group 2: Acceptance Criteria - Universal Wrong Player Rejection
// ============================================================================

func TestGatekeeper_WrongPlayerRejection_AllCommands(t *testing.T) {
	tests := []struct {
		name    string
		command commands.GameCommand
	}{
		{
			name: "SelectUnit from wrong player",
			command: commands.GameCommand{
				Type:   commands.CommandSelectUnit,
				Target: domain.GridPosition{X: 0, Y: 3},
			},
		},
		{
			name: "MoveUnit from wrong player",
			command: commands.GameCommand{
				Type:        commands.CommandMoveUnit,
				Destination: domain.GridPosition{X: 0, Y: 2},
			},
		},
		{
			name: "Attack from wrong player",
			command: commands.GameCommand{
				Type:   commands.CommandAttack,
				Target: domain.GridPosition{X: 0, Y: -3},
			},
		},
		{
			name: "EndActivation from wrong player",
			command: commands.GameCommand{
				Type: commands.CommandEndActivation,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := setupInitialState()
			r := resolver.NewResolver()
			q := commands.NewCommandQueue()

			// ActivePlayer is PlayerOne; sender is PlayerTwo
			q.PushIncoming(commands.NetworkCommand{
				Sender:  domain.PlayerTwo,
				Command: tc.command,
			})

			events := r.Resolve(state, q)

			if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
				t.Fatalf("expected EventCommandRejected, got %v", events)
			}
			if events[0].Reason != resolver.ReasonNotActivePlayer {
				t.Errorf("expected reason %q, got %q", resolver.ReasonNotActivePlayer, events[0].Reason)
			}

			// Verify absolute state preservation
			if state.ActivePlayer != domain.PlayerOne {
				t.Errorf("ActivePlayer changed unexpectedly: got %s", state.ActivePlayer)
			}
			if state.ActiveUnitID != nil {
				t.Errorf("ActiveUnitID mutated unexpectedly: got %v", *state.ActiveUnitID)
			}
			if state.Phase != domain.PhaseSelectUnit {
				t.Errorf("Phase mutated unexpectedly: got %s", state.Phase)
			}
			if err := state.Validate(); err != nil {
				t.Errorf("state validation failed: %v", err)
			}
		})
	}
}

func TestGatekeeper_RejectPlayerNone(t *testing.T) {
	r := resolver.New()
	state := setupInitialState()

	cmd := makeCmd(domain.PlayerNone, commands.CommandSelectUnit, domain.GridPosition{X: 0, Y: -3}, domain.GridPosition{})
	events := r.ResolveCommand(state, cmd)

	if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
		t.Fatalf("expected rejection for PlayerNone")
	}
	if events[0].Reason != resolver.ReasonNotActivePlayer {
		t.Errorf("expected reason %q, got %q", resolver.ReasonNotActivePlayer, events[0].Reason)
	}
}

func TestGatekeeper_ConsecutiveEndActivation(t *testing.T) {
	r := resolver.New()
	state := setupInitialState()

	// 1. PlayerOne ends activation -> turn passes to PlayerTwo
	cmd1 := makeCmd(domain.PlayerOne, commands.CommandEndActivation, domain.GridPosition{}, domain.GridPosition{})
	ev1 := r.ResolveCommand(state, cmd1)
	if len(ev1) != 1 || ev1[0].Type != resolver.EventActivationEnded {
		t.Fatalf("expected first EndActivation to succeed")
	}
	if state.ActivePlayer != domain.PlayerTwo {
		t.Fatalf("expected active player to be PlayerTwo")
	}

	// 2. PlayerOne tries to end activation again -> rejected because it is now PlayerTwo's turn!
	cmd2 := makeCmd(domain.PlayerOne, commands.CommandEndActivation, domain.GridPosition{}, domain.GridPosition{})
	ev2 := r.ResolveCommand(state, cmd2)
	if len(ev2) != 1 || ev2[0].Type != resolver.EventCommandRejected {
		t.Fatalf("expected second EndActivation from same player to be rejected")
	}
	if state.ActivePlayer != domain.PlayerTwo {
		t.Errorf("active player should remain PlayerTwo")
	}
}

// ============================================================================
// Group 3: SelectUnit Validation & Edge Cases
// ============================================================================

func TestResolver_SelectUnit_Success(t *testing.T) {
	r := resolver.New()
	state := setupInitialState()

	cmd := makeCmd(domain.PlayerOne, commands.CommandSelectUnit, domain.GridPosition{X: 0, Y: -3}, domain.GridPosition{})
	events := r.ResolveCommand(state, cmd)

	if len(events) != 1 || events[0].Type != resolver.EventUnitSelected {
		t.Fatalf("expected UnitSelected event, got %v", events)
	}
	if state.ActiveUnitID == nil || *state.ActiveUnitID != 1 {
		t.Errorf("expected ActiveUnitID to be 1, got %v", state.ActiveUnitID)
	}
	if state.Phase != domain.PhaseChooseAction {
		t.Errorf("expected PhaseChooseAction, got %s", state.Phase)
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("state validation failed: %v", err)
	}
}

func TestResolver_SelectUnit_OpponentUnit_Rejected(t *testing.T) {
	r := resolver.New()
	state := setupInitialState()

	cmd := makeCmd(domain.PlayerOne, commands.CommandSelectUnit, domain.GridPosition{X: 0, Y: 3}, domain.GridPosition{})
	events := r.ResolveCommand(state, cmd)

	if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
		t.Fatalf("expected CommandRejected for selecting opponent unit")
	}
	if events[0].Reason != resolver.ReasonUnitNotOwned {
		t.Errorf("expected reason %q, got %q", resolver.ReasonUnitNotOwned, events[0].Reason)
	}
	if state.ActiveUnitID != nil {
		t.Errorf("ActiveUnitID should remain nil")
	}
	if state.Phase != domain.PhaseSelectUnit {
		t.Errorf("Phase should remain SelectUnit")
	}
}

func TestResolver_SelectUnit_EmptyTile_Rejected(t *testing.T) {
	r := resolver.New()
	state := setupInitialState()

	cmd := makeCmd(domain.PlayerOne, commands.CommandSelectUnit, domain.GridPosition{X: 0, Y: 0}, domain.GridPosition{})
	events := r.ResolveCommand(state, cmd)

	if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
		t.Fatalf("expected CommandRejected for selecting empty tile")
	}
	if events[0].Reason != resolver.ReasonNoUnitAtTarget {
		t.Errorf("expected reason %q, got %q", resolver.ReasonNoUnitAtTarget, events[0].Reason)
	}
}

func TestResolver_SelectUnit_OutOfBounds_Rejected(t *testing.T) {
	r := resolver.New()
	state := setupInitialState()

	cmd := makeCmd(domain.PlayerOne, commands.CommandSelectUnit, domain.GridPosition{X: 6, Y: 0}, domain.GridPosition{})
	events := r.ResolveCommand(state, cmd)

	if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
		t.Fatalf("expected CommandRejected for out of bounds target")
	}
	if events[0].Reason != resolver.ReasonTargetOutOfBounds {
		t.Errorf("expected reason %q, got %q", resolver.ReasonTargetOutOfBounds, events[0].Reason)
	}
}

func TestResolver_SelectUnit_Reselection(t *testing.T) {
	r := resolver.New()
	state := setupInitialState()

	// Add second unit for PlayerOne at (1, -3)
	unit3 := &domain.Unit{
		ID:       3,
		Owner:    domain.PlayerOne,
		Position: domain.GridPosition{X: 1, Y: -3},
		Stats:    domain.DefaultBaseStats(),
		Tokens:   domain.DefaultActionTokens(),
		Types:    domain.NewSingleType(domain.ElementGrass),
	}
	state.Units = append(state.Units, unit3)

	// Select unit 1
	r.ResolveCommand(state, makeCmd(domain.PlayerOne, commands.CommandSelectUnit, domain.GridPosition{X: 0, Y: -3}, domain.GridPosition{}))
	if *state.ActiveUnitID != 1 {
		t.Fatalf("expected unit 1 active")
	}

	// Reselect unit 3
	events := r.ResolveCommand(state, makeCmd(domain.PlayerOne, commands.CommandSelectUnit, domain.GridPosition{X: 1, Y: -3}, domain.GridPosition{}))
	if len(events) != 1 || events[0].Type != resolver.EventUnitSelected {
		t.Fatalf("expected UnitSelected event on reselection")
	}
	if *state.ActiveUnitID != 3 {
		t.Errorf("expected ActiveUnitID to switch to 3, got %d", *state.ActiveUnitID)
	}
	if state.Phase != domain.PhaseChooseAction {
		t.Errorf("expected PhaseChooseAction, got %s", state.Phase)
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("state validation failed: %v", err)
	}
}

// ============================================================================
// Group 4: MoveUnit Validation & Edge Cases
// ============================================================================

func TestResolver_MoveUnit_Success(t *testing.T) {
	r := resolver.New()
	state := setupInitialState()

	// 1. Select Unit
	r.ResolveCommand(state, makeCmd(domain.PlayerOne, commands.CommandSelectUnit, domain.GridPosition{X: 0, Y: -3}, domain.GridPosition{}))

	// 2. Move Unit to (0, -2)
	dest := domain.GridPosition{X: 0, Y: -2}
	cmd := makeCmd(domain.PlayerOne, commands.CommandMoveUnit, domain.GridPosition{}, dest)
	events := r.ResolveCommand(state, cmd)

	if len(events) != 1 || events[0].Type != resolver.EventUnitMoved {
		t.Fatalf("expected UnitMoved event, got %v", events)
	}

	unit := state.GetUnit(1)
	if !unit.Position.Equals(dest) {
		t.Errorf("expected unit position %v, got %v", dest, unit.Position)
	}
	if state.Phase != domain.PhaseChooseAction {
		t.Errorf("expected phase preserved as ChooseAction")
	}
	if state.ActiveUnitID == nil || *state.ActiveUnitID != 1 {
		t.Errorf("expected ActiveUnitID preserved as 1")
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("state validation failed: %v", err)
	}
}

func TestResolver_MoveUnit_NoActiveUnit_Rejected(t *testing.T) {
	r := resolver.New()
	state := setupInitialState()

	cmd := makeCmd(domain.PlayerOne, commands.CommandMoveUnit, domain.GridPosition{}, domain.GridPosition{X: 0, Y: -2})
	events := r.ResolveCommand(state, cmd)

	if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
		t.Fatalf("expected rejection when moving without active unit")
	}
	if events[0].Reason != resolver.ReasonNoActiveUnit {
		t.Errorf("expected reason %q, got %q", resolver.ReasonNoActiveUnit, events[0].Reason)
	}
}

func TestResolver_MoveUnit_OccupiedTile_Rejected(t *testing.T) {
	r := resolver.New()
	state := setupInitialState()

	// Select Unit 1 at (0, -3)
	r.ResolveCommand(state, makeCmd(domain.PlayerOne, commands.CommandSelectUnit, domain.GridPosition{X: 0, Y: -3}, domain.GridPosition{}))

	// Attempt to move into Unit 2's tile at (0, 3)
	cmd := makeCmd(domain.PlayerOne, commands.CommandMoveUnit, domain.GridPosition{}, domain.GridPosition{X: 0, Y: 3})
	events := r.ResolveCommand(state, cmd)

	if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
		t.Fatalf("expected rejection when moving into occupied tile")
	}
	if events[0].Reason != resolver.ReasonDestinationOccupied {
		t.Errorf("expected reason %q, got %q", resolver.ReasonDestinationOccupied, events[0].Reason)
	}
	unit1 := state.GetUnit(1)
	if !unit1.Position.Equals(domain.GridPosition{X: 0, Y: -3}) {
		t.Errorf("unit position should remain unchanged")
	}
}

func TestResolver_MoveUnit_OutOfBounds_Rejected(t *testing.T) {
	r := resolver.New()
	state := setupInitialState()

	r.ResolveCommand(state, makeCmd(domain.PlayerOne, commands.CommandSelectUnit, domain.GridPosition{X: 0, Y: -3}, domain.GridPosition{}))

	cmd := makeCmd(domain.PlayerOne, commands.CommandMoveUnit, domain.GridPosition{}, domain.GridPosition{X: 0, Y: -6})
	events := r.ResolveCommand(state, cmd)

	if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
		t.Fatalf("expected rejection when moving out of bounds")
	}
	if events[0].Reason != resolver.ReasonDestOutOfBounds {
		t.Errorf("expected reason %q, got %q", resolver.ReasonDestOutOfBounds, events[0].Reason)
	}
}

// ============================================================================
// Group 5: Attack & Combat Resolution
// ============================================================================

func TestResolver_Attack_ElementalSuperEffective(t *testing.T) {
	r := resolver.New()
	state := setupInitialState()
	// Unit 1 (Water, Attack: 10) attacks Unit 2 (Fire, HP: 100)
	// Water vs Fire multiplier = 2.0 -> Damage = 20 -> RemHP = 80

	r.ResolveCommand(state, makeCmd(domain.PlayerOne, commands.CommandSelectUnit, domain.GridPosition{X: 0, Y: -3}, domain.GridPosition{}))

	cmd := makeCmd(domain.PlayerOne, commands.CommandAttack, domain.GridPosition{X: 0, Y: 3}, domain.GridPosition{})
	events := r.ResolveCommand(state, cmd)

	if len(events) != 1 || events[0].Type != resolver.EventUnitAttacked {
		t.Fatalf("expected UnitAttacked event, got %v", events)
	}

	ev := events[0]
	if ev.Damage != 20 {
		t.Errorf("expected 20 damage, got %d", ev.Damage)
	}
	if ev.Multiplier != 2.0 {
		t.Errorf("expected 2.0 multiplier, got %f", ev.Multiplier)
	}
	if ev.RemainingHP != 80 {
		t.Errorf("expected 80 remaining HP, got %d", ev.RemainingHP)
	}
	if ev.Defeated {
		t.Errorf("expected unit not defeated yet")
	}

	target := state.GetUnit(2)
	if target.Stats.HP != 80 {
		t.Errorf("target HP mismatch: want 80, got %d", target.Stats.HP)
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("state validation failed: %v", err)
	}
}

func TestResolver_Attack_DefeatedUnit(t *testing.T) {
	r := resolver.New()
	state := setupInitialState()

	// Set defender HP low
	unit2 := state.GetUnit(2)
	unit2.Stats.HP = 15

	r.ResolveCommand(state, makeCmd(domain.PlayerOne, commands.CommandSelectUnit, domain.GridPosition{X: 0, Y: -3}, domain.GridPosition{}))

	// Unit 1 deals 20 damage -> 15 HP -> 0 HP (Defeated)
	cmd := makeCmd(domain.PlayerOne, commands.CommandAttack, domain.GridPosition{X: 0, Y: 3}, domain.GridPosition{})
	events := r.ResolveCommand(state, cmd)

	if len(events) != 2 {
		t.Fatalf("expected 2 events (UnitAttacked and UnitDefeated), got %d", len(events))
	}
	if events[0].Type != resolver.EventUnitAttacked || !events[0].Defeated {
		t.Errorf("expected UnitAttacked with Defeated=true")
	}
	if events[1].Type != resolver.EventUnitDefeated {
		t.Errorf("expected UnitDefeated event")
	}
	if unit2.Stats.HP != 0 {
		t.Errorf("unit HP should be 0, got %d", unit2.Stats.HP)
	}
}

func TestResolver_Attack_FriendlyTarget_Rejected(t *testing.T) {
	r := resolver.New()
	state := setupInitialState()

	// Add second unit for PlayerOne at (1, -3)
	state.Units = append(state.Units, &domain.Unit{
		ID:       3,
		Owner:    domain.PlayerOne,
		Position: domain.GridPosition{X: 1, Y: -3},
		Stats:    domain.DefaultBaseStats(),
		Tokens:   domain.DefaultActionTokens(),
		Types:    domain.NewSingleType(domain.ElementWater),
	})

	r.ResolveCommand(state, makeCmd(domain.PlayerOne, commands.CommandSelectUnit, domain.GridPosition{X: 0, Y: -3}, domain.GridPosition{}))

	// PlayerOne unit attacks own unit at (1, -3)
	cmd := makeCmd(domain.PlayerOne, commands.CommandAttack, domain.GridPosition{X: 1, Y: -3}, domain.GridPosition{})
	events := r.ResolveCommand(state, cmd)

	if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
		t.Fatalf("expected rejection for attacking friendly unit")
	}
	if events[0].Reason != resolver.ReasonCannotAttackFriendly {
		t.Errorf("expected reason %q, got %q", resolver.ReasonCannotAttackFriendly, events[0].Reason)
	}
}

func TestResolver_Attack_NoActiveUnit_Rejected(t *testing.T) {
	r := resolver.New()
	state := setupInitialState()

	cmd := makeCmd(domain.PlayerOne, commands.CommandAttack, domain.GridPosition{X: 0, Y: 3}, domain.GridPosition{})
	events := r.ResolveCommand(state, cmd)

	if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
		t.Fatalf("expected rejection for attack without active unit")
	}
	if events[0].Reason != resolver.ReasonNoActiveUnit {
		t.Errorf("expected reason %q, got %q", resolver.ReasonNoActiveUnit, events[0].Reason)
	}
}

func TestResolver_Attack_EmptyTile_Rejected(t *testing.T) {
	r := resolver.New()
	state := setupInitialState()

	r.ResolveCommand(state, makeCmd(domain.PlayerOne, commands.CommandSelectUnit, domain.GridPosition{X: 0, Y: -3}, domain.GridPosition{}))

	cmd := makeCmd(domain.PlayerOne, commands.CommandAttack, domain.GridPosition{X: 0, Y: 0}, domain.GridPosition{})
	events := r.ResolveCommand(state, cmd)

	if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
		t.Fatalf("expected rejection for attack on empty tile")
	}
	if events[0].Reason != resolver.ReasonNoUnitAtTarget {
		t.Errorf("expected reason %q, got %q", resolver.ReasonNoUnitAtTarget, events[0].Reason)
	}
}

func TestResolver_Attack_OutOfBounds_Rejected(t *testing.T) {
	r := resolver.New()
	state := setupInitialState()

	r.ResolveCommand(state, makeCmd(domain.PlayerOne, commands.CommandSelectUnit, domain.GridPosition{X: 0, Y: -3}, domain.GridPosition{}))

	cmd := makeCmd(domain.PlayerOne, commands.CommandAttack, domain.GridPosition{X: 10, Y: 0}, domain.GridPosition{})
	events := r.ResolveCommand(state, cmd)

	if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
		t.Fatalf("expected rejection for attack out of bounds")
	}
	if events[0].Reason != resolver.ReasonTargetOutOfBounds {
		t.Errorf("expected reason %q, got %q", resolver.ReasonTargetOutOfBounds, events[0].Reason)
	}
}

func TestResolver_CustomCombatHook(t *testing.T) {
	customHook := func(attacker *domain.Unit, defender *domain.Unit) (uint32, float32) {
		return 50, 5.0
	}

	r := resolver.New(resolver.WithCombatHook(customHook))
	state := setupInitialState()

	r.ResolveCommand(state, makeCmd(domain.PlayerOne, commands.CommandSelectUnit, domain.GridPosition{X: 0, Y: -3}, domain.GridPosition{}))

	cmd := makeCmd(domain.PlayerOne, commands.CommandAttack, domain.GridPosition{X: 0, Y: 3}, domain.GridPosition{})
	events := r.ResolveCommand(state, cmd)

	if len(events) != 1 || events[0].Type != resolver.EventUnitAttacked {
		t.Fatalf("expected UnitAttacked event")
	}
	if events[0].Damage != 50 || events[0].Multiplier != 5.0 {
		t.Errorf("custom hook damage mismatch: got %d, mult %f", events[0].Damage, events[0].Multiplier)
	}
	if state.GetUnit(2).Stats.HP != 50 {
		t.Errorf("expected defender HP 50, got %d", state.GetUnit(2).Stats.HP)
	}
}

// ============================================================================
// Group 6: EndActivation & Turn Transitions
// ============================================================================

func TestResolver_EndActivation_Success(t *testing.T) {
	r := resolver.New()
	state := setupInitialState()

	// Select unit 1
	r.ResolveCommand(state, makeCmd(domain.PlayerOne, commands.CommandSelectUnit, domain.GridPosition{X: 0, Y: -3}, domain.GridPosition{}))

	// End activation
	cmd := makeCmd(domain.PlayerOne, commands.CommandEndActivation, domain.GridPosition{}, domain.GridPosition{})
	events := r.ResolveCommand(state, cmd)

	if len(events) != 1 || events[0].Type != resolver.EventActivationEnded {
		t.Fatalf("expected ActivationEnded event, got %v", events)
	}
	if *events[0].PreviousPlayer != domain.PlayerOne || *events[0].NextPlayer != domain.PlayerTwo {
		t.Errorf("player transition mismatch: %v -> %v", events[0].PreviousPlayer, events[0].NextPlayer)
	}

	if state.ActivePlayer != domain.PlayerTwo {
		t.Errorf("ActivePlayer should be PlayerTwo, got %s", state.ActivePlayer)
	}
	if state.ActiveUnitID != nil {
		t.Errorf("ActiveUnitID should be nil")
	}
	if state.Phase != domain.PhaseSelectUnit {
		t.Errorf("Phase should be SelectUnit, got %s", state.Phase)
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("state validation failed: %v", err)
	}
}

func TestResolver_EndActivation_WithoutAction(t *testing.T) {
	r := resolver.New()
	state := setupInitialState()

	// Pass turn immediately
	cmd := makeCmd(domain.PlayerOne, commands.CommandEndActivation, domain.GridPosition{}, domain.GridPosition{})
	events := r.ResolveCommand(state, cmd)

	if len(events) != 1 || events[0].Type != resolver.EventActivationEnded {
		t.Fatalf("expected ActivationEnded event, got %v", events)
	}
	if state.ActivePlayer != domain.PlayerTwo {
		t.Errorf("expected ActivePlayer PlayerTwo, got %s", state.ActivePlayer)
	}
	if state.Phase != domain.PhaseSelectUnit {
		t.Errorf("expected phase SelectUnit")
	}
	if state.ActiveUnitID != nil {
		t.Errorf("expected ActiveUnitID nil")
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("state validation failed: %v", err)
	}
}

// ============================================================================
// Group 7: Input Decoupling & Queue Purity
// ============================================================================

func TestResolver_InputDecoupling_QueuePurity(t *testing.T) {
	state := setupInitialState()
	q := commands.NewCommandQueue()

	// Simulate Input layer pushing multiple commands to outgoing queue
	q.PushOutgoing(commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3}))
	q.PushOutgoing(commands.NewMoveUnitCommand(domain.GridPosition{X: 1, Y: -3}))
	q.PushOutgoing(commands.NewEndActivationCommand())

	// Assert GameState is 100% untouched while commands reside in queue
	if state.ActiveUnitID != nil {
		t.Fatal("input layer mutated ActiveUnitID directly!")
	}
	if state.Phase != domain.PhaseSelectUnit {
		t.Fatal("input layer mutated Phase directly!")
	}
	if state.ActivePlayer != domain.PlayerOne {
		t.Fatal("input layer mutated ActivePlayer directly!")
	}

	// Transfer outgoing to incoming via loopback
	routed := q.RouteOutgoingToIncoming(domain.PlayerOne)
	if routed != 3 {
		t.Fatalf("expected 3 routed commands, got %d", routed)
	}

	// Assert GameState is STILL 100% untouched before Resolve() is called
	if state.ActiveUnitID != nil || state.Phase != domain.PhaseSelectUnit {
		t.Fatal("network queue mutated GameState before resolver execution!")
	}

	// Authoritative resolution executes
	r := resolver.NewResolver()
	events := r.Resolve(state, q)

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if state.ActivePlayer != domain.PlayerTwo {
		t.Errorf("expected ActivePlayer PlayerTwo after resolution, got %s", state.ActivePlayer)
	}
	u1 := state.GetUnit(1)
	if !u1.Position.Equals(domain.GridPosition{X: 1, Y: -3}) {
		t.Errorf("expected unit 1 moved to (1, -3), got %s", u1.Position)
	}
}

// ============================================================================
// Group 8: Queue Mechanics - Sequential FIFO & Poison Pill Resilience
// ============================================================================

func TestResolver_Queue_SequentialFIFO(t *testing.T) {
	r := resolver.New()
	state := setupInitialState()
	q := commands.NewCommandQueue()

	// Enqueue valid sequence:
	// 1. P1 selects unit 1 at (0, -3)
	// 2. P1 moves unit 1 to (0, -2)
	// 3. P1 ends activation
	// 4. P2 selects unit 2 at (0, 3)
	// 5. P2 moves unit 2 to (0, 2)
	// 6. P2 ends activation
	q.PushIncoming(makeCmd(domain.PlayerOne, commands.CommandSelectUnit, domain.GridPosition{X: 0, Y: -3}, domain.GridPosition{}))
	q.PushIncoming(makeCmd(domain.PlayerOne, commands.CommandMoveUnit, domain.GridPosition{}, domain.GridPosition{X: 0, Y: -2}))
	q.PushIncoming(makeCmd(domain.PlayerOne, commands.CommandEndActivation, domain.GridPosition{}, domain.GridPosition{}))
	q.PushIncoming(makeCmd(domain.PlayerTwo, commands.CommandSelectUnit, domain.GridPosition{X: 0, Y: 3}, domain.GridPosition{}))
	q.PushIncoming(makeCmd(domain.PlayerTwo, commands.CommandMoveUnit, domain.GridPosition{}, domain.GridPosition{X: 0, Y: 2}))
	q.PushIncoming(makeCmd(domain.PlayerTwo, commands.CommandEndActivation, domain.GridPosition{}, domain.GridPosition{}))

	events := r.Resolve(state, q)

	if len(events) != 6 {
		t.Fatalf("expected 6 events, got %d: %v", len(events), events)
	}
	if events[0].Type != resolver.EventUnitSelected {
		t.Errorf("event 0 want UnitSelected, got %s", events[0].Type)
	}
	if events[1].Type != resolver.EventUnitMoved {
		t.Errorf("event 1 want UnitMoved, got %s", events[1].Type)
	}
	if events[2].Type != resolver.EventActivationEnded {
		t.Errorf("event 2 want ActivationEnded, got %s", events[2].Type)
	}
	if events[3].Type != resolver.EventUnitSelected {
		t.Errorf("event 3 want UnitSelected, got %s", events[3].Type)
	}
	if events[4].Type != resolver.EventUnitMoved {
		t.Errorf("event 4 want UnitMoved, got %s", events[4].Type)
	}
	if events[5].Type != resolver.EventActivationEnded {
		t.Errorf("event 5 want ActivationEnded, got %s", events[5].Type)
	}

	// Queue must be completely drained
	if !q.IsEmpty() {
		t.Errorf("queue should be empty after resolution")
	}

	// Verify final state
	if state.ActivePlayer != domain.PlayerOne {
		t.Errorf("active player want PlayerOne, got %s", state.ActivePlayer)
	}
	if state.ActiveUnitID != nil {
		t.Errorf("active unit want nil, got %v", state.ActiveUnitID)
	}
	if state.Phase != domain.PhaseSelectUnit {
		t.Errorf("phase want SelectUnit, got %s", state.Phase)
	}
	if !state.GetUnit(1).Position.Equals(domain.GridPosition{X: 0, Y: -2}) {
		t.Errorf("unit 1 position not updated: %v", state.GetUnit(1).Position)
	}
	if !state.GetUnit(2).Position.Equals(domain.GridPosition{X: 0, Y: 2}) {
		t.Errorf("unit 2 position not updated: %v", state.GetUnit(2).Position)
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("final state validation failed: %v", err)
	}
}

func TestResolver_Queue_PoisonPillResilience(t *testing.T) {
	state := setupInitialState()
	r := resolver.NewResolver()
	q := commands.NewCommandQueue()

	// 1. Poison pill: wrong player command
	q.PushIncoming(makeCmd(domain.PlayerTwo, commands.CommandSelectUnit, domain.GridPosition{X: 0, Y: 3}, domain.GridPosition{}))
	// 2. Valid command: PlayerOne selects unit 1
	q.PushIncoming(makeCmd(domain.PlayerOne, commands.CommandSelectUnit, domain.GridPosition{X: 0, Y: -3}, domain.GridPosition{}))
	// 3. Poison pill: move out of bounds
	q.PushIncoming(makeCmd(domain.PlayerOne, commands.CommandMoveUnit, domain.GridPosition{}, domain.GridPosition{X: 0, Y: -99}))
	// 4. Valid command: PlayerOne moves unit 1 to valid tile
	q.PushIncoming(makeCmd(domain.PlayerOne, commands.CommandMoveUnit, domain.GridPosition{}, domain.GridPosition{X: 1, Y: -3}))

	events := r.Resolve(state, q)

	if len(events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(events))
	}
	if events[0].Type != resolver.EventCommandRejected {
		t.Errorf("event 0 expected rejection")
	}
	if events[1].Type != resolver.EventUnitSelected {
		t.Errorf("event 1 expected UnitSelected")
	}
	if events[2].Type != resolver.EventCommandRejected {
		t.Errorf("event 2 expected rejection")
	}
	if events[3].Type != resolver.EventUnitMoved {
		t.Errorf("event 3 expected UnitMoved")
	}

	u1 := state.GetUnit(1)
	if !u1.Position.Equals(domain.GridPosition{X: 1, Y: -3}) {
		t.Errorf("valid commands failed to process after poison pill: pos=%s", u1.Position)
	}
	if state.ActiveUnitID == nil || *state.ActiveUnitID != 1 {
		t.Errorf("unit 1 failed to remain active")
	}
	if !q.IsEmpty() {
		t.Errorf("queue was not fully drained")
	}
}

// ============================================================================
// Group 9: Determinism & Safety
// ============================================================================

func TestResolver_Determinism(t *testing.T) {
	stateA := setupInitialState()
	stateB := stateA.Clone()

	rA := resolver.NewResolver()
	rB := resolver.NewResolver()

	qA := commands.NewCommandQueue()
	qB := commands.NewCommandQueue()

	cmds := []commands.NetworkCommand{
		makeCmd(domain.PlayerOne, commands.CommandSelectUnit, domain.GridPosition{X: 0, Y: -3}, domain.GridPosition{}),
		makeCmd(domain.PlayerOne, commands.CommandMoveUnit, domain.GridPosition{}, domain.GridPosition{X: 1, Y: -3}),
		makeCmd(domain.PlayerOne, commands.CommandEndActivation, domain.GridPosition{}, domain.GridPosition{}),
	}

	for _, c := range cmds {
		qA.PushIncoming(c)
		qB.PushIncoming(c)
	}

	rA.Resolve(stateA, qA)
	rB.Resolve(stateB, qB)

	if stateA.ActivePlayer != stateB.ActivePlayer {
		t.Errorf("determinism violation: ActivePlayer mismatch")
	}
	if stateA.Phase != stateB.Phase {
		t.Errorf("determinism violation: Phase mismatch")
	}
	uA := stateA.GetUnit(1)
	uB := stateB.GetUnit(1)
	if !uA.Position.Equals(uB.Position) {
		t.Errorf("determinism violation: Unit 1 position mismatch")
	}
}

func TestResolver_NilAndEmptySafety(t *testing.T) {
	state := setupInitialState()
	r := resolver.NewResolver()

	// Empty queue
	emptyQ := commands.NewCommandQueue()
	events := r.Resolve(state, emptyQ)
	if len(events) != 0 {
		t.Errorf("expected 0 events for empty queue")
	}

	// Nil queue
	eventsNil := r.Resolve(state, nil)
	if len(eventsNil) != 0 {
		t.Errorf("expected 0 events for nil queue")
	}

	// Nil state
	eventsNilState := r.Resolve(nil, emptyQ)
	if len(eventsNilState) != 0 {
		t.Errorf("expected 0 events for nil state")
	}

	// Unknown command
	unknownCmd := makeCmd(domain.PlayerOne, commands.CommandType("Invalid"), domain.GridPosition{}, domain.GridPosition{})
	eventsUnknown := r.ResolveCommand(state, unknownCmd)
	if len(eventsUnknown) != 1 || eventsUnknown[0].Type != resolver.EventCommandRejected {
		t.Errorf("expected rejection for unknown command")
	}
}

func TestEvent_StringFormatting(t *testing.T) {
	uID := domain.UnitID(1)
	p := domain.PlayerOne
	p2 := domain.PlayerTwo
	pos := domain.GridPosition{X: 0, Y: -3}
	pos2 := domain.GridPosition{X: 0, Y: -2}

	events := []resolver.Event{
		resolver.NewUnitSelectedEvent(uID, p, pos),
		resolver.NewUnitMovedEvent(uID, p, pos, pos2),
		resolver.NewUnitAttackedEvent(uID, 2, domain.ElementWater, domain.NewSingleType(domain.ElementFire), 20, 2.0, 80, false),
		resolver.NewUnitDefeatedEvent(2, p2, pos2),
		resolver.NewActivationEndedEvent(p, p2),
		resolver.NewCommandRejectedEvent(p, commands.NewEndActivationCommand(), "test reason"),
		{Type: "CustomEvent"},
	}

	for _, ev := range events {
		str := ev.String()
		if len(str) == 0 {
			t.Errorf("empty string for event %s", ev.Type)
		}
	}
}
