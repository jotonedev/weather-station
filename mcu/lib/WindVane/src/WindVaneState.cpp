#include "WindVaneState.h"

#include <algorithm>
#include <cmath>

namespace sensors {

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
        diag_.markI2cError();
        glitchCount_++;
        return;
    }

    if (sample.hasDiag) {
        diag_.update(true, sample.diagStatus, sample.diagAgc, sample.diagMagnitude);
    }

    if (!diag_.sensorReady) {
        glitchCount_++;
        return;
    }

    const uint16_t angle = sample.rawAngle & 0x0FFFu;
    const float deg = static_cast<float>(angle) * kRawToDegrees;
    rawAngleDegrees_ = deg;

    float calibratedDeg = std::fmod(deg + calibrationOffset_, 360.0f);
    if (calibratedDeg < 0.0f) {
        calibratedDeg += 360.0f;
    }
    if (calibratedDeg >= 360.0f) {
        calibratedDeg = 0.0f;
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
        vecX_ = (alpha * cosVal) + ((1.0f - alpha) * vecX_);
        vecY_ = (alpha * sinVal) + ((1.0f - alpha) * vecY_);
        if ((vecX_ * vecX_) + (vecY_ * vecY_) < 1e-6f) {
            vecX_ = cosVal;
            vecY_ = sinVal;
        }
    }

    const float filteredRad = std::atan2(vecY_, vecX_);
    float filteredDeg = filteredRad * (180.0f / kPi);
    if (filteredDeg < 0.0f) {
        filteredDeg += 360.0f;
    }
    if (filteredDeg >= 360.0f) {
        filteredDeg = 0.0f;
    }
    currentAngle_ = filteredDeg;
}

} // namespace sensors
