package grpc

//go:generate go run go.uber.org/mock/mockgen -package mocks -destination ../../internal/mocks/grpc_health_client.go google.golang.org/grpc/health/grpc_health_v1 HealthClient

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// ClientFactory helps create gRPC clients with proper lifecycle management.
type ClientFactory struct {
	lc  fx.Lifecycle
	log *zap.Logger
}

func NewClientFactory(lc fx.Lifecycle, log *zap.Logger) *ClientFactory {
	return &ClientFactory{
		lc:  lc,
		log: log,
	}
}

// NewConnection creates a connection to a specific target.
// access to *grpc.ClientConn is managed by Fx lifecycle.
func (f *ClientFactory) NewConnection(target string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		f.log.Error("Failed to create gRPC client", zap.String("target", target), zap.Error(err))
		return nil, err
	}

	f.lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			f.log.Info("Checking gRPC server health", zap.String("target", target))
			client := grpc_health_v1.NewHealthClient(conn)
			resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
			if err != nil {
				// Fail startup fast so an unreachable dependency surfaces immediately,
				// mirroring the strict QuestDB ping.
				f.log.Error("gRPC health check failed", zap.String("target", target), zap.Error(err))
				return err
			}
			f.log.Info("gRPC health check successful", zap.String("target", target), zap.String("status", resp.Status.String()))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			f.log.Info("Closing gRPC client connection", zap.String("target", target))
			return conn.Close()
		},
	})

	return conn, nil
}
