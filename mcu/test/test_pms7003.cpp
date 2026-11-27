#include <PmsDutyCycle.h>
#include <gtest/gtest.h>

#include <array>
#include <cstdint>

namespace {

void writeBeU16(uint8_t* dst, uint16_t val) {
    dst[0] = static_cast<uint8_t>((val >> 8) & 0xFFu);
    dst[1] = static_cast<uint8_t>(val & 0xFFu);
}

std::array<uint8_t, 32> makePmsDataFrame(uint16_t pm1Sp, uint16_t pm25Sp, uint16_t pm10Sp,
                                         uint16_t pm1Ae, uint16_t pm25Ae, uint16_t pm10Ae,
                                         uint8_t version, uint8_t errorCode) {
    std::array<uint8_t, 32> frame{};
    frame[0] = 0x42u;
    frame[1] = 0x4Du;
    writeBeU16(&frame[2], 28u);
    writeBeU16(&frame[4], pm1Sp);
    writeBeU16(&frame[6], pm25Sp);
    writeBeU16(&frame[8], pm10Sp);
    writeBeU16(&frame[10], pm1Ae);
    writeBeU16(&frame[12], pm25Ae);
    writeBeU16(&frame[14], pm10Ae);
    frame[28] = version;
    frame[29] = errorCode;
    uint16_t sum = 0;
    for (size_t i = 0; i < 30; ++i) {
        sum = static_cast<uint16_t>(sum + frame[i]);
    }
    writeBeU16(&frame[30], sum);
    return frame;
}

std::array<uint8_t, 8> makePmsAckFrame(uint8_t cmd, uint8_t status) {
    std::array<uint8_t, 8> frame{};
    frame[0] = 0x42u;
    frame[1] = 0x4Du;
    writeBeU16(&frame[2], 4u);
    frame[4] = cmd;
    frame[5] = status;
    uint16_t sum = 0;
    for (size_t i = 0; i < 6; ++i) {
        sum = static_cast<uint16_t>(sum + frame[i]);
    }
    writeBeU16(&frame[6], sum);
    return frame;
}

} // namespace

TEST(PmsCodecTest, ParsesValidFrameWithDistinctSpAndAeValues) {
    const auto raw = makePmsDataFrame(11u, 22u, 33u, 14u, 27u, 41u, 0x91u, 0u);
    pms7003::PmsStreamParser parser;
    pms7003::PmsFrame frame{};

    bool gotFrame = false;
    for (size_t i = 0; i < raw.size(); ++i) {
        const bool ready = parser.feed(raw[i], frame);
        if (i + 1 < raw.size()) {
            EXPECT_FALSE(ready);
        } else {
            gotFrame = ready;
        }
    }

    ASSERT_TRUE(gotFrame);
    EXPECT_EQ(frame.pm1_0_sp, 11u);
    EXPECT_EQ(frame.pm2_5_sp, 22u);
    EXPECT_EQ(frame.pm10_0_sp, 33u);
    EXPECT_EQ(frame.pm1_0_ae, 14u);
    EXPECT_EQ(frame.pm2_5_ae, 27u);
    EXPECT_EQ(frame.pm10_0_ae, 41u);
    EXPECT_EQ(frame.version, 0x91u);
    EXPECT_EQ(frame.errorCode, 0u);
}

TEST(PmsCodecTest, RejectsChecksumMismatchAndNonZeroErrorCode) {
    pms7003::PmsStreamParser parser;
    pms7003::PmsFrame frame{};

    auto badCsum = makePmsDataFrame(1u, 2u, 3u, 4u, 5u, 6u, 0x80u, 0u);
    badCsum[31] ^= 0x01u;
    for (const uint8_t b : badCsum) {
        EXPECT_FALSE(parser.feed(b, frame));
    }
    EXPECT_EQ(parser.checksumErrorCount(), 1u);
    EXPECT_EQ(parser.sensorErrorCount(), 0u);

    const auto errFrame = makePmsDataFrame(1u, 2u, 3u, 4u, 5u, 6u, 0x80u, 0x02u);
    for (const uint8_t b : errFrame) {
        EXPECT_FALSE(parser.feed(b, frame));
    }
    EXPECT_EQ(parser.sensorErrorCount(), 1u);
    EXPECT_EQ(parser.lastErrorCode(), 0x02u);
}

