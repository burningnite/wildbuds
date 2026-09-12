package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Element represents one of the 10 elemental types in Wildbuds,
// as defined in some-mechanics-table.md.
type Element uint8

const (
	ElementBeast Element = iota
	ElementFlora
	ElementWater
	ElementFire
	ElementEarth
	ElementAir
	ElementCold
	ElementMetal
	ElementVoid
	ElementGleam
)

// NumElements is the total number of defined elemental types (10).
const NumElements = 10

// IsValid reports whether the element is within the valid range [ElementBeast, ElementGleam].
func (e Element) IsValid() bool {
	return e <= ElementGleam
}

// String returns the capitalized name of the element.
func (e Element) String() string {
	switch e {
	case ElementBeast:
		return "Beast"
	case ElementFlora:
		return "Flora"
	case ElementWater:
		return "Water"
	case ElementFire:
		return "Fire"
	case ElementEarth:
		return "Earth"
	case ElementAir:
		return "Air"
	case ElementCold:
		return "Cold"
	case ElementMetal:
		return "Metal"
	case ElementVoid:
		return "Void"
	case ElementGleam:
		return "Gleam"
	default:
		return fmt.Sprintf("Element(%d)", e)
	}
}

// Symbol returns the Unicode glyph symbol associated with the element.
func (e Element) Symbol() string {
	switch e {
	case ElementBeast:
		return "𖢥"
	case ElementFlora:
		return "𖧷"
	case ElementWater:
		return "𖦹"
	case ElementFire:
		return "𖥳"
	case ElementEarth:
		return "𖣯"
	case ElementAir:
		return "𖣘"
	case ElementCold:
		return "𖥑"
	case ElementMetal:
		return "𖢒"
	case ElementVoid:
		return "𖡷"
	case ElementGleam:
		return "𖤓"
	default:
		return "?"
	}
}

// MarshalJSON serializes the Element as its string representation.
func (e Element) MarshalJSON() ([]byte, error) {
	if !e.IsValid() {
		return nil, fmt.Errorf("invalid element value: %d", e)
	}
	return json.Marshal(e.String())
}

// UnmarshalJSON deserializes an Element from a string name (case-insensitive) or integer.
func (e *Element) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		parsed, err := ParseElement(s)
		if err != nil {
			return err
		}
		*e = parsed
		return nil
	}

	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		if i >= 0 && i < NumElements {
			*e = Element(i)
			return nil
		}
		return fmt.Errorf("element index out of bounds: %d", i)
	}

	return fmt.Errorf("invalid element data: %s", string(data))
}

// ParseElement parses a string into an Element (case-insensitive).
func ParseElement(s string) (Element, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "beast":
		return ElementBeast, nil
	case "flora":
		return ElementFlora, nil
	case "water":
		return ElementWater, nil
	case "fire":
		return ElementFire, nil
	case "earth":
		return ElementEarth, nil
	case "air":
		return ElementAir, nil
	case "cold":
		return ElementCold, nil
	case "metal":
		return ElementMetal, nil
	case "void":
		return ElementVoid, nil
	case "gleam":
		return ElementGleam, nil
	default:
		return 0, fmt.Errorf("unknown element: %q", s)
	}
}

// AllElements returns a slice of all 10 valid elements in canonical index order.
func AllElements() []Element {
	return []Element{
		ElementBeast,
		ElementFlora,
		ElementWater,
		ElementFire,
		ElementEarth,
		ElementAir,
		ElementCold,
		ElementMetal,
		ElementVoid,
		ElementGleam,
	}
}

// DualType represents a unit's typing, containing a primary element and
// an optional secondary element.
type DualType struct {
	Primary   Element  `json:"primary"`
	Secondary *Element `json:"secondary,omitempty"`
}

// NewSingleType creates a DualType with only a primary element.
func NewSingleType(primary Element) DualType {
	return DualType{
		Primary:   primary,
		Secondary: nil,
	}
}

// NewDualType creates a DualType with both primary and secondary elements.
func NewDualType(primary, secondary Element) DualType {
	sec := secondary
	return DualType{
		Primary:   primary,
		Secondary: &sec,
	}
}

// HasSecondary reports whether a secondary element is present.
func (dt DualType) HasSecondary() bool {
	return dt.Secondary != nil
}

// SecondaryElement returns the secondary element and true if present, or (0, false) if absent.
func (dt DualType) SecondaryElement() (Element, bool) {
	if dt.Secondary == nil {
		return 0, false
	}
	return *dt.Secondary, true
}

// Elements returns a slice of active elements (1 element if single, 2 if dual).
func (dt DualType) Elements() []Element {
	if dt.Secondary != nil {
		return []Element{dt.Primary, *dt.Secondary}
	}
	return []Element{dt.Primary}
}

// Contains Element reports whether the DualType contains the specified element.
func (dt DualType) Contains(e Element) bool {
	if dt.Primary == e {
		return true
	}
	if dt.Secondary != nil && *dt.Secondary == e {
		return true
	}
	return false
}

// Clone creates an independent deep copy of DualType.
func (dt DualType) Clone() DualType {
	if dt.Secondary == nil {
		return DualType{
			Primary:   dt.Primary,
			Secondary: nil,
		}
	}
	sec := *dt.Secondary
	return DualType{
		Primary:   dt.Primary,
		Secondary: &sec,
	}
}

