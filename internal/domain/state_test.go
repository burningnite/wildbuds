package domain

import (
	"encoding/json"
	"testing"
)

func TestBoardConstants(t *testing.T) {
	if GridMin != -5 {
		t.Errorf("GridMin = %d, want -5", GridMin)
	}
	if GridMax != 5 {
		t.Errorf("GridMax = %d, want 5", GridMax)
	}
	if GridSize != 11 {
		t.Errorf("GridSize = %d, want 11", GridSize)
	}
	if TotalTiles != 121 {
		t.Errorf("TotalTiles = %d, want 121", TotalTiles)
	}
	if TileSize != 30.0 || TileVisualSize != 30.0 {
		t.Errorf("TileSize = %v, want 30.0", TileSize)
	}
	if TileStride != 32.0 || TilePitch != 32.0 {
		t.Errorf("TileStride = %v, want 32.0", TileStride)
	}
	if TileGap != 2.0 {
		t.Errorf("TileGap = %v, want 2.0", TileGap)
	}
	if UnitVisualSize != 24.0 {
		t.Errorf("UnitVisualSize = %v, want 24.0", UnitVisualSize)
	}
	if CursorVisualSize != 32.0 {
		t.Errorf("CursorVisualSize = %v, want 32.0", CursorVisualSize)
	}
	if LayerBoard != 0.0 || LayerUnit != 1.0 || LayerCursor != 2.0 {
		t.Errorf("Layer depths invalid: board=%v, unit=%v, cursor=%v", LayerBoard, LayerUnit, LayerCursor)
	}
}

func TestPlayer_Rotation(t *testing.T) {
	p1 := PlayerOne
	p2 := p1.Next()
	if p2 != PlayerTwo {
		t.Fatalf("expected PlayerTwo, got %v", p2)
	}

	p1Again := p2.Next()
	if p1Again != PlayerOne {
		t.Fatalf("expected PlayerOne, got %v", p1Again)
	}

	pNone := PlayerNone.Next()
	if pNone != PlayerNone {
		t.Fatalf("expected PlayerNone, got %v", pNone)
	}

	pInvalid := Player(99).Next()
	if pInvalid != PlayerNone {
		t.Fatalf("expected PlayerNone, got %v", pInvalid)
	}

	if !PlayerOne.IsValid() || !PlayerTwo.IsValid() {
		t.Fatalf("PlayerOne and PlayerTwo must be valid")
	}
	if PlayerNone.IsValid() || Player(99).IsValid() {
		t.Fatalf("PlayerNone and Player(99) must be invalid")
	}

	if PlayerOne.String() != "PlayerOne" || PlayerTwo.String() != "PlayerTwo" || PlayerNone.String() != "PlayerNone" {
		t.Fatalf("unexpected Player string values")
	}
}

func TestPlayer_JSONSerialization(t *testing.T) {
	data, err := json.Marshal(PlayerOne)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if string(data) != `"One"` {
		t.Fatalf("expected \"One\", got %s", string(data))
	}

	data2, err := json.Marshal(PlayerTwo)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if string(data2) != `"Two"` {
		t.Fatalf("expected \"Two\", got %s", string(data2))
	}

	dataNone, err := json.Marshal(PlayerNone)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if string(dataNone) != `"None"` {
		t.Fatalf("expected \"None\", got %s", string(dataNone))
	}

	var p Player
	if err := json.Unmarshal([]byte(`"Two"`), &p); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if p != PlayerTwo {
		t.Fatalf("expected PlayerTwo, got %v", p)
	}

	if err := json.Unmarshal([]byte(`"PlayerOne"`), &p); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if p != PlayerOne {
		t.Fatalf("expected PlayerOne, got %v", p)
	}

	if err := json.Unmarshal([]byte(`"PlayerTwo"`), &p); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if p != PlayerTwo {
		t.Fatalf("expected PlayerTwo, got %v", p)
	}

	if err := json.Unmarshal([]byte(`"None"`), &p); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if p != PlayerNone {
		t.Fatalf("expected PlayerNone, got %v", p)
	}

	// Test unmarshaling numeric format
	if err := json.Unmarshal([]byte(`1`), &p); err != nil {
		t.Fatalf("unmarshal numeric error: %v", err)
	}
	if p != PlayerOne {
		t.Fatalf("expected PlayerOne, got %v", p)
	}

	if err := json.Unmarshal([]byte(`2`), &p); err != nil {
		t.Fatalf("unmarshal numeric error: %v", err)
	}
	if p != PlayerTwo {
		t.Fatalf("expected PlayerTwo, got %v", p)
	}

	if err := json.Unmarshal([]byte(`0`), &p); err != nil {
		t.Fatalf("unmarshal numeric error: %v", err)
	}
	if p != PlayerNone {
		t.Fatalf("expected PlayerNone, got %v", p)
	}

	// Invalid string error
	if err := json.Unmarshal([]byte(`"InvalidPlayer"`), &p); err == nil {
		t.Fatalf("expected error for invalid player string")
	}

	// Invalid integer error
	if err := json.Unmarshal([]byte(`99`), &p); err == nil {
		t.Fatalf("expected error for invalid player integer")
	}
}

