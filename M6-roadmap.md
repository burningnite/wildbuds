# M6: E2E Verification & Adversarial Hardening Roadmap
*Status: COMPLETED*

## 1. Test Harness Setup
- [x] Create `test/e2e/harness_test.go`.
- [x] Build a headless wrapper around `app.Game` that overrides the rendering step.
- [x] Implement utilities to inject simulated inputs into the `InputHandler`.
- [x] Implement utilities to assert the deterministic output state of `GameState`.

## 2. Standard Tiers (1-4)
- [x] **Tier 1 (Features):** Create `tier1_features_test.go`. Test the happy path (select, move, attack, end turn) ensuring standard functionality.
- [x] **Tier 2 (Boundaries):** Create `tier2_boundaries_test.go`. Test moving to grid edges `[-5, 5]`, out-of-bounds movement attempts, and elemental extreme calculations.
- [x] **Tier 3 (Combinations):** Create `tier3_combinations_test.go`. Test moving and attacking in the same turn, ending turn without actions, dual-type damage stacking.
- [x] **Tier 4 (Workloads):** Create `tier4_workloads_test.go`. Stress test the command queue with rapid, high-volume valid commands to ensure no dropped inputs.

## 3. Bug Fixing Phase
- [x] Run Tiers 1-4.
- [x] Isolate, log, and fix any domain logic or resolver state machine bugs discovered.
- [x] Ensure 100% pass rate for Tiers 1-4.

## 4. Tier 5 (Adversarial Hardening)
- [x] Create `test/e2e/tier5_adversarial_test.go`.
- [x] Write tests that simulate a modified, malicious client.
- [x] Inject commands out of turn (e.g., P2 sending a command during P1's turn).
- [x] Inject commands targeting units not owned by the sender.
- [x] Inject commands attempting to move units further than their speed allows.
- [x] Inject completely malformed or arbitrary state payloads directly into the incoming queue.
- [x] *Verification:* Assert that the `Resolver` successfully rejects all malicious payloads and the `GameState` remains perfectly synchronized and uncorrupted.
