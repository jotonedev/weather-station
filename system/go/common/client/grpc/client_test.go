package grpc

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type blockingHealthServer struct {
	grpc_health_v1.UnimplementedHealthServer
}

func (server *blockingHealthServer) Check(ctx context.Context, _ *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func startTestHealthServer(t *testing.T, service grpc_health_v1.HealthServer) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() unexpected error = %v", err)
	}

	server := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(server, service)

	go func() {
		_ = server.Serve(listener)
	}()

	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})

	return listener.Addr().String()
}

func reserveClosedAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() unexpected error = %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("listener.Close() unexpected error = %v", err)
	}
	return address
}

func TestClientFactoryNewConnection(t *testing.T) {
	testCases := []struct {
		name           string
		setupTarget    func(t *testing.T) string
		startTimeout   time.Duration
		wantNewConnErr bool
		wantStartErr   bool
		wantStartErrIs error
		verifyAfter    bool
	}{
		{
			name: "serving health check succeeds and stop closes connection",
			setupTarget: func(t *testing.T) string {
				healthServer := health.NewServer()
				healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
				return startTestHealthServer(t, healthServer)
			},
			verifyAfter: true,
		},
		{
			name: "not serving health check fails startup",
			setupTarget: func(t *testing.T) string {
				healthServer := health.NewServer()
				healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
				return startTestHealthServer(t, healthServer)
			},
			startTimeout:   time.Second,
			wantStartErr:   true,
			wantStartErrIs: ErrNotServing,
		},
		{
			name: "unreachable target fails startup",
			setupTarget: func(t *testing.T) string {
				return reserveClosedAddress(t)
			},
			startTimeout: 200 * time.Millisecond,
			wantStartErr: true,
		},
		{
			name: "slow health check times out on startup",
			setupTarget: func(t *testing.T) string {
				return startTestHealthServer(t, &blockingHealthServer{})
			},
			startTimeout: 30 * time.Millisecond,
			wantStartErr: true,
		},
		{
			name: "invalid target fails before lifecycle hooks",
			setupTarget: func(t *testing.T) string {
				return "dns:///%zz"
			},
			wantNewConnErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			lifecycle := fxtest.NewLifecycle(t)
			factory := NewClientFactory(lifecycle, zap.NewNop())
			target := testCase.setupTarget(t)

			conn, err := factory.NewConnection(target)
			if testCase.wantNewConnErr {
				if err == nil {
					if conn != nil {
						_ = conn.Close()
					}
					t.Fatal("NewConnection() want error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewConnection() unexpected error = %v", err)
			}
			t.Cleanup(func() {
				_ = conn.Close()
			})

			startCtx := context.Background()
			if testCase.startTimeout > 0 {
				var cancelStart context.CancelFunc
				startCtx, cancelStart = context.WithTimeout(startCtx, testCase.startTimeout)
				defer cancelStart()
			}

			startErr := lifecycle.Start(startCtx)
			if testCase.wantStartErr {
				if startErr == nil {
					t.Fatal("lifecycle.Start() want error, got nil")
				}
				if testCase.wantStartErrIs != nil && !errors.Is(startErr, testCase.wantStartErrIs) {
					t.Fatalf("lifecycle.Start() error = %v, want %v", startErr, testCase.wantStartErrIs)
				}
				return
			}
			if startErr != nil {
				t.Fatalf("lifecycle.Start() unexpected error = %v", startErr)
			}

			if testCase.verifyAfter {
				checkCtx, cancelCheck := context.WithTimeout(context.Background(), time.Second)
				defer cancelCheck()

				healthClient := grpc_health_v1.NewHealthClient(conn)
				resp, err := healthClient.Check(checkCtx, &grpc_health_v1.HealthCheckRequest{})
				if err != nil {
					t.Fatalf("Check() before stop unexpected error = %v", err)
				}
				if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
					t.Fatalf("Check() status = %v, want SERVING", resp.Status)
				}

				stopCtx, cancelStop := context.WithTimeout(context.Background(), time.Second)
				defer cancelStop()
				if err := lifecycle.Stop(stopCtx); err != nil {
					t.Fatalf("lifecycle.Stop() unexpected error = %v", err)
				}

				if _, err := healthClient.Check(checkCtx, &grpc_health_v1.HealthCheckRequest{}); err == nil {
					t.Fatal("Check() after stop want error on closed connection, got nil")
				}
			}
		})
	}
}
