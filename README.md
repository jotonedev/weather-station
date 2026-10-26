# Weather Station

Overly complex weather station with 3D-printed sensor enclosures and a microservice backend for storage, forecasting and visualization.

> This project is still a work in progress. The MCU side is not tested yet.

I wanted to design a personal weather station to collect data and estimate weather in my area for a long time, and I also wanted to experiment with microservices. So I built this to challenge myself. 
The microservices side is built using [gRPC](https://grpc.io/) and [Uber Fx](https://uber-go.github.io/fx/). 
The weather station was built by remixing existing open source systems, customizing them with sensors I had lying around. 
My weather station is powered directly with 220 V, so no solar panel is used in the design (maybe I will add one in the future, who knows).

## Structure

- [`3dmodels/`](3dmodels/) contains CAD files and STLs for the Stevenson screen air box, anemometer and wind vane assembly and tipping-bucket rain gauge
- [`mcu/`](mcu/) contains PlatformIO firmware for the sensor mast control node and the indoor base station
- [`system/`](system/) contains the Go and Python microservices, protobuf contracts and Docker Compose and Helm deployments

## Overview

```mermaid
flowchart LR
    subgraph Enclosures["3dmodels/"]
        AirBox["Air Sensor Box (AHT20, BMP180, SGP30, PMS7003)"]
        Wind["Anemometer and Wind Vane (2x AS5600)"]
        Rain["Rain Gauge (AS5600)"]
    end

    subgraph Firmware["mcu/"]
        CN["Control Node (ESP32-S3)"]
        BS["Base Station (LOLIN S3 Mini)"]
    end

    subgraph Backend["system/"]
        Ingest["data-ingest (Go)"]
        DB[("database (Go / DuckDB)")]
        Job["inference-job (Go CronJob)"]
        Inf["inference-server (Python)"]
        FE["frontend (Go)"]
    end

    AirBox -->|I2C / UART| CN
    Wind -->|I2C / SoftI2C| CN
    Rain -->|I2C| CN

    CN -->|433 MHz CC1101 ARQ| BS
    BS -->|HTTP POST /readings| Ingest

    Ingest -->|gRPC| DB
    FE -->|gRPC| DB
    Job -->|gRPC| DB
    Job -->|gRPC| Inf
```
