#ifndef ARQ_H
#define ARQ_H

#include <cstdint>
#include <cc1101.h>

namespace ARQ {

typedef std::uint8_t byte; 
typedef std::uint8_t address_t; 
typedef std::uint16_t length_t;
typedef std::uint16_t seq_t;
typedef std::uint8_t version_t;


constexpr length_t MAX_PAYLOAD_SIZE = 256;
constexpr version_t PROTOCOL_VERSION = 1;
constexpr uint8_t HEADER_SIZE = 7;
constexpr uint8_t MAX_RETRIES = 3;


enum MessageType : byte {
    DATA_FRAME = 0xFF,
    ACK = 0x01,
    NACK = 0x04,
};

enum Status : byte {
  STATUS_OK = 0,
  STATUS_BUSY,
  STATUS_ERROR,
  STATUS_INVALID_PARAM,
  STATUS_CHIP_NOT_FOUND,
  STATUS_INVALID_SIZE,
  STATUS_TIMEOUT,
  STATUS_NACK_RECEIVED,
  STATUS_ACK_NOT_RECEIVED,
};

enum class RadioState {
    IDLE,
    TRANSMITTING,
    WAITING_ACK,
    RECEIVING,
    SENDING_ACK,
};

/**
 * @brief Represents a data link message for the ARQ protocol.
 */
struct __attribute__((packed)) Message {
    version_t version;
    address_t source;
    MessageType type;
    seq_t sequence;
    length_t length;
    const byte* payload;
};


/**
 * @brief Allows to send messages over the ARQ protocol
 */
class ARQ {
    public:
        explicit ARQ(CC1101::Radio radio, address_t localAddress = 0)
            : radio(radio), localAddress(localAddress) {}
        ~ARQ();

        ARQ(const ARQ&) = delete;
        ARQ& operator=(const ARQ&) = delete;

        Status initializeCC1101(uint8_t packetLength = 32);

        void setLocalAddress(address_t addr) { localAddress = addr; }
        address_t getLocalAddress() const { return localAddress; }

        Status startSendData(const byte* data, length_t length, address_t target);
        Status startReceiveData(address_t localAddress);
        Status startReceiveData();
        Status enterRxMode();

        // Main unified polling loop called inside void loop()
        Status poll(uint32_t ackTimeoutMs = 1000);

        // Internal received message access
        bool available() const { return hasNewMessage; }
        Status readMessage(Message& outMessage, byte* outPayloadBuffer = nullptr, length_t maxLen = 0);

        // Signal quality and link metrics
        int8_t getLastRSSI() const { return lastRssi; }
        uint8_t getLastLQI() const { return lastLqi; }

        // Transmission statistics
        uint32_t getTxSuccessCount() const { return txSuccessCount; }
        uint32_t getTxTimeoutCount() const { return txTimeoutCount; }
        uint32_t getTxRetryCount() const { return totalRetries; }
        uint32_t getLastRTT() const { return lastRTT; }

        RadioState getState() const { return currentState; }
    protected:
        CC1101::Radio radio;
        uint8_t packetLength = 59;
        address_t localAddress = 0;
        volatile RadioState currentState = RadioState::IDLE;
        volatile bool packetDone = false;
        uint32_t opStartTime = 0;

        // Performance & signal metrics
        int8_t lastRssi = 0;
        uint8_t lastLqi = 0;
        uint32_t txStartTime = 0;
        uint32_t lastRTT = 0;
        uint32_t txSuccessCount = 0;
        uint32_t txTimeoutCount = 0;
        uint32_t totalRetries = 0;

        // TX state
        uint8_t retryCount = 0;
        address_t currentTarget = 0;
        MessageType currentTxType = DATA_FRAME;
        seq_t currentTxSeq = 0;
        uint8_t* txBuffer = nullptr;

        // Internal RX message storage
        bool hasNewMessage = false;
        Message rxMessage;
        uint8_t* rxPayloadBuffer = nullptr;
        bool hasLastReceived = false;
        address_t lastReceivedSource = 0;
        seq_t lastReceivedSeq = 0;

        Status sendACK(address_t target, seq_t sequence);
        Status sendNACK(address_t target, seq_t sequence);
        Status sendMessage(Message& message, address_t target);

        // State handlers for poll()
        Status handleStateTransmitting();
        Status handleStateWaitingAck(uint32_t ackTimeoutMs);
        Status handleStateSendingAck();
        Status handleStateReceiving();

        static ARQ* instance;
        static void isrHandler();
        void handleInterrupt();
};
    
} // namespace ARQ

#endif // ARQ_H
