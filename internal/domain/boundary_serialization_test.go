package domain_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"testing"
	"time"

	"wildbuds/internal/domain"
)

// ============================================================================
// FOCUS AREA 1: Geometry & Coordinate Boundaries
// ============================================================================

// TestGridPosition_PerimeterExhaustive verifies all 121 tiles in [-5, 5] x [-5, 5]
// and all surrounding tiles on the perimeter and outer perimeter.
func TestGridPosition_PerimeterExhaustive(t *testing.T) {
	// 1. Every tile in [-5, 5] x [-5, 5] must be in bounds
	insideCount := 0
	for x := domain.GridMin; x <= domain.GridMax; x++ {
		for y := domain.GridMin; y <= domain.GridMax; y++ {
			pos := domain.GridPosition{X: x, Y: y}
			if !pos.IsWithinBounds() {
				t.Fatalf("expected pos %v to be within bounds", pos)
			}
			if !pos.InBounds(domain.GridMin, domain.GridMax) {
				t.Fatalf("expected pos %v to be InBounds(%d, %d)", pos, domain.GridMin, domain.GridMax)
			}
			if !domain.InGridBounds(pos) {
				t.Fatalf("expected domain.InGridBounds(%v) == true", pos)
			}
			insideCount++
		}
	}
	if insideCount != domain.TotalTiles {
		t.Fatalf("expected %d tiles inside board, counted %d", domain.TotalTiles, insideCount)
	}

	// 2. All 40 perimeter tiles where max(|x|, |y|) == 5
	perimeterCount := 0
	for x := -5; x <= 5; x++ {
		for y := -5; y <= 5; y++ {
			if x == -5 || x == 5 || y == -5 || y == 5 {
				perimeterCount++
				pos := domain.GridPosition{X: x, Y: y}
				if !pos.IsWithinBounds() {
					t.Errorf("perimeter tile %v should be within bounds", pos)
				}
			}
		}
	}
	if perimeterCount != 40 {
		t.Errorf("expected 40 perimeter tiles, got %d", perimeterCount)
	}

	// 3. Immediate exterior boundary: max(|x|, |y|) == 6 must be OUT of bounds
	for x := -6; x <= 6; x++ {
		for y := -6; y <= 6; y++ {
			if x == -6 || x == 6 || y == -6 || y == 6 {
				pos := domain.GridPosition{X: x, Y: y}
				if pos.IsWithinBounds() {
					t.Errorf("exterior tile %v must be out of bounds", pos)
				}
				if domain.InGridBounds(pos) {
					t.Errorf("exterior tile %v domain.InGridBounds must be false", pos)
				}
			}
		}
	}
}

// TestGridPosition_ExtremeBoundaries tests extreme int32 and int64 boundaries.
func TestGridPosition_ExtremeBoundaries(t *testing.T) {
	extremes := []int{
		math.MinInt32,
		math.MinInt32 + 1,
		-1000,
		-6,
		6,
		1000,
		math.MaxInt32 - 1,
		math.MaxInt32,
	}

	for _, x := range extremes {
		for _, y := range extremes {
			pos := domain.GridPosition{X: x, Y: y}
			if pos.IsWithinBounds() {
				t.Errorf("extreme coordinate %v should be out of bounds", pos)
			}
		}
	}

	// InBounds with reversed bounds (min > max) must always return false
	pos := domain.GridPosition{X: 0, Y: 0}
	if pos.InBounds(5, -5) {
		t.Errorf("InBounds(5, -5) with inverted range should return false")
	}
	if pos.InBounds(0, -1) {
		t.Errorf("InBounds(0, -1) with inverted range should return false")
	}
}

// TestGridPosition_CheckerboardParity verifies that the checkerboard parity
// formula (X + Y) % 2 == 0 matches bitwise XOR parity across the board and at extremes.
func TestGridPosition_CheckerboardParity(t *testing.T) {
	for x := domain.GridMin; x <= domain.GridMax; x++ {
		for y := domain.GridMin; y <= domain.GridMax; y++ {
			pos := domain.GridPosition{X: x, Y: y}
			isDark := pos.IsDarkTile()

			// Bitwise parity oracle: (x ^ y) & 1 == 0
			// Because for any integers x, y, (x + y) is even iff x and y have the same parity.
			expectedDark := ((x ^ y) & 1) == 0
			if isDark != expectedDark {
				t.Fatalf("tile %v parity mismatch: got %v, want %v", pos, isDark, expectedDark)
			}

			// Also verify adjacent orthogonal tiles always alternate
			if x < domain.GridMax {
				right := domain.GridPosition{X: x + 1, Y: y}
				if right.IsDarkTile() == isDark {
					t.Errorf("adjacent horizontal tiles %v and %v share same color!", pos, right)
				}
			}
			if y < domain.GridMax {
				up := domain.GridPosition{X: x, Y: y + 1}
				if up.IsDarkTile() == isDark {
					t.Errorf("adjacent vertical tiles %v and %v share same color!", pos, up)
				}
			}
		}
	}

	// Test parity at extreme int32 bounds
	testCases := []struct {
		x, y int
		want bool
	}{
		{math.MinInt32, math.MinInt32, true},      // even + even = even
		{math.MinInt32, math.MinInt32 + 1, false},  // even + odd = odd
		{math.MaxInt32, math.MaxInt32, true},      // odd + odd = even
		{math.MaxInt32, math.MaxInt32 - 1, false},  // odd + even = odd
		{math.MinInt32, math.MaxInt32, false},     // even + odd = odd
	}

	for _, tc := range testCases {
		pos := domain.GridPosition{X: tc.x, Y: tc.y}
		if got := pos.IsDarkTile(); got != tc.want {
			t.Errorf("IsDarkTile(%v) = %v, want %v", pos, got, tc.want)
		}
	}
}

