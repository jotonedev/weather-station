#ifndef WEATHER_H
#define WEATHER_H

#include <cstddef>
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
    uint16_t rainRate;
    MagnetStatus magnetStatus;
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

// The wire format shared by the control node and the base station.
//
// Sensor floats are encoded as fixed-point integers so a full reading fits a
// single CC1101 packet. The scales live in this single place so both nodes
// interpret the packed fields identically.
namespace weather {

// One LSB is 0.1 deg C.
constexpr float kTemperatureScale = 10.0f;
// One LSB is 0.1 percent.
constexpr float kHumidityScale = 10.0f;
// One LSB is 0.1 hPa.
constexpr float kPressureScale = 10.0f;
// One LSB is 0.01 m/s.
constexpr float kWindSpeedScale = 100.0f;
// One LSB is 0.1 deg.
constexpr float kWindAngleScale = 10.0f;
// One LSB is 0.01 mm, matches the rain meter driver.
constexpr float kRainAccumulationScale = 100.0f;
// One LSB is 0.1 mm/h, matches the rain meter driver.
constexpr float kRainRateScale = 10.0f;

// A sentinel marks a field as absent so the receiver omits it from the ingest
// payload. A failed sensor writes sentinels instead of stale or invalid data.
constexpr int16_t kInvalidTemperature = static_cast<int16_t>(0x8000);
constexpr uint16_t kInvalidUint16 = 0xFFFF;
constexpr uint8_t kInvalidMagnetStatus = 0xFF;

// Largest payload the ARQ link accepts. ARQ initializes the CC1101 with a
// fixed packet length of 48 bytes, 1 of which is the address byte, and 7 of
// which form the ARQ header, leaving 40 bytes of data payload.
constexpr size_t kMaxWirePayload = 40;

// Round a sensor float to its fixed-point integer.
int16_t encodeTemperature(float celsius);
uint16_t encodeHumidity(float percent);
uint16_t encodePressure(float hectopascal);
uint16_t encodeWindSpeed(float metersPerSecond);
uint16_t encodeWindAngle(float degrees);

// Convert a fixed-point field back to the sensor float.
float decodeTemperature(int16_t raw);
float decodeHumidity(uint16_t raw);
float decodePressure(uint16_t raw);
float decodeWindSpeed(uint16_t raw);
float decodeWindAngle(uint16_t raw);

// Rain fields are pre-encoded by the rain meter driver. Decode to millimeters
// and millimeters per hour.
float decodeRainAccumulation(uint16_t raw);
float decodeRainRate(uint16_t raw);

// Copy a packed reading to or from a byte buffer.
bool packWeatherData(const WeatherData& data, uint8_t* out, size_t capacity);
bool unpackWeatherData(const uint8_t* in, size_t length, WeatherData& out);

}  // namespace weather

static_assert(sizeof(WeatherData) <= weather::kMaxWirePayload,
              "WeatherData must fit the ARQ payload budget");

#endif  // WEATHER_H
