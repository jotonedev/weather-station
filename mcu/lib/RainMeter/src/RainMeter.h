#ifndef RAIN_METER_H
#define RAIN_METER_H

#ifdef ARDUINO

#include <Arduino.h>
#include <As5600Device.h>
#include <RainMeterLogic.h>
#include <Sync.h>
#include <Weather.h>
#include <Wire.h>
#include <freertos/FreeRTOS.h>
#include <freertos/semphr.h>
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
    explicit RainMeter(TwoWire& wire, float mmPerTip = 1.894075f);
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
    float getCurrentAngleDegrees();
    float getTipTravelDegrees();
    uint8_t getLastStatus();
    uint8_t getLastAGC();
    uint16_t getLastMagnitude();
    uint16_t getAGCVariance();
    uint32_t getGlitchCount();
    uint32_t getLastHeartbeatMs();

    // Maintenance & Calibration Controls
    void resetStats();
    void clearCalibration();
    void setMmPerTip(float mm);
    float getMmPerTip() const;

  private:
    As5600Device device_;
    uint32_t samplePeriodMs_ = 30;
    uint32_t currentPeriodMs_ = 30;

    TaskHandle_t taskHandle_ = nullptr;
    static void taskTrampoline(void* arg);
    void taskLoop(uint32_t samplePeriodMs);

    mutable portMUX_TYPE spinlock_ = portMUX_INITIALIZER_UNLOCKED;
    sensors::RainMeterLogic logic_;

    bool loadNvsCalibration(sensors::RainCalibration& out);
    void saveNvsCalibration(const sensors::RainCalibration& cal);
    void loadRtcRollingBins();
    void saveRtcRollingBins();
};

#endif // ARDUINO

#endif // RAIN_METER_H
