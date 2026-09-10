use bevy::prelude::*;
use crate::commands::{CommandQueue, GameCommand, NetworkCommand};
use crate::state::{TurnState, TurnPhase, AppState, Player};
use crate::components::{GridPosition, Owner};

pub struct GameLogicPlugin;

impl Plugin for GameLogicPlugin {
    fn build(&self, app: &mut App) {
        app.add_systems(Update, resolve_commands.run_if(in_state(AppState::InGame)));
    }
}

fn resolve_commands(
    mut queue: ResMut<CommandQueue>,
    mut turn_state: ResMut<TurnState>,
    mut next_phase: ResMut<NextState<TurnPhase>>,
    mut units: Query<(Entity, &mut GridPosition, &Owner)>,
) {
    while let Some(network_cmd) = queue.incoming.pop_front() {
        let sender = network_cmd.sender;
        let command = network_cmd.command;

        // Validation: Must be the active player's turn to do anything
        if sender != turn_state.active_player {
            warn!("Rejected command from {:?} - not their turn", sender);
            continue;
        }

        match command {
            GameCommand::SelectUnit { target } => {
                // Find unit at target
                if let Some((entity, _, owner)) = units.iter().find(|(_, pos, _)| **pos == target) {
                    if owner.0 == sender {
                        info!("Unit selected by {:?}", sender);
                        turn_state.active_unit = Some(entity);
                        next_phase.set(TurnPhase::ChooseAction);
                    } else {
                        warn!("Rejected SelectUnit - unit belongs to {:?}", owner.0);
                    }
                } else {
                    warn!("Rejected SelectUnit - no unit at {:?}", target);
                }
            }
            GameCommand::MoveUnit { destination } => {
                if let Some(active_unit) = turn_state.active_unit {
                    // Find the active unit and update its position
                    if let Ok((_, mut pos, _)) = units.get_mut(active_unit) {
                        // TODO: Add pathfinding/range validation here
                        info!("Moving unit to {:?}", destination);
                        *pos = destination;
                    }
                } else {
                    warn!("Rejected MoveUnit - no active unit");
                }
            }
            GameCommand::Attack { target } => {
                if let Some(_active_unit) = turn_state.active_unit {
                    info!("Attacking {:?}", target);
                    // TODO: Implement combat
                } else {
                    warn!("Rejected Attack - no active unit");
                }
            }
            GameCommand::EndActivation => {
                info!("{:?} ended activation", sender);
                turn_state.active_unit = None;
                turn_state.active_player = turn_state.active_player.next();
                next_phase.set(TurnPhase::SelectUnit);
            }
        }
    }
}
