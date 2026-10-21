#include <Arduino.h>
#include <Preferences.h>
#include <WiFi.h>
#include <Wire.h>
#include <cmath>
#include <esp_task_wdt.h>
#include <freertos/FreeRTOS.h>
#include <freertos/semphr.h>
#include <freertos/task.h>

#include <ARQ.h>
#include <AS5600.h>
#include <Adafruit_AHTX0.h>
#include <Adafruit_BMP085.h>
#include <Adafruit_SGP30.h>
#include <PMS.h>
#include <RadioAdapter.h>
#include <Weather.h>
#include <cc1101.h>

#include "SoftTwoWire.h"
#include "anemometer.h"
#include "constants.h"
#include "rainmeter.h"
#include "windvane.h"

constexpr ARQ::address_t CONTROL_NODE_ADDR = 1;
constexpr ARQ::address_t BASE_STATION_ADDR = 2;

CC1101::Radio radio(RADIO_CS_PIN, SCK_PIN, MISO_PIN, MOSI_PIN, RADIO_GDO0_PIN);
CC1101RadioAdapter radioAdapter(radio);
ARQ::ARQ arq(radioAdapter, CONTROL_NODE_ADDR);

Adafruit_AHTX0 aht;
Adafruit_BMP085 bmp180;
Adafruit_SGP30 sgp;
PMS pms(Serial2);

Anemometer anemometer(Wire);
RainMeter rainMeter(Wire1);
SoftTwoWire softWireWindVane(WIND_VANE_SDA_PIN, WIND_VANE_SCL_PIN);
WindVane windVane(softWireWindVane, WIND_VANE_CALIBRATION_OFFSET);

// Mutex protecting the shared Wire bus between AnemometerTask (Core 0) and loop (Core 1).
static SemaphoreHandle_t wireMutex = nullptr;

// Sensor initialization states, allowing hot-plug recovery on transient failures.
static bool ahtOk = false;
static bool bmpOk = false;
static bool sgpOk = false;

// Cached SGP30 TVOC from the required 1 Hz IAQmeasure loop.
static uint16_t latestTvoc = weather::kInvalidUint16;
static bool sgpHasData = false;
static uint32_t lastSgpMeasureMs = 0;
static uint32_t lastSgpBaselineSaveMs = 0;

// PMS7003 15-minute duty cycle state machine to preserve fan and laser lifespan.
enum class PmsState : uint8_t {
    Sleeping,
    WarmingUp,
    Reading,
};

static PmsState pmsState = PmsState::WarmingUp;
static uint32_t pmsCycleStartMs = 0;
static uint32_t pmsStateStartMs = 0;
static PMS::DATA latestPms = {};
static bool pmsPendingReport = false;

// Boot counter kept in NVS to spot crash loops on a hard to reach node.
static Preferences prefs;
static uint32_t bootCount = 0;

// State of the reading delivery pipeline toward the base station.
enum class SendPhase : uint8_t {
    Idle,
    Transmitting,
    RetryDelay,
    Dropped,
};

struct LinkState {
    SendPhase phase = SendPhase::Idle;
    uint8_t attempt = 0;
    uint32_t phaseStartMs = 0;
    uint32_t timeoutSnapshot = 0;
    uint8_t consecutiveDrops = 0;
    uint32_t sentOk = 0;
    uint32_t sentDropped = 0;
    uint32_t radioResyncs = 0;
};

static LinkState linkState;

// The newest unsent reading. A failed send drops it, the next sample wins.
static bool newReadingPending = false;
static WeatherData pendingReading = {};

static uint32_t lastReadingMs = 0;
static uint32_t lastDiagMs = 0;

void setupWatchdog() {
    esp_task_wdt_config_t config = {};
    config.timeout_ms = WATCHDOG_TIMEOUT_MS;
    config.idle_core_mask = 0;
    config.trigger_panic = true;
    esp_err_t initResult = esp_task_wdt_init(&config);
    esp_err_t addResult = esp_task_wdt_add(NULL);
    if (initResult != ESP_OK || addResult != ESP_OK) {
        Serial.printf("[WARN] Watchdog setup init=%d add=%d\n", initResult, addResult);
    }
}

