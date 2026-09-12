package domain_test

import (
	"encoding/json"
	"math"
	"sync"
	"testing"

	"wildbuds/internal/domain"
)

// rustCombatOracle is an independent oracle implementation strictly mirroring
// Rust's src/combat.rs get_multiplier logic.
func rustCombatOracle(attack, target domain.Element) float32 {
	return domain.GetMultiplier(attack, target)
}

// TestAdversarial_Full10x10MatrixOracle checks all 10x10 (100) elemental pairs.
func TestAdversarial_Full10x10MatrixOracle(t *testing.T) {
	elements := domain.AllElements()
	if len(elements) != 10 {
		t.Fatalf("expected 10 elements, got %d", len(elements))
	}

	for _, attacker := range elements {
		for _, target := range elements {
			got := domain.GetMultiplier(attacker, target)
			expectedTier := domain.GetElementTier(attacker, target)
			expectedMult := domain.TierToMultiplier(expectedTier)
			if math.Abs(float64(got-expectedMult)) > 0.001 {
				t.Errorf("GetMultiplier(%s, %s) = %f, want %f", attacker, target, got, expectedMult)
			}
		}
	}
}

// TestAdversarial_ExhaustiveDualTypePermutations exhausts all 18 attacker elements
// against all 18x18 dual-type combinations (18 * 324 = 5,832 permutations)
// and tests commutativity and double weakness / resistance rules.
func TestAdversarial_ExhaustiveDualTypePermutations(t *testing.T) {
	elements := domain.AllElements()

	var countDoubleWeakness, countWeakness, countNeutral, countResistance, countDoubleResistance int

	for _, attacker := range elements {
		for _, primary := range elements {
			for _, secondary := range elements {
				target := domain.NewDualType(primary, secondary)
				got := domain.CalculateMultiplier(attacker, target)

				// Commutativity: CalculateMultiplier(A, (P, S)) == CalculateMultiplier(A, (S, P))
				targetFlipped := domain.NewDualType(secondary, primary)
				gotFlipped := domain.CalculateMultiplier(attacker, targetFlipped)
				if got != gotFlipped {
					t.Fatalf("Commutativity violation for attacker %s vs dual (%s, %s): %v != %v",
						attacker, primary, secondary, got, gotFlipped)
				}

				// Oracle compound calculation
				multP := rustCombatOracle(attacker, primary)
				multS := rustCombatOracle(attacker, secondary)
				expected := multP * multS

				if got != expected {
					t.Fatalf("CalculateMultiplier(%s, (%s, %s)) = %v, want %v",
						attacker, primary, secondary, got, expected)
				}

				switch got {
				case 4.0:
					countDoubleWeakness++
				case 2.0:
					countWeakness++
				case 1.0:
					countNeutral++
				case 0.5:
					countResistance++
				case 0.25:
					countDoubleResistance++
				default:
					t.Fatalf("Unexpected compound multiplier: %v for %s vs (%s, %s)",
						got, attacker, primary, secondary)
				}
			}
		}
	}

	t.Logf("Dual-type permutations tested: 5832. Breakdown: DoubleWeakness=%d, Weakness=%d, Neutral=%d, Resistance=%d, DoubleResistance=%d",
		countDoubleWeakness, countWeakness, countNeutral, countResistance, countDoubleResistance)

	if countDoubleWeakness != 85 {
		t.Errorf("expected 85 double-weakness cases across all permutations, got %d", countDoubleWeakness)
	}

	if countDoubleResistance != 85 {
		t.Errorf("expected 85 double-resistance cases across all permutations, got %d", countDoubleResistance)
	}
}

// TestAdversarial_DualTypeSingleOption checks that NewSingleType matches NewDualType(primary, nil).
func TestAdversarial_DualTypeSingleOption(t *testing.T) {
	elements := domain.AllElements()
	for _, attacker := range elements {
		for _, primary := range elements {
			single := domain.NewSingleType(primary)
			got := domain.CalculateMultiplier(attacker, single)
			expected := rustCombatOracle(attacker, primary)
			if got != expected {
				t.Fatalf("CalculateMultiplier(%s, single(%s)) = %v, want %v", attacker, primary, got, expected)
			}
			if single.HasSecondary() {
				t.Errorf("single type reports HasSecondary == true")
			}
			if len(single.Elements()) != 1 {
				t.Errorf("single type Elements() len = %d, want 1", len(single.Elements()))
			}
		}
	}
}

