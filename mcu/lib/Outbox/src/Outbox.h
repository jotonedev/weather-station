#ifndef OUTBOX_H
#define OUTBOX_H

#include <cstddef>
#include <cstdint>

#include "Weather.h"

// PendingReading is one buffered reading awaiting delivery to the ingest
// service.
struct PendingReading {
    WeatherData data;
    // Unix epoch seconds stamped by the base station on receipt. Zero when the
    // clock is not synced yet.
    uint32_t receivedAt;
    // Monotonic milliseconds (millis()) at receipt, used to retroactively
    // backfill receivedAt once NTP syncs.
    uint32_t receivedMs;
    // ARQ sequence number, kept for gap detection and diagnostics.
    uint16_t sequence;
};

// Outbox is a fixed-capacity ring buffer of readings. It is not thread safe,
// callers must guard access when sharing across tasks.
class Outbox {
  public:
    explicit Outbox(size_t capacity);
    Outbox(PendingReading* externalBuffer, size_t capacity);
    ~Outbox();

    Outbox(const Outbox&) = delete;
    Outbox& operator=(const Outbox&) = delete;

    // Add an entry, dropping the oldest entry when full. Returns false when
    // the entry is dropped because the buffer is full.
    bool push(const PendingReading& entry);

    // Remove and return the oldest entry. Returns false when empty.
    bool pop(PendingReading& out);

    // Read the oldest entry without removing it. Returns false when empty.
    bool peek(PendingReading& out) const;

    // Read the entry at the given offset from the oldest. Returns false when
    // out of range.
    bool peekAt(size_t index, PendingReading& out) const;

    size_t head() const { return head_; }
    size_t count() const { return count_; }
    size_t capacity() const { return capacity_; }
    bool empty() const { return count_ == 0; }
    bool full() const { return count_ == capacity_; }

    // Number of entries dropped because the buffer was full.
    uint32_t dropped() const { return dropped_; }

    // Restore ring buffer indices from persistent storage (e.g. RTC SRAM).
    // Returns false and resets the buffer if indices are out of bounds.
    bool restoreState(size_t head, size_t count, uint32_t dropped);

    // Backfill missing receivedAt timestamps using the current NTP epoch and
    // current monotonic milliseconds. Returns the number of updated entries.
    size_t backfillTimestamps(uint32_t currentEpoch, uint32_t currentMs);

    void reset();

  private:
    PendingReading* buffer_;
    size_t capacity_;
    size_t head_ = 0;
    size_t count_ = 0;
    uint32_t dropped_ = 0;
    bool ownsBuffer_ = true;
};

#endif // OUTBOX_H
