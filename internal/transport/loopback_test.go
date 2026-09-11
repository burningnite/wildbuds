package transport_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
	"wildbuds/internal/resolver"
	"wildbuds/internal/transport"
)

// ============================================================================
// Group 1: Initialization & Validation Tests
// ============================================================================

func TestLoopback_New_ValidPlayers(t *testing.T) {
	for _, p := range []domain.Player{domain.PlayerOne, domain.PlayerTwo} {
		lt, err := transport.NewLoopback(p)
		if err != nil {
			t.Fatalf("expected NewLoopback(%s) to succeed, got %v", p, err)
		}
		if lt.LocalPlayer() != p {
			t.Errorf("expected LocalPlayer %s, got %s", p, lt.LocalPlayer())
		}
		if lt.RemotePlayer() != p.Next() {
			t.Errorf("expected RemotePlayer %s, got %s", p.Next(), lt.RemotePlayer())
		}
		if lt.IsClosed() {
			t.Errorf("new transport should not be closed")
		}
		if err := lt.Close(); err != nil {
			t.Errorf("Close failed: %v", err)
		}
	}
}

func TestLoopback_New_InvalidPlayer(t *testing.T) {
	invalidPlayers := []domain.Player{
		domain.PlayerNone,
		domain.Player(99),
		domain.Player(-1),
	}

	for _, p := range invalidPlayers {
		lt, err := transport.NewLoopback(p)
		if err == nil {
			lt.Close()
			t.Errorf("expected error for NewLoopback with invalid player %v, got nil", p)
		}
		if !errors.Is(err, transport.ErrInvalidPlayer) {
			t.Errorf("expected ErrInvalidPlayer for player %v, got %v", p, err)
		}
	}
}

// ============================================================================
// Group 2: Stamping & Strict FIFO Delivery Tests
// ============================================================================

func TestLoopback_SendReceive_FIFO(t *testing.T) {
	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(64))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	testCmds := []commands.GameCommand{
		commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3}),
		commands.NewMoveUnitCommand(domain.GridPosition{X: 1, Y: -3}),
		commands.NewAttackCommand(domain.GridPosition{X: 1, Y: 0}),
		commands.NewEndActivationCommand(),
		commands.NewSelectUnitCommand(domain.GridPosition{X: -2, Y: 2}),
	}

	// Send all commands
	for i, cmd := range testCmds {
		if err := lt.Send(cmd); err != nil {
			t.Fatalf("failed to send command %d (%s): %v", i, cmd, err)
		}
	}

	// Receive all commands and verify strict FIFO sequence and payload equality
	for i, expected := range testCmds {
		select {
		case netCmd, ok := <-lt.Receive():
			if !ok {
				t.Fatalf("receive channel closed prematurely at index %d", i)
			}
			if netCmd.Sender != domain.PlayerOne {
				t.Errorf("[%d] expected sender %s, got %s", i, domain.PlayerOne, netCmd.Sender)
			}
			if !netCmd.Command.Equals(expected) {
				t.Errorf("[%d] command payload mismatch: expected %v, got %v", i, expected, netCmd.Command)
			}
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("timeout waiting for command index %d", i)
		}
	}
}

func TestLoopback_PlayerStamping_PlayerTwo(t *testing.T) {
	lt, err := transport.NewLoopback(domain.PlayerTwo, transport.WithBufferSize(16))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	cmd := commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: 3})
	if err := lt.Send(cmd); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	select {
	case netCmd := <-lt.Receive():
		if netCmd.Sender != domain.PlayerTwo {
			t.Errorf("expected Sender %s, got %s", domain.PlayerTwo, netCmd.Sender)
		}
		if netCmd.Command.Target.Y != 3 {
			t.Errorf("expected target Y 3, got %d", netCmd.Command.Target.Y)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timeout waiting for stamped command")
	}
}

// ============================================================================
// Group 3: Batch Operations Tests
// ============================================================================

func TestLoopback_SendBatch_FIFO(t *testing.T) {
	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(32))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	batch := []commands.GameCommand{
		commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3}),
		commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: -2}),
		commands.NewEndActivationCommand(),
	}

	if err := lt.SendBatch(batch...); err != nil {
		t.Fatalf("SendBatch failed: %v", err)
	}

	for i, expected := range batch {
		select {
		case netCmd := <-lt.Receive():
			if netCmd.Sender != domain.PlayerOne {
				t.Errorf("[%d] expected sender %s, got %s", i, domain.PlayerOne, netCmd.Sender)
			}
			if !netCmd.Command.Equals(expected) {
				t.Errorf("[%d] expected %v, got %v", i, expected, netCmd.Command)
			}
		case <-time.After(200 * time.Millisecond):
			t.Fatalf("timeout waiting for batch item %d", i)
		}
	}
}

