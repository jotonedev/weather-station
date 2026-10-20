#include <Arduino.h>
#include <algorithm>
#include <cmath>
#include <cstring>
#include <esp_attr.h>
#include <esp_rom_crc.h>

#include "rainmeter.h"

namespace {

constexpr uint32_t kRainRtcMagic = 0x5241494Eu; // "RAIN"

// Retained in RTC Slow SRAM across watchdog resets and software reboots without wearing flash.
struct RainRtcState {
    uint32_t magic;
    float minuteBins[60];
    float hourBins[24];
    uint16_t currentMinuteIdx;
    uint16_t currentHourIdx;
    float currentRainRate;
    uint32_t crc;
};

RTC_NOINIT_ATTR RainRtcState s_rtcRainState;

uint32_t computeRainRtcCrc(const RainRtcState& state) {
    return esp_rom_crc32_le(0, reinterpret_cast<const uint8_t*>(&state),
                            offsetof(RainRtcState, crc));
}

} // namespace

RainMeter::RainMeter(TwoWire& wire, float mmPerTip)
    : wire(wire), as5600(&wire), mmPerTip(mmPerTip) {}

RainMeter::~RainMeter() {
    stopTask();
}

void RainMeter::initialize(uint8_t directionPin, uint8_t direction) {
    as5600.begin(directionPin);
    as5600.setDirection(direction);
    // 3 LSB hysteresis to suppress angle jitter at rest
    as5600.setHysteresis(AS5600_HYST_LSB3);
    // Slow filter 16x for maximum noise rejection
    as5600.setSlowFilter(AS5600_SLOW_FILT_16X);
    // Disable fast filter so noise spikes do not bypass slow filter
    as5600.setFastFilter(AS5600_FAST_FILT_NONE);

    loadNvsCalibration();

    uint32_t nowMs = millis();
    portENTER_CRITICAL(&spinlock);
    initialized = true;
    sensorReady = false;
    lastHeartbeatMs = nowMs;
    currentPos = 0;
    prevPos = 0;
    filteredPos = 0.0f;
    stillCount = 0;
    currentState = calibrated ? RainMeterState::REST_LEFT : RainMeterState::CALIBRATING;
    transitStartMs = 0;
    lastTipMs = 0;
    prevTipMs = 0;
    lastMinuteTickMs = nowMs;
    lastHourTickMs = nowMs;
    glitchCount = 0;
    consecutiveGlitches = 0;
    agcCount = 0;
    agcMean = 0.0;
    agcM2 = 0.0;
    loadRtcRollingBins();
    portEXIT_CRITICAL(&spinlock);
}

bool RainMeter::ready() {
    if (taskHandle != nullptr) {
        portENTER_CRITICAL(&spinlock);
        bool r = sensorReady;
        portEXIT_CRITICAL(&spinlock);
        return r;
    }
    return as5600.isConnected() && as5600.magnetDetected() && !as5600.magnetTooStrong() &&
           !as5600.magnetTooWeak();
}

bool RainMeter::isConnected() {
    return as5600.isConnected();
}

bool RainMeter::magnetDetected() {
    return as5600.magnetDetected();
}

bool RainMeter::magnetTooStrong() {
    return as5600.magnetTooStrong();
}

bool RainMeter::magnetTooWeak() {
    return as5600.magnetTooWeak();
}

int RainMeter::getLastError() {
    return as5600.lastError();
}

void RainMeter::taskTrampoline(void* arg) {
    auto* self = static_cast<RainMeter*>(arg);
    self->taskLoop(self->samplePeriodMs);
}

bool RainMeter::startTask(uint32_t samplePeriodMs, UBaseType_t priority, BaseType_t core) {
    if (taskHandle != nullptr) {
        return true;
    }
    this->samplePeriodMs = samplePeriodMs;
    BaseType_t res = xTaskCreatePinnedToCore(taskTrampoline, "RainMeterTask", 4096, this, priority,
                                             &taskHandle, core);
    return res == pdPASS;
}

void RainMeter::stopTask() {
    if (taskHandle != nullptr) {
        vTaskDelete(taskHandle);
        taskHandle = nullptr;
    }
}

void RainMeter::taskLoop(uint32_t samplePeriodMs) {
    TickType_t xLastWakeTime = xTaskGetTickCount();
    const TickType_t xFrequency = pdMS_TO_TICKS(samplePeriodMs);
    while (true) {
        update();
        vTaskDelayUntil(&xLastWakeTime, xFrequency);
    }
}

