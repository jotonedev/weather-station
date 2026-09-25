#include <Arduino.h>
#include <Adafruit_AHTX0.h>
#include <Adafruit_BMP085.h>
#include <Adafruit_BMP280.h>
#include <Adafruit_SGP30.h>
#include <AS5600.h>
#include <ARQ.h>
#include <PMS.h>
#include <Weather.h>
#include <Wire.h>


#include "anemometer.h"
#include "windvane.h"
#include "constants.h"

constexpr ARQ::address_t CONTROL_NODE_ADDR = 1;
constexpr ARQ::address_t BASE_STATION_ADDR = 2;

CC1101::Radio radio(RADIO_CS_PIN, SCK_PIN, MISO_PIN, MOSI_PIN, RADIO_GDO0_PIN);
ARQ::ARQ arq(radio, CONTROL_NODE_ADDR);

Adafruit_AHTX0 aht;
Adafruit_BMP085 bmp180;
Adafruit_BMP280 bmp280;
Adafruit_SGP30 sgp;

Anemometer anemometer(Wire);
WindVane windVane(Wire1, WIND_VANE_CALIBRATION_OFFSET);

void setup() {
    Serial.begin(115200);
    delay(5000); // Wait for Serial to initialize
    Serial.println("Control Node Starting...");

/*
    Serial.println("Initializing ARQ...");
    auto arqStatus = arq.initializeCC1101();
    if (arqStatus != ARQ::STATUS_OK) {
        Serial.print("Failed to initialize ARQ: ");
        Serial.println(arqStatus);
    }
    Serial.println("ARQ initialized successfully.");

    Serial.println("Initializing Wind Vane...");
    windVane.initialize(WIND_VANE_DIRECTION_PIN, AS5600_CLOCK_WISE);
    if (!windVane.ready()) {
        Serial.println("Wind Vane not ready.");
    } else {
        Serial.println("Wind Vane initialized successfully.");
    }

    Serial.println("Initializing AHT sensor...");
    if (!aht.begin(&Wire1)) {
        Serial.println("Failed to initialize AHT sensor!");
    } else {
        Serial.println("AHT sensor initialized successfully.");
    }

    Serial.println("Initializing BMP180 sensor...");
    if (!bmp180.begin(&Wire1)) {
        Serial.println("Failed to initialize BMP180 sensor!");
    } else {
        Serial.println("BMP180 sensor initialized successfully.");
    }

    Serial.println("Initializing BMP280 sensor...");
    if (!bmp280.begin(&Wire1)) {
        Serial.println("Failed to initialize BMP280 sensor!");
    } else {
        Serial.println("BMP280 sensor initialized successfully.");
    }

    Serial.println("Initializing SGP30 sensor...");
    if (!sgp.begin(&Wire1)) {
        Serial.println("Failed to initialize SGP30 sensor!");
    } else {
        Serial.println("SGP30 sensor initialized successfully.");
    }
*/

    Wire.begin(I2C_1_SDA_PIN, I2C_1_SCL_PIN);
    Wire.setClock(400000);

    Wire1.begin(I2C_2_SDA_PIN, I2C_2_SCL_PIN);
    Wire1.setClock(400000);

    Serial.println("Initializing Anemometer...");
    anemometer.initialize(ANEMOMETER_DIRECTION_PIN, AS5600_CLOCK_WISE);
    bool aConnected = anemometer.isConnected();
    bool aDetected = anemometer.magnetDetected();
    bool aTooWeak = anemometer.magnetTooWeak();
    bool aTooStrong = anemometer.magnetTooStrong();
    Serial.printf("Anemometer -> I2C Connected: %s | Magnet Detected: %s | Weak: %s | Strong: %s\n",
                  aConnected ? "YES (0x36 found)" : "NO (Check SDA/SCL wiring)",
                  aDetected ? "YES" : "NO",
                  aTooWeak ? "YES" : "NO",
                  aTooStrong ? "YES" : "NO");

    if (anemometer.startTask(10, 1, 0)) {
        Serial.println("Anemometer background sampling task started (100 Hz on Core 0).");
    } else {
        Serial.println("Failed to start Anemometer background task!");
    }

    Serial.println("Initializing Wind Vane...");
    windVane.initialize(WIND_VANE_DIRECTION_PIN, AS5600_CLOCK_WISE);
    bool vConnected = windVane.isConnected();
    bool vDetected = windVane.magnetDetected();
    bool vTooWeak = windVane.magnetTooWeak();
    bool vTooStrong = windVane.magnetTooStrong();
    Serial.printf("Wind Vane -> I2C Connected: %s | Magnet Detected: %s | Weak: %s | Strong: %s\n",
                  vConnected ? "YES (0x36 found)" : "NO (Check SDA/SCL wiring)",
                  vDetected ? "YES" : "NO",
                  vTooWeak ? "YES" : "NO",
                  vTooStrong ? "YES" : "NO");

    if (windVane.startTask(10, 1, 0)) {
        Serial.println("Wind Vane background sampling task started (100 Hz on Core 0).");
    } else {
        Serial.println("Failed to start Wind Vane background task!");
    }
}


