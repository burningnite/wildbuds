package transport_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
	"wildbuds/internal/transport"
)

// ============================================================================
// 1. High-Throughput FIFO Ordering & Burst Stress
// ============================================================================

func TestAdversarial_HighThroughput_SequentialFIFO(t *testing.T) {
	const count = 10000
	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(count))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	// Send 10,000 commands with deterministic coordinates
	for i := 0; i < count; i++ {
		cmd := commands.NewMoveUnitCommand(domain.GridPosition{
			X: (i % 11) - 5,
			Y: ((i / 11) % 11) - 5,
		})
		if err := lt.Send(cmd); err != nil {
			t.Fatalf("Send failed at index %d: %v", i, err)
		}
	}

	if lt.Len() != count {
		t.Fatalf("expected queue length %d, got %d", count, lt.Len())
	}

	// Drain all and verify strict FIFO ordering
	drained := lt.Drain()
	if len(drained) != count {
		t.Fatalf("expected %d drained commands, got %d", count, len(drained))
	}

	for i := 0; i < count; i++ {
		expectedDest := domain.GridPosition{
			X: (i % 11) - 5,
			Y: ((i / 11) % 11) - 5,
		}
		if drained[i].Sender != domain.PlayerOne {
			t.Fatalf("[%d] sender corrupted: expected %s, got %s", i, domain.PlayerOne, drained[i].Sender)
		}
		if !drained[i].Command.Destination.Equals(expectedDest) {
			t.Fatalf("[%d] FIFO violation: expected %v, got %v", i, expectedDest, drained[i].Command.Destination)
		}
	}
}

func TestAdversarial_Burst_RepeatedFlush(t *testing.T) {
	const (
		numBursts = 50
		burstSize = 200
		bufferCap = 256
		totalCmds = numBursts * burstSize
	)

	lt, err := transport.NewLoopback(domain.PlayerTwo, transport.WithBufferSize(bufferCap))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	totalProcessed := 0
	for b := 0; b < numBursts; b++ {
		// Send burst
		for i := 0; i < burstSize; i++ {
			seq := b*burstSize + i
			cmd := commands.NewSelectUnitCommand(domain.GridPosition{
				X: (seq % 11) - 5,
				Y: 0,
			})
			if err := lt.Send(cmd); err != nil {
				t.Fatalf("burst %d item %d send failed: %v", b, i, err)
			}
		}

		// Drain burst immediately
		drained := lt.Drain()
		if len(drained) != burstSize {
			t.Fatalf("burst %d: expected %d drained, got %d", b, burstSize, len(drained))
		}

		for i, netCmd := range drained {
			seq := b*burstSize + i
			expectedTarget := domain.GridPosition{X: (seq % 11) - 5, Y: 0}
			if netCmd.Sender != domain.PlayerTwo {
				t.Fatalf("burst %d item %d: sender mismatch", b, i)
			}
			if !netCmd.Command.Target.Equals(expectedTarget) {
				t.Fatalf("burst %d item %d: target mismatch, expected %v got %v", b, i, expectedTarget, netCmd.Command.Target)
			}
		}
		totalProcessed += len(drained)
	}

	if totalProcessed != totalCmds {
		t.Fatalf("expected %d total processed commands, got %d", totalCmds, totalProcessed)
	}
}

