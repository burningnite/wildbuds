package app

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"wildbuds/internal/assets"
	"wildbuds/internal/transport"
)

var _ Scene = (*LobbyScene)(nil)

// LobbyScene represents the P2P lobby waiting state while searching for peers.
type LobbyScene struct {
	transport   transport.Transport
	backButton  menuButton
	backHovered bool
}

// NewLobbyScene initializes the lobby scene, calls transport.NewLibp2pTransport(), and stores it.
func NewLobbyScene() (*LobbyScene, error) {
	t, err := transport.NewLibp2pTransport()
	if err != nil {
		return nil, err
	}

	const (
		screenWidth  = 800
		screenHeight = 600
		btnW         = 200
		btnH         = 40
	)

	backButton := menuButton{
		label: "Back",
		x:     (screenWidth - btnW) / 2,
		y:     400,
		w:     btnW,
		h:     btnH,
	}

	return &LobbyScene{
		transport:  t,
		backButton: backButton,
	}, nil
}

// Update checks if the transport is connected. If connected, transitions to BattleScene passing the transport.
// Also handles Back button click to return to StartMenuScene.
func (s *LobbyScene) Update() (Transition, error) {
	mx, my := ebiten.CursorPosition()
	s.backHovered = s.backButton.contains(mx, my)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if s.backHovered {
			log.Println("[LobbyScene] Back button clicked")
			if s.transport != nil {
				_ = s.transport.Close()
			}
			return Transition{NextScene: NewStartMenuScene()}, nil
		}
	}

	if s.transport != nil && s.transport.IsConnected() {
		log.Println("[LobbyScene] Peer connected, transitioning to BattleScene")
		battle, err := NewBattleScene(s.transport)
		if err != nil {
			return Transition{}, err
		}
		return Transition{NextScene: battle}, nil
	}

	return Transition{}, nil
}

// Draw renders "Searching for local peers..." and the Back button.
func (s *LobbyScene) Draw(screen *ebiten.Image) {
	// Background fill
	screen.Fill(color.RGBA{R: 24, G: 28, B: 36, A: 255})

	msg := "Searching for local peers..."
	face := assets.GetFont(20)
	w, _ := text.Measure(msg, face, 0)
	x := (800 - w) / 2
	y := float64(600)/2 - 30

	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	text.Draw(screen, msg, face, op)

	// Draw Back button
	drawButton(screen, s.backButton, s.backHovered)
}
