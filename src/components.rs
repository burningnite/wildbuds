use bevy::prelude::*;
use serde::{Deserialize, Serialize};

#[derive(Component, Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub struct GridPosition {
    pub x: i32,
    pub y: i32,
}

#[derive(Component)]
pub struct GridVisual;

#[derive(Component)]
pub struct CursorVisual;

#[derive(Component)]
pub struct UnitVisual;

#[derive(Component, Debug, Clone)]
pub struct ActionTokens {
    pub movement: u8,
    pub attack: u8,
    pub special: u8,
}

#[derive(Component, Debug, Clone)]
pub struct BaseStats {
    pub hp: u32,
    pub attack: u32,
    pub defense: u32,
    pub speed: u32,
}

// Add an owner component so we know whose unit is whose
use crate::state::Player;

#[derive(Component, Debug, Clone, Copy, PartialEq, Eq)]
pub struct Owner(pub Player);
