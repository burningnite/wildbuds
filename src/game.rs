use bevy::prelude::*;
use crate::commands::{CommandQueue, GameCommand};
use crate::state::{TurnState, TurnPhase, AppState};
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::state::Player;
    use crate::commands::NetworkCommand;
    use crate::components::{GridPosition, Owner};

    fn setup_test_app() -> App {
        let mut app = App::new();
        app.add_plugins(MinimalPlugins);
        app.add_plugins(bevy::state::app::StatesPlugin);
        app.init_state::<AppState>();
        app.init_state::<TurnPhase>();
        app.init_resource::<CommandQueue>();
        app.insert_resource(TurnState {
            active_player: Player::One,
            active_unit: None,
            local_player: Player::One,
        });
        app.add_plugins(GameLogicPlugin);
        app.world_mut().resource_mut::<NextState<AppState>>().set(AppState::InGame);
        app.update(); app.update(); // transition
        app
    }

    #[test]
    fn test_select_unit_wrong_player() {
        let mut app = setup_test_app();
        app.world_mut().spawn((GridPosition { x: 0, y: 0 }, Owner(Player::One)));
        
        app.world_mut().resource_mut::<CommandQueue>().incoming.push_back(NetworkCommand {
            sender: Player::Two,
            command: GameCommand::SelectUnit { target: GridPosition { x: 0, y: 0 } },
        });
        app.update(); app.update();
        assert_eq!(app.world().resource::<TurnState>().active_unit, None);
    }

    #[test]
    fn test_select_unit_valid() {
        let mut app = setup_test_app();
        let unit_id = app.world_mut().spawn((GridPosition { x: 0, y: 0 }, Owner(Player::One))).id();
        
        app.world_mut().resource_mut::<CommandQueue>().incoming.push_back(NetworkCommand {
            sender: Player::One,
            command: GameCommand::SelectUnit { target: GridPosition { x: 0, y: 0 } },
        });
        app.update(); app.update();
        assert_eq!(app.world().resource::<TurnState>().active_unit, Some(unit_id));
        assert_eq!(*app.world().resource::<State<TurnPhase>>().get(), TurnPhase::ChooseAction);
    }
}
