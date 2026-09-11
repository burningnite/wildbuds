# M8: Asset & Sprite Integration Roadmap
*Status: PLANNED*

## 1. Asset Management System
- [ ] Create `internal/render/assets.go`.
- [ ] Implement an `AssetManager` struct to cache loaded images, avoiding redundant IO.
- [ ] Integrate Ebitengine's `image` loading utilities to load `.png` files.
- [ ] Define standard unit sprite dimensions and slice sprite sheets.
- [ ] Load and cache required TTF/OTF fonts for UI rendering.

## 2. Tilemap Rendering
- [ ] Acquire or create simple texture assets for grid tiles (Grass, Water, Stone, etc.).
- [ ] Update `internal/render/board.go`.
- [ ] Replace `ebitenutil.DrawRect` with Ebitengine's `DrawImage` using the loaded tile textures.
- [ ] Ensure tile textures align perfectly with the `GridToScreen` coordinate system.

## 3. Unit Sprites & Animation
- [ ] Assign specific sprite assets to distinct Unit configurations based on elements.
- [ ] Replace basic colored rectangle unit rendering in `Game.Draw()` with sprite rendering.
- [ ] Implement an `AnimationController` struct.
- [ ] Define states: `Idle`, `Moving`, `Attacking`.
- [ ] Update sprite frames based on a time delta to create idle breathing/bouncing animations.
- [ ] Implement directional facing (flip sprite horizontally if moving/attacking left).

## 4. Visual Effects (VFX)
- [ ] Create visual indicators for selection (e.g., a rotating reticle sprite under the unit).
- [ ] Implement floating text for combat results ("-15 HP", "Super Effective!").
- [ ] Add simple particle or flash effects when an attack connects, tinted by the elemental type (e.g., Red flash for fire).
