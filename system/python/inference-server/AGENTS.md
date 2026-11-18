# Python agents rules

This file extends the root `AGENTS.md` with rules specific to `system/python/inference-server/`.

## Testing

- Write tests with plain `pytest` functions, not test classes, and put shared fixtures in `tests/conftest.py`
- Use `pytest.mark.parametrize` for input variations of one behavior instead of copying test functions
- Use `pytest.raises` with `match` to assert the specific error and message
- Use `monkeypatch` for environment and attributes, and avoid patching private functions of the code under test
- Do not load real models in unit tests, use the backend interface with a stub
- Check numeric outputs with `pytest.approx` and explicit tolerances

## Typing and linting

- Annotate public functions so `uv run ty check` passes without new ignores
- Do not add `# noqa` or `# type: ignore` without a short reason