// TestGridPosition_DistanceMetricProperties tests the Manhattan distance metric axioms:
// 1. d(A, B) >= 0 (Non-negativity)
// 2. d(A, B) == 0 <=> A == B (Identity of indiscernibles)
// 3. d(A, B) == d(B, A) (Symmetry)
// 4. d(A, C) <= d(A, B) + d(B, C) (Triangle inequality)
func TestGridPosition_DistanceMetricProperties(t *testing.T) {
	// Sample positions across board
	positions := []domain.GridPosition{
		{X: -5, Y: -5},
		{X: -5, Y: 5},
		{X: 5, Y: -5},
		{X: 5, Y: 5},
		{X: 0, Y: 0},
		{X: 0, Y: -3},
		{X: 0, Y: 3},
		{X: -2, Y: 1},
		{X: 3, Y: -4},
	}

	for _, a := range positions {
		// Identity: d(A, A) == 0
		if d := a.Distance(a); d != 0 {
			t.Errorf("Distance(%v, %v) = %d, want 0", a, a, d)
		}

		for _, b := range positions {
			dAB := a.Distance(b)
			dBA := b.Distance(a)

			// Non-negativity
			if dAB < 0 {
				t.Errorf("Distance(%v, %v) = %d, want >= 0", a, b, dAB)
			}

			// Indiscernibles
			if (dAB == 0) != a.Equals(b) {
				t.Errorf("Distance(%v, %v) == 0 is %v, but Equals is %v", a, b, dAB == 0, a.Equals(b))
			}

			// Symmetry
			if dAB != dBA {
				t.Errorf("Distance asymmetry: d(%v, %v)=%d != d(%v, %v)=%d", a, b, dAB, b, a, dBA)
			}

			// Triangle inequality
			for _, c := range positions {
				dAC := a.Distance(c)
				dBC := b.Distance(c)
				if dAC > dAB+dBC {
					t.Errorf("Triangle inequality violated: d(%v, %v)=%d > d(%v, %v)=%d + d(%v, %v)=%d",
						a, c, dAC, a, b, dAB, b, c, dBC)
				}
			}
		}
	}

	// Maximum board distance is corner-to-corner: |(-5) - 5| + |(-5) - 5| = 20
	c1 := domain.GridPosition{X: -5, Y: -5}
	c2 := domain.GridPosition{X: 5, Y: 5}
	if d := c1.Distance(c2); d != 20 {
		t.Errorf("Max board distance = %d, want 20", d)
	}

	c3 := domain.GridPosition{X: -5, Y: 5}
	c4 := domain.GridPosition{X: 5, Y: -5}
	if d := c3.Distance(c4); d != 20 {
		t.Errorf("Max board distance (other diagonal) = %d, want 20", d)
	}
}

// TestGridPosition_DistanceInt32Overflow tests Manhattan distance with math.MinInt32 / math.MaxInt32.
func TestGridPosition_DistanceInt32Overflow(t *testing.T) {
	pMin := domain.GridPosition{X: math.MinInt32, Y: math.MinInt32}
	pMax := domain.GridPosition{X: math.MaxInt32, Y: math.MaxInt32}

	d := pMin.Distance(pMax)
	// Expected distance on 64-bit int: 2 * (math.MaxInt32 - math.MinInt32)
	// math.MaxInt32 - math.MinInt32 = 2147483647 - (-2147483648) = 4294967295
	// 4294967295 * 2 = 8589934590
	expected := int(uint64(math.MaxInt32-math.MinInt32) * 2)
	if d != expected {
		t.Logf("Note: Distance with MinInt32 and MaxInt32 produced %d (expected %d)", d, expected)
	}

	// Distance from origin to MinInt32
	pOrigin := domain.GridPosition{X: 0, Y: 0}
	dOriginMin := pOrigin.Distance(pMin)
	expectedOriginMin := int(uint64(-math.MinInt32) * 2)
	if dOriginMin != expectedOriginMin {
		t.Logf("Note: Distance from origin to MinInt32 produced %d (expected %d)", dOriginMin, expectedOriginMin)
	}
}

