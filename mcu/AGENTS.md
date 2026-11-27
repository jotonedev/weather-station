# MCU agents rules

This file extends the root `AGENTS.md` with rules specific to `mcu/`.

## Build and test boundaries

 - Verify changes in `mcu/` with `pio test -e native` and `pio run -e control-node -e base-station -e anemometer-test -e windvane-test`
- Verify `control-node` also compiles with `PLATFORMIO_BUILD_FLAGS='-DCALIBRATION_MODE=0'` when editing production sampling or radio code
- All PlatformIO environments compile with `-std=gnu++23`
- Never upload firmware (`-t upload`) or open a serial monitor unless the user explicitly asks

## Formatting and linting

- Format modified `.cpp` and `.h` files using `mcu/.clang-format`, excluding vendored upstream files
- Check formatting from `mcu/` with this command

  ```sh
  find src lib test -type f \( -name '*.cpp' -o -name '*.h' \) \
    ! -path 'lib/CC1101/*' ! -path 'lib/AS5600/src/AS5600.*' \
    ! -path 'lib/Aht20/src/Adafruit_AHTX0.*' ! -path 'lib/Bmp180/src/Adafruit_BMP085.*' \
    ! -path 'lib/Pms7003/src/PMS.*' \
    -exec clang-format --dry-run --Werror {} +
  ```

- Run `clang-tidy` from `mcu/` against `mcu/.clang-tidy` with this command

  ```sh
  clang-tidy --config-file=.clang-tidy \
    lib/ARQ/src/ARQ.cpp lib/ARQ/src/TxLink.cpp lib/Weather/src/Weather.cpp \
    lib/Ingest/src/Ingest.cpp lib/Outbox/src/Outbox.cpp \
    lib/AS5600/src/MagnetDiagnostics.cpp lib/Anemometer/src/AnemometerState.cpp \
    lib/WindVane/src/WindVaneState.cpp lib/RainMeter/src/RainMeterLogic.cpp \
    lib/Pms7003/src/PmsDutyCycle.cpp \
    test/test_arq.cpp test/test_weather.cpp test/test_outbox.cpp test/test_ingest.cpp \
    test/test_as5600.cpp test/test_anemometer.cpp test/test_wind_vane.cpp \
    test/test_rain_meter.cpp test/test_aht20.cpp test/test_bmp180.cpp \
    test/test_pms7003.cpp test/main.cpp \
    -- -std=gnu++17 \
    -Ilib/ARQ/src -Ilib/Weather/src -Ilib/Ingest/src -Ilib/Outbox/src \
    -Ilib/AS5600/src -Ilib/Anemometer/src -Ilib/WindVane/src -Ilib/RainMeter/src \
    -Ilib/Aht20/src -Ilib/Bmp180/src -Ilib/Pms7003/src \
    -I.pio/libdeps/native/googletest/googletest/include \
    -I.pio/libdeps/native/googletest/googlemock/include
  ```

- If `clang-format` or `clang-tidy` is not installed locally, follow `mcu/.clang-format` manually and report that the check did not run

## Library portability and protocol

- Organize sensor code as dedicated per-instrument libraries under `lib/` (sharing `lib/AS5600/` across rotary instruments) rather than catch-all modules
- Vendor third-party libraries needing bug fixes into `lib/<Name>/` with their upstream `LICENSE` file and apply surgical patches in place
- Keep node `main.cpp` files under roughly 200 lines by extracting focused support files with clear domain names
- Keep pure library logic free of Arduino headers and wrap hardware code in `#ifdef ARDUINO` for `[env:native]`
- Keep FreeRTOS guards in `lib/Sync/src/Sync.h` header-only and unused by native libraries or tests
- `lib/CC1101` and `lib/RadioAdapter` are ignored under `[env:native]`
- Changes to `WeatherData` or packet encoding in `lib/Weather/` affect both `control-node` and `base-station`
- Keep CC1101 FEC, whitening, CRC, and fixed-length frames enabled to preserve link reliability over noisy 433 MHz channels

## Testing

- Keep native tests strictly to one `test_<lib>.cpp` unit suite per library under `test/` and do not add cross-library integration suites
- Prioritize packet encoding round trips, ARQ retries, timeouts and sequence wraparound, and Outbox and Ingest boundary cases
- Inject time and radio behavior through the library interfaces and do not use real sleeps or threads for timing
- Do not test constants or trivial accessors

## Hardware, concurrency, and secrets

- Do not assign ESP32-S3 reserved pins listed in `src/control-node/constants.h` (GPIO 26-37 for Flash and PSRAM, GPIO 19-20 for USB, strapping pins 0, 3, 45, 46, and UART0 pins 43-44)
- Power all AS5600 sensors from `3.3V` (never `5V`) with `DIR` tied to `GND` so I2C stays ESP32-S3 safe
- Guard every access to shared `Wire` between `AnemometerTask` (Core 0) and `loop()` (Core 1) with `wireMutex`
- Release `wireMutex` before any sensor conversion wait so AHT20 and BMP180 delays never starve the 10 ms anemometer task
- Guard FreeRTOS critical sections and mutexes with `CriticalSection` and `MutexGuard` from `lib/Sync/src/Sync.h` instead of raw lock calls
- Never call `Preferences`, I2C, mutexes, `delay`, or `Serial` inside `CriticalSection`, and keep instrument state logic pure
- Keep FreeRTOS tasks and `loop()` non-blocking so task heartbeats reset the watchdog within `WATCHDOG_TIMEOUT_MS`
- Keep `I2C_CLOCK_HZ` at 100 kHz in `src/control-node/constants.h` to preserve signal integrity over 1 m sensor cables
- Keep active `AnemometerTask` polling at 10 ms to handle magnet wobble, and slow polling only during sustained standstill
- Keep `loop()` idle pacing on Core 1 so Core 0 sensor tasks run unaffected
- Do not enable WiFi modem sleep or WiFi duty-cycling on `base-station`
- Do not rely on deep sleep or bare-module power changes on `control-node`
- Defer `esp_pm` frequency and light-sleep verification on `control-node` until hardware assembly completes
- Never commit real WiFi or ingest credentials in `src/base-station/config.h` and do not change `CALIBRATION_MODE` unintentionally
