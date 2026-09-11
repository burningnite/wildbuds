package resolver_test

import (
	"fmt"
	"math"
	"reflect"
	"sync"
	"testing"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
	"wildbuds/internal/resolver"
)

// ============================================================================
// Adversarial Stress Harness: Gatekeeper Rejection & Zero Mutation
// ============================================================================

func TestChallenger_Gatekeeper_ZeroMutation_AllCombinations(t *testing.T) {
	allCommands := []struct {
		name string
		cmd  commands.GameCommand
	}{
		{
			name: "SelectUnit_OwnTile",
			cmd:  commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: 3}),
		},
		{
			name: "SelectUnit_OpponentTile",
			cmd:  commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3}),
		},
		{
			name: "SelectUnit_EmptyTile",
			cmd:  commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: 0}),
		},
		{
			name: "SelectUnit_OutOfBounds",
			cmd:  commands.NewSelectUnitCommand(domain.GridPosition{X: 99, Y: -99}),
		},
		{
			name: "MoveUnit_ValidTile",
			cmd:  commands.NewMoveUnitCommand(domain.GridPosition{X: 1, Y: 1}),
		},
		{
			name: "MoveUnit_OutOfBounds",
			cmd:  commands.NewMoveUnitCommand(domain.GridPosition{X: -10, Y: 0}),
		},
		{
			name: "Attack_ValidTarget",
			cmd:  commands.NewAttackCommand(domain.GridPosition{X: 0, Y: -3}),
		},
		{
			name: "Attack_EmptyTarget",
			cmd:  commands.NewAttackCommand(domain.GridPosition{X: 2, Y: 2}),
		},
		{
			name: "EndActivation",
			cmd:  commands.NewEndActivationCommand(),
		},
		{
			name: "BogusCommand",
			cmd:  commands.GameCommand{Type: "HACK_STEAL_TURN"},
		},
	}

	invalidSenders := []domain.Player{
		domain.PlayerTwo,  // Opponent during PlayerOne turn
		domain.PlayerNone, // Spectator / Unassigned
		domain.Player(99), // Corrupted player ID
		domain.Player(-1), // Negative player ID
	}

	for _, p := range invalidSenders {
		for _, tc := range allCommands {
			testName := fmt.Sprintf("Sender_%s_%s", p, tc.name)
			t.Run(testName, func(t *testing.T) {
				state := domain.NewInitialGameState()
				// ActivePlayer is PlayerOne
				snapshotBefore := state.Clone()

				r := resolver.NewResolver()
				netCmd := commands.NewNetworkCommand(p, tc.cmd)
				events := r.ResolveCommand(state, netCmd)

				// 1. Must emit exactly one rejection event
				if len(events) != 1 {
					t.Fatalf("expected exactly 1 event, got %d", len(events))
				}
				if events[0].Type != resolver.EventCommandRejected {
					t.Fatalf("expected EventCommandRejected, got %s", events[0].Type)
				}
				if events[0].Reason != resolver.ReasonNotActivePlayer {
					t.Errorf("expected reason %q, got %q", resolver.ReasonNotActivePlayer, events[0].Reason)
				}
				if events[0].Sender == nil || *events[0].Sender != p {
					t.Errorf("expected sender %s in rejection event, got %v", p, events[0].Sender)
				}

				// 2. Strict Zero Mutation: Byte-for-byte state preservation
				if state.ActivePlayer != snapshotBefore.ActivePlayer {
					t.Errorf("ActivePlayer mutated: got %s, want %s", state.ActivePlayer, snapshotBefore.ActivePlayer)
				}
				if !reflect.DeepEqual(state.ActiveUnitID, snapshotBefore.ActiveUnitID) {
					t.Errorf("ActiveUnitID mutated: got %v, want %v", state.ActiveUnitID, snapshotBefore.ActiveUnitID)
				}
				if state.Phase != snapshotBefore.Phase {
					t.Errorf("Phase mutated: got %s, want %s", state.Phase, snapshotBefore.Phase)
				}
				if len(state.Units) != len(snapshotBefore.Units) {
					t.Fatalf("Units slice length changed")
				}
				for i := range state.Units {
					uNow := state.Units[i]
					uPrev := snapshotBefore.Units[i]
					if !reflect.DeepEqual(uNow, uPrev) {
						t.Errorf("Unit %d mutated: got %+v, want %+v", i, uNow, uPrev)
					}
				}

				// 3. Game state invariant check
				if err := state.Validate(); err != nil {
					t.Fatalf("state invariant corrupted after rejection: %v", err)
				}
			})
		}
	}
}

// ============================================================================
// Adversarial Stress Harness: Unit Selection
// ============================================================================

