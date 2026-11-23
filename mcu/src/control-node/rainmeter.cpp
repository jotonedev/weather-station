#include <Arduino.h>
#include <Preferences.h>
#include <cstring>
#include <esp_attr.h>
#include <esp_rom_crc.h>

#include "rainmeter.h"

namespace {

constexpr uint32_t kRainRtcMagic = 0x5241494Eu; // "RAIN"

// Retained in RTC Slow SRAM across watchdog resets and software reboots without wearing flash.
struct RainRtcState {
    uint32_t magic;
    float minuteBins[60];
    float hourBins[24];
    uint16_t currentMinuteIdx;
    uint16_t currentHourIdx;
    float currentRainRate;
    uint32_t crc;
};

RTC_NOINIT_ATTR RainRtcState s_rtcRainState;

uint32_t computeRainRtcCrc(const RainRtcState& state) {
    return esp_rom_crc32_le(0, reinterpret_cast<const uint8_t*>(&state),
                            offsetof(RainRtcState, crc));
}

} // namespace

RainMeter::RainMeter(TwoWire& wire, float mmPerTip) : wire(wire), as5600(&wire), logic(mmPerTip) {}

RainMeter::~RainMeter() {
    stopTask();
}

void RainMeter::initialize(uint8_t directionPin, uint8_t direction) {
    as5600.begin(directionPin);
    as5600.setDirection(direction);
    // 3 LSB hysteresis to suppress angle jitter at rest
    as5600.setHysteresis(AS5600_HYST_LSB3);
    // Slow filter 16x for maximum noise rejection
    as5600.setSlowFilter(AS5600_SLOW_FILT_16X);
    // Disable fast filter so noise spikes do not bypass slow filter
    as5600.setFastFilter(AS5600_FAST_FILT_NONE);
    // 20 ms internal polling (LPM2) to provide fresh samples for the 30 ms task
    as5600.setPowerMode(AS5600_POWERMODE_LOW2);
    as5600.setWatchDog(AS5600_WATCHDOG_ON);

    sensors::RainCalibration cal;
    const bool hasCal = loadNvsCalibration(cal);

    const uint32_t nowMs = millis();
    CriticalSection lock(spinlock);
    lastDiagMs = 0;
    if (hasCal) {
        logic.applyCalibration(cal);
    }
    logic.reset(nowMs);
    loadRtcRollingBins();
}

bool RainMeter::ready() {
    if (taskHandle != nullptr) {
        CriticalSection lock(spinlock);
        return logic.ready();
    }
    return as5600.isConnected() && as5600.magnetDetected() && !as5600.magnetTooStrong() &&
           !as5600.magnetTooWeak();
}

bool RainMeter::isConnected() {
    return as5600.isConnected();
}

bool RainMeter::magnetDetected() {
    return as5600.magnetDetected();
}

bool RainMeter::magnetTooStrong() {
    return as5600.magnetTooStrong();
}

bool RainMeter::magnetTooWeak() {
    return as5600.magnetTooWeak();
}

int RainMeter::getLastError() {
    return as5600.lastError();
}

void RainMeter::taskTrampoline(void* arg) {
    auto* self = static_cast<RainMeter*>(arg);
    self->taskLoop(self->samplePeriodMs);
}

bool RainMeter::startTask(uint32_t samplePeriodMs, UBaseType_t priority, BaseType_t core) {
    if (taskHandle != nullptr) {
        return true;
    }
    this->samplePeriodMs = samplePeriodMs;
    BaseType_t res = xTaskCreatePinnedToCore(taskTrampoline, "RainMeterTask", 4096, this, priority,
                                             &taskHandle, core);
    return res == pdPASS;
}

void RainMeter::stopTask() {
    if (taskHandle != nullptr) {
        vTaskDelete(taskHandle);
        taskHandle = nullptr;
    }
}

void RainMeter::taskLoop(uint32_t samplePeriodMs) {
    constexpr uint32_t kRestPeriodMs = 100;
    const uint32_t activePeriodMs = (samplePeriodMs > 0) ? samplePeriodMs : 30;
    while (true) {
        update();
        bool resting = false;
        {
            CriticalSection lock(spinlock);
            resting = logic.isResting();
        }
        const uint32_t delayMs = resting ? kRestPeriodMs : activePeriodMs;
        vTaskDelay(pdMS_TO_TICKS(delayMs));
    }
}

void RainMeter::update() {
    const uint32_t nowMs = millis();
    sensors::RainMeterSample sample;
    sample.nowMs = nowMs;
    sample.samplePeriodMs = samplePeriodMs;
    sample.rawAngle = as5600.readAngle();
    sample.i2cOk = (as5600.lastError() == AS5600_OK);

    if (sample.i2cOk) {
        sample.hasDiag = (lastDiagMs == 0) || (nowMs - lastDiagMs >= 5000uL);
        if (sample.hasDiag) {
            lastDiagMs = nowMs;
            sample.diagStatus = as5600.readStatus();
            sample.diagAgc = as5600.readAGC();
        }
    }

    sensors::RainMeterStepResult result;
    {
        CriticalSection lock(spinlock);
        result = logic.step(sample);
        if (result.saveRtcBins) {
            saveRtcRollingBins();
        }
    }

    if (result.saveCalibration) {
        saveNvsCalibration(result.calibration);
    }
}

