#ifndef PMS_DUTY_CYCLE_H
#define PMS_DUTY_CYCLE_H

#include <cstddef>
#include <cstdint>

namespace pms7003 {

inline constexpr uint8_t kCmdSleep[7] = {0x42, 0x4D, 0xE4, 0x00, 0x00, 0x01, 0x73};
inline constexpr uint8_t kCmdWakeUp[7] = {0x42, 0x4D, 0xE4, 0x00, 0x01, 0x01, 0x74};
inline constexpr uint8_t kCmdActive[7] = {0x42, 0x4D, 0xE1, 0x00, 0x01, 0x01, 0x71};

struct PmsFrame {
    uint16_t pm1_0_sp = 0;
    uint16_t pm2_5_sp = 0;
    uint16_t pm10_0_sp = 0;
    uint16_t pm1_0_ae = 0;
    uint16_t pm2_5_ae = 0;
    uint16_t pm10_0_ae = 0;
    uint8_t version = 0;
    uint8_t errorCode = 0;
};

class PmsStreamParser {
  public:
    void reset() {
        index_ = 0;
        expectedTotal_ = 0;
    }

    bool feed(uint8_t byte, PmsFrame& out) {
        if (index_ == 0u) {
            if (byte == 0x42u) {
                buf_[0] = byte;
                index_ = 1u;
            }
            return false;
        }

        if (index_ == 1u) {
            if (byte == 0x4Du) {
                buf_[1] = byte;
                index_ = 2u;
            } else if (byte == 0x42u) {
                buf_[0] = byte;
                index_ = 1u;
            } else {
                index_ = 0u;
            }
            return false;
        }

        buf_[index_++] = byte;

        if (index_ == 4u) {
            const uint16_t frameLen = readBeU16(&buf_[2]);
            if (frameLen == 28u) {
                expectedTotal_ = 32u;
            } else if (frameLen == 4u) {
                expectedTotal_ = 8u;
            } else {
                reset();
            }
            return false;
        }

        if (index_ < 4u || index_ < expectedTotal_) {
            return false;
        }

        const size_t total = expectedTotal_;
        reset();

        uint16_t sum = 0u;
        for (size_t i = 0; i < total - 2u; ++i) {
            sum = static_cast<uint16_t>(sum + buf_[i]);
        }
        const uint16_t expectedChecksum = readBeU16(&buf_[total - 2u]);
        if (sum != expectedChecksum) {
            ++checksumErrorCount_;
            return false;
        }

        if (total == 8u) {
            ++ackCount_;
            return false;
        }

        if (buf_[29] != 0u) {
            lastErrorCode_ = buf_[29];
            ++sensorErrorCount_;
            return false;
        }

        out.pm1_0_sp = readBeU16(&buf_[4]);
        out.pm2_5_sp = readBeU16(&buf_[6]);
        out.pm10_0_sp = readBeU16(&buf_[8]);
        out.pm1_0_ae = readBeU16(&buf_[10]);
        out.pm2_5_ae = readBeU16(&buf_[12]);
        out.pm10_0_ae = readBeU16(&buf_[14]);
        out.version = buf_[28];
        out.errorCode = buf_[29];
        return true;
    }

    uint32_t checksumErrorCount() const { return checksumErrorCount_; }
    uint32_t sensorErrorCount() const { return sensorErrorCount_; }
    uint32_t ackCount() const { return ackCount_; }
    uint8_t lastErrorCode() const { return lastErrorCode_; }

  private:
    static uint16_t readBeU16(const uint8_t* p) {
        return static_cast<uint16_t>((static_cast<uint16_t>(p[0]) << 8) | p[1]);
    }

    uint8_t buf_[32] = {};
    size_t index_ = 0;
    size_t expectedTotal_ = 0;
    uint32_t checksumErrorCount_ = 0;
    uint32_t sensorErrorCount_ = 0;
    uint32_t ackCount_ = 0;
    uint8_t lastErrorCode_ = 0;
};

enum class PmsState : uint8_t {
    Sleeping,
    WarmingUp,
    Reading,
};

struct PmsConfig {
    uint32_t sampleIntervalMs = 900000u;
    uint32_t warmupMs = 30000u;
    uint32_t readTimeoutMs = 10000u;
    uint32_t reportExpiryMs = 300000u;
    uint32_t sleepRetryMs = 5000u;
};

struct PmsActions {
    bool sendWakeUp = false;
    bool sendSleep = false;
    bool flushRx = false;
    bool warnActiveInSleep = false;
    bool warnReadTimeout = false;
};

class PmsDutyCycle {
  public:
    explicit PmsDutyCycle(const PmsConfig& config = PmsConfig{}) : config_(config) {}

    void begin(uint32_t nowMs);
    PmsActions poll(uint32_t nowMs, const uint8_t* rxBytes, size_t rxLen);
    bool hasPendingReport(uint32_t nowMs);
    const PmsFrame& latestFrame() const;
    void markReportDelivered();
    PmsState state() const;
    bool timedOut() const;
    bool hadError() const;
    const PmsStreamParser& parser() const;

  private:
    PmsConfig config_{};
    PmsStreamParser parser_{};
    PmsState state_ = PmsState::WarmingUp;
    uint32_t cycleStartMs_ = 0;
    uint32_t stateStartMs_ = 0;
    uint32_t reportReadyMs_ = 0;
    uint32_t lastSleepRetryMs_ = 0;
    uint32_t readStartErrorCount_ = 0;
    PmsFrame latestFrame_{};
    bool pendingReport_ = false;
    bool timedOut_ = false;
    bool hadError_ = false;
};

} // namespace pms7003

#endif // PMS_DUTY_CYCLE_H
