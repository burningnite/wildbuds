package app

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"wildbuds/internal/i18n"
)

func TestStartMenuScene_ImplementsScene(t *testing.T) {
	menu := NewStartMenuScene()
	var _ Scene = menu
}

func TestStartMenuScene_Initialization(t *testing.T) {
	menu := NewStartMenuScene()
	expectedLabels := []string{
		i18n.Get("spar"),
		i18n.Get("duel"),
		i18n.Get("team_building"),
		i18n.Get("profile"),
		i18n.Get("options"),
		i18n.Get("debugger"),
		i18n.Get("quit"),
	}

	if len(menu.buttons) != len(expectedLabels) {
		t.Fatalf("expected %d buttons, got %d", len(expectedLabels), len(menu.buttons))
	}

	for i, label := range expectedLabels {
		if menu.buttons[i].label != label {
			t.Errorf("button %d: expected label %q, got %q", i, label, menu.buttons[i].label)
		}
		if menu.buttons[i].w <= 0 || menu.buttons[i].h <= 0 {
			t.Errorf("button %d (%s) has non-positive dimensions: w=%d, h=%d",
				i, label, menu.buttons[i].w, menu.buttons[i].h)
		}
		if i > 0 {
			prev := menu.buttons[i-1]
			curr := menu.buttons[i]
			if curr.y <= prev.y+prev.h {
				t.Errorf("button %d overlaps with button %d vertically: prev.bottom=%d, curr.top=%d",
					i, i-1, prev.y+prev.h, curr.y)
			}
		}
	}

	// Check language menu button
	if menu.langButton.x != 740 || menu.langButton.y != 540 || menu.langButton.w != 40 || menu.langButton.h != 40 {
		t.Errorf("unexpected langButton dimensions/position: %+v", menu.langButton)
	}
	if menu.langButton.label != i18n.Get("lang_menu") {
		t.Errorf("expected langButton label %q, got %q", i18n.Get("lang_menu"), menu.langButton.label)
	}
}

func TestStartMenuScene_ButtonContains(t *testing.T) {
	btn := menuButton{label: "Test", x: 100, y: 100, w: 50, h: 30}

	// Inside bounds
	if !btn.contains(100, 100) {
		t.Error("expected top-left corner (100, 100) to be contained")
	}
	if !btn.contains(125, 115) {
		t.Error("expected interior point (125, 115) to be contained")
	}
	if !btn.contains(150, 130) {
		t.Error("expected bottom-right corner (150, 130) to be contained")
	}

	// Outside bounds
	if btn.contains(99, 100) {
		t.Error("expected point left of button to not be contained")
	}
	if btn.contains(100, 99) {
		t.Error("expected point above button to not be contained")
	}
	if btn.contains(151, 100) {
		t.Error("expected point right of button to not be contained")
	}
	if btn.contains(100, 131) {
		t.Error("expected point below button to not be contained")
	}
}

func TestStartMenuScene_Transitions(t *testing.T) {
	menu := NewStartMenuScene()

	// Quit
	tr, err := menu.handleAction("Quit")
	if err != nil {
		t.Fatalf("unexpected error on Quit: %v", err)
	}
	if !tr.Quit {
		t.Errorf("expected Quit transition to have Quit: true")
	}
	if tr.NextScene != nil {
		t.Errorf("expected Quit transition to have nil NextScene")
	}

	// Spar
	tr, err = menu.handleAction("Spar")
	if err != nil {
		t.Fatalf("unexpected error on Spar: %v", err)
	}
	if tr.Quit {
		t.Errorf("expected Spar transition to have Quit: false")
	}
	if tr.NextScene == nil {
		t.Fatalf("expected Spar transition to have non-nil NextScene")
	}
	if _, ok := tr.NextScene.(*BattleScene); !ok {
		t.Errorf("expected Spar NextScene to be *BattleScene, got %T", tr.NextScene)
	}

	// Duel
	tr, err = menu.handleAction("Duel")
	if err != nil {
		t.Fatalf("unexpected error on Duel: %v", err)
	}
	if tr.Quit {
		t.Errorf("expected Duel transition to have Quit: false")
	}
	if tr.NextScene == nil {
		t.Fatalf("expected Duel transition to have non-nil NextScene")
	}
	if _, ok := tr.NextScene.(*BattleScene); !ok {
		t.Errorf("expected Duel NextScene to be *BattleScene, got %T", tr.NextScene)
	}

	// Debugger
	tr, err = menu.handleAction("Debugger")
	if err != nil {
		t.Fatalf("unexpected error on Debugger: %v", err)
	}
	if tr.Quit {
		t.Errorf("expected Debugger transition to have Quit: false")
	}
	if tr.NextScene == nil {
		t.Fatalf("expected Debugger transition to have non-nil NextScene")
	}
	if _, ok := tr.NextScene.(*DebuggerScene); !ok {
		t.Errorf("expected Debugger NextScene to be *DebuggerScene, got %T", tr.NextScene)
	}

	// Language menu transition
	tr, err = menu.handleAction("lang_menu")
	if err != nil {
		t.Fatalf("unexpected error on lang_menu: %v", err)
	}
	if tr.NextScene == nil {
		t.Fatalf("expected lang_menu transition to have non-nil NextScene")
	}
	if _, ok := tr.NextScene.(*LanguageScene); !ok {
		t.Errorf("expected lang_menu NextScene to be *LanguageScene, got %T", tr.NextScene)
	}

	// Other buttons do nothing
	noOpButtons := []string{"Team Building", "Profile", "Options", "Unknown"}
	for _, btn := range noOpButtons {
		tr, err = menu.handleAction(btn)
		if err != nil {
			t.Fatalf("unexpected error on %s: %v", btn, err)
		}
		if tr.Quit || tr.NextScene != nil {
			t.Errorf("expected no-op transition for %s, got %+v", btn, tr)
		}
	}
}

func TestStartMenuScene_Draw(t *testing.T) {
	menu := NewStartMenuScene()
	screen := ebiten.NewImage(800, 600)

	// Test drawing with no hover
	menu.hoveredIdx = -1
	menu.langHovered = false
	menu.Draw(screen)

	// Test drawing with hover on each button
	for i := range menu.buttons {
		menu.hoveredIdx = i
		menu.langHovered = false
		menu.Draw(screen)
	}

	// Test drawing with hover on lang button
	menu.hoveredIdx = -1
	menu.langHovered = true
	menu.Draw(screen)
}

func TestStartMenuScene_UpdateDefault(t *testing.T) {
	menu := NewStartMenuScene()
	// When running headlessly in tests with no mouse clicks, Update should return empty transition without error
	tr, err := menu.Update()
	if err != nil {
		t.Fatalf("unexpected error on Update(): %v", err)
	}
	if tr.Quit || tr.NextScene != nil {
		t.Errorf("expected default Update() to return empty Transition, got %+v", tr)
	}
}
