# M10: Audio System Roadmap
*Status: PLANNED*

## 1. Audio Engine Setup
- [ ] Import `github.com/hajimehoshi/ebiten/v2/audio`.
- [ ] Import necessary decoders (e.g., `audio/wav` or `audio/mp3`, `audio/vorbis`).
- [ ] Initialize the global Ebitengine `audio.Context` with a standard sample rate (e.g., 44100 Hz).
- [ ] Create an `AudioManager` struct in `internal/audio/manager.go`.

## 2. Sound Effects (SFX)
- [ ] Acquire short `.wav` or `.ogg` files for UI and combat interactions.
- [ ] Load and decode SFX assets into memory as byte arrays/players.
- [ ] Implement `AudioManager.PlaySFX(id string)` for fire-and-forget playback.
- [ ] Hook SFX triggers into the UI (Hover beep, Click confirm, Error buzz).
- [ ] Hook SFX triggers into the Resolver events (Movement footstep, attack swoosh, hit impact).

## 3. Music (BGM)
- [ ] Acquire a looping background music track.
- [ ] Load and decode the BGM stream.
- [ ] Implement `AudioManager.PlayBGM(id string)` ensuring infinite looping.
- [ ] Implement volume control, allowing BGM to play softly beneath the SFX.

## 4. Integration
- [ ] Pass the `AudioManager` instance into the main `app.Game` struct.
- [ ] Ensure audio loading happens asynchronously or during an initial loading screen to prevent startup stuttering.
- [ ] Expose basic volume controls (Mute, Up, Down) via keybinds in the `InputHandler`.
