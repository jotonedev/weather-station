package grpc

import (
	"testing"

	"go.uber.org/fx/fxtest"
	"go.uber.org/zap/zaptest"
)

func TestNewGRPCServer(t *testing.T) {
	lc := fxtest.NewLifecycle(t)
	log := zaptest.NewLogger(t)
	cfg := Config{Address: defaultAddress}

	var registrars []Registrar

	srv := NewGRPCServer(lc, log, cfg, registrars)

	if srv == nil {
		t.Error("Expected non-nil gRPC server")
	}
}