TEST(PmsCodecTest, ResyncsAfterGarbageAndRepeatedStartByte) {
    const auto valid = makePmsDataFrame(5u, 12u, 18u, 6u, 15u, 21u, 0x77u, 0u);
    pms7003::PmsStreamParser parser;
    pms7003::PmsFrame frame{};

    const std::array<uint8_t, 6> prefix = {0x00u, 0xFFu, 0x42u, 0x11u, 0x42u, 0x42u};
    for (const uint8_t b : prefix) {
        EXPECT_FALSE(parser.feed(b, frame));
    }

    // Feed starting from byte 1 (0x4D) to exercise 0x42 0x42 0x4D resync.
    bool gotFrame = false;
    for (size_t i = 1; i < valid.size(); ++i) {
        gotFrame = parser.feed(valid[i], frame);
    }
    ASSERT_TRUE(gotFrame);
    EXPECT_EQ(frame.pm2_5_ae, 15u);
    EXPECT_EQ(frame.version, 0x77u);
}

TEST(PmsCodecTest, ResetDiscardsPartialFrame) {
    const auto valid = makePmsDataFrame(7u, 8u, 9u, 10u, 11u, 12u, 0x55u, 0u);
    pms7003::PmsStreamParser parser;
    pms7003::PmsFrame frame{};

    for (size_t i = 0; i < 10; ++i) {
        EXPECT_FALSE(parser.feed(valid[i], frame));
    }
    parser.reset();

    bool gotFrame = false;
    for (const uint8_t b : valid) {
        gotFrame = parser.feed(b, frame);
    }
    ASSERT_TRUE(gotFrame);
    EXPECT_EQ(frame.pm1_0_ae, 10u);
}

TEST(PmsCodecTest, SkipsAckFrameAndParsesFollowingDataFrame) {
    const auto ack = makePmsAckFrame(0xE1u, 0x01u);
    const auto data = makePmsDataFrame(3u, 6u, 9u, 4u, 7u, 10u, 0x66u, 0u);
    pms7003::PmsStreamParser parser;
    pms7003::PmsFrame frame{};

    for (const uint8_t b : ack) {
        EXPECT_FALSE(parser.feed(b, frame));
    }
    EXPECT_EQ(parser.ackCount(), 1u);

    bool gotFrame = false;
    for (const uint8_t b : data) {
        gotFrame = parser.feed(b, frame);
    }
    ASSERT_TRUE(gotFrame);
    EXPECT_EQ(frame.pm10_0_ae, 10u);
    EXPECT_EQ(parser.ackCount(), 1u);
    EXPECT_EQ(parser.checksumErrorCount(), 0u);
}

