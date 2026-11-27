#ifdef ARDUINO

#include "WindVane.h"

WindVane::WindVane(TwoWire& wire, float calibrationOffset)
    : device_(wire, 0x000Fu, false), state_(calibrationOffset) {}

WindVane::~WindVane() {
    stopTask();
}

void WindVane::initialize(uint8_t directionPin, uint8_t direction) {
    device_.initialize(directionPin, direction);
    const uint32_t nowMs = millis();
    CriticalSection lock(spinlock_);
    state_.reset(nowMs);
}

bool WindVane::ready() {
    if (taskHandle_ != nullptr) {
        CriticalSection lock(spinlock_);
        return state_.ready();
    }
    return device_.probeReady();
}

bool WindVane::isConnected() {
    return device_.isConnected();
}

bool WindVane::magnetDetected() {
    return device_.magnetDetected();
}

bool WindVane::magnetTooStrong() {
    return device_.magnetTooStrong();
}

bool WindVane::magnetTooWeak() {
    return device_.magnetTooWeak();
}

int WindVane::getLastError() {
    return device_.lastError();
}

void WindVane::taskTrampoline(void* arg) {
    auto* self = static_cast<WindVane*>(arg);
    self->taskLoop(self->samplePeriodMs_);
}

bool WindVane::startTask(uint32_t samplePeriodMs, UBaseType_t priority, BaseType_t core) {
    if (taskHandle_ != nullptr) {
        return true;
    }
    samplePeriodMs_ = samplePeriodMs;
    const BaseType_t res = xTaskCreatePinnedToCore(taskTrampoline, "WindVaneTask", 4096, this,
                                                   priority, &taskHandle_, core);
    return res == pdPASS;
}

void WindVane::stopTask() {
    if (taskHandle_ != nullptr) {
        vTaskDelete(taskHandle_);
        taskHandle_ = nullptr;
    }
}

void WindVane::taskLoop(uint32_t samplePeriodMs) {
    const uint32_t periodMs = (samplePeriodMs > 0) ? samplePeriodMs : 1000;
    TickType_t xLastWakeTime = xTaskGetTickCount();
    while (true) {
        update();
        vTaskDelayUntil(&xLastWakeTime, pdMS_TO_TICKS(periodMs));
    }
}

void WindVane::update() {
    const As5600Reading reading = device_.poll();
    sensors::WindVaneSample sample;
    sample.nowMs = reading.nowMs;
    sample.rawAngle = reading.rawAngle;
    sample.i2cOk = reading.i2cOk;
    sample.hasDiag = reading.hasDiag;
    sample.diagStatus = reading.diagStatus;
    sample.diagAgc = reading.diagAgc;
    sample.diagMagnitude = reading.diagMagnitude;
    sample.samplePeriodMs = samplePeriodMs_;

    CriticalSection lock(spinlock_);
    state_.step(sample);
}

float WindVane::getAngle() {
    CriticalSection lock(spinlock_);
    return state_.angle();
}

float WindVane::getRawAngle() {
    CriticalSection lock(spinlock_);
    return state_.rawAngle();
}

void WindVane::setCalibrationOffset(float offset) {
    CriticalSection lock(spinlock_);
    state_.setCalibrationOffset(offset);
}

float WindVane::getCalibrationOffset() const {
    CriticalSection lock(spinlock_);
    return state_.calibrationOffset();
}

void WindVane::resetStats() {
    CriticalSection lock(spinlock_);
    state_.resetStats();
}

uint8_t WindVane::getLastStatus() {
    CriticalSection lock(spinlock_);
    return state_.lastStatus();
}

uint8_t WindVane::getLastAGC() {
    CriticalSection lock(spinlock_);
    return state_.lastAgc();
}

uint16_t WindVane::getLastMagnitude() {
    CriticalSection lock(spinlock_);
    return state_.lastMagnitude();
}

uint16_t WindVane::getAGCVariance() {
    CriticalSection lock(spinlock_);
    return state_.agcVariance();
}

uint32_t WindVane::getGlitchCount() {
    CriticalSection lock(spinlock_);
    return state_.glitchCount();
}

uint32_t WindVane::getLastHeartbeatMs() {
    CriticalSection lock(spinlock_);
    return state_.lastHeartbeatMs();
}

float WindVane::readAngle() {
    if (taskHandle_ == nullptr) {
        update();
    }
    return getAngle();
}

#endif // ARDUINO
