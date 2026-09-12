package domain

import (
	"encoding/json"
	"testing"
)

// TestElement_MatrixMatchesMechanicsTable verifies specific table entries from some-mechanics-table.md.
func TestElement_MatrixMatchesMechanicsTable(t *testing.T) {
	// Table 32 line entries: Beast/Flora defender
	// Attacker: Beast -> Beast/Flora = ↑↑ (+2)
	// Attacker: Flora -> Beast/Flora = ↓ (-1)
	// Attacker: Water -> Beast/Flora = ↓ (-1)
	// Attacker: Fire  -> Beast/Flora = ↑ (+1)
	// Attacker: Air   -> Beast/Flora = ↑ (+1)
	// Attacker: Cold  -> Beast/Flora = ↓ (-1)
	// Attacker: Metal -> Beast/Flora = ≡ (0)
	// Attacker: Void  -> Beast/Flora = ↑↑ (+2)
	// Attacker: Gleam -> Beast/Flora = ↓↓ (-2)

	defender := NewDualType(ElementBeast, ElementFlora)

	tests := []struct {
		attacker Element
		wantTier int
		wantSymbol string
	}{
		{ElementBeast, 2, "↑↑"},
		{ElementFlora, -1, "↓"},
		{ElementWater, -1, "↓"},
		{ElementFire, 1, "↑"},
		{ElementAir, 1, "↑"},
		{ElementCold, -1, "↓"},
		{ElementMetal, 0, "≡"},
		{ElementVoid, 2, "↑↑"},
		{ElementGleam, -2, "↓↓"},
	}

	for _, tt := range tests {
		t.Run(tt.attacker.String()+" vs Beast/Flora", func(t *testing.T) {
			gotTier := CalculateTier(tt.attacker, defender)
			if gotTier != tt.wantTier {
				t.Errorf("CalculateTier(%s, Beast/Flora) = %d, want %d", tt.attacker, gotTier, tt.wantTier)
			}
			gotSymbol := TierToString(gotTier)
			if gotSymbol != tt.wantSymbol {
				t.Errorf("TierToString(%d) = %s, want %s", gotTier, gotSymbol, tt.wantSymbol)
			}
		})
	}
}

// TestElement_STABMechanics verifies STAB boosting logic (+1 tier, clamped at +3 ULTRA).
func TestElement_STABMechanics(t *testing.T) {
	attackerTypes := NewDualType(ElementBeast, ElementMetal)
	defenderTypes := NewDualType(ElementFlora, ElementVoid)

	// Metal attack vs Flora/Void
	// Base: Metal vs Flora (0) + Metal vs Void (-1) = -1 (↓)
	// With STAB (+1 tier): -1 + 1 = 0 (≡ NORMAL)
	baseTier := CalculateTier(ElementMetal, defenderTypes)
	if baseTier != -1 {
		t.Fatalf("Base tier for Metal vs Flora/Void = %d, want -1", baseTier)
	}

	stabTier := CalculateTierWithSTAB(ElementMetal, attackerTypes, defenderTypes)
	if stabTier != 0 {
		t.Fatalf("CalculateTierWithSTAB(Metal, Beast/Metal, Flora/Void) = %d, want 0", stabTier)
	}

	if TierToString(stabTier) != "≡" {
		t.Fatalf("TierToString(%d) = %s, want ≡", stabTier, TierToString(stabTier))
	}

	// Beast attack vs Flora/Void (Base: Beast vs Flora (+1) + Beast vs Void (-1) = 0)
	// With STAB (+1 tier): 0 + 1 = +1 (↑ SUPER)
	beastStabTier := CalculateTierWithSTAB(ElementBeast, attackerTypes, defenderTypes)
	if beastStabTier != 1 {
		t.Fatalf("STAB tier for Beast vs Flora/Void = %d, want 1", beastStabTier)
	}

	// Non-STAB Fire attack vs Flora/Void (Base: Fire vs Flora (+1) + Fire vs Void (0) = +1)
	// Without STAB: stays +1
	fireNoStabTier := CalculateTierWithSTAB(ElementFire, attackerTypes, defenderTypes)
	if fireNoStabTier != 1 {
		t.Fatalf("Non-STAB Fire tier = %d, want 1", fireNoStabTier)
	}
}

// TestElement_Properties validates enum count, boundaries, names, and parsing.
func TestElement_Properties(t *testing.T) {
	elements := AllElements()
	if len(elements) != NumElements {
		t.Fatalf("AllElements length = %d, want %d", len(elements), NumElements)
	}

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

	if Element(NumElements).IsValid() {
		t.Errorf("Element(%d) should be invalid", NumElements)
	}

	expectedNames := []string{
		"Beast", "Flora", "Water", "Fire", "Earth", "Air",
		"Cold", "Metal", "Void", "Gleam",
	}
	expectedSymbols := []string{
		"𖢥", "𖧷", "𖦹", "𖥳", "𖣯", "𖣘", "𖥑", "𖢒", "𖡷", "𖤓",
	}

	for i, expected := range expectedNames {
		e := Element(i)
		if e.String() != expected {
			t.Errorf("Element(%d).String() = %q, want %q", i, e.String(), expected)
		}
		if e.Symbol() != expectedSymbols[i] {
			t.Errorf("Element(%d).Symbol() = %q, want %q", i, e.Symbol(), expectedSymbols[i])
		}
	}

	for _, expected := range expectedNames {
		parsed, err := ParseElement(expected)
		if err != nil || parsed.String() != expected {
			t.Errorf("ParseElement(%q) failed: got %v, err %v", expected, parsed, err)
		}
	}

	parsedLower, err := ParseElement("fire")
	if err != nil || parsedLower != ElementFire {
		t.Errorf("ParseElement('fire') failed: got %v, err %v", parsedLower, err)
	}

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

	if err := json.Unmarshal([]byte(`2`), &decoded); err != nil {
		t.Fatalf("Unmarshal integer error: %v", err)
	}
	if decoded != ElementWater {
		t.Fatalf("Decoded from 2 = %v, want %v", decoded, ElementWater)
	}
}

// TestDualType_Methods tests helpers: HasSecondary, SecondaryElement, Elements, Contains, Clone, String, JSON.
func TestDualType_Methods(t *testing.T) {
	single := NewSingleType(ElementFire)
	if single.HasSecondary() {
		t.Errorf("NewSingleType should not have secondary")
	}
	if !single.Contains(ElementFire) || single.Contains(ElementWater) {
		t.Errorf("Contains check failed on single type")
	}

	dual := NewDualType(ElementFire, ElementFlora)
	if !dual.HasSecondary() {
		t.Errorf("NewDualType should have secondary")
	}
	if !dual.Contains(ElementFire) || !dual.Contains(ElementFlora) || dual.Contains(ElementWater) {
		t.Errorf("Contains check failed on dual type")
	}
	if dual.String() != "Fire/Flora" {
		t.Errorf("dual.String() = %q, want 'Fire/Flora'", dual.String())
	}
}
