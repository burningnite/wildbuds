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
// 1. High-Concurrency MPSC Stress: 60+ Producers, Multi-Consumer Receive & Drain
// ============================================================================

func TestChallenger2_HighConcurrency_MPSC_Stress(t *testing.T) {
	const (
		numProducers    = 64
		cmdsPerProducer = 80
		totalCmds       = numProducers * cmdsPerProducer
		bufferCap       = 512
		numConsumers    = 6
	)

	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(bufferCap))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	var (
		sentCount     int64
		receivedCount int64
		stopConsumers = make(chan struct{})
		wgConsumers   sync.WaitGroup
		wgProducers   sync.WaitGroup
	)

	// Spawn 6 concurrent consumers:
	// 2 use Receive() channel
	// 2 use Drain() batch polling
	// 2 use Poll() single polling
	wgConsumers.Add(numConsumers)
	for c := 0; c < numConsumers; c++ {
		consumerID := c
		go func() {
			defer wgConsumers.Done()
			for {
				select {
				case <-stopConsumers:
					// Drain final residual messages
					for {
						drained := lt.Drain()
						if len(drained) == 0 {
							return
						}
						for _, cmd := range drained {
							if cmd.Sender != domain.PlayerOne {
								t.Errorf("corrupted sender: %v", cmd.Sender)
							}
							atomic.AddInt64(&receivedCount, 1)
						}
					}
				default:
					switch consumerID % 3 {
					case 0:
						// Receive() channel path
						select {
						case netCmd, ok := <-lt.Receive():
							if !ok {
								return
							}
							if netCmd.Sender != domain.PlayerOne {
								t.Errorf("corrupted sender in Receive: %v", netCmd.Sender)
							}
							atomic.AddInt64(&receivedCount, 1)
						case <-time.After(100 * time.Microsecond):
						}
					case 1:
						// Drain() batch path
						drained := lt.Drain()
						for _, cmd := range drained {
							if cmd.Sender != domain.PlayerOne {
								t.Errorf("corrupted sender in Drain: %v", cmd.Sender)
							}
							atomic.AddInt64(&receivedCount, 1)
						}
						time.Sleep(50 * time.Microsecond)
					case 2:
						// Poll() single-item path
						if cmd, ok := lt.Poll(); ok {
							if cmd.Sender != domain.PlayerOne {
								t.Errorf("corrupted sender in Poll: %v", cmd.Sender)
							}
							atomic.AddInt64(&receivedCount, 1)
						} else {
							time.Sleep(50 * time.Microsecond)
						}
					}
				}
			}
		}()
	}

	// Spawn 64 concurrent producers using alternating Send and SendBatch
	wgProducers.Add(numProducers)
	for p := 0; p < numProducers; p++ {
		producerID := p
		go func() {
			defer wgProducers.Done()
			for i := 0; i < cmdsPerProducer; i++ {
				cmd := commands.NewMoveUnitCommand(domain.GridPosition{
					X: (producerID%11 - 5),
					Y: (i%11 - 5),
				})

				for {
					var sendErr error
					if producerID%2 == 0 && i%2 == 0 {
						sendErr = lt.SendBatch(cmd)
					} else {
						sendErr = lt.Send(cmd)
					}

					if sendErr == nil {
						atomic.AddInt64(&sentCount, 1)
						break
					}
					if errors.Is(sendErr, transport.ErrQueueFull) {
						time.Sleep(100 * time.Microsecond)
						continue
					}
					t.Errorf("unexpected send error: %v", sendErr)
					return
				}
			}
		}()
	}

	wgProducers.Wait()

	// Await consumption
	deadline := time.Now().Add(5 * time.Second)
	for atomic.LoadInt64(&receivedCount) < int64(totalCmds) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	close(stopConsumers)
	wgConsumers.Wait()

	actualSent := atomic.LoadInt64(&sentCount)
	actualRecv := atomic.LoadInt64(&receivedCount)

	if actualSent != int64(totalCmds) {
		t.Fatalf("expected %d sent, got %d", totalCmds, actualSent)
	}
	if actualRecv != int64(totalCmds) {
		t.Fatalf("MPSC message loss: sent %d, received %d", actualSent, actualRecv)
	}
}

// ============================================================================
// 2. Concurrent Close Races: 50+ Closers vs 50+ Senders vs Receivers
// ============================================================================

