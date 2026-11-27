#include "counters_nvs.h"

#include <Preferences.h>

#include <Sync.h>

#include "config.h"

namespace {

constexpr uint32_t kCounterSaveIntervalMs = 300000uL;

} // namespace

CountersNvs::CountersNvs()
    : mutex_(nullptr), lastSaveMs_(0), hasNodeHealth_(false), lastNodeHealth_{} {}

void CountersNvs::init() {
    mutex_ = xSemaphoreCreateMutex();
}

void CountersNvs::load() {
    Preferences prefs;
    if (!prefs.begin(NVS_NAMESPACE, true)) {
        return;
    }
    MutexGuard lock(mutex_);
    if (lock.locked()) {
        counters_.received = prefs.getUInt(NVS_KEY_RECEIVED, 0);
        counters_.sent = prefs.getUInt(NVS_KEY_SENT, 0);
        counters_.dropped = prefs.getUInt(NVS_KEY_DROPPED, 0);
        counters_.framesInvalid = prefs.getUInt(NVS_KEY_FRAMES_INVALID, 0);
        counters_.bufferDropped = prefs.getUInt(NVS_KEY_BUFFER_DROPPED, 0);
        counters_.lastHttpCode = prefs.getInt(NVS_KEY_LAST_HTTP, 0);
        lastSaved_ = counters_;
    }
    prefs.end();
}

void CountersNvs::saveIfChanged() {
    BaseCounters snap = snapshot();

    if (snap.received == lastSaved_.received && snap.sent == lastSaved_.sent &&
        snap.dropped == lastSaved_.dropped && snap.framesInvalid == lastSaved_.framesInvalid &&
        snap.bufferDropped == lastSaved_.bufferDropped &&
        snap.lastHttpCode == lastSaved_.lastHttpCode) {
        return;
    }

    Preferences prefs;
    if (!prefs.begin(NVS_NAMESPACE, false)) {
        return;
    }
    prefs.putUInt(NVS_KEY_RECEIVED, snap.received);
    prefs.putUInt(NVS_KEY_SENT, snap.sent);
    prefs.putUInt(NVS_KEY_DROPPED, snap.dropped);
    prefs.putUInt(NVS_KEY_FRAMES_INVALID, snap.framesInvalid);
    prefs.putUInt(NVS_KEY_BUFFER_DROPPED, snap.bufferDropped);
    prefs.putInt(NVS_KEY_LAST_HTTP, snap.lastHttpCode);
    prefs.end();
    lastSaved_ = snap;
}

void CountersNvs::snapshotIfDue(uint32_t nowMs) {
    if (nowMs - lastSaveMs_ >= kCounterSaveIntervalMs) {
        lastSaveMs_ = nowMs;
        saveIfChanged();
    }
}

BaseCounters CountersNvs::snapshot() {
    BaseCounters snap;
    MutexGuard lock(mutex_);
    if (lock.locked()) {
        snap = counters_;
    }
    return snap;
}

void CountersNvs::incrementReceived(bool bufferDrop) {
    MutexGuard lock(mutex_);
    if (lock.locked()) {
        counters_.received++;
        if (bufferDrop) {
            counters_.bufferDropped++;
        }
    }
}

void CountersNvs::incrementFramesInvalid() {
    MutexGuard lock(mutex_);
    if (lock.locked()) {
        counters_.framesInvalid++;
    }
}

void CountersNvs::incrementRadioResyncs() {
    MutexGuard lock(mutex_);
    if (lock.locked()) {
        counters_.radioResyncs++;
    }
}

void CountersNvs::recordSent(uint32_t count, int httpCode) {
    MutexGuard lock(mutex_);
    if (lock.locked()) {
        counters_.sent += count;
        counters_.lastHttpCode = httpCode;
    }
}

void CountersNvs::recordDropped(uint32_t count, int httpCode) {
    MutexGuard lock(mutex_);
    if (lock.locked()) {
        counters_.dropped += count;
        if (httpCode != 0) {
            counters_.lastHttpCode = httpCode;
        }
    }
}

void CountersNvs::setLastHttpCode(int httpCode) {
    MutexGuard lock(mutex_);
    if (lock.locked()) {
        counters_.lastHttpCode = httpCode;
    }
}

void CountersNvs::setLastNodeHealth(const SystemHealthData& health) {
    MutexGuard lock(mutex_);
    if (lock.locked()) {
        lastNodeHealth_ = health;
        hasNodeHealth_ = true;
    }
}

bool CountersNvs::getLastNodeHealth(SystemHealthData& out) {
    MutexGuard lock(mutex_);
    if (!lock.locked() || !hasNodeHealth_) {
        return false;
    }
    out = lastNodeHealth_;
    return true;
}
