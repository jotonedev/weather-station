package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type healthRegistrar struct {
	server *health.Server
}

func (registrar *healthRegistrar) Register(serviceRegistrar grpc.ServiceRegistrar) {
	grpc_health_v1.RegisterHealthServer(serviceRegistrar, registrar.server)
}

func reserveFreeLoopbackAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen(NET_PROTOCOL, "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() unexpected error = %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("listener.Close() unexpected error = %v", err)
	}
	return address
}

func TestGRPCServerLifecycle(t *testing.T) {
	testCases := []struct {
		name         string
		setupAddress func(t *testing.T) string
		wantStartErr bool
	}{
		{
			name: "starts serves and stops gracefully",
			setupAddress: func(t *testing.T) string {
				return reserveFreeLoopbackAddress(t)
			},
		},
		{
			name: "returns error when port already in use",
			setupAddress: func(t *testing.T) string {
				listener, err := net.Listen(NET_PROTOCOL, "127.0.0.1:0")
				if err != nil {
					t.Fatalf("net.Listen() unexpected error = %v", err)
				}
				t.Cleanup(func() {
					_ = listener.Close()
				})
				return listener.Addr().String()
			},
			wantStartErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			lifecycle := fxtest.NewLifecycle(t)
			address := testCase.setupAddress(t)
			healthServer := health.NewServer()
			healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

			registrars := []Registrar{&healthRegistrar{server: healthServer}}
			server := NewGRPCServer(lifecycle, zap.NewNop(), Config{Address: address}, registrars)
			if server == nil {
				t.Fatal("NewGRPCServer() returned nil")
			}

			startCtx, cancelStart := context.WithTimeout(context.Background(), time.Second)
			defer cancelStart()

			startErr := lifecycle.Start(startCtx)
			if testCase.wantStartErr {
				if startErr == nil {
					t.Fatal("lifecycle.Start() want error, got nil")
				}
				return
			}
			if startErr != nil {
				t.Fatalf("lifecycle.Start() unexpected error = %v", startErr)
			}

			conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatalf("grpc.NewClient() unexpected error = %v", err)
			}
			t.Cleanup(func() {
				_ = conn.Close()
			})

			rpcCtx, cancelRPC := context.WithTimeout(context.Background(), time.Second)
			defer cancelRPC()

			healthClient := grpc_health_v1.NewHealthClient(conn)
			resp, err := healthClient.Check(rpcCtx, &grpc_health_v1.HealthCheckRequest{})
			if err != nil {
				t.Fatalf("Check() unexpected error = %v", err)
			}
			if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
				t.Fatalf("Check() status = %v, want SERVING", resp.Status)
			}

			stopCtx, cancelStop := context.WithTimeout(context.Background(), time.Second)
			defer cancelStop()
			if err := lifecycle.Stop(stopCtx); err != nil {
				t.Fatalf("lifecycle.Stop() unexpected error = %v", err)
			}

			if _, err := healthClient.Check(rpcCtx, &grpc_health_v1.HealthCheckRequest{}); err == nil {
				t.Fatal("Check() after stop want error, got nil")
			}
		})
	}
}
