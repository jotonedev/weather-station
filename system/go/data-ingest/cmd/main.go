package main

import (
	"weather-station/common/client"
	"weather-station/common/health"
	"weather-station/common/logging"
	"weather-station/common/server"

	"weather-station/data-ingest/controller"
	"weather-station/data-ingest/gateway"
	"weather-station/data-ingest/handler"

	"go.uber.org/fx"
)

func main() {
	fx.New(
		// Shared ingress / egress modules
		server.Module,
		client.Module,
		// Shared logging and health checks (HTTP + gRPC)
		logging.Module,
		health.Module,
		// Service-specific modules
		handler.Module,
		gateway.Module,
		controller.Module,
	).Run()
}
