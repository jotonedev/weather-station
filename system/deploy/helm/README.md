# Weather Station Helm Chart

This directory contains the production-ready Helm chart (`weather-station`) for deploying the cloud-native weather station platform on Kubernetes (k3s).

---

## Directory & File Overview

```text
system/deploy/helm/
├── README.md
└── weather-station/
    ├── Chart.yaml
    ├── values.yaml
    └── templates/
        ├── _helpers.tpl
        ├── NOTES.txt
        ├── deployment-database.yaml
        ├── service-database.yaml
        ├── pvc-database.yaml
        ├── deployment-data-ingest.yaml
        ├── service-data-ingest.yaml
        ├── deployment-frontend.yaml
        ├── service-frontend.yaml
        ├── deployment-inference-server.yaml
        ├── service-inference-server.yaml
        ├── pvc-model-cache.yaml
        ├── cronjob-inference.yaml
        ├── ingress.yaml
        └── networkpolicy.yaml
```

---

## Chart Root Files

### `Chart.yaml`
Declares Helm chart metadata including chart name (`weather-station`), version (`0.1.0`), target application version (`0.1.0`), description, and project maintainers.

### `values.yaml`
Central configuration contract for the chart with configurable parameters:
- Global settings (`global.storageClass`, `global.imagePullSecrets`)
- Microservice image repositories, tags, and pull policies
- CPU and memory resource requests and limits (low allocations for Go microservices, dedicated heavy resources for ML inference)
- Hardware placement rules (`nodeSelector`, `tolerations`, `affinity`)
- Persistent storage sizes and StorageClass selections (`truenas-iscsi` for database, `local-path` for model cache)
- Traefik ingress hostname and path routing
- NetworkPolicy toggles

---

## Template Manifests (`templates/`)

### 1. Helpers & Notes

#### `templates/_helpers.tpl`
Defines reusable Go template helpers:
- `weather-station.fullname`: Generates standard, truncated (<=63 characters) resource names.
- `weather-station.componentFullname`: Generates component-specific names (e.g. `release-weather-station-database`).
- `weather-station.labels` and `weather-station.componentLabels`: Generates standardized Kubernetes metadata labels (`app.kubernetes.io/name`, `app.kubernetes.io/instance`, `app.kubernetes.io/component`, `helm.sh/chart`).

#### `templates/NOTES.txt`
Rendered by Helm immediately upon release installation. Displays practical post-deployment instructions including frontend ingress URLs, log tailing commands, and pod status commands.

---

### 2. Embedded Database Tier

#### `templates/deployment-database.yaml`
Deploys the single-replica `database` microservice:
- Runs Go service with the embedded DuckDB columnar engine (`/data/weather.duckdb`).
- Eliminates external database servers, running with a lightweight ~50 MB RAM footprint.
- Mounts persistent storage at `/data`.
- Pinned to node `rpi3` via `nodeSelector` and tolerates the `dedicated=rpi3:PreferNoSchedule` taint.
- Uses `Recreate` deployment strategy to ensure safe single-writer file locking.
- Configures native Kubernetes gRPC liveness and readiness health checks on port 8090.

#### `templates/service-database.yaml`
Internal `ClusterIP` Service exposing gRPC port 8090. Allows `data-ingest`, `frontend`, and `inference-job` to communicate with the database service over gRPC.

#### `templates/pvc-database.yaml`
Creates the `PersistentVolumeClaim` (`10Gi`) using the `truenas-iscsi` StorageClass, binding to the iSCSI target portal on `rpi3`. Supports overriding with `existingClaim` if a pre-existing PVC is specified.

---

### 3. Ingest & Frontend Microservices

#### `templates/deployment-data-ingest.yaml`
Deploys the `data-ingest` microservice:
- Exposes HTTP port 8080 for sensor reading submissions via `POST /reading`.
- Sends parsed readings to `database` over gRPC port 8090.
- Pinned to node `rpi3` alongside the database service.
- Configures native gRPC readiness probe on port 8090.

#### `templates/service-data-ingest.yaml`
Internal `ClusterIP` Service exposing port 8080 (HTTP) for ingress traffic and port 8090 (gRPC) for health probes.

#### `templates/deployment-frontend.yaml`
Deploys the web dashboard UI:
- Serves the dashboard HTML and assets on HTTP port 8080.
- Queries `database` via gRPC port 8090 to render real-time and aggregate charts.
- Configures native gRPC readiness probe on port 8090.

#### `templates/service-frontend.yaml`
Internal `ClusterIP` Service exposing port 8080 (HTTP) for ingress traffic and port 8090 (gRPC) for health probes.

---

### 4. Machine Learning Inference & Batch Job

