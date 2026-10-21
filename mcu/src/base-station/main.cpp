#include <Arduino.h>
#include <HTTPClient.h>
#include <Preferences.h>
#include <WiFi.h>
#include <WiFiClientSecure.h>
#include <cstring>
#include <esp_attr.h>
#include <esp_rom_crc.h>
#include <esp_task_wdt.h>
#include <time.h>

#include <ARQ.h>
#include <Ingest.h>
#include <Outbox.h>
#include <RadioAdapter.h>
#include <Weather.h>
#include <cc1101.h>

#include "config.h"

namespace {

constexpr uint32_t kOutboxRtcMagic = 0x4F424F58u; // "OBOX"

// Retained in RTC Slow SRAM across watchdog resets and software reboots without wearing flash.
struct RtcOutboxState {
    uint32_t magic;
    uint16_t capacity;
    uint16_t head;
    uint16_t count;
    uint32_t dropped;
    PendingReading entries[OUTBOX_CAPACITY];
    uint32_t crc;
};

RTC_NOINIT_ATTR RtcOutboxState s_rtcOutbox;

uint32_t computeOutboxRtcCrc(const RtcOutboxState& state) {
    return esp_rom_crc32_le(0, reinterpret_cast<const uint8_t*>(&state),
                            offsetof(RtcOutboxState, crc));
}

} // namespace

CC1101::Radio radio(RADIO_CS_PIN, SCK_PIN, MISO_PIN, MOSI_PIN, RADIO_GDO0_PIN);
CC1101RadioAdapter radioAdapter(radio);
ARQ::ARQ arq(radioAdapter, BASE_STATION_ADDR);

static Outbox outbox(s_rtcOutbox.entries, OUTBOX_CAPACITY);
static SemaphoreHandle_t outboxMutex = nullptr;
static char nodeId[32] = "unknown";

static Preferences prefs;
static uint32_t bootCount = 0;
static bool healthyBootMarked = false;
static uint32_t lastCounterSaveMs = 0;
static uint32_t lastRadioRxMs = 0;

struct Counters {
    uint32_t received = 0;
    uint32_t sent = 0;
    uint32_t dropped = 0;
    uint32_t framesInvalid = 0;
    uint32_t bufferDropped = 0;
    uint32_t radioResyncs = 0;
    int lastHttpCode = 0;
};
static Counters counters;
static Counters lastSavedCounters;

// Heartbeats updated by background tasks and verified before feeding the hardware watchdog.
static volatile uint32_t lastSenderHeartbeatMs = 0;
static volatile uint32_t lastNtpHeartbeatMs = 0;

static BackoffPolicy backoff = {BACKOFF_MIN_MS, BACKOFF_MAX_MS, BACKOFF_MULTIPLIER, 0};

// The clock counts as synced once it passes roughly 2020.
constexpr time_t kClockSyncedEpoch = 1600000000;

// True once NTP has set the clock. Written by the NTP sync task, read by the
// loop when stamping received readings.
static volatile bool clockSynced = false;

void syncRtcOutbox() {
    s_rtcOutbox.magic = kOutboxRtcMagic;
    s_rtcOutbox.capacity = static_cast<uint16_t>(OUTBOX_CAPACITY);
    s_rtcOutbox.head = static_cast<uint16_t>(outbox.head());
    s_rtcOutbox.count = static_cast<uint16_t>(outbox.count());
    s_rtcOutbox.dropped = outbox.dropped();
    s_rtcOutbox.crc = computeOutboxRtcCrc(s_rtcOutbox);
}

