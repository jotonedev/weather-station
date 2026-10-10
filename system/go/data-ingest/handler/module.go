package handler

import (
	"weather-station/common/fxgroup"
	"weather-station/data-ingest/handler/reading"

	"go.uber.org/fx"
)

// Module registers HTTP routes for data-ingest.
var Module = fx.Module("handler",
	fx.Provide(
		fxgroup.AsHTTPRoute(reading.New),
		fxgroup.AsHTTPRoute(reading.NewBatch),
	),
)
