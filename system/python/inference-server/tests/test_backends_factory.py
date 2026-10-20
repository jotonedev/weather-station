import pytest

pytest.importorskip("chronos")
pytest.importorskip("timestral")

from inference.backends import available_backends, create_backend
from inference.backends.chronos import Chronos2Backend
from inference.backends.timestral import TimestralBackend


def test_available_backends_lists_registered_names() -> None:
    """Test_available_backends_lists_registered_names verifies the registry exposes both backends."""
    assert "chronos-2" in available_backends()
    assert "timestral" in available_backends()


def test_create_backend_instantiates_selected_backend() -> None:
    """Test_create_backend_instantiates_selected_backend verifies the factory returns the right type."""
    chronos_backend = create_backend("chronos-2", "amazon/chronos-2", 3600.0)
    assert isinstance(chronos_backend, Chronos2Backend)
    assert chronos_backend.model_name == "amazon/chronos-2"

    timestral_backend = create_backend("timestral", "mistralai/Timestral-209M", 3600.0)
    assert isinstance(timestral_backend, TimestralBackend)
    assert timestral_backend.model_name == "mistralai/Timestral-209M"
    assert timestral_backend.idle_timeout_seconds == 3600.0


def test_create_backend_unknown_name_raises() -> None:
    """Test_create_backend_unknown_name_raises verifies an unknown backend is rejected with a hint."""
    with pytest.raises(ValueError, match="unknown inference backend"):
        create_backend("does-not-exist", "model", 3600.0)
