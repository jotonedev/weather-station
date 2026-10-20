#include <Arduino.h>
#include <cmath>
#include <cstdlib>

#include "anemometer.h"

Anemometer::Anemometer(TwoWire& wire) : wire(wire), as5600(&wire) {}

Anemometer::~Anemometer() {
    stopTask();
}

void Anemometer::setBusMutex(SemaphoreHandle_t mutex) {
    busMutex = mutex;
}

void Anemometer::initialize(uint8_t directionPin, uint8_t direction) {
    if (busMutex != nullptr) {
        xSemaphoreTake(busMutex, portMAX_DELAY);
    }
    as5600.begin(directionPin);
    as5600.setDirection(direction);
    // Enable 3 LSB hysteresis to suppress standstill noise at silicon level
    as5600.setHysteresis(AS5600_HYST_LSB3);
    // Slow filter 16x for maximum noise rejection
    as5600.setSlowFilter(AS5600_SLOW_FILT_16X);
    // Disable fast filter so noise spikes don't bypass slow filter
    as5600.setFastFilter(AS5600_FAST_FILT_NONE);
    if (busMutex != nullptr) {
        xSemaphoreGive(busMutex);
    }

    uint32_t nowMs = millis();
    portENTER_CRITICAL(&spinlock);
    initialized = true;
    sensorReady = false;
    lastHeartbeatMs = nowMs;
    lastRawAngle = -1;
    lastValidMicros = 0;
    cumulativeCounts = 0;
    historyCount = 0;
    historyIdx = 0;
    filteredRpm = 0.0f;
    consecutiveGlitches = 0;
    glitchCount = 0;
    agcCount = 0;
    agcMean = 0.0;
    agcM2 = 0.0;
    portEXIT_CRITICAL(&spinlock);
}

bool Anemometer::ready() {
    if (taskHandle != nullptr) {
        portENTER_CRITICAL(&spinlock);
        bool r = sensorReady;
        portEXIT_CRITICAL(&spinlock);
        return r;
    }
    if (busMutex != nullptr) {
        xSemaphoreTake(busMutex, portMAX_DELAY);
    }
    bool r = as5600.isConnected() && as5600.magnetDetected() && !as5600.magnetTooStrong() &&
             !as5600.magnetTooWeak();
    if (busMutex != nullptr) {
        xSemaphoreGive(busMutex);
    }
    return r;
}

bool Anemometer::isConnected() {
    if (busMutex != nullptr) {
        xSemaphoreTake(busMutex, portMAX_DELAY);
    }
    bool res = as5600.isConnected();
    if (busMutex != nullptr) {
        xSemaphoreGive(busMutex);
    }
    return res;
}

bool Anemometer::magnetDetected() {
    if (busMutex != nullptr) {
        xSemaphoreTake(busMutex, portMAX_DELAY);
    }
    bool res = as5600.magnetDetected();
    if (busMutex != nullptr) {
        xSemaphoreGive(busMutex);
    }
    return res;
}

bool Anemometer::magnetTooStrong() {
    if (busMutex != nullptr) {
        xSemaphoreTake(busMutex, portMAX_DELAY);
    }
    bool res = as5600.magnetTooStrong();
    if (busMutex != nullptr) {
        xSemaphoreGive(busMutex);
    }
    return res;
}

bool Anemometer::magnetTooWeak() {
    if (busMutex != nullptr) {
        xSemaphoreTake(busMutex, portMAX_DELAY);
    }
    bool res = as5600.magnetTooWeak();
    if (busMutex != nullptr) {
        xSemaphoreGive(busMutex);
    }
    return res;
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
    TickType_t xLastWakeTime = xTaskGetTickCount();
    const TickType_t xFrequency = pdMS_TO_TICKS(samplePeriodMs);
    while (true) {
        update();
        vTaskDelayUntil(&xLastWakeTime, xFrequency);
    }
}

