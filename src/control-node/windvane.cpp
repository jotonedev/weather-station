#include <Arduino.h> 
#include <cmath>

#include "windvane.h"


WindVane::WindVane(TwoWire& wire, float calibrationOffset) 
    : wire(wire), as5600(&wire), calibrationOffset(calibrationOffset) {
}

void WindVane::initialize(uint8_t directionPin, uint8_t direction) {
    as5600.begin(directionPin);
    as5600.setDirection(direction);
}

bool WindVane::ready() {
    return as5600.isConnected() && as5600.magnetDetected() && !as5600.magnetTooStrong() && !as5600.magnetTooWeak();
}

float WindVane::readAngle() {
    if (!ready()) {
        return -1.0f;
    }

    float sumX = 0.0f;
    float sumY = 0.0f;
    constexpr int SAMPLES = 3;

    for (int i = 0; i < SAMPLES; i++) {
        uint16_t raw = as5600.readAngle();
        if (as5600.lastError() != AS5600_OK) {
            return -1.0f;
        }
        float deg = static_cast<float>(raw) * AS5600_RAW_TO_DEGREES;
        float rad = deg * (PI / 180.0f);
        sumX += cosf(rad);
        sumY += sinf(rad);
        if (i < SAMPLES - 1) {
            delayMicroseconds(500);
        }
    }

    float avgRad = atan2f(sumY, sumX);
    float avgDeg = avgRad * (180.0f / PI);
    if (avgDeg < 0.0f) {
        avgDeg += 360.0f;
    }
    return avgDeg;
}

WindDirection WindVane::readWindDirection() {
    float angle = readAngle();

    if (angle < 0.0f || isnan(angle)) {
        return UNKNOWN;
    }

    // Apply calibration offset and normalize to [0.0, 360.0)
    float calibratedAngle = fmodf(angle + calibrationOffset, 360.0f);
    if (calibratedAngle < 0.0f) {
        calibratedAngle += 360.0f;
    }

    // 8 sectors of 45 degrees, centered around 0, 45, 90, ... (+22.5 offset)
    int sector = static_cast<int>(floorf((calibratedAngle + 22.5f) / 45.0f)) % 8;
    return static_cast<WindDirection>(sector);
}
