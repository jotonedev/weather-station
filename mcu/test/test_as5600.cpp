#include <gtest/gtest.h>

#include <cstdint>

#include "MagnetDiagnostics.h"

namespace {

constexpr uint8_t kMagnetOk = 0x20;
constexpr uint8_t kMagnetWeak = 0x20 | 0x10;
constexpr uint8_t kMagnetStrong = 0x20 | 0x08;

} // namespace

TEST(Sensors, MagnetDiagnosticsStatusAndVariance) {
    sensors::MagnetDiagnostics diag;
    diag.reset();
    EXPECT_FALSE(diag.sensorReady);

    diag.update(false, kMagnetOk, 100);
    EXPECT_FALSE(diag.sensorReady);
    EXPECT_EQ(diag.agcCount, 0u);

    diag.update(true, kMagnetWeak, 100);
    EXPECT_FALSE(diag.sensorReady);
    EXPECT_EQ(diag.agcCount, 1u);

    diag.update(true, kMagnetStrong, 110);
    EXPECT_FALSE(diag.sensorReady);
    EXPECT_EQ(diag.agcCount, 2u);

    diag.update(true, kMagnetOk, 120);
    EXPECT_TRUE(diag.sensorReady);
    EXPECT_EQ(diag.agcCount, 3u);
    EXPECT_EQ(diag.agcVariance(), 100u);

    diag.resetStats();
    EXPECT_EQ(diag.agcCount, 0u);
    EXPECT_EQ(diag.agcVariance(), 0u);
    EXPECT_EQ(diag.lastStatus, kMagnetOk);
    EXPECT_EQ(diag.lastAgc, 120u);
}

TEST(Sensors, MagnetDiagnosticsValidationAndI2cErrorPreservesStats) {
    sensors::MagnetDiagnostics diag;
    EXPECT_EQ(diag.lastStatus, sensors::MagnetDiagnostics::kStatusInvalid);
    diag.reset();
    EXPECT_EQ(diag.lastStatus, sensors::MagnetDiagnostics::kStatusInvalid);
    EXPECT_EQ(diag.lastMagnitude, 0u);
    EXPECT_FALSE(diag.sensorReady);

    // Valid MD=0 (no magnet detected) keeps status 0x00, !sensorReady, and does not add AGC.
    diag.update(true, 0x00u, 55u, 1234u);
    EXPECT_EQ(diag.lastStatus, 0x00u);
    EXPECT_EQ(diag.lastAgc, 55u);
    EXPECT_EQ(diag.lastMagnitude, 1234u);
    EXPECT_FALSE(diag.sensorReady);
    EXPECT_EQ(diag.agcCount, 0u);

    // Valid MD=1 updates Welford stats and stores magnitude.
    diag.update(true, kMagnetOk, 90u, 2000u);
    diag.update(true, kMagnetOk, 110u, 2100u);
    EXPECT_TRUE(diag.sensorReady);
    EXPECT_EQ(diag.lastMagnitude, 2100u);
    EXPECT_EQ(diag.agcCount, 2u);
    const uint16_t varianceBeforeFault = diag.agcVariance();
    EXPECT_GT(varianceBeforeFault, 0u);

    // Stuck-high 0xFF, reserved-bit 0xE7, and impossible ML|MH 0x18 map to kStatusI2cError.
    const uint8_t invalidStatuses[] = {0xFFu, 0xE7u, 0x18u};
    for (const uint8_t badStatus : invalidStatuses) {
        diag.update(true, badStatus, 42u, 999u);
        EXPECT_EQ(diag.lastStatus, sensors::MagnetDiagnostics::kStatusI2cError);
        EXPECT_EQ(diag.lastAgc, 0u);
        EXPECT_EQ(diag.lastMagnitude, 0u);
        EXPECT_FALSE(diag.sensorReady);
        EXPECT_EQ(diag.agcCount, 2u);
        EXPECT_EQ(diag.agcVariance(), varianceBeforeFault);
    }

    // markI2cError() sets kStatusI2cError while preserving Welford stats.
    diag.update(true, kMagnetOk, 100u, 1500u);
    EXPECT_EQ(diag.agcCount, 3u);
    diag.markI2cError();
    EXPECT_EQ(diag.lastStatus, sensors::MagnetDiagnostics::kStatusI2cError);
    EXPECT_EQ(diag.lastAgc, 0u);
    EXPECT_EQ(diag.lastMagnitude, 0u);
    EXPECT_FALSE(diag.sensorReady);
    EXPECT_EQ(diag.agcCount, 3u);
}

TEST(As5600, MagnetDiagnosticsIsHealthyStatus) {
    EXPECT_TRUE(sensors::MagnetDiagnostics::isHealthyStatus(kMagnetOk));
    EXPECT_FALSE(sensors::MagnetDiagnostics::isHealthyStatus(kMagnetWeak));
    EXPECT_FALSE(sensors::MagnetDiagnostics::isHealthyStatus(kMagnetStrong));
    EXPECT_FALSE(sensors::MagnetDiagnostics::isHealthyStatus(0x00));
    EXPECT_FALSE(sensors::MagnetDiagnostics::isHealthyStatus(kMagnetOk | 0x01));
    EXPECT_FALSE(sensors::MagnetDiagnostics::isHealthyStatus(kMagnetOk | 0x18));
}
