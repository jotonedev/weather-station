#include <stdio.h>
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "driver/gpio.h"
#include "sdkconfig.h"


/* the app_main entry point **must** have C linkage. Thus extern "C". */
extern "C" void app_main()
{
    printf("Entry point\n");
}
