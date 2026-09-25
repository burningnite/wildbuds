## 2024-05-24 - [Denial of Service via Unbounded Network Read]
**Vulnerability:** In `internal/transport/libp2p.go`, `readLoop` used `bufio.NewReader(stream).ReadBytes('\\n')` to read commands from peer streams. This unbounded read would continue buffering data indefinitely if an attacker sent an endless stream of bytes without a newline character, eventually crashing the application due to out-of-memory (OOM).
**Learning:** Never trust input from network streams, especially in peer-to-peer contexts without a centralized authority. Functions that read until a delimiter must always have a strict length limit to prevent DoS attacks via resource exhaustion.
**Prevention:** Use `bufio.NewScanner` combined with `Scanner.Buffer` to enforce a hard upper bound on token sizes when reading delimited payloads from the network. If the token exceeds this bound, the scanner will safely fail rather than consuming infinite memory.

## 2026-09-25 - [Authorization Bypass via Identity Spoofing]
**Vulnerability:** The P2P transport layer (`readLoop` in `internal/transport/libp2p.go`) trusted the `Sender` field from the incoming JSON payload. A malicious peer could forge a `NetworkCommand` with the victim's player ID, effectively hijacking the victim's units and taking turns on their behalf.
**Learning:** Never trust identity or authorization claims from client payloads in a peer-to-peer network. Identity must be securely established at the transport/connection level.
**Prevention:** Always forcefully overwrite the `Sender` field of incoming commands with the securely authenticated identity of the peer on that connection, completely ignoring any identity claimed in the payload itself.
