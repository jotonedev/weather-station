#ifndef MAGNET_DIAGNOSTICS_H
#define MAGNET_DIAGNOSTICS_H

#include <cstdint>

namespace sensors {

constexpr float kRawToDegrees = 360.0f / 4096.0f;
constexpr float kPi = 3.14159265358979323846f;

struct MagnetDiagnostics {
    static constexpr uint8_t kStatusMagnetHigh = 0x08u;
    static constexpr uint8_t kStatusMagnetLow = 0x10u;
    static constexpr uint8_t kStatusMagnetDetect = 0x20u;
    static constexpr uint8_t kStatusReservedMask = 0xC7u;
    static constexpr uint8_t kStatusI2cError = 0x80u;
    static constexpr uint8_t kStatusInvalid = 0xFFu;

    uint8_t lastStatus = kStatusInvalid;
    uint8_t lastAgc = 0;
    uint16_t lastMagnitude = 0;
    bool sensorReady = false;
    uint32_t agcCount = 0;
    double agcMean = 0.0;
    double agcM2 = 0.0;

    static bool isValidStatus(uint8_t status);
    static bool isHealthyStatus(uint8_t status);

    void reset();
    void resetStats();
    void markI2cError();
    void update(bool i2cOk, uint8_t status, uint8_t agc, uint16_t magnitude = 0);
    uint16_t agcVariance() const;
};

} // namespace sensors

#endif // MAGNET_DIAGNOSTICS_H