void RainMeter::update() {
    uint32_t nowMs = millis();
    if (!as5600.isConnected()) {
        portENTER_CRITICAL(&spinlock);
        lastHeartbeatMs = nowMs;
        sensorReady = false;
        updateRollingWindows(nowMs);
        portEXIT_CRITICAL(&spinlock);
        return;
    }

    uint16_t raw = as5600.readAngle();
    bool i2cOk = (as5600.lastError() == AS5600_OK);
    uint8_t status = as5600.readStatus();
    uint8_t agc = as5600.readAGC();

    // Check hardware status: bit 5 = Magnet Detected (0x20), bit 4 = Magnet Low/Weak (0x10),
    // bit 3 = Magnet High/Strong (0x08)
    bool hwMagnetOk = i2cOk && (status & 0x20) && !(status & 0x10);
    bool fullReady = hwMagnetOk && !(status & 0x08);

    portENTER_CRITICAL(&spinlock);
    lastHeartbeatMs = nowMs;
    sensorReady = fullReady;
    lastStatus = status;
    lastAgc = agc;

    // Accumulate AGC statistics on detected magnet (Welford's algorithm)
    if (i2cOk && (status & 0x20)) {
        agcCount++;
        double deltaAgc = static_cast<double>(agc) - agcMean;
        agcMean += deltaAgc / agcCount;
        double deltaAgc2 = static_cast<double>(agc) - agcMean;
        agcM2 += deltaAgc * deltaAgc2;
    }

    if (!hwMagnetOk) {
        glitchCount++;
        consecutiveGlitches++;
        portEXIT_CRITICAL(&spinlock);
        return;
    }
    consecutiveGlitches = 0;

    // Establish reference angle on first valid reading
    if (!hasReference) {
        rawReference = raw;
        hasReference = true;
        currentPos = 0;
        prevPos = 0;
        filteredPos = 0.0f;
    } else {
        // Continuous unwrapped angular position [-2048, +2047] relative to rawReference
        currentPos = shortestAngleDiff(raw, rawReference);
    }

    // Kinematic outlier rejection: impossible mechanical jump (> 350 counts / 20ms)
    int32_t step = currentPos - prevPos;
    if (std::abs(step) > 350) {
        glitchCount++;
        portEXIT_CRITICAL(&spinlock);
        return;
    }
    prevPos = currentPos;

    // Exponential filter for smooth thresholding
    filteredPos = 0.35f * static_cast<float>(currentPos) + 0.65f * filteredPos;

    // Stationarity detection: motionless for >= 200 ms (10 samples @ 20ms)
    if (std::abs(step) <= 2) {
        if (stillCount < 1000) {
            stillCount++;
        }
    } else {
        stillCount = 0;
    }
    uint32_t stillThreshold = (samplePeriodMs > 0) ? (250 / samplePeriodMs) : 8;
    if (stillThreshold < 4) {
        stillThreshold = 4;
    }
    bool isStill = (stillCount >= stillThreshold);

    // =========================================================================
    // Auto-Calibration & State Machine
    // =========================================================================
    if (!calibrated) {
        currentState = RainMeterState::CALIBRATING;
        if (isStill) {
            if (!hasClusterA) {
                clusterA = currentPos;
                hasClusterA = true;
            } else if (!hasClusterB) {
                int32_t dist = std::abs(currentPos - clusterA);
                // At least 20 counts (~1.75 degrees) separation to constitute distinct resting
                // positions
                if (dist >= 20) {
                    clusterB = currentPos;
                    hasClusterB = true;
                    restLeft = std::min(clusterA, clusterB);
                    restRight = std::max(clusterA, clusterB);
                    calibrated = true;
                    recomputeGeometry();
                    saveNvsCalibration();

                    currentState =
                        (std::abs(currentPos - restLeft) < std::abs(currentPos - restRight))
                            ? RainMeterState::REST_LEFT
                            : RainMeterState::REST_RIGHT;
                } else {
                    // Refine cluster A
                    clusterA = static_cast<int32_t>(0.90f * clusterA + 0.10f * currentPos);
                }
            }
        }
    } else {
        // Active calibrated operation
        switch (currentState) {
        case RainMeterState::REST_LEFT:
            // Adaptive resting stop tracking while stationary
            if (isStill && std::abs(currentPos - restLeft) < travelSpan * 0.30f) {
                restLeft = static_cast<int32_t>(0.99f * restLeft + 0.01f * currentPos);
                recomputeGeometry();
            }
            // Check if tipped past center crest + hysteresis
            if (filteredPos > (centerPoint + hysteresis)) {
                currentState = RainMeterState::TRANSIT_TO_RIGHT;
                transitStartMs = nowMs;
            }
            break;

        case RainMeterState::TRANSIT_TO_RIGHT:
            // Check for successful landing in Right basin
            if (filteredPos >= (static_cast<float>(restRight) - travelSpan * 0.25f)) {
                recordTip(nowMs);
                currentState = RainMeterState::REST_RIGHT;
                jammed = false;
            } else if (filteredPos < (centerPoint - hysteresis)) {
                // Wind shake rejection: rocked past crest but fell back without reaching right stop
                currentState = RainMeterState::REST_LEFT;
            } else if (nowMs - transitStartMs > 2000) {
                // Stuck mid-air for > 2 seconds
                currentState = RainMeterState::JAMMED;
                jammed = true;
            }
            break;

        case RainMeterState::REST_RIGHT:
            // Adaptive resting stop tracking while stationary
            if (isStill && std::abs(currentPos - restRight) < travelSpan * 0.30f) {
                restRight = static_cast<int32_t>(0.99f * restRight + 0.01f * currentPos);
                recomputeGeometry();
            }
            // Check if tipped past center crest - hysteresis
            if (filteredPos < (centerPoint - hysteresis)) {
                currentState = RainMeterState::TRANSIT_TO_LEFT;
                transitStartMs = nowMs;
            }
            break;

        case RainMeterState::TRANSIT_TO_LEFT:
            // Check for successful landing in Left basin
            if (filteredPos <= (static_cast<float>(restLeft) + travelSpan * 0.25f)) {
                recordTip(nowMs);
                currentState = RainMeterState::REST_LEFT;
                jammed = false;
            } else if (filteredPos > (centerPoint + hysteresis)) {
                // Wind shake rejection: rocked past crest but fell back to right stop
                currentState = RainMeterState::REST_RIGHT;
            } else if (nowMs - transitStartMs > 2000) {
                // Stuck mid-air for > 2 seconds
                currentState = RainMeterState::JAMMED;
                jammed = true;
            }
            break;

        case RainMeterState::JAMMED:
            // Check if blockage cleared and bucket reached either resting basin
            if (isStill) {
                if (std::abs(currentPos - restLeft) < travelSpan * 0.30f) {
                    currentState = RainMeterState::REST_LEFT;
                    jammed = false;
                } else if (std::abs(currentPos - restRight) < travelSpan * 0.30f) {
                    currentState = RainMeterState::REST_RIGHT;
                    jammed = false;
                }
            }
            break;

        default:
            currentState = RainMeterState::CALIBRATING;
            break;
        }
    }

    // Advance rolling accumulation windows
    updateRollingWindows(nowMs);

    portEXIT_CRITICAL(&spinlock);
}

