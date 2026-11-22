#ifndef SENSORS_H
#define SENSORS_H

#include <cstddef>
#include <cstdint>

#include "Weather.h"

namespace sensors {

constexpr float kRawToDegrees = 360.0f / 4096.0f;
constexpr float kPi = 3.14159265358979323846f;

struct MagnetDiagnostics {
    uint8_t lastStatus = 0;
    uint8_t lastAgc = 0;
    bool sensorReady = false;
    uint32_t agcCount = 0;
    double agcMean = 0.0;
    double agcM2 = 0.0;

    void reset();
    void resetStats();
    void update(bool i2cOk, uint8_t status, uint8_t agc);
    uint16_t agcVariance() const;
};

struct AnemometerSample {
    uint32_t nowMs = 0;
    uint32_t nowUs = 0;
    uint16_t rawAngle = 0;
    bool i2cOk = false;
    bool hasDiag = false;
    uint8_t diagStatus = 0;
    uint8_t diagAgc = 0;
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
    int64_t cumulativeCounts_ = 0;
    uint32_t consecutiveGlitches_ = 0;
    bool inDropout_ = false;
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

struct WindVaneSample {
    uint32_t nowMs = 0;
    uint16_t rawAngle = 0;
    bool i2cOk = false;
    bool hasDiag = false;
    uint8_t diagStatus = 0;
    uint8_t diagAgc = 0;
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

enum class RainMeterState : uint8_t {
    UNINITIALIZED = 0,
    CALIBRATING,
    REST_LEFT,
    TRANSIT_TO_RIGHT,
    REST_RIGHT,
    TRANSIT_TO_LEFT,
    JAMMED
};

struct RainCalibration {
    uint16_t rawReference = 0;
    int32_t restLeft = 0;
    int32_t restRight = 0;
};

struct RainRollingBins {
    float minuteBins[60] = {};
    float hourBins[24] = {};
    uint16_t currentMinuteIdx = 0;
    uint16_t currentHourIdx = 0;
    float currentRainRate = 0.0f;
};

struct RainMeterSample {
    uint32_t nowMs = 0;
    uint16_t rawAngle = 0;
    bool i2cOk = false;
    bool hasDiag = false;
    uint8_t diagStatus = 0;
    uint8_t diagAgc = 0;
    uint32_t samplePeriodMs = 30;
};

struct RainMeterStepResult {
    bool saveCalibration = false;
    RainCalibration calibration = {};
    bool saveRtcBins = false;
};

class RainMeterLogic {
  public:
    explicit RainMeterLogic(float mmPerTip) : mmPerTip_(mmPerTip) {}

    void applyCalibration(const RainCalibration& cal);
    void reset(uint32_t nowMs);
    void loadRollingBins(const RainRollingBins& bins);
    void clearRollingBins();
    RainRollingBins exportRollingBins() const;

    RainMeterStepResult step(const RainMeterSample& sample);

    bool isResting() const;
    bool ready() const { return diag_.sensorReady; }
    float rainfall1h() const;
    float rainfall24h() const;
    float rainRate() const { return currentRainRate_; }
    RainData rainData() const;
    RainMeterState state() const { return currentState_; }

    bool isCalibrated() const { return calibrated_; }
    bool isJammed() const { return jammed_; }
    bool isTiltWarning() const { return tiltWarning_; }
    float tiltDegrees() const { return tiltDegrees_; }
    float currentAngleDegrees() const { return static_cast<float>(currentPos_) * kRawToDegrees; }
    float tipTravelDegrees() const { return travelSpan_ * kRawToDegrees; }
    uint8_t lastStatus() const { return diag_.lastStatus; }
    uint8_t lastAgc() const { return diag_.lastAgc; }
    uint16_t agcVariance() const { return diag_.agcVariance(); }
    uint32_t glitchCount() const { return glitchCount_; }
    uint32_t lastHeartbeatMs() const { return lastHeartbeatMs_; }

    void resetStats();
    void clearCalibration();
    void setMmPerTip(float mm) { mmPerTip_ = mm; }
    float mmPerTip() const { return mmPerTip_; }

    static int32_t shortestAngleDiff(uint16_t current, uint16_t reference);

  private:
    MagnetDiagnostics diag_;
    uint32_t lastHeartbeatMs_ = 0;
    float mmPerTip_ = 0.0f;

    bool hasReference_ = false;
    uint16_t rawReference_ = 0;
    int32_t currentPos_ = 0;
    int32_t prevPos_ = 0;
    float filteredPos_ = 0.0f;
    uint32_t stillCount_ = 0;

    bool calibrated_ = false;
    int32_t restLeft_ = 0;
    int32_t restRight_ = 0;
    float centerPoint_ = 0.0f;
    float hysteresis_ = 6.0f;
    float travelSpan_ = 0.0f;
    bool hasClusterA_ = false;
    bool hasClusterB_ = false;
    int32_t clusterA_ = 0;
    int32_t clusterB_ = 0;

    RainMeterState currentState_ = RainMeterState::UNINITIALIZED;
    uint32_t transitStartMs_ = 0;
    bool tiltWarning_ = false;
    float tiltDegrees_ = 0.0f;
    bool jammed_ = false;

    uint32_t lastTipMs_ = 0;
    float currentRainRate_ = 0.0f;

    float minuteBins_[60] = {};
    size_t currentMinuteIdx_ = 0;
    uint32_t lastMinuteTickMs_ = 0;

    float hourBins_[24] = {};
    size_t currentHourIdx_ = 0;
    uint32_t lastHourTickMs_ = 0;

    uint32_t glitchCount_ = 0;
    uint32_t consecutiveGlitches_ = 0;

    void recordTip(uint32_t nowMs);
    bool updateRollingWindows(uint32_t nowMs);
    void recomputeGeometry();
};

} // namespace sensors

#endif // SENSORS_H
