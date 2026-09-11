# M5: Application Integration & Game Loop Roadmap
*Status: COMPLETED*

## 1. Game Struct Definition
- [ ] Create `internal/app/game.go`.
- [ ] Define the main `Game` struct that implements the `ebiten.Game` interface.
- [ ] Add fields to `Game` struct: `state *domain.GameState`, `queue *domain.CommandQueue`, `transport transport.Transport`, `renderer *render.Renderer`, `input *input.InputHandler`, and `resolver *resolver.Resolver`.

## 2. Update Loop Implementation
- [ ] Implement `Game.Update() error`.
- [ ] Call `Game.input.Update(...)` to process user input and populate the outgoing queue.
- [ ] Drain the `CommandQueue.Outgoing` channel and push commands into `Game.transport.Send()`.
- [ ] Poll `Game.transport.Receive()` for incoming network commands.
- [ ] Push received `NetworkCommands` into `CommandQueue.Incoming`.
- [ ] Call `Game.resolver.Resolve(Game.state, Game.queue)` to process the incoming queue and mutate authoritative state deterministically.

## 3. Draw & Layout Implementation
- [ ] Implement `Game.Draw(screen *ebiten.Image)`.
- [ ] Clear the screen with a background color.
- [ ] Pass the authoritative `Game.state` and `screen` reference to the rendering subsystem (`board.Draw`, `units.Draw`, `cursor.Draw`).
- [ ] Implement `Game.Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int)`.
- [ ] Define fixed logical resolution (e.g., 800x600) and return it.

## 4. Main Entry Point
- [ ] Modify/Create `cmd/wildbuds/main.go`.
- [ ] Initialize the `GameState` via domain constructors.
- [ ] Initialize `CommandQueue` and `LoopbackTransport`.
- [ ] Initialize `Resolver`, `InputHandler`, and `Camera/Renderer`.
- [ ] Assemble the `app.Game` struct.
- [ ] Call `ebiten.SetWindowSize()` and `ebiten.SetWindowTitle("Wildbuds")`.
- [ ] Start the game loop via `ebiten.RunGame()`.
- [ ] Verify successful compilation with `go build ./cmd/wildbuds`.