func TestChallenger_SelectUnit_AdversarialMatrix(t *testing.T) {
	r := resolver.NewResolver()

	tests := []struct {
		name       string
		target     domain.GridPosition
		expectPass bool
		wantReason string
	}{
		// Coordinates inside board but empty
		{name: "Empty_Center", target: domain.GridPosition{X: 0, Y: 0}, expectPass: false, wantReason: resolver.ReasonNoUnitAtTarget},
		{name: "Empty_Corner_TopLeft", target: domain.GridPosition{X: -5, Y: 5}, expectPass: false, wantReason: resolver.ReasonNoUnitAtTarget},
		{name: "Empty_Corner_BottomRight", target: domain.GridPosition{X: 5, Y: -5}, expectPass: false, wantReason: resolver.ReasonNoUnitAtTarget},

		// Opponent unit at (0, 3)
		{name: "Opponent_Unit", target: domain.GridPosition{X: 0, Y: 3}, expectPass: false, wantReason: resolver.ReasonUnitNotOwned},

		// Boundary Violations
		{name: "OOB_X_Plus6", target: domain.GridPosition{X: 6, Y: 0}, expectPass: false, wantReason: resolver.ReasonTargetOutOfBounds},
		{name: "OOB_X_Minus6", target: domain.GridPosition{X: -6, Y: 0}, expectPass: false, wantReason: resolver.ReasonTargetOutOfBounds},
		{name: "OOB_Y_Plus6", target: domain.GridPosition{X: 0, Y: 6}, expectPass: false, wantReason: resolver.ReasonTargetOutOfBounds},
		{name: "OOB_Y_Minus6", target: domain.GridPosition{X: 0, Y: -6}, expectPass: false, wantReason: resolver.ReasonTargetOutOfBounds},
		{name: "OOB_Extreme_Positive", target: domain.GridPosition{X: 1000, Y: 1000}, expectPass: false, wantReason: resolver.ReasonTargetOutOfBounds},
		{name: "OOB_Extreme_Negative", target: domain.GridPosition{X: -1000, Y: -1000}, expectPass: false, wantReason: resolver.ReasonTargetOutOfBounds},
		{name: "OOB_MaxInt", target: domain.GridPosition{X: math.MaxInt32, Y: 0}, expectPass: false, wantReason: resolver.ReasonTargetOutOfBounds},
		{name: "OOB_MinInt", target: domain.GridPosition{X: 0, Y: math.MinInt32}, expectPass: false, wantReason: resolver.ReasonTargetOutOfBounds},

		// Legal friendly unit at (0, -3)
		{name: "Friendly_Unit", target: domain.GridPosition{X: 0, Y: -3}, expectPass: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := domain.NewInitialGameState()
			snapBefore := state.Clone()

			cmd := commands.NewNetworkCommand(
				domain.PlayerOne,
				commands.NewSelectUnitCommand(tc.target),
			)
			events := r.ResolveCommand(state, cmd)

			if tc.expectPass {
				if len(events) != 1 || events[0].Type != resolver.EventUnitSelected {
					t.Fatalf("expected EventUnitSelected, got %v", events)
				}
				if state.ActiveUnitID == nil || *state.ActiveUnitID != 1 {
					t.Fatalf("expected ActiveUnitID 1, got %v", state.ActiveUnitID)
				}
				if state.Phase != domain.PhaseChooseAction {
					t.Fatalf("expected PhaseChooseAction, got %s", state.Phase)
				}
				if err := state.Validate(); err != nil {
					t.Fatalf("state invalid after legal select: %v", err)
				}
			} else {
				if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
					t.Fatalf("expected EventCommandRejected, got %v", events)
				}
				if events[0].Reason != tc.wantReason {
					t.Errorf("rejection reason mismatch: got %q, want %q", events[0].Reason, tc.wantReason)
				}
				// Verify zero mutation
				if !reflect.DeepEqual(state, snapBefore) {
					t.Fatalf("state mutated on rejected SelectUnit")
				}
			}
		})
	}
}

// ============================================================================
// Adversarial Stress Harness: Unit Movement
// ============================================================================

