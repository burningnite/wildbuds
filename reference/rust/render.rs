use bevy::prelude::*;
use crate::components::{GridPosition, GridVisual, CursorVisual, UnitVisual, Owner};
use crate::state::Player;

pub struct RenderPlugin;

impl Plugin for RenderPlugin {
    fn build(&self, app: &mut App) {
        app.add_systems(Startup, setup_board)
           .add_systems(Update, sync_visuals);
    }
}

fn setup_board(mut commands: Commands) {
    for x in -5..=5 {
        for y in -5..=5 {
            let color = if (x + y) % 2 == 0 {
                Color::srgb(0.2, 0.2, 0.2)
            } else {
                Color::srgb(0.3, 0.3, 0.3)
            };
            
            commands.spawn((
                SpriteBundle {
                    sprite: Sprite {
                        color,
                        custom_size: Some(Vec2::new(30.0, 30.0)),
                        ..default()
                    },
                    transform: Transform::from_xyz(x as f32 * 32.0, y as f32 * 32.0, 0.0),
                    ..default()
                },
                GridPosition { x, y },
                GridVisual,
            ));
        }
    }

    commands.spawn((
        SpriteBundle {
            sprite: Sprite {
                color: Color::srgb(0.2, 0.2, 0.8),
                custom_size: Some(Vec2::new(24.0, 24.0)),
                ..default()
            },
            transform: Transform::from_xyz(0.0, -96.0, 1.0),
            ..default()
        },
        GridPosition { x: 0, y: -3 },
        Owner(Player::One),
        UnitVisual,
    ));

    commands.spawn((
        SpriteBundle {
            sprite: Sprite {
                color: Color::srgb(0.8, 0.2, 0.2),
                custom_size: Some(Vec2::new(24.0, 24.0)),
                ..default()
            },
            transform: Transform::from_xyz(0.0, 96.0, 1.0),
            ..default()
        },
        GridPosition { x: 0, y: 3 },
        Owner(Player::Two),
        UnitVisual,
    ));

    commands.spawn((
        SpriteBundle {
            sprite: Sprite {
                color: Color::srgba(1.0, 1.0, 1.0, 0.5),
                custom_size: Some(Vec2::new(32.0, 32.0)),
                ..default()
            },
            transform: Transform::from_xyz(0.0, 0.0, 2.0),
            ..default()
        },
        GridPosition { x: 0, y: 0 },
        CursorVisual,
    ));
}

fn sync_visuals(
    mut query: Query<(&GridPosition, &mut Transform), Changed<GridPosition>>
) {
    for (grid_pos, mut transform) in &mut query {
        transform.translation.x = grid_pos.x as f32 * 32.0;
        transform.translation.y = grid_pos.y as f32 * 32.0;
    }
}
