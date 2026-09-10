use bevy::prelude::*;
use crate::components::GridPosition;

pub struct InputPlugin;

impl Plugin for InputPlugin {
    fn build(&self, app: &mut App) {
        app.add_systems(Update, handle_input);
    }
}

fn handle_input(
    keyboard_input: Res<ButtonInput<KeyCode>>,
    mouse_input: Res<ButtonInput<MouseButton>>,
    windows: Query<&Window>,
    mut cursor_query: Query<&mut GridPosition>,
) {
    if let Ok(mut grid_pos) = cursor_query.get_single_mut() {
        // Keyboard movement
        if keyboard_input.just_pressed(KeyCode::ArrowUp) || keyboard_input.just_pressed(KeyCode::KeyW) {
            grid_pos.y += 1;
        }
        if keyboard_input.just_pressed(KeyCode::ArrowDown) || keyboard_input.just_pressed(KeyCode::KeyS) {
            grid_pos.y -= 1;
        }
        if keyboard_input.just_pressed(KeyCode::ArrowRight) || keyboard_input.just_pressed(KeyCode::KeyD) {
            grid_pos.x += 1;
        }
        if keyboard_input.just_pressed(KeyCode::ArrowLeft) || keyboard_input.just_pressed(KeyCode::KeyA) {
            grid_pos.x -= 1;
        }

        // Mouse click movement (very basic translation)
        if mouse_input.just_pressed(MouseButton::Left) {
            if let Ok(window) = windows.get_single() {
                if let Some(cursor_position) = window.cursor_position() {
                    // Assuming center of screen is 0,0 and grid cells are 32x32
                    let world_x = cursor_position.x - window.width() / 2.0;
                    let world_y = -(cursor_position.y - window.height() / 2.0); // y goes down in UI
                    
                    grid_pos.x = (world_x / 32.0).round() as i32;
                    grid_pos.y = (world_y / 32.0).round() as i32;
                }
            }
        }
    }
}
