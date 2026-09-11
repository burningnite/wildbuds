# E2E Test Infra: Wildbuds Go Port

## Test Philosophy
- **Opaque-box & Requirement-driven**: Derived strictly from `ORIGINAL_REQUEST.md`, `PROJECT.md § Feature Inventory`, and the authoritative Rust game specifications. No dependency on internal implementation details.
- **Strict Invariant Verification**: Programmatic verification of the command-first architecture (Input produces commands; only Resolver mutates state).
- **Multi-Tiered Coverage**: Category-Partition + Boundary Value Analysis (BVA) + Pairwise Combinations + Real-World Workload Simulation.

## Feature Inventory Coverage Matrix
| # | Feature | Requirement Source | Tier 1 | Tier 2 | Tier 3 |
|---|---------|-------------------|:------:|:------:|:------:|
| FI-01 | Command Pipeline & Queue | ORIGINAL_REQUEST §R1 | ≥5 | ≥5 | ✓ |
| FI-02 | Turn & Phase State Machine | ORIGINAL_REQUEST §R1 | ≥5 | ≥5 | ✓ |
| FI-03 | SelectUnit Command | ORIGINAL_REQUEST §R1 | ≥5 | ≥5 | ✓ |
| FI-04 | MoveUnit Command | ORIGINAL_REQUEST §R1 | ≥5 | ≥5 | ✓ |
| FI-05 | Attack Command | ORIGINAL_REQUEST §R1 | ≥5 | ≥5 | ✓ |
| FI-06 | EndActivation Command | ORIGINAL_REQUEST §R1 | ≥5 | ≥5 | ✓ |
| FI-07 | Board Geometry & Coordinates | ORIGINAL_REQUEST §R2 | ≥5 | ≥5 | ✓ |
| FI-08 | Input Command Production | ORIGINAL_REQUEST §R1, §AC | ≥5 | ≥5 | ✓ |
| FI-09 | Elemental Combat Effectiveness | Reference `src/combat.rs` | ≥5 | ≥5 | ✓ |
| FI-10 | Action Economy & Stats | Reference `src/components.rs`| ≥5 | ≥5 | ✓ |
| FI-11 | Loopback Transport | ORIGINAL_REQUEST §R1 | ≥5 | ≥5 | ✓ |
| FI-12 | Ebitengine Rendering & Transforms | ORIGINAL_REQUEST §R2 | ≥5 | ≥5 | ✓ |
| FI-13 | Application & Game Assembly | ORIGINAL_REQUEST §AC | ≥5 | ≥5 | ✓ |

## Test Architecture
- **Location**: `/home/jack/teamwork_projects/wildbuds_go/test/e2e/`
- **Invocation**: `go test -v ./test/e2e/...`
- **Pass/Fail Semantics**: Standard Go `testing` framework; zero panics, all assertions pass.

## Tier Breakdown & Coverage Goals
- **Tier 1 - Feature Coverage (≥5 per feature)**:
  - Minimum: 13 features × 5 = 65 tests.
  - Covers happy-path behavior, valid commands, state transitions in isolation.
- **Tier 2 - Boundary & Corner Cases (≥5 per feature)**:
  - Minimum: 13 features × 5 = 65 tests.
  - Covers invalid turns (wrong player), out-of-bounds destinations, selecting enemy units, selecting empty tiles, acting without selected unit, extreme coordinates, multiple activations, zero/empty queues.
- **Tier 3 - Cross-Feature Combinations (Pairwise Coverage)**:
  - Minimum: ≥15 interaction tests.
  - Covers sequential command sequences (Select ➔ Move ➔ EndActivation ➔ Select ➔ Move), rapid phase switches, loopback round-trip latency, transport disconnection/reconnection.
- **Tier 4 - Real-World Gameplay Scenarios**:
  - Minimum: ≥7 multi-turn realistic gameplay simulations.
  - Full match scenarios: opening moves, tactical repositioning, turn handoffs across 10+ turns, state verification after extended play.
- **Tier 5 - Adversarial Coverage Hardening**:
  - White-box branch/edge coverage and fuzz stress tests led by Challengers in Milestone 6.
