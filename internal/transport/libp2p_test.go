package transport_test

import (
	"testing"

	"wildbuds/internal/transport"
)

func TestLibp2pTransport_InterfaceAndLifecycle(t *testing.T) {
	var _ transport.Transport = (*transport.Libp2pTransport)(nil)

	tr, err := transport.NewLibp2pTransport()
	if err != nil {
		t.Fatalf("failed to create Libp2pTransport: %v", err)
	}
	defer tr.Close()

	if tr.IsClosed() {
		t.Errorf("expected new transport not to be closed")
	}
	if tr.IsConnected() {
		t.Errorf("expected new transport not to be connected initially")
	}

	if err := tr.Close(); err != nil {
		t.Errorf("failed to close transport: %v", err)
	}
	if !tr.IsClosed() {
		t.Errorf("expected transport to be closed after Close()")
	}
}
