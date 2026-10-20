#include "Outbox.h"

Outbox::Outbox(size_t capacity) : capacity_(capacity), ownsBuffer_(true) {
    buffer_ = (capacity_ > 0) ? new PendingReading[capacity_] : nullptr;
}

Outbox::Outbox(PendingReading* externalBuffer, size_t capacity)
    : buffer_(externalBuffer), capacity_((externalBuffer != nullptr) ? capacity : 0),
      ownsBuffer_(false) {}

Outbox::~Outbox() {
    if (ownsBuffer_) {
        delete[] buffer_;
    }
}

bool Outbox::push(const PendingReading& entry) {
    if (capacity_ == 0) {
        dropped_++;
        return false;
    }
    if (full()) {
        // Drop the oldest entry to make room for the newest.
        head_ = (head_ + 1) % capacity_;
        count_--;
        dropped_++;
    }
    size_t const tail = (head_ + count_) % capacity_;
    buffer_[tail] = entry;
    count_++;
    return true;
}

bool Outbox::pop(PendingReading& out) {
    if (!peek(out)) {
        return false;
    }
    head_ = (head_ + 1) % capacity_;
    count_--;
    return true;
}

bool Outbox::peek(PendingReading& out) const {
    return peekAt(0, out);
}

bool Outbox::peekAt(size_t index, PendingReading& out) const {
    if (index >= count_) {
        return false;
    }
    size_t const slot = (head_ + index) % capacity_;
    out = buffer_[slot];
    return true;
}

void Outbox::reset() {
    head_ = 0;
    count_ = 0;
}

bool Outbox::restoreState(size_t head, size_t count, uint32_t dropped) {
    if (capacity_ == 0 || head >= capacity_ || count > capacity_) {
        reset();
        dropped_ = 0;
        return false;
    }
    head_ = head;
    count_ = count;
    dropped_ = dropped;
    return true;
}

size_t Outbox::backfillTimestamps(uint32_t currentEpoch, uint32_t currentMs) {
    if (currentEpoch == 0 || count_ == 0) {
        return 0;
    }
    size_t updated = 0;
    for (size_t i = 0; i < count_; ++i) {
        size_t const slot = (head_ + i) % capacity_;
        if (buffer_[slot].receivedAt == 0) {
            uint32_t const elapsedSec = (currentMs - buffer_[slot].receivedMs) / 1000u;
            if (currentEpoch > elapsedSec) {
                buffer_[slot].receivedAt = currentEpoch - elapsedSec;
                updated++;
            }
        }
    }
    return updated;
}
