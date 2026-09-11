package commands

import (
	"sync"

	"wildbuds/internal/domain"
)

// CommandQueue manages thread-safe, decoupled FIFO queues for outgoing client commands
// and incoming network commands awaiting authoritative resolution.
type CommandQueue struct {
	outMu    sync.Mutex
	outgoing []GameCommand

	inMu     sync.Mutex
	incoming []NetworkCommand
}

// NewCommandQueue constructs an empty, initialized CommandQueue.
func NewCommandQueue() *CommandQueue {
	return &CommandQueue{
		outgoing: make([]GameCommand, 0, 16),
		incoming: make([]NetworkCommand, 0, 16),
	}
}

// --- Outgoing Queue Operations (Produced by Input Layer) ---

// PushOutgoing appends a GameCommand to the end of the outgoing queue.
func (q *CommandQueue) PushOutgoing(cmd GameCommand) {
	q.outMu.Lock()
	defer q.outMu.Unlock()
	q.outgoing = append(q.outgoing, cmd)
}

// PushOutgoingBatch appends multiple GameCommands to the outgoing queue atomically.
func (q *CommandQueue) PushOutgoingBatch(cmds ...GameCommand) {
	if len(cmds) == 0 {
		return
	}
	q.outMu.Lock()
	defer q.outMu.Unlock()
	q.outgoing = append(q.outgoing, cmds...)
}

// PopOutgoing pops the oldest GameCommand from the front of the outgoing queue.
// Returns false if the outgoing queue is empty.
func (q *CommandQueue) PopOutgoing() (GameCommand, bool) {
	q.outMu.Lock()
	defer q.outMu.Unlock()

	if len(q.outgoing) == 0 {
		return GameCommand{}, false
	}

	cmd := q.outgoing[0]
	q.outgoing[0] = GameCommand{} // prevent stale reference retention
	q.outgoing = q.outgoing[1:]

	if len(q.outgoing) == 0 {
		if cap(q.outgoing) > 1024 {
			q.outgoing = make([]GameCommand, 0, 16)
		} else {
			q.outgoing = q.outgoing[:0]
		}
	}

	return cmd, true
}

// PeekOutgoing inspects the next outgoing command without removing it.
func (q *CommandQueue) PeekOutgoing() (GameCommand, bool) {
	q.outMu.Lock()
	defer q.outMu.Unlock()

	if len(q.outgoing) == 0 {
		return GameCommand{}, false
	}
	return q.outgoing[0], true
}

// DrainOutgoing atomically extracts all commands from the outgoing queue.
func (q *CommandQueue) DrainOutgoing() []GameCommand {
	q.outMu.Lock()
	defer q.outMu.Unlock()

	if len(q.outgoing) == 0 {
		return nil
	}

	drained := make([]GameCommand, len(q.outgoing))
	copy(drained, q.outgoing)

	for i := range q.outgoing {
		q.outgoing[i] = GameCommand{}
	}
	q.outgoing = q.outgoing[:0]

	return drained
}

// LenOutgoing returns the current number of pending outgoing commands.
func (q *CommandQueue) LenOutgoing() int {
	q.outMu.Lock()
	defer q.outMu.Unlock()
	return len(q.outgoing)
}

// IsOutgoingEmpty checks if the outgoing queue is empty.
func (q *CommandQueue) IsOutgoingEmpty() bool {
	q.outMu.Lock()
	defer q.outMu.Unlock()
	return len(q.outgoing) == 0
}

// ClearOutgoing empties the outgoing queue.
func (q *CommandQueue) ClearOutgoing() {
	q.outMu.Lock()
	defer q.outMu.Unlock()
	for i := range q.outgoing {
		q.outgoing[i] = GameCommand{}
	}
	q.outgoing = q.outgoing[:0]
}

// OutgoingSnapshot returns a copy of all current outgoing commands without mutating the queue.
func (q *CommandQueue) OutgoingSnapshot() []GameCommand {
	q.outMu.Lock()
	defer q.outMu.Unlock()

	if len(q.outgoing) == 0 {
		return nil
	}
	cp := make([]GameCommand, len(q.outgoing))
	copy(cp, q.outgoing)
	return cp
}

// --- Incoming Queue Operations (Consumed by Resolver) ---

// PushIncoming appends a NetworkCommand to the end of the incoming queue.
func (q *CommandQueue) PushIncoming(cmd NetworkCommand) {
	q.inMu.Lock()
	defer q.inMu.Unlock()
	q.incoming = append(q.incoming, cmd)
}