func TestAdversarial_Concurrent_MPSC_HighVolume(t *testing.T) {
	const (
		numProducers    = 16
		cmdsPerProducer = 1000
		totalExpected   = numProducers * cmdsPerProducer
		bufferCap       = 512
	)

	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(bufferCap))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	var receivedCount int64
	doneConsumer := make(chan struct{})

	// Dedicated consumer draining as fast as possible
	go func() {
		defer close(doneConsumer)
		for {
			select {
			case netCmd, ok := <-lt.Receive():
				if !ok {
					return
				}
				if netCmd.Sender != domain.PlayerOne {
					t.Errorf("corrupted sender: %v", netCmd.Sender)
				}
				if atomic.AddInt64(&receivedCount, 1) == int64(totalExpected) {
					return
				}
			}
		}
	}()

	var wg sync.WaitGroup
	wg.Add(numProducers)

	for p := 0; p < numProducers; p++ {
		go func(producerID int) {
			defer wg.Done()
			for i := 0; i < cmdsPerProducer; i++ {
				cmd := commands.NewMoveUnitCommand(domain.GridPosition{
					X: (producerID % 11) - 5,
					Y: (i % 11) - 5,
				})
				// Retry on backpressure since bufferCap < totalExpected
				for {
					err := lt.Send(cmd)
					if err == nil {
						break
					}
					if errors.Is(err, transport.ErrQueueFull) {
						time.Sleep(10 * time.Microsecond)
						continue
					}
					t.Errorf("producer %d unexpected send error: %v", producerID, err)
					return
				}
			}
		}(p)
	}

	wg.Wait()

	select {
	case <-doneConsumer:
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout waiting for consumer to receive all %d commands; received: %d", totalExpected, atomic.LoadInt64(&receivedCount))
	}

	if actual := atomic.LoadInt64(&receivedCount); actual != int64(totalExpected) {
		t.Fatalf("expected %d total received, got %d", totalExpected, actual)
	}
}

// ============================================================================
// 2. Backpressure & Queue Overflow Tests
// ============================================================================

func TestAdversarial_Backpressure_StrictNonBlocking(t *testing.T) {
	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(1))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	cmd := commands.NewEndActivationCommand()
	if err := lt.Send(cmd); err != nil {
		t.Fatalf("first send failed: %v", err)
	}

	// Buffer is now full. Measure time to return ErrQueueFull to verify strictly non-blocking.
	start := time.Now()
	err = lt.Send(cmd)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected ErrQueueFull on full buffer, got nil")
	}
	if !errors.Is(err, transport.ErrQueueFull) {
		t.Fatalf("expected ErrQueueFull, got %v", err)
	}
	if elapsed > 10*time.Millisecond {
		t.Fatalf("Send blocked on full buffer! Took %v (expected < 10ms)", elapsed)
	}

	// Verify recovery after drain
	polled, ok := lt.Poll()
	if !ok || polled.Command.Type != commands.CommandEndActivation {
		t.Fatalf("failed to poll item")
	}

	if err := lt.Send(cmd); err != nil {
		t.Fatalf("expected send after drain to succeed, got %v", err)
	}
}

func TestAdversarial_SendBatch_Backpressure(t *testing.T) {
	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(3))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	batch := []commands.GameCommand{
		commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3}),
		commands.NewMoveUnitCommand(domain.GridPosition{X: 0, Y: -2}),
		commands.NewAttackCommand(domain.GridPosition{X: 0, Y: 0}),
		commands.NewEndActivationCommand(), // 4th item, should cause overflow
	}

	err = lt.SendBatch(batch...)
	if !errors.Is(err, transport.ErrQueueFull) {
		t.Fatalf("expected ErrQueueFull for batch exceeding buffer capacity, got %v", err)
	}

	// First 3 items should have been enqueued
	if lt.Len() != 3 {
		t.Fatalf("expected 3 items queued before overflow, got %d", lt.Len())
	}
}

// ============================================================================
// 3. Extreme Buffer Sizes (1, 0, 10000, Negative)
// ============================================================================

func TestAdversarial_ExtremeBufferSize_One(t *testing.T) {
	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(1))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	if lt.BufferCap() != 1 {
		t.Fatalf("expected BufferCap 1, got %d", lt.BufferCap())
	}

	cmd := commands.NewEndActivationCommand()
	if err := lt.Send(cmd); err != nil {
		t.Fatalf("first send failed: %v", err)
	}
	if err := lt.Send(cmd); !errors.Is(err, transport.ErrQueueFull) {
		t.Fatalf("expected ErrQueueFull on 2nd send with buffer=1, got %v", err)
	}
}