void Anemometer::update() {
    uint32_t nowMs = millis();
    if (busMutex != nullptr) {
        xSemaphoreTake(busMutex, portMAX_DELAY);
    }
    if (!as5600.isConnected()) {
        if (busMutex != nullptr) {
            xSemaphoreGive(busMutex);
        }
        portENTER_CRITICAL(&spinlock);
        lastHeartbeatMs = nowMs;
        sensorReady = false;
        portEXIT_CRITICAL(&spinlock);
        return;
    }

    uint32_t nowUs = micros();
    uint16_t raw = as5600.readAngle();
    bool i2cOk = (as5600.lastError() == AS5600_OK);
    uint8_t status = as5600.readStatus();
    uint8_t agc = as5600.readAGC();
    if (busMutex != nullptr) {
        xSemaphoreGive(busMutex);
    }

    // Check hardware status: bit 5 = Magnet Detected (0x20), bit 4 = Magnet Low/Weak (0x10),
    // bit 3 = Magnet High/Strong (0x08)
    bool hwMagnetOk = i2cOk && (status & 0x20) && !(status & 0x10);
    bool fullReady = hwMagnetOk && !(status & 0x08);

    portENTER_CRITICAL(&spinlock);
    lastHeartbeatMs = nowMs;
    sensorReady = fullReady;
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

    bool isGlitch = !hwMagnetOk;
    int32_t delta = 0;

    // Kinematic outlier rejection: even if status didn't flag ML, reject impossible jumps
    if (!isGlitch && lastRawAngle >= 0) {
        delta = static_cast<int32_t>(raw) - lastRawAngle;
        if (delta > 2048) {
            delta -= 4096;
        } else if (delta < -2048) {
            delta += 4096;
        }

        uint32_t dtUs = (lastValidMicros > 0) ? (nowUs - lastValidMicros) : 10000;
        float expectedStep = (filteredRpm * 4096.0f / 60.0f) * (dtUs * 1e-6f);
        float stepDiff = std::abs(static_cast<float>(std::abs(delta)) - expectedStep);

        // Reject impossible angular velocity/acceleration jumps (>250 counts deviation)
        if (std::abs(delta) > 350 || stepDiff > 250.0f) {
            isGlitch = true;
        }
    }

    if (isGlitch) {
        glitchCount++;
        consecutiveGlitches++;

        if (!inDropout) {
            inDropout = true;
            dropoutStartMicros = (lastValidMicros > 0) ? lastValidMicros : (nowUs - 10000);
            dropoutStartCounts = cumulativeCounts;
        }

        uint32_t elapsedGapUs = nowUs - dropoutStartMicros;

        // Inertial coasting: bridge blind spots up to 1.5 seconds (supports slow speeds down to 10
        // RPM)
        if (elapsedGapUs < 1500000 && filteredRpm > 0.5f && lastRawAngle >= 0) {
            // Coast with subtle bearing friction deceleration (0.1% per 10ms)
            filteredRpm *= 0.999f;
            int32_t projectedStep =
                rotationDirection * static_cast<int32_t>((filteredRpm * 4096.0f / 60.0f) * 0.010f);
            cumulativeCounts += projectedStep;

            history[historyIdx] = {cumulativeCounts, nowUs};
            historyIdx = (historyIdx + 1) % WINDOW_SIZE;
            if (historyCount < WINDOW_SIZE) {
                historyCount++;
            }
        } else if (elapsedGapUs >= 1500000) {
            // Genuine standstill or disconnected sensor: decay smoothly to zero
            filteredRpm *= 0.85f;
            if (filteredRpm < 0.2f) {
                filteredRpm = 0.0f;
            }
            lastRawAngle = -1; // Force clean re-seed when sensor recovers
        }

        portEXIT_CRITICAL(&spinlock);
        return;
    }

    // Valid sample received!
    consecutiveGlitches = 0;

    // If recovering from a dropout, seamlessly re-anchor to the new position without teleportation
    if (inDropout) {
        inDropout = false;
        lastRawAngle = raw;
        lastValidMicros = nowUs;
        portEXIT_CRITICAL(&spinlock);
        return;
    }

    lastValidMicros = nowUs;

    if (lastRawAngle < 0) {
        lastRawAngle = raw;
        cumulativeCounts = 0;
        historyIdx = 0;
        historyCount = 1;
        history[0] = {cumulativeCounts, nowUs};
        portEXIT_CRITICAL(&spinlock);
        return;
    }

    // Track physical rotation direction
    if (delta > 2) {
        rotationDirection = 1;
    } else if (delta < -2) {
        rotationDirection = -1;
    }

    cumulativeCounts += delta;
    lastRawAngle = raw;

    // Record sample in circular history buffer
    history[historyIdx] = {cumulativeCounts, nowUs};
    historyIdx = (historyIdx + 1) % WINDOW_SIZE;
    if (historyCount < WINDOW_SIZE) {
        historyCount++;
    }

    // Compute velocity over the window (150 ms)
    size_t oldestIdx = (historyCount < WINDOW_SIZE) ? 0 : historyIdx;
    int64_t dCounts = cumulativeCounts - history[oldestIdx].counts;
    uint32_t dtUs = nowUs - history[oldestIdx].timeUs;

    float rawRpm = 0.0f;
    if (dtUs > 0) {
        // Deadband: <= 3 counts in 150 ms (~0.26 deg) is standstill jitter
        if (std::abs(dCounts) <= 3) {
            rawRpm = 0.0f;
        } else {
            rawRpm = (static_cast<float>(std::abs(dCounts)) * 60.0f * 1e6f) /
                     (static_cast<float>(dtUs) * 4096.0f);
        }
    }

    // Exponential Moving Average filter (alpha = 0.20)
    if (rawRpm == 0.0f && filteredRpm < 0.5f) {
        filteredRpm = 0.0f;
    } else {
        filteredRpm = 0.20f * rawRpm + 0.80f * filteredRpm;
        if (filteredRpm < 0.1f) {
            filteredRpm = 0.0f;
        }
    }

    portEXIT_CRITICAL(&spinlock);
}