float RainMeter::getRainfall1h() {
    CriticalSection lock(spinlock);
    return logic.rainfall1h();
}

float RainMeter::getRainfall24h() {
    CriticalSection lock(spinlock);
    return logic.rainfall24h();
}

float RainMeter::getRainRate() {
    CriticalSection lock(spinlock);
    return logic.rainRate();
}

RainData RainMeter::getRainData() {
    CriticalSection lock(spinlock);
    return logic.rainData();
}

RainMeterState RainMeter::getState() {
    CriticalSection lock(spinlock);
    return logic.state();
}

bool RainMeter::isCalibrated() {
    CriticalSection lock(spinlock);
    return logic.isCalibrated();
}

bool RainMeter::isJammed() {
    CriticalSection lock(spinlock);
    return logic.isJammed();
}

bool RainMeter::isTiltWarning() {
    CriticalSection lock(spinlock);
    return logic.isTiltWarning();
}

float RainMeter::getTiltDegrees() {
    CriticalSection lock(spinlock);
    return logic.tiltDegrees();
}

float RainMeter::getCurrentAngleDegrees() {
    CriticalSection lock(spinlock);
    return logic.currentAngleDegrees();
}

float RainMeter::getTipTravelDegrees() {
    CriticalSection lock(spinlock);
    return logic.tipTravelDegrees();
}

uint8_t RainMeter::getLastStatus() {
    CriticalSection lock(spinlock);
    return logic.lastStatus();
}

uint8_t RainMeter::getLastAGC() {
    CriticalSection lock(spinlock);
    return logic.lastAgc();
}

uint16_t RainMeter::getAGCVariance() {
    CriticalSection lock(spinlock);
    return logic.agcVariance();
}

uint32_t RainMeter::getGlitchCount() {
    CriticalSection lock(spinlock);
    return logic.glitchCount();
}

uint32_t RainMeter::getLastHeartbeatMs() {
    CriticalSection lock(spinlock);
    return logic.lastHeartbeatMs();
}

void RainMeter::resetStats() {
    CriticalSection lock(spinlock);
    logic.resetStats();
}

void RainMeter::clearCalibration() {
    {
        CriticalSection lock(spinlock);
        logic.clearCalibration();
    }

    Preferences prefs;
    prefs.begin("rainmeter", false);
    prefs.putBool("cal", false);
    prefs.end();
}

void RainMeter::setMmPerTip(float mm) {
    CriticalSection lock(spinlock);
    logic.setMmPerTip(mm);
}

float RainMeter::getMmPerTip() const {
    CriticalSection lock(spinlock);
    return logic.mmPerTip();
}

bool RainMeter::loadNvsCalibration(sensors::RainCalibration& out) {
    Preferences prefs;
    prefs.begin("rainmeter", false);
    const bool isCal = prefs.getBool("cal", false);
    if (isCal) {
        out.rawReference = prefs.getUShort("rawRef", 0);
        out.restLeft = prefs.getInt("restL", 0);
        out.restRight = prefs.getInt("restR", 0);
    }
    prefs.end();
    return isCal;
}

void RainMeter::saveNvsCalibration(const sensors::RainCalibration& cal) {
    Preferences prefs;
    prefs.begin("rainmeter", false);
    prefs.putBool("cal", true);
    prefs.putUShort("rawRef", cal.rawReference);
    prefs.putInt("restL", cal.restLeft);
    prefs.putInt("restR", cal.restRight);
    prefs.end();
}

void RainMeter::loadRtcRollingBins() {
    if (s_rtcRainState.magic == kRainRtcMagic &&
        s_rtcRainState.crc == computeRainRtcCrc(s_rtcRainState) &&
        s_rtcRainState.currentMinuteIdx < 60 && s_rtcRainState.currentHourIdx < 24) {
        sensors::RainRollingBins bins;
        std::memcpy(bins.minuteBins, s_rtcRainState.minuteBins, sizeof(bins.minuteBins));
        std::memcpy(bins.hourBins, s_rtcRainState.hourBins, sizeof(bins.hourBins));
        bins.currentMinuteIdx = s_rtcRainState.currentMinuteIdx;
        bins.currentHourIdx = s_rtcRainState.currentHourIdx;
        bins.currentRainRate = s_rtcRainState.currentRainRate;
        logic.loadRollingBins(bins);
        return;
    }

    logic.clearRollingBins();
    saveRtcRollingBins();
}

void RainMeter::saveRtcRollingBins() {
    const sensors::RainRollingBins bins = logic.exportRollingBins();
    s_rtcRainState.magic = kRainRtcMagic;
    std::memcpy(s_rtcRainState.minuteBins, bins.minuteBins, sizeof(s_rtcRainState.minuteBins));
    std::memcpy(s_rtcRainState.hourBins, bins.hourBins, sizeof(s_rtcRainState.hourBins));
    s_rtcRainState.currentMinuteIdx = bins.currentMinuteIdx;
    s_rtcRainState.currentHourIdx = bins.currentHourIdx;
    s_rtcRainState.currentRainRate = bins.currentRainRate;
    s_rtcRainState.crc = computeRainRtcCrc(s_rtcRainState);
}
