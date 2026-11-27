# Shared libraries

## Project libraries

- `ARQ` handles the radio link state machine with retries, ACKs, and duplicate filtering
- `Weather` defines shared packed structs, fixed-point scales, and sentinel values
- `Outbox` provides a ring buffer for readings awaiting delivery
- `Ingest` formats JSON payload batches for the ingest server
- `RadioAdapter` bridges the ARQ radio interface to the CC1101 driver
- `Sync` provides header-only RAII wrappers for FreeRTOS spinlocks and mutexes
- `AS5600` provides the shared magnetic encoder driver and magnet diagnostics
- `Anemometer` computes wind speed and gusts from AS5600 encoder deltas
- `WindVane` filters wind direction from AS5600 magnetic encoder angles
- `RainMeter` tracks tipping-bucket motion and rainfall accumulation via AS5600
- `Aht20` wraps the AHT20 temperature and humidity sensor driver
- `Bmp180` wraps the BMP180 barometric pressure sensor driver
- `Pms7003` wraps the PMS7003 particulate matter sensor driver
- `CC1101` provides the 433 MHz SPI radio transceiver driver