float Anemometer::getRPM() {
    portENTER_CRITICAL(&spinlock);
    float val = filteredRpm;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

float Anemometer::getWindSpeed() {
    return getRPM() * speedFactor;
}

void Anemometer::resetStats() {
    portENTER_CRITICAL(&spinlock);
    agcCount = 0;
    agcMean = 0.0;
    agcM2 = 0.0;
    portEXIT_CRITICAL(&spinlock);
}

void Anemometer::setSpeedFactor(float factor) {
    portENTER_CRITICAL(&spinlock);
    speedFactor = factor;
    portEXIT_CRITICAL(&spinlock);
}

float Anemometer::getSpeedFactor() const {
    return speedFactor;
}

uint8_t Anemometer::getLastStatus() {
    portENTER_CRITICAL(&spinlock);
    uint8_t val = lastStatus;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

uint8_t Anemometer::getLastAGC() {
    portENTER_CRITICAL(&spinlock);
    uint8_t val = lastAgc;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

uint16_t Anemometer::getAGCVariance() {
    portENTER_CRITICAL(&spinlock);
    uint16_t var = (agcCount > 1) ? static_cast<uint16_t>(agcM2 / (agcCount - 1)) : 0;
    portEXIT_CRITICAL(&spinlock);
    return var;
}

uint32_t Anemometer::getGlitchCount() {
    portENTER_CRITICAL(&spinlock);
    uint32_t val = glitchCount;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

int64_t Anemometer::getCumulativeCounts() {
    portENTER_CRITICAL(&spinlock);
    int64_t val = cumulativeCounts;
    portEXIT_CRITICAL(&spinlock);
    return val;
}

uint32_t Anemometer::getLastHeartbeatMs() {
    portENTER_CRITICAL(&spinlock);
    uint32_t val = lastHeartbeatMs;
    portEXIT_CRITICAL(&spinlock);
    return val;
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
