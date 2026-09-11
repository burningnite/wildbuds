package app

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestDebuggerSceneInterface(t *testing.T) {
	scene := NewDebuggerScene()
	var _ Scene = scene

	if scene == nil {
		t.Fatal("expected non-nil DebuggerScene")
	}
}

func TestDebuggerSceneInBackBtn(t *testing.T) {
	scene := NewDebuggerScene()

	// Inside the button
	if !scene.inBackBtn(scene.backBtnX+5, scene.backBtnY+5) {
		t.Errorf("expected (%d, %d) to be inside back button", scene.backBtnX+5, scene.backBtnY+5)
	}

	// Outside button: to the left
	if scene.inBackBtn(scene.backBtnX-1, scene.backBtnY) {
		t.Error("expected left of button to be outside")
	}

	// Outside button: above
	if scene.inBackBtn(scene.backBtnX, scene.backBtnY-1) {
		t.Error("expected above button to be outside")
	}

	// Outside button: to the right
	if scene.inBackBtn(scene.backBtnX+scene.backBtnW+1, scene.backBtnY) {
		t.Error("expected right of button to be outside")
	}

	// Outside button: below
	if scene.inBackBtn(scene.backBtnX, scene.backBtnY+scene.backBtnH+1) {
		t.Error("expected below button to be outside")
	}
}

func TestDebuggerSceneUpdate(t *testing.T) {
	scene := NewDebuggerScene()
	transition, err := scene.Update()
	if err != nil {
		t.Fatalf("unexpected error on Update: %v", err)
	}
	// Without mouse click, next scene should be nil
	if transition.NextScene != nil {
		t.Errorf("expected NextScene to be nil without click, got %v", transition.NextScene)
	}
}

func TestDebuggerSceneDraw(t *testing.T) {
	scene := NewDebuggerScene()
	screen := ebiten.NewImage(800, 600)

	// Draw unhovered
	scene.backHovered = false
	scene.Draw(screen)

	// Draw hovered
	scene.backHovered = true
	scene.Draw(screen)
}
