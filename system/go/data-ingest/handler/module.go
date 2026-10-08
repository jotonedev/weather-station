package handler

import (
	weatherstation "weather-station/data-ingest/handler/weatherstation"

	"go.uber.org/fx"
)

var Module = fx.Module("handler",
	fx.Provide(
		weatherstation.New,
	),
)
