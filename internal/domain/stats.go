package domain

// BaseStats holds fundamental combat attributes for a unit matching src/components.rs.
type BaseStats struct {
	HP      uint32 `json:"hp"`
	Attack  uint32 `json:"attack"`
	Defense uint32 `json:"defense"`
	Speed   uint32 `json:"speed"`
}

// DefaultBaseStats returns standard initial base stats for wildbuds units.
func DefaultBaseStats() BaseStats {
	return BaseStats{
		HP:      100,
		Attack:  10,
		Defense: 5,
		Speed:   10,
	}
}

// Clone creates an independent copy of BaseStats.
func (b BaseStats) Clone() BaseStats {
	return b
}

// ActionTokens holds the tactical action economy for a unit matching src/components.rs.
type ActionTokens struct {
	Movement uint8 `json:"movement"`
	Attack   uint8 `json:"attack"`
	Special  uint8 `json:"special"`
}

// DefaultActionTokens returns standard initial action tokens (1 movement, 1 attack, 0 special).
func DefaultActionTokens() ActionTokens {
	return ActionTokens{
		Movement: 1,
		Attack:   1,
		Special:  0,
	}
}

// Clone creates an independent copy of ActionTokens.
func (a ActionTokens) Clone() ActionTokens {
	return a
}

// CanMove returns true if the unit has at least 1 movement token remaining.
func (a ActionTokens) CanMove() bool {
	return a.Movement > 0
}

// CanAttack returns true if the unit has at least 1 attack token remaining.
func (a ActionTokens) CanAttack() bool {
	return a.Attack > 0
}

// CanUseSpecial returns true if the unit has at least 1 special action token remaining.
func (a ActionTokens) CanUseSpecial() bool {
	return a.Special > 0
}
