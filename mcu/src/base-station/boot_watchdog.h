#ifndef BASE_BOOT_WATCHDOG_H
#define BASE_BOOT_WATCHDOG_H

#include <cstdint>

void setupWatchdog();
void loadBootCounter();
void markHealthyBoot();
void storeLastError(const char* tag);
uint32_t getBootCount();
bool tasksHealthy(uint32_t senderHb, uint32_t ntpHb);

#endif // BASE_BOOT_WATCHDOG_H
