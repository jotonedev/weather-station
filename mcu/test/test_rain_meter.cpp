#include <gtest/gtest.h>

#include <cmath>
#include <cstdint>

#include "RainMeterLogic.h"

namespace {

constexpr uint8_t kMagnetOk = 0x20;

} // namespace

TEST(Sensors, RainMeterAutoCalibrationTipAndWindShakeRejection) {
    sensors::RainMeterLogic meter(2.0f);
    meter.reset(0);

    uint32_t nowMs = 30;
    auto feed = [&](uint16_t raw, bool diag) {
        sensors::RainMeterSample s;
        s.nowMs = nowMs;
        s.rawAngle = raw;
        s.i2cOk = true;
        s.hasDiag = diag;
        s.diagStatus = kMagnetOk;
        s.diagAgc = 85;
        s.samplePeriodMs = 30;
        nowMs += 30;
        return meter.step(s);
    };

    // Establish first resting cluster around raw=100 (rel=0).
    for (int i = 0; i < 10; ++i) {
        const auto res = feed(100, i == 0);
        EXPECT_FALSE(res.saveCalibration);
    }
    EXPECT_EQ(meter.state(), sensors::RainMeterState::CALIBRATING);

    // Move smoothly to second resting cluster at raw=300 (rel=200) and settle.
    feed(250, false);
    sensors::RainMeterStepResult calRes;
    for (int i = 0; i < 10; ++i) {
        calRes = feed(300, false);
        if (calRes.saveCalibration) {
            break;
        }
    }
    ASSERT_TRUE(calRes.saveCalibration);
    EXPECT_EQ(calRes.calibration.rawReference, 100u);
    EXPECT_EQ(calRes.calibration.restLeft, 0);
    EXPECT_EQ(calRes.calibration.restRight, 200);
    EXPECT_TRUE(meter.isCalibrated());
    EXPECT_EQ(meter.state(), sensors::RainMeterState::REST_RIGHT);

    // Full tip from right basin to left basin records one tip.
    bool sawTipSave = false;
    for (int i = 0; i < 8; ++i) {
        const auto res = feed(100, false);
        sawTipSave = sawTipSave || res.saveRtcBins;
    }
    EXPECT_TRUE(sawTipSave);
    EXPECT_EQ(meter.state(), sensors::RainMeterState::REST_LEFT);
    EXPECT_NEAR(meter.rainfall1h(), 2.0f, 0.1f);
    EXPECT_NEAR(meter.rainRate(), 2.0f, 0.01f);

    const RainData packed = meter.rainData();
    EXPECT_EQ(packed.rainfall_1h, weather::encodeRainAccumulation(meter.rainfall1h()));
    EXPECT_EQ(packed.rainRate, 20u);

    // Wind shake: cross crest+hysteresis (140) into TRANSIT_TO_RIGHT, then fall back below 60.
    for (int i = 0; i < 10; ++i) {
        feed(247, false);
        if (meter.state() == sensors::RainMeterState::TRANSIT_TO_RIGHT) {
            break;
        }
    }
    ASSERT_EQ(meter.state(), sensors::RainMeterState::TRANSIT_TO_RIGHT);

    const float rainBeforeShake = meter.rainfall1h();
    for (int i = 0; i < 8; ++i) {
        feed(100, false);
        if (meter.state() == sensors::RainMeterState::REST_LEFT) {
            break;
        }
    }
    EXPECT_EQ(meter.state(), sensors::RainMeterState::REST_LEFT);
    EXPECT_FLOAT_EQ(meter.rainfall1h(), rainBeforeShake);
}

