package transport

import (
	"fmt"
	"sync"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
)

const (
	// DefaultBufferSize is the default capacity of the inbound command channel.
	// Provides plenty of headroom for 60Hz frame bursts without unbound heap usage.
	DefaultBufferSize = 256
)

// Compile-time interface satisfaction checks.
var (
	_ Transport        = (*LoopbackTransport)(nil)
	_ PollingTransport = (*LoopbackTransport)(nil)
)

// LoopbackOption configures a LoopbackTransport instance.
type LoopbackOption func(*LoopbackTransport)

// WithLocalPlayer sets the initial local player identity.
func WithLocalPlayer(p domain.Player) LoopbackOption {
	return func(lt *LoopbackTransport) {
		if p.IsValid() {
			lt.localPlayer = p
		}
	}
}

// WithBufferSize configures the capacity of the internal buffered channel.
func WithBufferSize(size int) LoopbackOption {
	return func(lt *LoopbackTransport) {
		if size > 0 {
			lt.bufferSize = size
		}
	}
}

// WithAutoPlayerToggle enables automatic alternation of localPlayer
// upon sending an EndActivation command (enabling seamless local hotseat play).
func WithAutoPlayerToggle(enabled bool) LoopbackOption {
	return func(lt *LoopbackTransport) {
		lt.autoToggle = enabled
	}
}

// WithValidateOnSend enables pre-transmission syntactic validation of GameCommand.
func WithValidateOnSend(validate bool) LoopbackOption {
	return func(lt *LoopbackTransport) {
		lt.validateOnSend = validate
	}
}

// WithBlockingSend configures Send to block when the buffer is full instead of
// returning ErrQueueFull. (Useful for batch tests; discouraged in 60 TPS game loops).
func WithBlockingSend(blocking bool) LoopbackOption {
	return func(lt *LoopbackTransport) {
		lt.blockingSend = blocking
	}
}

// LoopbackTransport routes outgoing GameCommands directly into an internal
// incoming channel, stamping each with the local player identity.
// It is fully thread-safe for concurrent sends, receives, and lifecycle operations.
type LoopbackTransport struct {
	mu             sync.RWMutex
	localPlayer    domain.Player
	autoToggle     bool
	bufferSize     int
	validateOnSend bool
	blockingSend   bool

	inCh      chan commands.NetworkCommand
	isClosed  bool
	closeOnce sync.Once
	done      chan struct{}

	peer *LoopbackTransport
}

// NewLoopback constructs an initialized, thread-safe LoopbackTransport.
// It requires a valid participant identity (PlayerOne or PlayerTwo).
func NewLoopback(player domain.Player, opts ...LoopbackOption) (*LoopbackTransport, error) {
	if !player.IsValid() {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPlayer, player)
	}

	lt := &LoopbackTransport{
		localPlayer:    player,
		bufferSize:     DefaultBufferSize,
		validateOnSend: true,
		done:           make(chan struct{}),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(lt)
		}
	}

	lt.inCh = make(chan commands.NetworkCommand, lt.bufferSize)
	return lt, nil
}

// NewLoopbackPair creates two interconnected LoopbackTransports representing
// PlayerOne and PlayerTwo. Commands sent by P1 are delivered to P2's inbound channel
// stamped with PlayerOne, and commands sent by P2 are delivered to P1's inbound channel
// stamped with PlayerTwo.
func NewLoopbackPair(opts ...LoopbackOption) (*LoopbackTransport, *LoopbackTransport, error) {
	p1, err := NewLoopback(domain.PlayerOne, opts...)
	if err != nil {
		return nil, nil, err
	}
	p2, err := NewLoopback(domain.PlayerTwo, opts...)
	if err != nil {
		_ = p1.Close()
		return nil, nil, err
	}

	p1.peer = p2
	p2.peer = p1

	return p1, p2, nil
}

// Send validates the command, stamps it with local player identity,
// and pushes it into the incoming channel.
// Uses fine-grained locking to prevent sending on a closed channel.
func (lt *LoopbackTransport) Send(cmd commands.GameCommand) error {
	if lt.validateOnSend {
		if err := cmd.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidCommand, err)
		}
	}

	lt.mu.Lock()
	if lt.isClosed {
		lt.mu.Unlock()
		return ErrTransportClosed
	}

	if !lt.localPlayer.IsValid() {
		lt.mu.Unlock()
		return ErrInvalidPlayer
	}

	sender := lt.localPlayer
	netCmd := commands.NewNetworkCommand(sender, cmd)
	shouldToggle := lt.autoToggle && cmd.Type == commands.CommandEndActivation

	peer := lt.peer
	inCh := lt.inCh
	done := lt.done
	blocking := lt.blockingSend
	lt.mu.Unlock()

	var sendErr error

	// If paired transport is active, deliver to peer's inbound channel
	if peer != nil {
		sendErr = lt.sendToTarget(peer, netCmd, done, blocking)
	} else {
		// Single loopback: deliver to own inbound channel
		sendErr = lt.sendToChan(inCh, netCmd, done, blocking)
	}

	// Only toggle active player AFTER successful delivery
	if sendErr == nil && shouldToggle {
		lt.mu.Lock()
		lt.localPlayer = lt.localPlayer.Next()
		lt.mu.Unlock()
	}

	return sendErr
}

