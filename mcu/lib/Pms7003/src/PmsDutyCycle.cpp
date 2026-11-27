#include "PmsDutyCycle.h"

namespace pms7003 {

void PmsDutyCycle::begin(uint32_t nowMs) {
    parser_.reset();
    state_ = PmsState::WarmingUp;
    cycleStartMs_ = nowMs;
    stateStartMs_ = nowMs;
    reportReadyMs_ = 0;
    lastSleepRetryMs_ = 0;
    readStartErrorCount_ = 0;
    latestFrame_ = PmsFrame{};
    pendingReport_ = false;
    timedOut_ = false;
    hadError_ = false;
}

PmsActions PmsDutyCycle::poll(uint32_t nowMs, const uint8_t* rxBytes, size_t rxLen) {
    PmsActions actions{};

    switch (state_) {
    case PmsState::Sleeping:
        if (rxLen > 0u) {
            if (lastSleepRetryMs_ == 0u || (nowMs - lastSleepRetryMs_ >= config_.sleepRetryMs)) {
                actions.warnActiveInSleep = true;
                actions.sendSleep = true;
                lastSleepRetryMs_ = nowMs;
            }
        }
        if (nowMs - cycleStartMs_ >= config_.sampleIntervalMs) {
            cycleStartMs_ = nowMs;
            stateStartMs_ = nowMs;
            actions.flushRx = true;
            actions.sendWakeUp = true;
            parser_.reset();
            state_ = PmsState::WarmingUp;
        }
        break;

    case PmsState::WarmingUp:
        if (nowMs - stateStartMs_ >= config_.warmupMs) {
            actions.flushRx = true;
            parser_.reset();
            readStartErrorCount_ = parser_.sensorErrorCount();
            stateStartMs_ = nowMs;
            state_ = PmsState::Reading;
        }
        break;

    case PmsState::Reading: {
        PmsFrame frame{};
        if (rxBytes != nullptr) {
            for (size_t i = 0; i < rxLen; ++i) {
                if (parser_.feed(rxBytes[i], frame)) {
                    latestFrame_ = frame;
                    pendingReport_ = true;
                    reportReadyMs_ = nowMs;
                    timedOut_ = false;
                    hadError_ = (parser_.sensorErrorCount() != readStartErrorCount_);
                    actions.sendSleep = true;
                    actions.flushRx = true;
                    parser_.reset();
                    lastSleepRetryMs_ = nowMs;
                    state_ = PmsState::Sleeping;
                    return actions;
                }
            }
        }
        if (nowMs - stateStartMs_ >= config_.readTimeoutMs) {
            actions.warnReadTimeout = true;
            timedOut_ = true;
            hadError_ = (parser_.sensorErrorCount() != readStartErrorCount_);
            actions.sendSleep = true;
            actions.flushRx = true;
            parser_.reset();
            lastSleepRetryMs_ = nowMs;
            state_ = PmsState::Sleeping;
        }
        break;
    }
    }

    return actions;
}

bool PmsDutyCycle::hasPendingReport(uint32_t nowMs) {
    if (pendingReport_ && (nowMs - reportReadyMs_ >= config_.reportExpiryMs)) {
        pendingReport_ = false;
    }
    return pendingReport_;
}

const PmsFrame& PmsDutyCycle::latestFrame() const {
    return latestFrame_;
}

void PmsDutyCycle::markReportDelivered() {
    pendingReport_ = false;
}

PmsState PmsDutyCycle::state() const {
    return state_;
}

bool PmsDutyCycle::timedOut() const {
    return timedOut_;
}

bool PmsDutyCycle::hadError() const {
    return hadError_;
}

const PmsStreamParser& PmsDutyCycle::parser() const {
    return parser_;
}

} // namespace pms7003
