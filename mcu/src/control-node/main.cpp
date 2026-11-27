#include <Arduino.h>
#include <WiFi.h>
#include <Wire.h>
#include <esp_task_wdt.h>
#include <freertos/FreeRTOS.h>
#include <freertos/semphr.h>
#include <freertos/task.h>

#include <ARQ.h>
#include <Adafruit_AHTX0.h>
#include <Adafruit_BMP085.h>
#include <Anemometer.h>
#include <PMS.h>
#include <RadioAdapter.h>
#include <RainMeter.h>
#include <Sync.h>
#include <TxLink.h>
#include <Weather.h>
#include <WindVane.h>
#include <cc1101.h>

#include "SoftTwoWire.h"
#include "boot_watchdog.h"
#include "constants.h"
#include "sensor_sampler.h"

CC1101::Radio radio(RADIO_CS_PIN, SCK_PIN, MISO_PIN, MOSI_PIN, RADIO_GDO0_PIN);
CC1101RadioAdapter radioAdapter(radio);
ARQ::ARQ arq(radioAdapter, weather::kControlNodeAddr);
ARQ::TxLinkController txLink(arq, weather::kBaseStationAddr,
                             {TX_ACK_TIMEOUT_MS, TX_MAX_ATTEMPTS, 2, TX_BACKOFF_MS,
                              TX_HARD_TIMEOUT_MS, RADIO_REINIT_THRESHOLD});

Adafruit_AHTX0 aht;
Adafruit_BMP085 bmpSensor;
Pms7003Driver pms(Serial2);

Anemometer anemometer(Wire);
RainMeter rainMeter(Wire1);
SoftTwoWire softWireWindVane(WIND_VANE_SDA_PIN, WIND_VANE_SCL_PIN);
WindVane windVane(softWireWindVane, WIND_VANE_CALIBRATION_OFFSET);

static SemaphoreHandle_t wireMutex = nullptr;
static TaskHandle_t mainLoopTaskHandle = nullptr;
static bool pendingHasPms = false;
static uint32_t lastReadingMs = 0;
static uint32_t lastDiagMs = 0;

void IRAM_ATTR onRadioPacketIsr() {
    if (mainLoopTaskHandle != nullptr) {
        BaseType_t higherPriorityWoken = pdFALSE;
        vTaskNotifyGiveFromISR(mainLoopTaskHandle, &higherPriorityWoken);
        portYIELD_FROM_ISR(higherPriorityWoken);
    }
}

void reinitRadio() {
    arq.setPacketInterruptCallback(onRadioPacketIsr);
    ARQ::Status radioStatus = arq.initializeCC1101();
    txLink.noteRadioResync();
    if (radioStatus != ARQ::STATUS_OK) {
        Serial.printf("[ERROR] Radio re-init failed, status %d\n", radioStatus);
        storeLastError("radio_reinit");
    } else {
        Serial.println("[OK] Radio re-initialized");
        arq.sleep();
    }
}

void setup() {
    Serial.begin(115200);
    delay(200);
    Serial.println("\n==================================");
    Serial.println("  Control Node Starting");
    Serial.println("==================================");

    WiFi.mode(WIFI_OFF);
    btStop();
#ifdef RGB_BUILTIN
    rgbLedWrite(RGB_BUILTIN, 0, 0, 0);
#endif

    setupPowerManagement();
    setupWatchdog();
    loadBootCounter();

    wireMutex = xSemaphoreCreateMutex();
    recoverI2cBusPins(SENSOR_I2C_SDA_PIN, SENSOR_I2C_SCL_PIN, true);
    Wire.begin(SENSOR_I2C_SDA_PIN, SENSOR_I2C_SCL_PIN, I2C_CLOCK_HZ);
    Wire.setTimeOut(I2C_TIMEOUT_MS);
    recoverI2cBusPins(RAIN_METER_SDA_PIN, RAIN_METER_SCL_PIN, true);
    Wire1.begin(RAIN_METER_SDA_PIN, RAIN_METER_SCL_PIN, I2C_CLOCK_HZ);
    Wire1.setTimeOut(I2C_TIMEOUT_MS);
    Serial2.begin(PMS_BAUD_RATE, SERIAL_8N1, PMS_RX_PIN, PMS_TX_PIN);
    softWireWindVane.beginSoftwareI2C();

    pms.begin(millis());
    aht.begin(&Wire, wireMutex);
    bmpSensor.begin(BMP085_ULTRAHIGHRES, &Wire, wireMutex);

    anemometer.setBusMutex(wireMutex);
    anemometer.initialize();
    anemometer.setSpeedFactor(ANEMOMETER_SPEED_FACTOR);
    windVane.initialize();
    rainMeter.initialize();

    anemometer.startTask(10, 2, 0);
    windVane.startTask(3000, 1, 0);
    rainMeter.startTask(30, 1, 0);

    mainLoopTaskHandle = xTaskGetCurrentTaskHandle();
    arq.setPacketInterruptCallback(onRadioPacketIsr);

    ARQ::Status radioStatus = arq.initializeCC1101();
    if (radioStatus != ARQ::STATUS_OK) {
        Serial.printf("[ERROR] Radio init failed, status %d\n", radioStatus);
        storeLastError("radio_init");
    } else {
        Serial.println("[OK] Radio initialized");
        arq.sleep();
    }
}

