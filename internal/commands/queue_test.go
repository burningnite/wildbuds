package commands_test

import (
	"sync"
	"testing"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
)

func TestCommandQueue_FIFO_Order(t *testing.T) {
	q := commands.NewCommandQueue()

	if !q.IsEmpty() || q.Len() != 0 {
		t.Fatalf("expected new queue to be empty")
	}

	// Enqueue 3 outgoing commands
	c1 := commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3})
	c2 := commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: -2})
	c3 := commands.NewEndActivationCommand()

	q.PushOutgoing(c1)
	q.PushOutgoing(c2)
	q.PushOutgoing(c3)

	if q.LenOutgoing() != 3 {
		t.Errorf("expected LenOutgoing == 3, got %d", q.LenOutgoing())
	}
	if q.Len() != 3 {
		t.Errorf("expected total Len == 3, got %d", q.Len())
	}

	// Peek
	peeked, ok := q.PeekOutgoing()
	if !ok || !peeked.Equals(c1) {
		t.Errorf("expected peek to yield c1, got %v", peeked)
	}

	// Snapshot
	snap := q.OutgoingSnapshot()
	if len(snap) != 3 {
		t.Errorf("expected snapshot length 3, got %d", len(snap))
	}

	// Pop sequentially
	p1, ok1 := q.PopOutgoing()
	p2, ok2 := q.PopOutgoing()
	p3, ok3 := q.PopOutgoing()
	p4, ok4 := q.PopOutgoing()

	if !ok1 || !p1.Equals(c1) {
		t.Errorf("pop 1 failed: expected %v, got %v", c1, p1)
	}
	if !ok2 || !p2.Equals(c2) {
		t.Errorf("pop 2 failed: expected %v, got %v", c2, p2)
	}
	if !ok3 || !p3.Equals(c3) {
		t.Errorf("pop 3 failed: expected %v, got %v", c3, p3)
	}
	if ok4 {
		t.Errorf("expected 4th pop to return false on empty queue, got %v", p4)
	}

	if !q.IsOutgoingEmpty() {
		t.Errorf("expected outgoing to be empty after popping all")
	}

	// Peek empty
	if _, ok := q.PeekOutgoing(); ok {
		t.Errorf("expected peek to return false on empty queue")
	}

	// Snapshot empty
	if snapEmpty := q.OutgoingSnapshot(); snapEmpty != nil {
		t.Errorf("expected nil snapshot on empty queue, got %v", snapEmpty)
	}
}

func TestCommandQueue_Incoming_FIFO_Order(t *testing.T) {
	q := commands.NewCommandQueue()

	nc1 := commands.NewNetworkCommand(domain.PlayerOne, commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3}))
	nc2 := commands.NewNetworkCommand(domain.PlayerOne, commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: -2}))

	q.PushIncoming(nc1)
	q.PushIncoming(nc2)

	if q.LenIncoming() != 2 {
		t.Errorf("expected LenIncoming == 2, got %d", q.LenIncoming())
	}

	peeked, ok := q.PeekIncoming()
	if !ok || !peeked.Equals(nc1) {
		t.Errorf("expected peek incoming to yield nc1")
	}

	snap := q.IncomingSnapshot()
	if len(snap) != 2 {
		t.Errorf("expected incoming snapshot length 2, got %d", len(snap))
	}

	p1, ok1 := q.PopIncoming()
	p2, ok2 := q.PopIncoming()
	p3, ok3 := q.PopIncoming()

	if !ok1 || !p1.Equals(nc1) {
		t.Errorf("pop incoming 1 failed")
	}
	if !ok2 || !p2.Equals(nc2) {
		t.Errorf("pop incoming 2 failed")
	}
	if ok3 {
		t.Errorf("expected 3rd pop incoming to return false, got %v", p3)
	}

	if !q.IsIncomingEmpty() {
		t.Errorf("expected incoming to be empty")
	}

	// Peek empty incoming
	if _, ok := q.PeekIncoming(); ok {
		t.Errorf("expected peek incoming false on empty")
	}

	// Snapshot empty incoming
	if snapEmpty := q.IncomingSnapshot(); snapEmpty != nil {
		t.Errorf("expected nil snapshot on empty incoming")
	}
}

