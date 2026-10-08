package grpc

import (
	"context"
	"net"

	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

const (
	NET_PROTOCOL = "tcp"
	NET_ADDRESS  = "0.0.0.0:8090"
)

func NewGRPCServer(lc fx.Lifecycle, log *zap.Logger, registrars []func(*grpc.Server)) *grpc.Server {
	srv := grpc.NewServer()

	for _, register := range registrars {
		register(srv)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen(NET_PROTOCOL, NET_ADDRESS)
			if err != nil {
				return err
			}
			log.Info("Starting gRPC server at", zap.String("addr", NET_ADDRESS))

			go func() {
				if err := srv.Serve(ln); err != nil {
					log.Error("gRPC server failed", zap.Error(err))
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			srv.GracefulStop()
			return nil
		},
	})

	return srv
}
