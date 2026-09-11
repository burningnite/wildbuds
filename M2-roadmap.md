# M2: Command Pipeline & Authoritative Resolver Roadmap
*Status: COMPLETED*

## 1. Command Definitions
- [x] Define `commands/command.go`.
- [x] Implement `CommandType` enum (`SelectUnit`, `MoveUnit`, `Attack`, `EndActivation`).
- [x] Implement `GameCommand` struct with Type, Target ID/Position, and Destination Position.
- [x] Implement `NetworkCommand` struct wrapping `Sender Player` and `Command GameCommand`.

## 2. Command Queue
- [x] Define `commands/queue.go`.
- [x] Implement `CommandQueue` struct.
- [x] Add an outgoing channel/queue for local input.
- [x] Add an incoming channel/queue for network/loopback receiving.

## 3. Authoritative Resolver Core
- [x] Define `resolver/resolver.go`.
- [x] Implement `Resolver` interface.
- [x] Create `Resolve(state *domain.GameState, queue *domain.CommandQueue) []domain.Event` method.
- [x] Implement base validation: Ensure `cmd.Sender == state.ActivePlayer`.

## 4. Command Handlers & State Transitions
- [x] Implement `handleSelectUnit`: Validate unit ownership, existence, and transition to `PhaseChooseAction`.
- [x] Implement `handleMoveUnit`: Validate active unit, check grid bounds, update unit `GridPosition`, consume movement token.
- [x] Implement `handleAttack`: Validate active unit, calculate damage via elemental matrix, deduct HP, consume attack token.
- [x] Implement `handleEndActivation`: Clear active unit, set `ActivePlayer = ActivePlayer.Next()`, transition to `PhaseSelectUnit`, refresh tokens.
- [x] Write `resolver_test.go` confirming invalid commands produce zero state mutations.
