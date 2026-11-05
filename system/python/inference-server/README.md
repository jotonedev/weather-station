# Inference Server

gRPC server for weather forecasting.

## Model backends

It supports the following model families.

| Backend name | Model family | Optional dependency |
| --- | --- | --- |
| `timestral` | `mistralai/Timestral-209M` (default) | `timestral` |
| `chronos-2` | `amazon/chronos-2` | `chronos-2` |

Select the active backend with the environment variables `INFERENCE_MODEL_BACKEND` and the checkpoint
with `INFERENCE_MODEL_NAME` (defaults `timestral` and
`mistralai/Timestral-209M`).

## Targets and covariates

Hourly forecasts predict temperature, humidity, rainfall, and pressure. Daily forecasts predict minimum and maximum temperature, minimum and maximum humidity, rainfall, and pressure.

Non-target observations and derived signals (wind components, gusts, rain rate, dew point, dew point depression, absolute humidity, pressure tendencies, and rolling rain totals) enter the model as past covariates. Solar geometry and seasonal signals (solar elevation, clear-sky irradiance, solar time, day of year, and daylight duration) enter as known covariates over the context and horizon.

Short interior gaps up to two steps use linear interpolation while longer outages remain `NaN` for the model mask. Rainfall expectations and rain probabilities are integrated across all checkpoint quantile knots.

## Dependencies

Core dependencies (gRPC, protobuf, torch, numpy, pandas) are always
installed. Model packages are declared as extras.

```bash
uv sync --extra timestral            # timestral backend only
uv sync --extra chronos-2            # chronos-2 backend only
uv sync --extra timestral --extra chronos-2   # both
```
