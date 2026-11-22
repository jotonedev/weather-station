#ifndef SYNC_H
#define SYNC_H

#include <freertos/FreeRTOS.h>
#include <freertos/semphr.h>

// RAII guard for a FreeRTOS spinlock critical section. Keeps enter/exit balanced
// across all control-flow paths. Never call Preferences, I2C, mutexes, delay, or
// Serial while a CriticalSection is active.
class CriticalSection {
  public:
    explicit CriticalSection(portMUX_TYPE& mux) : mux_(mux) { portENTER_CRITICAL(&mux_); }
    ~CriticalSection() { portEXIT_CRITICAL(&mux_); }

    CriticalSection(const CriticalSection&) = delete;
    CriticalSection& operator=(const CriticalSection&) = delete;

  private:
    portMUX_TYPE& mux_;
};

// RAII guard for an optional FreeRTOS mutex handle. A null handle is a no-op.
class MutexGuard {
  public:
    explicit MutexGuard(SemaphoreHandle_t mutex) : mutex_(mutex) {
        if (mutex_ != nullptr) {
            taken_ = (xSemaphoreTake(mutex_, portMAX_DELAY) == pdTRUE);
        }
    }
    ~MutexGuard() {
        if (taken_) {
            xSemaphoreGive(mutex_);
        }
    }

    MutexGuard(const MutexGuard&) = delete;
    MutexGuard& operator=(const MutexGuard&) = delete;

    bool locked() const { return taken_; }

  private:
    SemaphoreHandle_t mutex_;
    bool taken_ = false;
};

#endif // SYNC_H