void RainMeter::recordTip(uint32_t nowMs) {
    float rawRate = 0.0f;
    if (lastTipMs > 0) {
        float dtSeconds = (nowMs - lastTipMs) / 1000.0f;
        if (dtSeconds > 0.1f && dtSeconds < 3600.0f) {
            rawRate = (3600.0f / dtSeconds) * mmPerTip;
            currentRainRate = 0.30f * rawRate + 0.70f * currentRainRate;
        }
    } else {
        currentRainRate = mmPerTip;
    }

    // Dynamic siphon & transit loss compensation at high rain rates:
    // When water is pouring heavily (>20 mm/h), water enters the bucket during the 50-80ms tip
    // transit.
    float rateClamped = std::min(currentRainRate, 150.0f);
    float dynamicVolume = mmPerTip * (1.0f + 0.0015f * rateClamped);

    minuteBins[currentMinuteIdx] += dynamicVolume;
    hourBins[currentHourIdx] += dynamicVolume;

    prevTipMs = lastTipMs;
    lastTipMs = nowMs;
    saveRtcRollingBins();
}

void RainMeter::updateRollingWindows(uint32_t nowMs) {
    bool rotated = false;

    // 1-minute bucket rotation (drift-free step increments, bounded to 60 steps)
    size_t minSteps = 0;
    while (nowMs - lastMinuteTickMs >= 60000uL && minSteps < 60) {
        currentMinuteIdx = (currentMinuteIdx + 1) % 60;
        minuteBins[currentMinuteIdx] = 0.0f;
        lastMinuteTickMs += 60000uL;
        minSteps++;
        rotated = true;

        // Exponential decay of instantaneous rain rate when dry for > 10 min
        if (nowMs - lastTipMs > 600000uL) {
            currentRainRate *= 0.80f;
            if (currentRainRate < 0.05f) {
                currentRainRate = 0.0f;
            }
        }
    }
    if (nowMs - lastMinuteTickMs >= 60000uL) {
        lastMinuteTickMs = nowMs;
    }

    // 1-hour bucket rotation (drift-free step increments, bounded to 24 steps)
    size_t hourSteps = 0;
    while (nowMs - lastHourTickMs >= 3600000uL && hourSteps < 24) {
        currentHourIdx = (currentHourIdx + 1) % 24;
        hourBins[currentHourIdx] = 0.0f;
        lastHourTickMs += 3600000uL;
        hourSteps++;
        rotated = true;
    }
    if (nowMs - lastHourTickMs >= 3600000uL) {
        lastHourTickMs = nowMs;
    }

    if (rotated) {
        saveRtcRollingBins();
    }
}

