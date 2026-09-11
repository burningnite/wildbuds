package app

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestLobbyScene_ImplementsScene(t *testing.T) {
	lobby, err := NewLobbyScene()
	if err != nil {
		t.Fatalf("unexpected error on NewLobbyScene(): %v", err)
	}
	defer lobby.transport.Close()

	var _ Scene = lobby
}

func TestLobbyScene_Draw(t *testing.T) {
	lobby, err := NewLobbyScene()
	if err != nil {
		t.Fatalf("unexpected error on NewLobbyScene(): %v", err)
	}
	defer lobby.transport.Close()

	screen := ebiten.NewImage(800, 600)
	lobby.Draw(screen)
}

func TestLobbyScene_UpdateDefault(t *testing.T) {
	lobby, err := NewLobbyScene()
	if err != nil {
		t.Fatalf("unexpected error on NewLobbyScene(): %v", err)
	}
	defer lobby.transport.Close()

	tr, err := lobby.Update()
	if err != nil {
		t.Fatalf("unexpected error on LobbyScene.Update(): %v", err)
	}
	if tr.Quit || tr.NextScene != nil {
		t.Errorf("expected empty transition while waiting for peer, got %+v", tr)
	}
}
