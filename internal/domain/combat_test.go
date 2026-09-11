package domain

import (
	"encoding/json"
	"testing"
)

// TestRustBaselineParity verifies exact parity with Rust's
// combat::tests::test_dual_type_multiplier in /home/jack/bin/wildbuds/src/combat.rs.
func TestRustBaselineParity(t *testing.T) {
	// Rust test line 56-57:
	// let target = DualType(Element::Fire, Some(Element::Grass));
	// assert_eq!(calculate_multiplier(Element::Water, &target), 2.0 * 1.0);
	target := NewDualType(ElementFire, ElementGrass)
	got := CalculateMultiplier(ElementWater, target)
	expected := float32(2.0 * 1.0)
	if got != expected {
		t.Fatalf("CalculateMultiplier(ElementWater, DualType(Fire, Grass)) = %v, want %v", got, expected)
	}

	// Rust test line 59-60:
	// let target2 = DualType(Element::Fire, None);
	// assert_eq!(calculate_multiplier(Element::Water, &target2), 2.0);
	target2 := NewSingleType(ElementFire)
	got2 := CalculateMultiplier(ElementWater, target2)
	expected2 := float32(2.0)
	if got2 != expected2 {
		t.Fatalf("CalculateMultiplier(ElementWater, DualType(Fire, nil)) = %v, want %v", got2, expected2)
	}
}

// TestGetMultiplier_SelfResistance ensures all 18 elements have 0.5x self-resistance.
func TestGetMultiplier_SelfResistance(t *testing.T) {
	elements := AllElements()
	for _, elem := range elements {
		got := GetMultiplier(elem, elem)
		if got != MultiplierSelfResistance {
			t.Errorf("GetMultiplier(%s, %s) = %v, want %v", elem, elem, got, MultiplierSelfResistance)
		}
	}
}

// TestGetMultiplier_ElementalTriangle verifies the 3 primary triangle interactions.
func TestGetMultiplier_ElementalTriangle(t *testing.T) {
	triangleTests := []struct {
		attacker Element
		target   Element
		want     float32
	}{
		{ElementWater, ElementFire, MultiplierSuperEffective},
		{ElementFire, ElementGrass, MultiplierSuperEffective},
		{ElementGrass, ElementWater, MultiplierSuperEffective},
	}

	for _, tt := range triangleTests {
		got := GetMultiplier(tt.attacker, tt.target)
		if got != tt.want {
			t.Errorf("GetMultiplier(%s, %s) = %v, want %v", tt.attacker, tt.target, got, tt.want)
		}
	}
}

// TestGetMultiplier_ReverseTriangle verifies that reverse starter interactions are neutral (1.0).
func TestGetMultiplier_ReverseTriangle(t *testing.T) {
	reverseTests := []struct {
		attacker Element
		target   Element
	}{
		{ElementFire, ElementWater},
		{ElementGrass, ElementFire},
		{ElementWater, ElementGrass},
	}

	for _, tt := range reverseTests {
		got := GetMultiplier(tt.attacker, tt.target)
		if got != MultiplierNeutral {
			t.Errorf("GetMultiplier(%s, %s) = %v, want %v (neutral)", tt.attacker, tt.target, got, MultiplierNeutral)
		}
	}
}

// TestGetMultiplier_FullMatrixExhaustive exhausts all 18x18 (324) combinations and verifies distribution.
func TestGetMultiplier_FullMatrixExhaustive(t *testing.T) {
	elements := AllElements()
	var countResistance, countSuperEffective, countNeutral int

	for _, a := range elements {
		for _, b := range elements {
			m := GetMultiplier(a, b)
			switch m {
			case MultiplierSelfResistance:
				countResistance++
				if a != b {
					t.Errorf("Unexpected self-resistance for %s vs %s", a, b)
				}
			case MultiplierSuperEffective:
				countSuperEffective++
				isTriangle := (a == ElementWater && b == ElementFire) ||
					(a == ElementFire && b == ElementGrass) ||
					(a == ElementGrass && b == ElementWater)
				if !isTriangle {
					t.Errorf("Unexpected super effective for %s vs %s", a, b)
				}
			case MultiplierNeutral:
				countNeutral++
			default:
				t.Fatalf("Unexpected multiplier value %v for %s vs %s", m, a, b)
			}
		}
	}

	if countResistance != 18 {
		t.Errorf("Expected 18 resistance matchups, got %d", countResistance)
	}
	if countSuperEffective != 3 {
		t.Errorf("Expected 3 super-effective matchups, got %d", countSuperEffective)
	}
	if countNeutral != 303 {
		t.Errorf("Expected 303 neutral matchups, got %d", countNeutral)
	}
	if countResistance+countSuperEffective+countNeutral != 324 {
		t.Errorf("Total combinations %d != 324", countResistance+countSuperEffective+countNeutral)
	}
}

