#ifndef RAIN_METER_H
#define RAIN_METER_H

#include <Arduino.h>
#include <Wire.h>
#include <AS5600.h>
#include <Preferences.h>
#include <freertos/FreeRTOS.h>
#include <freertos/task.h>
#include <Weather.h>

/**
 * @brief Rain Meter Operational & Mechanical States
 */
enum class RainMeterState : uint8_t {
    UNINITIALIZED = 0,
    CALIBRATING,         // Discovering resting end-stops
    REST_LEFT,           // Resting steadily on left stop
    TRANSIT_TO_RIGHT,    // In motion towards right stop
    REST_RIGHT,          // Resting steadily on right stop
    TRANSIT_TO_LEFT,     // In motion towards left stop
    JAMMED               // Bucket stuck mid-transit (leaf/spiderweb/ice)
};

// Diagnostic & Status Flags (packed into RainData::flags)
constexpr uint8_t RAIN_FLAG_CALIBRATED   = 0x01; // Auto-calibration complete
constexpr uint8_t RAIN_FLAG_TILT_WARNING = 0x02; // Mechanical travel asymmetry > 15%
constexpr uint8_t RAIN_FLAG_JAMMED       = 0x04; // Bucket stuck between stops > 2s
constexpr uint8_t RAIN_FLAG_MAGNET_OK    = 0x08; // Magnet detected and within AGC limits

class RainMeter {
public:
    /**
     * @brief Construct a new Rain Meter object.
     * @param wire Reference to TwoWire or SoftTwoWire bus.
     * @param mmPerTip Rain volume per tip in mm (default 0.2 mm / tip).
     */
    RainMeter(TwoWire& wire, float mmPerTip = 0.2f);
    ~RainMeter();

    void initialize(uint8_t directionPin = AS5600_SW_DIRECTION_PIN, uint8_t direction = AS5600_CLOCK_WISE);
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
    uint32_t getTotalTips();
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

    // Maintenance & Calibration Controls
    void resetStats();
    void resetLifetimeTotals();
    void clearCalibration();
    void setMmPerTip(float mm);
    float getMmPerTip() const;

private:
    TwoWire& wire;
    AS5600 as5600;
    Preferences preferences;
    uint32_t samplePeriodMs = 30;

    TaskHandle_t taskHandle = nullptr;
    static void taskTrampoline(void* arg);
    void taskLoop(uint32_t samplePeriodMs);

    portMUX_TYPE spinlock = portMUX_INITIALIZER_UNLOCKED;

    bool initialized = false;
    float mmPerTip = 0.2f;

    // Angular tracking & unwrap logic
    bool hasReference = false;
    uint16_t rawReference = 0;
    int32_t currentPos = 0;       // Continuous unwrapped position relative to rawReference
    int32_t prevPos = 0;
    float filteredPos = 0.0f;
    uint32_t stillCount = 0;      // Counter for stationary samples

    // Auto-calibration state & resting clusters
    bool calibrated = false;
    int32_t restLeft = 0;
    int32_t restRight = 0;
    float centerPoint = 0.0f;
    float hysteresis = 6.0f;
    float travelSpan = 0.0f;
    bool hasClusterA = false;
    bool hasClusterB = false;
    int32_t clusterA = 0;
    int32_t clusterB = 0;

    // State machine
    RainMeterState currentState = RainMeterState::UNINITIALIZED;
    uint32_t transitStartMs = 0;
    bool tiltWarning = false;
    float tiltDegrees = 0.0f;
    bool jammed = false;

    // Rain counters & time tracking
    uint32_t totalTips = 0;
    uint32_t uncommittedTips = 0;
    uint32_t lastTipMs = 0;
    uint32_t prevTipMs = 0;
    float currentRainRate = 0.0f;

    // Sliding window accumulation
    // 60 bins of 1-minute rain for 1-hour total
    float minuteBins[60] = {0.0f};
    size_t currentMinuteIdx = 0;
    uint32_t lastMinuteTickMs = 0;

    // 24 bins of 1-hour rain for 24-hour total
    float hourBins[24] = {0.0f};
    size_t currentHourIdx = 0;
    uint32_t lastHourTickMs = 0;

    // Sensor health & diagnostics (matching Anemometer)
    uint8_t lastStatus = 0;
    uint8_t lastAgc = 0;
    uint32_t glitchCount = 0;
    uint32_t consecutiveGlitches = 0;

    // Welford's algorithm for AGC variance
    uint32_t agcCount = 0;
    double agcMean = 0.0;
    double agcM2 = 0.0;

    // Helper functions
    static inline int32_t shortestAngleDiff(uint16_t current, uint16_t reference) {
        int32_t diff = static_cast<int32_t>(current) - static_cast<int32_t>(reference);
        if (diff > 2048) diff -= 4096;
        else if (diff < -2048) diff += 4096;
        return diff;
    }

    void loadNvsCalibration();
    void saveNvsCalibration();
    void commitNvsTips();
    void recordTip(uint32_t nowMs);
    void updateRollingWindows(uint32_t nowMs);
    void recomputeGeometry();
};

#endif // RAIN_METER_H
