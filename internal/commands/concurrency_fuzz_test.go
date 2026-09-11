package commands_test

import (
	"crypto/rand"
	"encoding/json"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
)

// ============================================================================
// 1. High Concurrency Multi-Producer Multi-Consumer Stress Harness
// ============================================================================

func TestAdversarial_CommandQueue_HighConcurrencyStress(t *testing.T) {
	q := commands.NewCommandQueue()

	const numProducers = 16
	const numConsumers = 8
	const numDrainers = 4
	const numInspectors = 4
	const itemsPerProducer = 200

	var totalOutgoingPushed int64 = int64(numProducers * itemsPerProducer)
	var totalIncomingPushed int64 = int64(numProducers * itemsPerProducer)

	var outgoingPoppedCount int64
	var incomingPoppedCount int64
	var outgoingDrainedCount int64
	var incomingDrainedCount int64

	var wgProducers sync.WaitGroup
	var wgConsumers sync.WaitGroup
	var wgInspectors sync.WaitGroup

	stopConsumers := make(chan struct{})

	// Start Outgoing Producers
	wgProducers.Add(numProducers)
	for p := 0; p < numProducers; p++ {
		go func(prodID int) {
			defer wgProducers.Done()
			for i := 0; i < itemsPerProducer; i++ {
				cmd := commands.NewSelectUnitCommand(domain.GridPosition{
					X: (prodID % 5) - 2,
					Y: (i % 5) - 2,
				})
				if i%5 == 0 {
					// Test batch push
					q.PushOutgoingBatch(cmd, commands.NewEndActivationCommand())
					i++ // account for extra item
				} else {
					q.PushOutgoing(cmd)
				}
			}
		}(p)
	}

	// Start Incoming Producers
	wgProducers.Add(numProducers)
	for p := 0; p < numProducers; p++ {
		go func(prodID int) {
			defer wgProducers.Done()
			sender := domain.PlayerOne
			if prodID%2 == 1 {
				sender = domain.PlayerTwo
			}
			for i := 0; i < itemsPerProducer; i++ {
				netCmd := commands.NewNetworkCommand(
					sender,
					commands.NewMoveUnitCommand(domain.GridPosition{
						X: (prodID % 5) - 2,
						Y: (i % 5) - 2,
					}),
				)
				if i%5 == 0 {
					q.PushIncomingBatch(netCmd, commands.NewNetworkCommand(sender, commands.NewEndActivationCommand()))
					i++
				} else {
					q.PushIncoming(netCmd)
				}
			}
		}(p)
	}

	// Start Outgoing Consumers (PopOutgoing)
	wgConsumers.Add(numConsumers)
	for c := 0; c < numConsumers; c++ {
		go func() {
			defer wgConsumers.Done()
			for {
				select {
				case <-stopConsumers:
					// Drain any remnants
					for {
						if _, ok := q.PopOutgoing(); ok {
							atomic.AddInt64(&outgoingPoppedCount, 1)
						} else {
							break
						}
					}
					return
				default:
					if _, ok := q.PopOutgoing(); ok {
						atomic.AddInt64(&outgoingPoppedCount, 1)
					}
				}
			}
		}()
	}

	// Start Incoming Consumers (PopIncoming)
	wgConsumers.Add(numConsumers)
	for c := 0; c < numConsumers; c++ {
		go func() {
			defer wgConsumers.Done()
			for {
				select {
				case <-stopConsumers:
					for {
						if _, ok := q.PopIncoming(); ok {
							atomic.AddInt64(&incomingPoppedCount, 1)
						} else {
							break
						}
					}
					return
				default:
					if _, ok := q.PopIncoming(); ok {
						atomic.AddInt64(&incomingPoppedCount, 1)
					}
				}
			}
		}()
	}

	// Start Drainers running concurrently
	wgConsumers.Add(numDrainers * 2)
	for d := 0; d < numDrainers; d++ {
		go func() {
			defer wgConsumers.Done()
			for {
				select {
				case <-stopConsumers:
					return
				default:
					drained := q.DrainOutgoing()
					if len(drained) > 0 {
						atomic.AddInt64(&outgoingDrainedCount, int64(len(drained)))
					}
				}
			}
		}()

		go func() {
			defer wgConsumers.Done()
			for {
				select {
				case <-stopConsumers:
					return
				default:
					drained := q.DrainIncoming()
					if len(drained) > 0 {
						atomic.AddInt64(&incomingDrainedCount, int64(len(drained)))
					}
				}
			}
		}()
	}

	// Start Concurrent Inspectors (Peek, Snapshot, Len, IsEmpty)
	stopInspectors := make(chan struct{})
	wgInspectors.Add(numInspectors)
	for ins := 0; ins < numInspectors; ins++ {
		go func() {
			defer wgInspectors.Done()
			for {
				select {
				case <-stopInspectors:
					return
				default:
					_ = q.Len()
					_ = q.LenOutgoing()
					_ = q.LenIncoming()
					_ = q.IsEmpty()
					_ = q.IsOutgoingEmpty()
					_ = q.IsIncomingEmpty()
					_, _ = q.PeekOutgoing()
					_, _ = q.PeekIncoming()
					_ = q.OutgoingSnapshot()
					_ = q.IncomingSnapshot()
				}
			}
		}()
	}

	// Wait for all producers to finish
	wgProducers.Wait()

	// Let consumers drain everything remaining
	close(stopInspectors)
	wgInspectors.Wait()

	// Wait until queue is fully empty
	for {
		if q.Len() == 0 {
			break
		}
	}
	close(stopConsumers)
	wgConsumers.Wait()

	totalOutRecovered := atomic.LoadInt64(&outgoingPoppedCount) + atomic.LoadInt64(&outgoingDrainedCount)
	totalInRecovered := atomic.LoadInt64(&incomingPoppedCount) + atomic.LoadInt64(&incomingDrainedCount)

	if totalOutRecovered != totalOutgoingPushed {
		t.Fatalf("Outgoing item loss under concurrency: pushed %d, recovered %d (popped=%d, drained=%d)",
			totalOutgoingPushed, totalOutRecovered, outgoingPoppedCount, outgoingDrainedCount)
	}

	if totalInRecovered != totalIncomingPushed {
		t.Fatalf("Incoming item loss under concurrency: pushed %d, recovered %d (popped=%d, drained=%d)",
			totalIncomingPushed, totalInRecovered, incomingPoppedCount, incomingDrainedCount)
	}

	if !q.IsEmpty() {
		t.Errorf("Queue expected to be empty after stress test, got len=%d", q.Len())
	}
}

