#include <gtest/gtest.h>

#include <cstdint>
#include <limits>

#include "Ingest.h"

namespace {

// A reading with every measurement and diagnostic marked absent.
WeatherData sentinelReading() {
    WeatherData data = {};
    data.windData.windSpeed = weather::kInvalidUint16;
    data.windData.windAngle = weather::kInvalidUint16;
    data.windData.anemometerMagnetStatus.status = weather::kInvalidMagnetStatus;
    data.windData.windVaneMagnetStatus.status = weather::kInvalidMagnetStatus;
    data.airData.temperature = weather::kInvalidTemperature;
    data.airData.humidity = weather::kInvalidUint16;
    data.airData.pressure = weather::kInvalidUint16;
    data.rainData.rainfall_1h = weather::kInvalidUint16;
    data.rainData.rainfall_24h = weather::kInvalidUint16;
    data.rainData.rainRate = weather::kInvalidUint16;
    data.rainData.magnetStatus.status = weather::kInvalidMagnetStatus;
    data.airQualityData.pm1_0 = weather::kInvalidUint16;
    data.airQualityData.pm2_5 = weather::kInvalidUint16;
    data.airQualityData.pm10_0 = weather::kInvalidUint16;
    return data;
}

} // namespace

TEST(Ingest, FormatFloatTrimsZeros) {
    char out[32];
    EXPECT_EQ(formatFloat(out, sizeof(out), 21.50f), 4);
    EXPECT_STREQ(out, "21.5");

    EXPECT_EQ(formatFloat(out, sizeof(out), 10.00f), 2);
    EXPECT_STREQ(out, "10");

    EXPECT_EQ(formatFloat(out, sizeof(out), 0.0f), 1);
    EXPECT_STREQ(out, "0");

    EXPECT_EQ(formatFloat(out, sizeof(out), 0.05f), 4);
    EXPECT_STREQ(out, "0.05");

    EXPECT_EQ(formatFloat(out, sizeof(out), 0.10f), 3);
    EXPECT_STREQ(out, "0.1");
}

TEST(Ingest, FormatFloatRejectsNonFinite) {
    char out[32];
    EXPECT_EQ(formatFloat(out, sizeof(out), std::numeric_limits<float>::quiet_NaN()), 0);
    EXPECT_EQ(formatFloat(out, sizeof(out), std::numeric_limits<float>::infinity()), 0);
    EXPECT_EQ(formatFloat(out, sizeof(out), -std::numeric_limits<float>::infinity()), 0);
}

TEST(Ingest, FormatFloatNegativeNearZeroAndBufferBounds) {
    char out[32];
    EXPECT_EQ(formatFloat(out, sizeof(out), -3.2f), 4);
    EXPECT_STREQ(out, "-3.2");

    EXPECT_EQ(formatFloat(out, sizeof(out), -10.0f), 3);
    EXPECT_STREQ(out, "-10");

    EXPECT_EQ(formatFloat(out, sizeof(out), -0.0f), 1);
    EXPECT_STREQ(out, "0");

    EXPECT_EQ(formatFloat(out, sizeof(out), 0.001f), 1);
    EXPECT_STREQ(out, "0");

    EXPECT_EQ(formatFloat(out, sizeof(out), -0.001f), 1);
    EXPECT_STREQ(out, "0");

    EXPECT_EQ(formatFloat(out, 0, 1.0f), 0);
    EXPECT_EQ(formatFloat(nullptr, 8, 1.0f), 0);

    char buf5[5];
    char buf6[6];
    EXPECT_EQ(formatFloat(buf5, sizeof(buf5), 21.50f), 0);
    EXPECT_EQ(formatFloat(buf6, sizeof(buf6), 21.50f), 4);
    EXPECT_STREQ(buf6, "21.5");

    EXPECT_EQ(formatFloat(buf5, sizeof(buf5), 12.34f), 0);
    EXPECT_EQ(formatFloat(buf6, sizeof(buf6), 12.34f), 5);
    EXPECT_STREQ(buf6, "12.34");
}

TEST(Ingest, FormatIso8601) {
    char out[32];
    EXPECT_EQ(formatIso8601(0, out, sizeof(out)), 20);
    EXPECT_STREQ(out, "1970-01-01T00:00:00Z");

    EXPECT_EQ(formatIso8601(1700000000, out, sizeof(out)), 20);
    EXPECT_STREQ(out, "2023-11-14T22:13:20Z");
}

TEST(Ingest, FormatIso8601RejectsSmallBuffer) {
    char exactFail[20];
    EXPECT_EQ(formatIso8601(1700000000, exactFail, sizeof(exactFail)), 0);
    EXPECT_EQ(formatIso8601(1700000000, nullptr, 21), 0);

    char exactOk[21];
    EXPECT_EQ(formatIso8601(1700000000, exactOk, sizeof(exactOk)), 20);
    EXPECT_STREQ(exactOk, "2023-11-14T22:13:20Z");
}

