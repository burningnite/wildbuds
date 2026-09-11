# M5b: Application Scenes & Start Menu Roadmap
*Status: COMPLETED*

## 1. Scene Management Architecture
- [ ] Create `internal/app/scene.go`.
- [ ] Define a `Scene` interface with `Update() error` and `Draw(screen *ebiten.Image)` methods.
- [ ] Refactor `internal/app/game.go` to use a State Machine that manages the active `Scene`.
- [ ] Move the current gameplay logic (Resolver, CommandQueue, rendering) into a `BattleScene` struct that implements the `Scene` interface.

## 2. Main Menu Implementation
- [ ] Create `internal/ui/start_menu.go`.
- [ ] Implement a `StartMenuScene` that implements the `Scene` interface.
- [ ] Render dummy buttons for: "Spar", "Duel", "Team Building", "Profile", "Options", "Debugger", and "Quit".
- [ ] Implement mouse input detection for hovering and clicking the buttons.

## 3. Scene Routing & Debugger Hook
- [ ] Implement routing for the "Quit" button to cleanly exit the application.
- [ ] Implement routing for the "Debugger" button to transition to a `DebuggerScene`.
- [ ] Create `internal/ui/debugger.go` containing the `DebuggerScene`.
- [ ] Implement a submenu in the `DebuggerScene` for executing and visually observing any/all render-control tests.
- [ ] Route the "Spar" or "Duel" buttons to instantiate and transition to the `BattleScene` for actual gameplay.
