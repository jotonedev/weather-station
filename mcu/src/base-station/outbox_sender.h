#ifndef BASE_OUTBOX_SENDER_H
#define BASE_OUTBOX_SENDER_H

#include <cstdint>

class RtcOutbox;
class CountersNvs;

void startOutboxSenderTask(RtcOutbox& outbox, CountersNvs& counters, const char* nodeId,
                           volatile bool& clockSynced, volatile uint32_t& senderHeartbeatMs);

#endif // BASE_OUTBOX_SENDER_H