func TestChallenger_MoveUnit_AdversarialMatrix(t *testing.T) {
	r := resolver.NewResolver()

	t.Run("MovementWithoutActiveUnit", func(t *testing.T) {
		state := domain.NewInitialGameState()
		snapBefore := state.Clone()

		cmd := commands.NewNetworkCommand(
			domain.PlayerOne,
			commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: -2}),
		)
		events := r.ResolveCommand(state, cmd)

		if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
			t.Fatalf("expected rejection when moving without active unit")
		}
		if events[0].Reason != resolver.ReasonNoActiveUnit {
			t.Errorf("reason mismatch: got %q, want %q", events[0].Reason, resolver.ReasonNoActiveUnit)
		}
		if !reflect.DeepEqual(state, snapBefore) {
			t.Fatalf("state mutated on move without active unit")
		}
	})

	t.Run("MovementActiveUnitNotFoundInState", func(t *testing.T) {
		state := domain.NewInitialGameState()
		// Artificially set ActiveUnitID to non-existent unit ID 999
		ghostID := domain.UnitID(999)
		state.ActiveUnitID = &ghostID
		state.Phase = domain.PhaseChooseAction
		snapBefore := state.Clone()

		cmd := commands.NewNetworkCommand(
			domain.PlayerOne,
			commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: -2}),
		)
		events := r.ResolveCommand(state, cmd)

		if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
			t.Fatalf("expected rejection when active unit not found")
		}
		if events[0].Reason != resolver.ReasonActiveUnitNotFound {
			t.Errorf("reason mismatch: got %q, want %q", events[0].Reason, resolver.ReasonActiveUnitNotFound)
		}
		if !reflect.DeepEqual(state, snapBefore) {
			t.Fatalf("state mutated on phantom unit move")
		}
	})

	t.Run("MovementActiveUnitOwnedByOpponent", func(t *testing.T) {
		state := domain.NewInitialGameState()
		// Artificially set ActiveUnitID to opponent unit 2
		oppID := domain.UnitID(2)
		state.ActiveUnitID = &oppID
		state.Phase = domain.PhaseChooseAction
		snapBefore := state.Clone()

		cmd := commands.NewNetworkCommand(
			domain.PlayerOne,
			commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: 2}),
		)
		events := r.ResolveCommand(state, cmd)

		if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
			t.Fatalf("expected rejection when active unit belongs to opponent")
		}
		if events[0].Reason != resolver.ReasonActiveUnitNotOwned {
			t.Errorf("reason mismatch: got %q, want %q", events[0].Reason, resolver.ReasonActiveUnitNotOwned)
		}
		if !reflect.DeepEqual(state, snapBefore) {
			t.Fatalf("state mutated on unowned active unit move")
		}
	})

	t.Run("MovementCollisionObstaclesAndBounds", func(t *testing.T) {
		// Helper to setup unit 1 active
		setupActiveUnit := func() *domain.GameState {
			s := domain.NewInitialGameState()
			r.ResolveCommand(s, commands.NewNetworkCommand(domain.PlayerOne, commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3})))
			return s
		}

		collisionCases := []struct {
			name       string
			dest       domain.GridPosition
			expectPass bool
			wantReason string
		}{
			// Collide with Unit 2 at (0, 3)
			{name: "Collide_OpponentUnit", dest: domain.GridPosition{X: 0, Y: 3}, expectPass: false, wantReason: resolver.ReasonDestinationOccupied},

			// Out of bounds movements
			{name: "OOB_North", dest: domain.GridPosition{X: 0, Y: 6}, expectPass: false, wantReason: resolver.ReasonDestOutOfBounds},
			{name: "OOB_South", dest: domain.GridPosition{X: 0, Y: -6}, expectPass: false, wantReason: resolver.ReasonDestOutOfBounds},
			{name: "OOB_East", dest: domain.GridPosition{X: 6, Y: -3}, expectPass: false, wantReason: resolver.ReasonDestOutOfBounds},
			{name: "OOB_West", dest: domain.GridPosition{X: -6, Y: -3}, expectPass: false, wantReason: resolver.ReasonDestOutOfBounds},
			{name: "OOB_ExtremeCorner", dest: domain.GridPosition{X: -100, Y: 100}, expectPass: false, wantReason: resolver.ReasonDestOutOfBounds},

			// Boundary Extreme Legal Moves: 4 Corners
			{name: "Legal_Corner_TopLeft", dest: domain.GridPosition{X: -5, Y: 5}, expectPass: true},
			{name: "Legal_Corner_TopRight", dest: domain.GridPosition{X: 5, Y: 5}, expectPass: true},
			{name: "Legal_Corner_BottomLeft", dest: domain.GridPosition{X: -5, Y: -5}, expectPass: true},
			{name: "Legal_Corner_BottomRight", dest: domain.GridPosition{X: 5, Y: -5}, expectPass: true},

			// Move to self (same tile) - legal hold position
			{name: "Legal_MoveToSelf", dest: domain.GridPosition{X: 0, Y: -3}, expectPass: true},
		}

		for _, tc := range collisionCases {
			t.Run(tc.name, func(t *testing.T) {
				state := setupActiveUnit()
				snapBefore := state.Clone()

				cmd := commands.NewNetworkCommand(domain.PlayerOne, commands.NewMoveUnitCommand(tc.dest))
				events := r.ResolveCommand(state, cmd)

				if tc.expectPass {
					if len(events) != 1 || events[0].Type != resolver.EventUnitMoved {
						t.Fatalf("expected EventUnitMoved, got %v", events)
					}
					u := state.GetUnit(1)
					if !u.Position.Equals(tc.dest) {
						t.Errorf("unit position not updated: got %s, want %s", u.Position, tc.dest)
					}
					if err := state.Validate(); err != nil {
						t.Fatalf("state invariant failed after valid move: %v", err)
					}
				} else {
					if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
						t.Fatalf("expected EventCommandRejected, got %v", events)
					}
					if events[0].Reason != tc.wantReason {
						t.Errorf("reason mismatch: got %q, want %q", events[0].Reason, tc.wantReason)
					}
					if !reflect.DeepEqual(state, snapBefore) {
						t.Fatalf("state mutated on illegal movement")
					}
				}
			})
		}
	})

	t.Run("MovementCollisionWithFriendlyUnit", func(t *testing.T) {
		state := domain.NewInitialGameState()
		// Add friendly unit 3 at (0, -2)
		state.Units = append(state.Units, &domain.Unit{
			ID:       3,
			Owner:    domain.PlayerOne,
			Position: domain.GridPosition{X: 0, Y: -2},
			Stats:    domain.DefaultBaseStats(),
			Tokens:   domain.DefaultActionTokens(),
			Types:    domain.NewSingleType(domain.ElementWater),
		})

		// Select Unit 1 at (0, -3)
		r.ResolveCommand(state, commands.NewNetworkCommand(domain.PlayerOne, commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3})))
		snapBefore := state.Clone()

		// Attempt to move Unit 1 into Unit 3's tile (0, -2)
		cmd := commands.NewNetworkCommand(domain.PlayerOne, commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: -2}))
		events := r.ResolveCommand(state, cmd)

		if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
			t.Fatalf("expected EventCommandRejected for friendly collision, got %v", events)
		}
		if events[0].Reason != resolver.ReasonDestinationOccupied {
			t.Errorf("reason mismatch: got %q, want %q", events[0].Reason, resolver.ReasonDestinationOccupied)
		}
		if !reflect.DeepEqual(state, snapBefore) {
			t.Fatalf("state mutated on friendly unit collision")
		}
	})
}

