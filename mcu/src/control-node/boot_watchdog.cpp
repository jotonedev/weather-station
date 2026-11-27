#include "boot_watchdog.h"

#include <Arduino.h>
#include <Preferences.h>
#include <esp_pm.h>
#include <esp_system.h>
#include <esp_task_wdt.h>

#include "constants.h"

namespace {

Preferences prefs;
uint32_t bootCount = 0;
uint8_t lastResetReason = 0;
bool healthyBootMarked = false;

} // namespace

void setupPowerManagement() {
    esp_pm_config_t pmConfig = {
        .max_freq_mhz = 80,
        .min_freq_mhz = 80,
        .light_sleep_enable = true,
    };
    esp_err_t pmErr = esp_pm_configure(&pmConfig);
    if (pmErr == ESP_OK) {
        Serial.println("[OK] Automatic light sleep configured");
    } else if (pmErr == ESP_ERR_NOT_SUPPORTED) {
        // Prebuilt Arduino ESP32 core disables CONFIG_FREERTOS_USE_TICKLESS_IDLE.
        Serial.println("[INFO] Automatic light sleep unsupported in prebuilt Arduino core "
                       "(requires tickless idle)");
    } else {
        Serial.printf("[WARN] Power management config returned %d\n", pmErr);
    }
}

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
    const esp_reset_reason_t resetReason = esp_reset_reason();
    lastResetReason = static_cast<uint8_t>(resetReason);
    if (!prefs.begin(NVS_NAMESPACE, false)) {
        return;
    }
    const String lastErr = prefs.getString(NVS_KEY_LAST_ERR, "");
    if (lastErr.length() > 0) {
        prefs.remove(NVS_KEY_LAST_ERR);
    }
    bootCount = prefs.getUInt(NVS_KEY_BOOTS, 0) + 1;
    prefs.putUInt(NVS_KEY_BOOTS, bootCount);
    prefs.end();

    Serial.printf("[INFO] Boot #%lu resetReason=%d\n", static_cast<unsigned long>(bootCount),
                  static_cast<int>(resetReason));
    if (lastErr.length() > 0) {
        Serial.printf("[WARN] Previous last error: %s\n", lastErr.c_str());
    }
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

uint32_t getBootCount() {
    return bootCount;
}

uint8_t getLastResetReason() {
    return lastResetReason;
}

bool tasksHealthy(uint32_t anemoHb, uint32_t vaneHb, uint32_t rainHb) {
    const uint32_t nowMs = millis();
    static bool stallLogged = false;

    if (static_cast<int32_t>(nowMs - anemoHb) > static_cast<int32_t>(WATCHDOG_TIMEOUT_MS)) {
        if (!stallLogged) {
            Serial.println("[ERROR] Anemometer task stalled");
            storeLastError("anemo_wdt");
            stallLogged = true;
        }
        return false;
    }
    if (static_cast<int32_t>(nowMs - vaneHb) > static_cast<int32_t>(WATCHDOG_TIMEOUT_MS)) {
        if (!stallLogged) {
            Serial.println("[ERROR] WindVane task stalled");
            storeLastError("vane_wdt");
            stallLogged = true;
        }
        return false;
    }
    if (static_cast<int32_t>(nowMs - rainHb) > static_cast<int32_t>(WATCHDOG_TIMEOUT_MS)) {
        if (!stallLogged) {
            Serial.println("[ERROR] RainMeter task stalled");
            storeLastError("rain_wdt");
            stallLogged = true;
        }
        return false;
    }
    stallLogged = false;
    return true;
}
