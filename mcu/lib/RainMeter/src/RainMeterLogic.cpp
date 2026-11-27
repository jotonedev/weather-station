#include "RainMeterLogic.h"

#include <algorithm>
#include <cmath>
#include <cstring>

namespace sensors {

int32_t RainMeterLogic::shortestAngleDiff(uint16_t current, uint16_t reference) {
    int32_t diff = static_cast<int32_t>(current) - static_cast<int32_t>(reference);
    if (diff > 2048) {
        diff -= 4096;
    } else if (diff < -2048) {
        diff += 4096;
    }
    return diff;
}

void RainMeterLogic::applyCalibration(const RainCalibration& cal) {
    rawReference_ = cal.rawReference;
    hasReference_ = true;
    restLeft_ = cal.restLeft;
    restRight_ = cal.restRight;
    calibrated_ = true;
    recomputeGeometry();
}

void RainMeterLogic::reset(uint32_t nowMs) {
    diag_.reset();
    lastHeartbeatMs_ = nowMs;
    hasInitialSample_ = false;
    currentPos_ = 0;
    prevPos_ = 0;
    pendingJumpPos_ = 0;
    pendingJumpCount_ = 0;
    filteredPos_ = 0.0f;
    stillCount_ = 0;
    currentState_ = calibrated_ ? RainMeterState::REST_LEFT : RainMeterState::CALIBRATING;
    transitStartMs_ = 0;
    lastTipMs_ = 0;
    lastMinuteTickMs_ = nowMs;
    lastHourTickMs_ = nowMs;
    glitchCount_ = 0;
    consecutiveGlitches_ = 0;
}

void RainMeterLogic::loadRollingBins(const RainRollingBins& bins) {
    if (bins.currentMinuteIdx < 60 && bins.currentHourIdx < 24) {
        std::memcpy(minuteBins_, bins.minuteBins, sizeof(minuteBins_));
        std::memcpy(hourBins_, bins.hourBins, sizeof(hourBins_));
        currentMinuteIdx_ = bins.currentMinuteIdx;
        currentHourIdx_ = bins.currentHourIdx;
        currentRainRate_ = bins.currentRainRate;
        return;
    }
    clearRollingBins();
}

void RainMeterLogic::clearRollingBins() {
    std::memset(minuteBins_, 0, sizeof(minuteBins_));
    std::memset(hourBins_, 0, sizeof(hourBins_));
    currentMinuteIdx_ = 0;
    currentHourIdx_ = 0;
    currentRainRate_ = 0.0f;
}

RainRollingBins RainMeterLogic::exportRollingBins() const {
    RainRollingBins bins;
    std::memcpy(bins.minuteBins, minuteBins_, sizeof(minuteBins_));
    std::memcpy(bins.hourBins, hourBins_, sizeof(hourBins_));
    bins.currentMinuteIdx = static_cast<uint16_t>(currentMinuteIdx_);
    bins.currentHourIdx = static_cast<uint16_t>(currentHourIdx_);
    bins.currentRainRate = currentRainRate_;
    return bins;
}

RainMeterStepResult RainMeterLogic::step(const RainMeterSample& sample) {
    RainMeterStepResult result;
    lastHeartbeatMs_ = sample.nowMs;

    if (!sample.i2cOk) {
        diag_.markI2cError();
        glitchCount_++;
        consecutiveGlitches_++;
        result.saveRtcBins = updateRollingWindows(sample.nowMs);
        return result;
    }

    if (sample.hasDiag) {
        diag_.update(true, sample.diagStatus, sample.diagAgc, sample.diagMagnitude);
    }

    if (!diag_.sensorReady) {
        glitchCount_++;
        consecutiveGlitches_++;
        result.saveRtcBins = updateRollingWindows(sample.nowMs);
        return result;
    }
    consecutiveGlitches_ = 0;

    const uint16_t angle = sample.rawAngle & 0x0FFFu;
    if (!hasReference_) {
        rawReference_ = angle;
        hasReference_ = true;
        hasInitialSample_ = true;
        currentPos_ = 0;
        prevPos_ = 0;
        filteredPos_ = 0.0f;
    } else if (calibrated_ && !hasInitialSample_) {
        currentPos_ = shortestAngleDiff(angle, rawReference_);
        prevPos_ = currentPos_;
        filteredPos_ = static_cast<float>(currentPos_);
        hasInitialSample_ = true;
        currentState_ = (std::abs(currentPos_ - restLeft_) <= std::abs(currentPos_ - restRight_))
                            ? RainMeterState::REST_LEFT
                            : RainMeterState::REST_RIGHT;
    } else {
        currentPos_ = shortestAngleDiff(angle, rawReference_);
    }

    // Kinematic outlier rejection: impossible mechanical jump (> 350 counts / sample).
    // A real fast tip confirms on the 2nd consecutive sample near the new position.
    const int32_t stepDelta = currentPos_ - prevPos_;
    if (std::abs(stepDelta) > 350) {
        if (pendingJumpCount_ > 0 && std::abs(currentPos_ - pendingJumpPos_) <= 50) {
            prevPos_ = currentPos_;
            pendingJumpCount_ = 0;
            stillCount_ = 0;
            if (std::abs(static_cast<float>(currentPos_) - filteredPos_) > 50.0f) {
                filteredPos_ = static_cast<float>(currentPos_);
            }
        } else {
            glitchCount_++;
            pendingJumpPos_ = currentPos_;
            pendingJumpCount_ = 1;
            stillCount_ = 0;
            if (updateRollingWindows(sample.nowMs)) {
                result.saveRtcBins = true;
            }
            return result;
        }
    } else {
        pendingJumpCount_ = 0;
        prevPos_ = currentPos_;
    }

    filteredPos_ = (0.35f * static_cast<float>(currentPos_)) + (0.65f * filteredPos_);

    if (std::abs(stepDelta) <= 2) {
        if (stillCount_ < 1000) {
            stillCount_++;
        }
    } else {
        stillCount_ = 0;
    }
    uint32_t stillThreshold = (sample.samplePeriodMs > 0) ? (250u / sample.samplePeriodMs) : 8u;
    stillThreshold = std::max<uint32_t>(stillThreshold, 4u);
    const bool isStill = (stillCount_ >= stillThreshold);

    if (!calibrated_) {
        currentState_ = RainMeterState::CALIBRATING;
        if (isStill) {
            if (!hasClusterA_) {
                clusterA_ = currentPos_;
                hasClusterA_ = true;
            } else if (!hasClusterB_) {
                const int32_t dist = std::abs(currentPos_ - clusterA_);
                if (dist >= 20) {
                    clusterB_ = currentPos_;
                    hasClusterB_ = true;
                    restLeft_ = std::min(clusterA_, clusterB_);
                    restRight_ = std::max(clusterA_, clusterB_);
                    calibrated_ = true;
                    recomputeGeometry();
                    result.saveCalibration = true;
                    result.calibration.rawReference = rawReference_;
                    result.calibration.restLeft = restLeft_;
                    result.calibration.restRight = restRight_;

                    currentState_ =
                        (std::abs(currentPos_ - restLeft_) < std::abs(currentPos_ - restRight_))
                            ? RainMeterState::REST_LEFT
                            : RainMeterState::REST_RIGHT;
                } else {
                    clusterA_ = static_cast<int32_t>((0.90f * static_cast<float>(clusterA_)) +
                                                     (0.10f * static_cast<float>(currentPos_)));
                }
            }
        }
    } else {
        switch (currentState_) {
        case RainMeterState::REST_LEFT:
            if (isStill &&
                static_cast<float>(std::abs(currentPos_ - restLeft_)) < travelSpan_ * 0.30f) {
                restLeft_ = static_cast<int32_t>((0.99f * static_cast<float>(restLeft_)) +
                                                 (0.01f * static_cast<float>(currentPos_)));
                recomputeGeometry();
            }
            if (filteredPos_ > (centerPoint_ + hysteresis_)) {
                currentState_ = RainMeterState::TRANSIT_TO_RIGHT;
                transitStartMs_ = sample.nowMs;
            }
            break;

        case RainMeterState::TRANSIT_TO_RIGHT:
            if (filteredPos_ >= (static_cast<float>(restRight_) - (travelSpan_ * 0.25f))) {
                recordTip(sample.nowMs);
                result.saveRtcBins = true;
                currentState_ = RainMeterState::REST_RIGHT;
                jammed_ = false;
            } else if (filteredPos_ < (centerPoint_ - hysteresis_)) {
                currentState_ = RainMeterState::REST_LEFT;
            } else if (sample.nowMs - transitStartMs_ > 2000u) {
                currentState_ = RainMeterState::JAMMED;
                jammed_ = true;
            }
            break;

        case RainMeterState::REST_RIGHT:
            if (isStill &&
                static_cast<float>(std::abs(currentPos_ - restRight_)) < travelSpan_ * 0.30f) {
                restRight_ = static_cast<int32_t>((0.99f * static_cast<float>(restRight_)) +
                                                  (0.01f * static_cast<float>(currentPos_)));
                recomputeGeometry();
            }
            if (filteredPos_ < (centerPoint_ - hysteresis_)) {
                currentState_ = RainMeterState::TRANSIT_TO_LEFT;
                transitStartMs_ = sample.nowMs;
            }
            break;

        case RainMeterState::TRANSIT_TO_LEFT:
            if (filteredPos_ <= (static_cast<float>(restLeft_) + (travelSpan_ * 0.25f))) {
                recordTip(sample.nowMs);
                result.saveRtcBins = true;
                currentState_ = RainMeterState::REST_LEFT;
                jammed_ = false;
            } else if (filteredPos_ > (centerPoint_ + hysteresis_)) {
                currentState_ = RainMeterState::REST_RIGHT;
            } else if (sample.nowMs - transitStartMs_ > 2000u) {
                currentState_ = RainMeterState::JAMMED;
                jammed_ = true;
            }
            break;

        case RainMeterState::JAMMED:
            if (isStill) {
                if (static_cast<float>(std::abs(currentPos_ - restLeft_)) < travelSpan_ * 0.30f) {
                    currentState_ = RainMeterState::REST_LEFT;
                    jammed_ = false;
                } else if (static_cast<float>(std::abs(currentPos_ - restRight_)) <
                           travelSpan_ * 0.30f) {
                    currentState_ = RainMeterState::REST_RIGHT;
                    jammed_ = false;
                }
            }
            break;

        default:
            currentState_ = RainMeterState::CALIBRATING;
            break;
        }
    }

    if (updateRollingWindows(sample.nowMs)) {
        result.saveRtcBins = true;
    }

    return result;
}

bool RainMeterLogic::isResting() const {
    return (currentState_ == RainMeterState::REST_LEFT ||
            currentState_ == RainMeterState::REST_RIGHT) &&
           (stillCount_ >= 4);
}

float RainMeterLogic::rainfall1h() const {
    float sum = 0.0f;
    for (const float bin : minuteBins_) {
        sum += bin;
    }
    return sum;
}

float RainMeterLogic::rainfall24h() const {
    float sum = 0.0f;
    for (const float bin : hourBins_) {
        sum += bin;
    }
    return sum;
}

RainData RainMeterLogic::rainData() const {
    RainData data = {};
    data.rainfall_1h = weather::encodeRainAccumulation(rainfall1h());
    data.rainfall_24h = weather::encodeRainAccumulation(rainfall24h());
    data.rainRate = weather::encodeRainRate(currentRainRate_);
    data.magnetStatus.status = diag_.lastStatus;
    data.magnetStatus.agc = diag_.lastAgc;
    data.magnetStatus.agcVariance = diag_.agcVariance();
    return data;
}

void RainMeterLogic::resetStats() {
    diag_.resetStats();
    glitchCount_ = 0;
}

void RainMeterLogic::clearCalibration() {
    calibrated_ = false;
    hasClusterA_ = false;
    hasClusterB_ = false;
    currentState_ = RainMeterState::CALIBRATING;
}

void RainMeterLogic::recordTip(uint32_t nowMs) {
    if (lastTipMs_ > 0) {
        const float dtSeconds = static_cast<float>(nowMs - lastTipMs_) / 1000.0f;
        if (dtSeconds > 0.1f && dtSeconds < 3600.0f) {
            const float rawRate = (3600.0f / dtSeconds) * mmPerTip_;
            currentRainRate_ = (0.30f * rawRate) + (0.70f * currentRainRate_);
        }
    } else {
        currentRainRate_ = mmPerTip_;
    }

    // Dynamic transit loss compensation at high rain rates (>20 mm/h).
    const float rateClamped = std::min(currentRainRate_, 150.0f);
    const float dynamicVolume = mmPerTip_ * (1.0f + 0.0015f * rateClamped);

    minuteBins_[currentMinuteIdx_] += dynamicVolume;
    hourBins_[currentHourIdx_] += dynamicVolume;

    lastTipMs_ = nowMs;
}

bool RainMeterLogic::updateRollingWindows(uint32_t nowMs) {
    bool rotated = false;

    size_t minSteps = 0;
    while (nowMs - lastMinuteTickMs_ >= 60000uL && minSteps < 60) {
        currentMinuteIdx_ = (currentMinuteIdx_ + 1) % 60;
        minuteBins_[currentMinuteIdx_] = 0.0f;
        lastMinuteTickMs_ += 60000uL;
        minSteps++;
        rotated = true;

        if (nowMs - lastTipMs_ > 600000uL) {
            currentRainRate_ *= 0.80f;
            if (currentRainRate_ < 0.05f) {
                currentRainRate_ = 0.0f;
            }
        }
    }
    if (nowMs - lastMinuteTickMs_ >= 60000uL) {
        lastMinuteTickMs_ = nowMs;
    }

    size_t hourSteps = 0;
    while (nowMs - lastHourTickMs_ >= 3600000uL && hourSteps < 24) {
        currentHourIdx_ = (currentHourIdx_ + 1) % 24;
        hourBins_[currentHourIdx_] = 0.0f;
        lastHourTickMs_ += 3600000uL;
        hourSteps++;
        rotated = true;
    }
    if (nowMs - lastHourTickMs_ >= 3600000uL) {
        lastHourTickMs_ = nowMs;
    }

    return rotated;
}

void RainMeterLogic::recomputeGeometry() {
    if (restLeft_ > restRight_) {
        std::swap(restLeft_, restRight_);
    }
    travelSpan_ = static_cast<float>(restRight_ - restLeft_);
    centerPoint_ = static_cast<float>(restLeft_ + restRight_) / 2.0f;
    hysteresis_ = std::max(travelSpan_ * 0.20f, 6.0f);
}

} // namespace sensors