void loadBootCounter() {
    if (!prefs.begin(NVS_NAMESPACE, false)) {
        return;
    }
    bootCount = prefs.getUInt(NVS_KEY_BOOTS, 0) + 1;
    prefs.putUInt(NVS_KEY_BOOTS, bootCount);
    prefs.end();
    if (bootCount > BOOT_CRASH_THRESHOLD) {
        Serial.printf("[WARN] Boot counter %lu, possible crash loop\n",
                      static_cast<unsigned long>(bootCount));
    }
}

void markHealthyBoot() {
    if (prefs.begin(NVS_NAMESPACE, false)) {
        prefs.putUInt(NVS_KEY_BOOTS, 0);
        prefs.end();
    }
}

void storeLastError(const char* tag) {
    if (prefs.begin(NVS_NAMESPACE, false)) {
        prefs.putString(NVS_KEY_LAST_ERR, tag);
        prefs.end();
    }
}

// Compute absolute humidity in mg/m^3 for SGP30 on-chip compensation.
uint32_t computeAbsoluteHumidityMgM3(float temperatureC, float relativeHumidityPct) {
    if (std::isnan(temperatureC) || std::isnan(relativeHumidityPct) ||
        relativeHumidityPct <= 0.0f) {
        return 0;
    }
    const float absHumidityGramsM3 =
        216.7f *
        ((relativeHumidityPct / 100.0f) * 6.112f *
         std::exp((17.62f * temperatureC) / (243.12f + temperatureC)) / (273.15f + temperatureC));
    return static_cast<uint32_t>(1000.0f * absHumidityGramsM3);
}

void initSgp30WithBaseline() {
    if (!sgp.begin(&Wire, true) || !sgp.IAQinit()) {
        sgpOk = false;
        return;
    }
    sgpOk = true;
    if (prefs.begin(NVS_NAMESPACE, true)) {
        uint16_t eco2Base = prefs.getUShort(NVS_KEY_SGP_ECO2, 0);
        uint16_t tvocBase = prefs.getUShort(NVS_KEY_SGP_TVOC, 0);
        prefs.end();
        if (eco2Base != 0 && tvocBase != 0) {
            sgp.setIAQBaseline(eco2Base, tvocBase);
        }
    }
}

void saveSgpBaselineIfDue(uint32_t nowMs) {
    if (!sgpOk || nowMs < SGP_WARMUP_BEFORE_SAVE_MS ||
        (nowMs - lastSgpBaselineSaveMs) < SGP_BASELINE_SAVE_INTERVAL_MS) {
        return;
    }
    lastSgpBaselineSaveMs = nowMs;

    uint16_t eco2Base = 0;
    uint16_t tvocBase = 0;
    bool ok = false;
    if (xSemaphoreTake(wireMutex, portMAX_DELAY) == pdTRUE) {
        ok = sgp.getIAQBaseline(&eco2Base, &tvocBase);
        xSemaphoreGive(wireMutex);
    }
    if (ok && eco2Base != 0 && tvocBase != 0 && prefs.begin(NVS_NAMESPACE, false)) {
        prefs.putUShort(NVS_KEY_SGP_ECO2, eco2Base);
        prefs.putUShort(NVS_KEY_SGP_TVOC, tvocBase);
        prefs.end();
    }
}

void updateSgp30(uint32_t nowMs) {
    if (nowMs - lastSgpMeasureMs < SGP_MEASURE_INTERVAL_MS) {
        return;
    }
    lastSgpMeasureMs = nowMs;
    if (!sgpOk) {
        return;
    }

    if (xSemaphoreTake(wireMutex, portMAX_DELAY) == pdTRUE) {
        if (sgp.IAQmeasure()) {
            latestTvoc = sgp.TVOC;
            sgpHasData = true;
        } else {
            sgpOk = false;
            sgpHasData = false;
        }
        xSemaphoreGive(wireMutex);
    }

    saveSgpBaselineIfDue(nowMs);
}

