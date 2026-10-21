# Weather Station: Docker Compose

Local deployment of the weather-station microservices using compiled container images.

## Usage

```bash
# start services (frontend, data-ingest, database, inference-server)
docker compose up -d

# or point to your own container registry:
# REGISTRY=myregistry.example.com/user docker compose up -d

# web dashboard (http://localhost:8080)

# submit a reading (fields follow the data-ingest OpenAPI spec)
curl -X POST http://localhost:8081/reading -H 'Content-Type: application/json' \
  -d '{"node_id":"station-alpha","timestamp":"2026-10-20T12:00:00Z","temperature":18.5,"humidity":61.0}'

# stop services
docker compose down
# wipe stored data too
docker compose down -v
```

## Forecasts

Run the forecast job on demand. It pulls history from `database`, calls `inference-server`, and writes predictions back:

```bash
docker compose --profile job run --rm inference-job
```

To schedule it like the cluster CronJob (hourly), add a host cron entry:

```
0 * * * * cd /path/to/weather-station/system/deploy/compose && docker compose --profile job run --rm inference-job
```
