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

    size_t const updated = outbox.backfillTimestamps(1700000100u, 30000u);
    EXPECT_EQ(updated, 2);

    PendingReading out;
    ASSERT_TRUE(outbox.pop(out));
    EXPECT_EQ(out.receivedAt, 1700000080u);
    ASSERT_TRUE(outbox.pop(out));
    EXPECT_EQ(out.receivedAt, 1700000050u);
    ASSERT_TRUE(outbox.pop(out));
    EXPECT_EQ(out.receivedAt, 1700000095u);
}

TEST(Outbox, NullExternalBufferForcesZeroCapacity) {
    Outbox outbox(nullptr, 5);
    EXPECT_EQ(outbox.capacity(), 0u);
    EXPECT_TRUE(outbox.empty());
    EXPECT_TRUE(outbox.full());
    EXPECT_EQ(outbox.count(), 0u);

    EXPECT_FALSE(outbox.push(makeEntry(1)));
    EXPECT_EQ(outbox.dropped(), 1u);

    PendingReading out = {};
    EXPECT_FALSE(outbox.pop(out));
    EXPECT_FALSE(outbox.peek(out));

    EXPECT_FALSE(outbox.restoreState(0, 0, 3));
    EXPECT_EQ(outbox.dropped(), 0u);
    EXPECT_EQ(outbox.backfillTimestamps(1700000100u, 5000u), 0u);
}

TEST(Outbox, RestoreStateCapacityZeroAndWrappedBuffer) {
    Outbox zeroOwned(0);
    EXPECT_FALSE(zeroOwned.restoreState(0, 0, 2));
    EXPECT_EQ(zeroOwned.dropped(), 0u);

    PendingReading backing[4] = {};
    Outbox src(backing, 4);
    for (uint16_t i = 1; i <= 6; ++i) {
        ASSERT_TRUE(src.push(makeEntry(i)));
    }
    PendingReading popped = {};
    ASSERT_TRUE(src.pop(popped));
    EXPECT_EQ(popped.sequence, 3u);
    ASSERT_EQ(src.head(), 3u);
    ASSERT_EQ(src.count(), 3u);
    ASSERT_EQ(src.dropped(), 2u);

    Outbox restored(backing, 4);
    ASSERT_TRUE(restored.restoreState(src.head(), src.count(), src.dropped()));
    EXPECT_EQ(restored.head(), 3u);
    EXPECT_EQ(restored.count(), 3u);
    EXPECT_EQ(restored.dropped(), 2u);

    ASSERT_TRUE(restored.push(makeEntry(7)));
    for (uint16_t expected = 4; expected <= 7; ++expected) {
        ASSERT_TRUE(restored.pop(popped));
        EXPECT_EQ(popped.sequence, expected);
    }
    EXPECT_TRUE(restored.empty());

    ASSERT_TRUE(restored.restoreState(1, 2, 3));
    EXPECT_FALSE(restored.restoreState(4, 1, 9));
    EXPECT_TRUE(restored.empty());
    EXPECT_EQ(restored.head(), 0u);
    EXPECT_EQ(restored.count(), 0u);
    EXPECT_EQ(restored.dropped(), 0u);

    ASSERT_TRUE(restored.restoreState(1, 2, 3));
    EXPECT_FALSE(restored.restoreState(0, 5, 9));
    EXPECT_TRUE(restored.empty());
    EXPECT_EQ(restored.head(), 0u);
    EXPECT_EQ(restored.count(), 0u);
    EXPECT_EQ(restored.dropped(), 0u);
}

