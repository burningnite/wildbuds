package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Element represents one of the 18 elemental types in Wildbuds,
// mirroring the Rust enum in src/combat.rs.
type Element uint8

const (
	ElementNormal Element = iota
	ElementFire
	ElementWater
	ElementGrass
	ElementElectric
	ElementIce
	ElementFighting
	ElementPoison
	ElementGround
	ElementFlying
	ElementPsychic
	ElementBug
	ElementRock
	ElementGhost
	ElementDragon
	ElementDark
	ElementSteel
	ElementFairy
)

// NumElements is the total number of defined elemental types (18).
const NumElements = 18

// Multiplier constants
const (
	MultiplierSelfResistance float32 = 0.5
	MultiplierSuperEffective float32 = 2.0
	MultiplierNeutral        float32 = 1.0
)

// IsValid reports whether the element is within the valid range [ElementNormal, ElementFairy].
func (e Element) IsValid() bool {
	return e <= ElementFairy
}

// String returns the capitalized name of the element.
func (e Element) String() string {
	switch e {
	case ElementNormal:
		return "Normal"
	case ElementFire:
		return "Fire"
	case ElementWater:
		return "Water"
	case ElementGrass:
		return "Grass"
	case ElementElectric:
		return "Electric"
	case ElementIce:
		return "Ice"
	case ElementFighting:
		return "Fighting"
	case ElementPoison:
		return "Poison"
	case ElementGround:
		return "Ground"
	case ElementFlying:
		return "Flying"
	case ElementPsychic:
		return "Psychic"
	case ElementBug:
		return "Bug"
	case ElementRock:
		return "Rock"
	case ElementGhost:
		return "Ghost"
	case ElementDragon:
		return "Dragon"
	case ElementDark:
		return "Dark"
	case ElementSteel:
		return "Steel"
	case ElementFairy:
		return "Fairy"
	default:
		return fmt.Sprintf("Element(%d)", e)
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
	case "normal":
		return ElementNormal, nil
	case "fire":
		return ElementFire, nil
	case "water":
		return ElementWater, nil
	case "grass":
		return ElementGrass, nil
	case "electric":
		return ElementElectric, nil
	case "ice":
		return ElementIce, nil
	case "fighting":
		return ElementFighting, nil
	case "poison":
		return ElementPoison, nil
	case "ground":
		return ElementGround, nil
	case "flying":
		return ElementFlying, nil
	case "psychic":
		return ElementPsychic, nil
	case "bug":
		return ElementBug, nil
	case "rock":
		return ElementRock, nil
	case "ghost":
		return ElementGhost, nil
	case "dragon":
		return ElementDragon, nil
	case "dark":
		return ElementDark, nil
	case "steel":
		return ElementSteel, nil
	case "fairy":
		return ElementFairy, nil
	default:
		return 0, fmt.Errorf("unknown element: %q", s)
	}
}

// AllElements returns a slice of all 18 valid elements in canonical index order.
func AllElements() []Element {
	return []Element{
		ElementNormal,
		ElementFire,
		ElementWater,
		ElementGrass,
		ElementElectric,
		ElementIce,
		ElementFighting,
		ElementPoison,
		ElementGround,
		ElementFlying,
		ElementPsychic,
		ElementBug,
		ElementRock,
		ElementGhost,
		ElementDragon,
		ElementDark,
		ElementSteel,
		ElementFairy,
	}
}

// DualType represents a unit's typing, containing a primary element and
// an optional secondary element, mirroring Rust's `DualType(Element, Option<Element>)`.
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

// GetMultiplier returns the elemental damage multiplier when an attack of attackType
// hits a target of single targetType.
//
// Rules matching Rust src/combat.rs:
// 1. Self-resistance: if attackType == targetType, return 0.5.
// 2. Starter triangle advantages return 2.0:
//    - Water vs Fire -> 2.0
//    - Fire vs Grass -> 2.0
//    - Grass vs Water -> 2.0
// 3. Default: all other matchups return 1.0.
func GetMultiplier(attackType, targetType Element) float32 {
	if attackType == targetType {
		return MultiplierSelfResistance
	}
	switch attackType {
	case ElementWater:
		if targetType == ElementFire {
			return MultiplierSuperEffective
		}
	case ElementFire:
		if targetType == ElementGrass {
			return MultiplierSuperEffective
		}
	case ElementGrass:
		if targetType == ElementWater {
			return MultiplierSuperEffective
		}
	}
	return MultiplierNeutral
}

// CalculateMultiplier calculates the compound damage multiplier when an attack of attackType
// hits a target with a DualType (primary and optional secondary).
//
// Formula matching Rust src/combat.rs:
// mult = GetMultiplier(attackType, targetTypes.Primary)
// if targetTypes.Secondary != nil {
//     mult *= GetMultiplier(attackType, *targetTypes.Secondary)
// }
func CalculateMultiplier(attackType Element, targetTypes DualType) float32 {
	mult := GetMultiplier(attackType, targetTypes.Primary)
	if targetTypes.Secondary != nil {
		mult *= GetMultiplier(attackType, *targetTypes.Secondary)
	}
	return mult
}
