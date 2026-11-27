#ifndef CONTROL_NODE_SENSOR_SAMPLER_H
#define CONTROL_NODE_SENSOR_SAMPLER_H

#include <cstdint>

#include <freertos/FreeRTOS.h>
#include <freertos/semphr.h>

#include <ARQ.h>
#include <Adafruit_AHTX0.h>
#include <Adafruit_BMP085.h>
#include <Anemometer.h>
#include <PMS.h>
#include <RainMeter.h>
#include <TxLink.h>
#include <Weather.h>
#include <WindVane.h>

void recoverWireBus(SemaphoreHandle_t wireMutex);

WeatherData sampleAllSensors(uint32_t nowMs, Anemometer& anemometer, WindVane& windVane,
                             RainMeter& rainMeter, Adafruit_AHTX0& aht, Adafruit_BMP085& bmp180,
                             Pms7003Driver& pms, SemaphoreHandle_t wireMutex, const ARQ::ARQ& arq,
                             const ARQ::TxLinkController& txLink, bool& sampledHasPms);

#endif // CONTROL_NODE_SENSOR_SAMPLER_H
