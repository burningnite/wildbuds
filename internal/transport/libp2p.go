package transport

import (
	"bufio"
	"context"
	"fmt"
	"sync"
	"time"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

const (
	ProtocolWildbuds = protocol.ID("/wildbuds/1.0.0")
	RendezvousString = "wildbuds-local"
)

// Compile-time interface satisfaction checks.
var (
	_ Transport        = (*Libp2pTransport)(nil)
	_ PollingTransport = (*Libp2pTransport)(nil)
)

// mdnsNotifee implements mdns.Notifee to handle discovered peers.
type mdnsNotifee struct {
	t *Libp2pTransport
}

func (n *mdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	n.t.handlePeerFound(pi)
}

// Libp2pTransport implements transport.Transport using go-libp2p and mDNS discovery.
type Libp2pTransport struct {
	mu           sync.RWMutex
	host         host.Host
	mdnsService  mdns.Service
	stream       network.Stream
	localPlayer  domain.Player
	remotePlayer domain.Player
	connected    bool
	closed       bool
	closeOnce    sync.Once
	inCh         chan commands.NetworkCommand
	done         chan struct{}
}

// NewLibp2pTransport creates and initializes a Libp2pTransport instance.
// It starts a libp2p host and mDNS discovery service without blocking on connection.
func NewLibp2pTransport() (*Libp2pTransport, error) {
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}
	fmt.Printf("[Libp2p] Host started with ID: %s, Addrs: %v\n", h.ID().String(), h.Addrs())

	t := &Libp2pTransport{
		host:         h,
		localPlayer:  domain.PlayerOne, // Default until stream handshake
		remotePlayer: domain.PlayerTwo,
		inCh:         make(chan commands.NetworkCommand, DefaultBufferSize),
		done:         make(chan struct{}),
	}

	// Register stream handler
	h.SetStreamHandler(ProtocolWildbuds, func(stream network.Stream) {
		t.handleIncomingStream(stream)
	})

	// Setup mDNS discovery service
	notifee := &mdnsNotifee{t: t}
	mdnsSvc := mdns.NewMdnsService(h, RendezvousString, notifee)
	if err := mdnsSvc.Start(); err != nil {
		fmt.Printf("[Libp2p] WARNING: failed to start mDNS service: %v\n", err)
	}
	t.mdnsService = mdnsSvc

	return t, nil
}

func (t *Libp2pTransport) handlePeerFound(pi peer.AddrInfo) {
	if pi.ID == t.host.ID() {
		return
	}

	fmt.Printf("[Libp2p] Discovered peer: %s\n", pi.ID.String())

	t.mu.RLock()
	if t.connected || t.closed {
		t.mu.RUnlock()
		return
	}
	t.mu.RUnlock()

	go func() {
		fmt.Printf("[Libp2p] Attempting to connect to: %s\n", pi.ID.String())
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := t.host.Connect(ctx, pi); err != nil {
			return
		}

		localID := t.host.ID().String()
		remoteID := pi.ID.String()

		// To prevent both sides from opening a stream simultaneously and then closing each other's streams,
		// we deterministically assign the dialing responsibility to only one peer.
		if localID >= remoteID {
			return
		}

		t.mu.Lock()
		if t.connected || t.closed {
			t.mu.Unlock()
			return
		}
		t.mu.Unlock()

		stream, err := t.host.NewStream(ctx, pi.ID, ProtocolWildbuds)
		if err != nil {
			return
		}

		t.setupStream(stream, pi.ID)
	}()
}

func (t *Libp2pTransport) handleIncomingStream(stream network.Stream) {
	remotePeer := stream.Conn().RemotePeer()
	fmt.Printf("[Libp2p] Incoming stream from: %s\n", remotePeer.String())

	t.mu.Lock()
	if t.connected || t.closed {
		t.mu.Unlock()
		_ = stream.Close()
		return
	}
	t.mu.Unlock()

	t.setupStream(stream, remotePeer)
}

