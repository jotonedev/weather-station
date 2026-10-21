# MCU Firmware

Two ESP32-S3 boards. The control node sits at the sensor mast and sends readings
over a 433 MHz radio link. The base station receives them and posts them to the
data ingest service.

## Boards

- `control-node` (ESP32-S3 devkit) samples all weather sensors
- `base-station` (LOLIN S3 Mini) relays readings to the ingest service

Pin and policy configuration lives in `src/control-node/constants.h` and
`src/base-station/config.h`.

## Data flow

The control node gathers a reading every 60 seconds, packs it into the
fixed-point `WeatherData` struct and sends it with the ARQ protocol over
CC1101. A failed sensor is marked with a sentinel value so the receiver omits
its field from the payload.

The base station unpacks the frame, stamps it with the current NTP time and
pushes it into an outbox. A separate task builds a JSON batch and POSTs it to
the ingest `/readings` endpoint, retrying transient failures with backoff.
The outbox keeps roughly two hours of readings when the network is down.

```mermaid
sequenceDiagram
    participant CN as Control Node
    participant BS as Base Station
    participant IN as Ingest Service

    loop Every 60s
        CN->>CN: Sample sensors and pack WeatherData
        CN->>BS: ARQ DATA_FRAME (433 MHz CC1101)
        alt Frame valid
            BS-->>CN: ARQ ACK
            BS->>BS: Unpack, stamp NTP time and push to RTC outbox
        else No ACK
            CN->>BS: Retry up to TX_MAX_ATTEMPTS
        end
    end

    loop OutboxSender task
        BS->>BS: Peek batch from outbox and build JSON
        BS->>IN: POST /readings
        alt 2xx OK
            IN-->>BS: 200 OK
            BS->>BS: Pop sent batch from outbox
        else Transient failure (5xx, 408, 429, network error)
            IN-->>BS: Error or timeout
            BS->>BS: Keep batch and backoff before retry
        else Permanent 4xx error
            IN-->>BS: 4xx Client Error
            BS->>BS: Drop oldest entry from outbox
        end
    end
```

## Flash

```bash
pio run -e control-node -e base-station
pio run -t upload -e control-node
pio run -t upload -e base-station
```
