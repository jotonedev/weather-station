#include <Bmp180Codec.h>
#include <gtest/gtest.h>

#include <array>
#include <cstdint>

namespace {

void writeBeU16(uint8_t* dst, uint16_t val) {
    dst[0] = static_cast<uint8_t>((val >> 8) & 0xFFu);
    dst[1] = static_cast<uint8_t>(val & 0xFFu);
}

bmp180::Bmp180Calibration boschRefCal() {
    bmp180::Bmp180Calibration cal{};
    cal.ac1 = 408;
    cal.ac2 = -72;
    cal.ac3 = -14383;
    cal.ac4 = 32741;
    cal.ac5 = 32757;
    cal.ac6 = 23153;
    cal.b1 = 6190;
    cal.b2 = 4;
    cal.mb = -32768;
    cal.mc = -8711;
    cal.md = 2868;
    return cal;
}

} // namespace

TEST(Bmp180CodecTest, RejectsZeroOrAllOnesCalibrationWords) {
    const bmp180::Bmp180Calibration ref = boschRefCal();
    std::array<uint8_t, 22> raw{};
    writeBeU16(raw.data(), static_cast<uint16_t>(ref.ac1));
    writeBeU16(&raw[2], static_cast<uint16_t>(ref.ac2));
    writeBeU16(&raw[4], static_cast<uint16_t>(ref.ac3));
    writeBeU16(&raw[6], ref.ac4);
    writeBeU16(&raw[8], ref.ac5);
    writeBeU16(&raw[10], ref.ac6);
    writeBeU16(&raw[12], static_cast<uint16_t>(ref.b1));
    writeBeU16(&raw[14], static_cast<uint16_t>(ref.b2));
    writeBeU16(&raw[16], static_cast<uint16_t>(ref.mb));
    writeBeU16(&raw[18], static_cast<uint16_t>(ref.mc));
    writeBeU16(&raw[20], static_cast<uint16_t>(ref.md));

    bmp180::Bmp180Calibration parsed{};
    ASSERT_TRUE(bmp180::parseBmp180Calibration(raw.data(), parsed));
    EXPECT_EQ(parsed.ac1, ref.ac1);
    EXPECT_EQ(parsed.mb, ref.mb);

    auto badZero = raw;
    badZero[6] = 0x00u;
    badZero[7] = 0x00u;
    EXPECT_FALSE(bmp180::parseBmp180Calibration(badZero.data(), parsed));

    auto badOnes = raw;
    badOnes[20] = 0xFFu;
    badOnes[21] = 0xFFu;
    EXPECT_FALSE(bmp180::parseBmp180Calibration(badOnes.data(), parsed));
}

TEST(Bmp180CodecTest, BoschDatasheetReferenceVectorOss0) {
    const bmp180::Bmp180Calibration cal = boschRefCal();
    int32_t pressurePa = 0;
    ASSERT_TRUE(bmp180::computeBmp180PressurePa(cal, 27898, 23843, 0u, pressurePa));
    EXPECT_EQ(pressurePa, 69964);
}

TEST(Bmp180CodecTest, SupportsOss3AndRejectsInvalidOss) {
    const bmp180::Bmp180Calibration cal = boschRefCal();
    int32_t pressurePa = 0;
    // At oss=3, up is shifted left by 3 bits relative to oss=0 (23843 << 3 = 190744).
    ASSERT_TRUE(bmp180::computeBmp180PressurePa(cal, 27898, 23843 << 3, 3u, pressurePa));
    EXPECT_NEAR(pressurePa, 69964, 5);

    EXPECT_FALSE(bmp180::computeBmp180PressurePa(cal, 27898, 23843, 4u, pressurePa));
}

TEST(Bmp180CodecTest, GuardsAgainstDivisionByZero) {
    bmp180::Bmp180Calibration cal = boschRefCal();
    int32_t pressurePa = 0;

    // Force x1 + md == 0 by setting ut == ac6 (so x1 == 0) and md == 0.
    cal.md = 0;
    EXPECT_FALSE(bmp180::computeBmp180PressurePa(cal, cal.ac6, 23843, 0u, pressurePa));

    // Force b4 == 0 by setting ac4 == 0.
    cal = boschRefCal();
    cal.ac4 = 0u;
    EXPECT_FALSE(bmp180::computeBmp180PressurePa(cal, 27898, 23843, 0u, pressurePa));
}
