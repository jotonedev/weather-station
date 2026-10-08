package controller

import (
	"go.uber.org/fx"
)

// Module wires service-specific controllers.
var Module = fx.Module("controller",
	fx.Provide(),
)
