#ifndef WIND_VANE_STATE_H
#define WIND_VANE_STATE_H

#include <MagnetDiagnostics.h>
#include <cstdint>

namespace sensors {

struct WindVaneSample {
    uint32_t nowMs = 0;
    uint16_t rawAngle = 0;
    bool i2cOk = false;
    bool hasDiag = false;
    uint8_t diagStatus = 0;
    uint8_t diagAgc = 0;
    uint16_t diagMagnitude = 0;
    uint32_t samplePeriodMs = 1000;
};

class WindVaneState {
  public:
    explicit WindVaneState(float calibrationOffset = 0.0f)
        : calibrationOffset_(calibrationOffset) {}

    void reset(uint32_t nowMs);
    void step(const WindVaneSample& sample);

    void setCalibrationOffset(float offset);
    void resetStats() { diag_.resetStats(); }

    bool ready() const { return diag_.sensorReady; }
    float angle() const { return currentAngle_; }
    float rawAngle() const { return rawAngleDegrees_; }
    float calibrationOffset() const { return calibrationOffset_; }
    uint8_t lastStatus() const { return diag_.lastStatus; }
    uint8_t lastAgc() const { return diag_.lastAgc; }
    uint16_t lastMagnitude() const { return diag_.lastMagnitude; }
    uint16_t agcVariance() const { return diag_.agcVariance(); }
    uint32_t glitchCount() const { return glitchCount_; }
    uint32_t lastHeartbeatMs() const { return lastHeartbeatMs_; }

  private:
    MagnetDiagnostics diag_;
    float calibrationOffset_ = 0.0f;
    uint32_t lastHeartbeatMs_ = 0;
    float currentAngle_ = 0.0f;
    float rawAngleDegrees_ = 0.0f;
    float vecX_ = 1.0f;
    float vecY_ = 0.0f;
    bool hasFirstSample_ = false;
    uint32_t glitchCount_ = 0;
};

} // namespace sensors

#endif // WIND_VANE_STATE_H
