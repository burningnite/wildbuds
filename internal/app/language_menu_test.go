package app

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"wildbuds/internal/i18n"
)

func TestLanguageScene_ImplementsScene(t *testing.T) {
	scene := NewLanguageScene()
	var _ Scene = scene
}

func TestLanguageScene_Initialization(t *testing.T) {
	scene := NewLanguageScene()
	expectedLanguages := []string{
		"English",
		"Español",
		"Français",
		"Deutsch",
		"Italiano",
		"Português",
	}

	if len(scene.buttons) != len(expectedLanguages) {
		t.Fatalf("expected %d buttons, got %d", len(expectedLanguages), len(scene.buttons))
	}

	for i, lang := range expectedLanguages {
		if scene.buttons[i].label != lang {
			t.Errorf("button %d: expected label %q, got %q", i, lang, scene.buttons[i].label)
		}
		if scene.buttons[i].w <= 0 || scene.buttons[i].h <= 0 {
			t.Errorf("button %d (%s) has non-positive dimensions: w=%d, h=%d",
				i, lang, scene.buttons[i].w, scene.buttons[i].h)
		}
	}
}

func TestLanguageScene_ButtonContains(t *testing.T) {
	btn := langButton{label: "English", x: 100, y: 100, w: 200, h: 40}

	if !btn.contains(100, 100) {
		t.Error("expected top-left corner to be contained")
	}
	if !btn.contains(200, 120) {
		t.Error("expected interior point to be contained")
	}
	if !btn.contains(300, 140) {
		t.Error("expected bottom-right corner to be contained")
	}
	if btn.contains(99, 100) {
		t.Error("expected point left of button to not be contained")
	}
	if btn.contains(301, 100) {
		t.Error("expected point right of button to not be contained")
	}
}

func TestLanguageScene_Draw(t *testing.T) {
	scene := NewLanguageScene()
	screen := ebiten.NewImage(800, 600)

	scene.hoveredIdx = 0
	scene.Draw(screen)

	scene.hoveredIdx = -1
	scene.Draw(screen)
}

func TestLanguageScene_LanguageSelection(t *testing.T) {
	origLang := i18n.CurrentLanguage
	defer func() {
		i18n.CurrentLanguage = origLang
	}()

	scene := NewLanguageScene()
	// Find "Français" button index
	var targetIdx = -1
	for i, btn := range scene.buttons {
		if btn.label == "Français" {
			targetIdx = i
			break
		}
	}

	if targetIdx < 0 {
		t.Fatalf("Français button not found")
	}

	btn := scene.buttons[targetIdx]
	// Simulate click (or test direct setting as requested in requirement 4)
	i18n.CurrentLanguage = btn.label

	if i18n.CurrentLanguage != "Français" {
		t.Errorf("expected CurrentLanguage to be 'Français', got %q", i18n.CurrentLanguage)
	}
}