func TestAdversarial_ExtremeBufferSize_TenThousand(t *testing.T) {
	const size = 10000
	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(size))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	if lt.BufferCap() != size {
		t.Fatalf("expected BufferCap %d, got %d", size, lt.BufferCap())
	}

	for i := 0; i < size; i++ {
		cmd := commands.NewEndActivationCommand()
		if err := lt.Send(cmd); err != nil {
			t.Fatalf("send %d failed: %v", i, err)
		}
	}

	if lt.Len() != size {
		t.Fatalf("expected len %d, got %d", size, lt.Len())
	}
}

func TestAdversarial_ExtremeBufferSize_Zero(t *testing.T) {
	// What does WithBufferSize(0) do?
	// In loopback.go line 38: if size > 0 { lt.bufferSize = size }
	// So 0 is ignored and remains DefaultBufferSize (256).
	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(0))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	if lt.BufferCap() != transport.DefaultBufferSize {
		t.Errorf("expected fallback to DefaultBufferSize (256) when 0 passed, got %d", lt.BufferCap())
	}
}

func TestAdversarial_ExtremeBufferSize_Negative(t *testing.T) {
	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(-50))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	if lt.BufferCap() != transport.DefaultBufferSize {
		t.Errorf("expected fallback to DefaultBufferSize (256) when negative passed, got %d", lt.BufferCap())
	}
}

// ============================================================================
// 4. Zero-Length Batches, Nil Arguments & Bug Probes
// ============================================================================

func TestAdversarial_ZeroLengthBatches(t *testing.T) {
	lt, err := transport.NewLoopback(domain.PlayerOne)
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	// 1. Variadic with 0 arguments
	if err := lt.SendBatch(); err != nil {
		t.Errorf("SendBatch() returned error: %v", err)
	}

	// 2. Empty slice
	if err := lt.SendBatch([]commands.GameCommand{}...); err != nil {
		t.Errorf("SendBatch([]) returned error: %v", err)
	}

	// 3. Nil slice
	if err := lt.SendBatch(nil...); err != nil {
		t.Errorf("SendBatch(nil) returned error: %v", err)
	}

	if lt.Len() != 0 {
		t.Errorf("expected 0 items in queue, got %d", lt.Len())
	}
}

func TestAdversarial_NilOption_CrashProbe(t *testing.T) {
	// BUG PROBE: NewLoopback loops over opts without checking `if opt != nil`.
	// Passing a nil option `LoopbackOption(nil)` will cause a panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NIL POINTER DEFECT: NewLoopback panics on nil LoopbackOption: %v", r)
		}
	}()

	var nilOpt transport.LoopbackOption = nil
	_, _ = transport.NewLoopback(domain.PlayerOne, nilOpt)
}

func TestAdversarial_Pump_DataLossOnOverflow(t *testing.T) {
	// BUG PROBE: In Pump(), q.DrainOutgoing() empties q.Outgoing completely.
	// If t.Send() fails with ErrQueueFull, unsent commands are permanently discarded.
	// Furthermore, Phase 2 is skipped so already-sent commands are stranded in transport.
	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(2))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	q := commands.NewCommandQueue()
	q.PushOutgoing(commands.NewEndActivationCommand())
	q.PushOutgoing(commands.NewEndActivationCommand())
	q.PushOutgoing(commands.NewEndActivationCommand()) // 3rd item will overflow buffer of size 2
	q.PushOutgoing(commands.NewEndActivationCommand()) // 4th item

	sent, received, err := lt.Pump(q)
	if err == nil {
		t.Fatalf("expected error from Pump on buffer overflow")
	}
	if sent != 2 {
		t.Errorf("expected sent == 2, got %d", sent)
	}

	// Invariant: commands not sent must NOT be destroyed/lost from the pipeline.
	if q.LenOutgoing() != 2 {
		t.Errorf("DATA LOSS DEFECT: Pump drained 4 commands from q.Outgoing, failed on 3rd, and dropped remaining 2 commands (q.LenOutgoing() == %d, expected 2)", q.LenOutgoing())
	}

	// Invariant: commands successfully sent into transport must be pumped into q.Incoming
	if received != 2 || q.LenIncoming() != 2 {
		t.Errorf("PIPELINE STALL DEFECT: 2 sent commands were stranded in transport and not pumped to Incoming (received=%d, q.LenIncoming()=%d)", received, q.LenIncoming())
	}
}