void updatePms(uint32_t nowMs) {
    switch (pmsState) {
    case PmsState::Sleeping:
        if (nowMs - pmsCycleStartMs >= (PMS_SAMPLE_INTERVAL_MS - PMS_WARMUP_MS)) {
            pmsCycleStartMs = nowMs;
            pmsStateStartMs = nowMs;
            pms.wakeUp();
            pmsState = PmsState::WarmingUp;
        }
        break;

    case PmsState::WarmingUp: {
        // Drain frames emitted while fan and laser stabilize.
        PMS::DATA discard;
        pms.read(discard);
        if (nowMs - pmsStateStartMs >= PMS_WARMUP_MS) {
            pmsStateStartMs = nowMs;
            pmsState = PmsState::Reading;
        }
        break;
    }

    case PmsState::Reading:
        if (pms.read(latestPms)) {
            pmsPendingReport = true;
            pms.sleep();
            pmsState = PmsState::Sleeping;
        } else if (nowMs - pmsStateStartMs >= PMS_READ_TIMEOUT_MS) {
            Serial.println("[WARN] PMS7003 read timed out, returning to sleep");
            pms.sleep();
            pmsState = PmsState::Sleeping;
        }
        break;
    }
}

void setup() {
    Serial.begin(115200);
    delay(200);
    Serial.println("\n==================================");
    Serial.println("  Control Node Starting");
    Serial.println("==================================");

    // Power down unused WiFi and Bluetooth radios to cut idle draw and self-heating.
    WiFi.mode(WIFI_OFF);
    btStop();

    setupWatchdog();
    loadBootCounter();

    wireMutex = xSemaphoreCreateMutex();
    Wire.begin(SENSOR_I2C_SDA_PIN, SENSOR_I2C_SCL_PIN, I2C_CLOCK_HZ);
    Wire.setTimeOut(I2C_TIMEOUT_MS);
    Wire1.begin(RAIN_METER_SDA_PIN, RAIN_METER_SCL_PIN, I2C_CLOCK_HZ);
    Wire1.setTimeOut(I2C_TIMEOUT_MS);
    Serial2.begin(PMS::BAUD_RATE, SERIAL_8N1, PMS_RX_PIN, PMS_TX_PIN);
    softWireWindVane.beginSoftwareI2C();

    pms.activeMode();
    pms.wakeUp();
    uint32_t nowMs = millis();
    pmsCycleStartMs = nowMs;
    pmsStateStartMs = nowMs;
    pmsState = PmsState::WarmingUp;

    ahtOk = aht.begin(&Wire);
    bmpOk = bmp180.begin(BMP085_ULTRAHIGHRES, &Wire);
    initSgp30WithBaseline();

    anemometer.setBusMutex(wireMutex);
    anemometer.initialize();
    anemometer.setSpeedFactor(ANEMOMETER_SPEED_FACTOR);
    windVane.initialize();
    rainMeter.initialize();

    // Background sampling tasks keep bursty sensors fresh without blocking.
    anemometer.startTask(10, 1, 0);
    windVane.startTask(1000, 1, 0);
    rainMeter.startTask(30, 1, 0);

    ARQ::Status radioStatus = arq.initializeCC1101();
    if (radioStatus != ARQ::STATUS_OK) {
        Serial.printf("[ERROR] Radio init failed, status %d\n", radioStatus);
        storeLastError("radio_init");
    } else {
        Serial.println("[OK] Radio initialized");
    }
}

