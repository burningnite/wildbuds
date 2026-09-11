package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"wildbuds/internal/domain"
)

// Standard command validation errors.
var (
	ErrInvalidCommandType = errors.New("invalid command type")
	ErrTargetOutOfBounds  = errors.New("command target position out of grid bounds")
	ErrDestOutOfBounds    = errors.New("command destination position out of grid bounds")
	ErrInvalidSender      = errors.New("network command sender is invalid")
	ErrEmptyJSONData      = errors.New("empty JSON data for command")
)

// CommandType distinguishes the operational intent of a GameCommand.
type CommandType string

const (
	CommandSelectUnit    CommandType = "SelectUnit"
	CommandMoveUnit      CommandType = "MoveUnit"
	CommandAttack        CommandType = "Attack"
	CommandEndActivation CommandType = "EndActivation"
)

// IsValid checks whether the command type is one of the recognized operations.
func (ct CommandType) IsValid() bool {
	switch ct {
	case CommandSelectUnit, CommandMoveUnit, CommandAttack, CommandEndActivation:
		return true
	default:
		return false
	}
}

// String returns the string representation of the CommandType.
func (ct CommandType) String() string {
	return string(ct)
}

// MarshalJSON serializes CommandType as a JSON string.
func (ct CommandType) MarshalJSON() ([]byte, error) {
	if !ct.IsValid() {
		return nil, fmt.Errorf("%w: %q", ErrInvalidCommandType, ct)
	}
	return json.Marshal(string(ct))
}

// UnmarshalJSON deserializes CommandType and validates it.
func (ct *CommandType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	candidate := CommandType(s)
	if !candidate.IsValid() {
		return fmt.Errorf("%w: %q", ErrInvalidCommandType, s)
	}
	*ct = candidate
	return nil
}

// GameCommand encapsulates an action intent emitted by a client input system.
// It carries the operation type and any associated coordinate payload.
type GameCommand struct {
	Type        CommandType         `json:"type"`
	Target      domain.GridPosition `json:"target,omitempty"`
	Destination domain.GridPosition `json:"destination,omitempty"`
}

// NewSelectUnitCommand constructs a validated SelectUnit command.
func NewSelectUnitCommand(target domain.GridPosition) GameCommand {
	return GameCommand{
		Type:   CommandSelectUnit,
		Target: target,
	}
}

// NewMoveUnitCommand constructs a validated MoveUnit command.
func NewMoveUnitCommand(destination domain.GridPosition) GameCommand {
	return GameCommand{
		Type:        CommandMoveUnit,
		Destination: destination,
	}
}

// NewAttackCommand constructs a validated Attack command.
func NewAttackCommand(target domain.GridPosition) GameCommand {
	return GameCommand{
		Type:   CommandAttack,
		Target: target,
	}
}

// NewEndActivationCommand constructs a validated EndActivation command.
func NewEndActivationCommand() GameCommand {
	return GameCommand{
		Type: CommandEndActivation,
	}
}

// Validate verifies syntactic correctness and coordinate boundary constraints.
func (c GameCommand) Validate() error {
	switch c.Type {
	case CommandSelectUnit:
		if !c.Target.IsWithinBounds() {
			return ErrTargetOutOfBounds
		}
		return nil
	case CommandMoveUnit:
		if !c.Destination.IsWithinBounds() {
			return ErrDestOutOfBounds
		}
		return nil
	case CommandAttack:
		if !c.Target.IsWithinBounds() {
			return ErrTargetOutOfBounds
		}
		return nil
	case CommandEndActivation:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidCommandType, c.Type)
	}
}

// Equals checks structural and payload equivalence between two GameCommands.
func (c GameCommand) Equals(other GameCommand) bool {
	if c.Type != other.Type {
		return false
	}
	switch c.Type {
	case CommandSelectUnit, CommandAttack:
		return c.Target.Equals(other.Target)
	case CommandMoveUnit:
		return c.Destination.Equals(other.Destination)
	case CommandEndActivation:
		return true
	default:
		return false
	}
}

// String returns a human-readable representation of the GameCommand.
func (c GameCommand) String() string {
	switch c.Type {
	case CommandSelectUnit:
		return fmt.Sprintf("SelectUnit(target=%s)", c.Target)
	case CommandMoveUnit:
		return fmt.Sprintf("MoveUnit(destination=%s)", c.Destination)
	case CommandAttack:
		return fmt.Sprintf("Attack(target=%s)", c.Target)
	case CommandEndActivation:
		return "EndActivation"
	default:
		return fmt.Sprintf("UnknownCommand(%s)", c.Type)
	}
}

// MarshalJSON marshals GameCommand into minimal canonical JSON format.
func (c GameCommand) MarshalJSON() ([]byte, error) {
	if !c.Type.IsValid() {
		return nil, fmt.Errorf("%w: %q", ErrInvalidCommandType, c.Type)
	}

	switch c.Type {
	case CommandSelectUnit:
		return json.Marshal(struct {
			Type   CommandType         `json:"type"`
			Target domain.GridPosition `json:"target"`
		}{
			Type:   c.Type,
			Target: c.Target,
		})
	case CommandMoveUnit:
		return json.Marshal(struct {
			Type        CommandType         `json:"type"`
			Destination domain.GridPosition `json:"destination"`
		}{
			Type:        c.Type,
			Destination: c.Destination,
		})
	case CommandAttack:
		return json.Marshal(struct {
			Type   CommandType         `json:"type"`
			Target domain.GridPosition `json:"target"`
		}{
			Type:   c.Type,
			Target: c.Target,
		})
	case CommandEndActivation:
		return json.Marshal(struct {
			Type CommandType `json:"type"`
		}{
			Type: c.Type,
		})
	default:
		return nil, fmt.Errorf("%w: %q", ErrInvalidCommandType, c.Type)
	}
}

