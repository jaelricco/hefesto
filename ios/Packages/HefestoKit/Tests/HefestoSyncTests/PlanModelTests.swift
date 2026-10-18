import Foundation
import Testing
@testable import HefestoAPI
@testable import HefestoPlan
@testable import HefestoStore
@testable import HefestoSync

/// 2026-09-29 10:00 UTC, the Tuesday of the fixture plan's week.
let tuesday = Date(timeIntervalSince1970: 1_790_676_000)

func keepPlan(_ db: AppDatabase, id: String = UUIDv7.make(), plannedSessionId: String) throws {
    try db.savePlan(CachedPlan(
        weekStart: "2026-09-28", etag: nil, payload: Data(planJSON(id: id, plannedSessionId: plannedSessionId).utf8),
        fetchedAt: tuesday))
}

@MainActor
func planModel(_ db: AppDatabase, server: ScriptedServer? = nil, at now: Date = tuesday) -> PlanModel {
    PlanModel(
        db: db, sync: server.map { SyncEngine(client: $0.client, db: db) },
        timeZone: TimeZone(identifier: zurich) ?? .gmt, now: { now })
}

@Suite @MainActor struct PlanModelTests {
    @Test func aWeekHasItsSevenDaysAndItsItems() throws {
        let plannedSessionId = UUIDv7.make()
        let cached = CachedPlan(
            weekStart: "2026-09-28", etag: nil,
            payload: Data(planJSON(id: "plan-1", plannedSessionId: plannedSessionId).utf8), fetchedAt: tuesday)

        let week = try PlanWeek(decoding: cached)

        #expect(week.days.map(\.date) == [
            "2026-09-28", "2026-09-29", "2026-09-30", "2026-10-01", "2026-10-02", "2026-10-03", "2026-10-04",
        ])
        #expect(week.days[0].isRestDay)
        let session = try #require(week.days[1].session)
        #expect(session.id == plannedSessionId)
        #expect(session.status == .planned)
        #expect(session.kind == "full")
        let item = try #require(session.plannedItems.first)
        #expect(item.exerciseName == "Pull-up")
        #expect(item.sets == 3 && item.reps == 6 && item.reserve == 2 && item.restSeconds == 120)
        #expect(item.calibration)
        #expect(!item.isHold)
        #expect(week.item(id: item.id) == item)
        #expect(week.reasons.map(\.text) == ["Three sessions."])
        #expect(week.disclaimer == "Educational, not medical advice.")
    }

    @Test func theModelShowsTodaysSessionFromTheKeptWeek() throws {
        let db = try AppDatabase.inMemory()
        let plannedSessionId = UUIDv7.make()
        try keepPlan(db, plannedSessionId: plannedSessionId)

        let model = planModel(db)
        try model.reload()

        #expect(model.today == "2026-09-29")
        #expect(model.week?.weekStart == "2026-09-28")
        #expect(model.todaySession?.id == plannedSessionId)
        let wednesday = planModel(db, at: tuesday.addingTimeInterval(86_400))
        try wednesday.reload()
        #expect(wednesday.todaySession == nil, "the plan rests on Wednesday")
    }

    @Test func withoutAnOnboardingThereIsNoPlan() async throws {
        let db = try AppDatabase.inMemory()
        try keepPlan(db, plannedSessionId: UUIDv7.make())
        let server = ScriptedServer { _ in (409, problemJSON("onboarding-required", status: 409)) }
        let model = planModel(db, server: server)
        try model.reload()

        await model.refresh()

        #expect(model.status == .needsOnboarding)
        #expect(model.week == nil)
    }

    @Test func aStartSendsTheCheckInGivenAndRefreshesThePlan() async throws {
        let db = try AppDatabase.inMemory()
        let plannedSessionId = UUIDv7.make(), sessionId = UUIDv7.make()
        try keepPlan(db, plannedSessionId: plannedSessionId)
        let server = ScriptedServer { r in
            r.operation == "startPlannedSession"
                ? (201, draftJSON(sessionId, plannedSessionId: plannedSessionId, itemId: UUIDv7.make()))
                : (200, planJSON(id: UUIDv7.make(), plannedSessionId: plannedSessionId))
        }
        let model = planModel(db, server: server)
        try model.reload()
        let session = try #require(model.todaySession)

        let opened = try await model.start(session, sleepHours: 5.5, fatigue: nil)

        #expect(opened == sessionId)
        let start = try #require(server.requests("startPlannedSession").first)
        let checkIn = try #require(start.json["check_in"] as? [String: Any])
        #expect(checkIn["sleep_hours"] as? Double == 5.5)
        #expect(checkIn["fatigue"] == nil, "only what was answered is sent")
        #expect(server.requests("getTrainingPlan").count == 1, "the plan shows the start")
        #expect(try db.hasSession(id: sessionId))
    }

    @Test func aStartWithoutAnswersSendsNoCheckIn() async throws {
        let db = try AppDatabase.inMemory()
        let plannedSessionId = UUIDv7.make(), sessionId = UUIDv7.make()
        try keepPlan(db, plannedSessionId: plannedSessionId)
        let server = ScriptedServer { r in
            r.operation == "startPlannedSession"
                ? (201, draftJSON(sessionId, plannedSessionId: plannedSessionId, itemId: UUIDv7.make()))
                : (304, "")
        }
        let model = planModel(db, server: server)
        try model.reload()

        _ = try await model.start(try #require(model.todaySession), sleepHours: nil, fatigue: nil)

        let start = try #require(server.requests("startPlannedSession").first)
        #expect(start.json["check_in"] == nil)
    }

    @Test func aStoppedStartFetchesThePlanThatSaysWhy() async throws {
        let db = try AppDatabase.inMemory()
        let plannedSessionId = UUIDv7.make()
        try keepPlan(db, plannedSessionId: plannedSessionId)
        let server = ScriptedServer { r in
            r.operation == "startPlannedSession"
                ? (409, problemJSON("training-stopped", status: 409))
                : (200, planJSON(id: UUIDv7.make(), plannedSessionId: plannedSessionId))
        }
        let model = planModel(db, server: server)
        try model.reload()
        let session = try #require(model.todaySession)

        await #expect(throws: PlanError.trainingStopped) {
            _ = try await model.start(session, sleepHours: nil, fatigue: 9)
        }
        #expect(server.requests("getTrainingPlan").count == 1)
    }

    @Test func aDemoNeverReachesTheNetwork() async throws {
        let db = try AppDatabase.inMemory()
        try keepPlan(db, plannedSessionId: UUIDv7.make())
        let model = planModel(db)
        try model.reload()

        await model.refresh()
        #expect(model.status == .ready)
        let session = try #require(model.todaySession)
        await #expect(throws: PlanStartError.noServer) {
            _ = try await model.start(session, sleepHours: nil, fatigue: nil)
        }
    }

    @Test func aStartedSessionOpensItsLogSessionWhenKept() throws {
        let (db, session, _) = try seeded()
        let model = planModel(db)
        var planned = PlanSession(
            id: UUIDv7.make(), date: "2026-09-29", kind: "full", estMinutes: 40, status: .started,
            workoutSessionId: session.id, checkInApplied: false, blocks: [], reasons: [])
        #expect(model.logSession(for: planned) == session.id)
        planned.workoutSessionId = UUIDv7.make()
        #expect(model.logSession(for: planned) == nil, "not synced here yet")
    }
}
