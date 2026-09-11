package e2e

import (
	"wildbuds/internal/commands"
	"wildbuds/internal/domain"
	"wildbuds/internal/resolver"
	"wildbuds/internal/transport"
)

// Harness encapsulates a headless game instance for E2E testing.
type Harness struct {
	State     *domain.GameState
	Queue     *commands.CommandQueue
	Transport transport.Transport
	Resolver  *resolver.Resolver
}

// NewHarness sets up a complete, deterministic E2E test environment (P1 vs P2 via loopback).
func NewHarness() (*Harness, error) {
	state := domain.NewInitialGameState()
	queue := commands.NewCommandQueue()
	trans, err := transport.NewLoopback(
		domain.PlayerOne,
		transport.WithAutoPlayerToggle(true),
		transport.WithValidateOnSend(false),
	)
	if err != nil {
		return nil, err
	}
	res := resolver.NewResolver()

	return &Harness{
		State:     state,
		Queue:     queue,
		Transport: trans,
		Resolver:  res,
	}, nil
}

// InjectCommand simulates a local player taking an action by pushing it to the outgoing queue.
func (h *Harness) InjectCommand(cmd interface{}) {
	switch c := cmd.(type) {
	case commands.GameCommand:
		h.Queue.PushOutgoing(c)
	}
}

// InjectAdversarialPayload simulates receiving a malicious or arbitrary NetworkCommand from the network.
func (h *Harness) InjectAdversarialPayload(netCmd interface{}) {
	switch nc := netCmd.(type) {
	case commands.NetworkCommand:
		h.Queue.PushIncoming(nc)
	case domain.NetworkCommand:
		var gCmd commands.GameCommand
		switch p := nc.Payload.(type) {
		case commands.GameCommand:
			gCmd = p
		}
		h.Queue.PushIncoming(commands.NetworkCommand{
			Sender:  nc.Sender,
			Command: gCmd,
		})
	}
}

// Tick executes one logical update frame: pumping the transport and resolving incoming commands.
func (h *Harness) Tick() error {
	_, _, err := transport.Pump(h.Transport, h.Queue)
	if err != nil {
		return err
	}
	h.Resolver.Resolve(h.State, h.Queue)
	return nil
}
