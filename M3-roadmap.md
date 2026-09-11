# M3: Transport & Local Loopback Roadmap
*Status: COMPLETED*

## 1. Audit & Resolution
- [ ] Investigate the "Auditor Integrity Violation" blocking M3.
- [ ] Determine if the violation is in the state model, command pipeline, or transport design.
- [ ] Implement fix for the integrity violation.
- [ ] Re-run local validation tests to clear the block.

## 2. Transport Interface
- [ ] Create `internal/transport/transport.go`.
- [ ] Define the `Transport` interface.
- [ ] Add method `Send(cmd domain.GameCommand) error`.
- [ ] Add method `Receive() <-chan domain.NetworkCommand`.
- [ ] Add method `Close() error`.

## 3. Loopback Implementation
- [ ] Create `internal/transport/loopback.go`.
- [ ] Define `LoopbackTransport` struct implementing `Transport`.
- [ ] Initialize internal Go channels for loopback routing.
- [ ] Implement `Send()`: Wrap the incoming `GameCommand` in a `NetworkCommand`.
- [ ] In `Send()`, explicitly stamp the `Sender` field with the local `PlayerID`.
- [ ] Push the constructed `NetworkCommand` to the internal channel.
- [ ] Implement `Receive()`: Return the read-only channel containing stamped `NetworkCommands`.

## 4. Testing & Verification
- [ ] Write `loopback_test.go`.
- [ ] Test that a sent command is immediately available on the receive channel.
- [ ] Assert that the `Sender` field is accurately stamped and not mutated during transit.
- [ ] Verify thread safety/channel blocking behavior.
