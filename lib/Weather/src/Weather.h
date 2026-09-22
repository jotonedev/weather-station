#ifndef WEATHER_H
#define WEATHER_H

#include <cstdint>

enum WindDirection : uint8_t {
    NORTH = 0,
    NORTH_EAST = 1,
    EAST = 2,
    SOUTH_EAST = 3,
    SOUTH = 4,
    SOUTH_WEST = 5,
    WEST = 6,
    NORTH_WEST = 7,
    UNKNOWN = 255
};

struct __attribute__((packed)) WeatherData {
    uint16_t windSpeed;
    WindDirection windDirection;

    int16_t temperature;
    uint16_t humidity;
    uint16_t pressure;

    uint16_t rainfall_1h;
    uint16_t rainfall_24h;

    uint16_t pm1_0;
    uint16_t pm2_5;
    uint16_t pm10_0;
    uint16_t voc;
};

#endif // WEATHER_H