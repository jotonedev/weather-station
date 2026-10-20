#include <gtest/gtest.h>

#include <cstdint>

#include "Outbox.h"

namespace {

PendingReading makeEntry(uint16_t sequence) {
    PendingReading entry = {};
    entry.sequence = sequence;
    entry.receivedAt = 1000 + sequence;
    entry.data.airData.temperature = weather::encodeTemperature(20.0f + sequence);
    return entry;
}

} // namespace

TEST(Outbox, PushPopInOrder) {
    Outbox outbox(3);
    EXPECT_TRUE(outbox.empty());
    EXPECT_EQ(outbox.count(), 0);

    EXPECT_TRUE(outbox.push(makeEntry(1)));
    EXPECT_TRUE(outbox.push(makeEntry(2)));
    EXPECT_TRUE(outbox.push(makeEntry(3)));
    EXPECT_TRUE(outbox.full());
    EXPECT_FALSE(outbox.empty());

    PendingReading out;
    ASSERT_TRUE(outbox.pop(out));
    EXPECT_EQ(out.sequence, 1);
    ASSERT_TRUE(outbox.pop(out));
    EXPECT_EQ(out.sequence, 2);
    ASSERT_TRUE(outbox.pop(out));
    EXPECT_EQ(out.sequence, 3);
    EXPECT_TRUE(outbox.empty());
    EXPECT_FALSE(outbox.pop(out));
}

TEST(Outbox, OverflowDropsOldest) {
    Outbox outbox(2);
    outbox.push(makeEntry(1));
    outbox.push(makeEntry(2));
    outbox.push(makeEntry(3));

    EXPECT_EQ(outbox.count(), 2);
    EXPECT_EQ(outbox.dropped(), 1);

    PendingReading out;
    ASSERT_TRUE(outbox.pop(out));
    EXPECT_EQ(out.sequence, 2);
}

TEST(Outbox, PeekDoesNotRemove) {
    Outbox outbox(2);
    outbox.push(makeEntry(7));
    outbox.push(makeEntry(8));

    PendingReading out;
    ASSERT_TRUE(outbox.peek(out));
    EXPECT_EQ(out.sequence, 7);
    EXPECT_EQ(outbox.count(), 2);

    ASSERT_TRUE(outbox.peekAt(1, out));
    EXPECT_EQ(out.sequence, 8);
    EXPECT_FALSE(outbox.peekAt(2, out));
}

TEST(Outbox, WrapAroundKeepsOrder) {
    Outbox outbox(3);
    for (uint16_t i = 1; i <= 7; ++i) {
        outbox.push(makeEntry(i));
    }
    EXPECT_EQ(outbox.count(), 3);
    EXPECT_EQ(outbox.dropped(), 4);

    PendingReading out;
    ASSERT_TRUE(outbox.pop(out));
    EXPECT_EQ(out.sequence, 5);
    ASSERT_TRUE(outbox.pop(out));
    EXPECT_EQ(out.sequence, 6);
    ASSERT_TRUE(outbox.pop(out));
    EXPECT_EQ(out.sequence, 7);
    EXPECT_TRUE(outbox.empty());
}

TEST(Outbox, ResetClearsBuffer) {
    Outbox outbox(2);
    outbox.push(makeEntry(1));
    outbox.push(makeEntry(2));
    outbox.reset();
    EXPECT_TRUE(outbox.empty());
    EXPECT_EQ(outbox.dropped(), 0);
}

TEST(Outbox, ZeroCapacityDropsEverything) {
    Outbox outbox(0);
    EXPECT_EQ(outbox.capacity(), 0);
    EXPECT_FALSE(outbox.push(makeEntry(1)));
    EXPECT_EQ(outbox.dropped(), 1);
}

TEST(Outbox, ExternalBufferAndRestoreState) {
    PendingReading backing[4] = {};
    Outbox outbox(backing, 4);
    EXPECT_TRUE(outbox.push(makeEntry(10)));
    EXPECT_TRUE(outbox.push(makeEntry(11)));
    PendingReading popped;
    ASSERT_TRUE(outbox.pop(popped));
    EXPECT_EQ(popped.sequence, 10);
    EXPECT_EQ(outbox.head(), 1);
    EXPECT_EQ(outbox.count(), 1);

    Outbox restored(backing, 4);
    ASSERT_TRUE(restored.restoreState(1, 1, 5));
    EXPECT_EQ(restored.count(), 1);
    EXPECT_EQ(restored.dropped(), 5);
    ASSERT_TRUE(restored.pop(popped));
    EXPECT_EQ(popped.sequence, 11);

    EXPECT_FALSE(restored.restoreState(4, 1, 0));
    EXPECT_FALSE(restored.restoreState(0, 5, 0));
}

TEST(Outbox, BackfillTimestampsUpdatesOnlyZeroEpochEntries) {
    Outbox outbox(3);
    PendingReading e1 = makeEntry(1);
    e1.receivedAt = 0;
    e1.receivedMs = 10000;

    PendingReading e2 = makeEntry(2);
    e2.receivedAt = 1700000050u;
    e2.receivedMs = 20000;

    PendingReading e3 = makeEntry(3);
    e3.receivedAt = 0;
    e3.receivedMs = 25000;

    outbox.push(e1);
    outbox.push(e2);
    outbox.push(e3);

    size_t updated = outbox.backfillTimestamps(1700000100u, 30000u);
    EXPECT_EQ(updated, 2);

    PendingReading out;
    ASSERT_TRUE(outbox.pop(out));
    EXPECT_EQ(out.receivedAt, 1700000080u);
    ASSERT_TRUE(outbox.pop(out));
    EXPECT_EQ(out.receivedAt, 1700000050u);
    ASSERT_TRUE(outbox.pop(out));
    EXPECT_EQ(out.receivedAt, 1700000095u);
}