func TestLoopback_SendBatch_Empty(t *testing.T) {
	lt, err := transport.NewLoopback(domain.PlayerOne)
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	// Empty slice
	if err := lt.SendBatch(); err != nil {
		t.Errorf("expected empty SendBatch to return nil, got %v", err)
	}

	// Nil slice
	if err := lt.SendBatch([]commands.GameCommand(nil)...); err != nil {
		t.Errorf("expected nil SendBatch to return nil, got %v", err)
	}

	// Verify nothing was pushed
	select {
	case item := <-lt.Receive():
		t.Errorf("expected empty channel, got unexpected item: %v", item)
	default:
		// Success
	}
}

// ============================================================================
// Group 4: Lifecycle & Idempotency Tests
// ============================================================================

func TestLoopback_Close_Idempotent(t *testing.T) {
	lt, err := transport.NewLoopback(domain.PlayerOne)
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}

	// First Close
	if err := lt.Close(); err != nil {
		t.Errorf("first Close() returned error: %v", err)
	}
	if !lt.IsClosed() {
		t.Errorf("expected IsClosed() == true after Close()")
	}

	// Second Close (must not panic and return nil)
	if err := lt.Close(); err != nil {
		t.Errorf("second Close() returned error: %v", err)
	}

	// Third Close
	if err := lt.Close(); err != nil {
		t.Errorf("third Close() returned error: %v", err)
	}
}

func TestLoopback_Send_AfterClose(t *testing.T) {
	lt, err := transport.NewLoopback(domain.PlayerOne)
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}

	if err := lt.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	cmd := commands.NewEndActivationCommand()

	// Single Send after close
	err = lt.Send(cmd)
	if err == nil {
		t.Errorf("expected error on Send() after Close(), got nil")
	}
	if !errors.Is(err, transport.ErrTransportClosed) {
		t.Errorf("expected ErrTransportClosed, got %v", err)
	}

	// Batch Send after close
	err = lt.SendBatch(cmd, cmd)
	if err == nil {
		t.Errorf("expected error on SendBatch() after Close(), got nil")
	}
	if !errors.Is(err, transport.ErrTransportClosed) {
		t.Errorf("expected ErrTransportClosed, got %v", err)
	}
}

func TestLoopback_Receive_DrainAfterClose(t *testing.T) {
	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(16))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}

	// Send 3 commands before closing
	c1 := commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3})
	c2 := commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: -2})
	c3 := commands.NewEndActivationCommand()

	if err := lt.Send(c1); err != nil {
		t.Fatalf("Send c1 failed: %v", err)
	}
	if err := lt.Send(c2); err != nil {
		t.Fatalf("Send c2 failed: %v", err)
	}
	if err := lt.Send(c3); err != nil {
		t.Fatalf("Send c3 failed: %v", err)
	}

	// Close the transport
	if err := lt.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// The 3 sent items must still be readable
	r1, ok1 := <-lt.Receive()
	if !ok1 || !r1.Command.Equals(c1) {
		t.Errorf("failed to read c1 after close: ok=%v, cmd=%v", ok1, r1)
	}

	r2, ok2 := <-lt.Receive()
	if !ok2 || !r2.Command.Equals(c2) {
		t.Errorf("failed to read c2 after close: ok=%v, cmd=%v", ok2, r2)
	}

	r3, ok3 := <-lt.Receive()
	if !ok3 || !r3.Command.Equals(c3) {
		t.Errorf("failed to read c3 after close: ok=%v, cmd=%v", ok3, r3)
	}

	// After buffered items are drained, receive channel must signal EOF (closed)
	_, ok4 := <-lt.Receive()
	if ok4 {
		t.Errorf("expected receive channel to be closed after draining buffered items, but ok was true")
	}
}

// ============================================================================
// Group 5: Non-Blocking Drain, Poll & CommandQueue Pump Tests
// ============================================================================

