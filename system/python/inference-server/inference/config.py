import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Config:
    """Config holds runtime settings for the inference server."""

    host: str = "0.0.0.0"
    port: int = 50051
    model_backend: str = "timestral"
    model_name: str = "mistralai/Timestral-209M"
    idle_timeout_seconds: float = 3600.0
    max_workers: int = 4
    max_horizon_hours: int = 240

    @classmethod
    def from_env(cls) -> "Config":
        """from_env creates configuration using environment variables."""
        port_val = int(os.getenv("INFERENCE_PORT", "50051"))
        host_val = os.getenv("INFERENCE_HOST", "0.0.0.0")
        backend_val = os.getenv("INFERENCE_MODEL_BACKEND", "timestral")
        model_val = os.getenv("INFERENCE_MODEL_NAME", "mistralai/Timestral-209M")
        idle_val = float(os.getenv("INFERENCE_IDLE_TIMEOUT_SECONDS", "3600.0"))
        workers_val = int(os.getenv("INFERENCE_MAX_WORKERS", "4"))
        max_horizon_val = int(os.getenv("INFERENCE_MAX_HORIZON_HOURS", "240"))

        return cls(
            host=host_val,
            port=port_val,
            model_backend=backend_val,
            model_name=model_val,
            idle_timeout_seconds=idle_val,
            max_workers=workers_val,
            max_horizon_hours=max_horizon_val,
        )
