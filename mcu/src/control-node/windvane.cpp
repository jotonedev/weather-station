#include <Arduino.h>
#include <cmath>

#include "windvane.h"

WindVane::WindVane(TwoWire& wire, float calibrationOffset)
    : wire(wire), as5600(&wire), calibrationOffset(calibrationOffset) {}

WindVane::~WindVane() {
    stopTask();
}

void WindVane::initialize(uint8_t directionPin, uint8_t direction) {
    as5600.begin(directionPin);
    as5600.setDirection(direction);
    // Enable 3 LSB hysteresis to suppress angle jitter
    as5600.setHysteresis(AS5600_HYST_LSB3);
    // Slow filter 16x for maximum noise rejection
    as5600.setSlowFilter(AS5600_SLOW_FILT_16X);
    // Disable fast filter so noise spikes don't bypass slow filter
    as5600.setFastFilter(AS5600_FAST_FILT_NONE);

    portENTER_CRITICAL(&spinlock);
    initialized = true;
    currentAngle = 0.0f;
    rawAngleDegrees = 0.0f;
    vecX = 1.0f;
    vecY = 0.0f;
    hasFirstSample = false;
    glitchCount = 0;
    agcCount = 0;
    agcMean = 0.0;
    agcM2 = 0.0;
    portEXIT_CRITICAL(&spinlock);
}

bool WindVane::ready() {
    return as5600.isConnected() && as5600.magnetDetected() && !as5600.magnetTooStrong() &&
           !as5600.magnetTooWeak();
}

bool WindVane::isConnected() {
    return as5600.isConnected();
}

bool WindVane::magnetDetected() {
    return as5600.magnetDetected();
}

bool WindVane::magnetTooStrong() {
    return as5600.magnetTooStrong();
}

bool WindVane::magnetTooWeak() {
    return as5600.magnetTooWeak();
}

int WindVane::getLastError() {
    return as5600.lastError();
}

void WindVane::taskTrampoline(void* arg) {
    auto* self = static_cast<WindVane*>(arg);
    self->taskLoop(self->samplePeriodMs);
}

bool WindVane::startTask(uint32_t samplePeriodMs, UBaseType_t priority, BaseType_t core) {
    if (taskHandle != nullptr) {
        return true;
    }
    this->samplePeriodMs = samplePeriodMs;
    BaseType_t res = xTaskCreatePinnedToCore(taskTrampoline, "WindVaneTask", 4096, this, priority,
                                             &taskHandle, core);
    return res == pdPASS;
}

void WindVane::stopTask() {
    if (taskHandle != nullptr) {
        vTaskDelete(taskHandle);
        taskHandle = nullptr;
    }
}

void WindVane::taskLoop(uint32_t samplePeriodMs) {
    TickType_t xLastWakeTime = xTaskGetTickCount();
    const TickType_t xFrequency = pdMS_TO_TICKS(samplePeriodMs);
    while (true) {
        update();
        vTaskDelayUntil(&xLastWakeTime, xFrequency);
    }
}

void WindVane::update() {
    if (!as5600.isConnected()) {
        return;
    }

    uint16_t raw = as5600.readAngle();
    bool i2cOk = (as5600.lastError() == AS5600_OK);
    uint8_t status = as5600.readStatus();
    uint8_t agc = as5600.readAGC();

    // Check hardware status: bit 5 = Magnet Detected (0x20), bit 4 = Magnet Low/Weak (0x10)
    bool hwMagnetOk = i2cOk && (status & 0x20) && !(status & 0x10);

    portENTER_CRITICAL(&spinlock);
    lastStatus = status;
    lastAgc = agc;

    // Accumulate AGC stats on detected magnet
    if (i2cOk && (status & 0x20)) {
        agcCount++;
        double deltaAgc = static_cast<double>(agc) - agcMean;
        agcMean += deltaAgc / agcCount;
        double deltaAgc2 = static_cast<double>(agc) - agcMean;
        agcM2 += deltaAgc * deltaAgc2;
    }

    if (!hwMagnetOk) {
        glitchCount++;
        portEXIT_CRITICAL(&spinlock);
        return;
    }

    // Convert raw 12-bit to degrees [0.0, 360.0)
    float deg = static_cast<float>(raw) * AS5600_RAW_TO_DEGREES;
    rawAngleDegrees = deg;

    // Apply calibration offset
    float calibratedDeg = deg + calibrationOffset;
    while (calibratedDeg < 0.0f) {
        calibratedDeg += 360.0f;
    }
    while (calibratedDeg >= 360.0f) {
        calibratedDeg -= 360.0f;
    }

    // Circular vector smoothing with ~3 second time constant (tau = 3.0s)
    // alpha = dt / (tau + dt) adapts to 1 Hz, 2 Hz, or 4 Hz cleanly
    float dtSec = static_cast<float>(samplePeriodMs) * 0.001f;
    float alpha = std::min(std::max(dtSec / (3.0f + dtSec), 0.10f), 0.80f);

    float rad = calibratedDeg * (PI / 180.0f);
    float cosVal = cosf(rad);
    float sinVal = sinf(rad);

    if (!hasFirstSample) {
        vecX = cosVal;
        vecY = sinVal;
        hasFirstSample = true;
    } else {
        vecX = alpha * cosVal + (1.0f - alpha) * vecX;
        vecY = alpha * sinVal + (1.0f - alpha) * vecY;
    }

    float filteredRad = atan2f(vecY, vecX);
    float filteredDeg = filteredRad * (180.0f / PI);
    if (filteredDeg < 0.0f) {
        filteredDeg += 360.0f;
    }
    currentAngle = filteredDeg;

    portEXIT_CRITICAL(&spinlock);
}

float WindVane::getAngle() {
    portENTER_CRITICAL(&spinlock);
    float val = currentAngle;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

float WindVane::getRawAngle() {
    portENTER_CRITICAL(&spinlock);
    float val = rawAngleDegrees;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

void WindVane::setCalibrationOffset(float offset) {
    portENTER_CRITICAL(&spinlock);
    calibrationOffset = offset;
    hasFirstSample = false; // re-sync vector filter on offset change
    portEXIT_CRITICAL(&spinlock);
}

float WindVane::getCalibrationOffset() const {
    return calibrationOffset;
}

void WindVane::resetStats() {
    portENTER_CRITICAL(&spinlock);
    agcCount = 0;
    agcMean = 0.0;
    agcM2 = 0.0;
    portEXIT_CRITICAL(&spinlock);
}

uint8_t WindVane::getLastStatus() {
    portENTER_CRITICAL(&spinlock);
    uint8_t val = lastStatus;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

uint8_t WindVane::getLastAGC() {
    portENTER_CRITICAL(&spinlock);
    uint8_t val = lastAgc;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

uint16_t WindVane::getAGCVariance() {
    portENTER_CRITICAL(&spinlock);
    uint16_t var = (agcCount > 1) ? static_cast<uint16_t>(agcM2 / (agcCount - 1)) : 0;
    portEXIT_CRITICAL(&spinlock);
    return var;
}

uint32_t WindVane::getGlitchCount() {
    portENTER_CRITICAL(&spinlock);
    uint32_t val = glitchCount;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

float WindVane::readAngle() {
    if (taskHandle == nullptr) {
        update();
    }
    return getAngle();
}