func TestChallenger2_Concurrent_CloseRaces_ZeroPanics(t *testing.T) {
	const iterations = 15
	for iter := 0; iter < iterations; iter++ {
		lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(128))
		if err != nil {
			t.Fatalf("NewLoopback failed: %v", err)
		}

		const numGoroutines = 50
		var wg sync.WaitGroup
		var panicDetected int64

		// 50 goroutines calling Close() simultaneously
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						atomic.AddInt64(&panicDetected, 1)
						t.Errorf("PANIC on Close(): %v", r)
					}
				}()
				_ = lt.Close()
			}()
		}

		// 50 goroutines calling Send() simultaneously
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						atomic.AddInt64(&panicDetected, 1)
						t.Errorf("PANIC on Send(): %v", r)
					}
				}()
				cmd := commands.NewEndActivationCommand()
				err := lt.Send(cmd)
				if err != nil && !errors.Is(err, transport.ErrTransportClosed) && !errors.Is(err, transport.ErrQueueFull) {
					t.Errorf("unexpected error on Send: %v", err)
				}
			}()
		}

		// 50 goroutines calling SendBatch() simultaneously
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						atomic.AddInt64(&panicDetected, 1)
						t.Errorf("PANIC on SendBatch(): %v", r)
					}
				}()
				cmd := commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3})
				err := lt.SendBatch(cmd, cmd)
				if err != nil && !errors.Is(err, transport.ErrTransportClosed) && !errors.Is(err, transport.ErrQueueFull) {
					t.Errorf("unexpected error on SendBatch: %v", err)
				}
			}()
		}

		// 50 goroutines calling Receive/Poll/Drain
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			idx := i
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						atomic.AddInt64(&panicDetected, 1)
						t.Errorf("PANIC on consumer: %v", r)
					}
				}()
				if idx%3 == 0 {
					_ = lt.Drain()
				} else if idx%3 == 1 {
					_, _ = lt.Poll()
				} else {
					select {
					case _, ok := <-lt.Receive():
						_ = ok
					default:
					}
				}
			}()
		}

		wg.Wait()

		if atomic.LoadInt64(&panicDetected) > 0 {
			t.Fatalf("iteration %d: detected %d panics during concurrent close race", iter, atomic.LoadInt64(&panicDetected))
		}
		if !lt.IsClosed() {
			t.Fatalf("iteration %d: transport is not marked closed after Close() calls", iter)
		}
	}
}

// ============================================================================
// 3. In-Flight Draining & Channel Closure Without Deadlock
// ============================================================================

