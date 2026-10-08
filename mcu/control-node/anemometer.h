#ifndef ANEMOMETER_H
#define ANEMOMETER_H

#include <Arduino.h> 
#include <Wire.h>
#include <AS5600.h>
#include <freertos/FreeRTOS.h>
#include <freertos/task.h>

class Anemometer {
    public:
        Anemometer(TwoWire& wire);
        ~Anemometer();

        void initialize(uint8_t directionPin = AS5600_SW_DIRECTION_PIN, uint8_t direction = AS5600_CLOCK_WISE);
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
        uint16_t getAGCVariance();
        uint32_t getGlitchCount();
        int64_t getCumulativeCounts();

        // Synchronous / legacy fallback methods
        float readWindRPM();
        float readWindSpeed();

    private:
        TwoWire& wire;
        AS5600 as5600;

        TaskHandle_t taskHandle = nullptr;
        static void taskTrampoline(void* arg);
        void taskLoop(uint32_t samplePeriodMs);

        portMUX_TYPE spinlock = portMUX_INITIALIZER_UNLOCKED;

        bool initialized = false;
        int32_t lastRawAngle = -1;
        uint32_t lastValidMicros = 0;
        int64_t cumulativeCounts = 0;
        uint32_t consecutiveGlitches = 0;
        bool inDropout = false;
        uint32_t dropoutStartMicros = 0;
        int64_t dropoutStartCounts = 0;
        int8_t rotationDirection = 1;

        // Sliding window velocity estimation (15 samples @ 10ms = 150ms window)
        static constexpr size_t WINDOW_SIZE = 15;
        struct Sample {
            int64_t counts;
            uint32_t timeUs;
        };
        Sample history[WINDOW_SIZE];
        size_t historyIdx = 0;
        size_t historyCount = 0;

        // Filtered velocity
        float filteredRpm = 0.0f;
        float speedFactor = 0.05f; // m/s per RPM (calibration default)

        // Status & diagnostic counters
        uint8_t lastStatus = 0;
        uint8_t lastAgc = 0;
        uint32_t glitchCount = 0;

        // Running statistics for AGC variance (Welford's algorithm)
        uint32_t agcCount = 0;
        double agcMean = 0.0;
        double agcM2 = 0.0;
};

#endif // ANEMOMETER_H