// ============================================================================
// Adversarial Stress Harness: Combat Resolution & Multiplier Calculation
// ============================================================================

func TestChallenger_Combat_ElementalMultiplierAndHPDeduction(t *testing.T) {
	r := resolver.NewResolver()

	tests := []struct {
		name           string
		atkElement     domain.Element
		atkAttack      uint32
		defPrimary     domain.Element
		defSecondary   *domain.Element
		defStartHP     uint32
		expectedMult   float32
		expectedDamage uint32
		expectedRemHP  uint32
		expectedDefeat bool
	}{
		// 1. Water -> Fire (Super Effective x2.0)
		{
			name:           "Water_vs_Fire_SuperEffective",
			atkElement:     domain.ElementWater,
			atkAttack:      10,
			defPrimary:     domain.ElementFire,
			defStartHP:     100,
			expectedMult:   2.0,
			expectedDamage: 20,
			expectedRemHP:  80,
			expectedDefeat: false,
		},
		// 2. Fire -> Grass (Super Effective x2.0)
		{
			name:           "Fire_vs_Grass_SuperEffective",
			atkElement:     domain.ElementFire,
			atkAttack:      15,
			defPrimary:     domain.ElementGrass,
			defStartHP:     50,
			expectedMult:   2.0,
			expectedDamage: 30,
			expectedRemHP:  20,
			expectedDefeat: false,
		},
		// 3. Grass -> Water (Super Effective x2.0)
		{
			name:           "Grass_vs_Water_SuperEffective",
			atkElement:     domain.ElementGrass,
			atkAttack:      25,
			defPrimary:     domain.ElementWater,
			defStartHP:     50,
			expectedMult:   2.0,
			expectedDamage: 50,
			expectedRemHP:  0,
			expectedDefeat: true,
		},
		// 4. Water -> Water (Self-Resistant x0.5)
		{
			name:           "Water_vs_Water_Resistant",
			atkElement:     domain.ElementWater,
			atkAttack:      10,
			defPrimary:     domain.ElementWater,
			defStartHP:     100,
			expectedMult:   0.5,
			expectedDamage: 5,
			expectedRemHP:  95,
			expectedDefeat: false,
		},
		// 5. Fire -> Fire (Self-Resistant x0.5)
		{
			name:           "Fire_vs_Fire_Resistant",
			atkElement:     domain.ElementFire,
			atkAttack:      11, // 11 * 0.5 = 5.5 -> rounds to 6
			defPrimary:     domain.ElementFire,
			defStartHP:     100,
			expectedMult:   0.5,
			expectedDamage: 6,
			expectedRemHP:  94,
			expectedDefeat: false,
		},
		// 6. Neutral Matchup: Normal -> Water (x1.0)
		{
			name:           "Normal_vs_Water_Neutral",
			atkElement:     domain.ElementNormal,
			atkAttack:      12,
			defPrimary:     domain.ElementWater,
			defStartHP:     100,
			expectedMult:   1.0,
			expectedDamage: 12,
			expectedRemHP:  88,
			expectedDefeat: false,
		},
		// 7. Dual Type Defender: Water -> Fire/Fire (Double Weakness x4.0)
		{
			name:           "Water_vs_FireFire_DoubleWeakness",
			atkElement:     domain.ElementWater,
			atkAttack:      10,
			defPrimary:     domain.ElementFire,
			defSecondary:   func() *domain.Element { e := domain.ElementFire; return &e }(),
			defStartHP:     100,
			expectedMult:   4.0,
			expectedDamage: 40,
			expectedRemHP:  60,
			expectedDefeat: false,
		},
		// 8. Dual Type Defender: Water -> Fire/Water (Cancellation 2.0 * 0.5 = 1.0)
		{
			name:           "Water_vs_FireWater_Cancellation",
			atkElement:     domain.ElementWater,
			atkAttack:      10,
			defPrimary:     domain.ElementFire,
			defSecondary:   func() *domain.Element { e := domain.ElementWater; return &e }(),
			defStartHP:     100,
			expectedMult:   1.0,
			expectedDamage: 10,
			expectedRemHP:  90,
			expectedDefeat: false,
		},
		// 9. Dual Type Defender: Water -> Water/Water (Double Resistance 0.5 * 0.5 = 0.25)
		{
			name:           "Water_vs_WaterWater_DoubleResist",
			atkElement:     domain.ElementWater,
			atkAttack:      10, // 10 * 0.25 = 2.5 -> rounds to 3 (or 2 depending on rounding)
			defPrimary:     domain.ElementWater,
			defSecondary:   func() *domain.Element { e := domain.ElementWater; return &e }(),
			defStartHP:     100,
			expectedMult:   0.25,
			expectedDamage: uint32(math.Round(10.0 * 0.25)),
			expectedRemHP:  100 - uint32(math.Round(10.0*0.25)),
			expectedDefeat: false,
		},
		// 10. Overkill Lethal Damage (Dmg > RemHP => RemHP = 0, Defeated = true, no uint32 wrap)
		{
			name:           "Overkill_LethalDamage",
			atkElement:     domain.ElementWater,
			atkAttack:      100, // 100 * 2.0 = 200 dmg against 30 HP
			defPrimary:     domain.ElementFire,
			defStartHP:     30,
			expectedMult:   2.0,
			expectedDamage: 200,
			expectedRemHP:  0,
			expectedDefeat: true,
		},
		// 11. Minimum Damage Rule: Small attack with resist (raw > 0 => dmg >= 1)
		{
			name:           "MinDamageRule_NonZeroDamage",
			atkElement:     domain.ElementWater,
			atkAttack:      1, // 1 * 0.25 = 0.25 -> round = 0 -> minimum clamp = 1
			defPrimary:     domain.ElementWater,
			defSecondary:   func() *domain.Element { e := domain.ElementWater; return &e }(),
			defStartHP:     10,
			expectedMult:   0.25,
			expectedDamage: 1,
			expectedRemHP:  9,
			expectedDefeat: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := domain.NewInitialGameState()
			u1 := state.GetUnit(1)
			u2 := state.GetUnit(2)

			u1.Types = domain.NewSingleType(tc.atkElement)
			u1.Stats.Attack = tc.atkAttack

			if tc.defSecondary != nil {
				u2.Types = domain.NewDualType(tc.defPrimary, *tc.defSecondary)
			} else {
				u2.Types = domain.NewSingleType(tc.defPrimary)
			}
			u2.Stats.HP = tc.defStartHP

			// Select Unit 1
			r.ResolveCommand(state, commands.NewNetworkCommand(domain.PlayerOne, commands.NewSelectUnitCommand(u1.Position)))

			// Attack Unit 2
			cmd := commands.NewNetworkCommand(domain.PlayerOne, commands.NewAttackCommand(u2.Position))
			events := r.ResolveCommand(state, cmd)

			expectedEventCount := 1
			if tc.expectedDefeat {
				expectedEventCount = 2
			}

			if len(events) != expectedEventCount {
				t.Fatalf("expected %d events, got %d: %v", expectedEventCount, len(events), events)
			}

			// Check UnitAttacked event
			atkEvent := events[0]
			if atkEvent.Type != resolver.EventUnitAttacked {
				t.Fatalf("event 0 type mismatch: got %s, want %s", atkEvent.Type, resolver.EventUnitAttacked)
			}
			if atkEvent.Multiplier != tc.expectedMult {
				t.Errorf("multiplier mismatch: got %f, want %f", atkEvent.Multiplier, tc.expectedMult)
			}
			if atkEvent.Damage != tc.expectedDamage {
				t.Errorf("damage mismatch: got %d, want %d", atkEvent.Damage, tc.expectedDamage)
			}
			if atkEvent.RemainingHP != tc.expectedRemHP {
				t.Errorf("remaining HP mismatch in event: got %d, want %d", atkEvent.RemainingHP, tc.expectedRemHP)
			}
			if atkEvent.Defeated != tc.expectedDefeat {
				t.Errorf("defeated flag mismatch in event: got %t, want %t", atkEvent.Defeated, tc.expectedDefeat)
			}

			// Check Defender State in GameState
			if u2.Stats.HP != tc.expectedRemHP {
				t.Errorf("defender actual HP in state mismatch: got %d, want %d", u2.Stats.HP, tc.expectedRemHP)
			}

			// Check UnitDefeated event if lethal
			if tc.expectedDefeat {
				defEvent := events[1]
				if defEvent.Type != resolver.EventUnitDefeated {
					t.Fatalf("event 1 type mismatch: got %s, want %s", defEvent.Type, resolver.EventUnitDefeated)
				}
				if *defEvent.UnitID != u2.ID {
					t.Errorf("defeated unit ID mismatch: got %d, want %d", *defEvent.UnitID, u2.ID)
				}
				if *defEvent.Player != u2.Owner {
					t.Errorf("defeated owner mismatch: got %s, want %s", *defEvent.Player, u2.Owner)
				}
			}
		})
	}
}