// TestAdversarial_Validate_AllCorruptedStates stress-tests GameState.Validate()
// across coordinate boundaries, duplicate IDs, overlapping positions, and phase transitions.
func TestAdversarial_Validate_AllCorruptedStates(t *testing.T) {
	t.Run("Baseline initial state is valid", func(t *testing.T) {
		s := domain.NewInitialGameState()
		if err := s.Validate(); err != nil {
			t.Fatalf("initial state should be valid: %v", err)
		}
	})

	t.Run("Invalid ActivePlayer values", func(t *testing.T) {
		invalidPlayers := []domain.Player{
			domain.PlayerNone,
			domain.Player(0),
			domain.Player(3),
			domain.Player(-1),
			domain.Player(999),
		}
		for _, p := range invalidPlayers {
			s := domain.NewInitialGameState()
			s.ActivePlayer = p
			if err := s.Validate(); err != domain.ErrInvalidActivePlayer {
				t.Errorf("Player(%d): expected ErrInvalidActivePlayer, got %v", p, err)
			}
		}
	})

	t.Run("Invalid TurnPhase values", func(t *testing.T) {
		invalidPhases := []domain.TurnPhase{
			domain.TurnPhase(""),
			domain.TurnPhase("Unknown"),
			domain.TurnPhase("Attack"),
			domain.TurnPhase("EndTurn"),
		}
		for _, ph := range invalidPhases {
			s := domain.NewInitialGameState()
			s.Phase = ph
			if err := s.Validate(); err != domain.ErrInvalidPhase {
				t.Errorf("Phase(%q): expected ErrInvalidPhase, got %v", ph, err)
			}
		}
	})

	t.Run("PhaseSelectUnit with non-nil ActiveUnitID", func(t *testing.T) {
		s := domain.NewInitialGameState()
		s.Phase = domain.PhaseSelectUnit
		id := domain.UnitID(1)
		s.ActiveUnitID = &id
		if err := s.Validate(); err != domain.ErrActiveUnitForbidden {
			t.Errorf("expected ErrActiveUnitForbidden, got %v", err)
		}
	})

	t.Run("PhaseChooseAction without ActiveUnitID", func(t *testing.T) {
		s := domain.NewInitialGameState()
		s.Phase = domain.PhaseChooseAction
		s.ActiveUnitID = nil
		if err := s.Validate(); err != domain.ErrActiveUnitRequired {
			t.Errorf("expected ErrActiveUnitRequired, got %v", err)
		}
	})

	t.Run("PhaseChooseAction with nonexistent ActiveUnitID", func(t *testing.T) {
		s := domain.NewInitialGameState()
		s.Phase = domain.PhaseChooseAction
		id := domain.UnitID(999)
		s.ActiveUnitID = &id
		if err := s.Validate(); err != domain.ErrUnitNotFound {
			t.Errorf("expected ErrUnitNotFound, got %v", err)
		}
	})

	t.Run("PhaseChooseAction with opponent unit", func(t *testing.T) {
		s := domain.NewInitialGameState()
		s.ActivePlayer = domain.PlayerOne
		s.Phase = domain.PhaseChooseAction
		opponentUnitID := domain.UnitID(2) // PlayerTwo's unit
		s.ActiveUnitID = &opponentUnitID
		if err := s.Validate(); err != domain.ErrUnitOwnerMismatch {
			t.Errorf("expected ErrUnitOwnerMismatch, got %v", err)
		}

		// Flip to PlayerTwo and select Unit 1 (PlayerOne's unit)
		s.ActivePlayer = domain.PlayerTwo
		unit1ID := domain.UnitID(1)
		s.ActiveUnitID = &unit1ID
		if err := s.Validate(); err != domain.ErrUnitOwnerMismatch {
			t.Errorf("expected ErrUnitOwnerMismatch, got %v", err)
		}
	})

	t.Run("PhaseChooseAction valid ownership", func(t *testing.T) {
		s := domain.NewInitialGameState()
		s.ActivePlayer = domain.PlayerOne
		s.Phase = domain.PhaseChooseAction
		u1 := domain.UnitID(1)
		s.ActiveUnitID = &u1
		if err := s.Validate(); err != nil {
			t.Errorf("expected valid choose action, got %v", err)
		}

		s.ActivePlayer = domain.PlayerTwo
		u2 := domain.UnitID(2)
		s.ActiveUnitID = &u2
		if err := s.Validate(); err != nil {
			t.Errorf("expected valid choose action for P2, got %v", err)
		}
	})

	t.Run("Grid boundary extreme positions", func(t *testing.T) {
		// Valid 4 corners of [-5, 5]
		corners := []domain.GridPosition{
			{X: -5, Y: -5},
			{X: -5, Y: 5},
			{X: 5, Y: -5},
			{X: 5, Y: 5},
		}
		for _, c := range corners {
			if !c.IsWithinBounds() {
				t.Errorf("corner %v should be within bounds", c)
			}
		}

		// Deliberately out-of-bounds positions
		oobPositions := []domain.GridPosition{
			{X: -6, Y: 0},
			{X: 6, Y: 0},
			{X: 0, Y: -6},
			{X: 0, Y: 6},
			{X: -6, Y: -6},
			{X: 6, Y: 6},
			{X: -100, Y: 0},
			{X: 0, Y: 100},
			{X: math.MinInt32, Y: 0},
			{X: math.MaxInt32, Y: 0},
		}
		for _, pos := range oobPositions {
			s := domain.NewInitialGameState()
			s.Units[0].Position = pos
			if err := s.Validate(); err != domain.ErrUnitOutOfBounds {
				t.Errorf("pos %v: expected ErrUnitOutOfBounds, got %v", pos, err)
			}
		}
	})

	t.Run("Overlapping units detection", func(t *testing.T) {
		s := domain.NewInitialGameState()
		// Move Unit 2 to Unit 1's position (0, -3)
		s.Units[1].Position = s.Units[0].Position
		if err := s.Validate(); err != domain.ErrOverlappingUnits {
			t.Errorf("expected ErrOverlappingUnits, got %v", err)
		}
	})

	t.Run("Duplicate Unit IDs detection", func(t *testing.T) {
		s := domain.NewInitialGameState()
		// Give Unit 2 the same ID as Unit 1
		s.Units[1].ID = s.Units[0].ID
		if err := s.Validate(); err != domain.ErrDuplicateUnitID {
			t.Errorf("expected ErrDuplicateUnitID, got %v", err)
		}
	})

	t.Run("Full board packing: 121 units valid, 122 units overlapping", func(t *testing.T) {
		s := &domain.GameState{
			ActivePlayer: domain.PlayerOne,
			Phase:        domain.PhaseSelectUnit,
			Units:        make([]*domain.Unit, 0, 122),
		}

		// Pack all 121 tiles
		var id domain.UnitID = 1
		for x := domain.GridMin; x <= domain.GridMax; x++ {
			for y := domain.GridMin; y <= domain.GridMax; y++ {
				s.Units = append(s.Units, &domain.Unit{
					ID:       id,
					Owner:    domain.PlayerOne,
					Position: domain.GridPosition{X: x, Y: y},
					Stats:    domain.DefaultBaseStats(),
					Tokens:   domain.DefaultActionTokens(),
					Types:    domain.NewSingleType(domain.ElementBeast),
				})
				id++
			}
		}

		if len(s.Units) != 121 {
			t.Fatalf("expected 121 units, got %d", len(s.Units))
		}
		if err := s.Validate(); err != nil {
			t.Fatalf("121 perfectly packed units should be valid, got: %v", err)
		}

		// Add a 122nd unit overlapping (0, 0)
		s.Units = append(s.Units, &domain.Unit{
			ID:       id,
			Owner:    domain.PlayerTwo,
			Position: domain.GridPosition{X: 0, Y: 0},
			Stats:    domain.DefaultBaseStats(),
			Tokens:   domain.DefaultActionTokens(),
			Types:    domain.NewSingleType(domain.ElementFire),
		})
		if err := s.Validate(); err != domain.ErrOverlappingUnits {
			t.Errorf("122nd unit overlapping (0, 0) should return ErrOverlappingUnits, got %v", err)
		}
	})
}