func TestLoopback_Drain_NonBlocking(t *testing.T) {
	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(16))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	// Drain empty
	drained := lt.Drain()
	if len(drained) != 0 {
		t.Errorf("expected 0 drained items, got %d", len(drained))
	}

	// Send 4 commands
	for i := 0; i < 4; i++ {
		if err := lt.Send(commands.NewEndActivationCommand()); err != nil {
			t.Fatalf("Send %d failed: %v", i, err)
		}
	}

	if lt.Len() != 4 || lt.BufferLen() != 4 {
		t.Errorf("expected len 4, got Len=%d BufferLen=%d", lt.Len(), lt.BufferLen())
	}
	if lt.BufferCap() != 16 {
		t.Errorf("expected cap 16, got %d", lt.BufferCap())
	}

	drained = lt.Drain()
	if len(drained) != 4 {
		t.Errorf("expected 4 drained items, got %d", len(drained))
	}

	// Drain immediately again
	drainedAgain := lt.Drain()
	if len(drainedAgain) != 0 {
		t.Errorf("expected 0 items on second drain, got %d", len(drainedAgain))
	}
}

func TestLoopback_Poll_PollingTransport(t *testing.T) {
	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(8))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	// Empty poll
	cmd, ok := lt.Poll()
	if ok {
		t.Errorf("expected poll on empty transport to return false, got true: %v", cmd)
	}

	// Send one command
	sendCmd := commands.NewEndActivationCommand()
	if err := lt.Send(sendCmd); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Poll again
	cmd, ok = lt.Poll()
	if !ok {
		t.Fatalf("expected poll to succeed, got false")
	}
	if !cmd.Command.Equals(sendCmd) {
		t.Errorf("expected command %v, got %v", sendCmd, cmd.Command)
	}

	// Poll again -> empty
	cmd, ok = lt.Poll()
	if ok {
		t.Errorf("expected poll to be empty, got %v", cmd)
	}
}

func TestLoopback_Pump_CommandQueue(t *testing.T) {
	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(32))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	q := commands.NewCommandQueue()

	// Push 3 commands to Outgoing queue
	q.PushOutgoing(commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3}))
	q.PushOutgoing(commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: -2}))
	q.PushOutgoing(commands.NewEndActivationCommand())

	if q.LenOutgoing() != 3 {
		t.Fatalf("expected 3 outgoing commands, got %d", q.LenOutgoing())
	}

	// Pump: transfers outgoing -> transport.Send -> transport.Receive -> incoming
	sent, received, err := lt.Pump(q)
	if err != nil {
		t.Fatalf("Pump failed: %v", err)
	}

	if sent != 3 {
		t.Errorf("expected 3 sent, got %d", sent)
	}
	if received != 3 {
		t.Errorf("expected 3 received, got %d", received)
	}

	if q.LenOutgoing() != 0 {
		t.Errorf("expected outgoing queue to be emptied, got %d", q.LenOutgoing())
	}
	if q.LenIncoming() != 3 {
		t.Errorf("expected incoming queue to have 3 items, got %d", q.LenIncoming())
	}

	// Verify incoming items are stamped with PlayerOne
	for i := 0; i < 3; i++ {
		cmd, ok := q.PopIncoming()
		if !ok {
			t.Fatalf("failed to pop incoming item %d", i)
		}
		if cmd.Sender != domain.PlayerOne {
			t.Errorf("[%d] expected sender PlayerOne, got %s", i, cmd.Sender)
		}
	}
}

// ============================================================================
// Group 6: Auto Toggle, Player Setting & Backpressure Tests
// ============================================================================

func TestLoopback_AutoPlayerToggle(t *testing.T) {
	lt, err := transport.NewLoopback(
		domain.PlayerOne,
		transport.WithAutoPlayerToggle(true),
		transport.WithBufferSize(16),
	)
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	// Initially PlayerOne
	if lt.LocalPlayer() != domain.PlayerOne {
		t.Fatalf("expected PlayerOne, got %s", lt.LocalPlayer())
	}

	// Send SelectUnit -> should still be PlayerOne
	if err := lt.Send(commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3})); err != nil {
		t.Fatalf("Send failed: %v", err)
	}
	if lt.LocalPlayer() != domain.PlayerOne {
		t.Errorf("expected PlayerOne before EndActivation, got %s", lt.LocalPlayer())
	}

	// Send EndActivation -> should toggle to PlayerTwo
	if err := lt.Send(commands.NewEndActivationCommand()); err != nil {
		t.Fatalf("Send EndActivation failed: %v", err)
	}
	if lt.LocalPlayer() != domain.PlayerTwo {
		t.Errorf("expected PlayerTwo after EndActivation, got %s", lt.LocalPlayer())
	}

	// Send EndActivation again -> should toggle back to PlayerOne
	if err := lt.Send(commands.NewEndActivationCommand()); err != nil {
		t.Fatalf("Send EndActivation 2 failed: %v", err)
	}
	if lt.LocalPlayer() != domain.PlayerOne {
		t.Errorf("expected PlayerOne after second EndActivation, got %s", lt.LocalPlayer())
	}
}

