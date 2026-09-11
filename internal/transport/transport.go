package transport

import (
	"errors"
	"fmt"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
)

// Standard sentinel errors for transport operations.
var (
	// ErrTransportClosed is returned when attempting to operate on a closed transport.
	ErrTransportClosed = errors.New("transport is closed")

	// ErrQueueFull is returned when the transport buffer capacity is exhausted (backpressure).
	ErrQueueFull = errors.New("transport queue buffer is full (backpressure)")

	// ErrBufferFull is an alias for ErrQueueFull.
	ErrBufferFull = ErrQueueFull

	// ErrInvalidCommand is returned when command validation fails on Send.
	ErrInvalidCommand = errors.New("command validation failed")

	// ErrInvalidPlayer is returned when attempting to construct or assign an invalid player identity.
	ErrInvalidPlayer = errors.New("invalid player identity")

	// ErrNotConnected is returned when transport operation requires an active connection.
	ErrNotConnected = errors.New("transport is not connected")

	// ErrTimeout is returned when a transport operation exceeds its deadline.
	ErrTimeout = errors.New("transport operation timed out")

	// ErrNilQueue is returned when a nil CommandQueue is passed to Pump operations.
	ErrNilQueue = errors.New("command queue cannot be nil")
)

// Transport defines the bidirectional command transport layer.
// It bridges the local client input pipeline to the authoritative incoming queue.
// Implementations include LoopbackTransport for local single-machine play
// and future WebRTCTransport for peer-to-peer multiplayer via matchmaker-go.
type Transport interface {
	// Send transmits a GameCommand through the transport boundary.
	// For local loopback, it stamps the command with LocalPlayer and delivers
	// it to the inbound reception channel.
	// Returns ErrTransportClosed if closed, ErrQueueFull on backpressure,
	// or ErrInvalidCommand if command validation fails.
	Send(cmd commands.GameCommand) error

	// Receive returns a read-only channel delivering incoming NetworkCommands
	// stamped with authoritative sender identity. The channel is closed on Close().
	Receive() <-chan commands.NetworkCommand

	// LocalPlayer returns the player identifier assigned to this client endpoint.
	LocalPlayer() domain.Player

	// RemotePlayer returns the opponent player identifier (LocalPlayer().Next()).
	RemotePlayer() domain.Player

	// Close gracefully shuts down the transport and releases channel resources.
	// Calling Close must be strictly idempotent.
	Close() error

	// IsClosed returns true if the transport has been closed.
	IsClosed() bool
}

// PollingTransport is an interface extension for game engines like Ebitengine
// that prefer non-blocking polling or batch draining over channel select operations.
type PollingTransport interface {
	Transport

	// Poll retrieves the next available NetworkCommand without blocking.
	// Returns (cmd, true) if a command was available, or (zero, false) otherwise.
	Poll() (commands.NetworkCommand, bool)

	// Drain extracts all currently buffered NetworkCommands in a single batch.
	Drain() []commands.NetworkCommand

	// Len returns the number of pending incoming commands currently buffered.
	Len() int
}

// Pump is an orchestration helper that bridges a Transport with a CommandQueue:
// 1. Drains all outgoing GameCommands from q and transmits them via t.Send.
// 2. Drains all available incoming NetworkCommands from t.Receive and pushes them to q.Incoming.
// Returns the count of commands sent, received, and any transmission error encountered.
func Pump(t Transport, q *commands.CommandQueue) (sent int, received int, err error) {
	if t == nil {
		return 0, 0, errors.New("transport cannot be nil")
	}
	if q == nil {
		return 0, 0, ErrNilQueue
	}

	// Phase 1: Drain outgoing queue and transmit
	var sendError error
	outgoing := q.DrainOutgoing()
	for i, cmd := range outgoing {
		if sendErr := t.Send(cmd); sendErr != nil {
			// Re-enqueue unsent commands (including the failed one) back to outgoing
			for j := i; j < len(outgoing); j++ {
				q.PushOutgoing(outgoing[j])
			}
			sendError = fmt.Errorf("transport send failed: %w", sendErr)
			break
		}
		sent++
	}

	// Phase 2: Always drain transport incoming buffer into CommandQueue,
	// even if Phase 1 had an error — commands already sent must not be stranded.
	recvChan := t.Receive()
	if recvChan != nil {
		for {
			select {
			case netCmd, ok := <-recvChan:
				if !ok {
					if sendError != nil {
						return sent, received, sendError
					}
					return sent, received, ErrTransportClosed
				}
				q.PushIncoming(netCmd)
				received++
			default:
				goto done
			}
		}
	}

done:
	return sent, received, sendError
}