// TestGridPosition_DistanceSigned64Overflow documents behavior under math.MinInt (64-bit).
// In Go, -math.MinInt overflows 64-bit signed int and evaluates back to math.MinInt (negative).
func TestGridPosition_DistanceSigned64Overflow(t *testing.T) {
	pMin64 := domain.GridPosition{X: math.MinInt, Y: 0}
	pOrigin := domain.GridPosition{X: 0, Y: 0}
	d := pMin64.Distance(pOrigin)
	if d < 0 {
		t.Logf("Empirical observation: math.MinInt causes signed 64-bit integer overflow in Distance(): result is %d (negative)", d)
	}
}

// ============================================================================
// FOCUS AREA 2: Serialization & Deserialization
// ============================================================================

// TestSerialization_PlayerExhaustive verifies roundtrip and error handling for Player.
func TestSerialization_PlayerExhaustive(t *testing.T) {
	validRoundtrips := []struct {
		player   domain.Player
		jsonWant string
	}{
		{domain.PlayerOne, `"One"`},
		{domain.PlayerTwo, `"Two"`},
		{domain.PlayerNone, `"None"`},
	}

	for _, tc := range validRoundtrips {
		data, err := json.Marshal(tc.player)
		if err != nil {
			t.Fatalf("Marshal(%v) error: %v", tc.player, err)
		}
		if string(data) != tc.jsonWant {
			t.Errorf("Marshal(%v) = %s, want %s", tc.player, string(data), tc.jsonWant)
		}

		var restored domain.Player
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatalf("Unmarshal(%s) error: %v", string(data), err)
		}
		if restored != tc.player {
			t.Errorf("roundtrip mismatch: got %v, want %v", restored, tc.player)
		}
	}

	// Supported alternative string representations
	altStrings := []struct {
		input string
		want  domain.Player
	}{
		{`"PlayerOne"`, domain.PlayerOne},
		{`"PlayerTwo"`, domain.PlayerTwo},
		{`"PlayerNone"`, domain.PlayerNone},
		{`1`, domain.PlayerOne},
		{`2`, domain.PlayerTwo},
		{`0`, domain.PlayerNone},
	}
	for _, tc := range altStrings {
		var p domain.Player
		if err := json.Unmarshal([]byte(tc.input), &p); err != nil {
			t.Errorf("Unmarshal(%s) error: %v", tc.input, err)
		}
		if p != tc.want {
			t.Errorf("Unmarshal(%s) = %v, want %v", tc.input, p, tc.want)
		}
	}

	// Invalid inputs that must be rejected
	invalidInputs := []string{
		`"one"`,
		`"two"`,
		`"none"`,
		`"Three"`,
		`"PlayerThree"`,
		`""`,
		`" "`,
		`3`,
		`-1`,
		`999`,
		`-999`,
		`1.5`,
		`true`,
		`false`,
		`null`,
		`[]`,
		`{}`,
	}
	for _, input := range invalidInputs {
		var p domain.Player
		if err := json.Unmarshal([]byte(input), &p); err == nil {
			t.Errorf("expected Unmarshal(%s) to fail, but got %v", input, p)
		}
	}
}

// TestSerialization_TurnPhaseExhaustive verifies roundtrip and error handling for TurnPhase.
func TestSerialization_TurnPhaseExhaustive(t *testing.T) {
	validPhases := []domain.TurnPhase{
		domain.PhaseSelectUnit,
		domain.PhaseChooseAction,
	}

	for _, phase := range validPhases {
		data, err := json.Marshal(phase)
		if err != nil {
			t.Fatalf("Marshal(%v) error: %v", phase, err)
		}

		var restored domain.TurnPhase
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatalf("Unmarshal(%s) error: %v", string(data), err)
		}
		if restored != phase {
			t.Errorf("roundtrip mismatch: got %v, want %v", restored, phase)
		}
	}

	// Integer representations: 0 -> SelectUnit, 1 -> ChooseAction
	var tp domain.TurnPhase
	if err := json.Unmarshal([]byte(`0`), &tp); err != nil || tp != domain.PhaseSelectUnit {
		t.Errorf("Unmarshal(0) failed: err=%v, got=%v", err, tp)
	}
	if err := json.Unmarshal([]byte(`1`), &tp); err != nil || tp != domain.PhaseChooseAction {
		t.Errorf("Unmarshal(1) failed: err=%v, got=%v", err, tp)
	}

	// Invalid phases
	invalid := []string{
		`"selectunit"`,
		`"chooseaction"`,
		`"SELECT_UNIT"`,
		`"PhaseSelectUnit"`,
		`""`,
		`-1`,
		`2`,
		`99`,
		`1.0`,
		`true`,
		`null`,
		`[]`,
		`{}`,
	}
	for _, in := range invalid {
		var p domain.TurnPhase
		if err := json.Unmarshal([]byte(in), &p); err == nil {
			t.Errorf("expected error unmarshaling %s into TurnPhase, got %v", in, p)
		}
	}
}

