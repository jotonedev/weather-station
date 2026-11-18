import signal

import pytest

from inference.__main__ import build_config, main
from inference.config import Config

ENV_VARS = (
    "INFERENCE_PORT",
    "INFERENCE_HOST",
    "INFERENCE_MODEL_BACKEND",
    "INFERENCE_MODEL_NAME",
    "INFERENCE_IDLE_TIMEOUT_SECONDS",
    "INFERENCE_MAX_WORKERS",
    "INFERENCE_MAX_HORIZON_HOURS",
    "INFERENCE_TIMESTRAL_SYMMETRIC_AVERAGING",
)


def test_config_defaults(monkeypatch: pytest.MonkeyPatch) -> None:
    """Test_config_defaults verifies default Config values when env vars are unset."""
    for env_var in ENV_VARS:
        monkeypatch.delenv(env_var, raising=False)

    config = Config.from_env()
    assert config == Config(
        host="0.0.0.0",
        port=50051,
        model_backend="timestral",
        model_name="mistralai/Timestral-209M",
        idle_timeout_seconds=3600.0,
        max_workers=4,
        max_horizon_hours=240,
        timestral_symmetric_averaging=False,
    )


def test_config_from_env(monkeypatch: pytest.MonkeyPatch) -> None:
    """Test_config_from_env verifies parsing of all eight environment variables."""
    monkeypatch.setenv("INFERENCE_PORT", "60060")
    monkeypatch.setenv("INFERENCE_HOST", "127.0.0.1")
    monkeypatch.setenv("INFERENCE_MODEL_BACKEND", "chronos-2")
    monkeypatch.setenv("INFERENCE_MODEL_NAME", "custom-model")
    monkeypatch.setenv("INFERENCE_IDLE_TIMEOUT_SECONDS", "1800")
    monkeypatch.setenv("INFERENCE_MAX_WORKERS", "8")
    monkeypatch.setenv("INFERENCE_MAX_HORIZON_HOURS", "360")
    monkeypatch.setenv("INFERENCE_TIMESTRAL_SYMMETRIC_AVERAGING", "true")

    config = Config.from_env()
    assert config.port == 60060
    assert config.host == "127.0.0.1"
    assert config.model_backend == "chronos-2"
    assert config.model_name == "custom-model"
    assert config.idle_timeout_seconds == 1800.0
    assert config.max_workers == 8
    assert config.max_horizon_hours == 360
    assert config.timestral_symmetric_averaging is True


@pytest.mark.parametrize(
    ("raw_value", "expected"),
    [
        ("1", True),
        ("true", True),
        ("TRUE", True),
        (" Yes ", True),
        ("on", True),
        ("0", False),
        ("false", False),
        ("no", False),
        ("off", False),
        ("", False),
        ("invalid", False),
    ],
)
def test_config_from_env_symmetric_averaging_values(
    monkeypatch: pytest.MonkeyPatch,
    raw_value: str,
    expected: bool,
) -> None:
    """Test_config_from_env_symmetric_averaging_values checks boolean parsing of symmetric averaging env var."""
    monkeypatch.setenv("INFERENCE_TIMESTRAL_SYMMETRIC_AVERAGING", raw_value)
    assert Config.from_env().timestral_symmetric_averaging is expected


@pytest.mark.parametrize(
    ("env_var", "raw_value"),
    [
        ("INFERENCE_PORT", "invalid"),
        ("INFERENCE_IDLE_TIMEOUT_SECONDS", "invalid"),
        ("INFERENCE_MAX_WORKERS", "invalid"),
        ("INFERENCE_MAX_HORIZON_HOURS", "invalid"),
    ],
)
def test_config_from_env_invalid_numeric_raises(
    monkeypatch: pytest.MonkeyPatch,
    env_var: str,
    raw_value: str,
) -> None:
    """Test_config_from_env_invalid_numeric_raises verifies non-numeric env values raise ValueError."""
    monkeypatch.setenv(env_var, raw_value)
    with pytest.raises(ValueError):
        Config.from_env()


def test_cli_build_config_preserves_env_and_zero_overrides(monkeypatch: pytest.MonkeyPatch) -> None:
    """Test_cli_build_config_preserves_env_and_zero_overrides preserves env fields and zero CLI overrides."""
    monkeypatch.setenv("INFERENCE_PORT", "50055")
    monkeypatch.setenv("INFERENCE_IDLE_TIMEOUT_SECONDS", "1200")
    monkeypatch.setenv("INFERENCE_MAX_WORKERS", "8")
    monkeypatch.setenv("INFERENCE_MAX_HORIZON_HOURS", "360")
    monkeypatch.setenv("INFERENCE_TIMESTRAL_SYMMETRIC_AVERAGING", "true")

    env_only = build_config([])
    assert env_only.port == 50055
    assert env_only.idle_timeout_seconds == 1200.0
    assert env_only.max_workers == 8
    assert env_only.max_horizon_hours == 360
    assert env_only.timestral_symmetric_averaging is True

    zero_override = build_config(["--port", "0", "--idle-timeout", "0"])
    assert zero_override.port == 0
    assert zero_override.idle_timeout_seconds == 0.0
    assert zero_override.timestral_symmetric_averaging is True


def test_cli_build_config_overrides_all_flags(monkeypatch: pytest.MonkeyPatch) -> None:
    """Test_cli_build_config_overrides_all_flags verifies CLI flags override environment defaults."""
    for env_var in ENV_VARS:
        monkeypatch.delenv(env_var, raising=False)

    config = build_config(
        [
            "--host",
            "127.0.0.1",
            "--port",
            "6000",
            "--model-backend",
            "chronos-2",
            "--model-name",
            "custom/model",
            "--idle-timeout",
            "45.5",
        ]
    )
    assert config.host == "127.0.0.1"
    assert config.port == 6000
    assert config.model_backend == "chronos-2"
    assert config.model_name == "custom/model"
    assert config.idle_timeout_seconds == 45.5


@pytest.mark.parametrize(
    "argv",
    [
        ["--port", "abc"],
        ["--idle-timeout", "xyz"],
        ["--unknown"],
    ],
)
def test_cli_invalid_argument_exits(argv: list[str]) -> None:
    """Test_cli_invalid_argument_exits checks argparse exits with code 2 on invalid CLI arguments."""
    with pytest.raises(SystemExit) as exc_info:
        build_config(argv)
    assert exc_info.value.code == 2


def test_main_starts_server_and_stops_on_interrupt(monkeypatch: pytest.MonkeyPatch) -> None:
    """Test_main_starts_server_and_stops_on_interrupt verifies main lifecycle and graceful shutdown."""
    state: dict[str, object] = {}

    class FakeServer:
        def __init__(self, config: Config):
            state["config"] = config
            state["started"] = False
            state["stopped_grace"] = None

        def start(self) -> None:
            state["started"] = True

        def wait_for_termination(self) -> None:
            raise KeyboardInterrupt

        def stop(self, grace: float = 5.0) -> None:
            state["stopped_grace"] = grace

    registered_signals: list[int] = []
    monkeypatch.setattr("inference.__main__.InferenceServer", FakeServer)
    monkeypatch.setattr(
        "inference.__main__.signal.signal",
        lambda sig_num, _handler: registered_signals.append(int(sig_num)),
    )

    with pytest.raises(SystemExit) as exc_info:
        main(["--port", "0"])

    assert exc_info.value.code == 0
    assert state["started"] is True
    assert state["stopped_grace"] == 3.0
    assert isinstance(state["config"], Config)
    assert state["config"].port == 0
    assert signal.SIGINT in registered_signals
    assert signal.SIGTERM in registered_signals
