# M1: Core Domain & State Model Roadmap
*Status: COMPLETED*

## 1. Domain Constants
- [x] Define `domain/types.go`.
- [x] Create `GridPosition` struct `(X, Y int)`.
- [x] Define board geometry constants (11x11 grid, `[-5, 5]` boundaries).
- [x] Define phase enum (`PhaseSelectUnit`, `PhaseChooseAction`).
- [x] Define player enum (`PlayerOne`, `PlayerTwo`).

## 2. Combat Types & Matrix
- [x] Define `domain/combat.go`.
- [x] Define 18 Elemental types as constants (e.g., `ElementWater`, `ElementFire`).
- [x] Implement `DualType` struct to hold up to two elements.
- [x] Build the effectiveness matrix mapping `(AttackerElement, DefenderElement) -> float64` damage multiplier.
- [x] Write `combat_test.go` verifying Same-Type (0.5), Super Effective (2.0), and Default (1.0) multipliers.

## 3. Unit Stats
- [x] Define `domain/stats.go`.
- [x] Implement `BaseStats` struct (HP, Attack, Defense, Speed).
- [x] Implement `ActionTokens` struct (Movement, Attack, Special flags).
- [x] Create the `Unit` struct combining ID, Owner, `GridPosition`, `DualType`, `BaseStats`, and `ActionTokens`.

## 4. Game State
- [x] Define `domain/state.go`.
- [x] Implement `GameState` struct holding `Units`, `ActivePlayer`, `ActiveUnitID`, and `TurnPhase`.
- [x] Implement the `NewGameState()` constructor setting up initial units at `(0,-3)` and `(0,3)`.