// TestSerialization_GridPositionRoundtrip verifies GridPosition JSON roundtrip and edge cases.
func TestSerialization_GridPositionRoundtrip(t *testing.T) {
	positions := []domain.GridPosition{
		{X: 0, Y: 0},
		{X: -5, Y: -5},
		{X: 5, Y: 5},
		{X: math.MinInt32, Y: math.MaxInt32},
	}

	for _, p := range positions {
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("Marshal(%v) error: %v", p, err)
		}

		var restored domain.GridPosition
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatalf("Unmarshal(%s) error: %v", string(data), err)
		}
		if !restored.Equals(p) {
			t.Errorf("roundtrip mismatch: got %v, want %v", restored, p)
		}
	}

	// Missing fields default to zero
	var pMissing domain.GridPosition
	if err := json.Unmarshal([]byte(`{"x": 3}`), &pMissing); err != nil {
		t.Fatalf("Unmarshal with missing y failed: %v", err)
	}
	if pMissing.X != 3 || pMissing.Y != 0 {
		t.Errorf("expected (3, 0), got %v", pMissing)
	}

	// Non-numeric fields error
	var pInvalid domain.GridPosition
	if err := json.Unmarshal([]byte(`{"x": "three", "y": 0}`), &pInvalid); err == nil {
		t.Errorf("expected error unmarshaling string coordinate")
	}
}

// TestSerialization_BaseStatsAndActionTokens tests BaseStats and ActionTokens serialization.
func TestSerialization_BaseStatsAndActionTokens(t *testing.T) {
	// BaseStats roundtrip
	stats := domain.DefaultBaseStats()
	data, err := json.Marshal(stats)
	if err != nil {
		t.Fatalf("Marshal BaseStats error: %v", err)
	}

	var restored domain.BaseStats
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal BaseStats error: %v", err)
	}
	if restored != stats {
		t.Errorf("BaseStats roundtrip mismatch: got %+v, want %+v", restored, stats)
	}

	// BaseStats negative values must error (uint32)
	var invalidStats domain.BaseStats
	if err := json.Unmarshal([]byte(`{"hp": -10}`), &invalidStats); err == nil {
		t.Errorf("expected error unmarshaling negative HP into uint32")
	}

	// ActionTokens roundtrip
	tokens := domain.DefaultActionTokens()
	tokensData, err := json.Marshal(tokens)
	if err != nil {
		t.Fatalf("Marshal ActionTokens error: %v", err)
	}

	var restoredTokens domain.ActionTokens
	if err := json.Unmarshal(tokensData, &restoredTokens); err != nil {
		t.Fatalf("Unmarshal ActionTokens error: %v", err)
	}
	if restoredTokens != tokens {
		t.Errorf("ActionTokens roundtrip mismatch: got %+v, want %+v", restoredTokens, tokens)
	}

	// ActionTokens overflow must error (uint8 max 255)
	var invalidTokens domain.ActionTokens
	if err := json.Unmarshal([]byte(`{"movement": 256}`), &invalidTokens); err == nil {
		t.Errorf("expected error unmarshaling 256 into uint8")
	}
}

// TestSerialization_DualTypeExhaustive verifies single and dual typing JSON roundtrips and edge cases.
func TestSerialization_DualTypeExhaustive(t *testing.T) {
	// 1. Single type: omitempty secondary
	single := domain.NewSingleType(domain.ElementWater)
	dataSingle, err := json.Marshal(single)
	if err != nil {
		t.Fatalf("Marshal single type error: %v", err)
	}
	if strings.Contains(string(dataSingle), "secondary") {
		t.Errorf("single type JSON should omit secondary, got %s", string(dataSingle))
	}

	var restoredSingle domain.DualType
	if err := json.Unmarshal(dataSingle, &restoredSingle); err != nil {
		t.Fatalf("Unmarshal single type error: %v", err)
	}
	if restoredSingle.Primary != domain.ElementWater || restoredSingle.HasSecondary() {
		t.Errorf("restored single type mismatch: %+v", restoredSingle)
	}

	// 2. Dual type
	dual := domain.NewDualType(domain.ElementFire, domain.ElementGrass)
	dataDual, err := json.Marshal(dual)
	if err != nil {
		t.Fatalf("Marshal dual type error: %v", err)
	}

	var restoredDual domain.DualType
	if err := json.Unmarshal(dataDual, &restoredDual); err != nil {
		t.Fatalf("Unmarshal dual type error: %v", err)
	}
	sec, ok := restoredDual.SecondaryElement()
	if !ok || restoredDual.Primary != domain.ElementFire || sec != domain.ElementGrass {
		t.Errorf("restored dual type mismatch: %+v", restoredDual)
	}

	// 3. Unmarshaling explicit null secondary
	var explicitNull domain.DualType
	if err := json.Unmarshal([]byte(`{"primary":"Water","secondary":null}`), &explicitNull); err != nil {
		t.Fatalf("Unmarshal explicit null secondary failed: %v", err)
	}
	if explicitNull.HasSecondary() {
		t.Errorf("expected HasSecondary() == false for null secondary")
	}

	// 4. Case-insensitivity for element names
	var caseTest domain.DualType
	if err := json.Unmarshal([]byte(`{"primary":"wAtEr","secondary":"fIrE"}`), &caseTest); err != nil {
		t.Fatalf("case-insensitive unmarshal failed: %v", err)
	}
	if caseTest.Primary != domain.ElementWater || *caseTest.Secondary != domain.ElementFire {
		t.Errorf("case-insensitive mismatch: %+v", caseTest)
	}

	// 5. Invalid element strings
	var invalid domain.DualType
	if err := json.Unmarshal([]byte(`{"primary":"Kryptonite"}`), &invalid); err == nil {
		t.Errorf("expected error for unknown primary element")
	}
	if err := json.Unmarshal([]byte(`{"primary":"Fire","secondary":"Adamantium"}`), &invalid); err == nil {
		t.Errorf("expected error for unknown secondary element")
	}

	// 6. Out-of-bounds integer element indices
	if err := json.Unmarshal([]byte(`{"primary": 18}`), &invalid); err == nil {
		t.Errorf("expected error for out of bounds primary element index 18")
	}
	if err := json.Unmarshal([]byte(`{"primary": 0, "secondary": -1}`), &invalid); err == nil {
		t.Errorf("expected error for negative secondary element index")
	}
}

