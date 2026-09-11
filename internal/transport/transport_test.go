package transport_test

import (
	"errors"
	"os/exec"
	"strings"
	"testing"

	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
	"wildbuds/internal/transport"
)

// TestTransport_InterfaceContract verifies at compile time and runtime that
// LoopbackTransport implements the Transport and PollingTransport interfaces.
func TestTransport_InterfaceContract(t *testing.T) {
	// Compile-time assertions
	var _ transport.Transport = (*transport.LoopbackTransport)(nil)
	var _ transport.PollingTransport = (*transport.LoopbackTransport)(nil)

	// Runtime instantiation check
	lt, err := transport.NewLoopback(domain.PlayerOne)
	if err != nil {
		t.Fatalf("expected NewLoopback to succeed, got %v", err)
	}
	defer lt.Close()

	var tr transport.Transport = lt
	if tr.LocalPlayer() != domain.PlayerOne {
		t.Errorf("expected LocalPlayer %s, got %s", domain.PlayerOne, tr.LocalPlayer())
	}
	if tr.RemotePlayer() != domain.PlayerTwo {
		t.Errorf("expected RemotePlayer %s, got %s", domain.PlayerTwo, tr.RemotePlayer())
	}
	if tr.IsClosed() {
		t.Errorf("expected new transport to not be closed")
	}
}

// TestTransport_StandardErrors ensures expected error sentinels are exported, distinct, and match contract.
func TestTransport_StandardErrors(t *testing.T) {
	errorsList := []struct {
		name string
		err  error
	}{
		{"ErrTransportClosed", transport.ErrTransportClosed},
		{"ErrQueueFull", transport.ErrQueueFull},
		{"ErrBufferFull", transport.ErrBufferFull},
		{"ErrInvalidCommand", transport.ErrInvalidCommand},
		{"ErrInvalidPlayer", transport.ErrInvalidPlayer},
		{"ErrNotConnected", transport.ErrNotConnected},
		{"ErrTimeout", transport.ErrTimeout},
		{"ErrNilQueue", transport.ErrNilQueue},
	}

	for _, item := range errorsList {
		if item.err == nil {
			t.Errorf("expected error %s to be non-nil", item.name)
		}
	}

	// Distinctness checks
	if errors.Is(transport.ErrTransportClosed, transport.ErrInvalidPlayer) {
		t.Errorf("ErrTransportClosed should not equal ErrInvalidPlayer")
	}
	if errors.Is(transport.ErrTransportClosed, transport.ErrQueueFull) {
		t.Errorf("ErrTransportClosed should not equal ErrQueueFull")
	}
	if !errors.Is(transport.ErrBufferFull, transport.ErrQueueFull) {
		t.Errorf("ErrBufferFull should match ErrQueueFull")
	}
}

// TestTransport_Decoupling ensures internal/transport does not take forbidden dependencies
// on domain.GameState or resolver.Resolver, protecting architectural decoupling.
func TestTransport_Decoupling(t *testing.T) {
	cmd := exec.Command("go", "list", "-f", "{{.Imports}}", "wildbuds/internal/transport")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to query package imports via go list: %v (output: %s)", err, string(out))
	}

	importStr := string(out)
	if strings.Contains(importStr, "wildbuds/internal/resolver") {
		t.Errorf("architectural violation: internal/transport must NOT import wildbuds/internal/resolver")
	}
	if strings.Contains(importStr, "wildbuds/internal/domain/state") {
		t.Errorf("architectural violation: internal/transport must NOT import domain state directly")
	}
}

// TestTransport_Polymorphism exercises Transport consumers using interface polymorphism.
func TestTransport_Polymorphism(t *testing.T) {
	lt, err := transport.NewLoopback(domain.PlayerTwo, transport.WithBufferSize(32))
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	// Consumer accepting abstract Transport interface
	sendAndCollect := func(tr transport.Transport, cmd commands.GameCommand) (commands.NetworkCommand, error) {
		if err := tr.Send(cmd); err != nil {
			return commands.NetworkCommand{}, err
		}
		received := <-tr.Receive()
		return received, nil
	}

	cmd := commands.NewEndActivationCommand()
	netCmd, err := sendAndCollect(lt, cmd)
	if err != nil {
		t.Fatalf("sendAndCollect failed: %v", err)
	}

	if netCmd.Sender != domain.PlayerTwo {
		t.Errorf("expected sender %s, got %s", domain.PlayerTwo, netCmd.Sender)
	}
	if netCmd.Command.Type != commands.CommandEndActivation {
		t.Errorf("expected command EndActivation, got %s", netCmd.Command.Type)
	}
}

// TestTransport_Pump_NilGuards verifies error handling when Pump is invoked with nil parameters.
func TestTransport_Pump_NilGuards(t *testing.T) {
	q := commands.NewCommandQueue()
	lt, err := transport.NewLoopback(domain.PlayerOne)
	if err != nil {
		t.Fatalf("NewLoopback failed: %v", err)
	}
	defer lt.Close()

	// Nil transport
	sent, recv, err := transport.Pump(nil, q)
	if err == nil {
		t.Errorf("expected error with nil transport, got nil")
	}
	if sent != 0 || recv != 0 {
		t.Errorf("expected 0, 0 with nil transport, got %d, %d", sent, recv)
	}

	// Nil queue
	sent, recv, err = transport.Pump(lt, nil)
	if !errors.Is(err, transport.ErrNilQueue) {
		t.Errorf("expected ErrNilQueue with nil queue, got %v", err)
	}
	if sent != 0 || recv != 0 {
		t.Errorf("expected 0, 0 with nil queue, got %d, %d", sent, recv)
	}
}
