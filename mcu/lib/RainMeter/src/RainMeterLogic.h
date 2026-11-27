#ifndef RAIN_METER_LOGIC_H
#define RAIN_METER_LOGIC_H

#include <MagnetDiagnostics.h>
#include <Weather.h>
#include <cstddef>
#include <cstdint>

namespace sensors {

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
    uint16_t diagMagnitude = 0;
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
    float currentAngleDegrees() const { return static_cast<float>(currentPos_) * kRawToDegrees; }
    float tipTravelDegrees() const { return travelSpan_ * kRawToDegrees; }
    uint8_t lastStatus() const { return diag_.lastStatus; }
    uint8_t lastAgc() const { return diag_.lastAgc; }
    uint16_t lastMagnitude() const { return diag_.lastMagnitude; }
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
    bool hasInitialSample_ = false;
    uint16_t rawReference_ = 0;
    int32_t currentPos_ = 0;
    int32_t prevPos_ = 0;
    int32_t pendingJumpPos_ = 0;
    uint8_t pendingJumpCount_ = 0;
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

#endif // RAIN_METER_LOGIC_H
