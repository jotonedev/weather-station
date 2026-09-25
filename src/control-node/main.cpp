#include <Arduino.h>
#include <Adafruit_AHTX0.h>
#include <Adafruit_BMP085.h>
#include <Adafruit_SGP30.h>
#include <AS5600.h>
#include <ARQ.h>
#include <PMS.h>
#include <Weather.h>
#include <Wire.h>


#include "anemometer.h"
#include "windvane.h"
#include "rainmeter.h"
#include "constants.h"
#include "SoftTwoWire.h"

constexpr ARQ::address_t CONTROL_NODE_ADDR = 1;
constexpr ARQ::address_t BASE_STATION_ADDR = 2;

CC1101::Radio radio(RADIO_CS_PIN, SCK_PIN, MISO_PIN, MOSI_PIN, RADIO_GDO0_PIN);
ARQ::ARQ arq(radio, CONTROL_NODE_ADDR);

Adafruit_AHTX0 aht;
Adafruit_BMP085 bmp180;
Adafruit_SGP30 sgp;
PMS pms(Serial2);

Anemometer anemometer(Wire);
RainMeter rainMeter(Wire1);
SoftTwoWire softWireWindVane(WIND_VANE_SDA_PIN, WIND_VANE_SCL_PIN);
WindVane windVane(softWireWindVane, WIND_VANE_CALIBRATION_OFFSET);


void setup() {
    Serial.begin(115200);
}

void loop() {

}