func TestLoopback_SetLocalPlayer(t *testing.T) {
	lt, err := transport.NewLoopback(domain.PlayerOne)
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	if err := lt.SetLocalPlayer(domain.PlayerTwo); err != nil {
		t.Fatalf("SetLocalPlayer failed: %v", err)
	}
	if lt.LocalPlayer() != domain.PlayerTwo {
		t.Errorf("expected PlayerTwo, got %s", lt.LocalPlayer())
	}

	// Setting invalid player returns ErrInvalidPlayer
	if err := lt.SetLocalPlayer(domain.PlayerNone); !errors.Is(err, transport.ErrInvalidPlayer) {
		t.Errorf("expected ErrInvalidPlayer, got %v", err)
	}

	// Setting after close returns ErrTransportClosed
	lt.Close()
	if err := lt.SetLocalPlayer(domain.PlayerOne); !errors.Is(err, transport.ErrTransportClosed) {
		t.Errorf("expected ErrTransportClosed on SetLocalPlayer after close, got %v", err)
	}
}

func TestLoopback_QueueFull_Backpressure(t *testing.T) {
	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(2))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	// Fill buffer of size 2
	if err := lt.Send(commands.NewEndActivationCommand()); err != nil {
		t.Fatalf("send 1 failed: %v", err)
	}
	if err := lt.Send(commands.NewEndActivationCommand()); err != nil {
		t.Fatalf("send 2 failed: %v", err)
	}

	// Third send must fail with ErrQueueFull
	err = lt.Send(commands.NewEndActivationCommand())
	if !errors.Is(err, transport.ErrQueueFull) {
		t.Errorf("expected ErrQueueFull on backpressure, got %v", err)
	}
}

// ============================================================================
// Group 7: Concurrency Stress & Race Detection
// ============================================================================

func TestLoopback_Concurrent_MPSC_Stress(t *testing.T) {
	const (
		numProducers      = 16
		cmdsPerProducer   = 100
		totalExpectedCmds = numProducers * cmdsPerProducer
		bufferCap         = 2048
	)

	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(bufferCap))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	var wgProducers sync.WaitGroup
	var receivedCount int64
	var wgConsumer sync.WaitGroup

	// Consumer goroutine
	stopConsumer := make(chan struct{})
	wgConsumer.Add(1)
	go func() {
		defer wgConsumer.Done()
		for {
			select {
			case netCmd, ok := <-lt.Receive():
				if !ok {
					return
				}
				if netCmd.Sender != domain.PlayerOne {
					t.Errorf("corrupted sender: %v", netCmd.Sender)
				}
				atomic.AddInt64(&receivedCount, 1)
			case <-stopConsumer:
				// Drain any remaining
				for {
					select {
					case netCmd, ok := <-lt.Receive():
						if !ok {
							return
						}
						if netCmd.Sender != domain.PlayerOne {
							t.Errorf("corrupted sender: %v", netCmd.Sender)
						}
						atomic.AddInt64(&receivedCount, 1)
					default:
						return
					}
				}
			}
		}
	}()

	// Spawn producers
	wgProducers.Add(numProducers)
	for p := 0; p < numProducers; p++ {
		go func(producerID int) {
			defer wgProducers.Done()
			for i := 0; i < cmdsPerProducer; i++ {
				cmd := commands.NewMoveUnitCommand(domain.GridPosition{X: producerID % 5, Y: (i % 5)})
				if err := lt.Send(cmd); err != nil {
					t.Errorf("producer %d send error: %v", producerID, err)
				}
			}
		}(p)
	}

	wgProducers.Wait()

	// Allow consumer to catch up
	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt64(&receivedCount) < int64(totalExpectedCmds) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	close(stopConsumer)
	wgConsumer.Wait()

	actual := atomic.LoadInt64(&receivedCount)
	if actual != int64(totalExpectedCmds) {
		t.Fatalf("MPSC message count mismatch: expected %d, got %d", totalExpectedCmds, actual)
	}
}