// ============================================================================
// 5. Hotseat Alternation Multi-Turn & AutoToggle Invariant Stress
// ============================================================================

func TestAdversarial_HotseatAlternation_MultiTurn(t *testing.T) {
	const numTurns = 100
	lt, err := transport.NewLoopback(
		domain.PlayerOne,
		transport.WithAutoPlayerToggle(true),
		transport.WithBufferSize(500),
	)
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	expectedPlayer := domain.PlayerOne

	for turn := 0; turn < numTurns; turn++ {
		if lt.LocalPlayer() != expectedPlayer {
			t.Fatalf("turn %d: expected LocalPlayer %s, got %s", turn, expectedPlayer, lt.LocalPlayer())
		}

		// Send non-toggle commands
		cmdSelect := commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: 0})
		if err := lt.Send(cmdSelect); err != nil {
			t.Fatalf("turn %d select failed: %v", turn, err)
		}
		if lt.LocalPlayer() != expectedPlayer {
			t.Fatalf("turn %d: player toggled prematurely after select", turn)
		}

		cmdMove := commands.NewMoveUnitCommand(domain.GridPosition{X: 1, Y: 0})
		if err := lt.Send(cmdMove); err != nil {
			t.Fatalf("turn %d move failed: %v", turn, err)
		}
		if lt.LocalPlayer() != expectedPlayer {
			t.Fatalf("turn %d: player toggled prematurely after move", turn)
		}

		// Send EndActivation -> triggers toggle
		cmdEnd := commands.NewEndActivationCommand()
		if err := lt.Send(cmdEnd); err != nil {
			t.Fatalf("turn %d end activation failed: %v", turn, err)
		}

		// Now player should have toggled to next
		expectedPlayer = expectedPlayer.Next()
		if lt.LocalPlayer() != expectedPlayer {
			t.Fatalf("turn %d: player failed to toggle to %s, got %s", turn, expectedPlayer, lt.LocalPlayer())
		}
	}

	// Now drain and verify every single stamped command in the inbound stream
	drained := lt.Drain()
	expectedTotal := numTurns * 3
	if len(drained) != expectedTotal {
		t.Fatalf("expected %d drained commands, got %d", expectedTotal, len(drained))
	}

	curExpected := domain.PlayerOne
	for turn := 0; turn < numTurns; turn++ {
		c1 := drained[turn*3]
		c2 := drained[turn*3+1]
		c3 := drained[turn*3+2]

		if c1.Sender != curExpected {
			t.Fatalf("turn %d item 0 sender mismatch: expected %s, got %s", turn, curExpected, c1.Sender)
		}
		if c2.Sender != curExpected {
			t.Fatalf("turn %d item 1 sender mismatch: expected %s, got %s", turn, curExpected, c2.Sender)
		}
		if c3.Sender != curExpected {
			t.Fatalf("turn %d item 2 sender mismatch: expected %s, got %s", turn, curExpected, c3.Sender)
		}
		curExpected = curExpected.Next()
	}
}

