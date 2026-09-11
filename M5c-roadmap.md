# M5c: Localization & Fonts Roadmap
*Status: COMPLETED*

## 1. Font Embedding
- [ ] Download the Agave Nerd Font family.
- [ ] Create `internal/assets/fonts.go`.
- [ ] Embed the font file using `//go:embed` so it compiles into the binary.
- [ ] Create a utility function to parse and load the TTF/OTF file into an `ebiten.Font` face.

## 2. Translation Structure
- [ ] Create `internal/i18n/translations.go`.
- [ ] Define a simple translation dictionary structure (e.g., using `map[string]map[string]string` for `LanguageCode -> Key -> TranslatedString`).
- [ ] Implement English ("English") and Spanish ("Español") translations for all current UI elements ("Spar", "Duel", "Quit", "Back", etc.).
- [ ] Implement a global state or manager to track the active language (default: English).

## 3. Language Menu Scene
- [ ] Create `internal/app/language_menu.go`.
- [ ] Implement `LanguageScene` satisfying the `Scene` interface.
- [ ] Render a two-column, centered list of language options (Minecraft-style).
- [ ] Options must be written in their own language (e.g., "English", "Español", "Français", "Deutsch").
- [ ] English and Español will be functional; the rest will be dummies for now.
- [ ] Selecting an active language updates the global i18n state and returns to the Start Menu.

## 4. UI Integration
- [ ] Update `internal/app/start_menu.go` and `debugger.go` to use the `i18n` translation keys instead of hardcoded strings.
- [ ] Update `StartMenuScene` to render a small square button in the bottom-right corner.
- [ ] Make the square button transition to the `LanguageScene` when clicked.
- [ ] Ensure all text rendering uses the embedded Agave Nerd Font.