// String returns a human-readable representation like "Fire" or "Fire/Grass".
func (dt DualType) String() string {
	if dt.Secondary != nil {
		return fmt.Sprintf("%s/%s", dt.Primary, *dt.Secondary)
	}
	return dt.Primary.String()
}

// -----------------------------------------------------------------------------
// 10x10 Element Interaction Matrix
// -----------------------------------------------------------------------------
// Attacker is row index, Defender is column index.
// Tiers: +1 = SUPER (↑), 0 = NORMAL (≡), -1 = LESS (↓)
var elementInteractionMatrix = [NumElements][NumElements]int{
	// Beast (0)
	{1, 1, 1, 0, 0, -1, 0, -1, -1, 0},
	// Flora (1)
	{-1, 0, 1, -1, 1, 0, -1, 0, 0, 1},
	// Water (2)
	{0, -1, -1, 1, 0, -1, 1, 0, 1, 0},
	// Fire (3)
	{0, 1, -1, -1, 0, 0, 1, 1, 0, -1},
	// Earth (4)
	{0, -1, 0, 1, -1, 0, -1, 0, 1, 1},
	// Air (5)
	{1, 0, 1, -1, 0, 1, 0, -1, -1, 0},
	// Cold (6)
	{-1, 0, -1, 0, 1, 1, -1, 1, 0, 0},
	// Metal (7)
	{0, 0, 0, 0, 1, 0, 1, -1, -1, 0},
	// Void (8)
	{1, 1, 0, 1, -1, -1, 0, 0, 0, -1},
	// Gleam (9)
	{-1, -1, 0, 0, -1, 1, 0, 1, 1, 0},
}

// GetElementTier returns the base effectiveness tier (+1, 0, or -1) when an attack of attackType
// hits a defender of single targetType.
func GetElementTier(attackType, targetType Element) int {
	if !attackType.IsValid() || !targetType.IsValid() {
		return 0
	}
	return elementInteractionMatrix[attackType][targetType]
}

// CalculateTier calculates the un-clamped combined effectiveness tier when an attack of attackType
// hits a defender with DualType targetTypes.
func CalculateTier(attackType Element, targetTypes DualType) int {
	tier := GetElementTier(attackType, targetTypes.Primary)
	if targetTypes.Secondary != nil {
		tier += GetElementTier(attackType, *targetTypes.Secondary)
	}
	return tier
}

// CalculateTierWithSTAB calculates the final effectiveness tier including STAB bonus (+1 if attacker has matching type),
// clamped to the range [-2, +3] (LEAST to ULTRA).
func CalculateTierWithSTAB(attackType Element, attackerTypes DualType, targetTypes DualType) int {
	tier := CalculateTier(attackType, targetTypes)
	if attackerTypes.Contains(attackType) {
		tier += 1
	}
	// Clamp to range [-2, +3]
	if tier < -2 {
		tier = -2
	}
	if tier > 3 {
		tier = 3
	}
	return tier
}

// TierToString returns the symbolic string representation of a tier ("↓↓", "↓", "≡", "↑", "↑↑", "↑↑↑").
func TierToString(tier int) string {
	switch {
	case tier <= -2:
		return "↓↓"
	case tier == -1:
		return "↓"
	case tier == 0:
		return "≡"
	case tier == 1:
		return "↑"
	case tier == 2:
		return "↑↑"
	default:
		return "↑↑↑"
	}
}

// TierToName returns the English name of a tier ("LEAST", "LESS", "NORMAL", "SUPER", "MEGA", "ULTRA").
func TierToName(tier int) string {
	switch {
	case tier <= -2:
		return "LEAST"
	case tier == -1:
		return "LESS"
	case tier == 0:
		return "NORMAL"
	case tier == 1:
		return "SUPER"
	case tier == 2:
		return "MEGA"
	default:
		return "ULTRA"
	}
}

// TierToMultiplier returns the numeric damage multiplier for a given effectiveness tier:
// LEAST (<= -2) -> 0.25x
// LESS  (-1)    -> 0.5x
// NORMAL (0)    -> 1.0x
// SUPER (+1)    -> 2.0x
// MEGA  (+2)    -> 4.0x
// ULTRA (>= +3) -> 8.0x
func TierToMultiplier(tier int) float32 {
	switch {
	case tier <= -2:
		return 0.25
	case tier == -1:
		return 0.5
	case tier == 0:
		return 1.0
	case tier == 1:
		return 2.0
	case tier == 2:
		return 4.0
	default:
		return 8.0
	}
}

// GetMultiplier returns the damage multiplier for a single elemental matchup.
func GetMultiplier(attackType, targetType Element) float32 {
	return TierToMultiplier(GetElementTier(attackType, targetType))
}

// CalculateMultiplier returns the compound damage multiplier for an attack against a DualType.
func CalculateMultiplier(attackType Element, targetTypes DualType) float32 {
	return TierToMultiplier(CalculateTier(attackType, targetTypes))
}

// CalculateMultiplierWithSTAB returns the compound damage multiplier including STAB.
func CalculateMultiplierWithSTAB(attackType Element, attackerTypes DualType, targetTypes DualType) float32 {
	return TierToMultiplier(CalculateTierWithSTAB(attackType, attackerTypes, targetTypes))
}