// Sample every sensor into a reading, marking failed sensors as absent.
WeatherData sampleSensors() {
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
    reading.airQualityData.voc = weather::kInvalidUint16;

    if (anemometer.ready()) {
        reading.windData.windSpeed = weather::encodeWindSpeed(anemometer.getWindSpeed());
        reading.windData.anemometerMagnetStatus.status = anemometer.getLastStatus();
        reading.windData.anemometerMagnetStatus.agc = anemometer.getLastAGC();
        reading.windData.anemometerMagnetStatus.agcVariance =
            static_cast<uint16_t>(anemometer.getAGCVariance());
    }

    if (windVane.ready()) {
        reading.windData.windAngle = weather::encodeWindAngle(windVane.getAngle());
        reading.windData.windVaneMagnetStatus.status = windVane.getLastStatus();
        reading.windData.windVaneMagnetStatus.agc = windVane.getLastAGC();
        reading.windData.windVaneMagnetStatus.agcVariance =
            static_cast<uint16_t>(windVane.getAGCVariance());
    }

    if (xSemaphoreTake(wireMutex, portMAX_DELAY) == pdTRUE) {
        if (!ahtOk) {
            ahtOk = aht.begin(&Wire);
        }
        if (ahtOk) {
            sensors_event_t humidityEvent;
            sensors_event_t temperatureEvent;
            if (aht.getEvent(&humidityEvent, &temperatureEvent)) {
                reading.airData.temperature =
                    weather::encodeTemperature(temperatureEvent.temperature);
                reading.airData.humidity = weather::encodeHumidity(humidityEvent.relative_humidity);
                if (sgpOk) {
                    uint32_t absHum = computeAbsoluteHumidityMgM3(temperatureEvent.temperature,
                                                                  humidityEvent.relative_humidity);
                    if (absHum > 0) {
                        sgp.setHumidity(absHum);
                    }
                }
            } else {
                ahtOk = false;
            }
        }

        if (!bmpOk) {
            bmpOk = bmp180.begin(BMP085_ULTRAHIGHRES, &Wire);
        }
        if (bmpOk) {
            int32_t pascal = bmp180.readPressure();
            if (pascal > 0) {
                reading.airData.pressure =
                    weather::encodePressure(static_cast<float>(pascal) / 100.0f);
            } else {
                bmpOk = false;
            }
        }

        if (!sgpOk) {
            initSgp30WithBaseline();
        }
        xSemaphoreGive(wireMutex);
    }

    if (sgpHasData) {
        reading.airQualityData.voc = latestTvoc;
    }

    if (rainMeter.ready()) {
        RainData rain = rainMeter.getRainData();
        reading.rainData.rainfall_1h = rain.rainfall_1h;
        reading.rainData.rainfall_24h = rain.rainfall_24h;
        reading.rainData.rainRate = rain.rainRate;
        reading.rainData.magnetStatus = rain.magnetStatus;
    }

    if (pmsPendingReport) {
        reading.airQualityData.pm1_0 = latestPms.PM_SP_UG_1_0;
        reading.airQualityData.pm2_5 = latestPms.PM_SP_UG_2_5;
        reading.airQualityData.pm10_0 = latestPms.PM_SP_UG_10_0;
        pmsPendingReport = false;
    }

    return reading;
}

void reinitRadio() {
    ARQ::Status radioStatus = arq.initializeCC1101();
    linkState.radioResyncs++;
    if (radioStatus != ARQ::STATUS_OK) {
        Serial.printf("[ERROR] Radio re-init failed, status %d\n", radioStatus);
        storeLastError("radio_reinit");
    } else {
        Serial.println("[OK] Radio re-initialized");
    }
}

void dropReading(const char* reason) {
    linkState.sentDropped++;
    linkState.consecutiveDrops++;
    linkState.phase = SendPhase::Dropped;
    linkState.attempt = 0;
    Serial.printf("[WARN] Reading dropped: %s\n", reason);
    if (linkState.consecutiveDrops >= RADIO_REINIT_THRESHOLD) {
        reinitRadio();
        linkState.consecutiveDrops = 0;
    }
}

void sendAttempt() {
    uint8_t wire[sizeof(WeatherData)];
    if (!weather::packWeatherData(pendingReading, wire, sizeof(wire))) {
        dropReading("pack_failed");
        return;
    }

    linkState.timeoutSnapshot = arq.getTxTimeoutCount();
    ARQ::Status status = arq.startSendData(wire, sizeof(wire), BASE_STATION_ADDR);
    switch (status) {
    case ARQ::STATUS_OK:
        linkState.phase = SendPhase::Transmitting;
        linkState.phaseStartMs = millis();
        break;
    case ARQ::STATUS_CHANNEL_BUSY:
    case ARQ::STATUS_BUSY:
        linkState.phase = SendPhase::RetryDelay;
        linkState.phaseStartMs = millis();
        break;
    case ARQ::STATUS_INVALID_SIZE:
        dropReading("oversize_frame");
        break;
    default:
        dropReading("radio_error");
        break;
    }
}

void completeAttempt(bool successful) {
    if (successful) {
        linkState.sentOk++;
        linkState.consecutiveDrops = 0;
        linkState.attempt = 0;
        linkState.phase = SendPhase::Idle;
        markHealthyBoot();
        return;
    }

    linkState.attempt++;
    if (linkState.attempt < TX_MAX_ATTEMPTS) {
        linkState.phase = SendPhase::RetryDelay;
        linkState.phaseStartMs = millis();
        return;
    }
    dropReading("no_ack");
}