// TestCalculateMultiplier_DualTypeScenarios tests a wide variety of compound multiplier scenarios.
func TestCalculateMultiplier_DualTypeScenarios(t *testing.T) {
	tests := []struct {
		name     string
		attacker Element
		target   DualType
		want     float32
	}{
		// Single types
		{"Single Fire vs Water", ElementWater, NewSingleType(ElementFire), 2.0},
		{"Single Grass vs Fire", ElementFire, NewSingleType(ElementGrass), 2.0},
		{"Single Water vs Grass", ElementGrass, NewSingleType(ElementWater), 2.0},
		{"Single Fire vs Fire", ElementFire, NewSingleType(ElementFire), 0.5},
		{"Single Water vs Water", ElementWater, NewSingleType(ElementWater), 0.5},
		{"Single Normal vs Normal", ElementNormal, NewSingleType(ElementNormal), 0.5},
		{"Single Normal vs Electric", ElementElectric, NewSingleType(ElementNormal), 1.0},
		{"Single Dark vs Fairy", ElementDark, NewSingleType(ElementFairy), 1.0},

		// Dual types: Weakness + Neutral
		{"Dual Fire/Grass vs Water", ElementWater, NewDualType(ElementFire, ElementGrass), 2.0},
		{"Dual Grass/Fire vs Water (Commutative)", ElementWater, NewDualType(ElementGrass, ElementFire), 2.0},
		{"Dual Grass/Steel vs Fire", ElementFire, NewDualType(ElementGrass, ElementSteel), 2.0},
		{"Dual Water/Rock vs Grass", ElementGrass, NewDualType(ElementWater, ElementRock), 2.0},

		// Dual types: Weakness + Resistance cancellation (2.0 * 0.5 = 1.0)
		{"Dual Fire/Water vs Water", ElementWater, NewDualType(ElementFire, ElementWater), 1.0},
		{"Dual Water/Fire vs Water", ElementWater, NewDualType(ElementWater, ElementFire), 1.0},
		{"Dual Grass/Fire vs Fire", ElementFire, NewDualType(ElementGrass, ElementFire), 1.0},
		{"Dual Water/Grass vs Grass", ElementGrass, NewDualType(ElementWater, ElementGrass), 1.0},

		// Dual types: Double Resistance (0.5 * 0.5 = 0.25)
		{"Dual Fire/Fire vs Fire", ElementFire, NewDualType(ElementFire, ElementFire), 0.25},
		{"Dual Water/Water vs Water", ElementWater, NewDualType(ElementWater, ElementFire), 1.0},
		{"Dual Fire/Fire vs Fire (Explicit)", ElementFire, NewDualType(ElementFire, ElementFire), 0.25},
		{"Dual Water/Water vs Water (Explicit)", ElementWater, NewDualType(ElementWater, ElementWater), 0.25},
		{"Dual Grass/Grass vs Grass", ElementGrass, NewDualType(ElementGrass, ElementGrass), 0.25},

		// Dual types: Neutral + Resistance (1.0 * 0.5 = 0.5)
		{"Dual Normal/Fire vs Fire", ElementFire, NewDualType(ElementNormal, ElementFire), 0.5},
		{"Dual Fire/Normal vs Fire", ElementFire, NewDualType(ElementFire, ElementNormal), 0.5},
		{"Dual Dark/Water vs Water", ElementWater, NewDualType(ElementDark, ElementWater), 0.5},

		// Dual types: Double Neutral (1.0 * 1.0 = 1.0)
		{"Dual Normal/Dark vs Electric", ElementElectric, NewDualType(ElementNormal, ElementDark), 1.0},
		{"Dual Psychic/Ghost vs Dragon", ElementDragon, NewDualType(ElementPsychic, ElementGhost), 1.0},

		// Dual types: Double Weakness (2.0 * 2.0 = 4.0)
		{"Dual Fire/Fire vs Water", ElementWater, NewDualType(ElementFire, ElementFire), 4.0},
		{"Dual Grass/Grass vs Fire", ElementFire, NewDualType(ElementGrass, ElementGrass), 4.0},
		{"Dual Water/Water vs Grass", ElementGrass, NewDualType(ElementWater, ElementWater), 4.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateMultiplier(tt.attacker, tt.target)
			if got != tt.want {
				t.Errorf("CalculateMultiplier(%s, %v) = %v, want %v", tt.attacker, tt.target, got, tt.want)
			}
		})
	}
}

