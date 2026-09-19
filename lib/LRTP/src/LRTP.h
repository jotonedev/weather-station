
enum LRTPMessageType : uint8_t {
    DATA_LINK_MESSAGE_TYPE_COMMAND = 0x00,
    DATA_LINK_MESSAGE_TYPE_ACKNOWLEDGE = 0x01,
    DATA_LINK_MESSAGE_TYPE_EVENT = 0x02,
    DATA_LINK_MESSAGE_TYPE_ACK = 0x03,
    DATA_LINK_MESSAGE_TYPE_NACK = 0x04,
};

/**
 * @brief Represents a data link message for the LRTP protocol.
 * @details This struct is used to define the format of messages exchanged between nodes in the LRTP network.
 */
struct LRTPMessage {
    uint8_t version;
    uint8_t type;
    uint8_t target;
    uint8_t length;
    uint8_t payload[0];
};

class LRTP {
    public:
        LRTP();
        ~LRTP();
};
