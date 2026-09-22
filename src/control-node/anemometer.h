#ifndef ANEMOMETER_H
#define ANEMOMETER_H

#include <Arduino.h> 
#include <Wire.h>
#include <AS5600.h>


class Anemometer {
    public:
        Anemometer(TwoWire& wire);

        void initialize(uint8_t directionPin = AS5600_SW_DIRECTION_PIN, uint8_t direction = AS5600_CLOCK_WISE);
        bool ready();
        float readWindRPM();
        float readWindSpeed();

    private:
        TwoWire& wire;
        AS5600 as5600;
};

#endif // ANEMOMETER_H
