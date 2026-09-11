package domain

import (
	"errors"
)

// Standard state validation errors.
var (
	ErrInvalidActivePlayer = errors.New("active player must be PlayerOne or PlayerTwo")
	ErrInvalidPhase        = errors.New("invalid turn phase")
	ErrActiveUnitRequired  = errors.New("active unit required in ChooseAction phase")
	ErrActiveUnitForbidden = errors.New("active unit must be nil in SelectUnit phase")
	ErrUnitNotFound        = errors.New("active unit ID not found in units list")
	ErrUnitOwnerMismatch   = errors.New("active unit does not belong to active player")
	ErrUnitOutOfBounds     = errors.New("unit position out of grid bounds")
	ErrOverlappingUnits    = errors.New("multiple units occupy the same grid position")
	ErrDuplicateUnitID     = errors.New("duplicate unit ID in units list")
)

// Unit represents an active game piece on the board.
type Unit struct {
	ID       UnitID       `json:"id"`
	Owner    Player       `json:"owner"`
	Position GridPosition `json:"position"`
	Stats    BaseStats    `json:"stats"`
	Tokens   ActionTokens `json:"tokens"`
	Types    DualType     `json:"types"`
}

// Clone creates an independent deep copy of a Unit.
func (u *Unit) Clone() *Unit {
	if u == nil {
		return nil
	}
	clone := *u
	clone.Types = u.Types.Clone()
	return &clone
}

// GameState is the authoritative state of the game match.
type GameState struct {
	ActivePlayer Player    `json:"active_player"`
	ActiveUnitID *UnitID   `json:"active_unit_id,omitempty"`
	Phase        TurnPhase `json:"phase"`
	Units        []*Unit   `json:"units"`
}

// NewInitialGameState constructs the exact starting match state matching Rust setup_board:
// - Unit 1: PlayerOne at (0, -3), Water element
// - Unit 2: PlayerTwo at (0, 3), Fire element
// - ActivePlayer: PlayerOne
// - ActiveUnitID: nil
// - Phase: PhaseSelectUnit
func NewInitialGameState() *GameState {
	unit1 := &Unit{
		ID:       1,
		Owner:    PlayerOne,
		Position: GridPosition{X: 0, Y: -3},
		Stats:    DefaultBaseStats(),
		Tokens:   DefaultActionTokens(),
		Types:    NewSingleType(ElementWater),
	}

	unit2 := &Unit{
		ID:       2,
		Owner:    PlayerTwo,
		Position: GridPosition{X: 0, Y: 3},
		Stats:    DefaultBaseStats(),
		Tokens:   DefaultActionTokens(),
		Types:    NewSingleType(ElementFire),
	}

	return &GameState{
		ActivePlayer: PlayerOne,
		ActiveUnitID: nil,
		Phase:        PhaseSelectUnit,
		Units:        []*Unit{unit1, unit2},
	}
}

// Clone creates a deep, independent copy of GameState.
func (s *GameState) Clone() *GameState {
	if s == nil {
		return nil
	}

	clone := &GameState{
		ActivePlayer: s.ActivePlayer,
		Phase:        s.Phase,
		Units:        make([]*Unit, len(s.Units)),
	}

	if s.ActiveUnitID != nil {
		id := *s.ActiveUnitID
		clone.ActiveUnitID = &id
	}

	for i, u := range s.Units {
		if u != nil {
			clone.Units[i] = u.Clone()
		}
	}

	return clone
}

// GetActiveUnit returns a pointer to the currently active unit, or nil if none.
func (s *GameState) GetActiveUnit() *Unit {
	if s.ActiveUnitID == nil {
		return nil
	}
	return s.GetUnit(*s.ActiveUnitID)
}

// GetUnit finds a unit by its UnitID. Returns nil if not found.
func (s *GameState) GetUnit(id UnitID) *Unit {
	for _, u := range s.Units {
		if u != nil && u.ID == id {
			return u
		}
	}
	return nil
}

// GetUnitAt returns the unit occupying pos, or nil if tile is unoccupied.
func (s *GameState) GetUnitAt(pos GridPosition) *Unit {
	for _, u := range s.Units {
		if u != nil && u.Position.Equals(pos) {
			return u
		}
	}
	return nil
}

// FindUnitAt is an alias for GetUnitAt.
func (s *GameState) FindUnitAt(pos GridPosition) *Unit {
	return s.GetUnitAt(pos)
}

// GetUnitsByOwner returns all units owned by the specified player.
func (s *GameState) GetUnitsByOwner(owner Player) []*Unit {
	var result []*Unit
	for _, u := range s.Units {
		if u != nil && u.Owner == owner {
			result = append(result, u)
		}
	}
	return result
}

// SetActiveUnit sets the active unit ID (or clears it if id is nil).
func (s *GameState) SetActiveUnit(id *UnitID) {
	if id == nil {
		s.ActiveUnitID = nil
		return
	}
	val := *id
	s.ActiveUnitID = &val
}

// ClearActiveUnit resets the active unit to nil.
func (s *GameState) ClearActiveUnit() {
	s.ActiveUnitID = nil
}

// Validate checks all authoritative state invariants:
// 1. ActivePlayer must be PlayerOne or PlayerTwo.
// 2. Phase must be PhaseSelectUnit or PhaseChooseAction.
// 3. PhaseSelectUnit requires ActiveUnitID == nil.
// 4. PhaseChooseAction requires non-nil ActiveUnitID pointing to a unit owned by ActivePlayer.
// 5. All units must have coordinates within [-5, 5].
// 6. No two units may share identical coordinates.
// 7. All unit IDs must be unique.
func (s *GameState) Validate() error {
	if !s.ActivePlayer.IsValid() {
		return ErrInvalidActivePlayer
	}
	if !s.Phase.IsValid() {
		return ErrInvalidPhase
	}
	if s.Phase == PhaseSelectUnit && s.ActiveUnitID != nil {
		return ErrActiveUnitForbidden
	}
	if s.Phase == PhaseChooseAction {
		if s.ActiveUnitID == nil {
			return ErrActiveUnitRequired
		}
		activeUnit := s.GetUnit(*s.ActiveUnitID)
		if activeUnit == nil {
			return ErrUnitNotFound
		}
		if activeUnit.Owner != s.ActivePlayer {
			return ErrUnitOwnerMismatch
		}
	}

	seenPositions := make(map[GridPosition]bool)
	seenIDs := make(map[UnitID]bool)
	for _, u := range s.Units {
		if u == nil {
			continue
		}
		if !u.Position.IsWithinBounds() {
			return ErrUnitOutOfBounds
		}
		if seenPositions[u.Position] {
			return ErrOverlappingUnits
		}
		seenPositions[u.Position] = true

		if seenIDs[u.ID] {
			return ErrDuplicateUnitID
		}
		seenIDs[u.ID] = true
	}

	return nil
}
