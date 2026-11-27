#ifdef ARDUINO

#include "RainMeter.h"

#include <Preferences.h>
#include <cstring>
#include <esp_attr.h>
#include <esp_rom_crc.h>

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

RainMeter::RainMeter(TwoWire& wire, float mmPerTip)
    : device_(wire, 0x000Eu, false), logic_(mmPerTip) {}

RainMeter::~RainMeter() {
    stopTask();
}

void RainMeter::initialize(uint8_t directionPin, uint8_t direction) {
    device_.initialize(directionPin, direction);

    sensors::RainCalibration cal;
    const bool hasCal = loadNvsCalibration(cal);

    const uint32_t nowMs = millis();
    CriticalSection lock(spinlock_);
    if (hasCal) {
        logic_.applyCalibration(cal);
    }
    logic_.reset(nowMs);
    loadRtcRollingBins();
}

bool RainMeter::ready() {
    if (taskHandle_ != nullptr) {
        CriticalSection lock(spinlock_);
        return logic_.ready();
    }
    return device_.probeReady();
}

bool RainMeter::isConnected() {
    return device_.isConnected();
}

bool RainMeter::magnetDetected() {
    return device_.magnetDetected();
}

bool RainMeter::magnetTooStrong() {
    return device_.magnetTooStrong();
}

bool RainMeter::magnetTooWeak() {
    return device_.magnetTooWeak();
}

int RainMeter::getLastError() {
    return device_.lastError();
}

void RainMeter::taskTrampoline(void* arg) {
    auto* self = static_cast<RainMeter*>(arg);
    self->taskLoop(self->samplePeriodMs_);
}

bool RainMeter::startTask(uint32_t samplePeriodMs, UBaseType_t priority, BaseType_t core) {
    if (taskHandle_ != nullptr) {
        return true;
    }
    samplePeriodMs_ = samplePeriodMs;
    currentPeriodMs_ = (samplePeriodMs > 0) ? samplePeriodMs : 30;
    const BaseType_t res = xTaskCreatePinnedToCore(taskTrampoline, "RainMeterTask", 4096, this,
                                                   priority, &taskHandle_, core);
    return res == pdPASS;
}

void RainMeter::stopTask() {
    if (taskHandle_ != nullptr) {
        vTaskDelete(taskHandle_);
        taskHandle_ = nullptr;
    }
}

void RainMeter::taskLoop(uint32_t samplePeriodMs) {
    constexpr uint32_t kRestPeriodMs = 100;
    const uint32_t activePeriodMs = (samplePeriodMs > 0) ? samplePeriodMs : 30;
    currentPeriodMs_ = activePeriodMs;
    while (true) {
        update();
        bool resting = false;
        {
            CriticalSection lock(spinlock_);
            resting = logic_.isResting();
        }
        const uint32_t delayMs = resting ? kRestPeriodMs : activePeriodMs;
        vTaskDelay(pdMS_TO_TICKS(delayMs));
        currentPeriodMs_ = delayMs;
    }
}

void RainMeter::update() {
    const As5600Reading reading = device_.poll();
    sensors::RainMeterSample sample;
    sample.nowMs = reading.nowMs;
    sample.rawAngle = reading.rawAngle;
    sample.i2cOk = reading.i2cOk;
    sample.hasDiag = reading.hasDiag;
    sample.diagStatus = reading.diagStatus;
    sample.diagAgc = reading.diagAgc;
    sample.diagMagnitude = reading.diagMagnitude;
    sample.samplePeriodMs = currentPeriodMs_;

    sensors::RainMeterStepResult stepResult;
    {
        CriticalSection lock(spinlock_);
        stepResult = logic_.step(sample);
        if (stepResult.saveRtcBins) {
            saveRtcRollingBins();
        }
    }

    if (stepResult.saveCalibration) {
        saveNvsCalibration(stepResult.calibration);
    }
}

float RainMeter::getRainfall1h() {
    CriticalSection lock(spinlock_);
    return logic_.rainfall1h();
}

float RainMeter::getRainfall24h() {
    CriticalSection lock(spinlock_);
    return logic_.rainfall24h();
}

float RainMeter::getRainRate() {
    CriticalSection lock(spinlock_);
    return logic_.rainRate();
}

RainData RainMeter::getRainData() {
    CriticalSection lock(spinlock_);
    return logic_.rainData();
}

RainMeterState RainMeter::getState() {
    CriticalSection lock(spinlock_);
    return logic_.state();
}

bool RainMeter::isCalibrated() {
    CriticalSection lock(spinlock_);
    return logic_.isCalibrated();
}

bool RainMeter::isJammed() {
    CriticalSection lock(spinlock_);
    return logic_.isJammed();
}

float RainMeter::getCurrentAngleDegrees() {
    CriticalSection lock(spinlock_);
    return logic_.currentAngleDegrees();
}

float RainMeter::getTipTravelDegrees() {
    CriticalSection lock(spinlock_);
    return logic_.tipTravelDegrees();
}

uint8_t RainMeter::getLastStatus() {
    CriticalSection lock(spinlock_);
    return logic_.lastStatus();
}

uint8_t RainMeter::getLastAGC() {
    CriticalSection lock(spinlock_);
    return logic_.lastAgc();
}

uint16_t RainMeter::getLastMagnitude() {
    CriticalSection lock(spinlock_);
    return logic_.lastMagnitude();
}

uint16_t RainMeter::getAGCVariance() {
    CriticalSection lock(spinlock_);
    return logic_.agcVariance();
}

uint32_t RainMeter::getGlitchCount() {
    CriticalSection lock(spinlock_);
    return logic_.glitchCount();
}

uint32_t RainMeter::getLastHeartbeatMs() {
    CriticalSection lock(spinlock_);
    return logic_.lastHeartbeatMs();
}

void RainMeter::resetStats() {
    CriticalSection lock(spinlock_);
    logic_.resetStats();
}

void RainMeter::clearCalibration() {
    {
        CriticalSection lock(spinlock_);
        logic_.clearCalibration();
    }

    Preferences prefs;
    prefs.begin("rainmeter", false);
    prefs.putBool("cal", false);
    prefs.end();
}

void RainMeter::setMmPerTip(float mm) {
    CriticalSection lock(spinlock_);
    logic_.setMmPerTip(mm);
}

float RainMeter::getMmPerTip() const {
    CriticalSection lock(spinlock_);
    return logic_.mmPerTip();
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
        logic_.loadRollingBins(bins);
        return;
    }

    logic_.clearRollingBins();
    saveRtcRollingBins();
}

void RainMeter::saveRtcRollingBins() {
    const sensors::RainRollingBins bins = logic_.exportRollingBins();
    s_rtcRainState.magic = kRainRtcMagic;
    std::memcpy(s_rtcRainState.minuteBins, bins.minuteBins, sizeof(s_rtcRainState.minuteBins));
    std::memcpy(s_rtcRainState.hourBins, bins.hourBins, sizeof(s_rtcRainState.hourBins));
    s_rtcRainState.currentMinuteIdx = bins.currentMinuteIdx;
    s_rtcRainState.currentHourIdx = bins.currentHourIdx;
    s_rtcRainState.currentRainRate = bins.currentRainRate;
    s_rtcRainState.crc = computeRainRtcCrc(s_rtcRainState);
}

#endif // ARDUINO