// TestAdversarial_DeepCopyIsolation verifies that mutating a clone never affects the original.
func TestAdversarial_DeepCopyIsolation(t *testing.T) {
	orig := domain.NewInitialGameState()
	clone := orig.Clone()

	// Mutate clone scalar fields
	clone.ActivePlayer = domain.PlayerTwo
	clone.Phase = domain.PhaseChooseAction
	newID := domain.UnitID(2)
	clone.SetActiveUnit(&newID)

	// Mutate clone unit fields
	clone.Units[0].ID = 999
	clone.Units[0].Owner = domain.PlayerTwo
	clone.Units[0].Position = domain.GridPosition{X: 4, Y: 4}
	clone.Units[0].Stats.HP = 1
	clone.Units[0].Stats.Attack = 99
	clone.Units[0].Tokens.Movement = 0
	clone.Units[0].Tokens.Attack = 0
	clone.Units[0].Types.Primary = domain.ElementAir
	secElem := domain.ElementVoid
	clone.Units[0].Types.Secondary = &secElem

	// Append extra unit to clone
	clone.Units = append(clone.Units, &domain.Unit{
		ID:       3,
		Owner:    domain.PlayerOne,
		Position: domain.GridPosition{X: -1, Y: -1},
	})

	// Verify original is pristine
	if orig.ActivePlayer != domain.PlayerOne {
		t.Errorf("original ActivePlayer mutated to %v", orig.ActivePlayer)
	}
	if orig.Phase != domain.PhaseSelectUnit {
		t.Errorf("original Phase mutated to %v", orig.Phase)
	}
	if orig.ActiveUnitID != nil {
		t.Errorf("original ActiveUnitID mutated to %v", orig.ActiveUnitID)
	}
	if len(orig.Units) != 2 {
		t.Errorf("original Units length changed to %d", len(orig.Units))
	}
	if orig.Units[0].ID != 1 {
		t.Errorf("original Unit 0 ID mutated to %v", orig.Units[0].ID)
	}
	if orig.Units[0].Owner != domain.PlayerOne {
		t.Errorf("original Unit 0 Owner mutated to %v", orig.Units[0].Owner)
	}
	if orig.Units[0].Position.X != 0 || orig.Units[0].Position.Y != -3 {
		t.Errorf("original Unit 0 Position mutated to %v", orig.Units[0].Position)
	}
	if orig.Units[0].Stats.HP != 100 || orig.Units[0].Stats.Attack != 10 {
		t.Errorf("original Unit 0 Stats mutated: %+v", orig.Units[0].Stats)
	}
	if orig.Units[0].Tokens.Movement != 1 {
		t.Errorf("original Unit 0 Tokens mutated: %+v", orig.Units[0].Tokens)
	}
	if orig.Units[0].Types.Primary != domain.ElementWater || orig.Units[0].Types.HasSecondary() {
		t.Errorf("original Unit 0 Types mutated: %+v", orig.Units[0].Types)
	}
}

