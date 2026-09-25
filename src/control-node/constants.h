#ifndef CONSTANTS_H
#define CONSTANTS_H

// =============================================================================
// ESP32-S3 Clean & Conflict-Free Pin Assignment
// Avoids:
//   - Flash / PSRAM bus: GPIO 26–37 (strictly reserved on ESP32-S3)
//   - Native USB (D- / D+): GPIO 19, 20
//   - Strapping pins: GPIO 0, 3, 45, 46
//   - Hardware UART0 console: GPIO 43, 44
//   - Onboard RGB LED: GPIO 38 / 48
// =============================================================================

// --- CC1101 Radio SPI ---
#define RADIO_CS_PIN   10
#define MOSI_PIN       11
#define SCK_PIN        12
#define MISO_PIN       13
#define RADIO_GDO0_PIN 14

// --- I2C Bus 1 (Hardware Wire) ---
// Shared: Anemometer AS5600 (@ 0x36) + Environmental: AHT20 (@ 0x38), SGP30 (@ 0x58), BMP180 (@ 0x77)
#define I2C_1_SDA_PIN 4
#define I2C_1_SCL_PIN 5

#define SENSOR_I2C_SDA_PIN I2C_1_SDA_PIN
#define SENSOR_I2C_SCL_PIN I2C_1_SCL_PIN

#define ANEMOMETER_DIRECTION_PIN 6

// --- I2C Bus 2 (Hardware Wire1) ---
// Dedicated: Rain Meter AS5600 (@ 0x36)
#define RAIN_METER_SDA_PIN 8
#define RAIN_METER_SCL_PIN 9
#define RAIN_METER_DIRECTION_PIN 1

// --- I2C Bus 3 (Software I2C / SoftWire) ---
// Dedicated: Wind Vane AS5600 (@ 0x36)
#define WIND_VANE_SDA_PIN 15
#define WIND_VANE_SCL_PIN 16
#define WIND_VANE_DIRECTION_PIN 7
#define WIND_VANE_CALIBRATION_OFFSET 0.0f

// --- PMS7003 Particulate Matter Sensor (UART2) ---
#define PMS_RX_PIN 17  // Connects to PMS7003 TX
#define PMS_TX_PIN 18  // Connects to PMS7003 RX

#endif // CONSTANTS_H