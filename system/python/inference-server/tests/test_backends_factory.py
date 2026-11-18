from types import SimpleNamespace

import pytest

from inference.backends import available_backends, create_backend


def test_available_backends_lists_registered_names() -> None:
    """Test_available_backends_lists_registered_names verifies the registry exposes both backends."""
    assert "chronos-2" in available_backends()
    assert "timestral" in available_backends()


def test_create_backend_unknown_name_raises() -> None:
    """Test_create_backend_unknown_name_raises verifies an unknown backend is rejected with a hint."""
    with pytest.raises(ValueError, match="unknown inference backend") as exc_info:
        create_backend("does-not-exist", "model", 3600.0)

    message = str(exc_info.value)
    assert "chronos-2" in message
    assert "timestral" in message


def test_create_backend_forwards_kwargs_per_backend(monkeypatch: pytest.MonkeyPatch) -> None:
    """Test_create_backend_forwards_kwargs_per_backend forwards timestral-only symmetric averaging flag."""
    calls: list[tuple[str, dict[str, object]]] = []

    class FakeChronosBackend:
        def __init__(self, **kwargs: object):
            calls.append(("chronos-2", kwargs))

    class FakeTimestralBackend:
        def __init__(self, **kwargs: object):
            calls.append(("timestral", kwargs))

    modules = {
        "inference.backends.chronos": SimpleNamespace(Chronos2Backend=FakeChronosBackend),
        "inference.backends.timestral": SimpleNamespace(TimestralBackend=FakeTimestralBackend),
    }
    monkeypatch.setattr("inference.backends.factory.importlib.import_module", lambda path: modules[path])

    create_backend("chronos-2", "amazon/chronos-2", 1200.0, timestral_symmetric_averaging=True)
    create_backend("timestral", "mistralai/Timestral-209M", 1800.0, timestral_symmetric_averaging=True)

    assert calls == [
        ("chronos-2", {"model_name": "amazon/chronos-2", "idle_timeout_seconds": 1200.0}),
        (
            "timestral",
            {
                "model_name": "mistralai/Timestral-209M",
                "idle_timeout_seconds": 1800.0,
                "symmetric_averaging": True,
            },
        ),
    ]


def test_create_backend_instantiates_chronos_backend() -> None:
    """Test_create_backend_instantiates_chronos_backend verifies Chronos-2 backend instantiation."""
    pytest.importorskip("chronos")
    from inference.backends.chronos import Chronos2Backend

    backend = create_backend("chronos-2", "amazon/chronos-2", 3600.0)
    assert isinstance(backend, Chronos2Backend)
    assert backend.model_name == "amazon/chronos-2"
    assert backend.idle_timeout_seconds == 3600.0
    assert not backend.is_loaded()


def test_create_backend_instantiates_timestral_backend() -> None:
    """Test_create_backend_instantiates_timestral_backend verifies Timestral backend instantiation."""
    pytest.importorskip("timestral")
    from inference.backends.timestral import TimestralBackend

    backend = create_backend(
        "timestral",
        "mistralai/Timestral-209M",
        1800.0,
        timestral_symmetric_averaging=True,
    )
    assert isinstance(backend, TimestralBackend)
    assert backend.model_name == "mistralai/Timestral-209M"
    assert backend.idle_timeout_seconds == 1800.0
    assert backend.symmetric_averaging is True
    assert not backend.is_loaded()
