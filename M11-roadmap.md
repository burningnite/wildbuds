# M11: Dynamic Board & Terrain Roadmap
*Status: PLANNED*

## 1. Dynamic Grid Sizing
- [ ] Refactor `internal/domain/types.go` to remove hardcoded `GridMin`, `GridMax`, and `GridSize` constants.
- [ ] Introduce a `BoardConfig` struct to dynamically define grid boundaries and dimensions (e.g., NxM rectangular grids, or offset shapes).
- [ ] Update `InBounds()` and invariant validations across the `Resolver` and `GameState` to query the dynamic `BoardConfig`.
- [ ] Ensure `Camera` and `InputHandler` correctly adapt coordinate limits and screen centering based on the dynamic size.

## 2. Terrain System
- [ ] Define `TerrainType` enum (e.g., Grass, Water, Lava, HighGround).
- [ ] Update `GameState` to include a 2D array or map of `TerrainType` for each `GridPosition`.
- [ ] Implement environmental combat/movement modifiers (e.g., Water element units gain defense on Water terrain, Lava damages non-Fire units).
- [ ] Update `Resolver` to factor in terrain modifiers during movement validation and combat calculations.

## 3. Dynamic Rendering
- [ ] Update `internal/render/board.go` to query the `TerrainType` map instead of simple checkerboard parity.
- [ ] Map distinct visual colors/textures (or 3D Tetra assets) to each `TerrainType`.
- [ ] Support rendering irregularly shaped boards (ignoring or omitting out-of-bounds void tiles).
