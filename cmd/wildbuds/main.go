package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"wildbuds/internal/app"
)

func main() {
	startScene := app.NewStartMenuScene()

	game, err := app.NewGame(startScene)
	if err != nil {
		log.Fatalf("Failed to initialize game: %v", err)
	}

	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("Wildbuds (Go/Ebitengine Port)")
	
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