func TestCommandQueue_BatchesAndDraining(t *testing.T) {
	q := commands.NewCommandQueue()

	// Empty drain
	if drained := q.DrainOutgoing(); drained != nil {
		t.Errorf("expected nil on draining empty outgoing")
	}
	if drained := q.DrainIncoming(); drained != nil {
		t.Errorf("expected nil on draining empty incoming")
	}

	// Batch outgoing push
	c1 := commands.NewSelectUnitCommand(domain.GridPosition{X: 1, Y: 1})
	c2 := commands.NewEndActivationCommand()
	q.PushOutgoingBatch(c1, c2)
	q.PushOutgoingBatch() // Empty batch no-op

	if q.LenOutgoing() != 2 {
		t.Fatalf("expected 2 outgoing, got %d", q.LenOutgoing())
	}

	drainedOut := q.DrainOutgoing()
	if len(drainedOut) != 2 || !drainedOut[0].Equals(c1) || !drainedOut[1].Equals(c2) {
		t.Fatalf("unexpected drained outgoing: %v", drainedOut)
	}
	if !q.IsOutgoingEmpty() {
		t.Errorf("expected outgoing empty after drain")
	}

	// Batch incoming push
	nc1 := commands.NewNetworkCommand(domain.PlayerOne, c1)
	nc2 := commands.NewNetworkCommand(domain.PlayerTwo, c2)
	q.PushIncomingBatch(nc1, nc2)
	q.PushIncomingBatch() // Empty batch no-op

	if q.LenIncoming() != 2 {
		t.Fatalf("expected 2 incoming, got %d", q.LenIncoming())
	}

	drainedIn := q.DrainIncoming()
	if len(drainedIn) != 2 || !drainedIn[0].Equals(nc1) || !drainedIn[1].Equals(nc2) {
		t.Fatalf("unexpected drained incoming: %v", drainedIn)
	}
	if !q.IsIncomingEmpty() {
		t.Errorf("expected incoming empty after drain")
	}
}

func TestCommandQueue_DrainAndLoopback(t *testing.T) {
	q := commands.NewCommandQueue()

	c1 := commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3})
	c2 := commands.NewEndActivationCommand()

	q.PushOutgoing(c1)
	q.PushOutgoing(c2)

	// Route Outgoing to Incoming via loopback
	routed := q.RouteOutgoingToIncoming(domain.PlayerOne)
	if routed != 2 {
		t.Fatalf("expected 2 routed commands, got %d", routed)
	}

	// Routing empty outgoing returns 0
	if zeroRouted := q.RouteOutgoingToIncoming(domain.PlayerOne); zeroRouted != 0 {
		t.Fatalf("expected 0 routed commands, got %d", zeroRouted)
	}

	if !q.IsOutgoingEmpty() {
		t.Errorf("expected outgoing to be empty after loopback")
	}
	if q.LenIncoming() != 2 {
		t.Fatalf("expected 2 incoming commands, got %d", q.LenIncoming())
	}

	// Drain incoming
	incoming := q.DrainIncoming()
	if len(incoming) != 2 {
		t.Fatalf("expected 2 drained incoming, got %d", len(incoming))
	}
	if incoming[0].Sender != domain.PlayerOne || !incoming[0].Command.Equals(c1) {
		t.Errorf("mismatch on incoming[0]: %v", incoming[0])
	}
	if incoming[1].Sender != domain.PlayerOne || !incoming[1].Command.Equals(c2) {
		t.Errorf("mismatch on incoming[1]: %v", incoming[1])
	}

	if !q.IsEmpty() {
		t.Errorf("expected queue to be completely empty")
	}
}

