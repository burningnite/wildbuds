use bevy::prelude::*;
use bevy::tasks::IoTaskPool;
use matchbox_socket::WebRtcSocket;

pub struct NetworkPlugin;

impl Plugin for NetworkPlugin {
    fn build(&self, app: &mut App) {
        app.add_systems(Startup, setup_network)
           .add_systems(Update, process_network);
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

fn process_network(mut socket_res: ResMut<NetworkSocket>) {
    let socket = &mut socket_res.0;
    
    for (peer, state) in socket.update_peers() {
        info!("Peer {} state: {:?}", peer, state);
    }
    
    for (peer, payload) in socket.receive() {
        if let Ok(message) = std::str::from_utf8(&payload) {
            info!("Received message from {}: {}", peer, message);
        }
    }
}
