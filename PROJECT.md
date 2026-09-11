# Project: Wildbuds Go Port

## Architecture
Wildbuds is a turn-based tactical grid strategy game ported from Rust/Bevy to Go/Ebitengine. The architecture enforces a strict unidirectional, command-first deterministic pipeline:
$$\text{Input Events} \longrightarrow \text{CommandQueue (Outgoing)} \longrightarrow \text{Transport (Loopback)} \longrightarrow \text{CommandQueue (Incoming)} \longrightarrow \text{Authoritative Resolver} \longrightarrow \text{Authoritative State Mutation}$$

### Core Design Principles
1. **Input Immutability**: The input layer only updates the local client cursor and emits `GameCommand` values to the outgoing queue. It is strictly prohibited from mutating `GameState`, `TurnPhase`, or unit attributes directly.
2. **Authoritative Resolution**: Only the `Resolver` processes incoming `NetworkCommand`s sequentially. Commands are validated against turn ownership and phase preconditions. Rejected commands produce zero state mutations.
3. **Deterministic State Machine**: State at step $T$ is purely a function of initial state $S_0$ and the sequence of resolved commands.
4. **Transport Abstraction**: Outgoing commands pass through a `Transport` interface. For local play, `LoopbackTransport` stamps commands with the local player ID and enqueues them into the incoming queue. The boundary is designed to allow a drop-in replacement for future WebRTC peer connections via `matchmaker-go`.
5. **View Observation**: The Ebitengine renderer strictly observes authoritative state to project units and board coordinates into screen space.

---

## Feature Inventory
Every feature extracted during the Survey phase is enumerated here with its assigned milestone.

| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| FI-01 | Command Pipeline Architecture | CommandQueue separating outgoing and incoming commands | M2 | `src/commands.rs`, `ORIGINAL_REQUEST §R1` |
| FI-02 | Alternating Turn & Phase State Machine | Two-player alternation (`PlayerOne`, `PlayerTwo`), two phases (`SelectUnit`, `ChooseAction`) | M1 | `src/state.rs`, `src/game.rs` |
| FI-03 | Command: SelectUnit | Validates sender turn, unit existence, and ownership; transitions to `ChooseAction` | M2 | `src/game.rs:31-44` |
| FI-04 | Command: MoveUnit | Validates sender turn and active unit presence; updates unit `GridPosition` | M2 | `src/game.rs:45-56` |
| FI-05 | Command: Attack | Validates sender turn and active unit presence; logs attack intent / combat hook | M2 | `src/game.rs:57-64` |
| FI-06 | Command: EndActivation | Clears active unit, toggles active player to next, resets phase to `SelectUnit` | M2 | `src/game.rs:65-71` |
| FI-07 | Board Geometry & Grid Specifications | 11x11 grid centered at (0,0), range `[-5, 5]`, 32px pitch, 30px tile, initial units at (0,-3) and (0,3) | M1 | `src/render.rs:15-82` |
| FI-08 | Input Mapping & Cursor Navigation | WASD/Arrows for cursor motion, mouse click to grid, Space/Enter confirm, 'E' end activation; input never mutates state | M4 | `src/input.rs:20-73`, `ORIGINAL_REQUEST §R1` |
| FI-09 | Elemental Combat & Typing Matrix | 18 Elements, `DualType` support, damage effectiveness multipliers (own=0.5, Water>Fire=2.0, Fire>Grass=2.0, Grass>Water=2.0, default=1.0) | M1 | `src/combat.rs:4-63` |
| FI-10 | Tactical Action Economy & Base Stats | `ActionTokens` (movement, attack, special) and `BaseStats` (hp, attack, defense, speed) | M1 | `src/components.rs:20-39` |
| FI-11 | Transport & Local Loopback Pipeline | `Transport` interface; loopback outgoing commands to incoming queue stamped with player ID | M3 | `src/network.rs:95-98`, `ORIGINAL_REQUEST §R1` |
| FI-12 | Ebitengine Rendering System | Screen-to-grid and grid-to-screen coordinate mapping (inverting Y for +Y UP), checkerboard rendering, unit & cursor rendering | M4 | `src/render.rs`, `ORIGINAL_REQUEST §R2` |
| FI-13 | Application Integration & Game Loop | `ebiten.Game` implementation (`Update`, `Draw`, `Layout`), window setup, `main.go` entry point | M5 | `src/main.rs`, `ORIGINAL_REQUEST §Acceptance Criteria` |

---

## Milestones

