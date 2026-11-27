#ifndef BMP180_CODEC_H
#define BMP180_CODEC_H

#include <cstddef>
#include <cstdint>

namespace bmp180 {

struct Bmp180Calibration {
    int16_t ac1 = 0;
    int16_t ac2 = 0;
    int16_t ac3 = 0;
    uint16_t ac4 = 0;
    uint16_t ac5 = 0;
    uint16_t ac6 = 0;
    int16_t b1 = 0;
    int16_t b2 = 0;
    int16_t mb = 0;
    int16_t mc = 0;
    int16_t md = 0;
};

inline bool parseBmp180Calibration(const uint8_t raw[22], Bmp180Calibration& out) {
    uint16_t words[11] = {};
    for (size_t i = 0; i < 11; ++i) {
        const uint16_t w =
            (static_cast<uint16_t>(raw[i * 2u]) << 8) | static_cast<uint16_t>(raw[i * 2u + 1u]);
        if (w == 0x0000u || w == 0xFFFFu) {
            return false;
        }
        words[i] = w;
    }

    out.ac1 = static_cast<int16_t>(words[0]);
    out.ac2 = static_cast<int16_t>(words[1]);
    out.ac3 = static_cast<int16_t>(words[2]);
    out.ac4 = words[3];
    out.ac5 = words[4];
    out.ac6 = words[5];
    out.b1 = static_cast<int16_t>(words[6]);
    out.b2 = static_cast<int16_t>(words[7]);
    out.mb = static_cast<int16_t>(words[8]);
    out.mc = static_cast<int16_t>(words[9]);
    out.md = static_cast<int16_t>(words[10]);
    return true;
}

inline bool computeBmp180PressurePa(const Bmp180Calibration& cal, int32_t ut, int32_t up,
                                    uint8_t oss, int32_t& pressurePaOut) {
    if (oss > 3u) {
        return false;
    }

    int32_t x1 = ((ut - static_cast<int32_t>(cal.ac6)) * static_cast<int32_t>(cal.ac5)) >> 15;
    const int32_t denom = x1 + static_cast<int32_t>(cal.md);
    if (denom == 0) {
        return false;
    }
    int32_t x2 = (static_cast<int32_t>(cal.mc) << 11) / denom;
    const int32_t b5 = x1 + x2;

    const int32_t b6 = b5 - 4000;
    x1 = (static_cast<int32_t>(cal.b2) * ((b6 * b6) >> 12)) >> 11;
    x2 = (static_cast<int32_t>(cal.ac2) * b6) >> 11;
    int32_t x3 = x1 + x2;
    const int32_t b3 = (((static_cast<int32_t>(cal.ac1) * 4 + x3) << oss) + 2) / 4;

    x1 = (static_cast<int32_t>(cal.ac3) * b6) >> 13;
    x2 = (static_cast<int32_t>(cal.b1) * ((b6 * b6) >> 12)) >> 16;
    x3 = ((x1 + x2) + 2) >> 2;
    if (x3 + 32768 <= 0) {
        return false;
    }
    const uint32_t b4 = (static_cast<uint32_t>(cal.ac4) * static_cast<uint32_t>(x3 + 32768)) >> 15;
    if (b4 == 0u) {
        return false;
    }

    if (up <= b3) {
        return false;
    }
    const uint32_t b7 = static_cast<uint32_t>(up - b3) * (50000u >> oss);
    int32_t p = 0;
    if (b7 < 0x80000000u) {
        p = static_cast<int32_t>((b7 * 2u) / b4);
    } else {
        p = static_cast<int32_t>((b7 / b4) * 2u);
    }

    x1 = (p >> 8) * (p >> 8);
    x1 = (x1 * 3038) >> 16;
    x2 = (-7357 * p) >> 16;
    p = p + ((x1 + x2 + 3791) >> 4);

    if (p < 30000 || p > 110000) {
        return false;
    }

    pressurePaOut = p;
    return true;
}

} // namespace bmp180

#endif // BMP180_CODEC_H