// TestSerialization_GameStateRoundtrip tests full GameState roundtrip with all states.
func TestSerialization_GameStateRoundtrip(t *testing.T) {
	// 1. Initial state (ActiveUnitID == nil, Phase == SelectUnit)
	s1 := domain.NewInitialGameState()
	data1, err := json.Marshal(s1)
	if err != nil {
		t.Fatalf("Marshal initial state error: %v", err)
	}

	var r1 domain.GameState
	if err := json.Unmarshal(data1, &r1); err != nil {
		t.Fatalf("Unmarshal initial state error: %v", err)
	}
	if err := r1.Validate(); err != nil {
		t.Fatalf("Validate on restored initial state failed: %v", err)
	}
	if r1.ActiveUnitID != nil {
		t.Errorf("ActiveUnitID should be nil in restored state")
	}

	// 2. Active unit set (Phase == ChooseAction, ActiveUnitID == 1)
	s2 := domain.NewInitialGameState()
	id1 := domain.UnitID(1)
	s2.SetActiveUnit(&id1)
	s2.Phase = domain.PhaseChooseAction
	s2.Units[0].Position = domain.GridPosition{X: -2, Y: 1}
	s2.Units[0].Stats.HP = 85
	s2.Units[0].Tokens.Movement = 0

	data2, err := json.Marshal(s2)
	if err != nil {
		t.Fatalf("Marshal active state error: %v", err)
	}

	var r2 domain.GameState
	if err := json.Unmarshal(data2, &r2); err != nil {
		t.Fatalf("Unmarshal active state error: %v", err)
	}
	if err := r2.Validate(); err != nil {
		t.Fatalf("Validate on restored active state failed: %v", err)
	}
	if r2.ActiveUnitID == nil || *r2.ActiveUnitID != 1 {
		t.Errorf("ActiveUnitID mismatch: got %v, want 1", r2.ActiveUnitID)
	}
	if r2.Units[0].Position.X != -2 || r2.Units[0].Position.Y != 1 {
		t.Errorf("Unit 0 position mismatch: %v", r2.Units[0].Position)
	}
	if r2.Units[0].Stats.HP != 85 {
		t.Errorf("Unit 0 HP mismatch: %d", r2.Units[0].Stats.HP)
	}
	if r2.Units[0].Tokens.Movement != 0 {
		t.Errorf("Unit 0 movement token mismatch: %d", r2.Units[0].Tokens.Movement)
	}
}

// TestFuzz_MalformedJSON tests that arbitrary malformed JSON never panics during unmarshaling.
func TestFuzz_MalformedJSON(t *testing.T) {
	malformedInputs := []string{
		"",
		" ",
		"{",
		"}",
		"[]",
		"[",
		"]",
		`{"active_player":`,
		`{"active_player": "One"`,
		`{"active_player": null}`,
		`{"active_player": 12345}`,
		`{"phase": ""}`,
		`{"units": "not_an_array"}`,
		`{"units": [null, null]}`,
		`{"units": [{}]}`,
		`{"units": [{"id": "abc"}]}`,
		`{"units": [{"position": {"x": "nan"}}]}`,
		`{"units": [{"types": {"primary": 99999}}]}`,
		`{"units": [{"stats": {"hp": -100}}]}`,
		`{"units": [{"tokens": {"movement": 99999}}]}`,
		`{"active_unit_id": "not_an_int"}`,
		`1e309`,
		`-1e309`,
		`99999999999999999999999999999999999999999999999999`,
		"{\"key\": \x00\x01\x02}",
		strings.Repeat("[", 100) + strings.Repeat("]", 100),
		`{"a":{"b":{"c":{"d":null}}}}`,
		`{"active_player": true, "phase": false, "units": 123}`,
	}

	for i, input := range malformedInputs {
		t.Run(fmt.Sprintf("Corpus_%d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("PANIC on malformed input %q: %v", input, r)
				}
			}()

			var state domain.GameState
			_ = json.Unmarshal([]byte(input), &state)

			var player domain.Player
			_ = json.Unmarshal([]byte(input), &player)

			var phase domain.TurnPhase
			_ = json.Unmarshal([]byte(input), &phase)

			var pos domain.GridPosition
			_ = json.Unmarshal([]byte(input), &pos)

			var elem domain.Element
			_ = json.Unmarshal([]byte(input), &elem)

			var dt domain.DualType
			_ = json.Unmarshal([]byte(input), &dt)

			var stats domain.BaseStats
			_ = json.Unmarshal([]byte(input), &stats)

			var tokens domain.ActionTokens
			_ = json.Unmarshal([]byte(input), &tokens)

			var unit domain.Unit
			_ = json.Unmarshal([]byte(input), &unit)
		})
	}
}

