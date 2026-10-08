module weather-station/inference-job

go 1.26

require (
	go.uber.org/fx v1.24.0
	go.uber.org/zap v1.27.0
	weather-station/common v0.0.0
)

require (
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/grpc v1.79.1 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)

replace weather-station/common => ../common
