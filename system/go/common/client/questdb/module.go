package questdb

import (
	"go.uber.org/fx"
)

var Module = fx.Module("questdb-client",
	fx.Provide(
		NewConfig,
		NewClient,
	),
)
