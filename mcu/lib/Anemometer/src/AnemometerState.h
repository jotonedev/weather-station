#ifndef ANEMOMETER_STATE_H
#define ANEMOMETER_STATE_H

#include <MagnetDiagnostics.h>
#include <cstddef>
#include <cstdint>

namespace sensors {

struct AnemometerSample {
    uint32_t nowMs = 0;
    uint32_t nowUs = 0;
    uint16_t rawAngle = 0;
    bool i2cOk = false;
    bool hasDiag = false;
    uint8_t diagStatus = 0;
    uint8_t diagAgc = 0;
    uint16_t diagMagnitude = 0;
};

class AnemometerState {
  public:
    void reset(uint32_t nowMs);
    void step(const AnemometerSample& sample);

    float rpm() const { return filteredRpm_; }
    float windSpeed() const;
    void setSpeedFactor(float factor) { speedFactor_ = factor; }
    float speedFactor() const { return speedFactor_; }
    void resetStats() { diag_.resetStats(); }

    bool ready() const { return diag_.sensorReady; }
    bool isStandstill() const { return isStandstill_; }
    uint8_t lastStatus() const { return diag_.lastStatus; }
    uint8_t lastAgc() const { return diag_.lastAgc; }
    uint16_t lastMagnitude() const { return diag_.lastMagnitude; }
    uint16_t agcVariance() const { return diag_.agcVariance(); }
    uint32_t glitchCount() const { return glitchCount_; }
    int64_t cumulativeCounts() const { return cumulativeCounts_; }
    uint32_t lastHeartbeatMs() const { return lastHeartbeatMs_; }

  private:
    static constexpr size_t kWindowSize = 15;
    struct Sample {
        int64_t counts;
        uint32_t timeUs;
    };

    MagnetDiagnostics diag_;
    uint32_t lastHeartbeatMs_ = 0;
    int32_t lastRawAngle_ = -1;
    uint32_t lastValidMicros_ = 0;
    uint32_t lastStepMicros_ = 0;
    int64_t cumulativeCounts_ = 0;
    uint32_t consecutiveGlitches_ = 0;
    bool inDropout_ = false;
    bool dropoutExpired_ = false;
    uint32_t dropoutStartMicros_ = 0;
    int8_t rotationDirection_ = 1;

    Sample history_[kWindowSize] = {};
    size_t historyIdx_ = 0;
    size_t historyCount_ = 0;

    float filteredRpm_ = 0.0f;
    float speedFactor_ = 0.259f;
    float windOffset_ = 0.0f;
    uint32_t glitchCount_ = 0;

    static constexpr uint32_t kStandstillHoldMs = 3000u;
    bool standstillTracking_ = false;
    uint32_t standstillSinceMs_ = 0;
    bool isStandstill_ = false;
};

} // namespace sensors

#endif // ANEMOMETER_STATE_H
