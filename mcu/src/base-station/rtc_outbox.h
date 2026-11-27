#ifndef BASE_RTC_OUTBOX_H
#define BASE_RTC_OUTBOX_H

#include <cstddef>
#include <cstdint>
#include <freertos/FreeRTOS.h>
#include <freertos/semphr.h>

#include <Outbox.h>

class RtcOutbox {
  public:
    RtcOutbox();

    void init();
    bool pushReading(const PendingReading& reading, bool clockSynced, uint32_t nowEpoch,
                     bool& evictedOut);
    size_t count();
    size_t peekBatch(PendingReading* out, size_t maxCount, bool clockSynced, uint32_t nowEpoch,
                     uint32_t nowMs);
    void popCount(size_t n);
    size_t backfillTimestamps(uint32_t nowEpoch, uint32_t nowMs);

  private:
    void syncLocked();

    Outbox outbox_;
    SemaphoreHandle_t mutex_;
};

#endif // BASE_RTC_OUTBOX_H
