#include <gtest/gtest.h>

#include <cmath>
#include <cstdint>

#include "WindVaneState.h"

namespace {

constexpr uint8_t kMagnetOk = 0x20;
constexpr uint8_t kMagnetWeak = 0x20 | 0x10;

uint16_t degToRaw(float deg) {
    return static_cast<uint16_t>(std::lround(deg / sensors::kRawToDegrees)) & 0x0FFFu;
}

} // namespace

TEST(Sensors, WindVaneCircularWrapOffsetAndGlitches) {
    sensors::WindVaneState vane(0.0f);
    vane.reset(100);

    sensors::WindVaneSample s1;
    s1.nowMs = 100;
    s1.rawAngle = degToRaw(355.0f);
    s1.i2cOk = true;
    s1.hasDiag = true;
    s1.diagStatus = kMagnetOk;
    s1.diagAgc = 70;
    s1.samplePeriodMs = 1000;
    vane.step(s1);
    EXPECT_TRUE(vane.ready());
    EXPECT_NEAR(vane.angle(), 355.0f, 0.3f);

    // Averaging 355 deg and 5 deg must remain near 0/360 rather than jumping to 180 deg.
    sensors::WindVaneSample s2;
    s2.nowMs = 1100;
    s2.rawAngle = degToRaw(5.0f);
    s2.i2cOk = true;
    s2.samplePeriodMs = 1000;
    vane.step(s2);
    const float wrappedDist = (vane.angle() > 180.0f) ? (360.0f - vane.angle()) : vane.angle();
    EXPECT_LT(wrappedDist, 10.0f);

    // Changing calibration offset re-syncs the vector filter immediately on the next sample.
    vane.setCalibrationOffset(20.0f);
    sensors::WindVaneSample s3;
    s3.nowMs = 2100;
    s3.rawAngle = degToRaw(350.0f);
    s3.i2cOk = true;
    s3.samplePeriodMs = 1000;
    vane.step(s3);
    EXPECT_NEAR(vane.angle(), 10.0f, 0.3f);

    // I2C fault and unready magnet increment glitches and stop updating filtered angle.
    sensors::WindVaneSample fault;
    fault.nowMs = 3100;
    fault.i2cOk = false;
    vane.step(fault);
    EXPECT_FALSE(vane.ready());
    EXPECT_EQ(vane.glitchCount(), 1u);

    sensors::WindVaneSample weak;
    weak.nowMs = 4100;
    weak.rawAngle = degToRaw(120.0f);
    weak.i2cOk = true;
    weak.hasDiag = true;
    weak.diagStatus = kMagnetWeak;
    vane.step(weak);
    EXPECT_FALSE(vane.ready());
    EXPECT_EQ(vane.glitchCount(), 2u);
    EXPECT_NEAR(vane.angle(), 10.0f, 0.3f);
}

TEST(Sensors, WindVaneOppositeFlipAndWrapBounds) {
    sensors::WindVaneState vane(0.0f);
    vane.reset(0);

    // 180-degree opposite flip with samplePeriodMs=3000 (alpha=0.5f) must not collapse to 0 deg.
    sensors::WindVaneSample east;
    east.nowMs = 1000;
    east.rawAngle = degToRaw(90.0f);
    east.i2cOk = true;
    east.hasDiag = true;
    east.diagStatus = kMagnetOk;
    east.diagAgc = 75;
    east.samplePeriodMs = 3000;
    vane.step(east);
    ASSERT_NEAR(vane.angle(), 90.0f, 0.2f);

    sensors::WindVaneSample west;
    west.nowMs = 4000;
    west.rawAngle = degToRaw(270.0f);
    west.i2cOk = true;
    west.samplePeriodMs = 3000;
    vane.step(west);
    EXPECT_NEAR(vane.angle(), 270.0f, 0.5f);

    // Angle near 0/360 wrap remains strictly < 360.0f.
    vane.setCalibrationOffset(0.0001f);
    sensors::WindVaneSample wrapSample;
    wrapSample.nowMs = 5000;
    wrapSample.rawAngle = 0;
    wrapSample.i2cOk = true;
    wrapSample.samplePeriodMs = 1000;
    vane.step(wrapSample);
    EXPECT_GE(vane.angle(), 0.0f);
    EXPECT_LT(vane.angle(), 360.0f);

    vane.setCalibrationOffset(-0.0001f);
    wrapSample.nowMs = 6000;
    vane.step(wrapSample);
    EXPECT_GE(vane.angle(), 0.0f);
    EXPECT_LT(vane.angle(), 360.0f);
}