void RainMeter::recomputeGeometry() {
    if (restLeft > restRight) {
        std::swap(restLeft, restRight);
    }
    travelSpan = static_cast<float>(restRight - restLeft);
    centerPoint = static_cast<float>(restLeft + restRight) / 2.0f;
    hysteresis = std::max(travelSpan * 0.20f, 6.0f);

    // Station tilt & level diagnostic: check for asymmetry between left and right travel
    if (travelSpan > 15.0f) {
        float leftTravel = centerPoint - static_cast<float>(restLeft);
        float rightTravel = static_cast<float>(restRight) - centerPoint;
        float asymmetry = std::abs(rightTravel - leftTravel) / travelSpan;

        tiltWarning = (asymmetry > 0.15f);
        tiltDegrees = asymmetry * (travelSpan * 360.0f / 4096.0f);
    } else {
        tiltWarning = false;
        tiltDegrees = 0.0f;
    }
}

float RainMeter::getRainfall1h() {
    portENTER_CRITICAL(&spinlock);
    float sum = 0.0f;
    for (size_t i = 0; i < 60; i++) {
        sum += minuteBins[i];
    }
    portEXIT_CRITICAL(&spinlock);
    return sum;
}

float RainMeter::getRainfall24h() {
    portENTER_CRITICAL(&spinlock);
    float sum = 0.0f;
    for (size_t i = 0; i < 24; i++) {
        sum += hourBins[i];
    }
    portEXIT_CRITICAL(&spinlock);
    return sum;
}

