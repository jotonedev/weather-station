#include <Arduino.h>
#include <Wire.h>
#include <cmath>
#include <cstdint>
#include <cstdlib>

#include <AS5600.h>

#include "../control-node/constants.h"

#ifndef ANEMO_TEST_SDA_PIN
#ifdef CONFIG_IDF_TARGET_ESP32S3
#define ANEMO_TEST_SDA_PIN SENSOR_I2C_SDA_PIN
#else
#define ANEMO_TEST_SDA_PIN 21
#endif
#endif

#ifndef ANEMO_TEST_SCL_PIN
#ifdef CONFIG_IDF_TARGET_ESP32S3
#define ANEMO_TEST_SCL_PIN SENSOR_I2C_SCL_PIN
#else
#define ANEMO_TEST_SCL_PIN 22
#endif
#endif

namespace {

constexpr int kI2cSdaPin = ANEMO_TEST_SDA_PIN;
constexpr int kI2cSclPin = ANEMO_TEST_SCL_PIN;

constexpr uint8_t kStatusMagnetHigh = 0x08u;
constexpr uint8_t kStatusMagnetLow = 0x10u;
constexpr uint8_t kStatusMagnetDetect = 0x20u;

constexpr uint32_t kSamplePeriodMs = 20uL;
constexpr uint32_t kReportPeriodMs = 500uL;
constexpr size_t kVelocityWindowSize = 10;
constexpr uint16_t kSectorCount = 12;
constexpr int32_t kCountsPerRev = 4096;

AS5600 as5600(&Wire);

struct RunningStat {
    uint16_t minVal = UINT16_MAX;
    uint16_t maxVal = 0;
    uint32_t count = 0;
    double mean = 0.0;
    double m2 = 0.0;

    void reset() {
        minVal = UINT16_MAX;
        maxVal = 0;
        count = 0;
        mean = 0.0;
        m2 = 0.0;
    }

    void update(uint16_t value) {
        if (value < minVal) {
            minVal = value;
        }
        if (value > maxVal) {
            maxVal = value;
        }
        count++;
        double d1 = static_cast<double>(value) - mean;
        mean += d1 / static_cast<double>(count);
        double d2 = static_cast<double>(value) - mean;
        m2 += d1 * d2;
    }

    uint16_t span() const {
        if (count == 0) {
            return 0;
        }
        return static_cast<uint16_t>(maxVal - minVal);
    }

    double stdDev() const {
        if (count < 2) {
            return 0.0;
        }
        return std::sqrt(m2 / static_cast<double>(count - 1));
    }
};

struct Sample {
    int64_t counts;
    uint32_t timeUs;
};

struct HealthStats {
    bool connected = false;
    int lastErr = AS5600_OK;
    uint16_t rawAngle = 0;
    uint16_t filtAngle = 0;
    uint8_t status = 0;
    uint8_t agc = 0;
    uint16_t magnitude = 0;
    bool md = false;
    bool ml = false;
    bool mh = false;

    RunningStat agcStat;
    RunningStat magStat;

    uint32_t i2cErrors = 0;
    uint32_t mdLostCount = 0;
    uint32_t mlHits = 0;
    uint32_t mhHits = 0;
    uint32_t jumpCount = 0;

    uint16_t sectorMask = 0;
    int32_t lastValidRaw = -1;
    int64_t cumulativeCounts = 0;
    Sample history[kVelocityWindowSize] = {};
    size_t historyIdx = 0;
    size_t historyCount = 0;
    float smoothedRpm = 0.0f;

    void resetVelocity() {
        lastValidRaw = -1;
        historyIdx = 0;
        historyCount = 0;
        smoothedRpm = 0.0f;
    }

