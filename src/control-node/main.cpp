#include <Arduino.h>
#include <ARQ.h>

CC1101::Radio radio(5);
ARQ::ARQ arq(radio, 1);

void setup() {
    arq.initializeCC1101();
}

void loop() {
    
}

