#ifndef WIND_VANE_H
#define WIND_VANE_H

#ifdef ARDUINO

#include <Arduino.h>
#include <As5600Device.h>
#include <Sync.h>
#include <WindVaneState.h>
#include <Wire.h>
#include <freertos/FreeRTOS.h>
#include <freertos/semphr.h>
#include <freertos/task.h>

class WindVane {
  public:
    explicit WindVane(TwoWire& wire, float calibrationOffset = 0.0f);
    ~WindVane();

    void initialize(uint8_t directionPin = AS5600_SW_DIRECTION_PIN,
                    uint8_t direction = AS5600_CLOCK_WISE);
    bool ready();
    bool isConnected();
    bool magnetDetected();
    bool magnetTooStrong();
    bool magnetTooWeak();
    int getLastError();

    // Background sampling task (pinned to Core 0 by default, 1 Hz default)
    bool startTask(uint32_t samplePeriodMs = 1000, UBaseType_t priority = 1, BaseType_t core = 0);
    void stopTask();

    // Single sampling update (called by background task or polled manually)
    void update();

    // Filtered angle in degrees [0.0, 360.0)
    float getAngle();
    float getRawAngle();

    // Calibration
    void setCalibrationOffset(float offset);
    float getCalibrationOffset() const;

    // Reset statistics
    void resetStats();

    // Diagnostic & sensor health telemetry
    uint8_t getLastStatus();
    uint8_t getLastAGC();
    uint16_t getLastMagnitude();
    uint16_t getAGCVariance();
    uint32_t getGlitchCount();
    uint32_t getLastHeartbeatMs();

    // Synchronous / legacy fallback
    float readAngle();

  private:
    As5600Device device_;
    uint32_t samplePeriodMs_ = 1000;

    TaskHandle_t taskHandle_ = nullptr;
    static void taskTrampoline(void* arg);
    void taskLoop(uint32_t samplePeriodMs);

    mutable portMUX_TYPE spinlock_ = portMUX_INITIALIZER_UNLOCKED;
    sensors::WindVaneState state_;
};

#endif // ARDUINO

#endif // WIND_VANE_H
