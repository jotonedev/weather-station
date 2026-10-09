package handler

import (
	"go.uber.org/fx"

	"weather-station/frontend/handler/website"
)

var Module = fx.Module("handler",
	fx.Provide(
		website.New,
	),
)