// TestElement_Properties validates enum count, boundaries, names, and parsing.
func TestElement_Properties(t *testing.T) {
	elements := AllElements()
	if len(elements) != NumElements {
		t.Fatalf("AllElements length = %d, want %d", len(elements), NumElements)
	}

	// Verify all elements are unique
	seen := make(map[Element]bool)
	for _, e := range elements {
		if seen[e] {
			t.Errorf("Duplicate element in AllElements: %s", e)
		}
		seen[e] = true
		if !e.IsValid() {
			t.Errorf("Element %s marked as invalid", e)
		}
	}

	// Boundary check on IsValid
	if Element(NumElements).IsValid() {
		t.Errorf("Element(%d) should be invalid", NumElements)
	}
	if Element(255).IsValid() {
		t.Errorf("Element(255) should be invalid")
	}

	// String() checks
	expectedNames := []string{
		"Normal", "Fire", "Water", "Grass", "Electric", "Ice",
		"Fighting", "Poison", "Ground", "Flying", "Psychic", "Bug",
		"Rock", "Ghost", "Dragon", "Dark", "Steel", "Fairy",
	}
	for i, expected := range expectedNames {
		e := Element(i)
		if e.String() != expected {
			t.Errorf("Element(%d).String() = %q, want %q", i, e.String(), expected)
		}
	}

	// ParseElement checks (case-insensitive)
	for _, expected := range expectedNames {
		parsed, err := ParseElement(expected)
		if err != nil || parsed.String() != expected {
			t.Errorf("ParseElement(%q) failed: got %v, err %v", expected, parsed, err)
		}
	}
	// Case-insensitive test
	parsedLower, err := ParseElement("fire")
	if err != nil || parsedLower != ElementFire {
		t.Errorf("ParseElement('fire') failed: got %v, err %v", parsedLower, err)
	}

	// Unknown string test
	_, err = ParseElement("unknown_element")
	if err == nil {
		t.Errorf("ParseElement('unknown_element') expected error, got nil")
	}
}

// TestElement_JSONSerialization tests JSON marshaling/unmarshaling of Element.
func TestElement_JSONSerialization(t *testing.T) {
	e := ElementWater
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	if string(data) != `"Water"` {
		t.Fatalf("Marshal output = %s, want %s", string(data), `"Water"`)
	}

	var decoded Element
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded != ElementWater {
		t.Fatalf("Decoded = %v, want %v", decoded, ElementWater)
	}

	// Integer unmarshaling test
	if err := json.Unmarshal([]byte(`2`), &decoded); err != nil {
		t.Fatalf("Unmarshal integer error: %v", err)
	}
	if decoded != ElementWater {
		t.Fatalf("Decoded from 2 = %v, want %v", decoded, ElementWater)
	}
}