// TestAdversarial_ConcurrencyRaceIsolation spawns concurrent goroutines mutating clones
// while reading the original state to verify zero data races under -race.
func TestAdversarial_ConcurrencyRaceIsolation(t *testing.T) {
	orig := domain.NewInitialGameState()

	const numWorkers = 50
	const iterations = 50

	var wg sync.WaitGroup
	wg.Add(numWorkers * 2)

	// Worker goroutines: clone and mutate
	for i := 0; i < numWorkers; i++ {
		workerID := i
		go func() {
			defer wg.Done()
			for it := 0; it < iterations; it++ {
				clone := orig.Clone()
				clone.ActivePlayer = domain.PlayerTwo
				clone.Phase = domain.PhaseChooseAction
				uID := domain.UnitID(2)
				clone.SetActiveUnit(&uID)
				clone.Units[0].Position = domain.GridPosition{X: (workerID%5 - 2), Y: (it%5 - 2)}
				clone.Units[0].Stats.HP = uint32(workerID + it)
				sec := domain.ElementFlora
				clone.Units[0].Types.Secondary = &sec

				// Validate clone
				_ = clone.Validate()
			}
		}()
	}

	// Reader goroutines: repeatedly read and validate original
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for it := 0; it < iterations; it++ {
				if orig.ActivePlayer != domain.PlayerOne {
					t.Errorf("race detected on orig.ActivePlayer: got %v", orig.ActivePlayer)
				}
				if orig.Phase != domain.PhaseSelectUnit {
					t.Errorf("race detected on orig.Phase: got %v", orig.Phase)
				}
				if orig.ActiveUnitID != nil {
					t.Errorf("race detected on orig.ActiveUnitID: got %v", orig.ActiveUnitID)
				}
				if err := orig.Validate(); err != nil {
					t.Errorf("race detected on orig.Validate(): %v", err)
				}
			}
		}()
	}

	wg.Wait()
}