// sendToTarget delivers a command to a peer's inbound channel.
// Holds peer.mu.RLock through the channel operation to prevent Close from
// closing the channel concurrently (which would cause a race).
func (lt *LoopbackTransport) sendToTarget(peer *LoopbackTransport, netCmd commands.NetworkCommand, localDone chan struct{}, blocking bool) error {
	peer.mu.RLock()
	defer peer.mu.RUnlock()

	if peer.isClosed {
		return ErrTransportClosed
	}

	if blocking {
		select {
		case peer.inCh <- netCmd:
			return nil
		case <-peer.done:
			return ErrTransportClosed
		case <-localDone:
			return ErrTransportClosed
		}
	}

	select {
	case peer.inCh <- netCmd:
		return nil
	default:
		return ErrQueueFull
	}
}

// sendToChan delivers a command to a channel.
// Used for single-loopback mode where the target is our own inCh.
// Holds lt.mu.RLock through the channel operation to prevent Close from
// closing the channel concurrently.
func (lt *LoopbackTransport) sendToChan(ch chan commands.NetworkCommand, netCmd commands.NetworkCommand, done chan struct{}, blocking bool) error {
	lt.mu.RLock()
	defer lt.mu.RUnlock()

	if lt.isClosed {
		return ErrTransportClosed
	}

	if blocking {
		select {
		case ch <- netCmd:
			return nil
		case <-done:
			return ErrTransportClosed
		}
	}

	select {
	case ch <- netCmd:
		return nil
	default:
		return ErrQueueFull
	}
}

// SendBatch transmits a slice of GameCommands sequentially.
// An empty slice is a safe no-op returning nil.
func (lt *LoopbackTransport) SendBatch(cmds ...commands.GameCommand) error {
	if len(cmds) == 0 {
		return nil
	}
	for _, cmd := range cmds {
		if err := lt.Send(cmd); err != nil {
			return err
		}
	}
	return nil
}

// Receive returns the read-only channel delivering incoming NetworkCommands.
func (lt *LoopbackTransport) Receive() <-chan commands.NetworkCommand {
	return lt.inCh
}

// LocalPlayer returns the currently configured local player identity.
func (lt *LoopbackTransport) LocalPlayer() domain.Player {
	lt.mu.RLock()
	defer lt.mu.RUnlock()
	return lt.localPlayer
}

// RemotePlayer returns the opposing player identity.
func (lt *LoopbackTransport) RemotePlayer() domain.Player {
	return lt.LocalPlayer().Next()
}

// SetLocalPlayer dynamically updates the local player identity (e.g. for local hotseat play).
func (lt *LoopbackTransport) SetLocalPlayer(p domain.Player) error {
	if !p.IsValid() {
		return fmt.Errorf("%w: %v", ErrInvalidPlayer, p)
	}
	lt.mu.Lock()
	defer lt.mu.Unlock()
	if lt.isClosed {
		return ErrTransportClosed
	}
	lt.localPlayer = p
	return nil
}

// Close gracefully terminates the transport.
// Subsequent Send calls return ErrTransportClosed.
// Any already buffered commands remain readable from Receive() until drained.
// Close is strictly idempotent and thread-safe.
func (lt *LoopbackTransport) Close() error {
	lt.closeOnce.Do(func() {
		// Signal done FIRST (outside write lock) so that any goroutine
		// blocking in sendToTarget/sendToChan with RLock held can observe
		// <-done, return, and release RLock — preventing deadlock.
		close(lt.done)

		// Now acquire write lock to finalize shutdown.
		lt.mu.Lock()
		lt.isClosed = true
		close(lt.inCh)
		lt.mu.Unlock()
	})
	return nil
}

// IsClosed returns true if the transport has been closed.
func (lt *LoopbackTransport) IsClosed() bool {
	lt.mu.RLock()
	defer lt.mu.RUnlock()
	return lt.isClosed
}

// Poll retrieves the next available NetworkCommand without blocking.
// Returns (cmd, true) if an item was present, or (zero, false) otherwise.
func (lt *LoopbackTransport) Poll() (commands.NetworkCommand, bool) {
	select {
	case cmd, ok := <-lt.inCh:
		if !ok {
			return commands.NetworkCommand{}, false
		}
		return cmd, true
	default:
		return commands.NetworkCommand{}, false
	}
}

// Drain extracts all currently buffered NetworkCommands in a single slice without blocking.
func (lt *LoopbackTransport) Drain() []commands.NetworkCommand {
	var result []commands.NetworkCommand
	for {
		select {
		case cmd, ok := <-lt.inCh:
			if !ok {
				return result
			}
			result = append(result, cmd)
		default:
			return result
		}
	}
}

// Len returns the number of buffered commands awaiting read.
func (lt *LoopbackTransport) Len() int {
	return len(lt.inCh)
}

// BufferLen returns the number of unread commands in the inbound buffer.
func (lt *LoopbackTransport) BufferLen() int {
	return len(lt.inCh)
}

// BufferCap returns the maximum capacity of the inbound buffer.
func (lt *LoopbackTransport) BufferCap() int {
	return lt.bufferSize
}

// Pump bridges this transport with a specific CommandQueue synchronously.
func (lt *LoopbackTransport) Pump(q *commands.CommandQueue) (int, int, error) {
	return Pump(lt, q)
}
