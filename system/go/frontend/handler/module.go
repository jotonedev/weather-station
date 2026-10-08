package handler

import (
	"go.uber.org/fx"

	website "weather-station/frontend/handler/website"
)

var Module = fx.Module("handler",
	fx.Provide(
		website.New,
	),
)
