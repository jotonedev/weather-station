#include <Arduino.h>
#include <esp_task_wdt.h>
#include <freertos/task.h>
#include <time.h>

#include <ARQ.h>
#include <Ingest.h>
#include <Outbox.h>
#include <RadioAdapter.h>
#include <Weather.h>
#include <cc1101.h>

#include "boot_watchdog.h"
#include "config.h"
#include "counters_nvs.h"
#include "outbox_sender.h"
#include "rtc_outbox.h"
#include "wifi_ntp_http.h"

CC1101::Radio radio(RADIO_CS_PIN, SCK_PIN, MISO_PIN, MOSI_PIN, RADIO_GDO0_PIN);
CC1101RadioAdapter radioAdapter(radio);
ARQ::ARQ arq(radioAdapter, weather::kBaseStationAddr);

static RtcOutbox rtcOutbox;
static CountersNvs countersNvs;
static char nodeId[32] = "unknown";
static uint32_t lastRadioRxMs = 0;

// Heartbeats updated by background tasks and verified before feeding the hardware watchdog.
static volatile uint32_t lastSenderHeartbeatMs = 0;
static volatile uint32_t lastNtpHeartbeatMs = 0;
static TaskHandle_t radioRxTaskHandle = nullptr;

// True once NTP has set the clock. Written by the NTP sync task, read by the
// loop when stamping received readings.
static volatile bool clockSynced = false;

void IRAM_ATTR onRadioPacketIsr() {
    if (radioRxTaskHandle != nullptr) {
        BaseType_t higherPriorityWoken = pdFALSE;
        vTaskNotifyGiveFromISR(radioRxTaskHandle, &higherPriorityWoken);
        portYIELD_FROM_ISR(higherPriorityWoken);
    }
}

void handleMessage(const ARQ::Message& message, const uint8_t* frame) {
    if (message.type != ARQ::DATA_FRAME || message.source != weather::kControlNodeAddr) {
        return;
    }
    if (message.length != sizeof(WeatherData)) {
        countersNvs.incrementFramesInvalid();
        return;
    }
    WeatherData data;
    if (!weather::unpackWeatherData(frame, message.length, data)) {
        countersNvs.incrementFramesInvalid();
        return;
    }

    uint32_t nowMs = millis();
    lastRadioRxMs = nowMs;

    PendingReading entry = {};
    entry.data = data;
    entry.sequence = message.sequence;
    entry.receivedMs = nowMs;

    uint32_t nowEpoch = clockSynced ? static_cast<uint32_t>(time(nullptr)) : 0;
    bool evicted = false;
    bool pushed = rtcOutbox.pushReading(entry, clockSynced, nowEpoch, evicted);
    countersNvs.incrementReceived(!pushed || evicted);
    countersNvs.setLastNodeHealth(data.health);
    markHealthyBoot();
}

void reinitRadio() {
    radioRxTaskHandle = xTaskGetCurrentTaskHandle();
    arq.setPacketInterruptCallback(onRadioPacketIsr);

    ARQ::Status radioStatus = arq.initializeCC1101();
    countersNvs.incrementRadioResyncs();
    if (radioStatus != ARQ::STATUS_OK) {
        Serial.printf("[ERROR] Radio re-init failed, status %d\n", radioStatus);
        storeLastError("radio_reinit");
    } else if (arq.startReceiveData(weather::kBaseStationAddr) != ARQ::STATUS_OK) {
        Serial.println("[ERROR] Failed to restart receiving");
        storeLastError("radio_rx");
    } else {
        Serial.println("[OK] Radio re-initialized and listening");
    }
    lastRadioRxMs = millis();
}

