# M7: P2P Networking (go-libp2p) Roadmap
*Status: COMPLETED*

## 1. Dependencies & Boilerplate
- [x] Add `github.com/libp2p/go-libp2p` and its required dependencies to `go.mod`.
- [x] Define the `Libp2pTransport` struct in `internal/transport/libp2p.go`.
- [x] Ensure `Libp2pTransport` fully implements the `transport.Transport` interface.

## 2. Serialization Layer
- [x] Implement JSON or Gob marshaling for `domain.GameCommand` so it can be sent over a libp2p stream.
- [x] Implement unmarshaling for incoming streams to generate `domain.NetworkCommand`.
- [x] Write tests verifying serialization symmetry.

## 3. Peer Discovery & Connection
- [x] Implement an mDNS (Multicast DNS) discovery service so local peers can automatically find each other without a signaling server.
- [x] (Optional) Provide a manual Multiaddr input fallback.
- [x] Establish a persistent libp2p stream between two peers for exchanging game commands.
- [x] Determine Player One vs. Player Two based on Peer ID lexicographical sorting (or host vs guest logic) to prevent race conditions during initialization.

## 4. Lobby UI Integration
- [x] Create `internal/app/lobby_scene.go`.
- [x] Render a "Searching for peers..." UI or a manual connect screen.
- [x] Upon successful libp2p connection and handshake, transition seamlessly to `BattleScene`.

## 5. State Synchronization & Lifecycle
- [x] Synchronize and seed the deterministic RNG on both clients based on a shared value (e.g., XOR of both peer IDs).
- [x] Handle graceful disconnects, routing back to the Start Menu or Lobby on stream failure.