| # | Name | Scope | Dependencies | Status |
|---|------|-------|--------------|--------|
| E2E | E2E Testing Track | Independent opaque-box test runner & test suites (Tiers 1-4) | none | PLANNED |
| M1 | Core Domain & State Model | Types, board constants, elemental combat matrix, initial state constructor | none | DONE |
| M2 | Command Pipeline & Authoritative Resolver | `CommandQueue`, `Resolver.Resolve`, validation rules, deterministic state transitions | M1 | DONE |
| M3 | Transport & Local Loopback | `Transport` interface, `LoopbackTransport` routing outgoing to incoming | M2 | BLOCKED: Auditor Integrity Violation (Iteration 2 required) |
| M4 | Ebitengine Renderer & Input Handler | `Camera` (+Y UP coordinate math), tile/unit/cursor rendering, `InputHandler` producing commands | M1, M2 | PLANNED |
| M5 | Application Integration & Game Loop | `ebiten.Game` assembly (`Update`, `Draw`, `Layout`), CLI `main.go`, `go build` | M3, M4 | PLANNED |
| M6 | Final Milestone: E2E Verification & Adversarial Hardening | Pass 100% of E2E tests (Tiers 1-4) followed by Tier 5 adversarial hardening | E2E, M5 | PLANNED |

---

## Interface Contracts

### Domain ↔ Resolver (`internal/domain` ↔ `internal/resolver`)
- `domain.GameState`: Contains `Turn State` (`ActivePlayer`, `ActiveUnitID`, `Phase`), `Units []*Unit`.
- `domain.GameCommand`: Contains `Type CommandType`, `Target GridPosition`, `Destination GridPosition`.
- `domain.NetworkCommand`: Contains `Sender Player`, `Command GameCommand`.
- `resolver.Resolver`: Method `Resolve(state *domain.GameState, queue *domain.CommandQueue) []domain.Event`
  - Validates `cmd.Sender == state.ActivePlayer`.
  - On validation failure: state is unchanged, rejection logged.
  - On valid `SelectUnit`: `state.ActiveUnitID = &unit.ID`, `state.Phase = domain.PhaseChooseAction`.
  - On valid `MoveUnit`: unit position updated to destination.
  - On valid `EndActivation`: `state.ActiveUnitID = nil`, `state.ActivePlayer = state.ActivePlayer.Next()`, `state.Phase = domain.PhaseSelectUnit`.

### Commands ↔ Transport (`internal/commands` ↔ `internal/transport`)
- `transport.Transport`:
  - `Send(cmd domain.GameCommand) error`
  - `Receive() <-chan domain.NetworkCommand`
  - `Close() error`
- `transport.LoopbackTransport`: Directly wraps `GameCommand` in `NetworkCommand{Sender: localPlayer}` and pushes to incoming channel/queue.

### Camera ↔ Renderer & Input (`internal/render` ↔ `internal/input`)
- `render.Camera`:
  - `GridToScreen(gx, gy int) (float64, float64)`: $\text{screenX} = \text{OriginX} + gx \times 32.0$, $\text{screenY} = \text{OriginY} - gy \times 32.0$.
  - `ScreenToGrid(sx, sy float64) (int, int)`: $\text{gridX} = \text{round}((\text{sx} - \text{OriginX}) / 32.0)$, $\text{gridY} = \text{round}((\text{OriginY} - \text{sy}) / 32.0)$.
- `input.InputHandler`:
  - `Update(cam *render.Camera, phase domain.TurnPhase, q *domain.CommandQueue)`: Reads keyboard/mouse, updates local cursor, enqueues commands to `q.Outgoing`. Never touches `domain.GameState`.

---

## Code Layout
```
/home/jack/teamwork_projects/wildbuds_go/
├── cmd/
│   └── wildbuds/
│       └── main.go
├── internal/
│   ├── domain/
│   │   ├── types.go
│   │   ├── stats.go
│   │   ├── combat.go
│   │   ├── state.go
│   │   └── combat_test.go
│   ├── commands/
│   │   ├── command.go
│   │   └── queue.go
│   ├── resolver/
│   │   ├── resolver.go
│   │   └── resolver_test.go
│   ├── transport/
│   │   ├── transport.go
│   │   └── loopback.go
│   ├── render/
│   │   ├── camera.go
│   │   ├── camera_test.go
│   │   ├── colors.go
│   │   └── board.go
│   ├── input/
│   │   ├── input.go
│   │   └── input_test.go
│   └── app/
│       └── game.go
├── test/
│   └── e2e/
│       ├── harness_test.go
│       ├── tier1_features_test.go
│       ├── tier2_boundaries_test.go
│       ├── tier3_combinations_test.go
│       └── tier4_workloads_test.go
├── go.mod
├── PROJECT.md
└── TEST_INFRA.md
```
