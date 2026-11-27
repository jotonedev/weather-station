#include "AnemometerState.h"

#include <algorithm>
#include <cmath>

namespace sensors {

void AnemometerState::reset(uint32_t nowMs) {
    diag_.reset();
    lastHeartbeatMs_ = nowMs;
    lastRawAngle_ = -1;
    lastValidMicros_ = 0;
    lastStepMicros_ = 0;
    cumulativeCounts_ = 0;
    consecutiveGlitches_ = 0;
    inDropout_ = false;
    dropoutExpired_ = false;
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
    const uint16_t angle = sample.rawAngle & 0x0FFFu;
    if (!sample.i2cOk) {
        diag_.markI2cError();
    } else if (sample.hasDiag) {
        diag_.update(true, sample.diagStatus, sample.diagAgc, sample.diagMagnitude);
    }

    lastHeartbeatMs_ = sample.nowMs;
    bool isGlitch = !sample.i2cOk || !diag_.sensorReady;
    int32_t delta = 0;

    // Kinematic outlier rejection: even if status did not flag ML, reject impossible jumps.
    if (!isGlitch && lastRawAngle_ >= 0) {
        delta = static_cast<int32_t>(angle) - lastRawAngle_;
        if (delta > 2048) {
            delta -= 4096;
        } else if (delta < -2048) {
            delta += 4096;
        }

        const uint32_t dtUs = (lastValidMicros_ > 0) ? (sample.nowUs - lastValidMicros_) : 10000u;
        const float dtSec = static_cast<float>(dtUs) * 1e-6f;
        const float expectedStep = (filteredRpm_ * 4096.0f / 60.0f) * dtSec;
        const float stepDiff = std::abs(static_cast<float>(std::abs(delta)) - expectedStep);
        const float maxAbsStep = std::min(1800.0f, std::max(350.0f, 125000.0f * dtSec));
        const float maxStepDiff = std::min(1500.0f, std::max(350.0f, 35000.0f * dtSec));

        if (static_cast<float>(std::abs(delta)) > maxAbsStep || stepDiff > maxStepDiff) {
            isGlitch = true;
        }
    }

    if (isGlitch) {
        glitchCount_++;
        consecutiveGlitches_++;

        if (!inDropout_) {
            inDropout_ = true;
            dropoutExpired_ = false;
            dropoutStartMicros_ =
                (lastValidMicros_ > 0) ? lastValidMicros_ : (sample.nowUs - 10000u);
        }

        const uint32_t elapsedGapUs = sample.nowUs - dropoutStartMicros_;
        uint32_t coastDtUs = (lastStepMicros_ > 0) ? (sample.nowUs - lastStepMicros_) : 10000u;
        coastDtUs = std::min(coastDtUs, 200000u);
        const float coastDtSec = static_cast<float>(coastDtUs) * 1e-6f;

        // Inertial coasting: bridge blind spots up to 1.5 seconds (down to 10 RPM).
        if (!dropoutExpired_ && elapsedGapUs < 1500000u && filteredRpm_ > 0.5f &&
            lastRawAngle_ >= 0) {
            filteredRpm_ *= std::pow(0.999f, coastDtSec / 0.010f);
            const int32_t projectedStep =
                rotationDirection_ *
                static_cast<int32_t>((filteredRpm_ * 4096.0f / 60.0f) * coastDtSec);
            cumulativeCounts_ += projectedStep;

            history_[historyIdx_] = {cumulativeCounts_, sample.nowUs};
            historyIdx_ = (historyIdx_ + 1) % kWindowSize;
            if (historyCount_ < kWindowSize) {
                historyCount_++;
            }
        } else if (dropoutExpired_ || elapsedGapUs >= 1500000u) {
            dropoutExpired_ = true;
            filteredRpm_ *= 0.85f;
            if (filteredRpm_ < 0.2f) {
                filteredRpm_ = 0.0f;
            }
            lastRawAngle_ = -1;
        }
        standstillTracking_ = false;
        isStandstill_ = false;
        lastStepMicros_ = sample.nowUs;
        return;
    }

    consecutiveGlitches_ = 0;

    // Re-anchor cleanly after a dropout without introducing a velocity spike.
    if (inDropout_) {
        inDropout_ = false;
        dropoutExpired_ = false;
        lastRawAngle_ = angle;
        lastValidMicros_ = sample.nowUs;
        standstillTracking_ = false;
        isStandstill_ = false;
        lastStepMicros_ = sample.nowUs;
        return;
    }

    lastValidMicros_ = sample.nowUs;

    if (lastRawAngle_ < 0) {
        lastRawAngle_ = angle;
        cumulativeCounts_ = 0;
        historyIdx_ = 0;
        historyCount_ = 1;
        history_[0] = {cumulativeCounts_, sample.nowUs};
        standstillTracking_ = false;
        isStandstill_ = false;
        lastStepMicros_ = sample.nowUs;
        return;
    }

    if (delta > 2) {
        rotationDirection_ = 1;
    } else if (delta < -2) {
        rotationDirection_ = -1;
    }

    cumulativeCounts_ += delta;
    lastRawAngle_ = angle;

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
        filteredRpm_ = (0.20f * rawRpm) + (0.80f * filteredRpm_);
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
    lastStepMicros_ = sample.nowUs;
}

float AnemometerState::windSpeed() const {
    if (filteredRpm_ <= 0.0f) {
        return 0.0f;
    }
    return (filteredRpm_ * speedFactor_) + windOffset_;
}

} // namespace sensors