TEST(PmsDutyCycleTest, WarmupValidFrameSleepAnd15MinCycle) {
    pms7003::PmsDutyCycle dc;
    dc.begin(0u);
    EXPECT_EQ(dc.state(), pms7003::PmsState::WarmingUp);

    // Before 30s warmup completes, stays in WarmingUp.
    auto actions = dc.poll(29999u, nullptr, 0u);
    EXPECT_EQ(dc.state(), pms7003::PmsState::WarmingUp);
    EXPECT_FALSE(actions.flushRx);

    // At 30s, flushes RX and transitions to Reading.
    actions = dc.poll(30000u, nullptr, 0u);
    EXPECT_EQ(dc.state(), pms7003::PmsState::Reading);
    EXPECT_TRUE(actions.flushRx);

    // Valid frame transitions to Sleeping with pending report.
    const auto frame = makePmsDataFrame(1u, 2u, 3u, 10u, 20u, 30u, 0x91u, 0u);
    actions = dc.poll(30500u, frame.data(), frame.size());
    EXPECT_EQ(dc.state(), pms7003::PmsState::Sleeping);
    EXPECT_TRUE(actions.sendSleep);
    EXPECT_TRUE(actions.flushRx);
    EXPECT_FALSE(dc.timedOut());
    EXPECT_FALSE(dc.hadError());
    EXPECT_TRUE(dc.hasPendingReport(30500u));
    EXPECT_EQ(dc.latestFrame().pm2_5_ae, 20u);

    dc.markReportDelivered();
    EXPECT_FALSE(dc.hasPendingReport(30500u));

    // At 15 minutes (900000 ms), wakes up and enters WarmingUp.
    actions = dc.poll(899999u, nullptr, 0u);
    EXPECT_EQ(dc.state(), pms7003::PmsState::Sleeping);
    actions = dc.poll(900000u, nullptr, 0u);
    EXPECT_EQ(dc.state(), pms7003::PmsState::WarmingUp);
    EXPECT_TRUE(actions.sendWakeUp);
    EXPECT_TRUE(actions.flushRx);
}

TEST(PmsDutyCycleTest, ReadTimeoutAndByte29ErrorFlag) {
    pms7003::PmsDutyCycle dc;
    dc.begin(0u);
    (void)dc.poll(30000u, nullptr, 0u);
    EXPECT_EQ(dc.state(), pms7003::PmsState::Reading);

    const auto errFrame = makePmsDataFrame(1u, 2u, 3u, 4u, 5u, 6u, 0x80u, 0x03u);
    auto actions = dc.poll(31000u, errFrame.data(), errFrame.size());
    EXPECT_EQ(dc.state(), pms7003::PmsState::Reading);
    EXPECT_FALSE(actions.warnReadTimeout);

    actions = dc.poll(40000u, nullptr, 0u);
    EXPECT_EQ(dc.state(), pms7003::PmsState::Sleeping);
    EXPECT_TRUE(actions.warnReadTimeout);
    EXPECT_TRUE(actions.sendSleep);
    EXPECT_TRUE(actions.flushRx);
    EXPECT_TRUE(dc.timedOut());
    EXPECT_TRUE(dc.hadError());
    EXPECT_FALSE(dc.hasPendingReport(40000u));
}

TEST(PmsDutyCycleTest, StrayBytesDuringSleepRateLimitSleepRetry) {
    pms7003::PmsDutyCycle dc;
    dc.begin(0u);
    (void)dc.poll(30000u, nullptr, 0u);
    const auto frame = makePmsDataFrame(1u, 2u, 3u, 4u, 5u, 6u, 0x80u, 0u);
    (void)dc.poll(30500u, frame.data(), frame.size());
    EXPECT_EQ(dc.state(), pms7003::PmsState::Sleeping);

    const uint8_t stray = 0x42u;
    // Within 5s of entering sleep (lastSleepRetryMs == 30500), should not resend sleep.
    auto actions = dc.poll(32000u, &stray, 1u);
    EXPECT_FALSE(actions.sendSleep);
    EXPECT_FALSE(actions.warnActiveInSleep);

    // At >= 5s (35500), should warn and resend sleep.
    actions = dc.poll(35500u, &stray, 1u);
    EXPECT_TRUE(actions.sendSleep);
    EXPECT_TRUE(actions.warnActiveInSleep);
}

TEST(PmsDutyCycleTest, PendingReportExpiresAfterFiveMinutes) {
    pms7003::PmsDutyCycle dc;
    dc.begin(0u);
    (void)dc.poll(30000u, nullptr, 0u);
    const auto frame = makePmsDataFrame(1u, 2u, 3u, 4u, 5u, 6u, 0x80u, 0u);
    (void)dc.poll(30500u, frame.data(), frame.size());

    EXPECT_TRUE(dc.hasPendingReport(30500u + 299999u));
    EXPECT_FALSE(dc.hasPendingReport(30500u + 300000u));
}