void printDiagnostics(uint32_t nowMs) {
    Serial.printf("[Diag] up=%lu boots=%lu txok=%lu txdrop=%lu resyncs=%lu state=%d rssi=%d\n",
                  nowMs / 1000uL, getBootCount(), txLink.sentOk(), txLink.sentDropped(),
                  txLink.radioResyncs(), static_cast<int>(arq.getState()), arq.getLastRSSI());
    Serial.printf("[Diag] anemo=%d vane=%d rain=%d aht=%d bmp=%d pmsState=%d\n",
                  anemometer.ready() ? 1 : 0, windVane.ready() ? 1 : 0, rainMeter.ready() ? 1 : 0,
                  aht.isOk() ? 1 : 0, bmpSensor.isOk() ? 1 : 0, static_cast<int>(pms.state()));
}

void loop() {
    uint32_t nowMs = millis();

    if (tasksHealthy(anemometer.getLastHeartbeatMs(), windVane.getLastHeartbeatMs(),
                     rainMeter.getLastHeartbeatMs())) {
        esp_task_wdt_reset();
    }

    pms.poll(nowMs);

#if CALIBRATION_MODE
    if (nowMs - lastReadingMs >= 1000uL) {
        lastReadingMs = nowMs;
        Serial.printf("[CALIB] Vane raw=%.1f deg, cal=%.1f deg | Anemo RPM=%.1f, speed=%.2f m/s\n",
                      windVane.getRawAngle(), windVane.getAngle(), anemometer.getRPM(),
                      anemometer.getWindSpeed());
    }
#else
    ARQ::TxLinkStepResult txResult = txLink.poll(nowMs);
    if (txResult.delivered) {
        if (pendingHasPms) {
            pms.markReportDelivered();
            pendingHasPms = false;
        }
        markHealthyBoot();
    } else if (txResult.dropped) {
        pendingHasPms = false;
        Serial.printf("[WARN] Reading dropped: %s\n",
                      txResult.dropReason != nullptr ? txResult.dropReason : "unknown");
        if (txResult.requestReinit || txResult.requestStallResync) {
            reinitRadio();
        }
    }

    if (nowMs - lastReadingMs >= REPORT_INTERVAL_MS) {
        lastReadingMs = nowMs;
        bool sampledHasPms = false;
        WeatherData reading =
            sampleAllSensors(nowMs, anemometer, windVane, rainMeter, aht, bmpSensor, pms, wireMutex,
                             arq, txLink, sampledHasPms);
        uint8_t wire[sizeof(WeatherData)];
        if (weather::packWeatherData(reading, wire, sizeof(wire)) &&
            txLink.submit(wire, sizeof(wire))) {
            pendingHasPms = sampledHasPms;
            Serial.println("[INFO] Reading sampled");
        }
    }
#endif

    if (nowMs - lastDiagMs >= 10000uL) {
        lastDiagMs = nowMs;
        printDiagnostics(nowMs);
    }

#if !CALIBRATION_MODE
    if (txLink.phase() == ARQ::SendPhase::Transmitting) {
        ulTaskNotifyTake(pdTRUE, pdMS_TO_TICKS(50));
        return;
    }
#endif
    uint32_t delayMs = 10u;
#if !CALIBRATION_MODE
    if (txLink.phase() == ARQ::SendPhase::Idle && !txLink.hasPending() &&
        pms.state() == pms7003::PmsState::Sleeping) {
        delayMs = 500u;
    }
#endif
    vTaskDelay(pdMS_TO_TICKS(delayMs));
}
