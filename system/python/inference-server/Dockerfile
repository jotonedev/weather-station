# syntax=docker/dockerfile:1
# Build context: system root (run from ./system)
# Usage:
#   docker build -f python/inference-server/Dockerfile -t weather-station/inference-server .

ARG HEALTHPROBE_IMAGE=weather-station/healthprobe:latest
FROM ${HEALTHPROBE_IMAGE} AS healthprobe

FROM ghcr.io/astral-sh/uv:0.13.3 AS uv-bin

FROM python:3.15-slim-bookworm AS build-stage

# The forecast model backend to bundle. Matches a [project.optional-dependencies] entry:
# timestral (default) or chronos-2.
ARG INFERENCE_BACKEND=timestral

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
    uv sync --frozen --no-dev --no-install-project --extra ${INFERENCE_BACKEND}

# Copy application source code
COPY python/inference-server/inference/ ./inference/
COPY python/inference-server/README.md ./

# Install project into virtual environment
RUN --mount=type=cache,target=/root/.cache/uv \
    uv sync --frozen --no-dev --extra ${INFERENCE_BACKEND}

FROM python:3.15-slim-bookworm AS release-stage

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
COPY --from=healthprobe /grpc-health-probe /usr/local/bin/grpc-health-probe

USER 10001:10001

EXPOSE 50051

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD [ "/usr/local/bin/grpc-health-probe", "-addr=127.0.0.1:50051", "-service=ready" ]

ENTRYPOINT ["python", "-m", "inference"]