TEST(Sensors, RainMeterJamDetectionRecoveryAndOutlier) {
    sensors::RainMeterLogic meter(2.0f);
    const sensors::RainCalibration cal = {1000, 0, 400};
    meter.applyCalibration(cal);
    meter.reset(0);

    uint32_t nowMs = 30;
    auto feed = [&](uint16_t raw, bool diag) {
        sensors::RainMeterSample s;
        s.nowMs = nowMs;
        s.rawAngle = raw;
        s.i2cOk = true;
        s.hasDiag = diag;
        s.diagStatus = kMagnetOk;
        s.diagAgc = 85;
        s.samplePeriodMs = 30;
        nowMs += 30;
        return meter.step(s);
    };

    // First valid reading establishes prevPos=0 (at rawReference=1000).
    feed(1000, true);

    // Mechanical outlier > 350 counts increments glitchCount without moving state.
    feed(1450, false);
    EXPECT_EQ(meter.glitchCount(), 1u);
    EXPECT_EQ(meter.state(), sensors::RainMeterState::REST_LEFT);

    // Move in <350-count steps past crest+hysteresis (280) without reaching right stop (300).
    feed(1150, false);
    for (int i = 0; i < 8; ++i) {
        feed(1290, false);
        if (meter.state() == sensors::RainMeterState::TRANSIT_TO_RIGHT) {
            break;
        }
    }
    ASSERT_EQ(meter.state(), sensors::RainMeterState::TRANSIT_TO_RIGHT);

    // Stay mid-air past the 2000 ms jam timeout.
    nowMs += 2100;
    feed(1290, false);
    EXPECT_EQ(meter.state(), sensors::RainMeterState::JAMMED);
    EXPECT_TRUE(meter.isJammed());

    // Settle into right resting basin to clear the jam.
    for (int i = 0; i < 10; ++i) {
        feed(1400, false);
        if (!meter.isJammed()) {
            break;
        }
    }
    EXPECT_FALSE(meter.isJammed());
    EXPECT_EQ(meter.state(), sensors::RainMeterState::REST_RIGHT);
}

TEST(Sensors, RainMeterRollingWindowsAndDryDecay) {
    sensors::RainMeterLogic meter(2.0f);
    const sensors::RainCalibration cal = {0, 0, 400};
    meter.applyCalibration(cal);
    meter.reset(0);

    auto feedAt = [&](uint32_t tMs, uint16_t raw, bool diag) {
        sensors::RainMeterSample s;
        s.nowMs = tMs;
        s.rawAngle = raw;
        s.i2cOk = true;
        s.hasDiag = diag;
        s.diagStatus = kMagnetOk;
        s.diagAgc = 85;
        s.samplePeriodMs = 30;
        return meter.step(s);
    };

    // First tip from REST_LEFT to REST_RIGHT.
    feedAt(30, 0, true);
    feedAt(60, 200, false);
    uint32_t tMs = 90;
    for (int i = 0; i < 8; ++i) {
        feedAt(tMs, 400, false);
        tMs += 30;
    }
    ASSERT_EQ(meter.state(), sensors::RainMeterState::REST_RIGHT);
    const float initialRate = meter.rainRate();
    const float initialRain = meter.rainfall1h();
    ASSERT_GT(initialRate, 0.0f);
    ASSERT_GT(initialRain, 0.0f);

    // Advancing 12 minutes dry (>10 min threshold) triggers minute rotation and rate decay.
    const auto decayRes = feedAt(720000uL, 400, false);
    EXPECT_TRUE(decayRes.saveRtcBins);
    EXPECT_LT(meter.rainRate(), initialRate);

    // Advancing past 60 minutes flushes the 1h bins while retaining the 24h accumulation.
    const auto hourRes = feedAt(3660000uL, 400, false);
    EXPECT_TRUE(hourRes.saveRtcBins);
    EXPECT_FLOAT_EQ(meter.rainfall1h(), 0.0f);
    EXPECT_NEAR(meter.rainfall24h(), initialRain, 0.01f);

    // Advancing 24 one-hour steps clears the 24h bins as well.
    for (uint32_t h = 2; h <= 25; ++h) {
        feedAt((h * 3600000uL) + 60000uL, 400, false);
    }
    EXPECT_FLOAT_EQ(meter.rainfall24h(), 0.0f);
}