func TestTurnPhase_PropertiesAndJSON(t *testing.T) {
	if !PhaseSelectUnit.IsValid() {
		t.Errorf("PhaseSelectUnit should be valid")
	}
	if !PhaseChooseAction.IsValid() {
		t.Errorf("PhaseChooseAction should be valid")
	}
	if TurnPhase("UnknownPhase").IsValid() {
		t.Errorf("UnknownPhase should not be valid")
	}

	if PhaseSelectUnit.String() != "SelectUnit" {
		t.Errorf("PhaseSelectUnit string = %s, want 'SelectUnit'", PhaseSelectUnit.String())
	}
	if PhaseChooseAction.String() != "ChooseAction" {
		t.Errorf("PhaseChooseAction string = %s, want 'ChooseAction'", PhaseChooseAction.String())
	}

	// JSON Marshal
	data, err := json.Marshal(PhaseSelectUnit)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if string(data) != `"SelectUnit"` {
		t.Fatalf("expected \"SelectUnit\", got %s", string(data))
	}

	// JSON Unmarshal string
	var tp TurnPhase
	if err := json.Unmarshal([]byte(`"ChooseAction"`), &tp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if tp != PhaseChooseAction {
		t.Fatalf("expected PhaseChooseAction, got %v", tp)
	}

	// JSON Unmarshal int
	if err := json.Unmarshal([]byte(`0`), &tp); err != nil {
		t.Fatalf("unmarshal int error: %v", err)
	}
	if tp != PhaseSelectUnit {
		t.Fatalf("expected PhaseSelectUnit from 0, got %v", tp)
	}

	if err := json.Unmarshal([]byte(`1`), &tp); err != nil {
		t.Fatalf("unmarshal int error: %v", err)
	}
	if tp != PhaseChooseAction {
		t.Fatalf("expected PhaseChooseAction from 1, got %v", tp)
	}

	if err := json.Unmarshal([]byte(`"InvalidPhase"`), &tp); err == nil {
		t.Fatalf("expected error for invalid turn phase string")
	}
	if err := json.Unmarshal([]byte(`5`), &tp); err == nil {
		t.Fatalf("expected error for invalid turn phase integer")
	}
}

func TestGridPosition_Operations(t *testing.T) {
	pos1 := GridPosition{X: 0, Y: -3}
	pos2 := GridPosition{X: 0, Y: 3}
	posEqual := GridPosition{X: 0, Y: -3}

	if !pos1.Equals(posEqual) {
		t.Fatalf("expected pos1 == posEqual")
	}
	if pos1.Equals(pos2) {
		t.Fatalf("expected pos1 != pos2")
	}

	dist := pos1.Distance(pos2)
	if dist != 6 {
		t.Fatalf("expected distance 6, got %d", dist)
	}

	// Distance with negative and diagonal
	posDiag1 := GridPosition{X: -2, Y: -1}
	posDiag2 := GridPosition{X: 3, Y: 4}
	if posDiag1.Distance(posDiag2) != 10 { // |(-2) - 3| + |(-1) - 4| = 5 + 5 = 10
		t.Fatalf("expected distance 10, got %d", posDiag1.Distance(posDiag2))
	}

	if !pos1.InBounds(GridMin, GridMax) {
		t.Fatalf("pos1 should be in bounds")
	}
	if !pos1.IsWithinBounds() {
		t.Fatalf("pos1 should be within bounds")
	}
	if !InGridBounds(pos1) {
		t.Fatalf("pos1 should be InGridBounds")
	}

	outPos := GridPosition{X: 6, Y: 0}
	if outPos.IsWithinBounds() {
		t.Fatalf("outPos (6,0) should be out of bounds")
	}
	if InGridBounds(outPos) {
		t.Fatalf("outPos (6,0) should not be in bounds")
	}

	// Boundary extremes
	if !(GridPosition{X: -5, Y: -5}.IsWithinBounds()) {
		t.Errorf("(-5, -5) must be in bounds")
	}
	if !(GridPosition{X: 5, Y: 5}.IsWithinBounds()) {
		t.Errorf("(5, 5) must be in bounds")
	}
	if (GridPosition{X: -6, Y: 0}.IsWithinBounds()) {
		t.Errorf("(-6, 0) must be out of bounds")
	}
	if (GridPosition{X: 0, Y: 6}.IsWithinBounds()) {
		t.Errorf("(0, 6) must be out of bounds")
	}

	// Checkerboard parity
	if !(GridPosition{X: 0, Y: 0}.IsDarkTile()) {
		t.Errorf("(0, 0) should be dark tile")
	}
	if (GridPosition{X: 1, Y: 0}.IsDarkTile()) {
		t.Errorf("(1, 0) should be light tile")
	}
	if (GridPosition{X: 0, Y: 1}.IsDarkTile()) {
		t.Errorf("(0, 1) should be light tile")
	}
	if !(GridPosition{X: -1, Y: -1}.IsDarkTile()) {
		t.Errorf("(-1, -1) should be dark tile")
	}
	if !(GridPosition{X: -5, Y: -5}.IsDarkTile()) {
		t.Errorf("(-5, -5) should be dark tile")
	}
	if !(GridPosition{X: -5, Y: 5}.IsDarkTile()) {
		t.Errorf("(-5, 5) should be dark tile")
	}
	if !(GridPosition{X: 5, Y: 5}.IsDarkTile()) {
		t.Errorf("(5, 5) should be dark tile")
	}
	if (GridPosition{X: -5, Y: -4}.IsDarkTile()) {
		t.Errorf("(-5, -4) should be light tile")
	}

	if pos1.String() != "(0, -3)" {
		t.Errorf("pos1.String() = %q, want '(0, -3)'", pos1.String())
	}
}

func TestStatsAndTokens(t *testing.T) {
	stats := DefaultBaseStats()
	if stats.HP != 100 || stats.Attack != 10 || stats.Defense != 5 || stats.Speed != 10 {
		t.Errorf("unexpected DefaultBaseStats: %+v", stats)
	}

	clonedStats := stats.Clone()
	if clonedStats != stats {
		t.Errorf("cloned stats mismatch")
	}

	tokens := DefaultActionTokens()
	if tokens.Movement != 1 || tokens.Attack != 1 || tokens.Special != 0 {
		t.Errorf("unexpected DefaultActionTokens: %+v", tokens)
	}

	if !tokens.CanMove() {
		t.Errorf("expected CanMove == true")
	}
	if !tokens.CanAttack() {
		t.Errorf("expected CanAttack == true")
	}
	if tokens.CanUseSpecial() {
		t.Errorf("expected CanUseSpecial == false")
	}

	tokens.Movement = 0
	if tokens.CanMove() {
		t.Errorf("expected CanMove == false after consuming movement")
	}
	tokens.Special = 1
	if !tokens.CanUseSpecial() {
		t.Errorf("expected CanUseSpecial == true")
	}

	clonedTokens := tokens.Clone()
	if clonedTokens != tokens {
		t.Errorf("cloned tokens mismatch")
	}
}

func TestNewInitialGameState(t *testing.T) {
	state := NewInitialGameState()

	if state.ActivePlayer != PlayerOne {
		t.Fatalf("expected ActivePlayer PlayerOne, got %v", state.ActivePlayer)
	}
	if state.ActiveUnitID != nil {
		t.Fatalf("expected ActiveUnitID nil, got %v", state.ActiveUnitID)
	}
	if state.Phase != PhaseSelectUnit {
		t.Fatalf("expected PhaseSelectUnit, got %v", state.Phase)
	}
	if len(state.Units) != 2 {
		t.Fatalf("expected 2 units, got %d", len(state.Units))
	}

	u1 := state.GetUnit(1)
	if u1 == nil || u1.Owner != PlayerOne || !u1.Position.Equals(GridPosition{X: 0, Y: -3}) {
		t.Fatalf("unexpected Unit 1 properties: %+v", u1)
	}
	if u1.Types.Primary != ElementWater || u1.Types.HasSecondary() {
		t.Fatalf("unexpected Unit 1 types: %+v", u1.Types)
	}

	u2 := state.GetUnit(2)
	if u2 == nil || u2.Owner != PlayerTwo || !u2.Position.Equals(GridPosition{X: 0, Y: 3}) {
		t.Fatalf("unexpected Unit 2 properties: %+v", u2)
	}
	if u2.Types.Primary != ElementFire || u2.Types.HasSecondary() {
		t.Fatalf("unexpected Unit 2 types: %+v", u2.Types)
	}

	// Missing unit returns nil
	if state.GetUnit(99) != nil {
		t.Fatalf("expected nil for GetUnit(99)")
	}

	atP1 := state.GetUnitAt(GridPosition{X: 0, Y: -3})
	if atP1 == nil || atP1.ID != 1 {
		t.Fatalf("expected Unit 1 at (0, -3)")
	}

	atP2 := state.FindUnitAt(GridPosition{X: 0, Y: 3})
	if atP2 == nil || atP2.ID != 2 {
		t.Fatalf("expected Unit 2 at (0, 3)")
	}

	atEmpty := state.GetUnitAt(GridPosition{X: 0, Y: 0})
	if atEmpty != nil {
		t.Fatalf("expected nil at (0, 0)")
	}

	// By owner queries
	p1Units := state.GetUnitsByOwner(PlayerOne)
	if len(p1Units) != 1 || p1Units[0].ID != 1 {
		t.Fatalf("expected 1 unit for PlayerOne, got %v", p1Units)
	}
	p2Units := state.GetUnitsByOwner(PlayerTwo)
	if len(p2Units) != 1 || p2Units[0].ID != 2 {
		t.Fatalf("expected 1 unit for PlayerTwo, got %v", p2Units)
	}
	noneUnits := state.GetUnitsByOwner(PlayerNone)
	if len(noneUnits) != 0 {
		t.Fatalf("expected 0 units for PlayerNone, got %d", len(noneUnits))
	}

	if state.GetActiveUnit() != nil {
		t.Fatalf("expected GetActiveUnit() == nil initially")
	}

	if err := state.Validate(); err != nil {
		t.Fatalf("initial game state validation failed: %v", err)
	}
}

func TestGameState_ActiveUnitManagement(t *testing.T) {
	state := NewInitialGameState()
	unitID := UnitID(1)

	state.SetActiveUnit(&unitID)
	if state.ActiveUnitID == nil || *state.ActiveUnitID != 1 {
		t.Fatalf("expected ActiveUnitID to be 1")
	}
	active := state.GetActiveUnit()
	if active == nil || active.ID != 1 {
		t.Fatalf("expected GetActiveUnit() to return Unit 1")
	}

	state.ClearActiveUnit()
	if state.ActiveUnitID != nil {
		t.Fatalf("expected ActiveUnitID to be nil after ClearActiveUnit")
	}
	if state.GetActiveUnit() != nil {
		t.Fatalf("expected GetActiveUnit() to return nil")
	}

	state.SetActiveUnit(&unitID)
	state.SetActiveUnit(nil)
	if state.ActiveUnitID != nil {
		t.Fatalf("expected ActiveUnitID to be nil after SetActiveUnit(nil)")
	}
}

func TestGameState_DeepCopyImmutability(t *testing.T) {
	original := NewInitialGameState()
	clone := original.Clone()

	// Modify clone properties
	clone.ActivePlayer = PlayerTwo
	clone.Phase = PhaseChooseAction
	activeID := UnitID(1)
	clone.SetActiveUnit(&activeID)
	clone.Units[0].Position = GridPosition{X: 1, Y: 1}
	clone.Units[0].Stats.HP = 50
	clone.Units[0].Tokens.Movement = 0
	sec := ElementFlora
	clone.Units[0].Types.Secondary = &sec

	// Assert original remains completely unmodified
	if original.ActivePlayer != PlayerOne {
		t.Errorf("original ActivePlayer was mutated: got %v", original.ActivePlayer)
	}
	if original.Phase != PhaseSelectUnit {
		t.Errorf("original Phase was mutated: got %v", original.Phase)
	}
	if original.ActiveUnitID != nil {
		t.Errorf("original ActiveUnitID was mutated: got %v", original.ActiveUnitID)
	}
	if original.Units[0].Position.X != 0 || original.Units[0].Position.Y != -3 {
		t.Errorf("original unit position was mutated: got %v", original.Units[0].Position)
	}
	if original.Units[0].Stats.HP != 100 {
		t.Errorf("original unit HP was mutated: got %d", original.Units[0].Stats.HP)
	}
	if original.Units[0].Tokens.Movement != 1 {
		t.Errorf("original unit movement token was mutated: got %d", original.Units[0].Tokens.Movement)
	}
	if original.Units[0].Types.HasSecondary() {
		t.Errorf("original unit types was mutated: got %+v", original.Units[0].Types)
	}

	// Test nil clone edge cases
	var nilState *GameState
	if nilState.Clone() != nil {
		t.Errorf("nilState.Clone() should be nil")
	}
	var nilUnit *Unit
	if nilUnit.Clone() != nil {
		t.Errorf("nilUnit.Clone() should be nil")
	}
}

func TestGameState_Validate(t *testing.T) {
	// Baseline valid
	state := NewInitialGameState()
	if err := state.Validate(); err != nil {
		t.Fatalf("expected baseline state valid, got: %v", err)
	}

	// 1. Invalid ActivePlayer
	invalidPlayerState := state.Clone()
	invalidPlayerState.ActivePlayer = PlayerNone
	if err := invalidPlayerState.Validate(); err != ErrInvalidActivePlayer {
		t.Errorf("expected ErrInvalidActivePlayer, got %v", err)
	}

	// 2. Invalid Phase
	invalidPhaseState := state.Clone()
	invalidPhaseState.Phase = TurnPhase("BogusPhase")
	if err := invalidPhaseState.Validate(); err != ErrInvalidPhase {
		t.Errorf("expected ErrInvalidPhase, got %v", err)
	}

	// 3. SelectUnit with active unit forbidden
	selectUnitWithActive := state.Clone()
	id1 := UnitID(1)
	selectUnitWithActive.SetActiveUnit(&id1)
	if err := selectUnitWithActive.Validate(); err != ErrActiveUnitForbidden {
		t.Errorf("expected ErrActiveUnitForbidden, got %v", err)
	}

	// 4. ChooseAction with nil active unit required
	chooseActionNoActive := state.Clone()
	chooseActionNoActive.Phase = PhaseChooseAction
	chooseActionNoActive.ActiveUnitID = nil
	if err := chooseActionNoActive.Validate(); err != ErrActiveUnitRequired {
		t.Errorf("expected ErrActiveUnitRequired, got %v", err)
	}

	// 5. ChooseAction with non-existent unit
	chooseActionMissingUnit := state.Clone()
	chooseActionMissingUnit.Phase = PhaseChooseAction
	missingID := UnitID(99)
	chooseActionMissingUnit.SetActiveUnit(&missingID)
	if err := chooseActionMissingUnit.Validate(); err != ErrUnitNotFound {
		t.Errorf("expected ErrUnitNotFound, got %v", err)
	}

	// 6. ChooseAction with opponent's unit
	chooseActionOpponentUnit := state.Clone()
	chooseActionOpponentUnit.Phase = PhaseChooseAction
	p2UnitID := UnitID(2) // Belongs to PlayerTwo, but ActivePlayer is PlayerOne
	chooseActionOpponentUnit.SetActiveUnit(&p2UnitID)
	if err := chooseActionOpponentUnit.Validate(); err != ErrUnitOwnerMismatch {
		t.Errorf("expected ErrUnitOwnerMismatch, got %v", err)
	}

	// Valid ChooseAction
	validChooseAction := state.Clone()
	validChooseAction.Phase = PhaseChooseAction
	p1UnitID := UnitID(1)
	validChooseAction.SetActiveUnit(&p1UnitID)
	if err := validChooseAction.Validate(); err != nil {
		t.Errorf("expected valid choose action, got %v", err)
	}

	// 7. Unit out of bounds
	outOfBoundsState := state.Clone()
	outOfBoundsState.Units[0].Position = GridPosition{X: 6, Y: 0}
	if err := outOfBoundsState.Validate(); err != ErrUnitOutOfBounds {
		t.Errorf("expected ErrUnitOutOfBounds, got %v", err)
	}

	// 8. Overlapping units
	overlappingState := state.Clone()
	overlappingState.Units[1].Position = overlappingState.Units[0].Position
	if err := overlappingState.Validate(); err != ErrOverlappingUnits {
		t.Errorf("expected ErrOverlappingUnits, got %v", err)
	}

	// 9. Duplicate unit IDs
	duplicateIDState := state.Clone()
	duplicateIDState.Units[1].ID = duplicateIDState.Units[0].ID
	if err := duplicateIDState.Validate(); err != ErrDuplicateUnitID {
		t.Errorf("expected ErrDuplicateUnitID, got %v", err)
	}
}

func TestGameState_JSONRoundtrip(t *testing.T) {
	state := NewInitialGameState()
	activeID := UnitID(1)
	state.SetActiveUnit(&activeID)
	state.Phase = PhaseChooseAction

	bytes, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("json marshal error: %v", err)
	}

	var restored GameState
	if err := json.Unmarshal(bytes, &restored); err != nil {
		t.Fatalf("json unmarshal error: %v", err)
	}

	if restored.ActivePlayer != state.ActivePlayer {
		t.Errorf("restored ActivePlayer mismatch: got %v, want %v", restored.ActivePlayer, state.ActivePlayer)
	}
	if restored.Phase != state.Phase {
		t.Errorf("restored Phase mismatch: got %v, want %v", restored.Phase, state.Phase)
	}
	if restored.ActiveUnitID == nil || *restored.ActiveUnitID != 1 {
		t.Errorf("restored ActiveUnitID mismatch: got %v", restored.ActiveUnitID)
	}
	if len(restored.Units) != 2 {
		t.Fatalf("restored Units length mismatch: got %d", len(restored.Units))
	}
	if restored.Units[0].Position != state.Units[0].Position {
		t.Errorf("restored Unit 0 position mismatch: got %v, want %v", restored.Units[0].Position, state.Units[0].Position)
	}
	if restored.Units[1].Position != state.Units[1].Position {
		t.Errorf("restored Unit 1 position mismatch: got %v, want %v", restored.Units[1].Position, state.Units[1].Position)
	}

	if err := restored.Validate(); err != nil {
		t.Fatalf("restored state validation failed: %v", err)
	}
}