// ============================================================================
// 2. Concurrent Route Loopback Stress Harness
// ============================================================================

func TestAdversarial_CommandQueue_ConcurrentLoopback(t *testing.T) {
	q := commands.NewCommandQueue()
	const iterations = 300

	var wg sync.WaitGroup
	wg.Add(3)

	var totalRouted int64

	// Goroutine 1: Pushes to Outgoing
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			q.PushOutgoing(commands.NewSelectUnitCommand(domain.GridPosition{X: i % 5, Y: 0}))
		}
	}()

	// Goroutine 2: Continuously routes Outgoing to Incoming
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			n := q.RouteOutgoingToIncoming(domain.PlayerOne)
			atomic.AddInt64(&totalRouted, int64(n))
		}
	}()

	// Goroutine 3: Consumes Incoming
	var consumedIncoming int64
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			if _, ok := q.PopIncoming(); ok {
				atomic.AddInt64(&consumedIncoming, 1)
			}
		}
	}()

	wg.Wait()

	// Drain any remaining outgoing that wasn't routed yet
	lastRouted := q.RouteOutgoingToIncoming(domain.PlayerOne)
	atomic.AddInt64(&totalRouted, int64(lastRouted))

	// Drain remaining incoming
	remainingIn := q.DrainIncoming()
	atomic.AddInt64(&consumedIncoming, int64(len(remainingIn)))

	if totalRouted != int64(iterations) {
		t.Errorf("RouteOutgoingToIncoming count mismatch: expected %d, got %d", iterations, totalRouted)
	}
	if consumedIncoming != int64(iterations) {
		t.Errorf("Total consumed incoming mismatch: expected %d, got %d", iterations, consumedIncoming)
	}
}