func (t *Libp2pTransport) setupStream(stream network.Stream, remotePeer peer.ID) {
	localID := t.host.ID().String()
	remoteID := remotePeer.String()

	var lp, rp domain.Player
	if localID < remoteID {
		lp = domain.PlayerOne
		rp = domain.PlayerTwo
	} else {
		lp = domain.PlayerTwo
		rp = domain.PlayerOne
	}

	t.mu.Lock()
	if t.connected || t.closed {
		t.mu.Unlock()
		_ = stream.Close()
		return
	}
	t.stream = stream
	t.localPlayer = lp
	t.remotePlayer = rp
	t.connected = true
	t.mu.Unlock()

	fmt.Printf("[Libp2p] Stream fully established with %s. We are %v\n", rp, lp)
	go t.readLoop(stream, rp)
}

func (t *Libp2pTransport) readLoop(stream network.Stream, remotePlayer domain.Player) {
	// 🛡️ SECURITY: Use bufio.Scanner with a limited buffer to prevent
	// uncontrolled resource consumption (DoS) if a peer omits newlines.
	const maxCommandSize = 8192
	scanner := bufio.NewScanner(stream)
	scanner.Buffer(make([]byte, 4096), maxCommandSize)
	for {
		select {
		case <-t.done:
			return
		default:
		}

		if !scanner.Scan() {
			t.mu.Lock()
			t.connected = false
			t.mu.Unlock()
			break
		}
		line := scanner.Bytes()

		netCmd, err := DeserializeNetworkCommand(line)
		if err != nil {
			continue
		}

		if !netCmd.Sender.IsValid() {
			netCmd.Sender = remotePlayer
		}

		select {
		case t.inCh <- netCmd:
		case <-t.done:
			return
		}
	}
}

// Send serializes a GameCommand and writes it to the active peer stream.
func (t *Libp2pTransport) Send(cmd commands.GameCommand) error {
	if err := cmd.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCommand, err)
	}

	t.mu.RLock()
	if t.closed {
		t.mu.RUnlock()
		return ErrTransportClosed
	}
	if !t.connected || t.stream == nil {
		t.mu.RUnlock()
		return ErrNotConnected
	}
	stream := t.stream
	t.mu.RUnlock()

	data, err := SerializeGameCommand(cmd)
	if err != nil {
		return fmt.Errorf("failed to serialize game command: %w", err)
	}

	_, err = stream.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("failed to write to stream: %w", err)
	}

	// Mirror the command locally so the local resolver processes our own actions
	netCmd := commands.NewNetworkCommand(t.LocalPlayer(), cmd)
	select {
	case t.inCh <- netCmd:
	default:
		return ErrQueueFull
	}

	return nil
}

// Receive returns the read-only channel delivering incoming NetworkCommands.
func (t *Libp2pTransport) Receive() <-chan commands.NetworkCommand {
	return t.inCh
}

// LocalPlayer returns the player identifier assigned to this client endpoint.
func (t *Libp2pTransport) LocalPlayer() domain.Player {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.localPlayer
}

// RemotePlayer returns the opponent player identifier.
func (t *Libp2pTransport) RemotePlayer() domain.Player {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.remotePlayer
}

// Close gracefully shuts down the transport, stream, host, and channels.
func (t *Libp2pTransport) Close() error {
	t.closeOnce.Do(func() {
		close(t.done)

		t.mu.Lock()
		t.closed = true
		t.connected = false
		if t.stream != nil {
			_ = t.stream.Close()
		}
		if t.host != nil {
			_ = t.host.Close()
		}
		close(t.inCh)
		t.mu.Unlock()
	})
	return nil
}

// IsClosed returns true if the transport has been closed.
func (t *Libp2pTransport) IsClosed() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.closed
}

// IsConnected returns true if an active stream connection has been established.
func (t *Libp2pTransport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.connected && !t.closed
}

// Poll retrieves the next available NetworkCommand without blocking.
func (t *Libp2pTransport) Poll() (commands.NetworkCommand, bool) {
	select {
	case cmd, ok := <-t.inCh:
		if !ok {
			return commands.NetworkCommand{}, false
		}
		return cmd, true
	default:
		return commands.NetworkCommand{}, false
	}
}

// Drain extracts all currently buffered NetworkCommands in a single batch.
func (t *Libp2pTransport) Drain() []commands.NetworkCommand {
	var result []commands.NetworkCommand
	for {
		select {
		case cmd, ok := <-t.inCh:
			if !ok {
				return result
			}
			result = append(result, cmd)
		default:
			return result
		}
	}
}

// Len returns the number of pending incoming commands currently buffered.
func (t *Libp2pTransport) Len() int {
	return len(t.inCh)
}
