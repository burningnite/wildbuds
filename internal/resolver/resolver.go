package resolver

import (
	"math"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
)

// Rejection reason constants for diagnostic assertions and event payload inspection.
const (
	ReasonNotActivePlayer      = "sender is not the active player"
	ReasonNoUnitAtTarget       = "no unit at target position"
	ReasonUnitNotOwned         = "cannot select unit belonging to opponent"
	ReasonTargetOutOfBounds    = "target coordinate out of bounds"
	ReasonDestOutOfBounds      = "destination coordinate out of bounds"
	ReasonNoActiveUnit         = "no active unit selected"
	ReasonActiveUnitNotFound   = "active unit not found in state"
	ReasonActiveUnitNotOwned   = "active unit does not belong to active player"
	ReasonDestinationOccupied  = "destination tile is occupied by another unit"
	ReasonCannotAttackFriendly = "cannot attack friendly unit"
	ReasonUnknownCommand       = "unknown or unsupported command type"
)

// CombatHook defines the pluggable signature for resolving attacks between two units.
type CombatHook func(attacker *domain.Unit, defender *domain.Unit) (damage uint32, multiplier float32)

// DefaultCombatHook calculates damage strictly using domain.CalculateMultiplier:
// multiplier = domain.CalculateMultiplier(attacker.Types.Primary, defender.Types)
// rawDamage  = attacker.Stats.Attack * multiplier
// damage     = round(rawDamage) (minimum 1 if attacker.Attack > 0 and mult > 0)
func DefaultCombatHook(attacker *domain.Unit, defender *domain.Unit) (uint32, float32) {
	mult := domain.CalculateMultiplier(attacker.Types.Primary, defender.Types)
	raw := float32(attacker.Stats.Attack) * mult
	dmg := uint32(math.Round(float64(raw)))
	if dmg == 0 && attacker.Stats.Attack > 0 && mult > 0 {
		dmg = 1
	}
	return dmg, mult
}

// Resolver executes network commands authoritatively against GameState.
type Resolver struct {
	combatHook CombatHook
}

// Option configures a Resolver instance.
type Option func(*Resolver)

// WithCombatHook supplies a custom combat resolution hook.
func WithCombatHook(hook CombatHook) Option {
	return func(r *Resolver) {
		if hook != nil {
			r.combatHook = hook
		}
	}
}

