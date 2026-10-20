# Inference Server

gRPC server for probabilistic weather forecasting.

## Model backends

It supports the following model familiesç

| Backend name | Model family | Optional dependency |
| --- | --- | --- |
| `timestral` | `mistralai/Timestral-209M` (default) | `timestral` |
| `chronos-2` | `amazon/chronos-2` | `chronos-2` |

Select the active backend with the environment variables `INFERENCE_MODEL_BACKEND` and the checkpoint
with `INFERENCE_MODEL_NAME` (defaults `timestral` and
`mistralai/Timestral-209M`).

## Dependencies

Core dependencies (gRPC, protobuf, torch, numpy, pandas) are always
installed. Model packages are declared as extras.

```bash
uv sync --extra timestral            # timestral backend only
uv sync --extra chronos-2            # chronos-2 backend only
uv sync --extra timestral --extra chronos-2   # both
```
