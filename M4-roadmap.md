# M4: Ebitengine Renderer & Input Handler Roadmap
*Status: COMPLETED*

## 1. Camera & Coordinate Systems
- [ ] Create `internal/render/camera.go`.
- [ ] Define `Camera` struct with OriginX, OriginY, and Zoom/Scale parameters.
- [ ] Implement `GridToScreen(gx, gy int) (float64, float64)`. Ensure the mathematical inversion for +Y UP logic is correct (`screenY = OriginY - gy * Pitch`).
- [ ] Implement `ScreenToGrid(sx, sy float64) (int, int)` with accurate rounding.
- [ ] Write `camera_test.go` asserting bijective mapping between Grid and Screen coordinates.

## 2. Basic Rendering
- [ ] Create `internal/render/colors.go` and define the game's color palette (Tiles, Players, UI).
- [ ] Create `internal/render/board.go`.
- [ ] Implement a function to draw the 11x11 checkerboard grid using `ebitenutil.DrawRect`.
- [ ] Implement a function to iterate over `GameState.Units` and draw simple colored rectangles representing units based on their `GridPosition`.
- [ ] Implement rendering for the local player's cursor on the currently highlighted tile.

## 3. Input Handling
- [ ] Create `internal/input/input.go`.
- [ ] Define `InputHandler` struct to track local cursor state (GridX, GridY).
- [ ] Implement `Update()` function.
- [ ] Poll `ebiten.IsKeyPressed` for WASD / Arrow keys.
- [ ] Implement a debouncing/cooldown mechanism to prevent rapid cursor flying.
- [ ] Map cursor movement to update local internal coordinates (bounded to [-5, 5]).
- [ ] Poll `ebiten.IsMouseButtonPressed` and use `Camera.ScreenToGrid` to set cursor position on click.

## 4. Input-to-Command Mapping
- [ ] In `InputHandler.Update()`, poll for the 'Confirm' key (Space/Enter).
- [ ] If 'Confirm' is pressed during `PhaseSelectUnit`, enqueue a `SelectUnit` `GameCommand` to the `CommandQueue.Outgoing` channel targeting the cursor's coordinate.
- [ ] If 'Confirm' is pressed during `PhaseChooseAction`, enqueue a `MoveUnit` or `Attack` `GameCommand` based on contextual UI state.
- [ ] Poll for the 'End' key ('E'). If pressed, enqueue an `EndActivation` `GameCommand`.
- [ ] *CRITICAL VERIFICATION*: Ensure `InputHandler` *never* imports or directly mutates `domain.GameState`.
