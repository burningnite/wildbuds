# SYSTEM DIRECTIVE: PROJECT INITIALIZATION

## TARGET STACK
- Language: Rust
- Engine: Bevy (ECS)
- Networking: P2P (Matchbox / WebRTC)
- Sync: Deterministic Lockstep
- UI: Native (Zero Webcode)

## CORE MECHANICS
- Map: Grid-based
- Action Economy: Maleghast (Discrete phases/tokens)
- Combat: Pokemon (Elemental matrix, base stats)

## IMMEDIATE TASKS
1. Parse stack and mechanics.
2. Halt execution. Do not generate source code.
3. Output a sequential list of technical questions to finalize architecture before `cargo new`.

## REQUIRED USER DECISIONS
- Grid topology: Square or Hex?
- Display target: Bevy 2D Sprites or Terminal UI (Ratatui)?
- P2P signaling: Public Matchbox relay or custom lightweight local/self-hosted server?
- Element complexity: Single vs Dual typing?
- Input handling: Mouse-driven raycasting vs keyboard grid-snapping?
