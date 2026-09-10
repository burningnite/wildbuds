mod commands;
mod components;
mod combat;
mod game;
mod input;
mod network;
mod render;
mod state;

use bevy::prelude::*;
use commands::CommandQueue;
use state::{AppState, TurnPhase};

fn main() {
    App::new()
        .add_plugins(DefaultPlugins)
        .init_state::<AppState>()
        .init_state::<TurnPhase>()
        .init_resource::<CommandQueue>()
        .add_plugins(render::RenderPlugin)
        .add_plugins(input::InputPlugin)
        .add_plugins(network::NetworkPlugin)
        .add_plugins(game::GameLogicPlugin)
        .add_systems(Startup, setup)
        .run();
}

fn setup(mut commands: Commands) {
    commands.spawn(Camera2dBundle::default());
}
