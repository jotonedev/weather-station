#include <Arduino.h>
#include <Preferences.h>
#include <WiFi.h>
#include <WiFiClientSecure.h>
#include <HTTPClient.h>
#include <esp_task_wdt.h>
#include <time.h>

#include <ARQ.h>
#include <Ingest.h>
#include <Outbox.h>
#include <Weather.h>
#include <cc1101.h>
#include <RadioAdapter.h>

#include "config.h"

CC1101::Radio radio(RADIO_CS_PIN, SCK_PIN, MISO_PIN, MOSI_PIN, RADIO_GDO0_PIN);
CC1101RadioAdapter radioAdapter(radio);
ARQ::ARQ arq(radioAdapter, BASE_STATION_ADDR);

static Outbox* outbox = nullptr;
static SemaphoreHandle_t outboxMutex = nullptr;
static char nodeId[32] = "unknown";

static Preferences prefs;
static uint32_t lastCounterSaveMs = 0;

struct Counters {
    uint32_t received = 0;
    uint32_t sent = 0;
    uint32_t dropped = 0;
    uint32_t framesInvalid = 0;
    uint32_t bufferDropped = 0;
    int lastHttpCode = 0;
};
static Counters counters;

static BackoffPolicy backoff = {BACKOFF_MIN_MS, BACKOFF_MAX_MS, BACKOFF_MULTIPLIER, 0};

bool timeSynced() {
    return time(nullptr) >= 1600000000;  // roughly 2020
}

void syncTime() {
    configTime(0, 0, NTP_SERVER);
    uint32_t started = millis();
    while (!timeSynced() && millis() - started < NTP_TIMEOUT_MS) {
        delay(200);
    }
    if (timeSynced()) {
        time_t current = time(nullptr);
        struct tm now;
        gmtime_r(&current, &now);
        Serial.printf("[OK] NTP synced %04d-%02d-%02dT%02d:%02d:%02dZ\n",
                      now.tm_year + 1900, now.tm_mon + 1, now.tm_mday,
                      now.tm_hour, now.tm_min, now.tm_sec);
    } else {
        Serial.println("[WARN] NTP not synced, readings will be sent untimestamped");
    }
}

void ensureWifi() {
    static uint32_t lastAttempt = 0;
    if (WiFi.status() == WL_CONNECTED) {
        return;
    }
    uint32_t now = millis();
    if (now - lastAttempt < WIFI_RECONNECT_DELAY_MS) {
        return;
    }
    lastAttempt = now;

    WiFi.mode(WIFI_STA);
    WiFi.begin(WIFI_SSID, WIFI_PASSWORD);

    uint32_t started = millis();
    while (WiFi.status() != WL_CONNECTED && millis() - started < WIFI_CONNECT_TIMEOUT_MS) {
        delay(200);
    }
    if (WiFi.status() == WL_CONNECTED) {
        Serial.printf("[OK] WiFi connected, ip=%s\n", WiFi.localIP().toString().c_str());
    } else {
        Serial.println("[WARN] WiFi connect failed");
    }
}

int postToIngest(const char* url, const char* body) {
    bool isHttps = (strncmp(url, "https://", 8) == 0);
    WiFiClientSecure secureClient;
    HTTPClient http;
    http.setTimeout(HTTP_TIMEOUT_MS);
    if (isHttps) {
        // Trust the Arduino core cert bundle. Pin a specific cert for production.
        secureClient.setInsecure();
        http.begin(secureClient, url);
    } else {
        http.begin(url);
    }
    http.addHeader("Content-Type", "application/json");
    int code = http.POST(body);
    http.end();
    return code;
}

bool isTransientFailure(int httpCode) {
    if (httpCode < 0) {
        return true;  // network level error
    }
    if (httpCode >= 500) {
        return true;
    }
    return httpCode == 408 || httpCode == 429;
}

