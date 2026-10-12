package main

import (
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func TestHealthProbeRun(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()

	healthServer := health.NewServer()
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("ready", grpc_health_v1.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("broken", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	grpcServer := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	go func() {
		_ = grpcServer.Serve(lis)
	}()
	defer grpcServer.Stop()

	addr := lis.Addr().String()

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "checks default service successfully",
			args:    []string{"-addr=" + addr},
			wantErr: false,
		},
		{
			name:    "checks ready service successfully",
			args:    []string{"-addr=" + addr, "-service=ready"},
			wantErr: false,
		},
		{
			name:    "checks broken service returns error",
			args:    []string{"-addr=" + addr, "-service=broken"},
			wantErr: true,
		},
		{
			name:    "nonexistent address returns error",
			args:    []string{"-addr=127.0.0.1:1", "-timeout=50ms"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := run(tc.args)
			if tc.wantErr && err == nil {
				t.Fatal("expected error but got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