float RainMeter::getRainRate() {
    portENTER_CRITICAL(&spinlock);
    float val = currentRainRate;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

RainData RainMeter::getRainData() {
    portENTER_CRITICAL(&spinlock);
    RainData data = {};
    float r1h = 0.0f;
    for (size_t i = 0; i < 60; i++) {
        r1h += minuteBins[i];
    }
    float r24h = 0.0f;
    for (size_t i = 0; i < 24; i++) {
        r24h += hourBins[i];
    }

    data.rainfall_1h = static_cast<uint16_t>(r1h * 100.0f);         // 0.01 mm resolution
    data.rainfall_24h = static_cast<uint16_t>(r24h * 100.0f);       // 0.01 mm resolution
    data.rainRate = static_cast<uint16_t>(currentRainRate * 10.0f); // 0.1 mm/h
    data.magnetStatus.status = lastStatus;
    data.magnetStatus.agc = lastAgc;
    data.magnetStatus.agcVariance =
        (agcCount > 1) ? static_cast<uint16_t>(agcM2 / (agcCount - 1)) : 0;
    portEXIT_CRITICAL(&spinlock);
    return data;
}

RainMeterState RainMeter::getState() {
    portENTER_CRITICAL(&spinlock);
    RainMeterState val = currentState;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

bool RainMeter::isCalibrated() {
    portENTER_CRITICAL(&spinlock);
    bool val = calibrated;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

bool RainMeter::isJammed() {
    portENTER_CRITICAL(&spinlock);
    bool val = jammed;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

bool RainMeter::isTiltWarning() {
    portENTER_CRITICAL(&spinlock);
    bool val = tiltWarning;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

float RainMeter::getTiltDegrees() {
    portENTER_CRITICAL(&spinlock);
    float val = tiltDegrees;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

float RainMeter::getCurrentAngleDegrees() {
    portENTER_CRITICAL(&spinlock);
    float val = static_cast<float>(currentPos) * (360.0f / 4096.0f);
    portEXIT_CRITICAL(&spinlock);
    return val;
}

float RainMeter::getTipTravelDegrees() {
    portENTER_CRITICAL(&spinlock);
    float val = travelSpan * (360.0f / 4096.0f);
    portEXIT_CRITICAL(&spinlock);
    return val;
}

uint8_t RainMeter::getLastStatus() {
    portENTER_CRITICAL(&spinlock);
    uint8_t val = lastStatus;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

uint8_t RainMeter::getLastAGC() {
    portENTER_CRITICAL(&spinlock);
    uint8_t val = lastAgc;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

uint16_t RainMeter::getAGCVariance() {
    portENTER_CRITICAL(&spinlock);
    uint16_t val = (agcCount > 1) ? static_cast<uint16_t>(agcM2 / (agcCount - 1)) : 0;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

uint32_t RainMeter::getGlitchCount() {
    portENTER_CRITICAL(&spinlock);
    uint32_t val = glitchCount;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

uint32_t RainMeter::getLastHeartbeatMs() {
    portENTER_CRITICAL(&spinlock);
    uint32_t val = lastHeartbeatMs;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

void RainMeter::resetStats() {
    portENTER_CRITICAL(&spinlock);
    agcCount = 0;
    agcMean = 0.0;
    agcM2 = 0.0;
    glitchCount = 0;
    portEXIT_CRITICAL(&spinlock);
}

void RainMeter::clearCalibration() {
    portENTER_CRITICAL(&spinlock);
    calibrated = false;
    hasClusterA = false;
    hasClusterB = false;
    currentState = RainMeterState::CALIBRATING;
    preferences.begin("rainmeter", false);
    preferences.putBool("cal", false);
    preferences.end();
    portEXIT_CRITICAL(&spinlock);
}

void RainMeter::setMmPerTip(float mm) {
    portENTER_CRITICAL(&spinlock);
    mmPerTip = mm;
    portEXIT_CRITICAL(&spinlock);
}

float RainMeter::getMmPerTip() const {
    return mmPerTip;
}

void RainMeter::loadNvsCalibration() {
    preferences.begin("rainmeter", false);
    bool isCal = preferences.getBool("cal", false);
    if (isCal) {
        rawReference = preferences.getUShort("rawRef", 0);
        hasReference = true;
        restLeft = preferences.getInt("restL", 0);
        restRight = preferences.getInt("restR", 0);
        calibrated = true;
        recomputeGeometry();
    }
    preferences.end();
}

void RainMeter::saveNvsCalibration() {
    preferences.begin("rainmeter", false);
    preferences.putBool("cal", true);
    preferences.putUShort("rawRef", rawReference);
    preferences.putInt("restL", restLeft);
    preferences.putInt("restR", restRight);
    preferences.end();
}

void RainMeter::loadRtcRollingBins() {
    if (s_rtcRainState.magic == kRainRtcMagic &&
        s_rtcRainState.crc == computeRainRtcCrc(s_rtcRainState) &&
        s_rtcRainState.currentMinuteIdx < 60 && s_rtcRainState.currentHourIdx < 24) {
        std::memcpy(minuteBins, s_rtcRainState.minuteBins, sizeof(minuteBins));
        std::memcpy(hourBins, s_rtcRainState.hourBins, sizeof(hourBins));
        currentMinuteIdx = s_rtcRainState.currentMinuteIdx;
        currentHourIdx = s_rtcRainState.currentHourIdx;
        currentRainRate = s_rtcRainState.currentRainRate;
        return;
    }

    std::memset(minuteBins, 0, sizeof(minuteBins));
    std::memset(hourBins, 0, sizeof(hourBins));
    currentMinuteIdx = 0;
    currentHourIdx = 0;
    currentRainRate = 0.0f;
    saveRtcRollingBins();
}

void RainMeter::saveRtcRollingBins() {
    s_rtcRainState.magic = kRainRtcMagic;
    std::memcpy(s_rtcRainState.minuteBins, minuteBins, sizeof(minuteBins));
    std::memcpy(s_rtcRainState.hourBins, hourBins, sizeof(hourBins));
    s_rtcRainState.currentMinuteIdx = static_cast<uint16_t>(currentMinuteIdx);
    s_rtcRainState.currentHourIdx = static_cast<uint16_t>(currentHourIdx);
    s_rtcRainState.currentRainRate = currentRainRate;
    s_rtcRainState.crc = computeRainRtcCrc(s_rtcRainState);
}