// ============================================================================
// 3. Memory Safety: Reference Zeroing & Capacity Reset Verification
// ============================================================================

func TestAdversarial_CommandQueue_ZeroReferenceAndCapacityReset(t *testing.T) {
	q := commands.NewCommandQueue()

	// 1. Verify PopOutgoing capacity reset when exceeding 1024
	const largeCount = 2048
	for i := 0; i < largeCount; i++ {
		q.PushOutgoing(commands.NewSelectUnitCommand(domain.GridPosition{X: 1, Y: 1}))
		q.PushIncoming(commands.NewNetworkCommand(domain.PlayerOne, commands.NewEndActivationCommand()))
	}

	if q.LenOutgoing() != largeCount || q.LenIncoming() != largeCount {
		t.Fatalf("expected queue lengths %d, got out=%d, in=%d",
			largeCount, q.LenOutgoing(), q.LenIncoming())
	}

	// Pop all items one by one
	for i := 0; i < largeCount; i++ {
		cmdOut, okOut := q.PopOutgoing()
		if !okOut || cmdOut.Type != commands.CommandSelectUnit {
			t.Fatalf("item %d pop outgoing failed", i)
		}
		cmdIn, okIn := q.PopIncoming()
		if !okIn || cmdIn.Command.Type != commands.CommandEndActivation {
			t.Fatalf("item %d pop incoming failed", i)
		}
	}

	if !q.IsEmpty() {
		t.Fatalf("queue should be completely empty after popping all elements")
	}

	// 2. Capacity Reset Verification via Snapshot:
	// If reset succeeded, snapshots of empty queues must be nil.
	if snapOut := q.OutgoingSnapshot(); snapOut != nil {
		t.Errorf("expected nil snapshot on empty outgoing, got %v", snapOut)
	}
	if snapIn := q.IncomingSnapshot(); snapIn != nil {
		t.Errorf("expected nil snapshot on empty incoming, got %v", snapIn)
	}

	// 3. Re-push and verify normal operation after capacity reset
	q.PushOutgoing(commands.NewEndActivationCommand())
	q.PushIncoming(commands.NewNetworkCommand(domain.PlayerTwo, commands.NewEndActivationCommand()))

	if q.LenOutgoing() != 1 || q.LenIncoming() != 1 {
		t.Fatalf("failed to push after capacity reset")
	}

	outCmd, _ := q.PopOutgoing()
	inCmd, _ := q.PopIncoming()
	if outCmd.Type != commands.CommandEndActivation || inCmd.Sender != domain.PlayerTwo {
		t.Fatalf("corrupted commands after capacity reset")
	}

	// 4. Test Drain on Empty vs Pop on Empty:
	// Verify that repeated calls to Pop on empty queue never crash or panic
	for i := 0; i < 100; i++ {
		if _, ok := q.PopOutgoing(); ok {
			t.Errorf("expected false on empty PopOutgoing")
		}
		if _, ok := q.PopIncoming(); ok {
			t.Errorf("expected false on empty PopIncoming")
		}
		if drained := q.DrainOutgoing(); drained != nil {
			t.Errorf("expected nil on empty DrainOutgoing")
		}
		if drained := q.DrainIncoming(); drained != nil {
			t.Errorf("expected nil on empty DrainIncoming")
		}
	}
}

// ============================================================================
// 4. Dual-Schema JSON Serialization: Full Equivalence Across Formats
// ============================================================================

