# M7: P2P Networking (WebRTC) Roadmap
*Status: PLANNED*

## 1. Dependencies & Boilerplate
- [ ] Add `github.com/pion/webrtc/v3` or the specific `matchmaker-go` dependency to `go.mod`.
- [ ] Define the `WebRTCTransport` struct in `internal/transport/webrtc.go`.
- [ ] Ensure `WebRTCTransport` fully implements the `transport.Transport` interface.

## 2. Serialization Layer
- [ ] Define serialization interfaces.
- [ ] Implement robust JSON or Protobuf marshaling for `domain.GameCommand`.
- [ ] Implement unmarshaling for `domain.NetworkCommand` to ensure network payloads can be converted back to strict domain types.
- [ ] Write tests verifying serialization/deserialization symmetry.

## 3. Connection & Matchmaking UI
- [ ] Create a rudimentary Lobby State in Ebitengine.
- [ ] Add UI text inputs for entering a Lobby ID / Match Code.
- [ ] Implement a "Host Match" flow that generates a WebRTC offer.
- [ ] Implement a "Join Match" flow that processes an offer and generates an answer.
- [ ] (Optional based on library) Integrate `matchmaker-go` signaling server logic to exchange ICE candidates and SDPs automatically.

## 4. State Synchronization & Lifecycle
- [ ] Establish the WebRTC DataChannel.
- [ ] Implement a handshake protocol confirming both peers are connected.
- [ ] Synchronize and seed the deterministic RNG on both clients using a shared timestamp or host-provided seed.
- [ ] Ensure initial `GameState` is perfectly identical.
- [ ] Implement connection health checks (ping/pong or WebRTC internal mechanisms).
- [ ] Handle graceful disconnects and timeout logic in the game loop.
