#include <Arduino.h>

#include "windvane.h"

WindVane::WindVane(TwoWire& wire, float calibrationOffset)
    : wire(wire), as5600(&wire), state(calibrationOffset) {}

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
    // 100 ms internal polling (LPM3) is 10x faster than the 1 Hz sampling task
    as5600.setPowerMode(AS5600_POWERMODE_LOW3);
    as5600.setWatchDog(AS5600_WATCHDOG_ON);

    const uint32_t nowMs = millis();
    CriticalSection lock(spinlock);
    lastDiagMs = 0;
    state.reset(nowMs);
}

bool WindVane::ready() {
    if (taskHandle != nullptr) {
        CriticalSection lock(spinlock);
        return state.ready();
    }
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
    const uint32_t nowMs = millis();
    sensors::WindVaneSample sample;
    sample.nowMs = nowMs;
    sample.samplePeriodMs = samplePeriodMs;
    sample.rawAngle = as5600.readAngle();
    sample.i2cOk = (as5600.lastError() == AS5600_OK);

    if (sample.i2cOk) {
        sample.hasDiag = (lastDiagMs == 0) || (nowMs - lastDiagMs >= 5000uL);
        if (sample.hasDiag) {
            lastDiagMs = nowMs;
            sample.diagStatus = as5600.readStatus();
            sample.diagAgc = as5600.readAGC();
        }
    }

    CriticalSection lock(spinlock);
    state.step(sample);
}

float WindVane::getAngle() {
    CriticalSection lock(spinlock);
    return state.angle();
}

float WindVane::getRawAngle() {
    CriticalSection lock(spinlock);
    return state.rawAngle();
}

void WindVane::setCalibrationOffset(float offset) {
    CriticalSection lock(spinlock);
    state.setCalibrationOffset(offset);
}

float WindVane::getCalibrationOffset() const {
    CriticalSection lock(spinlock);
    return state.calibrationOffset();
}

void WindVane::resetStats() {
    CriticalSection lock(spinlock);
    state.resetStats();
}

uint8_t WindVane::getLastStatus() {
    CriticalSection lock(spinlock);
    return state.lastStatus();
}

uint8_t WindVane::getLastAGC() {
    CriticalSection lock(spinlock);
    return state.lastAgc();
}

uint16_t WindVane::getAGCVariance() {
    CriticalSection lock(spinlock);
    return state.agcVariance();
}

uint32_t WindVane::getGlitchCount() {
    CriticalSection lock(spinlock);
    return state.glitchCount();
}

uint32_t WindVane::getLastHeartbeatMs() {
    CriticalSection lock(spinlock);
    return state.lastHeartbeatMs();
}

float WindVane::readAngle() {
    if (taskHandle == nullptr) {
        update();
    }
    return getAngle();
}
