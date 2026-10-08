package main

import (
	"os"

	"weather-station/common/client"
	"weather-station/common/health"
	"weather-station/common/server"

	"weather-station/data-injest/controller"
	"weather-station/data-injest/gateway"
	"weather-station/data-injest/handler"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
)

func main() {
	fx.New(
		// Shared ingress / egress modules
		server.Module,
		client.Module,
		// Shared health check (HTTP + gRPC)
		health.Module,
		// Utility modules
		fx.Provide(getLogger),
		fx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
		// Service-specific modules
		handler.Module,
		gateway.Module,
		controller.Module,
	).Run()
}

func getLogger() (*zap.Logger, error) {
	if os.Getenv("GO_ENV") == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}
