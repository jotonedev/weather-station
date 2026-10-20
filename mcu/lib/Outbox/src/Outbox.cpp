#include "Outbox.h"

Outbox::Outbox(size_t capacity) : capacity_(capacity) {
    buffer_ = (capacity_ > 0) ? new PendingReading[capacity_] : nullptr;
}

Outbox::~Outbox() {
    delete[] buffer_;
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
