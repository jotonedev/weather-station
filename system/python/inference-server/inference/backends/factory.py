import importlib

from inference.backends.base import ForecastBackend

# Registry mapping backend names to (module, class). Modules are imported lazily at
# lookup time so an optional dependency for an unused backend is never loaded.
_BACKEND_REGISTRY: dict[str, tuple[str, str]] = {
    "chronos-2": ("inference.backends.chronos", "Chronos2Backend"),
    "timestral": ("inference.backends.timestral", "TimestralBackend"),
}


def available_backends() -> list[str]:
    """available_backends lists the registered backend names."""
    return sorted(_BACKEND_REGISTRY)


def create_backend(backend_name: str, model_name: str, idle_timeout_seconds: float) -> ForecastBackend:
    """create_backend instantiates the requested forecast backend by name.

    New architectures are added by subclassing ForecastBackend and registering one
    (module, class) entry in _BACKEND_REGISTRY.
    """
    try:
        module_path, class_name = _BACKEND_REGISTRY[backend_name]
    except KeyError:
        known = ", ".join(available_backends())
        raise ValueError(f"unknown inference backend {backend_name!r}; known backends are {known}") from None

    module = importlib.import_module(module_path)
    backend_cls = getattr(module, class_name)
    return backend_cls(model_name=model_name, idle_timeout_seconds=idle_timeout_seconds)