func TestAdversarial_DualSchema_FullEquivalence(t *testing.T) {
	testCases := []struct {
		name       string
		flatJSON   string
		rustJSON   string
		expected   commands.GameCommand
	}{
		{
			name:     "SelectUnit",
			flatJSON: `{"type":"SelectUnit","target":{"x":-3,"y":2}}`,
			rustJSON: `{"SelectUnit":{"target":{"x":-3,"y":2}}}`,
			expected: commands.NewSelectUnitCommand(domain.GridPosition{X: -3, Y: 2}),
		},
		{
			name:     "MoveUnit",
			flatJSON: `{"type":"MoveUnit","destination":{"x":4,"y":-1}}`,
			rustJSON: `{"MoveUnit":{"destination":{"x":4,"y":-1}}}`,
			expected: commands.NewMoveUnitCommand(domain.GridPosition{X: 4, Y: -1}),
		},
		{
			name:     "Attack",
			flatJSON: `{"type":"Attack","target":{"x":0,"y":3}}`,
			rustJSON: `{"Attack":{"target":{"x":0,"y":3}}}`,
			expected: commands.NewAttackCommand(domain.GridPosition{X: 0, Y: 3}),
		},
		{
			name:     "EndActivation_Object",
			flatJSON: `{"type":"EndActivation"}`,
			rustJSON: `{"EndActivation":{}}`,
			expected: commands.NewEndActivationCommand(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test Flat Deserialization
			var fromFlat commands.GameCommand
			if err := json.Unmarshal([]byte(tc.flatJSON), &fromFlat); err != nil {
				t.Fatalf("failed to unmarshal flat JSON: %v", err)
			}
			if !fromFlat.Equals(tc.expected) {
				t.Errorf("flat mismatch: got %v, want %v", fromFlat, tc.expected)
			}

			// Test Rust Serde Deserialization
			var fromRust commands.GameCommand
			if err := json.Unmarshal([]byte(tc.rustJSON), &fromRust); err != nil {
				t.Fatalf("failed to unmarshal Rust JSON: %v", err)
			}
			if !fromRust.Equals(tc.expected) {
				t.Errorf("rust mismatch: got %v, want %v", fromRust, tc.expected)
			}

			// Cross-equivalence: Flat and Rust inputs decode to identical structs
			if !fromFlat.Equals(fromRust) {
				t.Errorf("cross-format equivalence failed: flat %v != rust %v", fromFlat, fromRust)
			}

			// Re-serialization of both produces valid canonical JSON
			data, err := json.Marshal(fromRust)
			if err != nil {
				t.Fatalf("failed to marshal fromRust: %v", err)
			}
			var roundTripped commands.GameCommand
			if err := json.Unmarshal(data, &roundTripped); err != nil {
				t.Fatalf("failed to unmarshal roundtrip: %v", err)
			}
			if !roundTripped.Equals(tc.expected) {
				t.Errorf("roundtrip mismatch: got %v, want %v", roundTripped, tc.expected)
			}
		})
	}
}

func TestAdversarial_DualSchema_RustStringVariantEndActivation(t *testing.T) {
	raw := []byte(`"EndActivation"`)
	var cmd commands.GameCommand
	if err := json.Unmarshal(raw, &cmd); err != nil {
		t.Fatalf("failed to unmarshal string EndActivation: %v", err)
	}
	expected := commands.NewEndActivationCommand()
	if !cmd.Equals(expected) {
		t.Errorf("expected %v, got %v", expected, cmd)
	}

	// String variants of other commands must fail (only EndActivation is a unit variant in Rust)
	invalidStringVariants := []string{
		`"SelectUnit"`,
		`"MoveUnit"`,
		`"Attack"`,
		`"Unknown"`,
		`""`,
	}
	for _, rawStr := range invalidStringVariants {
		var c commands.GameCommand
		if err := json.Unmarshal([]byte(rawStr), &c); err == nil {
			t.Errorf("expected error unmarshaling bare string %s, got nil", rawStr)
		}
	}
}