func TestChallenger_Combat_PreconditionsAndRejections(t *testing.T) {
	r := resolver.NewResolver()

	t.Run("AttackWithoutActiveUnit", func(t *testing.T) {
		state := domain.NewInitialGameState()
		snap := state.Clone()

		cmd := commands.NewNetworkCommand(domain.PlayerOne, commands.NewAttackCommand(domain.GridPosition{X: 0, Y: 3}))
		events := r.ResolveCommand(state, cmd)

		if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
			t.Fatalf("expected EventCommandRejected")
		}
		if events[0].Reason != resolver.ReasonNoActiveUnit {
			t.Errorf("reason mismatch: got %q, want %q", events[0].Reason, resolver.ReasonNoActiveUnit)
		}
		if !reflect.DeepEqual(state, snap) {
			t.Fatalf("state mutated on unselected attack")
		}
	})

	t.Run("AttackFriendlyUnit", func(t *testing.T) {
		state := domain.NewInitialGameState()
		// Add friendly unit 3
		state.Units = append(state.Units, &domain.Unit{
			ID:       3,
			Owner:    domain.PlayerOne,
			Position: domain.GridPosition{X: 1, Y: -3},
			Stats:    domain.DefaultBaseStats(),
			Tokens:   domain.DefaultActionTokens(),
			Types:    domain.NewSingleType(domain.ElementWater),
		})

		r.ResolveCommand(state, commands.NewNetworkCommand(domain.PlayerOne, commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3})))
		snap := state.Clone()

		// Attack friendly unit 3
		cmd := commands.NewNetworkCommand(domain.PlayerOne, commands.NewAttackCommand(domain.GridPosition{X: 1, Y: -3}))
		events := r.ResolveCommand(state, cmd)

		if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
			t.Fatalf("expected EventCommandRejected for friendly attack")
		}
		if events[0].Reason != resolver.ReasonCannotAttackFriendly {
			t.Errorf("reason mismatch: got %q, want %q", events[0].Reason, resolver.ReasonCannotAttackFriendly)
		}
		if !reflect.DeepEqual(state, snap) {
			t.Fatalf("state mutated on friendly attack")
		}
	})

	t.Run("AttackEmptyTile", func(t *testing.T) {
		state := domain.NewInitialGameState()
		r.ResolveCommand(state, commands.NewNetworkCommand(domain.PlayerOne, commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3})))
		snap := state.Clone()

		cmd := commands.NewNetworkCommand(domain.PlayerOne, commands.NewAttackCommand(domain.GridPosition{X: 0, Y: 0}))
		events := r.ResolveCommand(state, cmd)

		if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
			t.Fatalf("expected EventCommandRejected for empty tile attack")
		}
		if events[0].Reason != resolver.ReasonNoUnitAtTarget {
			t.Errorf("reason mismatch: got %q, want %q", events[0].Reason, resolver.ReasonNoUnitAtTarget)
		}
		if !reflect.DeepEqual(state, snap) {
			t.Fatalf("state mutated on empty tile attack")
		}
	})

	t.Run("AttackOutOfBoundsTarget", func(t *testing.T) {
		state := domain.NewInitialGameState()
		r.ResolveCommand(state, commands.NewNetworkCommand(domain.PlayerOne, commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3})))
		snap := state.Clone()

		cmd := commands.NewNetworkCommand(domain.PlayerOne, commands.NewAttackCommand(domain.GridPosition{X: -6, Y: 0}))
		events := r.ResolveCommand(state, cmd)

		if len(events) != 1 || events[0].Type != resolver.EventCommandRejected {
			t.Fatalf("expected EventCommandRejected for out of bounds attack")
		}
		if events[0].Reason != resolver.ReasonTargetOutOfBounds {
			t.Errorf("reason mismatch: got %q, want %q", events[0].Reason, resolver.ReasonTargetOutOfBounds)
		}
		if !reflect.DeepEqual(state, snap) {
			t.Fatalf("state mutated on out of bounds attack")
		}
	})
}

