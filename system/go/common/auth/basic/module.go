package basic

import "go.uber.org/fx"

// Module provides HTTP Basic Authentication configuration and authenticator instances.
var Module = fx.Module("basic-auth",
	fx.Provide(
		NewConfig,
		NewAuthenticator,
	),
)
