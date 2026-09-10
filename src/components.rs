use bevy::prelude::*;

#[derive(Component, Debug, Clone, Copy, PartialEq, Eq)]
pub struct GridPosition {
    pub x: i32,
    pub y: i32,
}

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
