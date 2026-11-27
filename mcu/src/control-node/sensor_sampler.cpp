#include "sensor_sampler.h"

#include <Wire.h>

#include <Sync.h>

#include "SoftTwoWire.h"
#include "boot_watchdog.h"
#include "constants.h"

namespace {

uint8_t wireEnvFailures = 0;

} // namespace

void recoverWireBus(SemaphoreHandle_t wireMutex) {
    MutexGuard wireLock(wireMutex);
    if (!wireLock.locked()) {
        return;
    }
    Wire.end();
    recoverI2cBusPins(SENSOR_I2C_SDA_PIN, SENSOR_I2C_SCL_PIN, true);
    Wire.begin(SENSOR_I2C_SDA_PIN, SENSOR_I2C_SCL_PIN, I2C_CLOCK_HZ);
    Wire.setTimeOut(I2C_TIMEOUT_MS);
}

WeatherData sampleAllSensors(uint32_t nowMs, Anemometer& anemometer, WindVane& windVane,
                             RainMeter& rainMeter, Adafruit_AHTX0& aht, Adafruit_BMP085& bmp180,
                             Pms7003Driver& pms, SemaphoreHandle_t wireMutex, const ARQ::ARQ& arq,
                             const ARQ::TxLinkController& txLink, bool& sampledHasPms) {
    WeatherData reading = {};
    reading.windData.windSpeed = weather::kInvalidUint16;
    reading.windData.windAngle = weather::kInvalidUint16;
    reading.windData.anemometerMagnetStatus = {weather::kInvalidMagnetStatus, 0, 0};
    reading.windData.windVaneMagnetStatus = {weather::kInvalidMagnetStatus, 0, 0};
    reading.airData.temperature = weather::kInvalidTemperature;
    reading.airData.humidity = weather::kInvalidUint16;
    reading.airData.pressure = weather::kInvalidUint16;
    reading.rainData.rainfall_1h = weather::kInvalidUint16;
    reading.rainData.rainfall_24h = weather::kInvalidUint16;
    reading.rainData.rainRate = weather::kInvalidUint16;
    reading.rainData.magnetStatus = {weather::kInvalidMagnetStatus, 0, 0};
    reading.airQualityData.pm1_0 = weather::kInvalidUint16;
    reading.airQualityData.pm2_5 = weather::kInvalidUint16;
    reading.airQualityData.pm10_0 = weather::kInvalidUint16;

    reading.windData.anemometerMagnetStatus = {
        anemometer.getLastStatus(),
        anemometer.getLastAGC(),
        static_cast<uint16_t>(anemometer.getAGCVariance()),
    };
    if (anemometer.ready()) {
        reading.windData.windSpeed = weather::encodeWindSpeed(anemometer.getWindSpeed());
    }
    anemometer.resetStats();

    reading.windData.windVaneMagnetStatus = {
        windVane.getLastStatus(),
        windVane.getLastAGC(),
        static_cast<uint16_t>(windVane.getAGCVariance()),
    };
    if (windVane.ready()) {
        reading.windData.windAngle = weather::encodeWindAngle(windVane.getAngle());
    }
    windVane.resetStats();

    bool envReadOk = false;
    if (!aht.isOk()) {
        aht.begin(&Wire, wireMutex);
    }
    if (aht.isOk()) {
        aht20::Aht20Reading ahtReading;
        if (aht.read(ahtReading)) {
            reading.airData.temperature = weather::encodeTemperature(ahtReading.temperatureC);
            reading.airData.humidity = weather::encodeHumidity(ahtReading.humidityPct);
            envReadOk = true;
        }
    }

    if (!bmp180.isOk()) {
        bmp180.begin(BMP085_ULTRAHIGHRES, &Wire, wireMutex);
    }
    if (bmp180.isOk()) {
        int32_t pascal = 0;
        if (bmp180.readPressurePa(pascal)) {
            reading.airData.pressure = weather::encodePressure(static_cast<float>(pascal) / 100.0f);
            envReadOk = true;
        }
    }

    if (envReadOk) {
        wireEnvFailures = 0;
    } else {
        wireEnvFailures++;
        if (wireEnvFailures >= I2C_RECOVERY_FAIL_THRESHOLD) {
            recoverWireBus(wireMutex);
            wireEnvFailures = 0;
        }
    }

    reading.rainData.magnetStatus = {
        rainMeter.getLastStatus(),
        rainMeter.getLastAGC(),
        rainMeter.getAGCVariance(),
    };
    if (rainMeter.ready() && !rainMeter.isJammed() && rainMeter.isCalibrated()) {
        RainData rain = rainMeter.getRainData();
        reading.rainData.rainfall_1h = rain.rainfall_1h;
        reading.rainData.rainfall_24h = rain.rainfall_24h;
        reading.rainData.rainRate = rain.rainRate;
    }
    rainMeter.resetStats();

    sampledHasPms = false;
    if (pms.hasPendingReport(nowMs)) {
        const pms7003::PmsFrame& frame = pms.latestFrame();
        reading.airQualityData.pm1_0 = frame.pm1_0_ae;
        reading.airQualityData.pm2_5 = frame.pm2_5_ae;
        reading.airQualityData.pm10_0 = frame.pm10_0_ae;
        sampledHasPms = true;
    }

    uint8_t healthFlags = 0;
    if (aht.isOk()) {
        healthFlags |= weather::kHealthFlagAhtOk;
    }
    if (bmp180.isOk()) {
        healthFlags |= weather::kHealthFlagBmpOk;
    }
    if (rainMeter.isCalibrated()) {
        healthFlags |= weather::kHealthFlagRainCalibrated;
    }
    if (rainMeter.isJammed()) {
        healthFlags |= weather::kHealthFlagRainJammed;
    }
    if (pms.timedOut()) {
        healthFlags |= weather::kHealthFlagPmsTimeout;
    }
    if (pms.hadError()) {
        healthFlags |= weather::kHealthFlagPmsError;
    }

    const uint32_t boots = getBootCount();
    const bool hasAck = txLink.sentOk() > 0;
    reading.health.flags = healthFlags;
    reading.health.resetReason = getLastResetReason();
    reading.health.bootCount = (boots > 0xFFFFuL) ? 0xFFFFu : static_cast<uint16_t>(boots);
    reading.health.lastAckRssi = hasAck ? arq.getLastRSSI() : 0;
    reading.health.lastAckLqi = hasAck ? arq.getLastLQI() : weather::kInvalidLqi;

    return reading;
}
