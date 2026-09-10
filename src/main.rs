mod components;
mod combat;
mod input;
mod network;

use bevy::prelude::*;
use components::GridPosition;

fn main() {
    App::new()
        .add_plugins(DefaultPlugins)
        .add_plugins(input::InputPlugin)
        .add_plugins(network::NetworkPlugin)
        .add_systems(Startup, setup)
        .run();
}

fn setup(mut commands: Commands) {
    commands.spawn(Camera2dBundle::default());
    
    // Spawn a cursor entity
    commands.spawn(GridPosition { x: 0, y: 0 });
}
