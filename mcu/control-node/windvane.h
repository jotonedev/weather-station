#ifndef WINDVANE_H
#define WINDVANE_H

#include <Arduino.h>
#include <Wire.h>
#include <AS5600.h>
#include <Weather.h>
#include <freertos/FreeRTOS.h>
#include <freertos/task.h>

class WindVane {
    public:
        WindVane(TwoWire& wire, float calibrationOffset = 0.0f);
        ~WindVane();

        void initialize(uint8_t directionPin = AS5600_SW_DIRECTION_PIN, uint8_t direction = AS5600_CLOCK_WISE);
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
        uint16_t getAGCVariance();
        uint32_t getGlitchCount();

        // Synchronous / legacy fallback
        float readAngle();

    private:
        TwoWire& wire;
        AS5600 as5600;
        float calibrationOffset;
        uint32_t samplePeriodMs = 1000;

        TaskHandle_t taskHandle = nullptr;
        static void taskTrampoline(void* arg);
        void taskLoop(uint32_t samplePeriodMs);

        portMUX_TYPE spinlock = portMUX_INITIALIZER_UNLOCKED;

        bool initialized = false;
        float currentAngle = 0.0f;
        float rawAngleDegrees = 0.0f;

        // Circular vector exponential moving average (dampens vane flutter & handles 0/360 wrap)
        float vecX = 1.0f;
        float vecY = 0.0f;
        bool hasFirstSample = false;

        // Diagnostics
        uint8_t lastStatus = 0;
        uint8_t lastAgc = 0;
        uint32_t glitchCount = 0;

        // Running statistics for AGC variance (Welford's algorithm)
        uint32_t agcCount = 0;
        double agcMean = 0.0;
        double agcM2 = 0.0;
};

#endif // WINDVANE_H