func TestAdversarial_AutoToggle_CorruptionOnOverflow(t *testing.T) {
	// BUG PROBE: When WithAutoPlayerToggle(true) is enabled:
	// lt.localPlayer = lt.localPlayer.Next() is executed before Send pushes to inCh!
	// If inCh is full, Send returns ErrQueueFull.
	// But localPlayer was already toggled!
	lt, err := transport.NewLoopback(
		domain.PlayerOne,
		transport.WithAutoPlayerToggle(true),
		transport.WithBufferSize(1),
	)
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	// Fill the buffer with 1 item
	if err := lt.Send(commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: 0})); err != nil {
		t.Fatalf("initial send failed: %v", err)
	}

	if lt.LocalPlayer() != domain.PlayerOne {
		t.Fatalf("expected PlayerOne, got %s", lt.LocalPlayer())
	}

	// Now try to send EndActivation on full buffer -> will return ErrQueueFull
	err = lt.Send(commands.NewEndActivationCommand())
	if !errors.Is(err, transport.ErrQueueFull) {
		t.Fatalf("expected ErrQueueFull, got %v", err)
	}

	// Invariant: failed command submission must NOT advance game turn state!
	if lt.LocalPlayer() != domain.PlayerOne {
		t.Errorf("STATE CORRUPTION DEFECT: LocalPlayer advanced to %s even though Send() failed with ErrQueueFull", lt.LocalPlayer())
	}
}

// ============================================================================
// 6. Concurrency Stress & Deadlock Probes
// ============================================================================

func TestAdversarial_BlockingSend_DeadlockProbe(t *testing.T) {
	// BUG PROBE: WithBlockingSend(true).
	// When buffer is full, Send() holds lt.mu.RLock() while waiting on select { case inCh <- ...; case <-done: }.
	// Close() calls lt.mu.Lock() before closing done!
	// This causes a mutual deadlock:
	// Send() waits for done to close (or inCh to drain) while holding RLock().
	// Close() waits for RLock() to be released before it can acquire Lock() to close done!
	lt, err := transport.NewLoopback(
		domain.PlayerOne,
		transport.WithBufferSize(1),
		transport.WithBlockingSend(true),
	)
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}

	// Fill buffer
	if err := lt.Send(commands.NewEndActivationCommand()); err != nil {
		t.Fatalf("first send failed: %v", err)
	}

	sendBlocked := make(chan struct{})
	sendDone := make(chan error, 1)

	// Goroutine attempting to send 2nd command (will block because buffer is full)
	go func() {
		close(sendBlocked)
		err := lt.Send(commands.NewEndActivationCommand())
		sendDone <- err
	}()

	<-sendBlocked
	// Give the goroutine time to enter the blocking select while holding RLock
	time.Sleep(50 * time.Millisecond)

	closeDone := make(chan error, 1)
	// Now attempt to Close() the transport
	go func() {
		closeDone <- lt.Close()
	}()

	select {
	case err := <-closeDone:
		if err != nil {
			t.Errorf("Close returned unexpected error: %v", err)
		}
		select {
		case sendErr := <-sendDone:
			if !errors.Is(sendErr, transport.ErrTransportClosed) {
				t.Errorf("expected ErrTransportClosed on unblocked Send, got %v", sendErr)
			}
		case <-time.After(500 * time.Millisecond):
			t.Errorf("Send() did not unblock after Close()")
		}
	case <-time.After(500 * time.Millisecond):
		t.Errorf("DEADLOCK DEFECT: Close() blocked indefinitely on lt.mu.Lock() while Send() held lt.mu.RLock()")
		// Drain channel to unblock Send so test runner does not hang
		<-lt.Receive()
	}
}

func TestAdversarial_Concurrent_SendAndCloseRace(t *testing.T) {
	const (
		numSenders = 40
		numClosers = 10
	)
	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(64))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(numSenders + numClosers)

	for i := 0; i < numSenders; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				err := lt.Send(commands.NewEndActivationCommand())
				if err != nil && !errors.Is(err, transport.ErrTransportClosed) && !errors.Is(err, transport.ErrQueueFull) {
					t.Errorf("unexpected error in send: %v", err)
				}
			}
		}()
	}

	for i := 0; i < numClosers; i++ {
		go func() {
			defer wg.Done()
			_ = lt.Close()
		}()
	}

	wg.Wait()

	if !lt.IsClosed() {
		t.Errorf("expected transport to be closed")
	}
}