TEST(Ingest, SanitizeNodeId) {
    char out[32];
    EXPECT_TRUE(sanitizeNodeId("Weather Mast #1", out, sizeof(out)));
    EXPECT_STREQ(out, "WeatherMast1");

    EXPECT_TRUE(sanitizeNodeId("a/b c", out, sizeof(out)));
    EXPECT_STREQ(out, "abc");

    EXPECT_FALSE(sanitizeNodeId("!!!", out, sizeof(out)));
}

TEST(Ingest, SanitizeNodeIdEdgeCases) {
    char out[32];
    EXPECT_FALSE(sanitizeNodeId(nullptr, out, sizeof(out)));
    EXPECT_FALSE(sanitizeNodeId("n1", nullptr, 8));
    EXPECT_FALSE(sanitizeNodeId("n1", out, 0));
    EXPECT_FALSE(sanitizeNodeId("n1", out, 1));
    EXPECT_FALSE(sanitizeNodeId("", out, sizeof(out)));
    EXPECT_FALSE(sanitizeNodeId(" \t\n!@# ", out, sizeof(out)));

    char small[5];
    EXPECT_TRUE(sanitizeNodeId("  ab-cdef", small, sizeof(small)));
    EXPECT_STREQ(small, "ab-c");
}

TEST(Ingest, SinglePayloadFullFields) {
    WeatherData data = {};
    data.windData.windSpeed = weather::encodeWindSpeed(4.8f);
    data.windData.windAngle = weather::encodeWindAngle(180.0f);
    data.windData.anemometerMagnetStatus.status = 32;
    data.windData.anemometerMagnetStatus.agc = 128;
    data.windData.anemometerMagnetStatus.agcVariance = 2;
    data.windData.windVaneMagnetStatus.status = weather::kInvalidMagnetStatus;
    data.airData.temperature = weather::encodeTemperature(21.5f);
    data.airData.humidity = weather::encodeHumidity(55.2f);
    data.airData.pressure = weather::encodePressure(1013.5f);
    data.rainData.rainfall_1h = 25;
    data.rainData.rainfall_24h = 321;
    data.rainData.rainRate = 120;
    data.rainData.magnetStatus.status = 32;
    data.rainData.magnetStatus.agc = 130;
    data.rainData.magnetStatus.agcVariance = 0;
    data.airQualityData.pm1_0 = 8;
    data.airQualityData.pm2_5 = 14;
    data.airQualityData.pm10_0 = 22;

    IngestReading const reading = {data, 1700000000u};

    char payload[1024];
    int const len = buildSinglePayload(reading, "weather-mast", payload, sizeof(payload));
    ASSERT_GT(len, 0);
    EXPECT_STREQ(payload, "{\"node_id\":\"weather-mast\",\"timestamp\":\"2023-11-14T22:13:20Z\","
                          "\"wind_speed\":4.8,\"wind_angle\":180,\"anemo_status\":32,"
                          "\"anemo_agc\":128,\"anemo_agc_variance\":2,\"temperature\":21.5,"
                          "\"humidity\":55.2,\"pressure\":1013.5,\"rain_1h\":0.25,"
                          "\"rain_24h\":3.21,\"rain_rate\":12,\"rain_status\":32,\"rain_agc\":130,"
                          "\"rain_agc_variance\":0,\"pm1_0\":8,\"pm2_5\":14,\"pm10_0\":22}");
}

TEST(Ingest, SinglePayloadOmitsAbsentFieldsAndTimestampWhenNoEpoch) {
    IngestReading const reading = {sentinelReading(), 0};

    char payload[256];
    int const len = buildSinglePayload(reading, "weather-mast", payload, sizeof(payload));
    ASSERT_GT(len, 0);
    EXPECT_STREQ(payload, "{\"node_id\":\"weather-mast\"}");
}

TEST(Ingest, SinglePayloadSerializesWindVaneMagnetStatus) {
    WeatherData data = sentinelReading();
    data.windData.windVaneMagnetStatus = {32, 110, 5};
    IngestReading const reading = {data, 0};

    char payload[256];
    int const len = buildSinglePayload(reading, "n1", payload, sizeof(payload));
    ASSERT_GT(len, 0);
    EXPECT_STREQ(payload, "{\"node_id\":\"n1\",\"vane_status\":32,\"vane_agc\":110,"
                          "\"vane_agc_variance\":5}");
}

TEST(Ingest, BatchPayload) {
    IngestReading readings[2] = {
        {sentinelReading(), 0},
        {sentinelReading(), 1},
    };

    char payload[512];
    int const len = buildBatchPayload(readings, 2, "n1", payload, sizeof(payload));
    ASSERT_GT(len, 0);
    EXPECT_STREQ(payload, "[{\"node_id\":\"n1\"},{\"node_id\":\"n1\","
                          "\"timestamp\":\"1970-01-01T00:00:01Z\"}]");
}