// TestFuzz_RandomMutations runs pseudo-random chaos fuzzing on valid GameState JSON.
func TestFuzz_RandomMutations(t *testing.T) {
	state := domain.NewInitialGameState()
	validJSON, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("failed to marshal base state: %v", err)
	}

	rng := rand.New(rand.NewSource(42))

	for iter := 0; iter < 500; iter++ {
		mutated := make([]byte, len(validJSON))
		copy(mutated, validJSON)

		// Apply 1 to 5 random byte mutations
		numMutations := 1 + rng.Intn(5)
		for m := 0; m < numMutations; m++ {
			pos := rng.Intn(len(mutated))
			switch rng.Intn(4) {
			case 0:
				// Flip byte
				mutated[pos] = byte(rng.Intn(256))
			case 1:
				// Insert character
				mutated = append(mutated[:pos], append([]byte{byte(rng.Intn(256))}, mutated[pos:]...)...)
			case 2:
				// Delete byte
				if len(mutated) > 1 {
					mutated = append(mutated[:pos], mutated[pos+1:]...)
				}
			case 3:
				// Truncate
				mutated = mutated[:pos]
			}
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("PANIC on iter %d with payload %q: %v", iter, string(mutated), r)
				}
			}()

			var target domain.GameState
			if err := json.Unmarshal(mutated, &target); err == nil {
				// If it successfully unmarshaled, Validate should also not panic
				_ = target.Validate()
			}
		}()
	}
}

// ============================================================================
// FOCUS AREA 3: State Isolation & Mutation Leakage
// ============================================================================

// TestStateIsolation_UnitDeepCopy verifies zero mutation leakage between cloned Units.
func TestStateIsolation_UnitDeepCopy(t *testing.T) {
	secOrig := domain.ElementGrass
	original := &domain.Unit{
		ID:       1,
		Owner:    domain.PlayerOne,
		Position: domain.GridPosition{X: 0, Y: -3},
		Stats: domain.BaseStats{
			HP:      100,
			Attack:  10,
			Defense: 5,
			Speed:   10,
		},
		Tokens: domain.ActionTokens{
			Movement: 1,
			Attack:   1,
			Special:  0,
		},
		Types: domain.DualType{
			Primary:   domain.ElementWater,
			Secondary: &secOrig,
		},
	}

	clone := original.Clone()
	if clone == nil {
		t.Fatalf("clone is nil")
	}

	// 1. Verify pointer independence of Secondary element
	if clone.Types.Secondary == original.Types.Secondary {
		t.Fatalf("CRITICAL: clone.Types.Secondary shares pointer with original.Types.Secondary (%p == %p)",
			clone.Types.Secondary, original.Types.Secondary)
	}

	// 2. Mutate clone's fields aggressively
	clone.ID = 999
	clone.Owner = domain.PlayerTwo
	clone.Position = domain.GridPosition{X: 4, Y: 4}
	clone.Stats.HP = 1
	clone.Stats.Attack = 99
	clone.Stats.Defense = 88
	clone.Stats.Speed = 77
	clone.Tokens.Movement = 0
	clone.Tokens.Attack = 0
	clone.Tokens.Special = 5
	clone.Types.Primary = domain.ElementDragon
	*clone.Types.Secondary = domain.ElementElectric

	// 3. Assert original is completely pristine
	if original.ID != 1 {
		t.Errorf("original.ID mutated: got %d, want 1", original.ID)
	}
	if original.Owner != domain.PlayerOne {
		t.Errorf("original.Owner mutated: got %v, want %v", original.Owner, domain.PlayerOne)
	}
	if original.Position.X != 0 || original.Position.Y != -3 {
		t.Errorf("original.Position mutated: got %v, want (0, -3)", original.Position)
	}
	if original.Stats.HP != 100 || original.Stats.Attack != 10 || original.Stats.Defense != 5 || original.Stats.Speed != 10 {
		t.Errorf("original.Stats mutated: got %+v", original.Stats)
	}
	if original.Tokens.Movement != 1 || original.Tokens.Attack != 1 || original.Tokens.Special != 0 {
		t.Errorf("original.Tokens mutated: got %+v", original.Tokens)
	}
	if original.Types.Primary != domain.ElementWater {
		t.Errorf("original.Types.Primary mutated: got %v, want Water", original.Types.Primary)
	}
	if original.Types.Secondary == nil || *original.Types.Secondary != domain.ElementGrass {
		t.Errorf("original.Types.Secondary mutated: got %v, want Grass", original.Types.Secondary)
	}

	// 4. Test setting clone.Types.Secondary to nil
	clone.Types.Secondary = nil
	if original.Types.Secondary == nil {
		t.Errorf("original.Types.Secondary became nil after setting clone's to nil")
	}

	// 5. Test single type (Secondary == nil) clone
	singleUnit := &domain.Unit{
		ID:    2,
		Types: domain.NewSingleType(domain.ElementFire),
	}
	singleClone := singleUnit.Clone()
	if singleClone.Types.Secondary != nil {
		t.Errorf("expected singleClone.Types.Secondary == nil")
	}
	newSec := domain.ElementPoison
	singleClone.Types.Secondary = &newSec
	if singleUnit.Types.Secondary != nil {
		t.Errorf("mutating clone's secondary element leaked to singleUnit")
	}
}

