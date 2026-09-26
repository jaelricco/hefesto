import Foundation
import GRDB
import Testing
@testable import HefestoStore

/// A session with one block, the common starting point.
func seeded() throws -> (AppDatabase, Session, Block) {
    let db = try AppDatabase.inMemory()
    let session = Session(startedAt: Date(), timezone: "Europe/Zurich", localDate: "2026-09-25")
    try db.saveSession(session)
    let block = Block(sessionId: session.id, orderIndex: 0)
    try db.saveBlock(block)
    return (db, session, block)
}

func pullUps(_ set: SetEntry, reps: Int) -> SetElement {
    SetElement(setEntryId: set.id, orderIndex: 0, exerciseId: "pull-up-id", measure: "reps", reps: reps)
}

@Suite struct OutboxTests {
    @Test func writesQueueOpsInOrderParentsFirst() throws {
        let (db, session, block) = try seeded()
        let entry = SetEntry(sessionId: session.id, blockId: block.id, orderIndex: 0)
        try db.saveSet(SetWithElements(entry: entry, elements: [pullUps(entry, reps: 8)]))

        let ops = try db.queuedOps()
        #expect(ops.map(\.entity) == [.session, .block, .set])
        #expect(ops.map(\.op) == [.put, .put, .put])
        #expect(ops[2].sessionId == session.id)
    }

    @Test func aNewerPutReplacesTheQueuedOneInPlace() throws {
        let (db, session, _) = try seeded()
        var s = session
        s.title = "Pull day"
        try db.saveSession(s)

        let ops = try db.queuedOps()
        #expect(ops.count == 2, "still one op for the session")
        #expect(ops.first?.entity == .session, "and still ahead of its block")
        let stored = try db.reader.read { try Session.fetchOne($0, key: session.id) }
        #expect(stored?.title == "Pull day")
    }

    @Test func aDeleteReplacesAQueuedPut() throws {
        let (db, _, block) = try seeded()
        try db.deleteBlock(id: block.id, sessionId: block.sessionId)
        let ops = try db.queuedOps()
        #expect(ops.count == 2)
        #expect(ops[1].entity == .block && ops[1].op == .delete)
    }

    @Test func completionIsQueuedAfterThePutsItFollows() throws {
        let (db, session, _) = try seeded()
        try db.completeSession(id: session.id, perceivedFatigue: 6)
        var s = try #require(try db.reader.read { try Session.fetchOne($0, key: session.id) })
        #expect(s.status == "completed")
        s.notes = "felt strong"
        try db.saveSession(s)

        let ops = try db.queuedOps()
        #expect(ops.map(\.op) == [.put, .put, .complete], "the later put coalesces into the first, before the completion")
        // Completing twice queues nothing new.
        try db.completeSession(id: session.id)
        #expect(try db.queuedOps().count == 3)
    }

    @Test func aBatchIsFrozenAndSettledPerOp() throws {
        let (db, session, block) = try seeded()
        let ops = try db.queuedOps()
        try db.startBatch(key: "k-1", ops: ops.map { ($0.seq!, Data("{}".utf8)) })
        #expect(try db.queuedOps().isEmpty)
        let inFlight = try #require(try db.batchInFlight())
        #expect(inFlight.key == "k-1" && inFlight.ops.count == 2)
        #expect(inFlight.ops.allSatisfy { $0.payload == Data("{}".utf8) })

        // A write during the flight queues a new op beside the frozen one.
        var b = block
        b.notes = "later"
        try db.saveBlock(b)
        #expect(try db.queuedOps().count == 1)

        let superseded = try db.finishBatch(key: "k-1", outcomes: [
            ops[0].seq!: .applied,
            ops[1].seq!: .rejected(type: "validation", detail: "bad"),
        ])
        #expect(superseded.isEmpty)
        #expect(try db.batchInFlight()?.key == nil)
        #expect(try db.queuedOps().count == 1, "the op queued during the flight survives")
        #expect(try db.syncProblems().first?.rowId == block.id)
        _ = session
    }

    @Test func supersededOpsAreReturnedForRefetch() throws {
        let (db, session, _) = try seeded()
        let ops = try db.queuedOps()
        try db.startBatch(key: "k", ops: ops.map { ($0.seq!, nil) })
        let superseded = try db.finishBatch(key: "k", outcomes: [ops[0].seq!: .superseded])
        #expect(superseded.map(\.rowId) == [session.id])
    }
}

@Suite struct SetTests {
    @Test func aComboIsOneSetWithOrderedElements() throws {
        let (db, session, block) = try seeded()
        let entry = SetEntry(sessionId: session.id, blockId: block.id, orderIndex: 0)
        let a = SetElement(setEntryId: entry.id, orderIndex: 9, exerciseId: "tuck", measure: "hold_seconds", holdSeconds: 12)
        let b = pullUps(entry, reps: 5)
        try db.saveSet(SetWithElements(entry: entry, elements: [a, b]))

        let tree = try #require(try db.reader.read { try AppDatabase.sessionTree($0, id: session.id) })
        let set = try #require(tree.blocks.first?.sets.first)
        #expect(set.elements.map(\.exerciseId) == ["tuck", "pull-up-id"])
        #expect(set.elements.map(\.orderIndex) == [0, 1], "array order is element order")

        // Rewriting with one element removes the other.
        try db.saveSet(SetWithElements(entry: entry, elements: [b]))
        let again = try #require(try db.reader.read { try AppDatabase.sessionTree($0, id: session.id) })
        #expect(again.blocks[0].sets[0].elements.map(\.id) == [b.id])
    }

