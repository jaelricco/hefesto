import Foundation
import Testing
@testable import HefestoLogger
import HefestoStore

/// A clock the tests move by hand.
final class TestClock: @unchecked Sendable {
    var now = Date(timeIntervalSince1970: 1_790_000_000)
    func advance(_ seconds: TimeInterval) { now = now.addingTimeInterval(seconds) }
}

@MainActor
func logger(_ clock: TestClock) throws -> (AppDatabase, LoggerModel) {
    let db = try AppDatabase.inMemory()
    let model = try LoggerModel.startSession(
        db: db, timeZone: TimeZone(identifier: "Europe/Zurich")!, title: "Pull day", now: { clock.now })
    return (db, model)
}

@Suite struct RestTimerTests {
    @Test func computesFromTheWallClock() {
        let start = Date(timeIntervalSince1970: 0)
        let t = RestTimer(startedAt: start, plannedSeconds: 90)
        #expect(t.elapsed(at: start.addingTimeInterval(30)) == 30)
        #expect(t.remaining(at: start.addingTimeInterval(30)) == 60)
        #expect(t.remaining(at: start.addingTimeInterval(300)) == 0, "never negative")
        #expect(t.isDone(at: start.addingTimeInterval(90)))
        #expect(t.endsAt == start.addingTimeInterval(90))
        #expect(RestTimer(startedAt: start, plannedSeconds: nil).remaining(at: start) == nil)
        #expect(t.elapsed(at: start.addingTimeInterval(-5)) == 0, "a clock going back does not go negative")
    }
}

@Suite @MainActor struct LoggerModelTests {
    @Test func startsADraftWithOneBlockOnTheLocalDay() throws {
        let clock = TestClock()
        clock.now = ISO8601DateFormatter().date(from: "2026-09-24T22:30:00Z")! // 00:30 on the 25th in Zurich
        let (_, model) = try logger(clock)
        #expect(model.session.status == "draft")
        #expect(model.session.localDate == "2026-09-25")
        #expect(model.tree.blocks.count == 1)
    }

    @Test func aPlainSetAndAComboUseTheSamePath() throws {
        let clock = TestClock()
        let (db, model) = try logger(clock)
        let block = model.tree.blocks[0].id
        try model.logSet(in: block, element: ElementDraft(exerciseId: "pull-up", measure: "reps", reps: 8))
        clock.advance(120)
        try model.logCombo(in: block, elements: [
            ElementDraft(exerciseId: "tuck-fl", measure: "hold_seconds", holdSeconds: 10),
            ElementDraft(exerciseId: "pull-up", measure: "reps", reps: 5),
        ])

        let sets = model.tree.blocks[0].sets
        #expect(sets.count == 2)
        #expect(sets.map(\.elements.count) == [1, 2])
        #expect(sets[1].elements.map(\.exerciseId) == ["tuck-fl", "pull-up"])
        #expect(sets.map(\.entry.orderIndex) == [0, 1])
        // Every write is queued for sync: session, block, two sets and the
        // first set rewritten with its actual rest (coalesced into its put).
        let ops = try db.queuedOps()
        #expect(ops.filter { $0.entity == .set }.count == 2)
    }

    @Test func theRestTimerStartsOnASetAndIsRecordedOnTheNext() throws {
        let clock = TestClock()
        let (_, model) = try logger(clock)
        let block = model.tree.blocks[0].id
        let first = try model.logSet(in: block, element: ElementDraft(exerciseId: "dip", measure: "reps", reps: 10), restPlannedSeconds: 90)
        #expect(model.rest?.plannedSeconds == 90)
        #expect(model.rest?.remaining(at: clock.now.addingTimeInterval(30)) == 60)

        clock.advance(105)
        try model.logSet(in: block, element: ElementDraft(exerciseId: "dip", measure: "reps", reps: 9))
        let firstSet = model.tree.blocks[0].sets.first { $0.id == first }
        #expect(firstSet?.entry.restAfterActualS == 105)
        #expect(model.rest?.plannedSeconds == model.defaultRestSeconds)

        model.skipRest()
        #expect(model.rest == nil)
    }

    @Test func repeatLastSetCopiesTheWholeSet() throws {
        let clock = TestClock()
        let (_, model) = try logger(clock)
        let block = model.tree.blocks[0].id
        let band = Assistance(type: "band", bandId: "red", bandCount: 1)
        try model.logCombo(in: block, elements: [
            ElementDraft(exerciseId: "pull-up", measure: "reps", reps: 6, assistance: band),
            ElementDraft(exerciseId: "scap-pull", measure: "reps", reps: 10),
        ])
        clock.advance(90)
        let repeated = try #require(try model.repeatLastSet(of: "pull-up", in: block))
        let set = try #require(model.tree.blocks[0].sets.first { $0.id == repeated })
        #expect(set.elements.map(\.exerciseId) == ["pull-up", "scap-pull"])
        #expect(set.elements[0].reps == 6)
        #expect(set.elements[0].assistance?.bandId == "red")
        #expect(set.elements[0].assistance?.id != band.id, "a new assistance row")
        #expect(set.elements[0].assistanceClass == "assisted")

        #expect(try model.repeatLastSet(of: "never-logged", in: block) == nil)
    }

    @Test func addingAnElementMakesACombo() throws {
        let clock = TestClock()
        let (_, model) = try logger(clock)
        let block = model.tree.blocks[0].id
        let id = try model.logSet(in: block, element: ElementDraft(exerciseId: "pull-up", measure: "reps", reps: 5, loadKg: 10))
        try model.addElement(to: id, ElementDraft(exerciseId: "dip", measure: "reps", reps: 8))
        let set = try #require(model.tree.blocks[0].sets.first)
        #expect(set.elements.count == 2)
        #expect(set.elements[0].assistanceClass == "loaded")
    }

    @Test func completingIsFinalAndQueued() throws {
        let clock = TestClock()
        let (db, model) = try logger(clock)
        try model.logSet(in: model.tree.blocks[0].id, element: ElementDraft(exerciseId: "pull-up", measure: "reps", reps: 5))
        try model.complete(perceivedFatigue: 7)
        #expect(model.session.status == "completed")
        #expect(model.session.perceivedFatigue == 7)
        #expect(model.rest == nil)
        #expect(try db.queuedOps().contains { $0.op == .complete })
    }

    @Test func deletingASetRemovesItFromTheTree() throws {
        let clock = TestClock()
        let (_, model) = try logger(clock)
        let id = try model.logSet(in: model.tree.blocks[0].id, element: ElementDraft(exerciseId: "pull-up", measure: "reps", reps: 5))
        try model.deleteSet(id)
        #expect(model.tree.blocks[0].sets.isEmpty)
    }
}
