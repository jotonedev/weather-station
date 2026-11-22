#ifndef RAIN_METER_H
#define RAIN_METER_H

#include "constants.h"
#include <AS5600.h>
#include <Arduino.h>
#include <Sensors.h>
#include <Sync.h>
#include <Weather.h>
#include <Wire.h>
#include <freertos/FreeRTOS.h>
#include <freertos/task.h>

using sensors::RainMeterState;

class RainMeter {
  public:
    /**
     * @brief Construct a new Rain Meter object.
     * @param wire Reference to TwoWire or SoftTwoWire bus.
     * @param mmPerTip Rain volume per tip in mm (default calculated from funnel r=55mm, V=18ml:
     * ~1.894 mm/tip).
     */
    RainMeter(TwoWire& wire, float mmPerTip = RAIN_METER_CALIBRATION_MM_PER_TIP);
    ~RainMeter();

    void initialize(uint8_t directionPin = AS5600_SW_DIRECTION_PIN,
                    uint8_t direction = AS5600_CLOCK_WISE);
    bool ready();
    bool isConnected();
    bool magnetDetected();
    bool magnetTooStrong();
    bool magnetTooWeak();
    int getLastError();

    // Background sampling task (FreeRTOS, 30-33 Hz default)
    bool startTask(uint32_t samplePeriodMs = 30, UBaseType_t priority = 1, BaseType_t core = 0);
    void stopTask();

    // Single sampling update (called by background task or manual polling)
    void update();

    // Telemetry & Rainfall Metrics (thread-safe)
    float getRainfall1h();
    float getRainfall24h();
    float getRainRate();
    RainData getRainData();
    RainMeterState getState();

    // Diagnostic & Health Telemetry
    bool isCalibrated();
    bool isJammed();
    bool isTiltWarning();
    float getTiltDegrees();
    float getCurrentAngleDegrees();
    float getTipTravelDegrees();
    uint8_t getLastStatus();
    uint8_t getLastAGC();
    uint16_t getAGCVariance();
    uint32_t getGlitchCount();
    uint32_t getLastHeartbeatMs();

    // Maintenance & Calibration Controls
    void resetStats();
    void clearCalibration();
    void setMmPerTip(float mm);
    float getMmPerTip() const;

  private:
    TwoWire& wire;
    AS5600 as5600;
    uint32_t samplePeriodMs = 30;

    TaskHandle_t taskHandle = nullptr;
    static void taskTrampoline(void* arg);
    void taskLoop(uint32_t samplePeriodMs);

    mutable portMUX_TYPE spinlock = portMUX_INITIALIZER_UNLOCKED;

    uint32_t lastDiagMs = 0;
    sensors::RainMeterLogic logic;

    bool loadNvsCalibration(sensors::RainCalibration& out);
    void saveNvsCalibration(const sensors::RainCalibration& cal);
    void loadRtcRollingBins();
    void saveRtcRollingBins();
};

#endif // RAIN_METER_H
