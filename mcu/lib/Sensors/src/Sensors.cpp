#include "Sensors.h"

#include <algorithm>
#include <cmath>
#include <cstring>

namespace sensors {

void MagnetDiagnostics::reset() {
    lastStatus = 0;
    lastAgc = 0;
    sensorReady = false;
    resetStats();
}

void MagnetDiagnostics::resetStats() {
    agcCount = 0;
    agcMean = 0.0;
    agcM2 = 0.0;
}

void MagnetDiagnostics::update(bool i2cOk, uint8_t status, uint8_t agc) {
    // AS5600 status bits: 0x20 = magnet detected, 0x10 = magnet weak, 0x08 = magnet strong.
    const bool hwMagnetOk = i2cOk && ((status & 0x20) != 0) && ((status & 0x10) == 0);
    const bool fullReady = hwMagnetOk && ((status & 0x08) == 0);

    lastStatus = status;
    lastAgc = agc;
    sensorReady = fullReady;
    if (i2cOk && ((status & 0x20) != 0)) {
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

void AnemometerState::reset(uint32_t nowMs) {
    diag_.reset();
    lastHeartbeatMs_ = nowMs;
    lastRawAngle_ = -1;
    lastValidMicros_ = 0;
    cumulativeCounts_ = 0;
    consecutiveGlitches_ = 0;
    inDropout_ = false;
    dropoutStartMicros_ = 0;
    rotationDirection_ = 1;
    historyIdx_ = 0;
    historyCount_ = 0;
    filteredRpm_ = 0.0f;
    glitchCount_ = 0;
    standstillTracking_ = false;
    standstillSinceMs_ = 0;
    isStandstill_ = false;
}

void AnemometerState::step(const AnemometerSample& sample) {
    if (sample.hasDiag) {
        diag_.update(sample.i2cOk, sample.diagStatus, sample.diagAgc);
    }

    lastHeartbeatMs_ = sample.nowMs;
    bool isGlitch = !sample.i2cOk || !diag_.sensorReady;
    int32_t delta = 0;

    // Kinematic outlier rejection: even if status did not flag ML, reject impossible jumps.
    if (!isGlitch && lastRawAngle_ >= 0) {
        delta = static_cast<int32_t>(sample.rawAngle) - lastRawAngle_;
        if (delta > 2048) {
            delta -= 4096;
        } else if (delta < -2048) {
            delta += 4096;
        }

        const uint32_t dtUs = (lastValidMicros_ > 0) ? (sample.nowUs - lastValidMicros_) : 10000u;
        const float expectedStep =
            (filteredRpm_ * 4096.0f / 60.0f) * (static_cast<float>(dtUs) * 1e-6f);
        const float stepDiff = std::abs(static_cast<float>(std::abs(delta)) - expectedStep);

        if (std::abs(delta) > 350 || stepDiff > 250.0f) {
            isGlitch = true;
        }
    }

    if (isGlitch) {
        glitchCount_++;
        consecutiveGlitches_++;

        if (!inDropout_) {
            inDropout_ = true;
            dropoutStartMicros_ =
                (lastValidMicros_ > 0) ? lastValidMicros_ : (sample.nowUs - 10000u);
        }

        const uint32_t elapsedGapUs = sample.nowUs - dropoutStartMicros_;

        // Inertial coasting: bridge blind spots up to 1.5 seconds (down to 10 RPM).
        if (elapsedGapUs < 1500000u && filteredRpm_ > 0.5f && lastRawAngle_ >= 0) {
            filteredRpm_ *= 0.999f;
            const int32_t projectedStep =
                rotationDirection_ *
                static_cast<int32_t>((filteredRpm_ * 4096.0f / 60.0f) * 0.010f);
            cumulativeCounts_ += projectedStep;

            history_[historyIdx_] = {cumulativeCounts_, sample.nowUs};
            historyIdx_ = (historyIdx_ + 1) % kWindowSize;
            if (historyCount_ < kWindowSize) {
                historyCount_++;
            }
        } else if (elapsedGapUs >= 1500000u) {
            filteredRpm_ *= 0.85f;
            if (filteredRpm_ < 0.2f) {
                filteredRpm_ = 0.0f;
            }
            lastRawAngle_ = -1;
        }
        standstillTracking_ = false;
        isStandstill_ = false;
        return;
    }

    consecutiveGlitches_ = 0;

    // Re-anchor cleanly after a dropout without introducing a velocity spike.
    if (inDropout_) {
        inDropout_ = false;
        lastRawAngle_ = sample.rawAngle;
        lastValidMicros_ = sample.nowUs;
        standstillTracking_ = false;
        isStandstill_ = false;
        return;
    }

    lastValidMicros_ = sample.nowUs;

    if (lastRawAngle_ < 0) {
        lastRawAngle_ = sample.rawAngle;
        cumulativeCounts_ = 0;
        historyIdx_ = 0;
        historyCount_ = 1;
        history_[0] = {cumulativeCounts_, sample.nowUs};
        standstillTracking_ = false;
        isStandstill_ = false;
        return;
    }

    if (delta > 2) {
        rotationDirection_ = 1;
    } else if (delta < -2) {
        rotationDirection_ = -1;
    }

    cumulativeCounts_ += delta;
    lastRawAngle_ = sample.rawAngle;

    history_[historyIdx_] = {cumulativeCounts_, sample.nowUs};
    historyIdx_ = (historyIdx_ + 1) % kWindowSize;
    if (historyCount_ < kWindowSize) {
        historyCount_++;
    }

    const size_t oldestIdx = (historyCount_ < kWindowSize) ? 0 : historyIdx_;
    const int64_t dCounts = cumulativeCounts_ - history_[oldestIdx].counts;
    const uint32_t dtUs = sample.nowUs - history_[oldestIdx].timeUs;

    float rawRpm = 0.0f;
    if (dtUs > 0) {
        if (std::abs(dCounts) <= 3) {
            rawRpm = 0.0f;
        } else {
            rawRpm = (static_cast<float>(std::abs(dCounts)) * 60.0f * 1e6f) /
                     (static_cast<float>(dtUs) * 4096.0f);
        }
    }

    if (rawRpm == 0.0f && filteredRpm_ < 0.5f) {
        filteredRpm_ = 0.0f;
    } else {
        filteredRpm_ = 0.20f * rawRpm + 0.80f * filteredRpm_;
        if (filteredRpm_ < 0.1f) {
            filteredRpm_ = 0.0f;
        }
    }

    if (filteredRpm_ == 0.0f && std::abs(delta) <= 3) {
        if (!standstillTracking_) {
            standstillTracking_ = true;
            standstillSinceMs_ = sample.nowMs;
        }
        isStandstill_ = (sample.nowMs - standstillSinceMs_) >= kStandstillHoldMs;
    } else {
        standstillTracking_ = false;
        isStandstill_ = false;
    }
}

float AnemometerState::windSpeed() const {
    if (filteredRpm_ <= 0.0f) {
        return 0.0f;
    }
    return filteredRpm_ * speedFactor_ + windOffset_;
}

void WindVaneState::reset(uint32_t nowMs) {
    diag_.reset();
    lastHeartbeatMs_ = nowMs;
    currentAngle_ = 0.0f;
    rawAngleDegrees_ = 0.0f;
    vecX_ = 1.0f;
    vecY_ = 0.0f;
    hasFirstSample_ = false;
    glitchCount_ = 0;
}

void WindVaneState::setCalibrationOffset(float offset) {
    calibrationOffset_ = offset;
    hasFirstSample_ = false;
}

void WindVaneState::step(const WindVaneSample& sample) {
    lastHeartbeatMs_ = sample.nowMs;
    if (!sample.i2cOk) {
        diag_.sensorReady = false;
        glitchCount_++;
        return;
    }

    if (sample.hasDiag) {
        diag_.update(true, sample.diagStatus, sample.diagAgc);
    }

    if (!diag_.sensorReady) {
        glitchCount_++;
        return;
    }

    const float deg = static_cast<float>(sample.rawAngle) * kRawToDegrees;
    rawAngleDegrees_ = deg;

    float calibratedDeg = deg + calibrationOffset_;
    while (calibratedDeg < 0.0f) {
        calibratedDeg += 360.0f;
    }
    while (calibratedDeg >= 360.0f) {
        calibratedDeg -= 360.0f;
    }

    const float dtSec = static_cast<float>(sample.samplePeriodMs) * 0.001f;
    const float alpha = std::min(std::max(dtSec / (3.0f + dtSec), 0.10f), 0.80f);

    const float rad = calibratedDeg * (kPi / 180.0f);
    const float cosVal = std::cos(rad);
    const float sinVal = std::sin(rad);

    if (!hasFirstSample_) {
        vecX_ = cosVal;
        vecY_ = sinVal;
        hasFirstSample_ = true;
    } else {
        vecX_ = alpha * cosVal + (1.0f - alpha) * vecX_;
        vecY_ = alpha * sinVal + (1.0f - alpha) * vecY_;
    }

    const float filteredRad = std::atan2(vecY_, vecX_);
    float filteredDeg = filteredRad * (180.0f / kPi);
    if (filteredDeg < 0.0f) {
        filteredDeg += 360.0f;
    }
    currentAngle_ = filteredDeg;
}

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
    currentPos_ = 0;
    prevPos_ = 0;
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
        diag_.sensorReady = false;
        glitchCount_++;
        consecutiveGlitches_++;
        result.saveRtcBins = updateRollingWindows(sample.nowMs);
        return result;
    }

    if (sample.hasDiag) {
        diag_.update(true, sample.diagStatus, sample.diagAgc);
    }

    if (!diag_.sensorReady) {
        glitchCount_++;
        consecutiveGlitches_++;
        result.saveRtcBins = updateRollingWindows(sample.nowMs);
        return result;
    }
    consecutiveGlitches_ = 0;

    if (!hasReference_) {
        rawReference_ = sample.rawAngle;
        hasReference_ = true;
        currentPos_ = 0;
        prevPos_ = 0;
        filteredPos_ = 0.0f;
    } else {
        currentPos_ = shortestAngleDiff(sample.rawAngle, rawReference_);
    }

    // Kinematic outlier rejection: impossible mechanical jump (> 350 counts / sample).
    const int32_t stepDelta = currentPos_ - prevPos_;
    if (std::abs(stepDelta) > 350) {
        glitchCount_++;
        return result;
    }
    prevPos_ = currentPos_;

    filteredPos_ = 0.35f * static_cast<float>(currentPos_) + 0.65f * filteredPos_;

    if (std::abs(stepDelta) <= 2) {
        if (stillCount_ < 1000) {
            stillCount_++;
        }
    } else {
        stillCount_ = 0;
    }
    uint32_t stillThreshold = (sample.samplePeriodMs > 0) ? (250u / sample.samplePeriodMs) : 8u;
    if (stillThreshold < 4u) {
        stillThreshold = 4u;
    }
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
                    clusterA_ = static_cast<int32_t>(0.90f * static_cast<float>(clusterA_) +
                                                     0.10f * static_cast<float>(currentPos_));
                }
            }
        }
    } else {
        switch (currentState_) {
        case RainMeterState::REST_LEFT:
            if (isStill &&
                static_cast<float>(std::abs(currentPos_ - restLeft_)) < travelSpan_ * 0.30f) {
                restLeft_ = static_cast<int32_t>(0.99f * static_cast<float>(restLeft_) +
                                                 0.01f * static_cast<float>(currentPos_));
                recomputeGeometry();
            }
            if (filteredPos_ > (centerPoint_ + hysteresis_)) {
                currentState_ = RainMeterState::TRANSIT_TO_RIGHT;
                transitStartMs_ = sample.nowMs;
            }
            break;

        case RainMeterState::TRANSIT_TO_RIGHT:
            if (filteredPos_ >= (static_cast<float>(restRight_) - travelSpan_ * 0.25f)) {
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
                restRight_ = static_cast<int32_t>(0.99f * static_cast<float>(restRight_) +
                                                  0.01f * static_cast<float>(currentPos_));
                recomputeGeometry();
            }
            if (filteredPos_ < (centerPoint_ - hysteresis_)) {
                currentState_ = RainMeterState::TRANSIT_TO_LEFT;
                transitStartMs_ = sample.nowMs;
            }
            break;

        case RainMeterState::TRANSIT_TO_LEFT:
            if (filteredPos_ <= (static_cast<float>(restLeft_) + travelSpan_ * 0.25f)) {
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
    const float r1h = rainfall1h();
    const float r24h = rainfall24h();

    data.rainfall_1h = static_cast<uint16_t>(r1h * 100.0f);
    data.rainfall_24h = static_cast<uint16_t>(r24h * 100.0f);
    data.rainRate = static_cast<uint16_t>(currentRainRate_ * 10.0f);
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
            currentRainRate_ = 0.30f * rawRate + 0.70f * currentRainRate_;
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

    if (travelSpan_ > 15.0f) {
        const float leftTravel = centerPoint_ - static_cast<float>(restLeft_);
        const float rightTravel = static_cast<float>(restRight_) - centerPoint_;
        const float asymmetry = std::abs(rightTravel - leftTravel) / travelSpan_;

        tiltWarning_ = (asymmetry > 0.15f);
        tiltDegrees_ = asymmetry * (travelSpan_ * kRawToDegrees);
    } else {
        tiltWarning_ = false;
        tiltDegrees_ = 0.0f;
    }
}

} // namespace sensors
