#include "wifi_ntp_http.h"

#include <Arduino.h>
#include <HTTPClient.h>
#include <WiFi.h>
#include <WiFiClientSecure.h>
#include <cstring>
#include <freertos/FreeRTOS.h>
#include <freertos/task.h>
#include <time.h>

#include "config.h"
#include "rtc_outbox.h"

namespace {

struct NtpTaskArgs {
    RtcOutbox* outbox;
    volatile bool* clockSynced;
    volatile uint32_t* heartbeatMs;
};

NtpTaskArgs s_ntpArgs = {};

void ntpSyncTask(void* param) {
    auto* args = static_cast<NtpTaskArgs*>(param);
    RtcOutbox& outbox = *args->outbox;
    volatile bool& clockSynced = *args->clockSynced;
    volatile uint32_t& heartbeatMs = *args->heartbeatMs;

    heartbeatMs = millis();
    while (true) {
        heartbeatMs = millis();
        if (WiFi.status() != WL_CONNECTED) {
            delayWithHeartbeat(heartbeatMs, WIFI_RECONNECT_DELAY_MS);
            continue;
        }
        if (syncClockOnce(heartbeatMs, NTP_SYNC_TIMEOUT_MS)) {
            uint32_t nowEpoch = static_cast<uint32_t>(time(nullptr));
            uint32_t nowMs = millis();
            clockSynced = true;
            outbox.backfillTimestamps(nowEpoch, nowMs);
            delayWithHeartbeat(heartbeatMs, NTP_RESYNC_INTERVAL_MS);
        } else {
            if (!clockSynced) {
                Serial.println(
                    "[WARN] NTP unreachable, deferring outbox delivery until clock syncs");
            } else {
                Serial.println("[WARN] NTP re-sync failed, keeping the current clock");
            }
            delayWithHeartbeat(heartbeatMs, NTP_RETRY_INTERVAL_MS);
        }
    }
}

} // namespace

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

void ensureWifi(volatile uint32_t& heartbeatMs) {
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
        heartbeatMs = millis();
        vTaskDelay(pdMS_TO_TICKS(200));
    }
    heartbeatMs = millis();
    if (WiFi.status() == WL_CONNECTED) {
        Serial.printf("[OK] WiFi connected, ip=%s\n", WiFi.localIP().toString().c_str());
    } else {
        Serial.println("[WARN] WiFi connect failed");
    }
}

bool syncClockOnce(volatile uint32_t& heartbeatMs, uint32_t timeoutMs) {
    configTime(0, 0, NTP_SERVER);
    uint32_t started = millis();
    while (time(nullptr) < kClockSyncedEpoch && millis() - started < timeoutMs) {
        heartbeatMs = millis();
        vTaskDelay(pdMS_TO_TICKS(200));
    }
    heartbeatMs = millis();
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

int postToIngest(const char* url, const char* body, volatile uint32_t& heartbeatMs) {
    heartbeatMs = millis();
    WiFiClientSecure secureClient;
    HTTPClient http;
    http.setTimeout(HTTP_TIMEOUT_MS);
#if INGEST_HTTPS
    // Certificate verification is disabled (setInsecure()); pin a root CA/cert for production.
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
    heartbeatMs = millis();
    return code;
}

void startNtpSyncTask(RtcOutbox& outbox, volatile bool& clockSynced,
                      volatile uint32_t& ntpHeartbeatMs) {
    s_ntpArgs.outbox = &outbox;
    s_ntpArgs.clockSynced = &clockSynced;
    s_ntpArgs.heartbeatMs = &ntpHeartbeatMs;
    xTaskCreatePinnedToCore(ntpSyncTask, "NtpSync", 4096, &s_ntpArgs, 1, nullptr, 0);
}
