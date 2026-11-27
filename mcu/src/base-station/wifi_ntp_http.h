#ifndef BASE_WIFI_NTP_HTTP_H
#define BASE_WIFI_NTP_HTTP_H

#include <cstdint>
#include <ctime>

class RtcOutbox;

// The clock counts as synced once it passes roughly 2020.
constexpr time_t kClockSyncedEpoch = 1600000000;

void delayWithHeartbeat(volatile uint32_t& heartbeatMs, uint32_t totalDelayMs);
void ensureWifi(volatile uint32_t& heartbeatMs);
bool syncClockOnce(volatile uint32_t& heartbeatMs, uint32_t timeoutMs);
int postToIngest(const char* url, const char* body, volatile uint32_t& heartbeatMs);
void startNtpSyncTask(RtcOutbox& outbox, volatile bool& clockSynced,
                      volatile uint32_t& ntpHeartbeatMs);

#endif // BASE_WIFI_NTP_HTTP_H
