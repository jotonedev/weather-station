package controller

import (
	"go.uber.org/fx"

	homepage "weather-station/frontend/controller/homepage"
	nodeadmin "weather-station/frontend/controller/nodeadmin"
	static "weather-station/frontend/controller/static"
	weatherdata "weather-station/frontend/controller/weatherdata"
)

// Module wires service-specific controllers.
var Module = fx.Module("controller",
	fx.Provide(
		weatherdata.New,
		homepage.New,
		nodeadmin.New,
		static.New,
	),
)
