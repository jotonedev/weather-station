# Inference Server

gRPC server for probabilistic weather forecasting. It serves the
`weather.inference.v1.InferenceService` contract used by `inference-job`.

## Model backends

Forecasting is pluggable. Each model family is a backend behind a shared
abstraction, so new model architectures can be added without touching the
server, validation, features, or formatter.

| Backend name | Model family | Optional dependency |
| --- | --- | --- |
| `timestral` | `mistralai/Timestral-209M` (default) | `timestral` |
| `chronos-2` | `amazon/chronos-2` | `chronos-2` |

Select the active backend with `INFERENCE_MODEL_BACKEND` and the checkpoint
with `INFERENCE_MODEL_NAME` (defaults `timestral` and
`mistralai/Timestral-209M`). Model dependencies are optional so an image only
pulls the packages for the backend it runs.

## Dependencies

Core dependencies (gRPC, protobuf, torch, numpy, pandas) are always
installed. Model packages are declared as extras.

```bash
uv sync --extra timestral            # timestral backend only
uv sync --extra chronos-2            # chronos-2 backend only
uv sync --extra timestral --extra chronos-2   # both
```

## Docker

The `INFERENCE_BACKEND` build arg selects which optional dependency the image
bundles. It must match `INFERENCE_MODEL_BACKEND` at runtime.

```bash
docker build -f python/inference-server/Dockerfile -t weather-station/inference-server .          # timestral (default)
docker build --build-arg INFERENCE_BACKEND=chronos-2 -f python/inference-server/Dockerfile \
  -t weather-station/inference-server .                                                             # chronos-2
```

## Adding a backend

1. Subclass `inference.backends.base.ForecastBackend`.
2. Implement `_load()` to build the model and `_predict_loaded()` to run one
   forecast. `_predict_loaded` must return quantiles shaped
   `(variates, horizon, 3)` for levels 0.1, 0.5, 0.9 and a mean shaped
   `(variates, horizon)`. Lazy loading, CPU execution, and idle eviction are
   inherited from the base class.
3. Add a `(module, class)` entry to `_BACKEND_REGISTRY` in
   `inference.backends.factory.py`.
4. Add an optional dependency group in `pyproject.toml` and pass
   `--extra <name>` through the Docker `INFERENCE_BACKEND` build arg.
