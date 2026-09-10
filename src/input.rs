use bevy::prelude::*;
use crate::commands::{CommandQueue, GameCommand};
use crate::components::GridPosition;
use crate::state::{AppState, TurnPhase};

pub struct InputPlugin;

impl Plugin for InputPlugin {
    fn build(&self, app: &mut App) {
        app.add_systems(Update, handle_input.run_if(in_state(AppState::InGame)));
    }
}

fn handle_input(
    keyboard_input: Res<ButtonInput<KeyCode>>,
    mouse_input: Res<ButtonInput<MouseButton>>,
    windows: Query<&Window>,
    turn_phase: Res<State<TurnPhase>>,
    mut queue: ResMut<CommandQueue>,
    mut cursor_query: Query<&mut GridPosition, With<crate::components::CursorVisual>>,
) {
    let mut clicked = false;
    let mut target_pos = None;

    if let Ok(mut grid_pos) = cursor_query.get_single_mut() {
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
        
        if keyboard_input.just_pressed(KeyCode::Enter) || keyboard_input.just_pressed(KeyCode::Space) {
            clicked = true;
            target_pos = Some(*grid_pos);
        }

        if mouse_input.just_pressed(MouseButton::Left) {
            if let Ok(window) = windows.get_single() {
                if let Some(cursor_position) = window.cursor_position() {
                    let world_x = cursor_position.x - window.width() / 2.0;
                    let world_y = -(cursor_position.y - window.height() / 2.0);
                    
                    grid_pos.x = (world_x / 32.0).round() as i32;
                    grid_pos.y = (world_y / 32.0).round() as i32;
                    clicked = true;
                    target_pos = Some(*grid_pos);
                }
            }
        }
    }
    
    if keyboard_input.just_pressed(KeyCode::KeyE) {
        queue.outgoing.push_back(GameCommand::EndActivation);
        return;
    }

    if clicked {
        if let Some(target) = target_pos {
            match turn_phase.get() {
                TurnPhase::SelectUnit => {
                    queue.outgoing.push_back(GameCommand::SelectUnit { target });
                }
                TurnPhase::ChooseAction => {
                    queue.outgoing.push_back(GameCommand::MoveUnit { destination: target });
                }
            }
        }
    }
}