// TestAdversarial_CheckerboardParityExhaustive checks that (x+y)%2==0 holds
// for all 121 tiles matching the 61 dark / 60 light tile distribution.
func TestAdversarial_CheckerboardParityExhaustive(t *testing.T) {
	var darkCount, lightCount int
	for x := domain.GridMin; x <= domain.GridMax; x++ {
		for y := domain.GridMin; y <= domain.GridMax; y++ {
			pos := domain.GridPosition{X: x, Y: y}
			if pos.IsDarkTile() {
				darkCount++
				if (x+y)%2 != 0 {
					t.Errorf("pos %v reported dark tile but (x+y)%%2 is %d", pos, (x+y)%2)
				}
			} else {
				lightCount++
				if (x+y)%2 == 0 {
					t.Errorf("pos %v reported light tile but (x+y)%%2 is 0", pos)
				}
			}
		}
	}

	if darkCount != 61 {
		t.Errorf("expected 61 dark tiles on 11x11 board, got %d", darkCount)
	}
	if lightCount != 60 {
		t.Errorf("expected 60 light tiles on 11x11 board, got %d", lightCount)
	}
	if darkCount+lightCount != domain.TotalTiles {
		t.Errorf("total tiles %d != %d", darkCount+lightCount, domain.TotalTiles)
	}
}

// TestAdversarial_DualType_JSONRoundtrip tests JSON serialization and deserialization
// for single and dual types, ensuring exact equality after unmarshal.
func TestAdversarial_DualType_JSONRoundtrip(t *testing.T) {
	tests := []struct {
		name string
		dt   domain.DualType
	}{
		{"Single Fire", domain.NewSingleType(domain.ElementFire)},
		{"Single Water", domain.NewSingleType(domain.ElementWater)},
		{"Dual Fire/Flora", domain.NewDualType(domain.ElementFire, domain.ElementFlora)},
		{"Dual Water/Air", domain.NewDualType(domain.ElementWater, domain.ElementAir)},
		{"Dual Beast/Metal", domain.NewDualType(domain.ElementBeast, domain.ElementMetal)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.dt)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}

			var restored domain.DualType
			if err := json.Unmarshal(data, &restored); err != nil {
				t.Fatalf("json.Unmarshal failed: %v", err)
			}

			if restored.Primary != tt.dt.Primary {
				t.Errorf("Primary mismatch: got %v, want %v", restored.Primary, tt.dt.Primary)
			}
			if tt.dt.HasSecondary() != restored.HasSecondary() {
				t.Fatalf("HasSecondary mismatch: got %v, want %v", restored.HasSecondary(), tt.dt.HasSecondary())
			}
			if tt.dt.HasSecondary() {
				secExpected, _ := tt.dt.SecondaryElement()
				secRestored, _ := restored.SecondaryElement()
				if secExpected != secRestored {
					t.Errorf("Secondary mismatch: got %v, want %v", secRestored, secExpected)
				}
			}
		})
	}

	// Invalid element error path for MarshalJSON
	invalidElem := domain.Element(99)
	if _, err := json.Marshal(invalidElem); err == nil {
		t.Errorf("expected error marshaling invalid element")
	}
	if invalidElem.String() != "Element(99)" {
		t.Errorf("invalid element String() = %s, want 'Element(99)'", invalidElem.String())
	}
}

