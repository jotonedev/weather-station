#include <Arduino.h>

#include "anemometer.h"

Anemometer::Anemometer(TwoWire& wire) : wire(wire), as5600(&wire) {}

Anemometer::~Anemometer() {
    stopTask();
}

void Anemometer::setBusMutex(SemaphoreHandle_t mutex) {
    busMutex = mutex;
}

void Anemometer::initialize(uint8_t directionPin, uint8_t direction) {
    {
        MutexGuard busLock(busMutex);
        as5600.begin(directionPin);
        as5600.setDirection(direction);
        // Enable 3 LSB hysteresis to suppress standstill noise at silicon level
        as5600.setHysteresis(AS5600_HYST_LSB3);
        // Slow filter 8x to suppress noise spikes and jitter at low speeds
        as5600.setSlowFilter(AS5600_SLOW_FILT_8X);
        // Disable fast filter so noise spikes don't bypass slow filter
        as5600.setFastFilter(AS5600_FAST_FILT_NONE);
        // 5 ms internal polling (LPM1) to match the 10 ms task and watchdog to drop to LPM3 at rest
        as5600.setPowerMode(AS5600_POWERMODE_LOW1);
        as5600.setWatchDog(AS5600_WATCHDOG_ON);
    }

    const uint32_t nowMs = millis();
    CriticalSection lock(spinlock);
    lastDiagMs = 0;
    state.reset(nowMs);
}

bool Anemometer::ready() {
    if (taskHandle != nullptr) {
        CriticalSection lock(spinlock);
        return state.ready();
    }
    MutexGuard busLock(busMutex);
    return as5600.isConnected() && as5600.magnetDetected() && !as5600.magnetTooStrong() &&
           !as5600.magnetTooWeak();
}

bool Anemometer::isConnected() {
    MutexGuard busLock(busMutex);
    return as5600.isConnected();
}

bool Anemometer::magnetDetected() {
    MutexGuard busLock(busMutex);
    return as5600.magnetDetected();
}

bool Anemometer::magnetTooStrong() {
    MutexGuard busLock(busMutex);
    return as5600.magnetTooStrong();
}

bool Anemometer::magnetTooWeak() {
    MutexGuard busLock(busMutex);
    return as5600.magnetTooWeak();
}

int Anemometer::getLastError() {
    return as5600.lastError();
}

void Anemometer::taskTrampoline(void* arg) {
    auto* self = static_cast<Anemometer*>(arg);
    self->taskLoop(self->samplePeriodMs);
}

bool Anemometer::startTask(uint32_t samplePeriodMs, UBaseType_t priority, BaseType_t core) {
    if (taskHandle != nullptr) {
        return true;
    }
    this->samplePeriodMs = samplePeriodMs;
    BaseType_t res = xTaskCreatePinnedToCore(taskTrampoline, "AnemometerTask", 4096, this, priority,
                                             &taskHandle, core);
    return res == pdPASS;
}

void Anemometer::stopTask() {
    if (taskHandle != nullptr) {
        vTaskDelete(taskHandle);
        taskHandle = nullptr;
    }
}

void Anemometer::taskLoop(uint32_t samplePeriodMs) {
    constexpr uint32_t kStandstillPeriodMs = 40;
    const uint32_t activePeriodMs = (samplePeriodMs > 0) ? samplePeriodMs : 10;
    TickType_t xLastWakeTime = xTaskGetTickCount();
    bool wasStandstill = false;
    while (true) {
        update();
        bool standstill = false;
        {
            CriticalSection lock(spinlock);
            standstill = state.isStandstill();
        }
        if (standstill) {
            wasStandstill = true;
            vTaskDelay(pdMS_TO_TICKS(kStandstillPeriodMs));
        } else {
            if (wasStandstill) {
                wasStandstill = false;
                xLastWakeTime = xTaskGetTickCount();
            }
            vTaskDelayUntil(&xLastWakeTime, pdMS_TO_TICKS(activePeriodMs));
        }
    }
}

void Anemometer::update() {
    const uint32_t nowMs = millis();
    sensors::AnemometerSample sample;
    sample.nowMs = nowMs;
    {
        MutexGuard busLock(busMutex);
        sample.nowUs = micros();
        sample.rawAngle = as5600.readAngle();
        sample.i2cOk = (as5600.lastError() == AS5600_OK);

        sample.hasDiag = (lastDiagMs == 0) || (nowMs - lastDiagMs >= 5000uL) || (!sample.i2cOk);
        if (sample.hasDiag) {
            lastDiagMs = nowMs;
            sample.diagStatus = as5600.readStatus();
            sample.diagAgc = as5600.readAGC();
        }
    }

    CriticalSection lock(spinlock);
    state.step(sample);
}

float Anemometer::getRPM() {
    CriticalSection lock(spinlock);
    return state.rpm();
}

float Anemometer::getWindSpeed() {
    CriticalSection lock(spinlock);
    return state.windSpeed();
}

void Anemometer::resetStats() {
    CriticalSection lock(spinlock);
    state.resetStats();
}

void Anemometer::setSpeedFactor(float factor) {
    CriticalSection lock(spinlock);
    state.setSpeedFactor(factor);
}

float Anemometer::getSpeedFactor() const {
    CriticalSection lock(spinlock);
    return state.speedFactor();
}

uint8_t Anemometer::getLastStatus() {
    CriticalSection lock(spinlock);
    return state.lastStatus();
}

uint8_t Anemometer::getLastAGC() {
    CriticalSection lock(spinlock);
    return state.lastAgc();
}

uint16_t Anemometer::getAGCVariance() {
    CriticalSection lock(spinlock);
    return state.agcVariance();
}

uint32_t Anemometer::getGlitchCount() {
    CriticalSection lock(spinlock);
    return state.glitchCount();
}

int64_t Anemometer::getCumulativeCounts() {
    CriticalSection lock(spinlock);
    return state.cumulativeCounts();
}

uint32_t Anemometer::getLastHeartbeatMs() {
    CriticalSection lock(spinlock);
    return state.lastHeartbeatMs();
}

float Anemometer::readWindRPM() {
    if (taskHandle == nullptr) {
        update();
    }
    return getRPM();
}

float Anemometer::readWindSpeed() {
    return getWindSpeed();
}
