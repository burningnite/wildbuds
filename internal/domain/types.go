package domain

import (
	"encoding/json"
	"fmt"
)

// Board geometry and visual layout constants matching Rust src/render.rs.
const (
	GridMin          = -5
	GridMax          = 5
	GridSize         = 11 // Range [-5, 5] inclusive: (GridMax - GridMin + 1)
	TotalTiles       = 121 // 11 x 11 board

	TileSize         = 30.0
	TileVisualSize   = 30.0
	TileStride       = 32.0
	TilePitch        = 32.0
	TileGap          = 2.0 // TilePitch - TileVisualSize
	UnitVisualSize   = 24.0
	CursorVisualSize = 32.0

	LayerBoard  = 0.0
	LayerUnit   = 1.0
	LayerCursor = 2.0
)

// Player represents a match participant.
type Player int

const (
	PlayerNone Player = iota
	PlayerOne
	PlayerTwo
)

// Next toggles the active player between PlayerOne and PlayerTwo.
// Calling Next on PlayerNone or any invalid player returns PlayerNone.
func (p Player) Next() Player {
	switch p {
	case PlayerOne:
		return PlayerTwo
	case PlayerTwo:
		return PlayerOne
	default:
		return PlayerNone
	}
}

// IsValid checks whether the player is an active participant (PlayerOne or PlayerTwo).
func (p Player) IsValid() bool {
	return p == PlayerOne || p == PlayerTwo
}

// String provides a human-readable representation.
func (p Player) String() string {
	switch p {
	case PlayerOne:
		return "PlayerOne"
	case PlayerTwo:
		return "PlayerTwo"
	default:
		return "PlayerNone"
	}
}

// MarshalJSON serializes Player to JSON string matching Rust serde ("One", "Two", "None").
func (p Player) MarshalJSON() ([]byte, error) {
	switch p {
	case PlayerOne:
		return json.Marshal("One")
	case PlayerTwo:
		return json.Marshal("Two")
	default:
		return json.Marshal("None")
	}
}

// UnmarshalJSON deserializes Player from JSON string or numeric format.
func (p *Player) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		switch s {
		case "One", "PlayerOne":
			*p = PlayerOne
			return nil
		case "Two", "PlayerTwo":
			*p = PlayerTwo
			return nil
		case "None", "PlayerNone":
			*p = PlayerNone
			return nil
		default:
			return fmt.Errorf("invalid player string: %q", s)
		}
	}

	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		switch Player(i) {
		case PlayerOne:
			*p = PlayerOne
			return nil
		case PlayerTwo:
			*p = PlayerTwo
			return nil
		case PlayerNone:
			*p = PlayerNone
			return nil
		default:
			return fmt.Errorf("invalid player integer: %d", i)
		}
	}

	return fmt.Errorf("invalid player data: %s", string(data))
}

// TurnPhase represents the phase within a player's turn.
type TurnPhase string

const (
	PhaseSelectUnit   TurnPhase = "SelectUnit"
	PhaseChooseAction TurnPhase = "ChooseAction"
)

// IsValid checks whether the phase is a recognized turn phase.
func (tp TurnPhase) IsValid() bool {
	return tp == PhaseSelectUnit || tp == PhaseChooseAction
}

// String returns the string value of the turn phase.
func (tp TurnPhase) String() string {
	return string(tp)
}

// MarshalJSON serializes TurnPhase to JSON string.
func (tp TurnPhase) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(tp))
}

// UnmarshalJSON deserializes TurnPhase from JSON string or integer index.
func (tp *TurnPhase) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		switch s {
		case "SelectUnit":
			*tp = PhaseSelectUnit
			return nil
		case "ChooseAction":
			*tp = PhaseChooseAction
			return nil
		default:
			return fmt.Errorf("invalid turn phase string: %q", s)
		}
	}

	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		switch i {
		case 0:
			*tp = PhaseSelectUnit
			return nil
		case 1:
			*tp = PhaseChooseAction
			return nil
		default:
			return fmt.Errorf("invalid turn phase integer: %d", i)
		}
	}

	return fmt.Errorf("invalid turn phase data: %s", string(data))
}

// GridPosition represents a discrete coordinate on the board.
type GridPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Equals returns true if two positions share identical coordinates.
func (gp GridPosition) Equals(other GridPosition) bool {
	return gp.X == other.X && gp.Y == other.Y
}

// Distance returns the Manhattan distance (|dx| + |dy|) between two grid positions.
func (gp GridPosition) Distance(other GridPosition) int {
	dx := gp.X - other.X
	if dx < 0 {
		dx = -dx
	}
	dy := gp.Y - other.Y
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

// InBounds returns true if the position is within the specified inclusive range.
func (gp GridPosition) InBounds(minCoord, maxCoord int) bool {
	return gp.X >= minCoord && gp.X <= maxCoord && gp.Y >= minCoord && gp.Y <= maxCoord
}

// IsWithinBounds returns true if the position is within standard board boundaries [-5, 5].
func (gp GridPosition) IsWithinBounds() bool {
	return gp.InBounds(GridMin, GridMax)
}

// IsDarkTile returns true if the coordinate corresponds to a dark checkerboard tile.
// Parity rule matching Rust src/render.rs: (x + y) % 2 == 0.
func (gp GridPosition) IsDarkTile() bool {
	return (gp.X+gp.Y)%2 == 0
}

// String formats GridPosition as (X, Y).
func (gp GridPosition) String() string {
	return fmt.Sprintf("(%d, %d)", gp.X, gp.Y)
}

// InGridBounds checks if position is within standard board boundaries [-5, 5].
func InGridBounds(pos GridPosition) bool {
	return pos.IsWithinBounds()
}

// UnitID is a unique identifier for a unit.
type UnitID int