TEST(Ingest, BatchRejectsEmpty) {
    IngestReading const readings[1] = {{sentinelReading(), 0}};
    char payload[64];
    EXPECT_EQ(buildBatchPayload(nullptr, 1, "n1", payload, sizeof(payload)), 0);
    EXPECT_EQ(buildBatchPayload(readings, 0, "n1", payload, sizeof(payload)), 0);
    EXPECT_EQ(buildBatchPayload(readings, 1, "n1", nullptr, sizeof(payload)), 0);
    EXPECT_EQ(buildBatchPayload(readings, 1, "n1", payload, 0), 0);
    EXPECT_EQ(buildBatchPayload(readings, 1, nullptr, payload, sizeof(payload)), 0);
    EXPECT_EQ(buildSinglePayload(readings[0], nullptr, payload, sizeof(payload)), 0);
}

TEST(Ingest, BatchRejectsMidBatchOverflow) {
    IngestReading const readings[2] = {
        {sentinelReading(), 0},
        {sentinelReading(), 0},
    };
    char buf20[20];
    char buf35[35];
    char buf36[36];

    EXPECT_EQ(buildBatchPayload(readings, 2, "n1", buf20, sizeof(buf20)), 0);
    EXPECT_EQ(buildBatchPayload(readings, 2, "n1", buf35, sizeof(buf35)), 0);
    EXPECT_EQ(buildBatchPayload(readings, 2, "n1", buf36, sizeof(buf36)), 35);
    EXPECT_STREQ(buf36, R"([{"node_id":"n1"},{"node_id":"n1"}])");
}

TEST(Ingest, BackoffDelays) {
    BackoffPolicy policy;
    policy.minDelayMs = 1000;
    policy.maxDelayMs = 60000;
    policy.multiplier = 2;
    policy.maxAttempts = 0;

    EXPECT_EQ(backoffDelayMs(policy, 0), 0);
    EXPECT_EQ(backoffDelayMs(policy, 1), 1000);
    EXPECT_EQ(backoffDelayMs(policy, 2), 2000);
    EXPECT_EQ(backoffDelayMs(policy, 3), 4000);
    EXPECT_EQ(backoffDelayMs(policy, 7), 60000);
}

TEST(Ingest, BackoffDelayEdgeCases) {
    BackoffPolicy zeroMult;
    zeroMult.minDelayMs = 1000;
    zeroMult.maxDelayMs = 60000;
    zeroMult.multiplier = 0;
    EXPECT_EQ(backoffDelayMs(zeroMult, 0), 0u);
    EXPECT_EQ(backoffDelayMs(zeroMult, 1), 1000u);
    EXPECT_EQ(backoffDelayMs(zeroMult, 2), 0u);
    EXPECT_EQ(backoffDelayMs(zeroMult, 3), 0u);

    BackoffPolicy invertedBounds;
    invertedBounds.minDelayMs = 5000;
    invertedBounds.maxDelayMs = 1000;
    invertedBounds.multiplier = 2;
    EXPECT_EQ(backoffDelayMs(invertedBounds, 0), 0u);
    EXPECT_EQ(backoffDelayMs(invertedBounds, 1), 1000u);
    EXPECT_EQ(backoffDelayMs(invertedBounds, 2), 1000u);

    BackoffPolicy standard;
    standard.minDelayMs = 1000;
    standard.maxDelayMs = 60000;
    standard.multiplier = 2;
    EXPECT_EQ(backoffDelayMs(standard, 255), 60000u);

    BackoffPolicy largeMult;
    largeMult.minDelayMs = std::numeric_limits<uint32_t>::max() - 1;
    largeMult.maxDelayMs = std::numeric_limits<uint32_t>::max();
    largeMult.multiplier = std::numeric_limits<uint32_t>::max();
    EXPECT_EQ(backoffDelayMs(largeMult, 1), std::numeric_limits<uint32_t>::max() - 1);
    EXPECT_EQ(backoffDelayMs(largeMult, 2), std::numeric_limits<uint32_t>::max());
}

TEST(Ingest, BackoffRetryLimit) {
    BackoffPolicy policy;
    policy.maxAttempts = 3;
    EXPECT_TRUE(shouldRetry(policy, 0));
    EXPECT_TRUE(shouldRetry(policy, 1));
    EXPECT_TRUE(shouldRetry(policy, 2));
    EXPECT_FALSE(shouldRetry(policy, 3));

    policy.maxAttempts = 0;
    EXPECT_TRUE(shouldRetry(policy, 99));
}

TEST(Ingest, SinglePayloadRejectsSmallBuffer) {
    WeatherData data = sentinelReading();
    data.airData.temperature = weather::encodeTemperature(18.4f);
    IngestReading const reading = {data, 1};
    char payload[16];
    EXPECT_EQ(buildSinglePayload(reading, "n1", payload, sizeof(payload)), 0);
}