TEST(Sensors, RainMeterFastTipNvsBootRestRightAndClamp) {
    // Boot from NVS while resting at REST_RIGHT seeds REST_RIGHT with zero phantom tips.
    sensors::RainMeterLogic nvsMeter(0.2f);
    nvsMeter.applyCalibration({100, 0, 400});
    nvsMeter.reset(0);

    sensors::RainMeterSample bootSample;
    bootSample.nowMs = 100;
    bootSample.rawAngle = 500; // +400 relative to rawReference=100 (REST_RIGHT)
    bootSample.i2cOk = true;
    bootSample.hasDiag = true;
    bootSample.diagStatus = kMagnetOk;
    bootSample.diagAgc = 90;
    bootSample.samplePeriodMs = 100;
    const auto bootRes = nvsMeter.step(bootSample);
    EXPECT_EQ(nvsMeter.state(), sensors::RainMeterState::REST_RIGHT);
    EXPECT_EQ(nvsMeter.glitchCount(), 0u);
    EXPECT_FALSE(bootRes.saveRtcBins);
    EXPECT_FLOAT_EQ(nvsMeter.rainfall1h(), 0.0f);

    // Fast tip > 350 counts (400-count span in 100 ms) confirms on 2nd sample without deadlock,
    // and rejected 1st sample still rotates rolling windows.
    sensors::RainMeterSample spike1;
    spike1.nowMs = 61000;
    spike1.rawAngle = 100; // -400 jump back to REST_LEFT
    spike1.i2cOk = true;
    spike1.samplePeriodMs = 100;
    const auto res1 = nvsMeter.step(spike1);
    EXPECT_EQ(nvsMeter.glitchCount(), 1u);
    EXPECT_TRUE(res1.saveRtcBins);
    EXPECT_EQ(nvsMeter.state(), sensors::RainMeterState::REST_RIGHT);

    sensors::RainMeterSample spike2 = spike1;
    spike2.nowMs = 61100;
    nvsMeter.step(spike2);
    EXPECT_EQ(nvsMeter.state(), sensors::RainMeterState::TRANSIT_TO_LEFT);

    sensors::RainMeterSample settle = spike1;
    settle.nowMs = 61200;
    const auto tipRes = nvsMeter.step(settle);
    EXPECT_TRUE(tipRes.saveRtcBins);
    EXPECT_EQ(nvsMeter.state(), sensors::RainMeterState::REST_LEFT);
    EXPECT_GT(nvsMeter.rainfall1h(), 0.0f);

    // rainData() clamps >= 655.35 mm to kInvalidUint16 - 1.
    sensors::RainMeterLogic clampMeter(700.0f);
    clampMeter.applyCalibration({0, 0, 200});
    clampMeter.reset(0);
    auto feedClamp = [&](uint32_t tMs, uint16_t raw, bool diag) {
        sensors::RainMeterSample s;
        s.nowMs = tMs;
        s.rawAngle = raw;
        s.i2cOk = true;
        s.hasDiag = diag;
        s.diagStatus = kMagnetOk;
        s.diagAgc = 80;
        s.samplePeriodMs = 30;
        clampMeter.step(s);
    };
    feedClamp(30, 0, true);
    feedClamp(60, 120, false);
    for (int i = 0; i < 8; ++i) {
        feedClamp(90 + (static_cast<uint32_t>(i) * 30u), 200, false);
    }
    ASSERT_GT(clampMeter.rainfall1h(), 655.35f);
    const RainData packed = clampMeter.rainData();
    EXPECT_EQ(packed.rainfall_1h, weather::kInvalidUint16 - 1u);
    EXPECT_EQ(packed.rainfall_24h, weather::kInvalidUint16 - 1u);
    EXPECT_LE(packed.rainRate, weather::kInvalidUint16 - 1u);
}