void loop() {
    float speed = anemometer.getRPM();
    auto aStatus = anemometer.getLastStatus();
    auto aAgc = anemometer.getLastAGC();
    auto aAgcVar = anemometer.getAGCVariance();
    auto aGlitches = anemometer.getGlitchCount();
    bool aConn = anemometer.isConnected();
    bool aDet = anemometer.magnetDetected();

    float angle = windVane.getAngle();
    float rawAngle = windVane.getRawAngle();
    auto vStatus = windVane.getLastStatus();
    auto vAgc = windVane.getLastAGC();
    auto vAgcVar = windVane.getAGCVariance();
    auto vGlitches = windVane.getGlitchCount();
    bool vConn = windVane.isConnected();
    bool vDet = windVane.magnetDetected();
    bool vWeak = windVane.magnetTooWeak();

    Serial.printf("[Anemometer] I2C: %4s | Mag: %3s | Speed: %6.2f RPM | Status: %2u | AGC: %3u | AGC Var: %4u | Glitches: %u\n",
                  aConn ? "OK" : "FAIL", aDet ? "YES" : "NO", speed, aStatus, aAgc, aAgcVar, aGlitches);
    Serial.printf("[Wind Vane]  I2C: %4s | Mag: %3s (Weak: %s) | Angle: %6.2f deg (Raw: %6.2f) | Status: %2u | AGC: %3u | Glitches: %u\n",
                  vConn ? "OK" : "FAIL", vDet ? "YES" : "NO", vWeak ? "YES" : "NO",
                  angle, rawAngle, vStatus, vAgc, vGlitches);

/*
    // Create zero-initialized WeatherData struct
    WeatherData weatherData = {};

    // Read wind data
    WindData windData = readWindData();
    weatherData.windData = windData;

    // Read air data
    AirData airData = readAirData();
    weatherData.airData = airData;

    // Read air quality data
    AirQualityData airQualityData = readAirQualityData();
    weatherData.airQualityData = airQualityData;

    // Send the weather data to the base station
    ARQ::Status sendStatus = arq.startSendData((uint8_t*)&weatherData, sizeof(weatherData), BASE_STATION_ADDR);
*/
    delay(200);
}


WindData readWindData() {
    WindData windData = {};
    windData.windSpeed = static_cast<uint16_t>(anemometer.getWindSpeed() * 10.0f);
    windData.windAngle = static_cast<uint16_t>(windVane.getAngle() * 10.0f);
    windData.anemometerMagnetStatus.status = anemometer.getLastStatus();
    windData.anemometerMagnetStatus.agc = anemometer.getLastAGC();
    windData.anemometerMagnetStatus.agcVariance = anemometer.getAGCVariance();
    windData.windVaneMagnetStatus.status = windVane.getLastStatus();
    windData.windVaneMagnetStatus.agc = windVane.getLastAGC();
    windData.windVaneMagnetStatus.agcVariance = windVane.getAGCVariance();
    return windData;
}


AirData readAirData() {
    AirData airData;
    // Populate the airData struct with the readings
    return airData;
}

AirQualityData readAirQualityData() {
    AirQualityData airQualityData;
    // Populate the airQualityData struct with the readings
    return airQualityData;
}

