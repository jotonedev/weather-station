#include <gtest/gtest.h>

#include <cmath>
#include <cstdint>

#include "Sensors.h"

namespace {

constexpr uint8_t kMagnetOk = 0x20;
constexpr uint8_t kMagnetWeak = 0x20 | 0x10;
constexpr uint8_t kMagnetStrong = 0x20 | 0x08;

uint16_t degToRaw(float deg) {
    return static_cast<uint16_t>(std::lround(deg / sensors::kRawToDegrees)) & 0x0FFFu;
}

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

TEST(Sensors, AnemometerSteadyRotationAndWraparound) {
    sensors::AnemometerState state;
    state.reset(100);
    state.setSpeedFactor(0.1f);

    uint32_t nowMs = 100;
    uint32_t nowUs = 100000;
    uint16_t raw = 3800;

    // Step forward at +100 counts per 10 ms (~146.48 RPM), crossing 4096 wraparound.
    for (int i = 0; i < 40; ++i) {
        sensors::AnemometerSample sample;
        sample.nowMs = nowMs;
        sample.nowUs = nowUs;
        sample.rawAngle = raw;
        sample.i2cOk = true;
        sample.hasDiag = (i == 0);
        sample.diagStatus = kMagnetOk;
        sample.diagAgc = 90;
        state.step(sample);

        nowMs += 10;
        nowUs += 10000;
        raw = static_cast<uint16_t>((raw + 100u) & 0x0FFFu);
    }

    const float expectedRpm = (100.0f * 60.0f) / (0.010f * 4096.0f);
    EXPECT_TRUE(state.ready());
    EXPECT_EQ(state.glitchCount(), 0u);
    EXPECT_EQ(state.cumulativeCounts(), 3900);
    EXPECT_NEAR(state.rpm(), expectedRpm, 1.5f);
    EXPECT_NEAR(state.windSpeed(), state.rpm() * 0.1f, 0.05f);

    // Reverse (CCW) rotation still yields positive wind RPM and decrements counts.
    state.reset(1000);
    nowMs = 1000;
    nowUs = 1000000;
    raw = 200;
    for (int i = 0; i < 40; ++i) {
        sensors::AnemometerSample sample;
        sample.nowMs = nowMs;
        sample.nowUs = nowUs;
        sample.rawAngle = raw;
        sample.i2cOk = true;
        sample.hasDiag = (i == 0);
        sample.diagStatus = kMagnetOk;
        sample.diagAgc = 90;
        state.step(sample);

        nowMs += 10;
        nowUs += 10000;
        raw = static_cast<uint16_t>((raw + 4096u - 100u) & 0x0FFFu);
    }
    EXPECT_EQ(state.cumulativeCounts(), -3900);
    EXPECT_NEAR(state.rpm(), expectedRpm, 1.5f);

    // Standstill jitter <= 3 counts over the window snaps filtered RPM to zero.
    for (int i = 0; i < 45; ++i) {
        sensors::AnemometerSample sample;
        sample.nowMs = nowMs;
        sample.nowUs = nowUs;
        sample.rawAngle = static_cast<uint16_t>((raw + (i % 2)) & 0x0FFFu);
        sample.i2cOk = true;
        state.step(sample);
        nowMs += 10;
        nowUs += 10000;
    }
    EXPECT_FLOAT_EQ(state.rpm(), 0.0f);
    EXPECT_FLOAT_EQ(state.windSpeed(), 0.0f);
}

TEST(Sensors, AnemometerOutlierCoastingRecoveryAndDecay) {
    sensors::AnemometerState state;
    state.reset(100);

    uint32_t nowMs = 100;
    uint32_t nowUs = 100000;
    uint16_t raw = 0;

    for (int i = 0; i < 30; ++i) {
        sensors::AnemometerSample sample;
        sample.nowMs = nowMs;
        sample.nowUs = nowUs;
        sample.rawAngle = raw;
        sample.i2cOk = true;
        sample.hasDiag = (i == 0);
        sample.diagStatus = kMagnetOk;
        sample.diagAgc = 80;
        state.step(sample);
        nowMs += 10;
        nowUs += 10000;
        raw = static_cast<uint16_t>((raw + 100u) & 0x0FFFu);
    }
    const float rpmBeforeDropout = state.rpm();
    const int64_t countsBeforeDropout = state.cumulativeCounts();

    // Impossible >350-count jump enters dropout and coasts forward with mild friction.
    sensors::AnemometerSample glitch;
    glitch.nowMs = nowMs;
    glitch.nowUs = nowUs;
    glitch.rawAngle = static_cast<uint16_t>((raw + 600u) & 0x0FFFu);
    glitch.i2cOk = true;
    state.step(glitch);

    EXPECT_EQ(state.glitchCount(), 1u);
    EXPECT_GT(state.cumulativeCounts(), countsBeforeDropout);
    EXPECT_NEAR(state.rpm(), rpmBeforeDropout * 0.999f, 0.2f);

    // Valid sample within <1.5 s re-anchors cleanly without teleporting cumulativeCounts.
    const int64_t countsAfterCoast = state.cumulativeCounts();
    nowMs += 10;
    nowUs += 10000;
    sensors::AnemometerSample recover;
    recover.nowMs = nowMs;
    recover.nowUs = nowUs;
    recover.rawAngle = raw;
    recover.i2cOk = true;
    state.step(recover);
    EXPECT_EQ(state.cumulativeCounts(), countsAfterCoast);

    // Prolonged I2C dropout >= 1.5 s decays filtered RPM to zero.
    nowMs += 1600;
    nowUs += 1600000;
    for (int i = 0; i < 45; ++i) {
        sensors::AnemometerSample lost;
        lost.nowMs = nowMs;
        lost.nowUs = nowUs;
        lost.i2cOk = false;
        lost.hasDiag = true;
        state.step(lost);
        nowMs += 10;
        nowUs += 10000;
    }
    EXPECT_FALSE(state.ready());
    EXPECT_FLOAT_EQ(state.rpm(), 0.0f);
}

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
    EXPECT_EQ(packed.rainfall_1h, static_cast<uint16_t>(meter.rainfall1h() * 100.0f));
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
        feedAt(h * 3600000uL + 60000uL, 400, false);
    }
    EXPECT_FLOAT_EQ(meter.rainfall24h(), 0.0f);
}