// ============================================================================
// Adversarial Stress Harness: Turn Cycling & State Transition Invariants
// ============================================================================

func TestChallenger_TurnCycling_AlternationAndReset(t *testing.T) {
	r := resolver.NewResolver()
	state := domain.NewInitialGameState()

	// 10 Full Rounds of Alternating Turns (20 half-turns)
	for round := 1; round <= 10; round++ {
		// --- PlayerOne Turn ---
		if state.ActivePlayer != domain.PlayerOne {
			t.Fatalf("round %d: expected ActivePlayer PlayerOne, got %s", round, state.ActivePlayer)
		}
		if state.Phase != domain.PhaseSelectUnit {
			t.Fatalf("round %d: expected PhaseSelectUnit, got %s", round, state.Phase)
		}
		if state.ActiveUnitID != nil {
			t.Fatalf("round %d: expected ActiveUnitID nil at start of turn", round)
		}

		// PlayerTwo tries to steal turn -> MUST BE REJECTED
		stealP2 := commands.NewNetworkCommand(domain.PlayerTwo, commands.NewEndActivationCommand())
		evSteal2 := r.ResolveCommand(state, stealP2)
		if len(evSteal2) != 1 || evSteal2[0].Type != resolver.EventCommandRejected {
			t.Fatalf("round %d: turn theft by PlayerTwo succeeded!", round)
		}

		// P1 Selects Unit 1
		u1 := state.GetUnit(1)
		evSel1 := r.ResolveCommand(state, commands.NewNetworkCommand(domain.PlayerOne, commands.NewSelectUnitCommand(u1.Position)))
		if len(evSel1) != 1 || evSel1[0].Type != resolver.EventUnitSelected {
			t.Fatalf("round %d: P1 select failed", round)
		}
		if state.Phase != domain.PhaseChooseAction {
			t.Fatalf("round %d: expected PhaseChooseAction", round)
		}

		// P1 Ends Activation
		evEnd1 := r.ResolveCommand(state, commands.NewNetworkCommand(domain.PlayerOne, commands.NewEndActivationCommand()))
		if len(evEnd1) != 1 || evEnd1[0].Type != resolver.EventActivationEnded {
			t.Fatalf("round %d: P1 EndActivation failed", round)
		}
		if *evEnd1[0].PreviousPlayer != domain.PlayerOne || *evEnd1[0].NextPlayer != domain.PlayerTwo {
			t.Fatalf("round %d: player transition event mismatch", round)
		}

		// Assertions after P1 ends activation
		if state.ActivePlayer != domain.PlayerTwo {
			t.Fatalf("round %d: expected ActivePlayer to switch to PlayerTwo, got %s", round, state.ActivePlayer)
		}
		if state.ActiveUnitID != nil {
			t.Fatalf("round %d: ActiveUnitID not cleared after EndActivation", round)
		}
		if state.Phase != domain.PhaseSelectUnit {
			t.Fatalf("round %d: Phase not reset to PhaseSelectUnit, got %s", round, state.Phase)
		}
		if err := state.Validate(); err != nil {
			t.Fatalf("round %d: state validation failed after P1 turn: %v", round, err)
		}

		// --- PlayerTwo Turn ---
		// PlayerOne tries to end activation again -> MUST BE REJECTED
		stealP1 := commands.NewNetworkCommand(domain.PlayerOne, commands.NewEndActivationCommand())
		evSteal1 := r.ResolveCommand(state, stealP1)
		if len(evSteal1) != 1 || evSteal1[0].Type != resolver.EventCommandRejected {
			t.Fatalf("round %d: turn theft by PlayerOne succeeded!", round)
		}

		// P2 Selects Unit 2
		u2 := state.GetUnit(2)
		evSel2 := r.ResolveCommand(state, commands.NewNetworkCommand(domain.PlayerTwo, commands.NewSelectUnitCommand(u2.Position)))
		if len(evSel2) != 1 || evSel2[0].Type != resolver.EventUnitSelected {
			t.Fatalf("round %d: P2 select failed", round)
		}

		// P2 Ends Activation
		evEnd2 := r.ResolveCommand(state, commands.NewNetworkCommand(domain.PlayerTwo, commands.NewEndActivationCommand()))
		if len(evEnd2) != 1 || evEnd2[0].Type != resolver.EventActivationEnded {
			t.Fatalf("round %d: P2 EndActivation failed", round)
		}

		// Assertions after P2 ends activation
		if state.ActivePlayer != domain.PlayerOne {
			t.Fatalf("round %d: expected ActivePlayer to alternate back to PlayerOne, got %s", round, state.ActivePlayer)
		}
		if state.ActiveUnitID != nil {
			t.Fatalf("round %d: ActiveUnitID not cleared after P2 turn", round)
		}
		if state.Phase != domain.PhaseSelectUnit {
			t.Fatalf("round %d: Phase not reset to PhaseSelectUnit", round)
		}
		if err := state.Validate(); err != nil {
			t.Fatalf("round %d: state validation failed after P2 turn: %v", round, err)
		}
	}
}

