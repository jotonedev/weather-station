#ifdef ARDUINO

#include "As5600Device.h"

As5600Device::As5600Device(TwoWire& wire, uint16_t expectedConf, bool useRawAngle)
    : as5600_(&wire), expectedConf_(expectedConf), useRawAngle_(useRawAngle) {}

As5600Device::~As5600Device() {
    if (ownsMutex_ && busMutex_ != nullptr) {
        vSemaphoreDelete(busMutex_);
        busMutex_ = nullptr;
        ownsMutex_ = false;
    }
}

void As5600Device::setBusMutex(SemaphoreHandle_t mutex) {
    if (ownsMutex_ && busMutex_ != nullptr) {
        vSemaphoreDelete(busMutex_);
        ownsMutex_ = false;
    }
    busMutex_ = mutex;
}

bool As5600Device::configureSensor(int& errOut) {
    as5600_.setConfiguration(expectedConf_);
    errOut = as5600_.lastError();
    if (errOut != AS5600_OK) {
        return false;
    }
    const uint16_t actual = as5600_.getConfiguration();
    errOut = as5600_.lastError();
    return errOut == AS5600_OK && actual == expectedConf_;
}

int As5600Device::initialize(uint8_t directionPin, uint8_t direction) {
    if (busMutex_ == nullptr) {
        busMutex_ = xSemaphoreCreateMutex();
        ownsMutex_ = (busMutex_ != nullptr);
    }

    int err = AS5600_OK;
    {
        MutexGuard busLock(busMutex_);
        as5600_.begin(directionPin);
        as5600_.setDirection(direction);
        configVerified_ = configureSensor(err);
        needsDiag_ = true;
        lastDiagMs_ = 0;
        lastMagnetHealthy_ = false;
    }

    CriticalSection lock(spinlock_);
    lastErrorCode_ = err;
    return err;
}

As5600Reading As5600Device::poll() {
    As5600Reading reading;
    int err = AS5600_OK;
    {
        MutexGuard busLock(busMutex_);
        const uint32_t nowMs = millis();
        reading.nowMs = nowMs;
        reading.nowUs = micros();
        reading.rawAngle = useRawAngle_ ? as5600_.rawAngle() : as5600_.readAngle();
        err = as5600_.lastError();
        reading.i2cOk = (err == AS5600_OK);

        if (!reading.i2cOk) {
            needsDiag_ = true;
        } else {
            const uint32_t diagIntervalMs = lastMagnetHealthy_ ? 5000uL : 1000uL;
            const bool shouldDiag = needsDiag_ || !configVerified_ || (lastDiagMs_ == 0) ||
                                    (nowMs - lastDiagMs_ >= diagIntervalMs);
            if (shouldDiag) {
                const uint16_t conf = as5600_.getConfiguration();
                err = as5600_.lastError();
                if (err != AS5600_OK) {
                    reading.i2cOk = false;
                    needsDiag_ = true;
                } else if (!configVerified_ || conf != expectedConf_) {
                    configVerified_ = configureSensor(err);
                    if (!configVerified_) {
                        reading.i2cOk = false;
                        needsDiag_ = true;
                    }
                }

                uint8_t status = 0;
                uint8_t agc = 0;
                uint16_t mag = 0;
                if (reading.i2cOk) {
                    status = as5600_.readStatus();
                    err = as5600_.lastError();
                    if (err != AS5600_OK) {
                        reading.i2cOk = false;
                        needsDiag_ = true;
                    }
                }
                if (reading.i2cOk) {
                    agc = as5600_.readAGC();
                    err = as5600_.lastError();
                    if (err != AS5600_OK) {
                        reading.i2cOk = false;
                        needsDiag_ = true;
                    }
                }
                if (reading.i2cOk) {
                    mag = as5600_.readMagnitude();
                    err = as5600_.lastError();
                    if (err != AS5600_OK) {
                        reading.i2cOk = false;
                        needsDiag_ = true;
                    }
                }
                if (reading.i2cOk) {
                    reading.hasDiag = true;
                    reading.diagStatus = status;
                    reading.diagAgc = agc;
                    reading.diagMagnitude = mag;
                    needsDiag_ = false;
                    lastDiagMs_ = nowMs;
                    lastMagnetHealthy_ = sensors::MagnetDiagnostics::isHealthyStatus(status);
                }
            }
        }
    }

    reading.err = err;
    CriticalSection lock(spinlock_);
    lastErrorCode_ = err;
    return reading;
}

bool As5600Device::probeReady() {
    uint8_t status = 0;
    int err = AS5600_OK;
    {
        MutexGuard busLock(busMutex_);
        status = as5600_.readStatus();
        err = as5600_.lastError();
    }
    {
        CriticalSection lock(spinlock_);
        lastErrorCode_ = err;
    }
    return err == AS5600_OK && sensors::MagnetDiagnostics::isHealthyStatus(status);
}

bool As5600Device::isConnected() {
    bool ok = false;
    int err = AS5600_OK;
    {
        MutexGuard busLock(busMutex_);
        ok = as5600_.isConnected();
        err = as5600_.lastError();
    }
    CriticalSection lock(spinlock_);
    lastErrorCode_ = err;
    return ok;
}

bool As5600Device::magnetDetected() {
    bool ok = false;
    int err = AS5600_OK;
    {
        MutexGuard busLock(busMutex_);
        ok = as5600_.magnetDetected();
        err = as5600_.lastError();
    }
    CriticalSection lock(spinlock_);
    lastErrorCode_ = err;
    return ok;
}

bool As5600Device::magnetTooStrong() {
    bool ok = false;
    int err = AS5600_OK;
    {
        MutexGuard busLock(busMutex_);
        ok = as5600_.magnetTooStrong();
        err = as5600_.lastError();
    }
    CriticalSection lock(spinlock_);
    lastErrorCode_ = err;
    return ok;
}

bool As5600Device::magnetTooWeak() {
    bool ok = false;
    int err = AS5600_OK;
    {
        MutexGuard busLock(busMutex_);
        ok = as5600_.magnetTooWeak();
        err = as5600_.lastError();
    }
    CriticalSection lock(spinlock_);
    lastErrorCode_ = err;
    return ok;
}

int As5600Device::lastError() const {
    CriticalSection lock(spinlock_);
    return lastErrorCode_;
}

#endif // ARDUINO
