#include "MagnetDiagnostics.h"

namespace sensors {

bool MagnetDiagnostics::isValidStatus(uint8_t status) {
    return ((status & kStatusReservedMask) == 0u) &&
           ((status & (kStatusMagnetLow | kStatusMagnetHigh)) !=
            (kStatusMagnetLow | kStatusMagnetHigh));
}

bool MagnetDiagnostics::isHealthyStatus(uint8_t status) {
    return isValidStatus(status) && ((status & (kStatusMagnetDetect | kStatusMagnetLow |
                                                kStatusMagnetHigh)) == kStatusMagnetDetect);
}

void MagnetDiagnostics::reset() {
    lastStatus = kStatusInvalid;
    lastAgc = 0;
    lastMagnitude = 0;
    sensorReady = false;
    resetStats();
}

void MagnetDiagnostics::resetStats() {
    agcCount = 0;
    agcMean = 0.0;
    agcM2 = 0.0;
}

void MagnetDiagnostics::markI2cError() {
    lastStatus = kStatusI2cError;
    lastAgc = 0;
    lastMagnitude = 0;
    sensorReady = false;
}

void MagnetDiagnostics::update(bool i2cOk, uint8_t status, uint8_t agc, uint16_t magnitude) {
    if (!i2cOk || !isValidStatus(status)) {
        markI2cError();
        return;
    }

    lastStatus = status;
    lastAgc = agc;
    lastMagnitude = magnitude;
    sensorReady = isHealthyStatus(status);
    if ((status & kStatusMagnetDetect) != 0u) {
        agcCount++;
        const double deltaAgc = static_cast<double>(agc) - agcMean;
        agcMean += deltaAgc / agcCount;
        const double deltaAgc2 = static_cast<double>(agc) - agcMean;
        agcM2 += deltaAgc * deltaAgc2;
    }
}

uint16_t MagnetDiagnostics::agcVariance() const {
    return (agcCount > 1) ? static_cast<uint16_t>(agcM2 / (agcCount - 1)) : 0;
}

} // namespace sensors