// UnmarshalJSON unmarshals GameCommand, supporting both canonical flat Go JSON
// and Rust Serde externally-tagged JSON formats.
func (c *GameCommand) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return ErrEmptyJSONData
	}

	// Format A: Unit string variant from Rust Serde, e.g. "EndActivation"
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err != nil {
			return err
		}
		if CommandType(s) == CommandEndActivation {
			*c = GameCommand{Type: CommandEndActivation}
			return nil
		}
		return fmt.Errorf("%w: %q", ErrInvalidCommandType, s)
	}

	// Format B: Canonical Flat Go JSON, e.g. {"type":"SelectUnit","target":{"x":0,"y":-3}}
	type flatCommand struct {
		Type        CommandType          `json:"type"`
		Target      *domain.GridPosition `json:"target,omitempty"`
		Destination *domain.GridPosition `json:"destination,omitempty"`
	}
	var flat flatCommand
	if err := json.Unmarshal(trimmed, &flat); err == nil && flat.Type != "" {
		if !flat.Type.IsValid() {
			return fmt.Errorf("%w: %q", ErrInvalidCommandType, flat.Type)
		}
		c.Type = flat.Type
		if flat.Target != nil {
			c.Target = *flat.Target
		} else {
			c.Target = domain.GridPosition{}
		}
		if flat.Destination != nil {
			c.Destination = *flat.Destination
		} else {
			c.Destination = domain.GridPosition{}
		}
		return nil
	}

	// Format C: Rust Serde externally tagged JSON, e.g.:
	// {"SelectUnit":{"target":{"x":0,"y":-3}}} or {"EndActivation":{}}
	var ext map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &ext); err != nil {
		return fmt.Errorf("malformed command JSON: %w", err)
	}

	if len(ext) != 1 {
		return fmt.Errorf("invalid command format: expected 1 key or flat 'type', got %d keys", len(ext))
	}

	for key, rawPayload := range ext {
		cmdType := CommandType(key)
		switch cmdType {
		case CommandSelectUnit:
			var payload struct {
				Target *domain.GridPosition `json:"target"`
			}
			if err := json.Unmarshal(rawPayload, &payload); err != nil {
				return fmt.Errorf("failed to parse SelectUnit payload: %w", err)
			}
			if payload.Target == nil {
				return fmt.Errorf("missing target in SelectUnit payload")
			}
			*c = GameCommand{Type: CommandSelectUnit, Target: *payload.Target}
			return nil

		case CommandMoveUnit:
			var payload struct {
				Destination *domain.GridPosition `json:"destination"`
			}
			if err := json.Unmarshal(rawPayload, &payload); err != nil {
				return fmt.Errorf("failed to parse MoveUnit payload: %w", err)
			}
			if payload.Destination == nil {
				return fmt.Errorf("missing destination in MoveUnit payload")
			}
			*c = GameCommand{Type: CommandMoveUnit, Destination: *payload.Destination}
			return nil

		case CommandAttack:
			var payload struct {
				Target *domain.GridPosition `json:"target"`
			}
			if err := json.Unmarshal(rawPayload, &payload); err != nil {
				return fmt.Errorf("failed to parse Attack payload: %w", err)
			}
			if payload.Target == nil {
				return fmt.Errorf("missing target in Attack payload")
			}
			*c = GameCommand{Type: CommandAttack, Target: *payload.Target}
			return nil

		case CommandEndActivation:
			*c = GameCommand{Type: CommandEndActivation}
			return nil

		default:
			return fmt.Errorf("%w: %q", ErrInvalidCommandType, key)
		}
	}

	return fmt.Errorf("unrecognized command JSON structure: %s", string(trimmed))
}

// NetworkCommand wraps a GameCommand with authoritative sender identity.
type NetworkCommand struct {
	Sender  domain.Player `json:"sender"`
	Command GameCommand   `json:"command"`
}

// NewNetworkCommand constructs a NetworkCommand.
func NewNetworkCommand(sender domain.Player, cmd GameCommand) NetworkCommand {
	return NetworkCommand{
		Sender:  sender,
		Command: cmd,
	}
}

// Validate verifies sender validity and command validity.
func (nc NetworkCommand) Validate() error {
	if !nc.Sender.IsValid() {
		return ErrInvalidSender
	}
	return nc.Command.Validate()
}

// Equals checks structural equality between two NetworkCommands.
func (nc NetworkCommand) Equals(other NetworkCommand) bool {
	return nc.Sender == other.Sender && nc.Command.Equals(other.Command)
}

// String provides a human-readable representation of the NetworkCommand.
func (nc NetworkCommand) String() string {
	return fmt.Sprintf("NetworkCommand(sender=%s, cmd=%s)", nc.Sender, nc.Command)
}
