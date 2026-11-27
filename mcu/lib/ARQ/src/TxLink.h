#ifndef TX_LINK_H
#define TX_LINK_H

#include <cstddef>
#include <cstdint>

#include "ARQ.h"

namespace ARQ {

enum class SendPhase : uint8_t {
    Idle,
    Transmitting,
    RetryDelay,
    Dropped,
};

struct TxLinkConfig {
    uint32_t ackTimeoutMs = 1200;
    uint8_t maxAttempts = 1;
    uint8_t maxBusyRetries = 2;
    uint32_t backoffMs = 2000;
    uint32_t hardTimeoutMs = 12000;
    uint8_t reinitThreshold = 3;
};

struct TxLinkStepResult {
    bool delivered = false;
    bool dropped = false;
    const char* dropReason = nullptr;
    bool requestReinit = false;
    bool requestStallResync = false;
};

class TxLinkController {
  public:
    static constexpr size_t kMaxPayload = 48;

    TxLinkController(ARQ& arq, address_t targetAddr, TxLinkConfig config = {});

    bool submit(const uint8_t* payload, size_t length);
    TxLinkStepResult poll(uint32_t nowMs);

    SendPhase phase() const { return phase_; }
    bool hasPending() const { return hasPending_; }
    uint32_t sentOk() const { return sentOk_; }
    uint32_t sentDropped() const { return sentDropped_; }
    uint32_t radioResyncs() const { return radioResyncs_; }
    void noteRadioResync() { radioResyncs_++; }

  private:
    void dropReading(const char* reason, TxLinkStepResult& result);
    void sendAttempt(uint32_t nowMs, TxLinkStepResult& result);
    void completeAttempt(bool successful, uint32_t nowMs, TxLinkStepResult& result);

    ARQ& arq_;
    address_t targetAddr_;
    TxLinkConfig config_;

    SendPhase phase_ = SendPhase::Idle;
    uint8_t attempt_ = 0;
    uint32_t phaseStartMs_ = 0;
    uint32_t timeoutSnapshot_ = 0;
    uint8_t consecutiveDrops_ = 0;
    uint32_t sentOk_ = 0;
    uint32_t sentDropped_ = 0;
    uint32_t radioResyncs_ = 0;

    bool hasPending_ = false;
    uint8_t pendingPayload_[kMaxPayload] = {};
    size_t pendingLength_ = 0;
};

} // namespace ARQ

#endif // TX_LINK_H
