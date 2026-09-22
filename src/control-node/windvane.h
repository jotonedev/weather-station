#ifndef WINDVANE_H
#define WINDVANE_H

#include <Arduino.h>
#include <Wire.h>
#include <AS5600.h>
#include <Weather.h>

class WindVane {
    public:
        WindVane(TwoWire& wire, float calibrationOffset = 0.0f);

        void initialize(uint8_t directionPin = AS5600_SW_DIRECTION_PIN, uint8_t direction = AS5600_CLOCK_WISE);
        bool ready();
        WindDirection readWindDirection();
        float readAngle();

    private:
        TwoWire& wire;
        AS5600 as5600;
        float calibrationOffset;
};

#endif // WINDVANE_H
