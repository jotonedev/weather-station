#include <Arduino.h>
#include <ARQ.h>

#define RADIO_CS_PIN   19
#define RADIO_GDO0_PIN 18
#define SCK_PIN        14
#define MISO_PIN       12
#define MOSI_PIN       13

constexpr ARQ::address_t CONTROL_NODE_ADDR = 1;
constexpr ARQ::address_t BASE_STATION_ADDR = 2;

CC1101::Radio radio(RADIO_CS_PIN, SCK_PIN, MISO_PIN, MOSI_PIN, RADIO_GDO0_PIN);
ARQ::ARQ arq(radio, CONTROL_NODE_ADDR);

uint32_t lastSendTime = 0;

void setup() {
    Serial.begin(115200);
    
    Serial.println("\n--- Control Node Starting ---");
    Serial.printf("Configured Pins: CS=%d, SCK=%d, MISO=%d, MOSI=%d, GDO0=%d\n",
                  RADIO_CS_PIN, SCK_PIN, MISO_PIN, MOSI_PIN, RADIO_GDO0_PIN);

    ARQ::Status initStatus = arq.initializeCC1101();
    if (initStatus != ARQ::STATUS_OK) {
        Serial.printf("[ERROR] Radio initialization failed! Error code: %d\n", initStatus);
        Serial.printf("       CC1101 Part Number read: 0x%02X (expected 0x00)\n", radio.getChipPartNumber());
        Serial.printf("       CC1101 Version read:     0x%02X (expected 0x14 or 0x04)\n", radio.getChipVersion());
        Serial.println("       Tip: If reading 0x00 or 0xFF, check MISO/MOSI (SO/SI), SCK, CS wiring and 3.3V power.");
    } else {
        Serial.println("[OK] Radio initialized successfully. Control Node ready.");
    }
}

static bool txPending = false;

void loop() {
    ARQ::Status status = arq.poll(2000);

    if (txPending) {
        if (status == ARQ::STATUS_OK) {
            txPending = false;
            Serial.printf("[ACK] Delivery confirmed! RTT: %u ms\n", arq.getLastRTT());
        } else if (status == ARQ::STATUS_TIMEOUT) {
            txPending = false;
            Serial.println("[TIMEOUT] Delivery failed, no ACK received.");
        }
    }

    if (!txPending && (millis() - lastSendTime >= 3500)) {
        lastSendTime = millis();

        const char message[] = "Hello World";
        Serial.println("\n[TX] Sending: Hello World");
        ARQ::Status sendStatus = arq.startSendData(reinterpret_cast<const uint8_t*>(message), sizeof(message), BASE_STATION_ADDR);
        if (sendStatus == ARQ::STATUS_OK) {
            txPending = true;
        } else {
            Serial.printf("[TX ERROR] startSendData failed: %d\n", sendStatus);
        }
    }
}
