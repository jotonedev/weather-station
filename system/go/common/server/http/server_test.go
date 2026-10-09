package http

import (
	"net/http"
	"testing"

	"go.uber.org/fx/fxtest"
	"go.uber.org/zap/zaptest"
)

func TestNewHTTPServer(t *testing.T) {
	lc := fxtest.NewLifecycle(t)
	mux := http.NewServeMux()
	log := zaptest.NewLogger(t)
	cfg := Config{Address: defaultAddress}

	srv := NewHTTPServer(lc, mux, log, cfg)

	if srv == nil {
		t.Error("Expected non-nil HTTP server")
	}

	if srv.Addr != cfg.Address {
		t.Errorf("Expected address %s, got %s", cfg.Address, srv.Addr)
	}
}
