use bevy::prelude::*;
use serde::{Deserialize, Serialize};

#[derive(States, Debug, Clone, Copy, Eq, PartialEq, Hash, Default)]
pub enum AppState {
    #[default]
    Connecting,
    InGame,
}

#[derive(States, Debug, Clone, Copy, Eq, PartialEq, Hash, Default)]
pub enum TurnPhase {
    #[default]
    SelectUnit,
    ChooseAction,
}

#[derive(Debug, Clone, Copy, Eq, PartialEq, Serialize, Deserialize)]
pub enum Player {
    One,
    Two,
}

impl Player {
    pub fn next(self) -> Self {
        match self {
            Player::One => Player::Two,
            Player::Two => Player::One,
        }
    }
}

#[derive(Resource)]
pub struct TurnState {
    pub active_player: Player,
    pub active_unit: Option<Entity>,
    pub local_player: Player, // Local client identity; not replicated game state
}
