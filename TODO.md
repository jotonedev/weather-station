# TODO

## Backend and protobuf health telemetry

- Add optional `health_flags`, `reset_reason`, `boot_count`, `ack_rssi`, `ack_lqi`, `rx_rssi`, `rx_lqi`, and `buffer_dropped` fields in `system/proto/weather/database/v1/reading.proto`
- Regenerate protobuf bindings under `system/proto/gen/` after updating `system/proto/weather/database/v1/reading.proto`
- Serialize `SystemHealthData` from `mcu/lib/Weather/src/Weather.h` alongside base-station `RSSI`, `LQI`, and `bufferDropped` in `mcu/lib/Ingest/src/Ingest.cpp`
- Update `system/go/data-ingest/controller/weatherstation/model.go` and `system/go/database/` to parse, validate, and store the new health and radio fields
- Surface AS5600 fault codes (`0x00`, `0x28`, `0x30`, `0x80`) and `3.3V` `AGC` bounds (`0` to `128`) in `system/go/frontend/`

## MCU hardware assembly follow-ups

- Verify `esp_pm` frequency scaling and light sleep on `control-node` once hardware assembly finishes
- Switch `CALIBRATION_MODE` to `0` in `mcu/src/control-node/constants.h` once physical sensor calibration finishes
- Pin a CA certificate in `mcu/src/base-station/` instead of calling `setInsecure()`