func TestLoopback_Concurrent_CloseRace(t *testing.T) {
	const numGoroutines = 50
	lt, err := transport.NewLoopback(domain.PlayerTwo, transport.WithBufferSize(256))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Half goroutines try to Close()
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = lt.Close()
		}()
	}

	// Half goroutines try to Send()
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			err := lt.Send(commands.NewEndActivationCommand())
			if err != nil && !errors.Is(err, transport.ErrTransportClosed) {
				t.Errorf("unexpected error on send during close race: %v", err)
			}
		}()
	}

	wg.Wait()

	if !lt.IsClosed() {
		t.Errorf("expected transport to be closed after close race")
	}
}

// ============================================================================
// Group 8: Paired Multi-Client Simulation & Full Pipeline E2E
// ============================================================================

func TestLoopback_Paired_MultiplayerSim(t *testing.T) {
	p1Transport, p2Transport, err := transport.NewLoopbackPair()
	if err != nil {
		t.Fatalf("NewLoopbackPair failed: %v", err)
	}
	defer p1Transport.Close()
	defer p2Transport.Close()

	if p1Transport.LocalPlayer() != domain.PlayerOne {
		t.Errorf("expected p1Transport local player PlayerOne, got %s", p1Transport.LocalPlayer())
	}
	if p2Transport.LocalPlayer() != domain.PlayerTwo {
		t.Errorf("expected p2Transport local player PlayerTwo, got %s", p2Transport.LocalPlayer())
	}

	// P1 sends a command -> P2 receives stamped with PlayerOne
	p1Cmd := commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3})
	if err := p1Transport.Send(p1Cmd); err != nil {
		t.Fatalf("P1 Send failed: %v", err)
	}

	select {
	case netCmd := <-p2Transport.Receive():
		if netCmd.Sender != domain.PlayerOne {
			t.Errorf("P2 received command with wrong sender: expected %s, got %s", domain.PlayerOne, netCmd.Sender)
		}
		if !netCmd.Command.Equals(p1Cmd) {
			t.Errorf("P2 received command payload mismatch: expected %v, got %v", p1Cmd, netCmd.Command)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timeout waiting for P2 to receive P1's command")
	}

	// P2 sends a command -> P1 receives stamped with PlayerTwo
	p2Cmd := commands.NewEndActivationCommand()
	if err := p2Transport.Send(p2Cmd); err != nil {
		t.Fatalf("P2 Send failed: %v", err)
	}

	select {
	case netCmd := <-p1Transport.Receive():
		if netCmd.Sender != domain.PlayerTwo {
			t.Errorf("P1 received command with wrong sender: expected %s, got %s", domain.PlayerTwo, netCmd.Sender)
		}
		if !netCmd.Command.Equals(p2Cmd) {
			t.Errorf("P1 received command payload mismatch: expected %v, got %v", p2Cmd, netCmd.Command)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timeout waiting for P1 to receive P2's command")
	}
}

func TestLoopback_EndToEnd_WithAuthoritativeResolver(t *testing.T) {
	// Full Pipeline Verification:
	// Outgoing Input -> Loopback Transport -> Incoming Queue -> Resolver -> State Mutation
	state := domain.NewInitialGameState()
	q := commands.NewCommandQueue()
	r := resolver.NewResolver()

	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(32))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	// Initial preconditions
	if state.ActivePlayer != domain.PlayerOne {
		t.Fatalf("expected initial active player PlayerOne, got %s", state.ActivePlayer)
	}
	if state.ActiveUnitID != nil {
		t.Fatalf("expected initial ActiveUnitID nil")
	}

	// 1. Client simulates user selecting unit at (0, -3)
	selectCmd := commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3})
	q.PushOutgoing(selectCmd)

	// 2. Transport pumps outgoing to incoming
	sent, recv, err := lt.Pump(q)
	if err != nil {
		t.Fatalf("Pump failed: %v", err)
	}
	if sent != 1 || recv != 1 {
		t.Fatalf("expected 1 sent and 1 received, got sent=%d recv=%d", sent, recv)
	}

	// 3. Resolver executes authoritative incoming commands
	events := r.Resolve(state, q)
	if len(events) == 0 {
		t.Fatalf("expected events emitted from valid resolution, got 0")
	}

	// 4. Assert authoritative state mutated correctly
	if state.ActiveUnitID == nil {
		t.Fatalf("expected ActiveUnitID to be set, got nil")
	}
	if *state.ActiveUnitID != 1 {
		t.Errorf("expected ActiveUnitID 1, got %d", *state.ActiveUnitID)
	}
	if state.Phase != domain.PhaseChooseAction {
		t.Errorf("expected PhaseChooseAction, got %s", state.Phase)
	}
}