// ============================================================================
// 5. Dual-Schema Fuzzing & Malformed Payload Resilience
// ============================================================================

func TestAdversarial_DualSchema_FuzzMalformedPayloads(t *testing.T) {
	malformedPayloads := []struct {
		name string
		json string
	}{
		// Syntax anomalies
		{"EmptyString", ``},
		{"WhitespaceOnly", `    `},
		{"NullBytes", "\x00\x00\x00"},
		{"TruncatedObject", `{"type":"SelectUnit"`},
		{"TruncatedRust", `{"SelectUnit":{"target":`},
		{"TrailingComma", `{"type":"EndActivation",}`},
		{"UnquotedKey", `{type:"EndActivation"}`},
		{"SingleQuotes", `{'type':'EndActivation'}`},

		// Type confusion
		{"NullLiteral", `null`},
		{"NumberLiteral", `12345`},
		{"BoolTrue", `true`},
		{"BoolFalse", `false`},
		{"ArrayRoot", `["SelectUnit", "MoveUnit"]`},
		{"EmptyObject", `{}`},
		{"TypeIsNumber", `{"type": 123}`},
		{"TypeIsObject", `{"type": {"sub":"SelectUnit"}}`},
		{"TypeIsArray", `{"type": ["SelectUnit"]}`},
		{"TypeIsNull", `{"type": null}`},

		// Rust format mutations
		{"RustEmptyObject", `{}`},
		{"RustMultiKey", `{"SelectUnit":{},"MoveUnit":{}}`},
		{"RustThreeKeys", `{"A":{},"B":{},"C":{}}`},
		{"RustUnknownKey", `{"InvalidCommandType":{}}`},
		{"RustPayloadNotObject", `{"SelectUnit":"not_an_object"}`},
		{"RustPayloadNumber", `{"SelectUnit": 999}`},
		{"RustPayloadArray", `{"MoveUnit": [1, 2, 3]}`},
		{"RustPayloadNull", `{"Attack": null}`},
		{"RustSelectMissingTarget", `{"SelectUnit":{}}`},
		{"RustMoveMissingDest", `{"MoveUnit":{}}`},
		{"RustAttackMissingTarget", `{"Attack":{}}`},

		// Coordinate type mutations
		{"TargetIsString", `{"type":"SelectUnit","target":"0,0"}`},
		{"TargetIsNumber", `{"type":"SelectUnit","target": 0}`},
		{"TargetIsArray", `{"type":"SelectUnit","target":[0, 0]}`},
		{"TargetXIsString", `{"type":"SelectUnit","target":{"x":"zero","y":0}}`},
		{"TargetYIsBool", `{"type":"SelectUnit","target":{"x":0,"y":false}}`},
		{"DestinationIsArray", `{"type":"MoveUnit","destination":[1, 2]}`},

		// Number extremes (JSON numbers)
		{"Coordinate1e309", `{"type":"SelectUnit","target":{"x":1e309,"y":0}}`},
		{"CoordinateNeg1e309", `{"type":"SelectUnit","target":{"x":0,"y":-1e309}}`},
		{"CoordinateMaxInt64", `{"type":"SelectUnit","target":{"x":9223372036854775807,"y":0}}`},
		{"CoordinateMinInt64", `{"type":"SelectUnit","target":{"x":0,"y":-9223372036854775808}}`},

		// Deep nesting / JSON bomb
		{"DeeplyNestedObject", `{"a":{"b":{"c":{"d":{"e":{"f":{"g":{"h":{}}}}}}}}}`},

		// Unknown command type in flat format
		{"FlatUnknownType", `{"type":"NukeEntireBoard","target":{"x":0,"y":0}}`},
		{"FlatEmptyType", `{"type":""}`},
	}

	for _, tc := range malformedPayloads {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("PANIC on malformed payload %s: %v", tc.name, r)
				}
			}()

			var cmd commands.GameCommand
			err := json.Unmarshal([]byte(tc.json), &cmd)

			// Every malformed payload MUST return an error or decode to an invalid command
			if err == nil {
				// If it didn't return an error at unmarshal time, Validate() MUST reject it
				valErr := cmd.Validate()
				if valErr == nil && cmd.Type != "" {
					t.Errorf("malformed payload %s unexpectedly succeeded validation: cmd=%v", tc.name, cmd)
				}
			}
		})
	}
}