#### `templates/deployment-inference-server.yaml`
Deploys the Python PyTorch model serving workload. The active model family is selected
through `inferenceServer.modelBackend` and `inferenceServer.modelName` in `values.yaml`
(default `timestral` with `mistralai/Timestral-209M`, alternative `chronos-2`):
- Pinned to large nodes with label `node-size=large` (`arm-01` and `arm-02` with ~12 GB RAM).
- Allocated 1-4 CPU cores and 2-8 GB RAM.
- Mounts dedicated model cache PVC at `/home/appuser/.cache/huggingface` with `fsGroup: 10001` so non-root `appuser` can persist model weights across restarts.
- Exposes gRPC port 50051 with native gRPC readiness check.

#### `templates/service-inference-server.yaml`
Internal `ClusterIP` Service exposing gRPC port 50051 for inference forecast requests.

#### `templates/pvc-model-cache.yaml`
PersistentVolumeClaim (10Gi, `local-path` StorageClass) providing persistent storage for downloaded HuggingFace model weights.

#### `templates/cronjob-inference.yaml`
Kubernetes `batch/v1` `CronJob` that executes periodic forecasts (default: hourly `0 * * * *`):
- Queries historical time-series weather readings from `database:8090`.
- Dispatches inference requests to `inference-server:50051`.
- Writes forecast predictions back into `database:8090`.
- Configured with `concurrencyPolicy: Forbid`.

---

### 5. Ingress & Zero-Trust Security

#### `templates/ingress.yaml`
Traefik Kubernetes Ingress routing HTTP traffic from a single entrypoint:
- `/reading` -> routes to `data-ingest:8080` (IoT sensor submissions)
- `/` -> routes to `frontend:8080` (Web UI dashboard)

#### `templates/networkpolicy.yaml`
Zero-trust Kubernetes `NetworkPolicy` rules operating at the kernel/CNI layer with zero proxy overhead:
1. `default-deny`: Denies all incoming and outgoing namespace traffic by default.
2. `allow-dns`: Allows outbound UDP/TCP port 53 traffic to CoreDNS.
3. `database`: Accepts gRPC port 8090 only from `data-ingest`, `frontend`, and `inference-job`. Has zero outbound egress requirements.
4. `data-ingest`: Accepts HTTP port 8080 from Traefik ingress, egress allowed only to `database:8090`.
5. `frontend`: Accepts HTTP port 8080 from Traefik ingress, egress allowed only to `database:8090`.
6. `inference-server`: Accepts gRPC port 50051 only from `inference-job`, egress allowed on ports 443/80 to download model weights.
7. `inference-job`: Egress allowed only to `database:8090` and `inference-server:50051`.

---

## Deployment Quickstart

### Lint and Template Validation
```bash
# Validate syntax and template rendering
wsl helm lint /mnt/s/dev/weather-station/system/deploy/helm/weather-station
wsl helm template ws /mnt/s/dev/weather-station/system/deploy/helm/weather-station
```

### Dry-Run against Cluster
```bash
wsl helm install ws /mnt/s/dev/weather-station/system/deploy/helm/weather-station \
  --kubeconfig /mnt/c/Users/john/Downloads/admin-k3s-cluster.yaml \
  --namespace weather-station \
  --create-namespace \
  --dry-run
```

### Installation
```bash
wsl helm install ws /mnt/s/dev/weather-station/system/deploy/helm/weather-station \
  --kubeconfig /mnt/c/Users/john/Downloads/admin-k3s-cluster.yaml \
  --namespace weather-station \
  --create-namespace
```

### Upgrades with SSA field-manager conflicts

If `helm upgrade` fails with `conflict with "keel" using apps/v1: ...containers[name=...].image`, Keel's server-side-apply field manager still owns the image field from a previous automated update. Reclaim ownership:
```bash
helm upgrade ws ./weather-station -n weather-station --force-conflicts
```

---

## Automatic Image Updates (Keel)

The chart ships Keel annotations (`keel.enabled: true` in `values.yaml`) so the platform's deployments update automatically:

- `keel.sh/policy: force`, `keel.sh/trigger: poll`, `keel.sh/pollSchedule: @every 15m`
- `keel.sh/match-tag: "true"` - **critical**: switches Keel's `force` policy to the single-tag watcher, which tracks only the **digest of the tag in use** (`latest`). Without it, Keel's multi-tag watcher treats *any* other tag in the repository as an update candidate (the current `latest` tag is non-semver, so no version comparison happens), and stale bare-SHA tags have been observed winning and pinning broken builds.

### Update model: `latest` + digest tracking

- CI (`.github/workflows/docker.yml`) pushes `latest` (rebuilt on every push to `main`), `sha-<shortsha>` tags (human debugging only - never deploy these directly), and semver tags on git tags.
- Deployments always reference `image.tag: latest`. Keel polls the registry every 15 minutes, compares the `latest` digest with what the workloads run, and rolls deployments on digest change.
- To pin an exact build, override the tag via `--set <component>.image.tag=sha-<shortsha>` **and** set `--set keel.enabled=false`, otherwise Keel will roll you back onto `latest` on its next poll.
- The `verify-latest` job in the CI workflow fails the run if `latest` is missing or does not contain every declared platform (`linux/amd64`, `linux/arm64`), preventing Keel from rolling the cluster onto a partially pushed multi-arch image index.