// PushIncomingBatch appends multiple NetworkCommands to the incoming queue atomically.
func (q *CommandQueue) PushIncomingBatch(cmds ...NetworkCommand) {
	if len(cmds) == 0 {
		return
	}
	q.inMu.Lock()
	defer q.inMu.Unlock()
	q.incoming = append(q.incoming, cmds...)
}

// PopIncoming pops the oldest NetworkCommand from the front of the incoming queue.
// Returns false if the incoming queue is empty.
func (q *CommandQueue) PopIncoming() (NetworkCommand, bool) {
	q.inMu.Lock()
	defer q.inMu.Unlock()

	if len(q.incoming) == 0 {
		return NetworkCommand{}, false
	}

	cmd := q.incoming[0]
	q.incoming[0] = NetworkCommand{} // prevent stale reference retention
	q.incoming = q.incoming[1:]

	if len(q.incoming) == 0 {
		if cap(q.incoming) > 1024 {
			q.incoming = make([]NetworkCommand, 0, 16)
		} else {
			q.incoming = q.incoming[:0]
		}
	}

	return cmd, true
}

// PeekIncoming inspects the next incoming command without removing it.
func (q *CommandQueue) PeekIncoming() (NetworkCommand, bool) {
	q.inMu.Lock()
	defer q.inMu.Unlock()

	if len(q.incoming) == 0 {
		return NetworkCommand{}, false
	}
	return q.incoming[0], true
}

// DrainIncoming atomically extracts all commands from the incoming queue.
func (q *CommandQueue) DrainIncoming() []NetworkCommand {
	q.inMu.Lock()
	defer q.inMu.Unlock()

	if len(q.incoming) == 0 {
		return nil
	}

	drained := make([]NetworkCommand, len(q.incoming))
	copy(drained, q.incoming)

	for i := range q.incoming {
		q.incoming[i] = NetworkCommand{}
	}
	q.incoming = q.incoming[:0]

	return drained
}

// LenIncoming returns the current number of pending incoming commands.
func (q *CommandQueue) LenIncoming() int {
	q.inMu.Lock()
	defer q.inMu.Unlock()
	return len(q.incoming)
}

// IsIncomingEmpty checks if the incoming queue is empty.
func (q *CommandQueue) IsIncomingEmpty() bool {
	q.inMu.Lock()
	defer q.inMu.Unlock()
	return len(q.incoming) == 0
}

// ClearIncoming empties the incoming queue.
func (q *CommandQueue) ClearIncoming() {
	q.inMu.Lock()
	defer q.inMu.Unlock()
	for i := range q.incoming {
		q.incoming[i] = NetworkCommand{}
	}
	q.incoming = q.incoming[:0]
}

// IncomingSnapshot returns a copy of all current incoming commands without mutating the queue.
func (q *CommandQueue) IncomingSnapshot() []NetworkCommand {
	q.inMu.Lock()
	defer q.inMu.Unlock()

	if len(q.incoming) == 0 {
		return nil
	}
	cp := make([]NetworkCommand, len(q.incoming))
	copy(cp, q.incoming)
	return cp
}

// --- Composite & Utility Operations ---

// Len returns total pending commands across both outgoing and incoming queues.
func (q *CommandQueue) Len() int {
	return q.LenOutgoing() + q.LenIncoming()
}

// IsEmpty returns true if both outgoing and incoming queues are empty.
func (q *CommandQueue) IsEmpty() bool {
	return q.IsOutgoingEmpty() && q.IsIncomingEmpty()
}

// Clear empties both outgoing and incoming queues.
func (q *CommandQueue) Clear() {
	q.ClearOutgoing()
	q.ClearIncoming()
}

// RouteOutgoingToIncoming implements local loopback transport behavior:
// It atomically drains all outgoing commands, wraps each in a NetworkCommand stamped
// with sender, and enqueues them into the incoming queue. Returns count routed.
func (q *CommandQueue) RouteOutgoingToIncoming(sender domain.Player) int {
	outgoing := q.DrainOutgoing()
	if len(outgoing) == 0 {
		return 0
	}

	netCmds := make([]NetworkCommand, len(outgoing))
	for i, cmd := range outgoing {
		netCmds[i] = NetworkCommand{
			Sender:  sender,
			Command: cmd,
		}
	}

	q.PushIncomingBatch(netCmds...)
	return len(netCmds)
}
