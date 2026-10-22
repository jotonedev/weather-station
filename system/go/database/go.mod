module weather-station/database

go 1.27

require (
	github.com/duckdb/duckdb-go/v2 v2.20000.0
	go.uber.org/fx v1.24.0
	go.uber.org/zap v1.28.0
	google.golang.org/grpc v1.84.0
	weather-station/common v0.0.0
	weather-station/gen/go v0.0.0-00010101000000-000000000000
)

require (
	github.com/apache/arrow-go/v18 v18.5.1 // indirect
	github.com/duckdb/duckdb-go-bindings v0.20000.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/darwin-amd64 v0.10506.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/darwin-arm64 v0.10506.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/linux-amd64 v0.10506.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/linux-arm64 v0.10506.0 // indirect
	github.com/duckdb/duckdb-go-bindings/lib/windows-amd64 v0.10506.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/google/flatbuffers v25.12.19+incompatible // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.3 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.25 // indirect
	github.com/zeebo/xxh3 v1.1.0 // indirect
	go.duckdb.org/libduckdb/darwin-amd64 v0.20000.0 // indirect
	go.duckdb.org/libduckdb/darwin-arm64 v0.20000.0 // indirect
	go.duckdb.org/libduckdb/linux-amd64 v0.20000.0 // indirect
	go.duckdb.org/libduckdb/linux-arm64 v0.20000.0 // indirect
	go.duckdb.org/libduckdb/windows-amd64 v0.20000.0 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20260112195511-716be5621a96 // indirect
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/telemetry v0.0.0-20260625142307-59b4966ccb57 // indirect
	golang.org/x/text v0.40.0 // indirect
	golang.org/x/tools v0.47.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260706201446-f0a921348800 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace weather-station/common => ../common

replace weather-station/gen/go => ../../proto/gen/go
