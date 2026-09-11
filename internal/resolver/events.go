package resolver

import (
	"fmt"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
)

// EventType enumerates the types of domain events produced by the Resolver.
type EventType string

const (
	EventUnitSelected    EventType = "UnitSelected"
	EventUnitMoved       EventType = "UnitMoved"
	EventUnitAttacked    EventType = "UnitAttacked"
	EventUnitDefeated    EventType = "UnitDefeated"
	EventActivationEnded EventType = "ActivationEnded"
	EventCommandRejected EventType = "CommandRejected"

	// Compatibility aliases matching alternative event naming conventions
	EventTypeUnitSelected    = EventUnitSelected
	EventTypeUnitMoved       = EventUnitMoved
	EventTypeUnitAttacked    = EventUnitAttacked
	EventTypeUnitDefeated    = EventUnitDefeated
	EventTypeActivationEnded = EventActivationEnded
	EventTypeCommandRejected = EventCommandRejected
)

// Event represents an authoritative state change or command rejection emitted by Resolver.
type Event struct {
	Type EventType `json:"type"`

	// Unit identification
	UnitID   *domain.UnitID       `json:"unit_id,omitempty"`
	Player   *domain.Player       `json:"player,omitempty"`
	Position *domain.GridPosition `json:"position,omitempty"`
	FromPos  *domain.GridPosition `json:"from_position,omitempty"`
	ToPos    *domain.GridPosition `json:"to_position,omitempty"`

	// Combat resolution details
	AttackerID   *domain.UnitID   `json:"attacker_id,omitempty"`
	DefenderID   *domain.UnitID   `json:"defender_id,omitempty"`
	AttackerType *domain.Element  `json:"attacker_type,omitempty"`
	DefenderType *domain.DualType `json:"defender_type,omitempty"`
	Damage       uint32           `json:"damage,omitempty"`
	Multiplier   float32          `json:"multiplier,omitempty"`
	RemainingHP  uint32           `json:"remaining_hp,omitempty"`
	Defeated     bool             `json:"defeated,omitempty"`

	// Turn & activation transitions
	PreviousPlayer *domain.Player `json:"previous_player,omitempty"`
	NextPlayer     *domain.Player `json:"next_player,omitempty"`

	// Rejection metadata
	Sender  *domain.Player        `json:"sender,omitempty"`
	Command *commands.GameCommand `json:"command,omitempty"`
	Reason  string                `json:"reason,omitempty"`
}

// String returns a human-readable description of the event.
func (e Event) String() string {
	switch e.Type {
	case EventUnitSelected:
		uid := domain.UnitID(0)
		if e.UnitID != nil {
			uid = *e.UnitID
		}
		p := domain.PlayerNone
		if e.Player != nil {
			p = *e.Player
		}
		pos := domain.GridPosition{}
		if e.Position != nil {
			pos = *e.Position
		}
		return fmt.Sprintf("UnitSelected: Unit %d by %s at %v", uid, p, pos)
	case EventUnitMoved:
		uid := domain.UnitID(0)
		if e.UnitID != nil {
			uid = *e.UnitID
		}
		p := domain.PlayerNone
		if e.Player != nil {
			p = *e.Player
		}
		from := domain.GridPosition{}
		if e.FromPos != nil {
			from = *e.FromPos
		}
		to := domain.GridPosition{}
		if e.ToPos != nil {
			to = *e.ToPos
		}
		return fmt.Sprintf("UnitMoved: Unit %d by %s from %v to %v", uid, p, from, to)
	case EventUnitAttacked:
		atk := domain.UnitID(0)
		if e.AttackerID != nil {
			atk = *e.AttackerID
		}
		def := domain.UnitID(0)
		if e.DefenderID != nil {
			def = *e.DefenderID
		}
		return fmt.Sprintf("UnitAttacked: Attacker %d -> Defender %d | Dmg=%d Mult=%.2f RemHP=%d Defeated=%t",
			atk, def, e.Damage, e.Multiplier, e.RemainingHP, e.Defeated)
	case EventUnitDefeated:
		uid := domain.UnitID(0)
		if e.UnitID != nil {
			uid = *e.UnitID
		}
		p := domain.PlayerNone
		if e.Player != nil {
			p = *e.Player
		}
		pos := domain.GridPosition{}
		if e.Position != nil {
			pos = *e.Position
		}
		return fmt.Sprintf("UnitDefeated: Unit %d (%s) defeated at %v", uid, p, pos)
	case EventActivationEnded:
		prev := domain.PlayerNone
		if e.PreviousPlayer != nil {
			prev = *e.PreviousPlayer
		}
		next := domain.PlayerNone
		if e.NextPlayer != nil {
			next = *e.NextPlayer
		}
		return fmt.Sprintf("ActivationEnded: %s -> %s", prev, next)
	case EventCommandRejected:
		snd := domain.PlayerNone
		if e.Sender != nil {
			snd = *e.Sender
		}
		return fmt.Sprintf("CommandRejected: Sender %s, Reason: %s", snd, e.Reason)
	default:
		return fmt.Sprintf("Event(%s)", e.Type)
	}
}