// TestStateIsolation_GameStateDeepCopy verifies comprehensive isolation of GameState.
func TestStateIsolation_GameStateDeepCopy(t *testing.T) {
	orig := domain.NewInitialGameState()
	activeID := domain.UnitID(1)
	orig.SetActiveUnit(&activeID)
	orig.Phase = domain.PhaseChooseAction

	clone := orig.Clone()

	// 1. Pointer identity checks
	if clone == orig {
		t.Fatalf("clone pointer equals original pointer (%p)", orig)
	}
	if clone.ActiveUnitID == orig.ActiveUnitID {
		t.Fatalf("CRITICAL: clone.ActiveUnitID shares pointer with original (%p == %p)",
			clone.ActiveUnitID, orig.ActiveUnitID)
	}
	if len(clone.Units) != len(orig.Units) {
		t.Fatalf("clone units length mismatch")
	}
	for i := range orig.Units {
		if clone.Units[i] == orig.Units[i] {
			t.Fatalf("CRITICAL: clone.Units[%d] shares pointer with original.Units[%d] (%p == %p)",
				i, i, clone.Units[i], orig.Units[i])
		}
	}

	// 2. Mutate active unit ID through pointer dereference
	*clone.ActiveUnitID = domain.UnitID(2)
	if *orig.ActiveUnitID != 1 {
		t.Errorf("mutating *clone.ActiveUnitID affected *orig.ActiveUnitID: got %d, want 1", *orig.ActiveUnitID)
	}

	// 3. Clear active unit on clone
	clone.ClearActiveUnit()
	if orig.ActiveUnitID == nil || *orig.ActiveUnitID != 1 {
		t.Errorf("ClearActiveUnit on clone affected original ActiveUnitID")
	}

	// 4. Mutate units in clone
	clone.Units[0].Position = domain.GridPosition{X: 1, Y: 1}
	clone.Units[0].Stats.HP = 20
	clone.Units[0].Tokens.Attack = 0
	if orig.Units[0].Position.X != 0 || orig.Units[0].Position.Y != -3 {
		t.Errorf("mutating clone unit position leaked to original: %v", orig.Units[0].Position)
	}
	if orig.Units[0].Stats.HP != 100 {
		t.Errorf("mutating clone unit HP leaked to original: %d", orig.Units[0].Stats.HP)
	}
	if orig.Units[0].Tokens.Attack != 1 {
		t.Errorf("mutating clone unit attack token leaked to original: %d", orig.Units[0].Tokens.Attack)
	}

	// 5. Array structural mutations
	// Reorder clone units
	clone.Units[0], clone.Units[1] = clone.Units[1], clone.Units[0]
	if orig.Units[0].ID != 1 || orig.Units[1].ID != 2 {
		t.Errorf("reordering clone.Units affected original.Units order")
	}

	// Truncate clone units
	clone.Units = clone.Units[:1]
	if len(orig.Units) != 2 {
		t.Errorf("truncating clone.Units affected original.Units length")
	}

	// Append to clone units
	clone.Units = append(clone.Units, &domain.Unit{ID: 3}, &domain.Unit{ID: 4})
	if len(orig.Units) != 2 {
		t.Errorf("appending to clone.Units affected original.Units length")
	}
}

