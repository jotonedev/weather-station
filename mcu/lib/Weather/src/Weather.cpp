#include "Weather.h"

#include <algorithm>
#include <cmath>
#include <cstring>
#include <limits>

namespace weather {

int16_t encodeTemperature(float celsius) {
    long rounded = std::lroundf(celsius * kTemperatureScale);
    rounded = std::clamp(rounded,
                         static_cast<long>(std::numeric_limits<int16_t>::min()) + 1L,
                         static_cast<long>(std::numeric_limits<int16_t>::max()));
    return static_cast<int16_t>(rounded);
}

uint16_t encodeUnsigned(float value, float scale) {
    long rounded = std::lroundf(value * scale);
    rounded = std::clamp(rounded, 0L,
                         static_cast<long>(std::numeric_limits<uint16_t>::max()));
    return static_cast<uint16_t>(rounded);
}

uint16_t encodeHumidity(float percent) {
    return encodeUnsigned(percent, kHumidityScale);
}

uint16_t encodePressure(float hectopascal) {
    return encodeUnsigned(hectopascal, kPressureScale);
}

uint16_t encodeWindSpeed(float metersPerSecond) {
    return encodeUnsigned(metersPerSecond, kWindSpeedScale);
}

uint16_t encodeWindAngle(float degrees) {
    return encodeUnsigned(degrees, kWindAngleScale);
}

float decodeTemperature(int16_t raw) {
    return static_cast<float>(raw) / kTemperatureScale;
}

float decodeHumidity(uint16_t raw) {
    return static_cast<float>(raw) / kHumidityScale;
}

float decodePressure(uint16_t raw) {
    return static_cast<float>(raw) / kPressureScale;
}

float decodeWindSpeed(uint16_t raw) {
    return static_cast<float>(raw) / kWindSpeedScale;
}

float decodeWindAngle(uint16_t raw) {
    return static_cast<float>(raw) / kWindAngleScale;
}

float decodeRainAccumulation(uint16_t raw) {
    return static_cast<float>(raw) / kRainAccumulationScale;
}

float decodeRainRate(uint16_t raw) {
    return static_cast<float>(raw) / kRainRateScale;
}

bool packWeatherData(const WeatherData& data, uint8_t* out, size_t capacity) {
    if (capacity < sizeof(WeatherData)) {
        return false;
    }
    std::memcpy(out, &data, sizeof(WeatherData));
    return true;
}

bool unpackWeatherData(const uint8_t* in, size_t length, WeatherData& out) {
    if (length < sizeof(WeatherData)) {
        return false;
    }
    std::memcpy(&out, in, sizeof(WeatherData));
    return true;
}

}  // namespace weather
