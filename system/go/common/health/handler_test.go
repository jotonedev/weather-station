package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type staticChecker struct {
	name string
	err  error
}

func (s *staticChecker) Name() string {
	return s.name
}

func (s *staticChecker) Check(_ context.Context) error {
	return s.err
}

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "returns ok",
			wantStatus: http.StatusOK,
			wantBody:   "ok",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler, err := NewHealth(HealthHandlerParams{Logger: zap.NewNop()})
			if err != nil {
				t.Fatalf("unexpected error creating handler: %v", err)
			}

			if handler.Pattern() != "/health" {
				t.Errorf("expected pattern /health, got %s", handler.Pattern())
			}

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}

			if rec.Body.String() != tc.wantBody {
				t.Errorf("expected body %q, got %q", tc.wantBody, rec.Body.String())
			}
		})
	}
}

func TestReadyHandler(t *testing.T) {
	tests := []struct {
		name       string
		checkers   []Checker
		wantStatus int
		wantBody   string
	}{
		{
			name:       "empty checkers returns ok",
			checkers:   nil,
			wantStatus: http.StatusOK,
			wantBody:   "ok",
		},
		{
			name: "passing checker returns ok",
			checkers: []Checker{
				&staticChecker{name: "db", err: nil},
			},
			wantStatus: http.StatusOK,
			wantBody:   "ok",
		},
		{
			name: "failing checker returns 503",
			checkers: []Checker{
				&staticChecker{name: "db", err: errors.New("connection refused")},
			},
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   "db: connection refused\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler, err := NewReady(ReadyHandlerParams{
				Logger:   zap.NewNop(),
				Checkers: tc.checkers,
			})
			if err != nil {
				t.Fatalf("unexpected error creating handler: %v", err)
			}

			if handler.Pattern() != "/ready" {
				t.Errorf("expected pattern /ready, got %s", handler.Pattern())
			}

			req := httptest.NewRequest(http.MethodGet, "/ready", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}

			if rec.Body.String() != tc.wantBody {
				t.Errorf("expected body %q, got %q", tc.wantBody, rec.Body.String())
			}
		})
	}
}

func TestGRPCHandler(t *testing.T) {
	tests := []struct {
		name        string
		checkers    []Checker
		service     string
		wantStatus  grpc_health_v1.HealthCheckResponse_ServingStatus
		expectError bool
	}{
		{
			name:       "default service returns serving",
			checkers:   nil,
			service:    "",
			wantStatus: grpc_health_v1.HealthCheckResponse_SERVING,
		},
		{
			name:       "ready service with no checkers returns serving",
			checkers:   nil,
			service:    "ready",
			wantStatus: grpc_health_v1.HealthCheckResponse_SERVING,
		},
		{
			name: "ready service with passing checkers returns serving",
			checkers: []Checker{
				&staticChecker{name: "storage", err: nil},
			},
			service:    "ready",
			wantStatus: grpc_health_v1.HealthCheckResponse_SERVING,
		},
		{
			name: "ready service with failing checker returns not serving",
			checkers: []Checker{
				&staticChecker{name: "storage", err: errors.New("timeout")},
			},
			service:    "ready",
			wantStatus: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lc := fxtest.NewLifecycle(t)
			h := NewGRPC(lc, ReadyHandlerParams{
				Logger:   zap.NewNop(),
				Checkers: tc.checkers,
			})

			server := grpc.NewServer()
			h.Register(server)

			checkServer := &grpcHealthServer{
				HealthServer: h.base,
				checkers:     tc.checkers,
			}

			resp, err := checkServer.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{
				Service: tc.service,
			})

			if tc.expectError && err == nil {
				t.Fatal("expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Status != tc.wantStatus {
				t.Errorf("expected status %v, got %v", tc.wantStatus, resp.Status)
			}
		})
	}
}
