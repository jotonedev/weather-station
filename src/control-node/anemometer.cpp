#include <Arduino.h> 
#include <cmath>

#include "anemometer.h"


Anemometer::Anemometer(TwoWire& wire) : wire(wire), as5600(&wire) {
}

void Anemometer::initialize(uint8_t directionPin, uint8_t direction) {
    as5600.begin(directionPin);
    as5600.setDirection(direction);
}

bool Anemometer::ready() {
    return as5600.isConnected() && as5600.magnetDetected() && !as5600.magnetTooStrong() && !as5600.magnetTooWeak();
}

float Anemometer::readWindRPM() {
    if (!ready()) {
        return -1.0f;
    }

    // Prime the baseline timestamp and angle for getAngularSpeed
    as5600.getAngularSpeed(AS5600_MODE_RPM, true);
    if (as5600.lastError() != AS5600_OK) {
        return -1.0f;
    }

    float sum = 0.0f;
    constexpr int SAMPLES = 3;

    for (int i = 0; i < SAMPLES; i++) {
        delay(10);
        float rpm = as5600.getAngularSpeed(AS5600_MODE_RPM, true);
        if (as5600.lastError() != AS5600_OK || isnan(rpm)) {
            return -1.0f;
        }
        sum += fabsf(rpm);
    }

    return sum / SAMPLES;
}

float Anemometer::readWindSpeed() {
    return readWindRPM();  // TODO: implement conversion from RPM to wind speed
}