void initRtcOutbox() {
    if (s_rtcOutbox.magic == kOutboxRtcMagic && s_rtcOutbox.capacity == OUTBOX_CAPACITY &&
        s_rtcOutbox.crc == computeOutboxRtcCrc(s_rtcOutbox) &&
        outbox.restoreState(s_rtcOutbox.head, s_rtcOutbox.count, s_rtcOutbox.dropped)) {
        Serial.printf("[OK] Restored %u buffered readings from RTC SRAM\n",
                      static_cast<unsigned>(outbox.count()));
        return;
    }
    std::memset(&s_rtcOutbox, 0, sizeof(s_rtcOutbox));
    outbox.reset();
    syncRtcOutbox();
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
    if (healthyBootMarked) {
        return;
    }
    healthyBootMarked = true;
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

void delayWithHeartbeat(volatile uint32_t& heartbeatMs, uint32_t totalDelayMs) {
    uint32_t remaining = totalDelayMs;
    while (remaining > 0) {
        heartbeatMs = millis();
        uint32_t step = (remaining > 2000uL) ? 2000uL : remaining;
        vTaskDelay(pdMS_TO_TICKS(step));
        remaining -= step;
    }
    heartbeatMs = millis();
}

bool syncClockOnce(uint32_t timeoutMs) {
    configTime(0, 0, NTP_SERVER);
    uint32_t started = millis();
    while (time(nullptr) < kClockSyncedEpoch && millis() - started < timeoutMs) {
        lastNtpHeartbeatMs = millis();
        vTaskDelay(pdMS_TO_TICKS(200));
    }
    lastNtpHeartbeatMs = millis();
    bool synced = time(nullptr) >= kClockSyncedEpoch;
    if (synced) {
        time_t current = time(nullptr);
        struct tm now;
        gmtime_r(&current, &now);
        Serial.printf("[OK] NTP synced %04d-%02d-%02dT%02d:%02d:%02dZ\n", now.tm_year + 1900,
                      now.tm_mon + 1, now.tm_mday, now.tm_hour, now.tm_min, now.tm_sec);
    } else {
        Serial.println("[WARN] NTP sync failed, server unreachable");
    }
    return synced;
}

// Resync the clock every 12 hours. When the server is unreachable retry sooner
// and keep a previously synced clock running instead of demoting it.
void ntpSyncTask(void*) {
    lastNtpHeartbeatMs = millis();
    while (true) {
        lastNtpHeartbeatMs = millis();
        if (WiFi.status() != WL_CONNECTED) {
            delayWithHeartbeat(lastNtpHeartbeatMs, WIFI_RECONNECT_DELAY_MS);
            continue;
        }
        if (syncClockOnce(NTP_SYNC_TIMEOUT_MS)) {
            clockSynced = true;
            uint32_t nowEpoch = static_cast<uint32_t>(time(nullptr));
            uint32_t nowMs = millis();
            if (xSemaphoreTake(outboxMutex, portMAX_DELAY) == pdTRUE) {
                if (outbox.backfillTimestamps(nowEpoch, nowMs) > 0) {
                    syncRtcOutbox();
                }
                xSemaphoreGive(outboxMutex);
            }
            delayWithHeartbeat(lastNtpHeartbeatMs, NTP_RESYNC_INTERVAL_MS);
        } else {
            if (!clockSynced) {
                Serial.println("[WARN] NTP unreachable, readings will be sent untimestamped");
            } else {
                Serial.println("[WARN] NTP re-sync failed, keeping the current clock");
            }
            delayWithHeartbeat(lastNtpHeartbeatMs, NTP_RETRY_INTERVAL_MS);
        }
    }
}

void ensureWifi() {
    static uint32_t lastAttempt = 0;
    if (WiFi.status() == WL_CONNECTED) {
        return;
    }
    uint32_t now = millis();
    if (lastAttempt != 0 && now - lastAttempt < WIFI_RECONNECT_DELAY_MS) {
        return;
    }
    lastAttempt = now;

    WiFi.persistent(false);
    WiFi.setAutoReconnect(true);
    WiFi.mode(WIFI_STA);
    WiFi.begin(WIFI_SSID, WIFI_PASSWORD);

    uint32_t started = millis();
    while (WiFi.status() != WL_CONNECTED && millis() - started < WIFI_CONNECT_TIMEOUT_MS) {
        lastSenderHeartbeatMs = millis();
        vTaskDelay(pdMS_TO_TICKS(200));
    }
    lastSenderHeartbeatMs = millis();
    if (WiFi.status() == WL_CONNECTED) {
        Serial.printf("[OK] WiFi connected, ip=%s\n", WiFi.localIP().toString().c_str());
    } else {
        Serial.println("[WARN] WiFi connect failed");
    }
}

int postToIngest(const char* url, const char* body) {
    lastSenderHeartbeatMs = millis();
    WiFiClientSecure secureClient;
    HTTPClient http;
    http.setTimeout(HTTP_TIMEOUT_MS);
#if INGEST_HTTPS
    // Trust the Arduino core cert bundle. Pin a specific cert for production.
    secureClient.setInsecure();
    http.begin(secureClient, url);
#else
    http.begin(url);
#endif
    if (std::strlen(INGEST_AUTH_USER) > 0) {
        http.setAuthorization(INGEST_AUTH_USER, INGEST_AUTH_PASSWORD);
    }
    http.addHeader("Content-Type", "application/json");
    int code = http.POST(body);
    http.end();
    lastSenderHeartbeatMs = millis();
    return code;
}

bool isTransientFailure(int httpCode) {
    if (httpCode < 0) {
        return true; // network level error
    }
    if (httpCode >= 500) {
        return true;
    }
    return httpCode == 408 || httpCode == 429;
}

void loadCounters() {
    if (!prefs.begin(NVS_NAMESPACE, true)) {
        return;
    }
    counters.received = prefs.getUInt(NVS_KEY_RECEIVED, 0);
    counters.sent = prefs.getUInt(NVS_KEY_SENT, 0);
    counters.dropped = prefs.getUInt(NVS_KEY_DROPPED, 0);
    counters.framesInvalid = prefs.getUInt(NVS_KEY_FRAMES_INVALID, 0);
    counters.bufferDropped = prefs.getUInt(NVS_KEY_BUFFER_DROPPED, 0);
    counters.lastHttpCode = prefs.getInt(NVS_KEY_LAST_HTTP, 0);
    prefs.end();
    lastSavedCounters = counters;
}

void saveCounters() {
    Counters snapshot;
    if (xSemaphoreTake(outboxMutex, portMAX_DELAY) == pdTRUE) {
        snapshot = counters;
        xSemaphoreGive(outboxMutex);
    }

    if (snapshot.received == lastSavedCounters.received &&
        snapshot.sent == lastSavedCounters.sent && snapshot.dropped == lastSavedCounters.dropped &&
        snapshot.framesInvalid == lastSavedCounters.framesInvalid &&
        snapshot.bufferDropped == lastSavedCounters.bufferDropped &&
        snapshot.lastHttpCode == lastSavedCounters.lastHttpCode) {
        return;
    }

    if (!prefs.begin(NVS_NAMESPACE, false)) {
        return;
    }
    prefs.putUInt(NVS_KEY_RECEIVED, snapshot.received);
    prefs.putUInt(NVS_KEY_SENT, snapshot.sent);
    prefs.putUInt(NVS_KEY_DROPPED, snapshot.dropped);
    prefs.putUInt(NVS_KEY_FRAMES_INVALID, snapshot.framesInvalid);
    prefs.putUInt(NVS_KEY_BUFFER_DROPPED, snapshot.bufferDropped);
    prefs.putInt(NVS_KEY_LAST_HTTP, snapshot.lastHttpCode);
    prefs.end();
    lastSavedCounters = snapshot;
}

void handleMessage(const ARQ::Message& message, const uint8_t* frame) {
    if (message.type != ARQ::DATA_FRAME || message.source != CONTROL_NODE_ADDR) {
        return;
    }
    if (message.length != sizeof(WeatherData)) {
        if (xSemaphoreTake(outboxMutex, portMAX_DELAY) == pdTRUE) {
            counters.framesInvalid++;
            xSemaphoreGive(outboxMutex);
        }
        return;
    }
    WeatherData data;
    if (!weather::unpackWeatherData(frame, message.length, data)) {
        if (xSemaphoreTake(outboxMutex, portMAX_DELAY) == pdTRUE) {
            counters.framesInvalid++;
            xSemaphoreGive(outboxMutex);
        }
        return;
    }

    uint32_t nowMs = millis();
    lastRadioRxMs = nowMs;

    PendingReading entry = {};
    entry.data = data;
    entry.sequence = message.sequence;
    entry.receivedAt = clockSynced ? static_cast<uint32_t>(time(nullptr)) : 0;
    entry.receivedMs = nowMs;

    if (xSemaphoreTake(outboxMutex, portMAX_DELAY) == pdTRUE) {
        bool pushed = outbox.push(entry);
        syncRtcOutbox();
        counters.received++;
        if (!pushed) {
            counters.bufferDropped++;
        }
        xSemaphoreGive(outboxMutex);
    }
    markHealthyBoot();
}

void snapshotCountersIfDue() {
    uint32_t now = millis();
    if (now - lastCounterSaveMs >= 300000uL) {
        lastCounterSaveMs = now;
        saveCounters();
    }
}

void outboxSenderTask(void*) {
    static PendingReading batch[BATCH_FLUSH_MAX];
    static IngestReading ingestBatch[BATCH_FLUSH_MAX];
    static char payload[INGEST_BUFFER_SIZE];

    uint8_t retryIndex = 0;
    lastSenderHeartbeatMs = millis();

    while (true) {
        lastSenderHeartbeatMs = millis();
        if (WiFi.status() != WL_CONNECTED) {
            ensureWifi();
            retryIndex = 0;
            if (WiFi.status() != WL_CONNECTED) {
                delayWithHeartbeat(lastSenderHeartbeatMs, WIFI_RECONNECT_DELAY_MS);
                continue;
            }
        }

        size_t available = 0;
        if (xSemaphoreTake(outboxMutex, portMAX_DELAY) == pdTRUE) {
            available = outbox.count();
            xSemaphoreGive(outboxMutex);
        }
        if (available == 0) {
            retryIndex = 0;
            snapshotCountersIfDue();
            delayWithHeartbeat(lastSenderHeartbeatMs, FLUSH_IDLE_DELAY_MS);
            continue;
        }

        size_t batchCount = 0;
        size_t limit = available < BATCH_FLUSH_MAX ? available : BATCH_FLUSH_MAX;
        if (xSemaphoreTake(outboxMutex, portMAX_DELAY) == pdTRUE) {
            for (size_t i = 0; i < limit; i++) {
                outbox.peekAt(i, batch[i]);
            }
            xSemaphoreGive(outboxMutex);
        }
        batchCount = limit;

        for (size_t i = 0; i < batchCount; i++) {
            ingestBatch[i].data = batch[i].data;
            ingestBatch[i].epoch = batch[i].receivedAt;
        }

        int payloadLength =
            buildBatchPayload(ingestBatch, batchCount, nodeId, payload, sizeof(payload));
        if (payloadLength <= 0) {
            // Payload builder should never fail with valid inputs, drop as a
            // safety net so the outbox drains.
            if (xSemaphoreTake(outboxMutex, portMAX_DELAY) == pdTRUE) {
                PendingReading discard;
                for (size_t i = 0; i < batchCount; i++) {
                    outbox.pop(discard);
                }
                syncRtcOutbox();
                counters.dropped += batchCount;
                xSemaphoreGive(outboxMutex);
            }
            retryIndex = 0;
            continue;
        }

        int httpCode = postToIngest(INGEST_URL, payload);

        if (httpCode >= 200 && httpCode < 300) {
            if (xSemaphoreTake(outboxMutex, portMAX_DELAY) == pdTRUE) {
                PendingReading discard;
                for (size_t i = 0; i < batchCount; i++) {
                    outbox.pop(discard);
                }
                syncRtcOutbox();
                counters.lastHttpCode = httpCode;
                counters.sent += batchCount;
                xSemaphoreGive(outboxMutex);
            }
            retryIndex = 0;
            markHealthyBoot();
        } else if (isTransientFailure(httpCode)) {
            if (xSemaphoreTake(outboxMutex, portMAX_DELAY) == pdTRUE) {
                counters.lastHttpCode = httpCode;
                xSemaphoreGive(outboxMutex);
            }
            // Keep the batch buffered and retry with backoff.
            retryIndex++;
            uint32_t delayMs = backoffDelayMs(backoff, retryIndex);
            delayWithHeartbeat(lastSenderHeartbeatMs, delayMs);
        } else {
            // Permanent client error, the payload will never succeed. Drop the
            // oldest entry and try the rest on the next pass.
            if (xSemaphoreTake(outboxMutex, portMAX_DELAY) == pdTRUE) {
                PendingReading discard;
                outbox.pop(discard);
                syncRtcOutbox();
                counters.lastHttpCode = httpCode;
                counters.dropped++;
                xSemaphoreGive(outboxMutex);
            }
            retryIndex = 0;
        }
        snapshotCountersIfDue();
    }
}

void reinitRadio() {
    ARQ::Status radioStatus = arq.initializeCC1101();
    if (xSemaphoreTake(outboxMutex, portMAX_DELAY) == pdTRUE) {
        counters.radioResyncs++;
        xSemaphoreGive(outboxMutex);
    }
    if (radioStatus != ARQ::STATUS_OK) {
        Serial.printf("[ERROR] Radio re-init failed, status %d\n", radioStatus);
        storeLastError("radio_reinit");
    } else if (arq.startReceiveData(BASE_STATION_ADDR) != ARQ::STATUS_OK) {
        Serial.println("[ERROR] Failed to restart receiving");
        storeLastError("radio_rx");
    } else {
        Serial.println("[OK] Radio re-initialized and listening");
    }
    lastRadioRxMs = millis();
}

bool tasksHealthy(uint32_t nowMs) {
    if (nowMs - lastSenderHeartbeatMs > WATCHDOG_TIMEOUT_MS) {
        Serial.println("[ERROR] OutboxSender task stalled");
        storeLastError("sender_wdt");
        return false;
    }
    if (nowMs - lastNtpHeartbeatMs > WATCHDOG_TIMEOUT_MS) {
        Serial.println("[ERROR] NtpSync task stalled");
        storeLastError("ntp_wdt");
        return false;
    }
    return true;
}

void setup() {
    Serial.begin(115200);
    delay(200);
    Serial.println("\n==================================");
    Serial.println("  Base Station Starting");
    Serial.println("==================================");

    // Disable Bluetooth to save power and reduce 2.4 GHz coexistence interference.
    btStop();

    esp_task_wdt_config_t config = {};
    config.timeout_ms = WATCHDOG_TIMEOUT_MS;
    config.idle_core_mask = 0;
    config.trigger_panic = true;
    esp_task_wdt_init(&config);
    esp_task_wdt_add(NULL);

    loadBootCounter();

    if (!sanitizeNodeId(NODE_ID, nodeId, sizeof(nodeId))) {
        snprintf(nodeId, sizeof(nodeId), "station");
    }

    outboxMutex = xSemaphoreCreateMutex();
    initRtcOutbox();
    loadCounters();

    uint32_t nowMs = millis();
    lastRadioRxMs = nowMs;
    lastSenderHeartbeatMs = nowMs;
    lastNtpHeartbeatMs = nowMs;

    ARQ::Status radioStatus = arq.initializeCC1101();
    if (radioStatus != ARQ::STATUS_OK) {
        Serial.printf("[ERROR] Radio initialization failed, status %d\n", radioStatus);
        storeLastError("radio_init");
    } else {
        Serial.println("[OK] Radio initialized");
        if (arq.startReceiveData(BASE_STATION_ADDR) != ARQ::STATUS_OK) {
            Serial.println("[ERROR] Failed to start receiving");
            storeLastError("radio_rx");
        } else {
            Serial.println("[OK] Listening mode active");
        }
    }

    xTaskCreatePinnedToCore(ntpSyncTask, "NtpSync", 4096, nullptr, 1, nullptr, 0);
    xTaskCreatePinnedToCore(outboxSenderTask, "OutboxSender", 8192, nullptr, 1, nullptr, 0);
    Serial.printf("Outbox capacity %d, node id %s\n", OUTBOX_CAPACITY, nodeId);
}

void loop() {
    uint32_t now = millis();
    if (tasksHealthy(now)) {
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

    // Re-sync the CC1101 receiver if no valid frame has arrived for 5 minutes.
    if (now - lastRadioRxMs >= RADIO_SILENCE_REINIT_MS) {
        Serial.println("[WARN] Radio silence threshold exceeded, re-initializing CC1101");
        reinitRadio();
    } else if (arq.getState() != ARQ::RadioState::RECEIVING) {
        // Keep the receiver armed. A crash or re-init leaves the radio idle.
        arq.startReceiveData(BASE_STATION_ADDR);
    }

    static uint32_t lastDiag = 0;
    if (now - lastDiag >= 10000uL) {
        lastDiag = now;
        Counters snap;
        size_t queued = 0;
        if (xSemaphoreTake(outboxMutex, portMAX_DELAY) == pdTRUE) {
            snap = counters;
            queued = outbox.count();
            xSemaphoreGive(outboxMutex);
        }
        Serial.printf(
            "[Diag] up=%lu boots=%lu outbox=%u rssi=%d lqi=%u state=%d resyncs=%lu http=%d | "
            "recv=%lu sent=%lu drop=%lu inv=%lu bufdrop=%lu\n",
            now / 1000uL, static_cast<unsigned long>(bootCount), static_cast<unsigned>(queued),
            arq.getLastRSSI(), arq.getLastLQI(), static_cast<int>(arq.getState()),
            static_cast<unsigned long>(snap.radioResyncs), snap.lastHttpCode,
            static_cast<unsigned long>(snap.received), static_cast<unsigned long>(snap.sent),
            static_cast<unsigned long>(snap.dropped),
            static_cast<unsigned long>(snap.framesInvalid),
            static_cast<unsigned long>(snap.bufferDropped));
    }

    // Pace the receiver polling loop to reduce SPI bus chatter and Core 1 power draw
    // while responding well within the 1000 ms ARQ ACK window.
    vTaskDelay(pdMS_TO_TICKS(5));
}
