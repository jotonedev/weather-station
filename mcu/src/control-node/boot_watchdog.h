#ifndef CONTROL_NODE_BOOT_WATCHDOG_H
#define CONTROL_NODE_BOOT_WATCHDOG_H

#include <cstdint>

void setupPowerManagement();
void setupWatchdog();
void loadBootCounter();
void markHealthyBoot();
void storeLastError(const char* tag);

uint32_t getBootCount();
uint8_t getLastResetReason();
bool tasksHealthy(uint32_t anemoHb, uint32_t vaneHb, uint32_t rainHb);

#endif // CONTROL_NODE_BOOT_WATCHDOG_H