func TestChallenger2_InFlightDraining_ChannelClosure_NoDeadlock(t *testing.T) {
	const count = 400
	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(500))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}

	// 1. Send exactly 400 messages before Close()
	for i := 0; i < count; i++ {
		cmd := commands.NewMoveUnitCommand(domain.GridPosition{X: (i % 11) - 5, Y: 0})
		if err := lt.Send(cmd); err != nil {
			t.Fatalf("Send failed before close at %d: %v", i, err)
		}
	}

	if lt.Len() != count {
		t.Fatalf("expected buffer length %d, got %d", count, lt.Len())
	}

	// 2. Close the transport
	if err := lt.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// 3. Confirm subsequent sends fail immediately with ErrTransportClosed
	if err := lt.Send(commands.NewEndActivationCommand()); !errors.Is(err, transport.ErrTransportClosed) {
		t.Errorf("expected ErrTransportClosed on post-close Send, got %v", err)
	}

	// 4. Drain all 400 messages concurrently using 4 worker goroutines
	var (
		wgDraining sync.WaitGroup
		drainedSum int64
	)
	const numDrainers = 4
	wgDraining.Add(numDrainers)
	for d := 0; d < numDrainers; d++ {
		go func() {
			defer wgDraining.Done()
			for {
				cmd, ok := <-lt.Receive()
				if !ok {
					// Channel is closed and drained
					return
				}
				if cmd.Sender != domain.PlayerOne {
					t.Errorf("corrupted sender: %v", cmd.Sender)
				}
				atomic.AddInt64(&drainedSum, 1)
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wgDraining.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Succeeded
	case <-time.After(3 * time.Second):
		t.Fatalf("DEADLOCK: In-flight draining did not complete within deadline")
	}

	if atomic.LoadInt64(&drainedSum) != count {
		t.Fatalf("expected exactly %d messages drained, got %d", count, atomic.LoadInt64(&drainedSum))
	}

	// 5. Verify subsequent reads yield channel closure without deadlock
	select {
	case cmd, ok := <-lt.Receive():
		if ok {
			t.Errorf("expected Receive() to be closed (ok=false), got ok=true: %v", cmd)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("DEADLOCK: Receive() blocked on closed, drained transport")
	}

	if _, ok := lt.Poll(); ok {
		t.Errorf("expected Poll() to return false on closed drained transport")
	}

	if drained := lt.Drain(); len(drained) != 0 {
		t.Errorf("expected Drain() to return empty slice on closed drained transport, got %d", len(drained))
	}
}

// ============================================================================
// 4. Paired Transport Fuzzing: Cross-Fire Bidirectional Messaging
// ============================================================================

func TestChallenger2_PairedTransport_CrossFire_Bidirectional(t *testing.T) {
	p1, p2, err := transport.NewLoopbackPair(transport.WithBufferSize(512))
	if err != nil {
		t.Fatalf("NewLoopbackPair failed: %v", err)
	}
	defer p1.Close()
	defer p2.Close()

	const (
		sendersPerSide = 35
		cmdsPerSender  = 50
		totalPerSide   = sendersPerSide * cmdsPerSender
	)

	var (
		p1Sent int64
		p2Sent int64
		p1Recv int64
		p2Recv int64
		stop   = make(chan struct{})
		wgRecv sync.WaitGroup
	)

	// P1 receiver: consumes messages sent by P2
	wgRecv.Add(1)
	go func() {
		defer wgRecv.Done()
		for {
			select {
			case netCmd, ok := <-p1.Receive():
				if !ok {
					return
				}
				if netCmd.Sender != domain.PlayerTwo {
					t.Errorf("P1 received command with wrong sender: %v", netCmd.Sender)
				}
				atomic.AddInt64(&p1Recv, 1)
			case <-stop:
				for {
					select {
					case netCmd, ok := <-p1.Receive():
						if !ok {
							return
						}
						if netCmd.Sender != domain.PlayerTwo {
							t.Errorf("P1 received command with wrong sender: %v", netCmd.Sender)
						}
						atomic.AddInt64(&p1Recv, 1)
					default:
						return
					}
				}
			}
		}
	}()

	// P2 receiver: consumes messages sent by P1
	wgRecv.Add(1)
	go func() {
		defer wgRecv.Done()
		for {
			select {
			case netCmd, ok := <-p2.Receive():
				if !ok {
					return
				}
				if netCmd.Sender != domain.PlayerOne {
					t.Errorf("P2 received command with wrong sender: %v", netCmd.Sender)
				}
				atomic.AddInt64(&p2Recv, 1)
			case <-stop:
				for {
					select {
					case netCmd, ok := <-p2.Receive():
						if !ok {
							return
						}
						if netCmd.Sender != domain.PlayerOne {
							t.Errorf("P2 received command with wrong sender: %v", netCmd.Sender)
						}
						atomic.AddInt64(&p2Recv, 1)
					default:
						return
					}
				}
			}
		}
	}()

	// Spawn cross-fire senders
	var wgSenders sync.WaitGroup
	wgSenders.Add(sendersPerSide * 2)

	// P1 -> P2 senders
	for i := 0; i < sendersPerSide; i++ {
		sid := i
		go func() {
			defer wgSenders.Done()
			for k := 0; k < cmdsPerSender; k++ {
				cmd := commands.NewSelectUnitCommand(domain.GridPosition{X: sid % 5, Y: -(k % 5)})
				for {
					sendErr := p1.Send(cmd)
					if sendErr == nil {
						atomic.AddInt64(&p1Sent, 1)
						break
					}
					if errors.Is(sendErr, transport.ErrQueueFull) {
						time.Sleep(50 * time.Microsecond)
						continue
					}
					t.Errorf("P1 send error: %v", sendErr)
					return
				}
			}
		}()
	}

	// P2 -> P1 senders
	for i := 0; i < sendersPerSide; i++ {
		sid := i
		go func() {
			defer wgSenders.Done()
			for k := 0; k < cmdsPerSender; k++ {
				cmd := commands.NewMoveUnitCommand(domain.GridPosition{X: -(sid % 5), Y: k % 5})
				for {
					sendErr := p2.Send(cmd)
					if sendErr == nil {
						atomic.AddInt64(&p2Sent, 1)
						break
					}
					if errors.Is(sendErr, transport.ErrQueueFull) {
						time.Sleep(50 * time.Microsecond)
						continue
					}
					t.Errorf("P2 send error: %v", sendErr)
					return
				}
			}
		}()
	}

	wgSenders.Wait()

	// Wait for receivers
	deadline := time.Now().Add(5 * time.Second)
	for (atomic.LoadInt64(&p1Recv) < int64(totalPerSide) || atomic.LoadInt64(&p2Recv) < int64(totalPerSide)) && time.Now().Before(deadline) {
		time.Sleep(2 * time.Millisecond)
	}

	close(stop)
	wgRecv.Wait()

	if atomic.LoadInt64(&p1Sent) != int64(totalPerSide) || atomic.LoadInt64(&p2Recv) != int64(totalPerSide) {
		t.Fatalf("P1->P2 cross-fire mismatch: sent=%d, recv=%d", atomic.LoadInt64(&p1Sent), atomic.LoadInt64(&p2Recv))
	}
	if atomic.LoadInt64(&p2Sent) != int64(totalPerSide) || atomic.LoadInt64(&p1Recv) != int64(totalPerSide) {
		t.Fatalf("P2->P1 cross-fire mismatch: sent=%d, recv=%d", atomic.LoadInt64(&p2Sent), atomic.LoadInt64(&p1Recv))
	}
}

// ============================================================================
// 5. Paired Transport Close Cross-Fire Race
// ============================================================================

func TestChallenger2_Paired_Close_CrossFire_Race(t *testing.T) {
	for iter := 0; iter < 10; iter++ {
		p1, p2, err := transport.NewLoopbackPair(transport.WithBufferSize(64))
		if err != nil {
			t.Fatalf("NewLoopbackPair failed: %v", err)
		}

		var wg sync.WaitGroup
		var panics int64

		// P1 closer
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					atomic.AddInt64(&panics, 1)
					t.Errorf("panic in P1 close: %v", r)
				}
			}()
			_ = p1.Close()
		}()

		// P2 closer
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					atomic.AddInt64(&panics, 1)
					t.Errorf("panic in P2 close: %v", r)
				}
			}()
			_ = p2.Close()
		}()

		// Cross senders
		for i := 0; i < 20; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						atomic.AddInt64(&panics, 1)
						t.Errorf("panic in P1 send: %v", r)
					}
				}()
				_ = p1.Send(commands.NewEndActivationCommand())
			}()
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						atomic.AddInt64(&panics, 1)
						t.Errorf("panic in P2 send: %v", r)
					}
				}()
				_ = p2.Send(commands.NewEndActivationCommand())
			}()
		}

		// Receivers
		for i := 0; i < 10; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				_, _ = p1.Poll()
			}()
			go func() {
				defer wg.Done()
				_, _ = p2.Poll()
			}()
		}

		wg.Wait()

		if atomic.LoadInt64(&panics) > 0 {
			t.Fatalf("iteration %d: panics during paired close race: %d", iter, panics)
		}
	}
}

