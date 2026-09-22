#include <Arduino.h>
#include <Wire.h>
#include <ARQ.h>

#include "anemometer.h"
#include "windvane.h"
#include "constants.h"

constexpr ARQ::address_t CONTROL_NODE_ADDR = 1;
constexpr ARQ::address_t BASE_STATION_ADDR = 2;

CC1101::Radio radio(RADIO_CS_PIN, SCK_PIN, MISO_PIN, MOSI_PIN, RADIO_GDO0_PIN);
ARQ::ARQ arq(radio, CONTROL_NODE_ADDR);

void setup() {
    Serial.begin(115200);

    Wire.begin(I2C_1_SDA_PIN, I2C_1_SCL_PIN);
    Wire1.begin(I2C_2_SDA_PIN, I2C_2_SCL_PIN);
}


void loop() {

}