func TestCommandQueue_Clear(t *testing.T) {
	q := commands.NewCommandQueue()

	q.PushOutgoing(commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: 0}))
	q.PushIncoming(commands.NewNetworkCommand(domain.PlayerOne, commands.NewEndActivationCommand()))

	if q.IsEmpty() {
		t.Fatalf("expected non-empty queue")
	}

	q.ClearOutgoing()
	if !q.IsOutgoingEmpty() || q.IsIncomingEmpty() {
		t.Errorf("expected outgoing cleared but incoming preserved")
	}

	q.ClearIncoming()
	if !q.IsEmpty() {
		t.Errorf("expected queue completely empty")
	}

	// Combined Clear
	q.PushOutgoing(commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: 0}))
	q.PushIncoming(commands.NewNetworkCommand(domain.PlayerOne, commands.NewEndActivationCommand()))
	q.Clear()
	if !q.IsEmpty() {
		t.Errorf("expected queue completely empty after Clear()")
	}
}

func TestCommandQueue_LargeCapacityReset(t *testing.T) {
	q := commands.NewCommandQueue()

	// Push 2000 items to expand capacity past 1024
	for i := 0; i < 2000; i++ {
		q.PushOutgoing(commands.NewEndActivationCommand())
		q.PushIncoming(commands.NewNetworkCommand(domain.PlayerOne, commands.NewEndActivationCommand()))
	}

	// Pop all
	for i := 0; i < 2000; i++ {
		q.PopOutgoing()
		q.PopIncoming()
	}

	if !q.IsEmpty() {
		t.Errorf("expected queue to be empty after popping all 2000 items")
	}
}

func TestCommandQueue_ConcurrencyStress(t *testing.T) {
	q := commands.NewCommandQueue()
	const numProducers = 10
	const numConsumers = 10
	const itemsPerProducer = 100

	var wg sync.WaitGroup
	wg.Add(numProducers * 2) // outgoing producers + incoming producers

	// Outgoing Producers
	for i := 0; i < numProducers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < itemsPerProducer; j++ {
				q.PushOutgoing(commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: id%5}))
			}
		}(i)
	}

	// Incoming Producers
	for i := 0; i < numProducers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < itemsPerProducer; j++ {
				q.PushIncoming(commands.NewNetworkCommand(
					domain.PlayerOne,
					commands.NewEndActivationCommand(),
				))
			}
		}(i)
	}

	// Consumers running concurrently
	var popWg sync.WaitGroup
	var poppedOutgoingCount int64
	var poppedIncomingCount int64
	var countMu sync.Mutex

	popWg.Add(numConsumers * 2)

	for i := 0; i < numConsumers; i++ {
		go func() {
			defer popWg.Done()
			for {
				if _, ok := q.PopOutgoing(); ok {
					countMu.Lock()
					poppedOutgoingCount++
					countMu.Unlock()
				} else {
					countMu.Lock()
					done := poppedOutgoingCount == int64(numProducers*itemsPerProducer)
					countMu.Unlock()
					if done {
						break
					}
				}
			}
		}()

		go func() {
			defer popWg.Done()
			for {
				if _, ok := q.PopIncoming(); ok {
					countMu.Lock()
					poppedIncomingCount++
					countMu.Unlock()
				} else {
					countMu.Lock()
					done := poppedIncomingCount == int64(numProducers*itemsPerProducer)
					countMu.Unlock()
					if done {
						break
					}
				}
			}
		}()
	}

	wg.Wait()
	popWg.Wait()

	if poppedOutgoingCount != int64(numProducers*itemsPerProducer) {
		t.Errorf("expected %d outgoing popped, got %d", numProducers*itemsPerProducer, poppedOutgoingCount)
	}
	if poppedIncomingCount != int64(numProducers*itemsPerProducer) {
		t.Errorf("expected %d incoming popped, got %d", numProducers*itemsPerProducer, poppedIncomingCount)
	}
	if !q.IsEmpty() {
		t.Errorf("expected queue to be empty after stress test, len=%d", q.Len())
	}
}