    void resetStats() {
        agcStat.reset();
        magStat.reset();
        i2cErrors = 0;
        mdLostCount = 0;
        mlHits = 0;
        mhHits = 0;
        jumpCount = 0;
        sectorMask = 0;
        cumulativeCounts = 0;
        resetVelocity();
    }
};

HealthStats stats;
uint32_t lastSampleMs = 0;
uint32_t lastReportMs = 0;

const char* describeI2cAddress(uint8_t addr) {
    switch (addr) {
    case 0x36:
        return "AS5600 (anemometer)";
    case 0x38:
        return "AHT20";
    case 0x77:
        return "BMP180";
    default:
        return "unknown";
    }
}

void scanI2cBus() {
    Serial.printf("[I2C] Scanning Wire (SDA=GPIO%d, SCL=GPIO%d, %lu Hz)...\n", kI2cSdaPin,
                  kI2cSclPin, static_cast<unsigned long>(I2C_CLOCK_HZ));
    uint8_t foundCount = 0;
    bool foundAs5600 = false;

    for (uint8_t addr = 0x08u; addr <= 0x77u; addr++) {
        Wire.beginTransmission(addr);
        uint8_t err = Wire.endTransmission();
        if (err == 0) {
            foundCount++;
            if (addr == AS5600_DEFAULT_ADDRESS) {
                foundAs5600 = true;
            }
            Serial.printf("  - 0x%02X : %s\n", static_cast<unsigned int>(addr),
                          describeI2cAddress(addr));
        }
    }

    if (foundCount == 0) {
        Serial.println("[WARN] No I2C devices responded on Wire");
    }
    if (!foundAs5600) {
        Serial.printf(
            "[WARN] AS5600 (0x36) not found! Check SDA=GPIO%d, SCL=GPIO%d, 3.3V, GND, pull-ups\n",
            kI2cSdaPin, kI2cSclPin);
    }
}

bool configureSensor() {
    if (!as5600.begin(AS5600_SW_DIRECTION_PIN)) {
        stats.lastErr = as5600.lastError();
        return false;
    }

    as5600.setDirection(AS5600_CLOCK_WISE);
    as5600.setHysteresis(AS5600_HYST_LSB3);
    if (as5600.lastError() != AS5600_OK) {
        return false;
    }
    as5600.setSlowFilter(AS5600_SLOW_FILT_8X);
    if (as5600.lastError() != AS5600_OK) {
        return false;
    }
    as5600.setFastFilter(AS5600_FAST_FILT_NONE);
    if (as5600.lastError() != AS5600_OK) {
        return false;
    }
    as5600.setPowerMode(AS5600_POWERMODE_NOMINAL);
    if (as5600.lastError() != AS5600_OK) {
        return false;
    }
    as5600.setWatchDog(AS5600_WATCHDOG_OFF);
    return as5600.lastError() == AS5600_OK;
}

template <typename Fn, typename T> bool readChecked(Fn fn, T& out, int& errOut) {
    out = fn();
    errOut = as5600.lastError();
    return errOut == AS5600_OK;
}

void dumpRegisters() {
    uint16_t conf = 0;
    uint8_t pm = 0;
    uint8_t hyst = 0;
    uint8_t sf = 0;
    uint8_t ff = 0;
    uint8_t wd = 0;
    uint8_t zmco = 0;
    uint16_t zpos = 0;
    uint16_t mpos = 0;
    uint16_t mang = 0;
    int err = AS5600_OK;

    bool ok = readChecked([]() { return as5600.getConfiguration(); }, conf, err) &&
              readChecked([]() { return as5600.getPowerMode(); }, pm, err) &&
              readChecked([]() { return as5600.getHysteresis(); }, hyst, err) &&
              readChecked([]() { return as5600.getSlowFilter(); }, sf, err) &&
              readChecked([]() { return as5600.getFastFilter(); }, ff, err) &&
              readChecked([]() { return as5600.getWatchDog(); }, wd, err) &&
              readChecked([]() { return as5600.getZMCO(); }, zmco, err) &&
              readChecked([]() { return as5600.getZPosition(); }, zpos, err) &&
              readChecked([]() { return as5600.getMPosition(); }, mpos, err) &&
              readChecked([]() { return as5600.getMaxAngle(); }, mang, err);

    if (!ok) {
        Serial.printf("[WARN] Failed to dump AS5600 registers (err=%d)\n", err);
        return;
    }

    Serial.printf("[REG] CONF=0x%04X (PM=%u HYST=%u SF=%u FF=%u WD=%u) OTP(ZMCO=%u ZPOS=%u MPOS=%u "
                  "MANG=%u)\n",
                  static_cast<unsigned int>(conf), static_cast<unsigned int>(pm),
                  static_cast<unsigned int>(hyst), static_cast<unsigned int>(sf),
                  static_cast<unsigned int>(ff), static_cast<unsigned int>(wd),
                  static_cast<unsigned int>(zmco), static_cast<unsigned int>(zpos),
                  static_cast<unsigned int>(mpos), static_cast<unsigned int>(mang));

    if (zmco > 0 || zpos != 0 || mpos != 0 || mang != 0) {
        Serial.println(
            "[WARN] Non-default OTP or range registers detected; angular range may be clamped");
    }
}

uint16_t countCoveredSectors(uint16_t mask) {
    uint16_t covered = 0;
    for (uint16_t i = 0; i < kSectorCount; i++) {
        if ((mask & static_cast<uint16_t>(1u << i)) != 0) {
            covered++;
        }
    }
    return covered;
}

float computeMagRipplePct(const RunningStat& magStat) {
    if (magStat.count == 0 || magStat.mean <= 0.0) {
        return 0.0f;
    }
    return (static_cast<float>(magStat.span()) / static_cast<float>(magStat.mean)) * 100.0f;
}

const char* evaluateVerdict(uint16_t coveredSectors, float magRipplePct) {
    if (!stats.connected) {
        return "FAIL: NO I2C";
    }
    if (!stats.md) {
        return "FAIL: NO MAGNET (missing, too far, or axial)";
    }
    if (stats.ml) {
        return "WARN: MAGNET TOO WEAK (move closer / check diametric)";
    }
    if (stats.mh) {
        return "WARN: MAGNET TOO STRONG (increase air gap)";
    }

    bool agcNearRail = false;
    if (stats.agcStat.maxVal <= 128u) {
        agcNearRail = (stats.agc < 15u || stats.agc > 113u);
    } else {
        agcNearRail = (stats.agc < 25u || stats.agc > 230u);
    }
    if (agcNearRail) {
        return "WARN: AGC NEAR RAIL (adjust magnet air gap)";
    }

    if (stats.agcStat.count >= 20u && (stats.agcStat.span() > 20u || magRipplePct > 25.0f)) {
        return "WARN: WOBBLE/OFF-CENTER (high AGC or Mag ripple)";
    }
    if (stats.i2cErrors > 0u || stats.mdLostCount > 0u || stats.mlHits > 0u || stats.mhHits > 0u ||
        stats.jumpCount > 0u) {
        return "WARN: PAST FAULTS (press 'r' and re-spin)";
    }
    if (coveredSectors < kSectorCount) {
        return "OK: SPIN 360 DEG TO COMPLETE CHECK";
    }
    return "OK: ASSEMBLY GOOD";
}

void sampleSensor(uint32_t nowUs) {
    if (!stats.connected) {
        if (!as5600.isConnected()) {
            stats.lastErr = AS5600_ERROR_I2C_READ_0;
            stats.i2cErrors++;
            stats.resetVelocity();
            return;
        }
        Serial.println("[INFO] AS5600 detected on I2C bus, configuring...");
        if (!configureSensor()) {
            stats.i2cErrors++;
            stats.resetVelocity();
            return;
        }
        dumpRegisters();
        stats.resetVelocity();
        stats.connected = true;
    }

    uint16_t raw = 0;
    uint16_t filt = 0;
    uint8_t status = 0;
    uint8_t agc = 0;
    uint16_t mag = 0;
    int err = AS5600_OK;

    bool ok = readChecked([]() { return as5600.rawAngle(); }, raw, err) &&
              readChecked([]() { return as5600.readAngle(); }, filt, err) &&
              readChecked([]() { return as5600.readStatus(); }, status, err) &&
              readChecked([]() { return as5600.readAGC(); }, agc, err) &&
              readChecked([]() { return as5600.readMagnitude(); }, mag, err);

    if (!ok) {
        stats.connected = false;
        stats.lastErr = err;
        stats.i2cErrors++;
        stats.resetVelocity();
        return;
    }

    stats.connected = true;
    stats.lastErr = AS5600_OK;
    stats.rawAngle = raw;
    stats.filtAngle = filt;
    stats.status = status;
    stats.agc = agc;
    stats.magnitude = mag;
    stats.md = (status & kStatusMagnetDetect) != 0u;
    stats.ml = (status & kStatusMagnetLow) != 0u;
    stats.mh = (status & kStatusMagnetHigh) != 0u;

    if (!stats.md) {
        stats.mdLostCount++;
    }
    if (stats.ml) {
        stats.mlHits++;
    }
    if (stats.mh) {
        stats.mhHits++;
    }

    if (!stats.md) {
        stats.resetVelocity();
        return;
    }

    stats.agcStat.update(agc);
    stats.magStat.update(mag);

    uint16_t sector =
        static_cast<uint16_t>((static_cast<uint32_t>(raw) * kSectorCount) / kCountsPerRev);
    if (sector < kSectorCount) {
        stats.sectorMask |= static_cast<uint16_t>(1u << sector);
    }

    if (stats.lastValidRaw < 0) {
        stats.lastValidRaw = static_cast<int32_t>(raw);
        stats.historyIdx = 0;
        stats.historyCount = 1;
        stats.history[0] = {stats.cumulativeCounts, nowUs};
        return;
    }

    int32_t delta = static_cast<int32_t>(raw) - stats.lastValidRaw;
    if (delta > 2048) {
        delta -= kCountsPerRev;
    } else if (delta < -2048) {
        delta += kCountsPerRev;
    }

    if (std::abs(delta) > 1024) {
        stats.jumpCount++;
    }

    stats.cumulativeCounts += delta;
    stats.lastValidRaw = static_cast<int32_t>(raw);

    stats.history[stats.historyIdx] = {stats.cumulativeCounts, nowUs};
    stats.historyIdx = (stats.historyIdx + 1) % kVelocityWindowSize;
    if (stats.historyCount < kVelocityWindowSize) {
        stats.historyCount++;
    }

    size_t oldestIdx = (stats.historyCount < kVelocityWindowSize) ? 0 : stats.historyIdx;
    int64_t dCounts = stats.cumulativeCounts - stats.history[oldestIdx].counts;
    uint32_t dtUs = nowUs - stats.history[oldestIdx].timeUs;

    float rawRpm = 0.0f;
    if (dtUs > 0 && std::abs(dCounts) > 3) {
        rawRpm = (static_cast<float>(dCounts) * 60.0f * 1e6f) /
                 (static_cast<float>(dtUs) * static_cast<float>(kCountsPerRev));
    }

    if (rawRpm == 0.0f && std::fabs(stats.smoothedRpm) < 0.2f) {
        stats.smoothedRpm = 0.0f;
    } else {
        stats.smoothedRpm = 0.30f * rawRpm + 0.70f * stats.smoothedRpm;
        if (std::fabs(stats.smoothedRpm) < 0.1f) {
            stats.smoothedRpm = 0.0f;
        }
    }
}

void printReport() {
    uint16_t coveredSectors = countCoveredSectors(stats.sectorMask);
    float magRipplePct = computeMagRipplePct(stats.magStat);
    const char* verdict = evaluateVerdict(coveredSectors, magRipplePct);

    if (!stats.connected) {
        Serial.printf("[%s] err=%d (check SDA=GPIO%d SCL=GPIO%d 3.3V) | faults: i2c=%lu\n", verdict,
                      stats.lastErr, kI2cSdaPin, kI2cSclPin,
                      static_cast<unsigned long>(stats.i2cErrors));
        return;
    }

    const char* dirLabel = "STOP";
    if (stats.smoothedRpm > 0.05f) {
        dirLabel = "CW";
    } else if (stats.smoothedRpm < -0.05f) {
        dirLabel = "CCW";
    }

    float angleDeg = static_cast<float>(stats.rawAngle) * AS5600_RAW_TO_DEGREES;
    float speedMs = std::fabs(stats.smoothedRpm) * ANEMOMETER_SPEED_FACTOR;
    float revolutions =
        static_cast<float>(stats.cumulativeCounts) / static_cast<float>(kCountsPerRev);

    uint16_t agcMin = (stats.agcStat.count > 0) ? stats.agcStat.minVal : 0;
    uint16_t agcMax = (stats.agcStat.count > 0) ? stats.agcStat.maxVal : 0;
    uint16_t magMin = (stats.magStat.count > 0) ? stats.magStat.minVal : 0;
    uint16_t magMax = (stats.magStat.count > 0) ? stats.magStat.maxVal : 0;

    char covBar[kSectorCount + 1];
    for (uint16_t i = 0; i < kSectorCount; i++) {
        covBar[i] = ((stats.sectorMask & static_cast<uint16_t>(1u << i)) != 0) ? '#' : '.';
    }
    covBar[kSectorCount] = '\0';

    Serial.printf(
        "[%s] angle=%6.2f deg (raw=%4u filt=%4u) | rpm=%+6.1f (%s) %.2f m/s | rev=%+.2f\n", verdict,
        angleDeg, static_cast<unsigned int>(stats.rawAngle),
        static_cast<unsigned int>(stats.filtAngle), stats.smoothedRpm, dirLabel, speedMs,
        revolutions);
    Serial.printf(
        "  status=0x%02X [MD=%u ML=%u MH=%u] | agc=%3u (min=%u max=%u span=%u sd=%.1f) | mag=%4u "
        "(min=%u max=%u rip=%.1f%%)\n",
        static_cast<unsigned int>(stats.status), stats.md ? 1u : 0u, stats.ml ? 1u : 0u,
        stats.mh ? 1u : 0u, static_cast<unsigned int>(stats.agc), static_cast<unsigned int>(agcMin),
        static_cast<unsigned int>(agcMax), static_cast<unsigned int>(stats.agcStat.span()),
        stats.agcStat.stdDev(), static_cast<unsigned int>(stats.magnitude),
        static_cast<unsigned int>(magMin), static_cast<unsigned int>(magMax), magRipplePct);
    Serial.printf(
        "  360cov=[%s] %u/12 | faults: i2c=%lu mdLoss=%lu ml=%lu mh=%lu jumps=%lu\n", covBar,
        static_cast<unsigned int>(coveredSectors), static_cast<unsigned long>(stats.i2cErrors),
        static_cast<unsigned long>(stats.mdLostCount), static_cast<unsigned long>(stats.mlHits),
        static_cast<unsigned long>(stats.mhHits), static_cast<unsigned long>(stats.jumpCount));
}

void handleSerialCommands() {
    while (Serial.available() > 0) {
        int ch = Serial.read();
        if (ch == 'r' || ch == 'R') {
            stats.resetStats();
            Serial.println("[INFO] Stats reset");
        } else if (ch == 's' || ch == 'S') {
            scanI2cBus();
            if (as5600.isConnected()) {
                dumpRegisters();
            }
        }
    }
}

void printBanner() {
    Serial.println("\n====================================================");
    Serial.println("  Anemometer AS5600 Assembly Test");
    Serial.println("====================================================");
    Serial.printf("Pins: SDA=GPIO%d, SCL=GPIO%d @ %lu Hz\n", kI2cSdaPin, kI2cSclPin,
                  static_cast<unsigned long>(I2C_CLOCK_HZ));
    Serial.println("Healthy targets (3.3V): MD=1 ML=0 MH=0 | AGC ~32..96 (ideal ~64)");
    Serial.println("Healthy spin check: AGC span <20, Mag ripple <20%, 12/12 sectors, 0 faults");
    Serial.println("Commands: 'r' = reset session stats | 's' = rescan I2C & dump registers");
    Serial.println("----------------------------------------------------");
}

} // namespace

void setup() {
    Serial.begin(115200);
    delay(500);
    printBanner();

    Wire.begin(kI2cSdaPin, kI2cSclPin, I2C_CLOCK_HZ);
    Wire.setTimeOut(I2C_TIMEOUT_MS);

    scanI2cBus();
    if (configureSensor()) {
        stats.connected = true;
        dumpRegisters();
    } else {
        stats.connected = false;
        Serial.println("[WARN] Initial AS5600 configuration failed; will retry on Wire");
    }

    lastSampleMs = millis();
    lastReportMs = lastSampleMs;
}

void loop() {
    handleSerialCommands();

    uint32_t nowMs = millis();
    if (nowMs - lastSampleMs >= kSamplePeriodMs) {
        lastSampleMs = nowMs;
        sampleSensor(micros());
    }

    if (nowMs - lastReportMs >= kReportPeriodMs) {
        lastReportMs = nowMs;
        printReport();
    }

    delay(1);
}
