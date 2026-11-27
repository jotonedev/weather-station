#include <Aht20Codec.h>
#include <gtest/gtest.h>

#include <array>
#include <cstdint>

namespace {

std::array<uint8_t, 7> makeAht20Frame(uint8_t status, uint32_t rawH, uint32_t rawT) {
    std::array<uint8_t, 7> frame{};
    frame[0] = status;
    frame[1] = static_cast<uint8_t>((rawH >> 12) & 0xFFu);
    frame[2] = static_cast<uint8_t>((rawH >> 4) & 0xFFu);
    frame[3] = static_cast<uint8_t>(((rawH & 0x0Fu) << 4) | ((rawT >> 16) & 0x0Fu));
    frame[4] = static_cast<uint8_t>((rawT >> 8) & 0xFFu);
    frame[5] = static_cast<uint8_t>(rawT & 0xFFu);
    frame[6] = aht20::aht20Crc8(frame.data(), 6);
    return frame;
}

} // namespace

TEST(Aht20CodecTest, DecodesValidFrameAndVerifiesCrc) {
    // 50% RH = 524288 (0x80000), 25.0 C = 393216 (0x60000).
    const auto frame = makeAht20Frame(0x18u, 0x80000u, 0x60000u);
    aht20::Aht20Reading out{};
    ASSERT_TRUE(aht20::decodeAht20Frame(frame.data(), out));
    EXPECT_NEAR(out.humidityPct, 50.0f, 0.01f);
    EXPECT_NEAR(out.temperatureC, 25.0f, 0.01f);
}

TEST(Aht20CodecTest, RejectsOneBitCorruption) {
    auto frame = makeAht20Frame(0x18u, 0x80000u, 0x60000u);
    frame[4] ^= 0x04u;
    aht20::Aht20Reading out{};
    EXPECT_FALSE(aht20::decodeAht20Frame(frame.data(), out));
}

TEST(Aht20CodecTest, RejectsBusyAndAllOnesFrames) {
    const auto busy = makeAht20Frame(0x88u, 0x80000u, 0x60000u);
    aht20::Aht20Reading out{};
    EXPECT_FALSE(aht20::decodeAht20Frame(busy.data(), out));

    const std::array<uint8_t, 7> allOnes = {0xFFu, 0xFFu, 0xFFu, 0xFFu, 0xFFu, 0xFFu, 0xFFu};
    EXPECT_FALSE(aht20::decodeAht20Frame(allOnes.data(), out));
}

TEST(Aht20CodecTest, RejectsOutOfSpecTemperature) {
    // rawT = 0 -> -50.0 C (< -40 C), rawT = 0xFFFFF -> ~150.0 C (> 85 C).
    const auto tooCold = makeAht20Frame(0x18u, 0x80000u, 0x00000u);
    const auto tooHot = makeAht20Frame(0x18u, 0x80000u, 0xFFFFFu);
    aht20::Aht20Reading out{};
    EXPECT_FALSE(aht20::decodeAht20Frame(tooCold.data(), out));
    EXPECT_FALSE(aht20::decodeAht20Frame(tooHot.data(), out));
}
