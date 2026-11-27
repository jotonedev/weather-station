#include "outbox_sender.h"

#include <Arduino.h>
#include <WiFi.h>
#include <freertos/FreeRTOS.h>
#include <freertos/task.h>
#include <time.h>

#include <Ingest.h>
#include <Outbox.h>

#include "boot_watchdog.h"
#include "config.h"
#include "counters_nvs.h"
#include "rtc_outbox.h"
#include "wifi_ntp_http.h"

namespace {

struct SenderTaskArgs {
    RtcOutbox* outbox;
    CountersNvs* counters;
    const char* nodeId;
    volatile bool* clockSynced;
    volatile uint32_t* heartbeatMs;
};

SenderTaskArgs s_senderArgs = {};

void outboxSenderTask(void* param) {
    auto* args = static_cast<SenderTaskArgs*>(param);
    RtcOutbox& outbox = *args->outbox;
    CountersNvs& counters = *args->counters;
    const char* nodeId = args->nodeId;
    volatile bool& clockSynced = *args->clockSynced;
    volatile uint32_t& heartbeatMs = *args->heartbeatMs;

    static PendingReading batch[BATCH_FLUSH_MAX];
    static IngestReading ingestBatch[BATCH_FLUSH_MAX];
    static char payload[INGEST_BUFFER_SIZE];

    BackoffPolicy backoff = {BACKOFF_MIN_MS, BACKOFF_MAX_MS, BACKOFF_MULTIPLIER, 0};
    uint8_t retryIndex = 0;
    heartbeatMs = millis();

    while (true) {
        heartbeatMs = millis();
        if (WiFi.status() != WL_CONNECTED) {
            ensureWifi(heartbeatMs);
            retryIndex = 0;
            if (WiFi.status() != WL_CONNECTED) {
                delayWithHeartbeat(heartbeatMs, WIFI_RECONNECT_DELAY_MS);
                continue;
            }
        }

        if (!clockSynced) {
            retryIndex = 0;
            counters.snapshotIfDue(millis());
            delayWithHeartbeat(heartbeatMs, FLUSH_IDLE_DELAY_MS);
            continue;
        }

        uint32_t nowEpoch = static_cast<uint32_t>(time(nullptr));
        bool syncedNow = nowEpoch >= static_cast<uint32_t>(kClockSyncedEpoch);
        size_t batchCount = outbox.peekBatch(batch, BATCH_FLUSH_MAX, syncedNow, nowEpoch, millis());
        if (batchCount == 0) {
            retryIndex = 0;
            counters.snapshotIfDue(millis());
            delayWithHeartbeat(heartbeatMs, FLUSH_IDLE_DELAY_MS);
            continue;
        }

        bool allTimestamped = true;
        for (size_t i = 0; i < batchCount; i++) {
            if (batch[i].receivedAt == 0) {
                allTimestamped = false;
                break;
            }
            ingestBatch[i].data = batch[i].data;
            ingestBatch[i].epoch = batch[i].receivedAt;
        }
        if (!allTimestamped) {
            counters.snapshotIfDue(millis());
            delayWithHeartbeat(heartbeatMs, FLUSH_IDLE_DELAY_MS);
            continue;
        }

        int payloadLength =
            buildBatchPayload(ingestBatch, batchCount, nodeId, payload, sizeof(payload));
        if (payloadLength <= 0) {
            // Payload builder should never fail with valid inputs, drop as a
            // safety net so the outbox drains.
            outbox.popCount(batchCount);
            counters.recordDropped(static_cast<uint32_t>(batchCount), 0);
            retryIndex = 0;
            continue;
        }

        int httpCode = postToIngest(INGEST_URL, payload, heartbeatMs);

        if (httpCode >= 200 && httpCode < 300) {
            outbox.popCount(batchCount);
            counters.recordSent(static_cast<uint32_t>(batchCount), httpCode);
            retryIndex = 0;
            markHealthyBoot();
        } else if (isTransientHttpStatus(httpCode)) {
            counters.setLastHttpCode(httpCode);
            // Keep the batch buffered and retry with backoff.
            retryIndex++;
            uint32_t delayMs = backoffDelayMs(backoff, retryIndex);
            delayWithHeartbeat(heartbeatMs, delayMs);
        } else {
            // Permanent client error, the payload will never succeed. Drop the
            // oldest entry and try the rest on the next pass.
            outbox.popCount(1);
            counters.recordDropped(1, httpCode);
            retryIndex = 0;
        }
        counters.snapshotIfDue(millis());
    }
}

} // namespace

void startOutboxSenderTask(RtcOutbox& outbox, CountersNvs& counters, const char* nodeId,
                           volatile bool& clockSynced, volatile uint32_t& senderHeartbeatMs) {
    s_senderArgs.outbox = &outbox;
    s_senderArgs.counters = &counters;
    s_senderArgs.nodeId = nodeId;
    s_senderArgs.clockSynced = &clockSynced;
    s_senderArgs.heartbeatMs = &senderHeartbeatMs;
    xTaskCreatePinnedToCore(outboxSenderTask, "OutboxSender", 8192, &s_senderArgs, 1, nullptr, 0);
}