void loadCounters() {
    if (!prefs.begin(NVS_NAMESPACE, false)) {
        return;
    }
    counters.received = prefs.getUInt(NVS_KEY_RECEIVED, 0);
    counters.sent = prefs.getUInt(NVS_KEY_SENT, 0);
    counters.dropped = prefs.getUInt(NVS_KEY_DROPPED, 0);
    counters.framesInvalid = prefs.getUInt(NVS_KEY_FRAMES_INVALID, 0);
    counters.bufferDropped = prefs.getUInt(NVS_KEY_BUFFER_DROPPED, 0);
    counters.lastHttpCode = prefs.getInt(NVS_KEY_LAST_HTTP, 0);
    prefs.end();
}

void saveCounters() {
    if (!prefs.begin(NVS_NAMESPACE, false)) {
        return;
    }
    prefs.putUInt(NVS_KEY_RECEIVED, counters.received);
    prefs.putUInt(NVS_KEY_SENT, counters.sent);
    prefs.putUInt(NVS_KEY_DROPPED, counters.dropped);
    prefs.putUInt(NVS_KEY_FRAMES_INVALID, counters.framesInvalid);
    prefs.putUInt(NVS_KEY_BUFFER_DROPPED, counters.bufferDropped);
    prefs.putInt(NVS_KEY_LAST_HTTP, counters.lastHttpCode);
    prefs.end();
}

void handleMessage(const ARQ::Message& message, const uint8_t* frame) {
    if (message.type != ARQ::DATA_FRAME || message.source != CONTROL_NODE_ADDR) {
        return;
    }
    if (message.length != sizeof(WeatherData)) {
        counters.framesInvalid++;
        return;
    }
    WeatherData data;
    if (!weather::unpackWeatherData(frame, message.length, data)) {
        counters.framesInvalid++;
        return;
    }

    PendingReading entry = {};
    entry.data = data;
    entry.sequence = message.sequence;
    entry.receivedAt = timeSynced() ? static_cast<uint32_t>(time(nullptr)) : 0;

    if (xSemaphoreTake(outboxMutex, portMAX_DELAY) == pdTRUE) {
        bool pushed = outbox->push(entry);
        xSemaphoreGive(outboxMutex);
        counters.received++;
        if (!pushed) {
            counters.bufferDropped++;
        }
    }
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

    while (true) {
        if (WiFi.status() != WL_CONNECTED) {
            ensureWifi();
            if (WiFi.status() == WL_CONNECTED && !timeSynced()) {
                syncTime();
            }
            retryIndex = 0;
            vTaskDelay(pdMS_TO_TICKS(WIFI_RECONNECT_DELAY_MS));
            continue;
        }
        if (!timeSynced()) {
            syncTime();
        }

        size_t available = 0;
        if (xSemaphoreTake(outboxMutex, portMAX_DELAY) == pdTRUE) {
            available = outbox->count();
            xSemaphoreGive(outboxMutex);
        }
        if (available == 0) {
            retryIndex = 0;
            snapshotCountersIfDue();
            vTaskDelay(pdMS_TO_TICKS(FLUSH_IDLE_DELAY_MS));
            continue;
        }

        size_t batchCount = 0;
        size_t limit = available < BATCH_FLUSH_MAX ? available : BATCH_FLUSH_MAX;
        if (xSemaphoreTake(outboxMutex, portMAX_DELAY) == pdTRUE) {
            for (size_t i = 0; i < limit; i++) {
                outbox->peekAt(i, batch[i]);
            }
            xSemaphoreGive(outboxMutex);
        }
        batchCount = limit;

        for (size_t i = 0; i < batchCount; i++) {
            ingestBatch[i].data = batch[i].data;
            ingestBatch[i].epoch = batch[i].receivedAt;
        }

        int payloadLength = buildBatchPayload(ingestBatch, batchCount, nodeId, payload,
                                              sizeof(payload));
        if (payloadLength <= 0) {
            // Payload builder should never fail with valid inputs, drop as a
            // safety net so the outbox drains.
            if (xSemaphoreTake(outboxMutex, portMAX_DELAY) == pdTRUE) {
                PendingReading discard;
                for (size_t i = 0; i < batchCount; i++) {
                    outbox->pop(discard);
                }
                xSemaphoreGive(outboxMutex);
            }
            counters.dropped += batchCount;
            retryIndex = 0;
            continue;
        }

        int httpCode = postToIngest(INGEST_URL, payload);
        counters.lastHttpCode = httpCode;

        if (httpCode >= 200 && httpCode < 300) {
            if (xSemaphoreTake(outboxMutex, portMAX_DELAY) == pdTRUE) {
                PendingReading discard;
                for (size_t i = 0; i < batchCount; i++) {
                    outbox->pop(discard);
                }
                xSemaphoreGive(outboxMutex);
            }
            counters.sent += batchCount;
            retryIndex = 0;
        } else if (isTransientFailure(httpCode)) {
            // Keep the batch buffered and retry with backoff.
            retryIndex++;
            uint32_t delayMs = backoffDelayMs(backoff, retryIndex);
            vTaskDelay(pdMS_TO_TICKS(delayMs));
        } else {
            // Permanent client error, the payload will never succeed. Drop the
            // oldest entry and try the rest on the next pass.
            if (xSemaphoreTake(outboxMutex, portMAX_DELAY) == pdTRUE) {
                PendingReading discard;
                outbox->pop(discard);
                xSemaphoreGive(outboxMutex);
            }
            counters.dropped++;
            retryIndex = 0;
        }
        snapshotCountersIfDue();
    }
}

