use bevy::prelude::*;
use bevy::tasks::IoTaskPool;
use matchbox_socket::WebRtcSocket;
use crate::commands::{CommandQueue, GameCommand, NetworkCommand};
use crate::state::{AppState, Player, TurnState};

pub struct NetworkPlugin;

impl Plugin for NetworkPlugin {
    fn build(&self, app: &mut App) {
        app.add_systems(OnEnter(AppState::Connecting), setup_network)
           .add_systems(Update, (
               process_connections.run_if(in_state(AppState::Connecting)),
               process_network_traffic.run_if(in_state(AppState::InGame))
           ));
    }
}

fn setup_network(mut commands: Commands) {
    let room_url = "ws://matchbox.machengine.org:3536/wildbuds";
    info!("connecting to matchbox server: {}", room_url);
    
    let (socket, message_loop) = WebRtcSocket::builder(room_url)
        .add_reliable_channel()
        .build();

    IoTaskPool::get().spawn(message_loop).detach();

    commands.insert_resource(NetworkSocket(socket));
}

#[derive(Resource)]
struct NetworkSocket(WebRtcSocket);

fn process_connections(
    mut socket_res: ResMut<NetworkSocket>,
    mut next_app_state: ResMut<NextState<AppState>>,
    mut commands: Commands,
) {
    let socket = &mut socket_res.0;
    
    for (peer, state) in socket.update_peers() {
        info!("Peer {} state: {:?}", peer, state);
    }
    
    let connected_peers: Vec<_> = socket.connected_peers().collect();
    if !connected_peers.is_empty() {
        let my_id = socket.id().unwrap().to_string();
        let their_id = connected_peers[0].to_string();
        
        let local_player = if my_id < their_id {
            Player::One
        } else {
            Player::Two
        };
        
        info!("Connected! I am {:?}", local_player);
        
        commands.insert_resource(TurnState {
            active_player: Player::One,
            active_unit: None,
            local_player,
        });
        
        next_app_state.set(AppState::InGame);
    }
}

fn process_network_traffic(
    mut socket_res: ResMut<NetworkSocket>,
    mut queue: ResMut<CommandQueue>,
    turn_state: Res<TurnState>,
) {
    let socket = &mut socket_res.0;
    
    for (_peer, payload) in socket.receive() {
        if let Ok(command) = serde_json::from_slice::<GameCommand>(&payload) {
            let remote_player = turn_state.local_player.next();
            queue.incoming.push_back(NetworkCommand {
                sender: remote_player,
                command,
            });
        }
    }
    
    while let Some(command) = queue.outgoing.pop_front() {
        if let Ok(payload) = serde_json::to_vec(&command) {
            let connected_peers: Vec<_> = socket.connected_peers().collect();
            for peer in connected_peers {
                socket.send(payload.clone().into(), peer);
            }
        }
        
        queue.incoming.push_back(NetworkCommand {
            sender: turn_state.local_player,
            command,
        });
    }
}
