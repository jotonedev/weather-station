#ifndef ANEMOMETER_H
#define ANEMOMETER_H

#ifdef ARDUINO

#include <AnemometerState.h>
#include <Arduino.h>
#include <As5600Device.h>
#include <Sync.h>
#include <Wire.h>
#include <freertos/FreeRTOS.h>
#include <freertos/semphr.h>
#include <freertos/task.h>

class Anemometer {
  public:
    explicit Anemometer(TwoWire& wire);
    ~Anemometer();

    void setBusMutex(SemaphoreHandle_t mutex);
    void initialize(uint8_t directionPin = AS5600_SW_DIRECTION_PIN,
                    uint8_t direction = AS5600_CLOCK_WISE);
    bool ready();
    bool isConnected();
    bool magnetDetected();
    bool magnetTooStrong();
    bool magnetTooWeak();
    int getLastError();

    // Background sampling task (pinned to Core 0 by default)
    bool startTask(uint32_t samplePeriodMs = 10, UBaseType_t priority = 1, BaseType_t core = 0);
    void stopTask();

    // Single sampling update (called by background task or polled manually)
    void update();

    // Velocity & wind telemetry (thread-safe)
    float getRPM();
    float getWindSpeed();

    // Reset statistics (e.g. between reporting intervals)
    void resetStats();

    // Calibration
    void setSpeedFactor(float factor);
    float getSpeedFactor() const;

    // Diagnostic & sensor health telemetry
    uint8_t getLastStatus();
    uint8_t getLastAGC();
    uint16_t getLastMagnitude();
    uint16_t getAGCVariance();
    uint32_t getGlitchCount();
    int64_t getCumulativeCounts();
    uint32_t getLastHeartbeatMs();

    // Synchronous / legacy fallback methods
    float readWindRPM();
    float readWindSpeed();

  private:
    As5600Device device_;
    uint32_t samplePeriodMs_ = 10;

    TaskHandle_t taskHandle_ = nullptr;
    static void taskTrampoline(void* arg);
    void taskLoop(uint32_t samplePeriodMs);

    mutable portMUX_TYPE spinlock_ = portMUX_INITIALIZER_UNLOCKED;
    sensors::AnemometerState state_;
};

#endif // ARDUINO

#endif // ANEMOMETER_H