void setup() {
    Serial.begin(115200);
    delay(200);
    Serial.println("\n==================================");
    Serial.println("  Base Station Starting");
    Serial.println("==================================");

    // Disable Bluetooth to save power and reduce 2.4 GHz coexistence interference.
    btStop();

    setupWatchdog();
    loadBootCounter();

    if (!sanitizeNodeId(NODE_ID, nodeId, sizeof(nodeId))) {
        snprintf(nodeId, sizeof(nodeId), "station");
    }

    rtcOutbox.init();
    countersNvs.init();
    countersNvs.load();

    uint32_t nowMs = millis();
    lastRadioRxMs = nowMs;
    lastSenderHeartbeatMs = nowMs;
    lastNtpHeartbeatMs = nowMs;

    radioRxTaskHandle = xTaskGetCurrentTaskHandle();
    arq.setPacketInterruptCallback(onRadioPacketIsr);

    ARQ::Status radioStatus = arq.initializeCC1101();
    if (radioStatus != ARQ::STATUS_OK) {
        Serial.printf("[ERROR] Radio initialization failed, status %d\n", radioStatus);
        storeLastError("radio_init");
    } else {
        Serial.println("[OK] Radio initialized");
        if (arq.startReceiveData(weather::kBaseStationAddr) != ARQ::STATUS_OK) {
            Serial.println("[ERROR] Failed to start receiving");
            storeLastError("radio_rx");
        } else {
            Serial.println("[OK] Listening mode active");
        }
    }

    startNtpSyncTask(rtcOutbox, clockSynced, lastNtpHeartbeatMs);
    startOutboxSenderTask(rtcOutbox, countersNvs, nodeId, clockSynced, lastSenderHeartbeatMs);
    Serial.printf("Outbox capacity %d, node id %s\n", OUTBOX_CAPACITY, nodeId);
}

void loop() {
    if (tasksHealthy(lastSenderHeartbeatMs, lastNtpHeartbeatMs)) {
        esp_task_wdt_reset();
    }

    arq.poll();

    if (arq.available()) {
        uint8_t frame[weather::kMaxWirePayload];
        ARQ::Message message;
        if (arq.readMessage(message, frame, sizeof(frame)) == ARQ::STATUS_OK) {
            handleMessage(message, frame);
        }
    }

    uint32_t now = millis();
    // Re-sync the CC1101 receiver if no valid frame has arrived for 5 minutes.
    if (static_cast<int32_t>(now - lastRadioRxMs) >=
        static_cast<int32_t>(RADIO_SILENCE_REINIT_MS)) {
        Serial.println("[WARN] Radio silence threshold exceeded, re-initializing CC1101");
        reinitRadio();
    } else if (arq.getState() != ARQ::RadioState::RECEIVING) {
        // Keep the receiver armed. A crash or re-init leaves the radio idle.
        arq.startReceiveData(weather::kBaseStationAddr);
    }

    static uint32_t lastDiag = 0;
    if (now - lastDiag >= 10000uL) {
        lastDiag = now;
        BaseCounters snap = countersNvs.snapshot();
        size_t queued = rtcOutbox.count();
        SystemHealthData diagHealth = {};
        bool diagHasHealth = countersNvs.getLastNodeHealth(diagHealth);
        Serial.printf(
            "[Diag] up=%lu boots=%lu outbox=%u rssi=%d lqi=%u state=%d resyncs=%lu http=%d | "
            "recv=%lu sent=%lu drop=%lu inv=%lu bufdrop=%lu\n",
            now / 1000uL, static_cast<unsigned long>(getBootCount()), static_cast<unsigned>(queued),
            arq.getLastRSSI(), arq.getLastLQI(), static_cast<int>(arq.getState()),
            static_cast<unsigned long>(snap.radioResyncs), snap.lastHttpCode,
            static_cast<unsigned long>(snap.received), static_cast<unsigned long>(snap.sent),
            static_cast<unsigned long>(snap.dropped),
            static_cast<unsigned long>(snap.framesInvalid),
            static_cast<unsigned long>(snap.bufferDropped));
        if (diagHasHealth) {
            Serial.printf("[Diag] node flags=0x%02X rst=%u boots=%u ackRssi=%d ackLqi=%u\n",
                          diagHealth.flags, diagHealth.resetReason, diagHealth.bootCount,
                          diagHealth.lastAckRssi, diagHealth.lastAckLqi);
        }
    }

    // Block on GDO0 packet interrupt, waking every 500 ms for watchdog and fallback checks.
    ulTaskNotifyTake(pdTRUE, pdMS_TO_TICKS(500));
}
