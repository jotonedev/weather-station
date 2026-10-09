package grpc

import (
	"testing"

	"go.uber.org/fx/fxtest"
	"go.uber.org/zap/zaptest"
)

func TestNewGRPCServer(t *testing.T) {
	lc := fxtest.NewLifecycle(t)
	log := zaptest.NewLogger(t)

	// No registrars needed for this test
	var registrars []Registrar

	srv := NewGRPCServer(lc, log, registrars)

	if srv == nil {
		t.Error("Expected non-nil gRPC server")
	}

	// Verify lifecycle hooks
	// In a real test we might want to ensure hooks are appended, but fx.Lifecycle doesn't expose that easily.
	// fxtest.Lifecycle verifies that hooks run without error during Start/Stop.
	// However, we agreed NOT to run the server.
	// So just verifying NewGRPCServer returns a server is sufficient as per requirements.
}
