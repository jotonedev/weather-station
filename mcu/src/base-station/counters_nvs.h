#ifndef BASE_COUNTERS_NVS_H
#define BASE_COUNTERS_NVS_H

#include <cstdint>
#include <freertos/FreeRTOS.h>
#include <freertos/semphr.h>

#include <Weather.h>

struct BaseCounters {
    uint32_t received = 0;
    uint32_t sent = 0;
    uint32_t dropped = 0;
    uint32_t framesInvalid = 0;
    uint32_t bufferDropped = 0;
    uint32_t radioResyncs = 0;
    int lastHttpCode = 0;
};

class CountersNvs {
  public:
    CountersNvs();

    void init();
    void load();
    void saveIfChanged();
    void snapshotIfDue(uint32_t nowMs);
    BaseCounters snapshot();

    void incrementReceived(bool bufferDrop);
    void incrementFramesInvalid();
    void incrementRadioResyncs();
    void recordSent(uint32_t count, int httpCode);
    void recordDropped(uint32_t count, int httpCode);
    void setLastHttpCode(int httpCode);

    void setLastNodeHealth(const SystemHealthData& health);
    bool getLastNodeHealth(SystemHealthData& out);

  private:
    SemaphoreHandle_t mutex_;
    BaseCounters counters_;
    BaseCounters lastSaved_;
    uint32_t lastSaveMs_;
    bool hasNodeHealth_;
    SystemHealthData lastNodeHealth_;
};

#endif // BASE_COUNTERS_NVS_H
