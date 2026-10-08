# Backend & Frontend for the Weather Station

This folder contains the backend and frontend for the weather station.

It uses a microservice architecture with gRPC being used to communicate between services.

## Repository layout

This is a multi-module Go workspace (`go.work`) with a Rust inference server and a single source of truth for protobuf contracts.

```
system/
├── go.work                    # wires the Go modules together
├── proto/                     # single source of truth for .proto files
│   ├── buf.yaml               # lint & breaking-change config
│   ├── buf.gen.yaml           # codegen plugins (Go + Rust)
│   ├── weather/.../*.proto    # service contracts
│   └── gen/                   # generated stubs (committed)
│       ├── go/                #   Go module weather-station/gen/go
│       └── rust/              #   prost/tonic crate
├── go/
│   ├── common/                # shared infrastructure module
│   │   ├── server/            #   shared HTTP + gRPC servers
│   │   ├── client/            #   shared HTTP + gRPC clients
│   │   ├── health/            #   shared health-check (registers via fx groups)
│   │   └── fxgroup/           #   shared helpers for tagging handlers into fx groups
│   ├── database/              # service (duckdb storage)
│   ├── data-injest/           # service (ingests MCU data)
│   ├── frontend/              # service (HTTP for data)
│   └── inference-job/         # service (cron prediction)
└── rust/
    └── inference-server/        # Rust inference server (chronos-2)
```

Each Go service owns its service-specific code — `handler/` (endpoints),
`gateway/` (outbound dependencies), `controller/` — plus a thin
`cmd/main.go` that wires those together with the shared `common` fx
modules (`server`, `client`, `health`).

## Development

```sh
# build everything in the workspace
go build ./go/...

# build / run a single service
cd go/frontend && go run ./cmd
```

## Regenerating protobuf code

Protos live in `proto/`; generated Go and Rust stubs under `proto/gen/` are
committed and regenerated automatically by the CI workflow
(`.github/workflows/proto-gen.yml`) whenever a `.proto` file changes.

To regenerate locally (requires `buf`, `protoc-gen-go`,
`protoc-gen-go-grpc`, `protoc-gen-prost`, `protoc-gen-tonic`):

```sh
cd proto && buf generate
```

Run lint / breaking checks locally with:

```sh
cd proto && buf lint && buf breaking --against "${BUF_BREAKING_AGAINST}"
```

## Infrastructure

- `database` (golang): duckdb database for weather ground and prediction data
- `frontend` (golang): http interface for the data
- `data-injest` (golang): injest service for the data coming from the MCU
- `inference-job` (golang): cron-job that retrieves data from database and predicts the future weather
- `inference-server` (rust): inference server for chronos-2 model. This is splitted from `inference-job` to allow it to be deployed on more performant nodes or to be replaced it with commercial services.



```mermaid
architecture-beta
    group api(cloud)[API]
    group source(cloud)[Weather Stations]

    service db(database)[Database] in api
    service inference-server(server)[Inference Server] in api
    service inference-job(server)[Inference Job] in api
    service data-injest(server)[Data Injest] in api
    service frontend(server)[Frontend] in api

    service weather-station(server)[Weather Station] in source

    db:R -- L:inference-job
    inference-job:R -- L:inference-server
    db:T -- B:frontend
    db:L -- R:data-injest
    weather-station:T -- B:data-injest
```
