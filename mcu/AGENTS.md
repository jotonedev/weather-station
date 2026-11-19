# MCU agents rules

This file extends the root `AGENTS.md` with rules specific to `mcu/`.

## Build and test boundaries

- Verify changes in `mcu/` with `pio test -e native` and `pio run -e control-node -e base-station`
- All three PlatformIO environments compile with `-std=gnu++23`
- Never upload firmware (`-t upload`) or open a serial monitor unless the user explicitly asks

## Formatting and linting

- Format modified `.cpp` and `.h` files using `mcu/.clang-format`, excluding `mcu/lib/CC1101/`
- Check formatting from `mcu/` with this command

  ```sh
  find src lib test -type f \( -name '*.cpp' -o -name '*.h' \) ! -path 'lib/CC1101/*' \
    -exec clang-format-18 --dry-run --Werror {} +
  ```

- Run `clang-tidy` from `mcu/` against `mcu/.clang-tidy` with this command

  ```sh
  clang-tidy --config-file=.clang-tidy \
    lib/ARQ/src/ARQ.cpp lib/Weather/src/Weather.cpp lib/Ingest/src/Ingest.cpp \
    lib/Outbox/src/Outbox.cpp \
    test/test_arq.cpp test/test_weather.cpp test/test_outbox.cpp test/test_ingest.cpp test/main.cpp \
    -- -std=gnu++17 \
    -Ilib/ARQ/src -Ilib/Weather/src -Ilib/Ingest/src -Ilib/Outbox/src \
    -I.pio/libdeps/native/googletest/googletest/include \
    -I.pio/libdeps/native/googletest/googlemock/include
  ```

- If `clang-format` or `clang-tidy` is not installed locally, follow `mcu/.clang-format` manually and report that the check did not run

## Library portability and protocol

- Keep `lib/ARQ`, `lib/Weather`, `lib/Ingest`, and `lib/Outbox` free of Arduino and ESP32 headers so `[env:native]` tests compile without hardware
- `lib/CC1101` and `lib/RadioAdapter` are ignored under `[env:native]`
- Changes to `WeatherData` or packet encoding in `lib/Weather/` affect both `control-node` and `base-station`

## Testing

- Native tests use GoogleTest and GoogleMock under `test/`, one `test_<lib>.cpp` per library
- Prioritize packet encoding round trips, ARQ retries, timeouts and sequence wraparound, and Outbox and Ingest boundary cases
- Inject time and radio behavior through the library interfaces and do not use real sleeps or threads for timing
- Do not test constants or trivial accessors

## Hardware, concurrency, and secrets

- Do not assign ESP32-S3 reserved pins listed in `src/control-node/constants.h` (GPIO 26-37 for Flash and PSRAM, GPIO 19-20 for USB, strapping pins 0, 3, 45, 46, and UART0 pins 43-44)
- Guard every access to shared `Wire` between `AnemometerTask` (Core 0) and `loop()` (Core 1) with `wireMutex`
- Keep FreeRTOS tasks and `loop()` non-blocking so task heartbeats reset the watchdog within `WATCHDOG_TIMEOUT_MS`
- Never commit real WiFi or ingest credentials in `src/base-station/config.h` and do not change `CALIBRATION_MODE` unintentionally