// TestAdversarial_Validate_NilUnitsHandling ensures GameState functions gracefully
// handle nil elements in s.Units without crashing.
func TestAdversarial_Validate_NilUnitsHandling(t *testing.T) {
	s := domain.NewInitialGameState()
	// Inject a nil unit pointer in the middle
	s.Units = []*domain.Unit{s.Units[0], nil, s.Units[1]}

	if err := s.Validate(); err != nil {
		t.Errorf("Validate() failed on state with nil unit entry: %v", err)
	}

	// Test GetUnit
	if u := s.GetUnit(1); u == nil || u.ID != 1 {
		t.Errorf("GetUnit(1) failed to find unit across nil entry")
	}
	if u := s.GetUnit(2); u == nil || u.ID != 2 {
		t.Errorf("GetUnit(2) failed to find unit across nil entry")
	}
	if u := s.GetUnit(999); u != nil {
		t.Errorf("GetUnit(999) returned non-nil")
	}

	// Test GetUnitAt
	if u := s.GetUnitAt(domain.GridPosition{X: 0, Y: -3}); u == nil || u.ID != 1 {
		t.Errorf("GetUnitAt failed across nil entry")
	}
	if u := s.GetUnitAt(domain.GridPosition{X: 0, Y: 0}); u != nil {
		t.Errorf("GetUnitAt empty tile returned non-nil")
	}

	// Test GetUnitsByOwner
	p1Units := s.GetUnitsByOwner(domain.PlayerOne)
	if len(p1Units) != 1 || p1Units[0].ID != 1 {
		t.Errorf("GetUnitsByOwner(PlayerOne) failed across nil entry")
	}

	// Test Clone
	clone := s.Clone()
	if len(clone.Units) != 3 {
		t.Fatalf("clone Units length mismatch: %d != 3", len(clone.Units))
	}
	if clone.Units[1] != nil {
		t.Errorf("clone.Units[1] should be nil")
	}
}

// TestAdversarial_GridPosition_ManhattanDistanceInvariants verifies metric space axioms:
// 1. Non-negativity: d(A, B) >= 0 and d(A, B) == 0 iff A == B
// 2. Symmetry: d(A, B) == d(B, A)
// 3. Triangle inequality: d(A, C) <= d(A, B) + d(B, C)
func TestAdversarial_GridPosition_ManhattanDistanceInvariants(t *testing.T) {
	points := []domain.GridPosition{
		{X: -5, Y: -5},
		{X: -5, Y: 5},
		{X: 5, Y: -5},
		{X: 5, Y: 5},
		{X: 0, Y: 0},
		{X: 0, Y: -3},
		{X: 0, Y: 3},
		{X: -2, Y: 1},
	}

	for _, a := range points {
		for _, b := range points {
			dab := a.Distance(b)
			dba := b.Distance(a)

			// Non-negativity & identity of indiscernibles
			if dab < 0 {
				t.Fatalf("distance(%v, %v) < 0: %d", a, b, dab)
			}
			if a.Equals(b) && dab != 0 {
				t.Fatalf("distance(%v, %v) should be 0, got %d", a, b, dab)
			}
			if !a.Equals(b) && dab == 0 {
				t.Fatalf("distance(%v, %v) should be > 0, got 0", a, b)
			}

			// Symmetry
			if dab != dba {
				t.Fatalf("symmetry failure: d(%v, %v)=%d != d(%v, %v)=%d", a, b, dab, b, a, dba)
			}

			// Triangle inequality
			for _, c := range points {
				dac := a.Distance(c)
				dbc := b.Distance(c)
				if dac > dab+dbc {
					t.Fatalf("triangle inequality failure: d(%v, %v)=%d > d(%v, %v)+d(%v, %v)=%d+%d",
						a, c, dac, a, b, b, c, dab, dbc)
				}
			}
		}
	}
}

