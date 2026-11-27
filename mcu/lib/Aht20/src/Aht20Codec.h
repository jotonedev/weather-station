#ifndef AHT20_CODEC_H
#define AHT20_CODEC_H

#include <cstddef>
#include <cstdint>

namespace aht20 {

inline uint8_t aht20Crc8(const uint8_t* data, size_t len) {
    if (data == nullptr) {
        return 0xFFu;
    }
    uint8_t crc = 0xFFu;
    for (size_t i = 0; i < len; ++i) {
        crc ^= data[i];
        for (int bit = 0; bit < 8; ++bit) {
            if ((crc & 0x80u) != 0u) {
                crc = static_cast<uint8_t>((crc << 1) ^ 0x31u);
            } else {
                crc = static_cast<uint8_t>(crc << 1);
            }
        }
    }
    return crc;
}

struct Aht20Reading {
    float temperatureC = 0.0f;
    float humidityPct = 0.0f;
};

inline bool decodeAht20Frame(const uint8_t frame[7], Aht20Reading& out) {
    if ((frame[0] & 0x80u) != 0u) {
        return false;
    }
    if (aht20Crc8(frame, 6) != frame[6]) {
        return false;
    }

    const uint32_t rawH = (static_cast<uint32_t>(frame[1]) << 12) |
                          (static_cast<uint32_t>(frame[2]) << 4) |
                          (static_cast<uint32_t>(frame[3]) >> 4);
    const uint32_t rawT = (static_cast<uint32_t>(frame[3] & 0x0Fu) << 16) |
                          (static_cast<uint32_t>(frame[4]) << 8) | static_cast<uint32_t>(frame[5]);

    const float humidity = static_cast<float>(rawH) * 100.0f / 1048576.0f;
    const float temperature = (static_cast<float>(rawT) * 200.0f / 1048576.0f) - 50.0f;

    if (temperature < -40.0f || temperature > 85.0f || humidity < 0.0f || humidity > 100.0f) {
        return false;
    }

    out.temperatureC = temperature;
    out.humidityPct = humidity;
    return true;
}

} // namespace aht20

#endif // AHT20_CODEC_H