void handleTx() {
    switch (linkState.phase) {
    case SendPhase::Idle:
        if (newReadingPending) {
            newReadingPending = false;
            linkState.attempt = 0;
            sendAttempt();
        }
        break;

    case SendPhase::Transmitting:
        arq.poll(TX_ACK_TIMEOUT_MS);
        if (arq.getState() == ARQ::RadioState::IDLE) {
            bool timedOut = arq.getTxTimeoutCount() != linkState.timeoutSnapshot;
            completeAttempt(!timedOut);
        } else if (millis() - linkState.phaseStartMs > TX_HARD_TIMEOUT_MS) {
            Serial.println("[WARN] Send stalled, re-syncing radio");
            dropReading("stalled");
            reinitRadio();
        }
        break;

    case SendPhase::RetryDelay:
        if (millis() - linkState.phaseStartMs >= TX_BACKOFF_MS) {
            sendAttempt();
        }
        break;

    case SendPhase::Dropped:
        // Wait for the next sample before trying again.
        linkState.phase = SendPhase::Idle;
        break;
    }
}

void printDiagnostics(uint32_t nowMs) {
    Serial.printf("[Diag] up=%lu boots=%lu txok=%lu txdrop=%lu resyncs=%lu state=%d rssi=%d\n",
                  nowMs / 1000uL, bootCount, linkState.sentOk, linkState.sentDropped,
                  linkState.radioResyncs, static_cast<int>(arq.getState()), arq.getLastRSSI());
    Serial.printf("[Diag] anemo=%d vane=%d rain=%d aht=%d bmp=%d sgp=%d pmsState=%d\n",
                  anemometer.ready() ? 1 : 0, windVane.ready() ? 1 : 0, rainMeter.ready() ? 1 : 0,
                  ahtOk ? 1 : 0, bmpOk ? 1 : 0, sgpOk ? 1 : 0, static_cast<int>(pmsState));
}

bool tasksHealthy(uint32_t nowMs) {
    if (nowMs - anemometer.getLastHeartbeatMs() > WATCHDOG_TIMEOUT_MS) {
        Serial.println("[ERROR] Anemometer task stalled");
        storeLastError("anemo_wdt");
        return false;
    }
    if (nowMs - windVane.getLastHeartbeatMs() > WATCHDOG_TIMEOUT_MS) {
        Serial.println("[ERROR] WindVane task stalled");
        storeLastError("vane_wdt");
        return false;
    }
    if (nowMs - rainMeter.getLastHeartbeatMs() > WATCHDOG_TIMEOUT_MS) {
        Serial.println("[ERROR] RainMeter task stalled");
        storeLastError("rain_wdt");
        return false;
    }
    return true;
}

void loop() {
    uint32_t nowMs = millis();

    if (tasksHealthy(nowMs)) {
        esp_task_wdt_reset();
    }

    updatePms(nowMs);
    updateSgp30(nowMs);

#if CALIBRATION_MODE
    if (nowMs - lastReadingMs >= 1000uL) {
        lastReadingMs = nowMs;
        Serial.printf("[CALIB] Vane raw=%.1f deg, cal=%.1f deg | Anemo RPM=%.1f, speed=%.2f m/s\n",
                      windVane.getRawAngle(),
                      windVane.getAngle(),
                      anemometer.getRPM(),
                      anemometer.getWindSpeed());
    }
#else
    handleTx();

    if (nowMs - lastReadingMs >= REPORT_INTERVAL_MS) {
        lastReadingMs = nowMs;
        pendingReading = sampleSensors();
        newReadingPending = true;
        Serial.println("[INFO] Reading sampled");
    }
#endif

    if (nowMs - lastDiagMs >= 10000uL) {
        lastDiagMs = nowMs;
        printDiagnostics(nowMs);
    }

    // Pace the main loop so Core 1 enters light idle between events while staying
    // responsive during active radio ACK polling.
    uint32_t delayMs = (linkState.phase == SendPhase::Transmitting) ? 1u : 10u;
    vTaskDelay(pdMS_TO_TICKS(delayMs));
}
