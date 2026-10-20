import argparse
import logging
import signal
import sys

from inference.config import Config
from inference.server import InferenceServer

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("inference")


def main() -> None:
    """Main parses arguments and runs the inference server until interrupted."""
    parser = argparse.ArgumentParser(description="Weather Inference Server")
    parser.add_argument("--host", default=None, help="Host address to bind")
    parser.add_argument("--port", type=int, default=None, help="Port to listen on")
    parser.add_argument("--model-backend", default=None, help="Forecast backend name (chronos-2, timestral)")
    parser.add_argument("--model-name", default=None, help="HuggingFace model repository or checkpoint path")
    parser.add_argument("--idle-timeout", type=float, default=None, help="Idle timeout seconds before model eviction")

    args = parser.parse_args()

    default_config = Config.from_env()
    config = Config(
        host=args.host or default_config.host,
        port=args.port or default_config.port,
        model_backend=args.model_backend or default_config.model_backend,
        model_name=args.model_name or default_config.model_name,
        idle_timeout_seconds=args.idle_timeout or default_config.idle_timeout_seconds,
        max_workers=default_config.max_workers,
        max_horizon_hours=default_config.max_horizon_hours,
    )

    server = InferenceServer(config)
    server.start()

    def handle_signal(_sig_num, _frame) -> None:
        logger.info("shutdown signal received")
        server.stop(grace=3.0)
        sys.exit(0)

    signal.signal(signal.SIGINT, handle_signal)
    signal.signal(signal.SIGTERM, handle_signal)

    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        handle_signal(signal.SIGINT, None)


if __name__ == "__main__":
    main()