TEST(Sensors, AnemometerStandstillDetectionAndInstantWake) {
    sensors::AnemometerState state;
    state.reset(0);
    EXPECT_FALSE(state.isStandstill());

    uint32_t nowMs = 0;
    uint32_t nowUs = 0;
    uint16_t raw = 1000;

    // Spin briefly so filteredRpm_ > 0, confirming standstill stays false.
    for (int i = 0; i < 20; ++i) {
        sensors::AnemometerSample s;
        s.nowMs = nowMs;
        s.nowUs = nowUs;
        s.rawAngle = raw;
        s.i2cOk = true;
        s.hasDiag = (i == 0);
        s.diagStatus = kMagnetOk;
        s.diagAgc = 80;
        state.step(s);
        nowMs += 10;
        nowUs += 10000;
        raw = static_cast<uint16_t>((raw + 100u) & 0x0FFFu);
    }
    EXPECT_FALSE(state.isStandstill());

    // Stop moving; decay to 0.0 RPM first, then hold below 3000 ms.
    for (int i = 0; i < 45; ++i) {
        sensors::AnemometerSample s;
        s.nowMs = nowMs;
        s.nowUs = nowUs;
        s.rawAngle = raw;
        s.i2cOk = true;
        state.step(s);
        nowMs += 10;
        nowUs += 10000;
    }
    ASSERT_FLOAT_EQ(state.rpm(), 0.0f);

    for (int i = 0; i < 60; ++i) {
        sensors::AnemometerSample s;
        s.nowMs = nowMs;
        s.nowUs = nowUs;
        s.rawAngle = static_cast<uint16_t>(raw + (i % 2));
        s.i2cOk = true;
        state.step(s);
        nowMs += 40;
        nowUs += 40000;
    }
    // 60 * 40 ms = 2400 ms < 3000 ms threshold.
    EXPECT_FALSE(state.isStandstill());

    // Advance past 3000 ms of sustained standstill jitter.
    for (int i = 0; i < 25; ++i) {
        sensors::AnemometerSample s;
        s.nowMs = nowMs;
        s.nowUs = nowUs;
        s.rawAngle = static_cast<uint16_t>(raw + (i % 2));
        s.i2cOk = true;
        state.step(s);
        nowMs += 40;
        nowUs += 40000;
    }
    EXPECT_TRUE(state.isStandstill());

    // A single sample with |delta| > 3 snaps out of standstill immediately.
    sensors::AnemometerSample wake;
    wake.nowMs = nowMs;
    wake.nowUs = nowUs;
    wake.rawAngle = static_cast<uint16_t>((raw + 5u) & 0x0FFFu);
    wake.i2cOk = true;
    state.step(wake);
    EXPECT_FALSE(state.isStandstill());

    // Re-enter standstill, then verify an I2C glitch clears standstill.
    raw = wake.rawAngle;
    for (int i = 0; i < 90; ++i) {
        nowMs += 40;
        nowUs += 40000;
        sensors::AnemometerSample s;
        s.nowMs = nowMs;
        s.nowUs = nowUs;
        s.rawAngle = raw;
        s.i2cOk = true;
        state.step(s);
    }
    ASSERT_TRUE(state.isStandstill());

    sensors::AnemometerSample fault;
    fault.nowMs = nowMs + 40;
    fault.nowUs = nowUs + 40000;
    fault.i2cOk = false;
    state.step(fault);
    EXPECT_FALSE(state.isStandstill());
}