// TestDualType_Methods tests helpers: HasSecondary, SecondaryElement, Elements, Clone, String, JSON.
func TestDualType_Methods(t *testing.T) {
	// Single type
	single := NewSingleType(ElementFire)
	if single.HasSecondary() {
		t.Errorf("NewSingleType should not have secondary")
	}
	if _, ok := single.SecondaryElement(); ok {
		t.Errorf("SecondaryElement on single type should return false")
	}
	if len(single.Elements()) != 1 || single.Elements()[0] != ElementFire {
		t.Errorf("Elements() on single type should have [ElementFire]")
	}
	if single.String() != "Fire" {
		t.Errorf("single.String() = %q, want 'Fire'", single.String())
	}

	// Dual type
	dual := NewDualType(ElementFire, ElementGrass)
	if !dual.HasSecondary() {
		t.Errorf("NewDualType should have secondary")
	}
	sec, ok := dual.SecondaryElement()
	if !ok || sec != ElementGrass {
		t.Errorf("SecondaryElement returned (%v, %v), want (ElementGrass, true)", sec, ok)
	}
	elems := dual.Elements()
	if len(elems) != 2 || elems[0] != ElementFire || elems[1] != ElementGrass {
		t.Errorf("Elements() on dual type should have [ElementFire, ElementGrass]")
	}
	if dual.String() != "Fire/Grass" {
		t.Errorf("dual.String() = %q, want 'Fire/Grass'", dual.String())
	}

	// Clone independence
	cloned := dual.Clone()
	if cloned.String() != dual.String() {
		t.Errorf("Cloned string = %q, want %q", cloned.String(), dual.String())
	}
	// Verify pointers are distinct
	if cloned.Secondary == dual.Secondary {
		t.Errorf("Clone should allocate a separate pointer for Secondary")
	}
	*cloned.Secondary = ElementWater
	if *dual.Secondary != ElementGrass {
		t.Errorf("Modifying clone secondary modified original secondary!")
	}

	// Single Clone independence
	singleCloned := single.Clone()
	if singleCloned.HasSecondary() {
		t.Errorf("singleCloned should have nil secondary")
	}

	// JSON marshaling of DualType
	dualJSON, err := json.Marshal(dual)
	if err != nil {
		t.Fatalf("Failed to marshal DualType: %v", err)
	}
	var unmarshaledDual DualType
	if err := json.Unmarshal(dualJSON, &unmarshaledDual); err != nil {
		t.Fatalf("Failed to unmarshal DualType: %v", err)
	}
	if unmarshaledDual.Primary != ElementFire || *unmarshaledDual.Secondary != ElementGrass {
		t.Fatalf("Unmarshaled dual mismatch: got %v, want %v", unmarshaledDual, dual)
	}

	singleJSON, err := json.Marshal(single)
	if err != nil {
		t.Fatalf("Failed to marshal single DualType: %v", err)
	}
	var unmarshaledSingle DualType
	if err := json.Unmarshal(singleJSON, &unmarshaledSingle); err != nil {
		t.Fatalf("Failed to unmarshal single DualType: %v", err)
	}
	if unmarshaledSingle.Primary != ElementFire || unmarshaledSingle.Secondary != nil {
		t.Fatalf("Unmarshaled single mismatch: got %v, want %v", unmarshaledSingle, single)
	}
}

// TestEdgeCases tests resilience against zero values and out-of-range elements.
func TestEdgeCases(t *testing.T) {
	// Zero value DualType
	var zero DualType // Primary=0 (ElementNormal), Secondary=nil
	if zero.HasSecondary() {
		t.Errorf("Zero DualType has secondary")
	}
	got := CalculateMultiplier(ElementWater, zero)
	if got != 1.0 {
		t.Errorf("CalculateMultiplier with zero DualType = %v, want 1.0", got)
	}

	// Out of range element values should not panic
	outOfRangeA := Element(99)
	outOfRangeB := Element(100)
	if GetMultiplier(outOfRangeA, outOfRangeA) != MultiplierSelfResistance {
		t.Errorf("GetMultiplier with identical out-of-range elements should return 0.5")
	}
	if GetMultiplier(outOfRangeA, outOfRangeB) != MultiplierNeutral {
		t.Errorf("GetMultiplier with distinct out-of-range elements should return 1.0")
	}
}
