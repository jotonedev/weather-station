# Agents rules

## Rules for writing docs

- Follow the same writing style
- Do not use em dashes
- Try to not use colon and semicolon
- Do not chain adjectives
- Do not repeat information said in other files
- Do not use "It's not X, it's Y"
- Be clear and concise

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
