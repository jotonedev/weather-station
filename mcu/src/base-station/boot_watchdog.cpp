#include "boot_watchdog.h"

#include <Arduino.h>
#include <Preferences.h>
#include <esp_system.h>
#include <esp_task_wdt.h>

#include "config.h"

namespace {

Preferences s_prefs;
uint32_t s_bootCount = 0;
bool s_healthyBootMarked = false;
bool s_stallLogged = false;

} // namespace

void setupWatchdog() {
    esp_task_wdt_config_t config = {};
    config.timeout_ms = WATCHDOG_TIMEOUT_MS;
    config.idle_core_mask = 0;
    config.trigger_panic = true;
    esp_task_wdt_init(&config);
    esp_task_wdt_add(nullptr);
}

void loadBootCounter() {
    Serial.printf("[Boot] reset_reason=%d\n", static_cast<int>(esp_reset_reason()));
    if (!s_prefs.begin(NVS_NAMESPACE, false)) {
        return;
    }
    String lastErr = s_prefs.getString(NVS_KEY_LAST_ERR, "");
    if (lastErr.length() > 0) {
        Serial.printf("[Boot] last_err=%s\n", lastErr.c_str());
        s_prefs.remove(NVS_KEY_LAST_ERR);
    }
    s_bootCount = s_prefs.getUInt(NVS_KEY_BOOTS, 0) + 1;
    s_prefs.putUInt(NVS_KEY_BOOTS, s_bootCount);
    s_prefs.end();
    if (s_bootCount > BOOT_CRASH_THRESHOLD) {
        Serial.printf("[WARN] Boot counter %lu, possible crash loop\n",
                      static_cast<unsigned long>(s_bootCount));
    }
}

void markHealthyBoot() {
    if (s_healthyBootMarked) {
        return;
    }
    s_healthyBootMarked = true;
    if (s_prefs.begin(NVS_NAMESPACE, false)) {
        s_prefs.putUInt(NVS_KEY_BOOTS, 0);
        s_prefs.end();
    }
}

void storeLastError(const char* tag) {
    if (s_prefs.begin(NVS_NAMESPACE, false)) {
        s_prefs.putString(NVS_KEY_LAST_ERR, tag);
        s_prefs.end();
    }
}

uint32_t getBootCount() {
    return s_bootCount;
}

bool tasksHealthy(uint32_t senderHb, uint32_t ntpHb) {
    uint32_t nowMs = millis();
    if (static_cast<int32_t>(nowMs - senderHb) > static_cast<int32_t>(WATCHDOG_TIMEOUT_MS)) {
        if (!s_stallLogged) {
            Serial.println("[ERROR] OutboxSender task stalled");
            storeLastError("sender_wdt");
            s_stallLogged = true;
        }
        return false;
    }
    if (static_cast<int32_t>(nowMs - ntpHb) > static_cast<int32_t>(WATCHDOG_TIMEOUT_MS)) {
        if (!s_stallLogged) {
            Serial.println("[ERROR] NtpSync task stalled");
            storeLastError("ntp_wdt");
            s_stallLogged = true;
        }
        return false;
    }
    s_stallLogged = false;
    return true;
}