// New creates an authoritative Resolver with the default elemental combat hook.
func New(opts ...Option) *Resolver {
	r := &Resolver{
		combatHook: DefaultCombatHook,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// NewResolver is an alias constructor for New, ensuring seamless compatibility.
func NewResolver(opts ...Option) *Resolver {
	return New(opts...)
}

// Resolve processes all pending incoming commands in strict FIFO order,
// mutates state authoritatively, and returns all emitted domain events.
func (r *Resolver) Resolve(state *domain.GameState, queue *commands.CommandQueue) []Event {
	if state == nil || queue == nil {
		return nil
	}

	var allEvents []Event
	for {
		cmd, ok := queue.PopIncoming()
		if !ok {
			break
		}
		events := r.ResolveCommand(state, cmd)
		allEvents = append(allEvents, events...)
	}
	return allEvents
}

// ResolveCommand processes a single network command against authoritative state.
// If any precondition fails, zero state mutations occur and an EventCommandRejected is emitted.
func (r *Resolver) ResolveCommand(state *domain.GameState, cmd commands.NetworkCommand) []Event {
	if state == nil {
		return nil
	}

	// 1. Gatekeeper: Sender must be an active participant and the currently active player
	if !cmd.Sender.IsValid() || cmd.Sender != state.ActivePlayer {
		return []Event{NewCommandRejectedEvent(cmd.Sender, cmd.Command, ReasonNotActivePlayer)}
	}

	// 2. Dispatch command
	switch cmd.Command.Type {
	case commands.CommandSelectUnit:
		return r.resolveSelectUnit(state, cmd)
	case commands.CommandMoveUnit:
		return r.resolveMoveUnit(state, cmd)
	case commands.CommandAttack:
		return r.resolveAttack(state, cmd)
	case commands.CommandEndActivation:
		return r.resolveEndActivation(state, cmd)
	default:
		return []Event{NewCommandRejectedEvent(cmd.Sender, cmd.Command, ReasonUnknownCommand)}
	}
}

// resolveSelectUnit validates and activates a unit at the target position.
func (r *Resolver) resolveSelectUnit(state *domain.GameState, cmd commands.NetworkCommand) []Event {
	target := cmd.Command.Target

	if !target.IsWithinBounds() {
		return []Event{NewCommandRejectedEvent(cmd.Sender, cmd.Command, ReasonTargetOutOfBounds)}
	}

	unit := state.GetUnitAt(target)
	if unit == nil {
		return []Event{NewCommandRejectedEvent(cmd.Sender, cmd.Command, ReasonNoUnitAtTarget)}
	}

	if unit.Owner != cmd.Sender {
		return []Event{NewCommandRejectedEvent(cmd.Sender, cmd.Command, ReasonUnitNotOwned)}
	}

	// State Mutation
	state.SetActiveUnit(&unit.ID)
	state.Phase = domain.PhaseChooseAction

	return []Event{NewUnitSelectedEvent(unit.ID, cmd.Sender, unit.Position)}
}

// resolveMoveUnit moves the currently active unit to the destination tile.
func (r *Resolver) resolveMoveUnit(state *domain.GameState, cmd commands.NetworkCommand) []Event {
	if state.ActiveUnitID == nil {
		return []Event{NewCommandRejectedEvent(cmd.Sender, cmd.Command, ReasonNoActiveUnit)}
	}

	activeUnit := state.GetActiveUnit()
	if activeUnit == nil {
		return []Event{NewCommandRejectedEvent(cmd.Sender, cmd.Command, ReasonActiveUnitNotFound)}
	}

	if activeUnit.Owner != cmd.Sender {
		return []Event{NewCommandRejectedEvent(cmd.Sender, cmd.Command, ReasonActiveUnitNotOwned)}
	}

	dest := cmd.Command.Destination
	if !dest.IsWithinBounds() {
		return []Event{NewCommandRejectedEvent(cmd.Sender, cmd.Command, ReasonDestOutOfBounds)}
	}

	// Check if destination is occupied by another unit
	occupant := state.GetUnitAt(dest)
	if occupant != nil && occupant.ID != activeUnit.ID {
		return []Event{NewCommandRejectedEvent(cmd.Sender, cmd.Command, ReasonDestinationOccupied)}
	}

	// State Mutation
	fromPos := activeUnit.Position
	activeUnit.Position = dest

	return []Event{NewUnitMovedEvent(activeUnit.ID, cmd.Sender, fromPos, dest)}
}

// resolveAttack conducts combat between the active unit and the target position.
func (r *Resolver) resolveAttack(state *domain.GameState, cmd commands.NetworkCommand) []Event {
	if state.ActiveUnitID == nil {
		return []Event{NewCommandRejectedEvent(cmd.Sender, cmd.Command, ReasonNoActiveUnit)}
	}

	activeUnit := state.GetActiveUnit()
	if activeUnit == nil {
		return []Event{NewCommandRejectedEvent(cmd.Sender, cmd.Command, ReasonActiveUnitNotFound)}
	}

	if activeUnit.Owner != cmd.Sender {
		return []Event{NewCommandRejectedEvent(cmd.Sender, cmd.Command, ReasonActiveUnitNotOwned)}
	}

	targetPos := cmd.Command.Target
	if !targetPos.IsWithinBounds() {
		return []Event{NewCommandRejectedEvent(cmd.Sender, cmd.Command, ReasonTargetOutOfBounds)}
	}

	targetUnit := state.GetUnitAt(targetPos)
	if targetUnit == nil {
		return []Event{NewCommandRejectedEvent(cmd.Sender, cmd.Command, ReasonNoUnitAtTarget)}
	}

	if targetUnit.Owner == cmd.Sender {
		return []Event{NewCommandRejectedEvent(cmd.Sender, cmd.Command, ReasonCannotAttackFriendly)}
	}

	// Combat Calculation
	damage, mult := r.combatHook(activeUnit, targetUnit)

	// State Mutation: Deduct HP
	if damage >= targetUnit.Stats.HP {
		targetUnit.Stats.HP = 0
	} else {
		targetUnit.Stats.HP -= damage
	}

	defeated := (targetUnit.Stats.HP == 0)

	events := []Event{
		NewUnitAttackedEvent(
			activeUnit.ID,
			targetUnit.ID,
			activeUnit.Types.Primary,
			targetUnit.Types,
			damage,
			mult,
			targetUnit.Stats.HP,
			defeated,
		),
	}

	if defeated {
		events = append(events, NewUnitDefeatedEvent(targetUnit.ID, targetUnit.Owner, targetUnit.Position))
	}

	return events
}

// resolveEndActivation clears active unit, advances active player, and resets phase to SelectUnit.
func (r *Resolver) resolveEndActivation(state *domain.GameState, cmd commands.NetworkCommand) []Event {
	prevPlayer := state.ActivePlayer
	nextPlayer := prevPlayer.Next()

	// State Mutation
	state.ClearActiveUnit()
	state.ActivePlayer = nextPlayer
	state.Phase = domain.PhaseSelectUnit

	return []Event{NewActivationEndedEvent(prevPlayer, nextPlayer)}
}
