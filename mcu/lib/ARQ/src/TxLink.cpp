#include "TxLink.h"

#include <cstring>

namespace ARQ {

TxLinkController::TxLinkController(ARQ& arq, address_t targetAddr, TxLinkConfig config)
    : arq_(arq), targetAddr_(targetAddr), config_(config) {}

bool TxLinkController::submit(const uint8_t* payload, size_t length) {
    if (payload == nullptr || length == 0 || length > kMaxPayload) {
        return false;
    }
    std::memcpy(pendingPayload_, payload, length);
    pendingLength_ = length;
    hasPending_ = true;
    return true;
}

void TxLinkController::dropReading(const char* reason, TxLinkStepResult& result) {
    sentDropped_++;
    consecutiveDrops_++;
    phase_ = SendPhase::Dropped;
    attempt_ = 0;
    result.dropped = true;
    result.dropReason = reason;
    if (consecutiveDrops_ >= config_.reinitThreshold) {
        result.requestReinit = true;
        consecutiveDrops_ = 0;
    }
    arq_.sleep();
}

void TxLinkController::sendAttempt(uint32_t nowMs, TxLinkStepResult& result) {
    timeoutSnapshot_ = arq_.getTxTimeoutCount();
    const Status status =
        arq_.startSendData(pendingPayload_, static_cast<length_t>(pendingLength_), targetAddr_);
    switch (status) {
    case STATUS_OK:
        phase_ = SendPhase::Transmitting;
        phaseStartMs_ = nowMs;
        break;
    case STATUS_CHANNEL_BUSY:
    case STATUS_BUSY:
        attempt_++;
        if (attempt_ <= config_.maxBusyRetries) {
            arq_.sleep();
            phase_ = SendPhase::RetryDelay;
            phaseStartMs_ = nowMs;
        } else {
            dropReading("channel_busy", result);
        }
        break;
    case STATUS_INVALID_SIZE:
        dropReading("oversize_frame", result);
        break;
    default:
        dropReading("radio_error", result);
        break;
    }
}

void TxLinkController::completeAttempt(bool successful, uint32_t nowMs, TxLinkStepResult& result) {
    if (successful) {
        sentOk_++;
        consecutiveDrops_ = 0;
        attempt_ = 0;
        phase_ = SendPhase::Idle;
        arq_.sleep();
        result.delivered = true;
        return;
    }

    attempt_++;
    if (attempt_ < config_.maxAttempts) {
        arq_.sleep();
        phase_ = SendPhase::RetryDelay;
        phaseStartMs_ = nowMs;
        return;
    }
    dropReading("no_ack", result);
}

TxLinkStepResult TxLinkController::poll(uint32_t nowMs) {
    TxLinkStepResult result;
    switch (phase_) {
    case SendPhase::Idle:
        if (hasPending_) {
            hasPending_ = false;
            attempt_ = 0;
            sendAttempt(nowMs, result);
        } else if (arq_.getState() == RadioState::IDLE) {
            arq_.sleep();
        }
        break;

    case SendPhase::Transmitting:
        arq_.poll(config_.ackTimeoutMs);
        if (arq_.getState() == RadioState::IDLE) {
            const bool timedOut = arq_.getTxTimeoutCount() != timeoutSnapshot_;
            completeAttempt(!timedOut, nowMs, result);
        } else if (nowMs - phaseStartMs_ > config_.hardTimeoutMs) {
            dropReading("stalled", result);
            if (!result.requestReinit) {
                result.requestStallResync = true;
            }
        }
        break;

    case SendPhase::RetryDelay:
        if (nowMs - phaseStartMs_ >= config_.backoffMs) {
            sendAttempt(nowMs, result);
        }
        break;

    case SendPhase::Dropped:
        phase_ = SendPhase::Idle;
        break;
    }
    return result;
}

} // namespace ARQ
