#include <Arduino.h>
#include <ARQ.h>

#define RADIO_CS_PIN   16
#define RADIO_GDO0_PIN 18
#define SCK_PIN        12
#define MISO_PIN       13
#define MOSI_PIN       11

constexpr ARQ::address_t CONTROL_NODE_ADDR = 1;
constexpr ARQ::address_t BASE_STATION_ADDR = 2;

CC1101::Radio radio(RADIO_CS_PIN, SCK_PIN, MISO_PIN, MOSI_PIN, RADIO_GDO0_PIN);
ARQ::ARQ arq(radio, BASE_STATION_ADDR);

void setup() {
    Serial.begin(115200);

    Serial.println("\n==================================");
    Serial.println("  Base Station Starting (COM6)");
    Serial.println("==================================");

    ARQ::Status initStatus = arq.initializeCC1101();
    if (initStatus != ARQ::STATUS_OK) {
        Serial.printf("[ERROR] Radio initialization failed! Error code: %d\n", initStatus);
        Serial.printf("       CC1101 Part Number read: 0x%02X (expected 0x00)\n", radio.getChipPartNumber());
        Serial.printf("       CC1101 Version read:     0x%02X (expected 0x14 or 0x04)\n", radio.getChipVersion());
        Serial.println("       Tip: If reading 0x00 or 0xFF, check MISO/MOSI (SO/SI), SCK, CS wiring and 3.3V power.");
    } else {
        delay(100);
        Serial.println("[OK] Radio initialized successfully.");
        ARQ::Status rxStatus = arq.startReceiveData();
        if (rxStatus == ARQ::STATUS_OK) {
            Serial.println("[OK] Listening mode active. Entering main loop...");
        } else {
            Serial.printf("[WARN] startReceiveData returned status %d. Entering main loop...\n", rxStatus);
            uint8_t marcstate = radio.readReg(CC1101_REG_MARCSTATE) & 0x1F;
            Serial.printf("       Radio State: %d (0=IDLE, 1=RX, 2=TX, 4=CAL, 5=SETTLING, 6=OVF)\n", radio.getState());
            Serial.printf("       MARCSTATE: 0x%02X (0x0D is RX, 0x01 is IDLE, 0x11 is RX_OVF)\n", marcstate);
        }
    }
}

static uint32_t lastHeartbeat = 0;

void loop() {
    arq.poll();

    // Heartbeat every 3 seconds to confirm the board is active and monitor RF state
    if (millis() - lastHeartbeat >= 3000) {
        lastHeartbeat = millis();
        uint8_t rxBytes = radio.readReg(CC1101_REG_RXBYTES) & 0x7F;
        uint8_t pktStatus = radio.readReg(CC1101_REG_PKTSTATUS);
        uint8_t marcState = radio.readReg(CC1101_REG_MARCSTATE) & 0x1F;
        int8_t curRssi = radio.getRSSI();
        Serial.printf("[Status] MARC: 0x%02X | RXBYTES: %u | PKTSTATUS: 0x%02X | GDO0 pin: %d | RSSI: %d dBm\n",
                      marcState, rxBytes, pktStatus, digitalRead(RADIO_GDO0_PIN), curRssi);
    }

    if (arq.available()) {
        ARQ::Message msg;
        arq.readMessage(msg);

        Serial.printf("[RX] Received: %s\n", reinterpret_cast<const char*>(msg.payload));
        Serial.printf("     RSSI: %d dBm | LQI: %u\n", arq.getLastRSSI(), arq.getLastLQI());
    }
}