// ============================================================================
// 6. Chaos Random Byte Mutation Fuzzing (1,000 Iterations)
// ============================================================================

func TestAdversarial_DualSchema_ChaosByteFuzzing(t *testing.T) {
	seeds := []string{
		`{"type":"SelectUnit","target":{"x":0,"y":-3}}`,
		`{"SelectUnit":{"target":{"x":0,"y":-3}}}`,
		`{"type":"MoveUnit","destination":{"x":1,"y":2}}`,
		`{"MoveUnit":{"destination":{"x":1,"y":2}}}`,
		`{"type":"Attack","target":{"x":0,"y":3}}`,
		`{"Attack":{"target":{"x":0,"y":3}}}`,
		`{"type":"EndActivation"}`,
		`{"EndActivation":{}}`,
		`"EndActivation"`,
	}

	const iterations = 1000

	for i := 0; i < iterations; i++ {
		seed := seeds[i%len(seeds)]
		mutated := mutateRandomBytes([]byte(seed))

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("iteration %d: PANIC on input %q: %v", i, string(mutated), r)
				}
			}()

			var cmd commands.GameCommand
			_ = json.Unmarshal(mutated, &cmd)
		}()
	}
}

// Helper function to mutate bytes randomly
func mutateRandomBytes(data []byte) []byte {
	if len(data) == 0 {
		return []byte(`{}`)
	}

	cp := make([]byte, len(data))
	copy(cp, data)

	mode, _ := rand.Int(rand.Reader, big.NewInt(4))
	idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(cp))))
	i := int(idx.Int64())

	switch mode.Int64() {
	case 0:
		// Bit flip
		bit, _ := rand.Int(rand.Reader, big.NewInt(8))
		cp[i] ^= 1 << bit.Int64()
	case 1:
		// Byte insertion
		randomByte := make([]byte, 1)
		_, _ = rand.Read(randomByte)
		cp = append(cp[:i], append(randomByte, cp[i:]...)...)
	case 2:
		// Byte deletion
		if len(cp) > 1 {
			cp = append(cp[:i], cp[i+1:]...)
		}
	case 3:
		// Truncation
		cp = cp[:i]
	}

	return cp
}

// ============================================================================
// 7. NetworkCommand JSON Fuzzing
// ============================================================================

func TestAdversarial_NetworkCommand_JSON_Fuzz(t *testing.T) {
	invalidNetworkJSONs := []struct {
		name string
		json string
	}{
		{"InvalidSenderString", `{"sender":"PlayerThree","command":{"type":"EndActivation"}}`},
		{"InvalidSenderNumber", `{"sender":99,"command":{"type":"EndActivation"}}`},
		{"InvalidSenderNegative", `{"sender":-1,"command":{"type":"EndActivation"}}`},
		{"NullSender", `{"sender":null,"command":{"type":"EndActivation"}}`},
		{"InvalidCommandInside", `{"sender":"PlayerOne","command":{"type":"InvalidCommand"}}`},
		{"EmptyCommandInside", `{"sender":"PlayerOne","command":{}}`},
		{"NullCommandInside", `{"sender":"PlayerOne","command":null}`},
	}

	for _, tc := range invalidNetworkJSONs {
		t.Run(tc.name, func(t *testing.T) {
			var nc commands.NetworkCommand
			err := json.Unmarshal([]byte(tc.json), &nc)
			if err == nil {
				valErr := nc.Validate()
				if valErr == nil {
					t.Errorf("expected validation error for invalid NetworkCommand payload %s, got nil: %v", tc.name, nc)
				}
			}
		})
	}
}
