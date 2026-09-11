# Wildbuds Go Port - Development Roadmap

This document outlines the highly atomized, incremental roadmap for the Wildbuds Go port. It is designed to be consumed by LLM agents (e.g., via Antigravity CLI) to drive autonomous development. Each task represents a small, verifiable step.

## Phase 1: Core Systems (M1 - M2)
*Status: COMPLETED*

- [x] **M1: Core Domain & State Model**
  - [x] Define board constants and grid specifications.
  - [x] Implement elemental combat matrix and `DualType` support.
  - [x] Define `ActionTokens` and `BaseStats` for units.
  - [x] Implement initial state constructor.
- [x] **M2: Command Pipeline & Authoritative Resolver**
  - [x] Implement `CommandQueue` separating outgoing and incoming commands.
  - [x] Implement `SelectUnit` command and validation logic.
  - [x] Implement `MoveUnit` command and validation logic.
  - [x] Implement `Attack` command and validation logic.
  - [x] Implement `EndActivation` command and validation logic.
  - [x] Implement deterministic state machine transitions.

## Phase 2: Application Assembly (M3 - M6)
*Status: IN PROGRESS*

### M3: Transport & Local Loopback
*Currently Blocked: Auditor Integrity Violation (Iteration 2 required)*
- [ ] Investigate and resolve the "Auditor Integrity Violation" from Iteration 1.
- [ ] Define `Transport` interface (`Send`, `Receive`, `Close`) in `internal/transport/transport.go`.
- [ ] Implement `LoopbackTransport` struct.
- [ ] Implement `LoopbackTransport.Send()` to stamp outgoing `GameCommand` with the local player ID.
- [ ] Route the stamped command to the incoming `NetworkCommand` queue/channel.
- [ ] Write unit tests for `LoopbackTransport` to ensure no data corruption during loopback.

### M4: Ebitengine Renderer & Input Handler
- [ ] Implement `render.Camera` struct in `internal/render/camera.go`.
- [ ] Implement `GridToScreen` coordinate mapping, ensuring +Y is UP.
- [ ] Implement `ScreenToGrid` coordinate mapping.
- [ ] Write unit tests for `GridToScreen` and `ScreenToGrid` transformations.
- [ ] Implement basic checkerboard grid rendering in `internal/render/board.go`.
- [ ] Implement basic unit rendering (e.g., colored rectangles) using `internal/render/colors.go`.
- [ ] Implement cursor rendering to highlight the currently selected grid tile.
- [ ] Implement `input.InputHandler` struct in `internal/input/input.go`.
- [ ] Map WASD/Arrow keys to cursor motion.
- [ ] Map Mouse clicks to update grid selection.
- [ ] Map Space/Enter to confirm actions and 'E' to end activation.
- [ ] Ensure `InputHandler` enqueues commands to `CommandQueue.Outgoing` strictly without mutating `GameState`.

### M5: Application Integration & Game Loop
- [ ] Define the `ebiten.Game` interface implementation in `internal/app/game.go`.
- [ ] Implement `Game.Update()` to poll input, process the transport loopback, and invoke the `Resolver`.
- [ ] Implement `Game.Draw()` to pass the authoritative state to the renderer.
- [ ] Implement `Game.Layout()` to handle screen sizing and scaling.
- [ ] Create the main entry point in `cmd/wildbuds/main.go`.
- [ ] Initialize window properties (title, dimensions) and start `ebiten.RunGame`.
- [ ] Verify local build via `go build ./cmd/wildbuds`.

### M6: E2E Verification & Adversarial Hardening
- [ ] Scaffold E2E test runner in `test/e2e/harness_test.go`.
- [ ] Write and pass Tier 1 (Features) tests in `test/e2e/tier1_features_test.go`.
- [ ] Write and pass Tier 2 (Boundaries) tests in `test/e2e/tier2_boundaries_test.go`.
- [ ] Write and pass Tier 3 (Combinations) tests in `test/e2e/tier3_combinations_test.go`.
- [ ] Write and pass Tier 4 (Workloads) tests in `test/e2e/tier4_workloads_test.go`.
- [ ] Fix any functional bugs discovered during Tiers 1-4.
- [ ] Implement Tier 5 (Adversarial Hardening) tests (fuzzing inputs, invalid state injections).
- [ ] Patch vulnerabilities found during Tier 5 testing.

## Phase 3: Future Enhancements (M7+)
*Status: PLANNED*

### M7: P2P Networking (WebRTC)
- [ ] Integrate `matchmaker-go` dependency into `go.mod`.
- [ ] Implement `WebRTCTransport` struct adhering to the `Transport` interface.
- [ ] Implement serialization logic (JSON or Protobuf) for `GameCommand` and `NetworkCommand`.
- [ ] Implement a basic matchmaking lobby or connection UI via Ebitengine.
- [ ] Implement connection establishment and peer discovery.
- [ ] Synchronize the initial deterministic RNG seed and board state between peers.
- [ ] Implement connection loss detection and graceful timeouts.
- [ ] Implement basic state-resync or re-connection handling.

### M8: Asset & Sprite Integration
- [ ] Create an asset loading manager in `internal/render/assets.go` (images, fonts).
- [ ] Load and slice sprite sheets for game units.
- [ ] Replace primitive shape unit rendering with static sprite rendering.
- [ ] Implement a simple animation controller for units (Idle, Move, Attack states).
- [ ] Replace basic checkerboard rendering with textured tilemaps.
- [ ] Implement particle effects or visual indicators for elemental combat effectiveness (e.g., "Super Effective!").

### M9: User Interface & User Experience
- [ ] Implement a Turn & Phase announcement banner (e.g., "Player One's Turn").
- [ ] Implement an on-screen Action Menu when a unit is selected (Move, Attack, Special, End).
- [ ] Implement a Unit Info Panel displaying Stats, Dual-Typing, and remaining Action Tokens when hovered.
- [ ] Implement a Combat Log or floating text to display damage numbers.

### M10: Audio System
- [ ] Integrate Ebitengine's `audio` package.
- [ ] Create an audio playback manager for SFX and Music.
- [ ] Add sound effects for UI interaction (hover, click, error).
- [ ] Add sound effects for unit actions (movement, specific elemental attacks).
- [ ] Implement background music streaming with looping.
