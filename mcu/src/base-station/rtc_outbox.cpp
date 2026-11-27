#include "rtc_outbox.h"

#include <Arduino.h>
#include <cstddef>
#include <cstring>
#include <esp_attr.h>
#include <esp_rom_crc.h>

#include <Sync.h>

#include "config.h"

namespace {

constexpr uint32_t kOutboxRtcMagic = 0x4F424F58u; // "OBOX"

struct RtcOutboxState {
    uint32_t magic;
    uint16_t capacity;
    uint16_t head;
    uint16_t count;
    uint32_t dropped;
    PendingReading entries[OUTBOX_CAPACITY];
    uint32_t crc;
};

RTC_NOINIT_ATTR RtcOutboxState s_rtcOutbox;

uint32_t computeOutboxRtcCrc(const RtcOutboxState& state) {
    return esp_rom_crc32_le(0, reinterpret_cast<const uint8_t*>(&state),
                            offsetof(RtcOutboxState, crc));
}

} // namespace

RtcOutbox::RtcOutbox() : outbox_(s_rtcOutbox.entries, OUTBOX_CAPACITY), mutex_(nullptr) {}

void RtcOutbox::syncLocked() {
    s_rtcOutbox.magic = kOutboxRtcMagic;
    s_rtcOutbox.capacity = static_cast<uint16_t>(OUTBOX_CAPACITY);
    s_rtcOutbox.head = static_cast<uint16_t>(outbox_.head());
    s_rtcOutbox.count = static_cast<uint16_t>(outbox_.count());
    s_rtcOutbox.dropped = outbox_.dropped();
    s_rtcOutbox.crc = computeOutboxRtcCrc(s_rtcOutbox);
}

void RtcOutbox::init() {
    mutex_ = xSemaphoreCreateMutex();
    if (s_rtcOutbox.magic == kOutboxRtcMagic && s_rtcOutbox.capacity == OUTBOX_CAPACITY &&
        s_rtcOutbox.crc == computeOutboxRtcCrc(s_rtcOutbox) &&
        outbox_.restoreState(s_rtcOutbox.head, s_rtcOutbox.count, s_rtcOutbox.dropped)) {
        size_t discarded = outbox_.dropUntimestamped();
        if (discarded > 0) {
            syncLocked();
            Serial.printf("[WARN] Dropped %u untimestamped readings from RTC SRAM after reboot\n",
                          static_cast<unsigned>(discarded));
        }
        Serial.printf("[OK] Restored %u buffered readings from RTC SRAM\n",
                      static_cast<unsigned>(outbox_.count()));
        return;
    }
    std::memset(&s_rtcOutbox, 0, sizeof(s_rtcOutbox));
    outbox_.reset();
    syncLocked();
}

bool RtcOutbox::pushReading(const PendingReading& reading, bool clockSynced, uint32_t nowEpoch,
                            bool& evictedOut) {
    evictedOut = false;
    MutexGuard lock(mutex_);
    if (!lock.locked()) {
        return false;
    }
    PendingReading entry = reading;
    entry.receivedAt = clockSynced ? nowEpoch : 0;
    bool pushed = outbox_.push(entry, &evictedOut);
    syncLocked();
    return pushed;
}

size_t RtcOutbox::count() {
    MutexGuard lock(mutex_);
    if (!lock.locked()) {
        return 0;
    }
    return outbox_.count();
}

size_t RtcOutbox::peekBatch(PendingReading* out, size_t maxCount, bool clockSynced,
                            uint32_t nowEpoch, uint32_t nowMs) {
    MutexGuard lock(mutex_);
    if (!lock.locked()) {
        return 0;
    }
    if (clockSynced && outbox_.backfillTimestamps(nowEpoch, nowMs) > 0) {
        syncLocked();
    }
    size_t available = outbox_.count();
    size_t limit = available < maxCount ? available : maxCount;
    for (size_t i = 0; i < limit; i++) {
        outbox_.peekAt(i, out[i]);
    }
    return limit;
}

void RtcOutbox::popCount(size_t n) {
    MutexGuard lock(mutex_);
    if (!lock.locked()) {
        return;
    }
    PendingReading discard;
    for (size_t i = 0; i < n; i++) {
        if (!outbox_.pop(discard)) {
            break;
        }
    }
    syncLocked();
}

size_t RtcOutbox::backfillTimestamps(uint32_t nowEpoch, uint32_t nowMs) {
    MutexGuard lock(mutex_);
    if (!lock.locked()) {
        return 0;
    }
    size_t updated = outbox_.backfillTimestamps(nowEpoch, nowMs);
    if (updated > 0) {
        syncLocked();
    }
    return updated;
}
