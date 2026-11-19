# Agents rules

## Rules for writing docs

- Use the same heading levels and bullet markers as the surrounding sections
- Do not use em dashes
- Do not use colons or semicolons in prose, code blocks and file paths are exempt
- Use at most two adjectives before a noun
- State each rule once across all AGENTS.md files and point to the owning file instead of restating it
- Do not use "It's not X, it's Y" constructions
- Keep each bullet to one sentence under 25 words

## Code quality

- Keep solutions simple and direct, with no speculative abstractions or unused code
- Follow the existing style and naming of the surrounding package
- Fix root causes and do not silence linters without a short comment explaining why

## Testing

- Test behavior that matters such as core logic, data and protocol contracts, error handling, and concurrency or boundary conditions
- Do not test trivial code such as getters, constants, struct literals, plain wiring, generated code, or third-party libraries
- Assert on observable results, not internals, so a behavior-preserving refactor does not break tests
- Cover each behavior once and delete duplicate tests instead of adding near-identical cases
- Add a test that fails without the fix for every bug fix
- Keep tests deterministic by injecting clocks, randomness, and I/O and by avoiding sleeps
- Prefer real values over mocks and mock only at process or network boundaries
- Never weaken or remove an assertion on valid behavior to make a test pass

## Go services

- The Go workspace lives in `system/` with `system/go.work`
- Do not leave unintended edits in `system/go.work.sum`
- `system/go/database` requires `CGO_ENABLED=1` for DuckDB
- On this Windows host, run `database` builds and tests in WSL using `/snap/bin/go` since Windows lacks `gcc`
- Verify changes with `gofmt`, `go vet ./...`, and `go test -race ./...`

## Python inference server

- Located in `system/python/inference-server` and managed with `uv`
- Verify changes with `uv run ruff check .`, `uv run ruff format --check .`, `uv run ty check`, and `uv run pytest`

## MCU firmware

- Located in `mcu/` and built with PlatformIO
- Follow `mcu/.clang-format` for all C++ changes (4-space indent, 100-column limit, braces on every `if` and loop, no single-line `if`)
- Do not reformat `mcu/lib/CC1101/`
- Run native unit tests with `pio test -e native` in `mcu/`

## Protobuf contracts

- Source `.proto` files live in `system/proto/`
- Do not hand-edit generated files under `system/proto/gen/`

## Git commits

- Commit one subsystem at a time with `mcu/`, `system/go/`, and `system/python/` in separate commits
- Include only files that belong to the stated change and leave unrelated work unstaged
- Write subjects in imperative mood under 72 characters and describe the outcome rather than the steps
- Prefix subjects with `[<system>]`, using `[go,<service_name>]` for Go services and names such as `python`, `mcu`, or `workflow` elsewhere
- Add a body only when the reason for the change is not clear from the subject
- Run the relevant tests and linters before committing
- Do not amend, rebase, or force push unless the user asks
