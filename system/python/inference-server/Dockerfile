# syntax=docker/dockerfile:1
# Build context: system root (run from ./system)
# Usage:
#   docker build -f python/inference-server/Dockerfile -t weather-station/inference-server .

ARG GO_VERSION=1.27
FROM golang:${GO_VERSION}-alpine AS probe-builder

ENV GOWORK=off

WORKDIR /workspace/go/common
COPY go/common/go.mod go/common/go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY go/common/ ./

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /grpc-health-probe \
    ./cmd/healthprobe

FROM ghcr.io/astral-sh/uv:0.6.5 AS uv-bin

FROM python:3.14-slim-bookworm AS build-stage

WORKDIR /workspace

COPY --from=uv-bin /uv /uvx /bin/

ENV UV_COMPILE_BYTECODE=1 \
    UV_LINK_MODE=copy \
    UV_PYTHON_DOWNLOADS=never \
    PYTHONUNBUFFERED=1

# Copy package metadata first to optimize Docker layer caching
COPY proto/gen/python/pyproject.toml proto/gen/python/README.md ./proto/gen/python/
COPY proto/gen/python/weather/ ./proto/gen/python/weather/
COPY python/inference-server/pyproject.toml python/inference-server/uv.lock python/inference-server/.python-version ./python/inference-server/

# Install dependencies into virtual environment
WORKDIR /workspace/python/inference-server
RUN --mount=type=cache,target=/root/.cache/uv \
    uv sync --frozen --no-dev --no-install-project

# Copy application source code
COPY python/inference-server/inference/ ./inference/
COPY python/inference-server/README.md ./

# Install project into virtual environment
RUN --mount=type=cache,target=/root/.cache/uv \
    uv sync --frozen --no-dev

FROM python:3.14-slim-bookworm AS release-stage

# Install runtime ca-certificates
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace/python/inference-server

# Create unprivileged runtime user and cache directories
RUN useradd -u 10001 -m appuser \
    && mkdir -p /home/appuser/.cache \
    && chown -R appuser:appuser /home/appuser

ENV PYTHONUNBUFFERED=1 \
    PATH="/workspace/python/inference-server/.venv/bin:$PATH" \
    HF_HOME="/home/appuser/.cache/huggingface"

# Copy virtualenv and required code from build stage
COPY --from=build-stage --chown=appuser:appuser /workspace/proto/gen/python /workspace/proto/gen/python
COPY --from=build-stage --chown=appuser:appuser /workspace/python/inference-server /workspace/python/inference-server
COPY --from=probe-builder /grpc-health-probe /usr/local/bin/grpc-health-probe

USER 10001:10001

EXPOSE 50051

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD [ "/usr/local/bin/grpc-health-probe", "-addr=127.0.0.1:50051", "-service=ready" ]

ENTRYPOINT ["python", "-m", "inference"]