// ============================================================================
// 6. Poison-Pill Immunity: Malformed / Corrupted Payloads
// ============================================================================

func TestChallenger2_PoisonPill_Immunity(t *testing.T) {
	lt, err := transport.NewLoopback(domain.PlayerOne, transport.WithBufferSize(16))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	poisonPills := []commands.GameCommand{
		{Type: commands.CommandType("InvalidActionType"), Target: domain.GridPosition{X: 0, Y: 0}},
		{Type: commands.CommandSelectUnit, Target: domain.GridPosition{X: 100, Y: 0}},
		{Type: commands.CommandMoveUnit, Destination: domain.GridPosition{X: 0, Y: -99}},
		{Type: commands.CommandAttack, Target: domain.GridPosition{X: -6, Y: 0}},
		{},
	}

	for idx, pill := range poisonPills {
		err := lt.Send(pill)
		if err == nil {
			t.Fatalf("[%d] poison pill was accepted without error! Expected ErrInvalidCommand", idx)
		}
		if !errors.Is(err, transport.ErrInvalidCommand) {
			t.Errorf("[%d] expected ErrInvalidCommand, got %v", idx, err)
		}
	}

	// Buffer must be empty
	if lt.Len() != 0 {
		t.Fatalf("poison pills were enqueued into buffer! Len=%d", lt.Len())
	}

	// Valid command immediately succeeds
	validCmd := commands.NewEndActivationCommand()
	if err := lt.Send(validCmd); err != nil {
		t.Fatalf("valid command failed after poison pill: %v", err)
	}
	netCmd, ok := <-lt.Receive()
	if !ok || !netCmd.Command.Equals(validCmd) {
		t.Fatalf("failed to receive valid command: %v", netCmd)
	}
}

// ============================================================================
// 7. Defect Probe: WithBlockingSend Deadlock on Close
// ============================================================================

