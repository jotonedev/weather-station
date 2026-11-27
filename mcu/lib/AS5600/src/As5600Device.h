#ifndef AS5600_DEVICE_H
#define AS5600_DEVICE_H

#ifdef ARDUINO

#include <AS5600.h>
#include <Arduino.h>
#include <MagnetDiagnostics.h>
#include <Sync.h>
#include <Wire.h>
#include <freertos/FreeRTOS.h>
#include <freertos/semphr.h>

struct As5600Reading {
    uint32_t nowMs = 0;
    uint32_t nowUs = 0;
    uint16_t rawAngle = 0;
    bool i2cOk = false;
    bool hasDiag = false;
    uint8_t diagStatus = 0;
    uint8_t diagAgc = 0;
    uint16_t diagMagnitude = 0;
    int err = 0;
};

class As5600Device {
  public:
    As5600Device(TwoWire& wire, uint16_t expectedConf, bool useRawAngle);
    ~As5600Device();

    void setBusMutex(SemaphoreHandle_t mutex);
    int initialize(uint8_t directionPin, uint8_t direction);
    As5600Reading poll();
    bool probeReady();

    bool isConnected();
    bool magnetDetected();
    bool magnetTooStrong();
    bool magnetTooWeak();
    int lastError() const;

  private:
    bool configureSensor(int& errOut);

    AS5600 as5600_;
    uint16_t expectedConf_;
    bool useRawAngle_;

    SemaphoreHandle_t busMutex_ = nullptr;
    bool ownsMutex_ = false;
    mutable portMUX_TYPE spinlock_ = portMUX_INITIALIZER_UNLOCKED;

    bool configVerified_ = false;
    bool needsDiag_ = true;
    bool lastMagnetHealthy_ = false;
    uint32_t lastDiagMs_ = 0;
    int lastErrorCode_ = AS5600_OK;
};

#endif // ARDUINO

#endif // AS5600_DEVICE_H
