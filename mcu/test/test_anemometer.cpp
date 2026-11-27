#include <gtest/gtest.h>

#include <cmath>
#include <cstdint>

#include "AnemometerState.h"

namespace {

constexpr uint8_t kMagnetOk = 0x20;

} // namespace

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

TEST(Sensors, AnemometerStormSpeedStandstillGustActualDtCoastAndImmediateRecovery) {
    sensors::AnemometerState state;
    state.reset(0);

    // Storm wind: ramp up and hold ~1400 RPM (~955 counts per 10 ms) without glitching.
    uint32_t nowMs = 0;
    uint32_t nowUs = 0;
    uint16_t raw = 0;
    for (int i = 0; i < 80; ++i) {
        sensors::AnemometerSample s;
        s.nowMs = nowMs;
        s.nowUs = nowUs;
        s.rawAngle = raw;
        s.i2cOk = true;
        s.hasDiag = (i == 0);
        s.diagStatus = kMagnetOk;
        s.diagAgc = 80;
        s.diagMagnitude = 1800;
        state.step(s);
        nowMs += 10;
        nowUs += 10000;
        const uint16_t stepCounts = static_cast<uint16_t>(std::min(955, 200 + (i * 20)));
        raw = static_cast<uint16_t>((raw + stepCounts) & 0x0FFFu);
    }
    EXPECT_EQ(state.glitchCount(), 0u);
    EXPECT_NEAR(state.rpm(), 1400.0f, 15.0f);
    EXPECT_EQ(state.lastMagnitude(), 1800u);

    // Transient I2C error marks unready; next healthy sample with hasDiag recovers immediately.
    sensors::AnemometerSample fault;
    fault.nowMs = nowMs;
    fault.nowUs = nowUs;
    fault.i2cOk = false;
    state.step(fault);
    EXPECT_FALSE(state.ready());
    EXPECT_EQ(state.lastStatus(), sensors::MagnetDiagnostics::kStatusI2cError);

    nowMs += 10;
    nowUs += 10000;
    sensors::AnemometerSample heal;
    heal.nowMs = nowMs;
    heal.nowUs = nowUs;
    heal.rawAngle = raw;
    heal.i2cOk = true;
    heal.hasDiag = true;
    heal.diagStatus = kMagnetOk;
    heal.diagAgc = 82;
    heal.diagMagnitude = 1810;
    state.step(heal);
    EXPECT_TRUE(state.ready());
    EXPECT_EQ(state.lastMagnitude(), 1810u);

    // Actual-dt coasting: 20 ms glitch step projects roughly double a 10 ms step.
    const int64_t countsBeforeCoast = state.cumulativeCounts();
    const float rpmBeforeCoast = state.rpm();
    nowMs += 20;
    nowUs += 20000;
    sensors::AnemometerSample coast20ms;
    coast20ms.nowMs = nowMs;
    coast20ms.nowUs = nowUs;
    coast20ms.i2cOk = false;
    state.step(coast20ms);
    const auto expected20msStep =
        static_cast<int64_t>((rpmBeforeCoast * std::pow(0.999f, 2.0f) * 4096.0f / 60.0f) * 0.020f);
    EXPECT_NEAR(static_cast<double>(state.cumulativeCounts() - countsBeforeCoast),
                static_cast<double>(expected20msStep), 5.0);

    // 40 ms standstill wake gust (600 counts in 40 ms) is accepted without glitching.
    state.reset(0);
    sensors::AnemometerSample s0;
    s0.nowMs = 40;
    s0.nowUs = 40000;
    s0.rawAngle = 100;
    s0.i2cOk = true;
    s0.hasDiag = true;
    s0.diagStatus = kMagnetOk;
    s0.diagAgc = 80;
    state.step(s0);

    sensors::AnemometerSample wakeGust;
    wakeGust.nowMs = 80;
    wakeGust.nowUs = 80000;
    wakeGust.rawAngle = 700;
    wakeGust.i2cOk = true;
    state.step(wakeGust);
    EXPECT_EQ(state.glitchCount(), 0u);
    EXPECT_EQ(state.cumulativeCounts(), 600);
}
