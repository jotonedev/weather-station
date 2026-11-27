#ifdef ARDUINO

#include "Anemometer.h"

Anemometer::Anemometer(TwoWire& wire) : device_(wire, 0x000Du, true) {}

Anemometer::~Anemometer() {
    stopTask();
}

void Anemometer::setBusMutex(SemaphoreHandle_t mutex) {
    device_.setBusMutex(mutex);
}

void Anemometer::initialize(uint8_t directionPin, uint8_t direction) {
    device_.initialize(directionPin, direction);
    const uint32_t nowMs = millis();
    CriticalSection lock(spinlock_);
    state_.reset(nowMs);
}

bool Anemometer::ready() {
    if (taskHandle_ != nullptr) {
        CriticalSection lock(spinlock_);
        return state_.ready();
    }
    return device_.probeReady();
}

bool Anemometer::isConnected() {
    return device_.isConnected();
}

bool Anemometer::magnetDetected() {
    return device_.magnetDetected();
}

bool Anemometer::magnetTooStrong() {
    return device_.magnetTooStrong();
}

bool Anemometer::magnetTooWeak() {
    return device_.magnetTooWeak();
}

int Anemometer::getLastError() {
    return device_.lastError();
}

void Anemometer::taskTrampoline(void* arg) {
    auto* self = static_cast<Anemometer*>(arg);
    self->taskLoop(self->samplePeriodMs_);
}

bool Anemometer::startTask(uint32_t samplePeriodMs, UBaseType_t priority, BaseType_t core) {
    if (taskHandle_ != nullptr) {
        return true;
    }
    samplePeriodMs_ = samplePeriodMs;
    const BaseType_t res = xTaskCreatePinnedToCore(taskTrampoline, "AnemometerTask", 4096, this,
                                                   priority, &taskHandle_, core);
    return res == pdPASS;
}

void Anemometer::stopTask() {
    if (taskHandle_ != nullptr) {
        vTaskDelete(taskHandle_);
        taskHandle_ = nullptr;
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
            CriticalSection lock(spinlock_);
            standstill = state_.isStandstill();
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
    const As5600Reading reading = device_.poll();
    sensors::AnemometerSample sample;
    sample.nowMs = reading.nowMs;
    sample.nowUs = reading.nowUs;
    sample.rawAngle = reading.rawAngle;
    sample.i2cOk = reading.i2cOk;
    sample.hasDiag = reading.hasDiag;
    sample.diagStatus = reading.diagStatus;
    sample.diagAgc = reading.diagAgc;
    sample.diagMagnitude = reading.diagMagnitude;

    CriticalSection lock(spinlock_);
    state_.step(sample);
}

float Anemometer::getRPM() {
    CriticalSection lock(spinlock_);
    return state_.rpm();
}

float Anemometer::getWindSpeed() {
    CriticalSection lock(spinlock_);
    return state_.windSpeed();
}

void Anemometer::resetStats() {
    CriticalSection lock(spinlock_);
    state_.resetStats();
}

void Anemometer::setSpeedFactor(float factor) {
    CriticalSection lock(spinlock_);
    state_.setSpeedFactor(factor);
}

float Anemometer::getSpeedFactor() const {
    CriticalSection lock(spinlock_);
    return state_.speedFactor();
}

uint8_t Anemometer::getLastStatus() {
    CriticalSection lock(spinlock_);
    return state_.lastStatus();
}

uint8_t Anemometer::getLastAGC() {
    CriticalSection lock(spinlock_);
    return state_.lastAgc();
}

uint16_t Anemometer::getLastMagnitude() {
    CriticalSection lock(spinlock_);
    return state_.lastMagnitude();
}

uint16_t Anemometer::getAGCVariance() {
    CriticalSection lock(spinlock_);
    return state_.agcVariance();
}

uint32_t Anemometer::getGlitchCount() {
    CriticalSection lock(spinlock_);
    return state_.glitchCount();
}

int64_t Anemometer::getCumulativeCounts() {
    CriticalSection lock(spinlock_);
    return state_.cumulativeCounts();
}

uint32_t Anemometer::getLastHeartbeatMs() {
    CriticalSection lock(spinlock_);
    return state_.lastHeartbeatMs();
}

float Anemometer::readWindRPM() {
    if (taskHandle_ == nullptr) {
        update();
    }
    return getRPM();
}

float Anemometer::readWindSpeed() {
    return getWindSpeed();
}

#endif // ARDUINO