TEST(Outbox, BackfillTimestampsIgnoresZeroEpochAndEmptyOutbox) {
    Outbox emptyBox(3);
    EXPECT_EQ(emptyBox.backfillTimestamps(1700000100u, 10000u), 0u);

    Outbox outbox(3);
    PendingReading entry = makeEntry(1);
    entry.receivedAt = 0;
    entry.receivedMs = 5000u;
    ASSERT_TRUE(outbox.push(entry));

    EXPECT_EQ(outbox.backfillTimestamps(0u, 10000u), 0u);
    PendingReading peeked = {};
    ASSERT_TRUE(outbox.peek(peeked));
    EXPECT_EQ(peeked.receivedAt, 0u);
}

TEST(Outbox, BackfillTimestampsHandlesMillisWraparoundAndEpochUnderflow) {
    Outbox wrapBox(3);
    PendingReading dummy = makeEntry(0);
    ASSERT_TRUE(wrapBox.push(dummy));
    ASSERT_TRUE(wrapBox.pop(dummy));

    PendingReading e1 = makeEntry(1);
    e1.receivedAt = 0;
    e1.receivedMs = 0xFFFFF000u;
    PendingReading e2 = makeEntry(2);
    e2.receivedAt = 1700000010u;
    e2.receivedMs = 1000u;
    PendingReading e3 = makeEntry(3);
    e3.receivedAt = 1700000020u;
    e3.receivedMs = 2000u;

    ASSERT_TRUE(wrapBox.push(e1));
    ASSERT_TRUE(wrapBox.push(e2));
    ASSERT_TRUE(wrapBox.push(e3));

    EXPECT_EQ(wrapBox.backfillTimestamps(1700000100u, 6000u), 1u);
    PendingReading out = {};
    ASSERT_TRUE(wrapBox.pop(out));
    EXPECT_EQ(out.receivedAt, 1700000090u);

    Outbox underflowBox(3);
    ASSERT_TRUE(underflowBox.push(dummy));
    ASSERT_TRUE(underflowBox.pop(dummy));

    PendingReading eB = makeEntry(10);
    eB.receivedAt = 0;
    eB.receivedMs = 10000u;
    PendingReading eC = makeEntry(11);
    eC.receivedAt = 0;
    eC.receivedMs = 0u;
    PendingReading eD = makeEntry(12);
    eD.receivedAt = 0;
    eD.receivedMs = 15000u;

    ASSERT_TRUE(underflowBox.push(eB));
    ASSERT_TRUE(underflowBox.push(eC));
    ASSERT_TRUE(underflowBox.push(eD));

    EXPECT_EQ(underflowBox.backfillTimestamps(10u, 20000u), 1u);
    ASSERT_TRUE(underflowBox.pop(out));
    EXPECT_EQ(out.receivedAt, 0u);
    ASSERT_TRUE(underflowBox.pop(out));
    EXPECT_EQ(out.receivedAt, 0u);
    ASSERT_TRUE(underflowBox.pop(out));
    EXPECT_EQ(out.receivedAt, 5u);
}

TEST(Outbox, ResetClearsEntriesButPreservesDroppedCount) {
    Outbox outbox(2);
    ASSERT_TRUE(outbox.push(makeEntry(1)));
    ASSERT_TRUE(outbox.push(makeEntry(2)));
    ASSERT_TRUE(outbox.push(makeEntry(3)));
    EXPECT_EQ(outbox.dropped(), 1u);

    outbox.reset();
    EXPECT_TRUE(outbox.empty());
    EXPECT_EQ(outbox.count(), 0u);
    PendingReading out = {};
    EXPECT_FALSE(outbox.pop(out));
    EXPECT_EQ(outbox.dropped(), 1u);

    ASSERT_TRUE(outbox.push(makeEntry(4)));
    ASSERT_TRUE(outbox.push(makeEntry(5)));
    ASSERT_TRUE(outbox.push(makeEntry(6)));
    EXPECT_EQ(outbox.dropped(), 2u);
    ASSERT_TRUE(outbox.pop(out));
    EXPECT_EQ(out.sequence, 5u);
    ASSERT_TRUE(outbox.pop(out));
    EXPECT_EQ(out.sequence, 6u);
}
