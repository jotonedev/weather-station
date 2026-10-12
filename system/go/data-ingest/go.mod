module weather-station/data-ingest

go 1.27

require (
	go.uber.org/fx v1.24.0
	go.uber.org/mock v0.6.0
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.84.0
	weather-station/common v0.0.0
	weather-station/gen/go v0.0.0-00010101000000-000000000000
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260706201446-f0a921348800 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace weather-station/common => ../common

replace weather-station/gen/go => ../../proto/gen/go
