module weather-station/database

go 1.26

require (
	github.com/jackc/pgx/v5 v5.11.0
	go.uber.org/fx v1.24.0
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.84.0
	weather-station/common v0.0.0
	weather-station/gen/go v0.0.0-00010101000000-000000000000
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260706201446-f0a921348800 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace weather-station/common => ../common

replace weather-station/gen/go => ../../proto/gen/go
