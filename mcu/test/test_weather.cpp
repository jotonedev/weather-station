#include <gtest/gtest.h>

#include <cstdint>

#include "Weather.h"

TEST(Weather, WireSizeFitsArqBudget) {
    EXPECT_EQ(sizeof(WeatherData), 36);
    EXPECT_LE(sizeof(WeatherData), weather::kMaxWirePayload);
}

TEST(Weather, TemperatureRoundTrip) {
    EXPECT_EQ(weather::encodeTemperature(21.5f), 215);
    EXPECT_EQ(weather::encodeTemperature(-3.2f), -32);
    EXPECT_NEAR(weather::decodeTemperature(215), 21.5f, 0.01f);
    EXPECT_NEAR(weather::decodeTemperature(-32), -3.2f, 0.01f);
}

TEST(Weather, HumidityPressureWindRoundTrip) {
    EXPECT_EQ(weather::encodeHumidity(55.2f), 552);
    EXPECT_NEAR(weather::decodeHumidity(552), 55.2f, 0.01f);

    EXPECT_EQ(weather::encodePressure(1013.5f), 10135);
    EXPECT_NEAR(weather::decodePressure(10135), 1013.5f, 0.01f);

    EXPECT_EQ(weather::encodeWindSpeed(4.8f), 480);
    EXPECT_NEAR(weather::decodeWindSpeed(480), 4.8f, 0.01f);

    EXPECT_EQ(weather::encodeWindAngle(180.0f), 1800);
    EXPECT_NEAR(weather::decodeWindAngle(1800), 180.0f, 0.01f);
}

TEST(Weather, RainDecode) {
    EXPECT_NEAR(weather::decodeRainAccumulation(25), 0.25f, 0.01f);
    EXPECT_NEAR(weather::decodeRainAccumulation(321), 3.21f, 0.01f);
    EXPECT_NEAR(weather::decodeRainRate(120), 12.0f, 0.01f);
}

TEST(Weather, EncodeClampsToRange) {
    EXPECT_EQ(weather::encodeTemperature(4000.0f), 32767);
    EXPECT_EQ(weather::encodeHumidity(101.0f), 1010);
    EXPECT_EQ(weather::encodeWindSpeed(-5.0f), 0);
    EXPECT_EQ(weather::encodeWindAngle(9000.0f), 65535);
}

TEST(Weather, PackAndUnpackRoundTrip) {
    WeatherData data = {};
    data.windData.windSpeed = weather::encodeWindSpeed(3.25f);
    data.windData.windAngle = weather::encodeWindAngle(90.5f);
    data.airData.temperature = weather::encodeTemperature(18.4f);
    data.airData.humidity = weather::encodeHumidity(61.0f);
    data.rainData.rainfall_1h = 25;
    data.airQualityData.pm2_5 = 14;

    uint8_t wire[weather::kMaxWirePayload];
    ASSERT_TRUE(weather::packWeatherData(data, wire, sizeof(wire)));

    WeatherData out = {};
    ASSERT_TRUE(weather::unpackWeatherData(wire, sizeof(WeatherData), out));
    EXPECT_EQ(out.windData.windSpeed, data.windData.windSpeed);
    EXPECT_EQ(out.windData.windAngle, data.windData.windAngle);
    EXPECT_EQ(out.airData.temperature, data.airData.temperature);
    EXPECT_EQ(out.rainData.rainfall_1h, data.rainData.rainfall_1h);
    EXPECT_EQ(out.airQualityData.pm2_5, data.airQualityData.pm2_5);
    EXPECT_EQ(out.windData.anemometerMagnetStatus.status,
              data.windData.anemometerMagnetStatus.status);
}

TEST(Weather, PackRejectsSmallBuffer) {
    WeatherData const data = {};
    uint8_t small[4];
    EXPECT_FALSE(weather::packWeatherData(data, small, sizeof(small)));
}

TEST(Weather, UnpackRejectsShortFrame) {
    uint8_t wire[10] = {0};
    WeatherData out = {};
    EXPECT_FALSE(weather::unpackWeatherData(wire, sizeof(wire), out));
}