// ============================================================================
// Adversarial Stress Harness: Interleaved Queue Stress & Flood Resilience
// ============================================================================

func TestChallenger_Queue_InterleavedBombardment(t *testing.T) {
	r := resolver.NewResolver()
	state := domain.NewInitialGameState()
	q := commands.NewCommandQueue()

	// We interleave 100 invalid commands ("garbage flood") with 3 valid commands
	for i := 0; i < 50; i++ {
		// Wrong player attack
		q.PushIncoming(commands.NewNetworkCommand(domain.PlayerTwo, commands.NewAttackCommand(domain.GridPosition{X: 0, Y: -3})))
		// Invalid coords select
		q.PushIncoming(commands.NewNetworkCommand(domain.PlayerOne, commands.NewSelectUnitCommand(domain.GridPosition{X: -99, Y: 99})))
	}

	// Valid command 1: P1 selects unit 1 at (0, -3)
	q.PushIncoming(commands.NewNetworkCommand(domain.PlayerOne, commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3})))

	for i := 0; i < 20; i++ {
		// Collide with unit 2
		q.PushIncoming(commands.NewNetworkCommand(domain.PlayerOne, commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: 3})))
		// Wrong player move
		q.PushIncoming(commands.NewNetworkCommand(domain.PlayerTwo, commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: 2})))
	}

	// Valid command 2: P1 moves unit 1 to (0, -2)
	q.PushIncoming(commands.NewNetworkCommand(domain.PlayerOne, commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: -2})))

	// Valid command 3: P1 ends activation
	q.PushIncoming(commands.NewNetworkCommand(domain.PlayerOne, commands.NewEndActivationCommand()) )

	totalExpectedEvents := (50 * 2) + 1 + (20 * 2) + 1 + 1 // 100 + 1 + 40 + 1 + 1 = 143

	events := r.Resolve(state, q)

	if len(events) != totalExpectedEvents {
		t.Fatalf("expected %d events, got %d", totalExpectedEvents, len(events))
	}

	if !q.IsEmpty() {
		t.Errorf("queue not fully drained after resolving flood")
	}

	// Verify that valid commands executed despite massive poison pill bombardment
	u1 := state.GetUnit(1)
	if !u1.Position.Equals(domain.GridPosition{X: 0, Y: -2}) {
		t.Errorf("unit 1 position failed to update: got %s, want (0, -2)", u1.Position)
	}
	if state.ActivePlayer != domain.PlayerTwo {
		t.Errorf("turn failed to pass to PlayerTwo: got %s", state.ActivePlayer)
	}
	if state.Phase != domain.PhaseSelectUnit {
		t.Errorf("phase failed to reset to SelectUnit: got %s", state.Phase)
	}
	if state.ActiveUnitID != nil {
		t.Errorf("active unit failed to clear: got %v", state.ActiveUnitID)
	}
	if err := state.Validate(); err != nil {
		t.Fatalf("state invalid after bombardment: %v", err)
	}
}

// ============================================================================
// Adversarial Stress Harness: Concurrent Queue Safety Under Mutex Splitting
// ============================================================================

func TestChallenger_Queue_ConcurrentPushDrainStress(t *testing.T) {
	q := commands.NewCommandQueue()
	const iterations = 500

	var wg sync.WaitGroup
	wg.Add(4)

	// Goroutine 1: Rapid Outgoing Pusher
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			q.PushOutgoing(commands.NewSelectUnitCommand(domain.GridPosition{X: i % 5, Y: 0}))
		}
	}()

	// Goroutine 2: Rapid Outgoing Drainer
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_ = q.DrainOutgoing()
		}
	}()

	// Goroutine 3: Rapid Incoming Pusher
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			q.PushIncoming(commands.NewNetworkCommand(
				domain.PlayerOne,
				commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: i % 5}),
			))
		}
	}()

	// Goroutine 4: Rapid Incoming Drainer
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_ = q.DrainIncoming()
		}
	}()

	wg.Wait()
	// Clear any stragglers
	q.Clear()
	if !q.IsEmpty() {
		t.Fatalf("queue not empty after concurrent stress")
	}
}
