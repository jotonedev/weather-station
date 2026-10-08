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

	srv := NewHTTPServer(lc, mux, log)

	if srv == nil {
		t.Error("Expected non-nil HTTP server")
	}

	if srv.Addr != NET_ADDRESS {
		t.Errorf("Expected address %s, got %s", NET_ADDRESS, srv.Addr)
	}
}
