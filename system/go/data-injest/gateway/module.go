package gateway

import (
	"go.uber.org/fx"
)

// Module wires service-specific gateways (outbound dependencies).
var Module = fx.Module("gateway",
	fx.Provide(),
)
