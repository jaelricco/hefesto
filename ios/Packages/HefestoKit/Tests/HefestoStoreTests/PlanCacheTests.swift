import Foundation
import GRDB
import Testing
@testable import HefestoStore

func cachedPlan(_ weekStart: String, etag: String? = nil) -> CachedPlan {
    CachedPlan(weekStart: weekStart, etag: etag, payload: Data("{}".utf8), fetchedAt: Date())
}

@Suite struct PlanCacheTests {
    @Test func aDayFindsThePlanOfItsWeek() throws {
        let db = try AppDatabase.inMemory()
        try db.savePlan(cachedPlan("2026-09-21"))
        try db.savePlan(cachedPlan("2026-09-28"))

        #expect(try db.plan(on: "2026-09-27")?.weekStart == "2026-09-21")
        #expect(try db.plan(on: "2026-09-28")?.weekStart == "2026-09-28")
        #expect(try db.plan(on: "2026-10-04")?.weekStart == "2026-09-28")
        #expect(try db.plan(on: "2026-10-05") == nil, "no plan of that week is kept")
        #expect(try db.plan(on: "2026-09-20") == nil)
    }

    @Test func aNewFetchReplacesItsWeekAndDropsOldWeeks() throws {
        let db = try AppDatabase.inMemory()
        try db.savePlan(cachedPlan("2026-08-24"))
        try db.savePlan(cachedPlan("2026-09-07"))
        try db.savePlan(cachedPlan("2026-09-28", etag: "a"))
        try db.savePlan(cachedPlan("2026-09-28", etag: "b"))

        let weeks = try db.reader.read { try CachedPlan.order(Column("weekStart")).fetchAll($0) }
        #expect(weeks.map(\.weekStart) == ["2026-09-07", "2026-09-28"], "more than four weeks older goes")
        #expect(weeks.last?.etag == "b")
    }

    @Test func clearingForgetsEveryPlan() throws {
        let db = try AppDatabase.inMemory()
        try db.savePlan(cachedPlan("2026-09-28"))
        try db.clearPlans()
        #expect(try db.plan(on: "2026-09-28") == nil)
    }

    @Test func theLinksToThePlanAreKept() throws {
        let db = try AppDatabase.inMemory()
        let session = Session(
            startedAt: Date(), timezone: "Europe/Zurich", localDate: "2026-09-29", plannedSessionId: "ps-1")
        try db.saveSession(session)
        let block = Block(sessionId: session.id, orderIndex: 0)
        try db.saveBlock(block)
        let entry = SetEntry(sessionId: session.id, blockId: block.id, orderIndex: 0, isPlanned: true,
                             plannedItemId: "item-1")
        try db.saveSet(SetWithElements(entry: entry, elements: [pullUps(entry, reps: 6)]))

        let tree = try #require(try db.reader.read { try AppDatabase.sessionTree($0, id: session.id) })
        #expect(tree.session.plannedSessionId == "ps-1")
        #expect(tree.blocks.first?.sets.first?.entry.plannedItemId == "item-1")
        #expect(try db.hasSession(id: session.id))
    }
}
