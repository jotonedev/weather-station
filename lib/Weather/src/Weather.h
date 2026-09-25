#ifndef WEATHER_H
#define WEATHER_H

#include <cstdint>

struct __attribute__((packed)) MagnetStatus {
    uint8_t status;
    uint8_t agc;
    uint16_t agcVariance;
};

struct __attribute__((packed)) WindData {
    uint16_t windSpeed;
    uint16_t windAngle;
    MagnetStatus anemometerMagnetStatus;
    MagnetStatus windVaneMagnetStatus;
};

struct __attribute__((packed)) AirData {
    int16_t temperature;
    uint16_t humidity;
    uint16_t pressure;
};

struct __attribute__((packed)) RainData {
    uint16_t rainfall_1h;
    uint16_t rainfall_24h;
};

struct __attribute__((packed)) AirQualityData {
    uint16_t pm1_0;
    uint16_t pm2_5;
    uint16_t pm10_0;
    uint16_t voc;
};

struct __attribute__((packed)) WeatherData {
    WindData windData;
    AirData airData;
    RainData rainData;
    AirQualityData airQualityData;
};

#endif // WEATHER_H