void setup() {
    Serial.begin(115200);
    delay(200);
    Serial.println("\n==================================");
    Serial.println("  Base Station Starting");
    Serial.println("==================================");

    esp_task_wdt_config_t config = {};
    config.timeout_ms = 10000uL;
    config.idle_core_mask = 0;
    config.trigger_panic = true;
    esp_task_wdt_init(&config);
    esp_task_wdt_add(NULL);

    if (!sanitizeNodeId(NODE_ID, nodeId, sizeof(nodeId))) {
        snprintf(nodeId, sizeof(nodeId), "station");
    }

    outbox = new Outbox(OUTBOX_CAPACITY);
    outboxMutex = xSemaphoreCreateMutex();
    loadCounters();

    ARQ::Status radioStatus = arq.initializeCC1101();
    if (radioStatus != ARQ::STATUS_OK) {
        Serial.printf("[ERROR] Radio initialization failed, status %d\n", radioStatus);
    } else {
        Serial.println("[OK] Radio initialized");
        if (arq.startReceiveData(BASE_STATION_ADDR) != ARQ::STATUS_OK) {
            Serial.println("[ERROR] Failed to start receiving");
        } else {
            Serial.println("[OK] Listening mode active");
        }
    }

    xTaskCreatePinnedToCore(outboxSenderTask, "OutboxSender", 8192, nullptr, 1, nullptr, 0);
    Serial.printf("Outbox capacity %d, node id %s\n", OUTBOX_CAPACITY, nodeId);
}

void loop() {
    esp_task_wdt_reset();

    arq.poll();

    if (arq.available()) {
        uint8_t frame[weather::kMaxWirePayload];
        ARQ::Message message;
        if (arq.readMessage(message, frame, sizeof(frame)) == ARQ::STATUS_OK) {
            handleMessage(message, frame);
        }
    }

    // Keep the receiver armed. A crash or re-init leaves the radio idle.
    if (arq.getState() != ARQ::RadioState::RECEIVING) {
        arq.startReceiveData(BASE_STATION_ADDR);
    }

    static uint32_t lastDiag = 0;
    uint32_t now = millis();
    if (now - lastDiag >= 10000uL) {
        lastDiag = now;
        Serial.printf("[Diag] outbox=%u rssi=%d lqi=%u state=%d http=%d | recv=%lu sent=%lu "
                      "drop=%lu inv=%lu bufdrop=%lu\n",
                      outbox->count(), arq.getLastRSSI(), arq.getLastLQI(),
                      static_cast<int>(arq.getState()), counters.lastHttpCode,
                      static_cast<unsigned long>(counters.received),
                      static_cast<unsigned long>(counters.sent),
                      static_cast<unsigned long>(counters.dropped),
                      static_cast<unsigned long>(counters.framesInvalid),
                      static_cast<unsigned long>(counters.bufferDropped));
    }
}