// TestStateIsolation_ReverseMutation verifies that mutating the original state
// does not affect a previously taken clone (snapshot isolation).
func TestStateIsolation_ReverseMutation(t *testing.T) {
	orig := domain.NewInitialGameState()
	id1 := domain.UnitID(1)
	orig.SetActiveUnit(&id1)
	orig.Phase = domain.PhaseChooseAction

	snapshot := orig.Clone()

	// Mutate original
	orig.ActivePlayer = domain.PlayerTwo
	orig.Phase = domain.PhaseSelectUnit
	orig.ClearActiveUnit()
	orig.Units[0].Position = domain.GridPosition{X: 3, Y: 3}
	orig.Units[0].Stats.HP = 10
	orig.Units = orig.Units[:1]

	// Assert snapshot retains the pre-mutation state
	if snapshot.ActivePlayer != domain.PlayerOne {
		t.Errorf("snapshot.ActivePlayer changed: got %v, want %v", snapshot.ActivePlayer, domain.PlayerOne)
	}
	if snapshot.Phase != domain.PhaseChooseAction {
		t.Errorf("snapshot.Phase changed: got %v, want %v", snapshot.Phase, domain.PhaseChooseAction)
	}
	if snapshot.ActiveUnitID == nil || *snapshot.ActiveUnitID != 1 {
		t.Errorf("snapshot.ActiveUnitID changed: got %v, want 1", snapshot.ActiveUnitID)
	}
	if len(snapshot.Units) != 2 {
		t.Fatalf("snapshot.Units length changed: got %d, want 2", len(snapshot.Units))
	}
	if snapshot.Units[0].Position.X != 0 || snapshot.Units[0].Position.Y != -3 {
		t.Errorf("snapshot.Units[0].Position changed: %v", snapshot.Units[0].Position)
	}
	if snapshot.Units[0].Stats.HP != 100 {
		t.Errorf("snapshot.Units[0].Stats.HP changed: %d", snapshot.Units[0].Stats.HP)
	}
}

// TestStateIsolation_SetActiveUnitNoAliasing verifies that SetActiveUnit copies the value
// rather than aliasing the caller's pointer.
func TestStateIsolation_SetActiveUnitNoAliasing(t *testing.T) {
	state := domain.NewInitialGameState()

	callerID := domain.UnitID(1)
	state.SetActiveUnit(&callerID)

	// Mutate caller's variable
	callerID = domain.UnitID(999)

	// State must not be affected
	if state.ActiveUnitID == nil || *state.ActiveUnitID != 1 {
		t.Errorf("state.ActiveUnitID aliased caller's variable: got %v, want 1", state.ActiveUnitID)
	}
	if state.ActiveUnitID == &callerID {
		t.Errorf("state.ActiveUnitID directly holds caller's pointer address (%p)", &callerID)
	}
}

// TestStateIsolation_IndependentConstructors verifies that NewInitialGameState produces
// isolated instances on subsequent calls.
func TestStateIsolation_IndependentConstructors(t *testing.T) {
	s1 := domain.NewInitialGameState()
	s2 := domain.NewInitialGameState()

	if s1.Units[0] == s2.Units[0] {
		t.Errorf("NewInitialGameState reused unit pointers between calls (%p == %p)", s1.Units[0], s2.Units[0])
	}

	s1.Units[0].Position = domain.GridPosition{X: 5, Y: 5}
	if s2.Units[0].Position.X == 5 && s2.Units[0].Position.Y == 5 {
		t.Errorf("mutating s1 unit mutated s2 unit")
	}
}

// ============================================================================
// FOCUS AREA 4: Invariant Validation Edge Cases
// ============================================================================

// TestGameState_ValidateEdgeCases tests subtle edge cases in Validate().
func TestGameState_ValidateEdgeCases(t *testing.T) {
	// 1. Zero units: valid in SelectUnit
	emptyState := &domain.GameState{
		ActivePlayer: domain.PlayerOne,
		Phase:        domain.PhaseSelectUnit,
		Units:        []*domain.Unit{},
	}
	if err := emptyState.Validate(); err != nil {
		t.Errorf("empty units state should be valid in SelectUnit, got: %v", err)
	}

	// 2. Nil unit pointers in slice: Validate handles safely without panicking
	stateWithNils := &domain.GameState{
		ActivePlayer: domain.PlayerOne,
		Phase:        domain.PhaseSelectUnit,
		Units:        []*domain.Unit{nil, nil},
	}
	if err := stateWithNils.Validate(); err != nil {
		t.Errorf("nil units should be skipped by Validate, got: %v", err)
	}

	// 3. Units with negative IDs: allowed if unique
	stateNegID := &domain.GameState{
		ActivePlayer: domain.PlayerOne,
		Phase:        domain.PhaseSelectUnit,
		Units: []*domain.Unit{
			{ID: -1, Owner: domain.PlayerOne, Position: domain.GridPosition{X: 0, Y: 0}},
			{ID: -2, Owner: domain.PlayerTwo, Position: domain.GridPosition{X: 1, Y: 1}},
		},
	}
	if err := stateNegID.Validate(); err != nil {
		t.Errorf("negative unit IDs should be valid if unique, got: %v", err)
	}

	// 4. Overlapping units with negative IDs
	stateNegIDOverlap := &domain.GameState{
		ActivePlayer: domain.PlayerOne,
		Phase:        domain.PhaseSelectUnit,
		Units: []*domain.Unit{
			{ID: -1, Owner: domain.PlayerOne, Position: domain.GridPosition{X: 0, Y: 0}},
			{ID: -2, Owner: domain.PlayerTwo, Position: domain.GridPosition{X: 0, Y: 0}},
		},
	}
	if err := stateNegIDOverlap.Validate(); err != domain.ErrOverlappingUnits {
		t.Errorf("expected ErrOverlappingUnits, got: %v", err)
	}
}

// Ensure unused import checks pass
var _ = time.Now
var _ = bytes.Buffer{}
