package input

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	
	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
	"wildbuds/internal/render"
)

// InputHandler manages local cursor state and translates user input into GameCommands.
type InputHandler struct {
	CursorX int
	CursorY int
	
	queue     *commands.CommandQueue
	camera    *render.Camera
	
	// Cooldown for keyboard movement
	lastMove  time.Time
	moveDelay time.Duration
}

// NewInputHandler creates an InputHandler connected to the given CommandQueue.
func NewInputHandler(queue *commands.CommandQueue, camera *render.Camera) *InputHandler {
	return &InputHandler{
		CursorX:   0,
		CursorY:   0,
		queue:     queue,
		camera:    camera,
		moveDelay: 150 * time.Millisecond,
	}
}

// Update polls Ebitengine input state and queues commands if actions are triggered.
// It explicitly requires the current GameState to know which commands are valid contextually,
// but it NEVER mutates the game state itself.
func (h *InputHandler) Update(state *domain.GameState) {
	h.handleMovement()
	h.handleMouse()
	h.handleActions(state)
}

func (h *InputHandler) handleMovement() {
	now := time.Now()
	
	// Key presses with repeat (allows holding down to move, but regulated by delay)
	dx, dy := 0, 0
	
	// Note: +Y is UP in our grid, so W/Up arrow increases Y.
	// We use standard inpututil for just pressed to ensure snappy single-taps,
	// and fallback to continuous check for holding.
	if inpututil.IsKeyJustPressed(ebiten.KeyW) || inpututil.IsKeyJustPressed(ebiten.KeyUp) ||
		(ebiten.IsKeyPressed(ebiten.KeyW) && now.Sub(h.lastMove) > h.moveDelay) ||
		(ebiten.IsKeyPressed(ebiten.KeyUp) && now.Sub(h.lastMove) > h.moveDelay) {
		dy = 1
	} else if inpututil.IsKeyJustPressed(ebiten.KeyS) || inpututil.IsKeyJustPressed(ebiten.KeyDown) ||
		(ebiten.IsKeyPressed(ebiten.KeyS) && now.Sub(h.lastMove) > h.moveDelay) ||
		(ebiten.IsKeyPressed(ebiten.KeyDown) && now.Sub(h.lastMove) > h.moveDelay) {
		dy = -1
	}
	
	if inpututil.IsKeyJustPressed(ebiten.KeyD) || inpututil.IsKeyJustPressed(ebiten.KeyRight) ||
		(ebiten.IsKeyPressed(ebiten.KeyD) && now.Sub(h.lastMove) > h.moveDelay) ||
		(ebiten.IsKeyPressed(ebiten.KeyRight) && now.Sub(h.lastMove) > h.moveDelay) {
		dx = 1
	} else if inpututil.IsKeyJustPressed(ebiten.KeyA) || inpututil.IsKeyJustPressed(ebiten.KeyLeft) ||
		(ebiten.IsKeyPressed(ebiten.KeyA) && now.Sub(h.lastMove) > h.moveDelay) ||
		(ebiten.IsKeyPressed(ebiten.KeyLeft) && now.Sub(h.lastMove) > h.moveDelay) {
		dx = -1
	}

	if dx != 0 || dy != 0 {
		h.CursorX += dx
		h.CursorY += dy
		
		// Clamp to board bounds
		if h.CursorX < domain.GridMin { h.CursorX = domain.GridMin }
		if h.CursorX > domain.GridMax { h.CursorX = domain.GridMax }
		if h.CursorY < domain.GridMin { h.CursorY = domain.GridMin }
		if h.CursorY > domain.GridMax { h.CursorY = domain.GridMax }
		
		h.lastMove = now
	}
}

func (h *InputHandler) handleMouse() {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		gx, gy := h.camera.ScreenToGrid(float64(mx), float64(my))
		
		if domain.InGridBounds(domain.GridPosition{X: gx, Y: gy}) {
			h.CursorX = gx
			h.CursorY = gy
		}
	}
}

func (h *InputHandler) handleActions(state *domain.GameState) {
	// 'Confirm' action (Space or Enter, or double click could be added later)
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		target := domain.GridPosition{X: h.CursorX, Y: h.CursorY}
		
		if state != nil && state.Phase == domain.PhaseSelectUnit {
			// Enqueue SelectUnit command
			h.queue.PushOutgoing(commands.NewSelectUnitCommand(target))
		} else if state != nil && state.Phase == domain.PhaseChooseAction {
			// In a real UI, this would depend on which action button is highlighted.
			// For now, assume Confirm means Move if no enemy, Attack if enemy.
			// The resolver handles validation, we just enqueue the command.
			
			// Currently defaulting to MoveUnit as the primary action for testing.
			h.queue.PushOutgoing(commands.NewMoveUnitCommand(target))
		}
	}
	
	// 'End Activation' action
	if inpututil.IsKeyJustPressed(ebiten.KeyE) || inpututil.IsKeyJustPressed(ebiten.KeyEnd) {
		h.queue.PushOutgoing(commands.NewEndActivationCommand())
	}
}