func TestChallenger2_BlockingSend_Close_DeadlockProbe(t *testing.T) {
	// Probe the deadlock in WithBlockingSend(true):
	// Send() holds lt.mu.RLock() while blocking on full channel.
	// Close() attempts lt.mu.Lock(), blocking forever waiting for RLock to release.
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
		t.Fatalf("initial send failed: %v", err)
	}

	sendBlocked := make(chan struct{})
	sendDone := make(chan error, 1)

	go func() {
		close(sendBlocked)
		err := lt.Send(commands.NewEndActivationCommand())
		sendDone <- err
	}()

	<-sendBlocked
	time.Sleep(50 * time.Millisecond)

	closeDone := make(chan error, 1)
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
				t.Errorf("expected ErrTransportClosed on unblocked send, got %v", sendErr)
			}
		case <-time.After(500 * time.Millisecond):
			t.Errorf("DEADLOCK DEFECT: Send() did not unblock after Close()")
		}
	case <-time.After(500 * time.Millisecond):
		t.Errorf("DEADLOCK DEFECT: Close() blocked indefinitely on lt.mu.Lock() while Send() held lt.mu.RLock()")
		// Unblock Send() so test runner does not hang
		<-lt.Receive()
	}
}

// ============================================================================
// 8. Defect Probe: AutoToggle State Corruption Under Backpressure Failure
// ============================================================================

func TestChallenger2_AutoToggle_CorruptionOnOverflowProbe(t *testing.T) {
	// Probe state corruption in autoToggle:
	// lt.localPlayer is updated before the command is placed into channel.
	// When Send fails with ErrQueueFull, localPlayer has already been permanently mutated.
	lt, err := transport.NewLoopback(
		domain.PlayerOne,
		transport.WithBufferSize(1),
		transport.WithAutoPlayerToggle(true),
	)
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	// Fill buffer
	if err := lt.Send(commands.NewSelectUnitCommand(domain.GridPosition{X: 0, Y: -3})); err != nil {
		t.Fatalf("fill buffer send failed: %v", err)
	}

	if lt.LocalPlayer() != domain.PlayerOne {
		t.Fatalf("expected PlayerOne, got %s", lt.LocalPlayer())
	}

	// Attempt EndActivation into full buffer -> returns ErrQueueFull
	err = lt.Send(commands.NewEndActivationCommand())
	if !errors.Is(err, transport.ErrQueueFull) {
		t.Fatalf("expected ErrQueueFull, got %v", err)
	}

	// Assert invariant: since command was rejected, player identity must NOT have toggled
	if lt.LocalPlayer() != domain.PlayerOne {
		t.Errorf("STATE CORRUPTION DEFECT: LocalPlayer toggled to %s even though Send failed with ErrQueueFull", lt.LocalPlayer())
	}
}

// ============================================================================
// 9. Defect Probe: Paired Transport WithBlockingSend Deadlock on Close
// ============================================================================

func TestChallenger2_Paired_BlockingSend_Close_DeadlockProbe(t *testing.T) {
	// In paired transport with blocking send:
	// P1 Send() acquires P2.mu.RLock() and blocks on full P2 channel.
	// P2.Close() attempts P2.mu.Lock(), blocking forever waiting for P1's RLock to release.
	p1, p2, err := transport.NewLoopbackPair(
		transport.WithBufferSize(1),
		transport.WithBlockingSend(true),
	)
	if err != nil {
		t.Fatalf("NewLoopbackPair failed: %v", err)
	}

	// Fill P2's incoming buffer via P1
	if err := p1.Send(commands.NewEndActivationCommand()); err != nil {
		t.Fatalf("initial send failed: %v", err)
	}

	sendBlocked := make(chan struct{})
	sendDone := make(chan error, 1)

	go func() {
		close(sendBlocked)
		err := p1.Send(commands.NewEndActivationCommand())
		sendDone <- err
	}()

	<-sendBlocked
	time.Sleep(50 * time.Millisecond)

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- p2.Close()
	}()

	select {
	case err := <-closeDone:
		if err != nil {
			t.Errorf("P2 Close returned unexpected error: %v", err)
		}
		select {
		case sendErr := <-sendDone:
			if !errors.Is(sendErr, transport.ErrTransportClosed) {
				t.Errorf("expected ErrTransportClosed on unblocked P1 send, got %v", sendErr)
			}
		case <-time.After(500 * time.Millisecond):
			t.Errorf("DEADLOCK DEFECT: P1 Send() did not unblock after P2.Close()")
		}
	case <-time.After(500 * time.Millisecond):
		t.Errorf("DEADLOCK DEFECT: P2.Close() blocked indefinitely on p2.mu.Lock() while P1.Send() held p2.mu.RLock()")
		// Unblock Send() by draining P2 so test does not hang
		<-p2.Receive()
	}
}
