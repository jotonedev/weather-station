# Shared libraries

## Project libraries

- `ARQ` - radio link state machine with retries, ACK and duplicate handling
- `Weather` - shared packed structs, fixed-point scales and sentinel values
- `Outbox` - ring buffer for readings awaiting delivery
- `Ingest` - JSON payload builders
- `RadioAdapter` - bridges the ARQ radio interface to the CC1101 driver

## Sensors

## AHT20

- Temperature and humidity sensor
- I2C

## BMP280

- Pressure sensor
- I2C

## Neo 6

- GPS sensor
- UART communication

## CC1101

- Wireless module for 433 MHz transmissions
- SPI communication

## PMS7003

- Particle concentration sensor
- UART communication
- Requires 5v power but uses 3.3v UART

## SGP30

- Air quality sensor
- I2C