// NewUnitSelectedEvent constructs a UnitSelected event.
func NewUnitSelectedEvent(id domain.UnitID, player domain.Player, pos domain.GridPosition) Event {
	unitID := id
	p := player
	posCopy := pos
	return Event{
		Type:     EventUnitSelected,
		UnitID:   &unitID,
		Player:   &p,
		Position: &posCopy,
	}
}

// NewUnitMovedEvent constructs a UnitMoved event.
func NewUnitMovedEvent(id domain.UnitID, player domain.Player, from, to domain.GridPosition) Event {
	unitID := id
	p := player
	fromCopy := from
	toCopy := to
	return Event{
		Type:    EventUnitMoved,
		UnitID:  &unitID,
		Player:  &p,
		FromPos: &fromCopy,
		ToPos:   &toCopy,
	}
}

// NewUnitAttackedEvent constructs a UnitAttacked event.
func NewUnitAttackedEvent(
	attackerID, defenderID domain.UnitID,
	attackerType domain.Element,
	defenderType domain.DualType,
	damage uint32,
	multiplier float32,
	remainingHP uint32,
	defeated bool,
) Event {
	atkID := attackerID
	defID := defenderID
	atkType := attackerType
	defType := defenderType.Clone()
	return Event{
		Type:         EventUnitAttacked,
		AttackerID:   &atkID,
		DefenderID:   &defID,
		AttackerType: &atkType,
		DefenderType: &defType,
		Damage:       damage,
		Multiplier:   multiplier,
		RemainingHP:  remainingHP,
		Defeated:     defeated,
	}
}

// NewUnitDefeatedEvent constructs a UnitDefeated event.
func NewUnitDefeatedEvent(id domain.UnitID, player domain.Player, pos domain.GridPosition) Event {
	unitID := id
	p := player
	posCopy := pos
	return Event{
		Type:     EventUnitDefeated,
		UnitID:   &unitID,
		Player:   &p,
		Position: &posCopy,
	}
}

// NewActivationEndedEvent constructs an ActivationEnded event.
func NewActivationEndedEvent(prev, next domain.Player) Event {
	p1 := prev
	p2 := next
	return Event{
		Type:           EventActivationEnded,
		PreviousPlayer: &p1,
		NextPlayer:     &p2,
	}
}

// NewCommandRejectedEvent constructs a CommandRejected event.
func NewCommandRejectedEvent(sender domain.Player, cmd commands.GameCommand, reason string) Event {
	p := sender
	cmdCopy := cmd
	return Event{
		Type:    EventCommandRejected,
		Sender:  &p,
		Command: &cmdCopy,
		Reason:  reason,
	}
}