    @Test func aSetNeedsAnElement() throws {
        let (db, session, block) = try seeded()
        let entry = SetEntry(sessionId: session.id, blockId: block.id, orderIndex: 0)
        #expect(throws: StoreError.emptySet) { try db.saveSet(SetWithElements(entry: entry, elements: [])) }
    }

    @Test func lastSetFindsTheMostRecentPerformedSet() throws {
        let (db, session, block) = try seeded()
        let early = SetEntry(sessionId: session.id, blockId: block.id, orderIndex: 0, completedAt: Date(timeIntervalSinceNow: -60))
        let late = SetEntry(sessionId: session.id, blockId: block.id, orderIndex: 1, completedAt: Date())
        try db.saveSet(SetWithElements(entry: early, elements: [pullUps(early, reps: 5)]))
        try db.saveSet(SetWithElements(entry: late, elements: [pullUps(late, reps: 7)]))
        let last = try db.reader.read { try AppDatabase.lastSet($0, exerciseId: "pull-up-id") }
        #expect(last?.entry.id == late.id)
        #expect(last?.elements.first?.reps == 7)
    }

    @Test func deletingASessionTombstonesItsTree() throws {
        let (db, session, block) = try seeded()
        let entry = SetEntry(sessionId: session.id, blockId: block.id, orderIndex: 0)
        try db.saveSet(SetWithElements(entry: entry, elements: [pullUps(entry, reps: 5)]))
        try db.deleteSession(id: session.id)
        #expect(try db.reader.read { try AppDatabase.sessionTree($0, id: session.id) } == nil)
        #expect(try db.reader.read { try AppDatabase.recentSessions($0) }.isEmpty)
    }
}

@Suite struct PullTests {
    @Test func aPageUpdatesRowsWithoutQueuedOpsAndAdvancesTheCursor() throws {
        let db = try AppDatabase.inMemory()
        var theirs = Session(startedAt: Date(), timezone: "UTC", localDate: "2026-09-25", title: "from the server", serverSeq: 7)
        try db.apply(ServerPage(cursor: 7, sessions: [theirs]))
        #expect(try db.cursor() == 7)

        // A local edit is queued; a later page must not overwrite it.
        var mine = theirs
        mine.title = "mine"
        try db.saveSession(mine)
        theirs.title = "server again"
        theirs.serverSeq = 9
        try db.apply(ServerPage(cursor: 9, sessions: [theirs]))
        let stored = try db.reader.read { try Session.fetchOne($0, key: theirs.id) }
        #expect(stored?.title == "mine")
        #expect(try db.cursor() == 9)

        // The cursor never goes back.
        try db.apply(ServerPage(cursor: 3))
        #expect(try db.cursor() == 9)
    }

    @Test func aPulledSetReplacesItsElements() throws {
        let (db, session, block) = try seeded()
        let entry = SetEntry(sessionId: session.id, blockId: block.id, orderIndex: 0)
        let old = pullUps(entry, reps: 5)
        try db.saveSet(SetWithElements(entry: entry, elements: [old]))
        // The op went out and was applied.
        let ops = try db.queuedOps()
        try db.startBatch(key: "k", ops: ops.map { ($0.seq!, nil) })
        try db.finishBatch(key: "k", outcomes: [:])

        let replacement = SetElement(setEntryId: entry.id, orderIndex: 0, exerciseId: "dip-id", measure: "reps", reps: 10)
        try db.apply(ServerPage(cursor: 20, sets: [SetWithElements(entry: entry, elements: [replacement])]))
        let tree = try #require(try db.reader.read { try AppDatabase.sessionTree($0, id: session.id) })
        #expect(tree.blocks[0].sets[0].elements.map(\.exerciseId) == ["dip-id"])
    }

    @Test func aMissingServerTreeDeletesTheLocalOne() throws {
        let (db, session, _) = try seeded()
        try db.replaceSessionTree(id: session.id, with: nil)
        #expect(try db.reader.read { try AppDatabase.sessionTree($0, id: session.id) } == nil)
    }
}

@Suite struct UUIDv7Tests {
    @Test func idsAreVersion7AndSortByTime() {
        let a = UUIDv7.make(now: Date(timeIntervalSince1970: 1_000))
        let b = UUIDv7.make(now: Date(timeIntervalSince1970: 2_000))
        #expect(a < b)
        #expect(a.count == 36)
        let chars = Array(a)
        #expect(chars[14] == "7")
        #expect(UUID(uuidString: a) != nil)
    }
}
