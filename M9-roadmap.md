# M9: User Interface & User Experience Roadmap
*Status: PLANNED*

## 1. HUD & Turn Management
- [ ] Create `internal/ui/hud.go`.
- [ ] Implement a Top Banner displaying the current Turn state (e.g., "Player One's Turn").
- [ ] Color-code the banner based on the active player.
- [ ] Implement a clear Phase Indicator ("Select Unit" vs "Choose Action").

## 2. Unit Context Panel
- [ ] Create an informational side/bottom panel.
- [ ] When a unit is hovered or selected, display its portrait/sprite.
- [ ] Render the unit's `BaseStats` (HP, Atk, Def, Spd).
- [ ] Render the unit's `DualType` icons.
- [ ] Visually represent remaining `ActionTokens` (e.g., grayed out icons if already moved/attacked).

## 3. Action Menu
- [ ] Create `internal/ui/menu.go`.
- [ ] When a unit is selected (transitioning to `PhaseChooseAction`), pop up a contextual Action Menu near the unit.
- [ ] Menu options: "Move", "Attack", "Special", "End Activation".
- [ ] Disable options based on available `ActionTokens`.
- [ ] Integrate menu selection with the `InputHandler` to trigger appropriate `GameCommands`.

## 4. Feedback & Combat Log
- [ ] Implement a sliding Combat Log panel on the screen edge.
- [ ] Whenever `Resolver` emits a combat event, append formatted text to the log.
- [ ] (e.g., "WaterUnit hit FireUnit for 24 damage. It's super effective!").
- [ ] Add visual polish: ensure UI elements scale correctly with `Game.Layout` resolution changes.
