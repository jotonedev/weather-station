package health

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type staticChecker struct {
	name string
	err  error
}

func (checker *staticChecker) Name() string {
	return checker.name
}

func (checker *staticChecker) Check(_ context.Context) error {
	return checker.err
}

func TestHealthHandler(t *testing.T) {
	testCases := []struct {
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

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			handler, err := NewHealth(HealthHandlerParams{Logger: zap.NewNop()})
			if err != nil {
				t.Fatalf("unexpected error creating handler: %v", err)
			}

			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != testCase.wantStatus {
				t.Errorf("expected status %d, got %d", testCase.wantStatus, rec.Code)
			}

			if rec.Body.String() != testCase.wantBody {
				t.Errorf("expected body %q, got %q", testCase.wantBody, rec.Body.String())
			}
		})
	}
}

func TestReadyHandler(t *testing.T) {
	testCases := []struct {
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

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			handler, err := NewReady(ReadyHandlerParams{
				Logger:   zap.NewNop(),
				Checkers: testCase.checkers,
			})
			if err != nil {
				t.Fatalf("unexpected error creating handler: %v", err)
			}

			req := httptest.NewRequest(http.MethodGet, "/ready", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != testCase.wantStatus {
				t.Errorf("expected status %d, got %d", testCase.wantStatus, rec.Code)
			}

			if rec.Body.String() != testCase.wantBody {
				t.Errorf("expected body %q, got %q", testCase.wantBody, rec.Body.String())
			}
		})
	}
}

func TestGRPCHandler(t *testing.T) {
	testCases := []struct {
		name       string
		checkers   []Checker
		service    string
		stopBefore bool
		wantStatus grpc_health_v1.HealthCheckResponse_ServingStatus
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
		{
			name:       "default service returns not serving after lifecycle stop",
			checkers:   nil,
			service:    "",
			stopBefore: true,
			wantStatus: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			lifecycle := fxtest.NewLifecycle(t)
			handler := NewGRPC(lifecycle, ReadyHandlerParams{
				Logger:   zap.NewNop(),
				Checkers: testCase.checkers,
			})

			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("net.Listen() unexpected error = %v", err)
			}

			server := grpc.NewServer()
			handler.Register(server)
			go func() {
				_ = server.Serve(listener)
			}()
			t.Cleanup(func() {
				server.Stop()
				_ = listener.Close()
			})

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			if err := lifecycle.Start(ctx); err != nil {
				t.Fatalf("lifecycle.Start() unexpected error = %v", err)
			}
			if testCase.stopBefore {
				if err := lifecycle.Stop(ctx); err != nil {
					t.Fatalf("lifecycle.Stop() unexpected error = %v", err)
				}
			}

			conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("grpc.NewClient() unexpected error = %v", err)
			}
			t.Cleanup(func() {
				_ = conn.Close()
			})

			healthClient := grpc_health_v1.NewHealthClient(conn)
			resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{
				Service: testCase.service,
			})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Status != testCase.wantStatus {
				t.Errorf("expected status %v, got %v", testCase.wantStatus, resp.Status)
			}
		})
	}
}
