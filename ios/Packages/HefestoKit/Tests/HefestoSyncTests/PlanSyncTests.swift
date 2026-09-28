import Foundation
import HTTPTypes
import Testing
@testable import HefestoAPI
@testable import HefestoStore
@testable import HefestoSync

@Suite struct PlanSyncTests {
    @Test func keepsThePlanAndAsksWithItsETag() async throws {
        let db = try AppDatabase.inMemory()
        let planId = UUIDv7.make()
        let server = ScriptedServer { r in
            if r.headers[.ifNoneMatch] == #""plan-1""# { return (304, "") }
            return (200, planJSON(id: planId, plannedSessionId: UUIDv7.make()))
        }
        server.responseHeaders[HTTPField.Name("ETag")!] = #""plan-1""#
        let engine = SyncEngine(client: server.client, db: db)

        #expect(try await engine.refreshPlan(on: "2026-09-30") == .updated)
        #expect(try await engine.refreshPlan(on: "2026-09-30") == .unchanged)

        let requests = server.requests("getTrainingPlan")
        #expect(requests.count == 2)
        #expect(requests[0].path.contains("week=2026-09-30"))
        #expect(requests[0].headers[.ifNoneMatch] == nil, "nothing kept yet")
        #expect(requests[1].headers[.ifNoneMatch] == #""plan-1""#)
        let kept = try #require(try db.plan(on: "2026-09-30"))
        #expect(kept.weekStart == "2026-09-28")
        #expect(kept.etag == #""plan-1""#)
        let plan = try kept.decoded()
        #expect(plan.id == planId)
        #expect(plan.sessions.first?.blocks.first?.items.first?.calibration == true)
        #expect(plan.reasons.first?.ruleId == "WEEK-01")
        #expect(plan.restDays == ["2026-09-28"])
    }

    @Test func withoutAnOnboardingNothingKeptApplies() async throws {
        let db = try AppDatabase.inMemory()
        try db.savePlan(CachedPlan(weekStart: "2026-09-28", etag: nil, payload: Data("{}".utf8), fetchedAt: Date()))
        let server = ScriptedServer { _ in (409, problemJSON("onboarding-required", status: 409)) }

        await #expect(throws: PlanError.onboardingRequired) {
            try await SyncEngine(client: server.client, db: db).refreshPlan(on: "2026-09-30")
        }
        #expect(try db.plan(on: "2026-09-30") == nil)
    }

    @Test func aStartKeepsTheDraftWithItsPlannedSets() async throws {
        let db = try AppDatabase.inMemory()
        let plannedSessionId = UUIDv7.make(), sessionId = UUIDv7.make(), itemId = UUIDv7.make()
        let server = ScriptedServer { _ in
            (201, draftJSON(sessionId, plannedSessionId: plannedSessionId, itemId: itemId))
        }
        let engine = SyncEngine(client: server.client, db: db)

        let started = try await engine.startPlannedSession(
            plannedSessionId, checkIn: .init(sleepHours: 5.5, fatigue: 8),
            timeZone: try #require(TimeZone(identifier: zurich)), id: sessionId)

        #expect(started == sessionId)
        let request = try #require(server.requests("startPlannedSession").first)
        #expect(request.path.contains(plannedSessionId))
        #expect(request.json["id"] as? String == sessionId)
        #expect(request.json["timezone"] as? String == zurich)
        let checkIn = try #require(request.json["check_in"] as? [String: Any])
        #expect(checkIn["sleep_hours"] as? Double == 5.5)
        #expect(checkIn["fatigue"] as? Int == 8)

        let tree = try #require(try fetch(db) { try AppDatabase.sessionTree($0, id: sessionId) })
        #expect(tree.session.plannedSessionId == plannedSessionId)
        let set = try #require(tree.blocks.first?.sets.first)
        #expect(set.entry.isPlanned)
        #expect(set.entry.plannedItemId == itemId)
        #expect(set.entry.sirS == 3, "a planned hold carries its target reserve")
        #expect(try db.queuedOps().isEmpty, "the server's rows are not sent back")
    }

    @Test func aStopRefusesTheStartAndKeepsNothing() async throws {
        let db = try AppDatabase.inMemory()
        let server = ScriptedServer { _ in (409, problemJSON("training-stopped", status: 409)) }
        let sessionId = UUIDv7.make()

        await #expect(throws: PlanError.trainingStopped) {
            try await SyncEngine(client: server.client, db: db).startPlannedSession(UUIDv7.make(), id: sessionId)
        }
        #expect(try db.hasSession(id: sessionId) == false)
    }

    @Test func aRepeatedStartLeavesTheKeptSessionAlone() async throws {
        let (db, session, block) = try seeded()
        _ = try logSet(db, session, block, reps: 7)
        let server = ScriptedServer { _ in
            (200, draftJSON(session.id, plannedSessionId: UUIDv7.make(), itemId: UUIDv7.make()))
        }

        _ = try await SyncEngine(client: server.client, db: db).startPlannedSession(UUIDv7.make())

        let tree = try #require(try fetch(db) { try AppDatabase.sessionTree($0, id: session.id) })
        #expect(tree.blocks.first?.sets.first?.elements.first?.reps == 7, "the unsent set stays")
        #expect(tree.blocks.first?.sets.first?.entry.isPlanned == false)
    }
}

// MARK: - server payloads

func problemJSON(_ slug: String, status: Int) -> String {
    #"{"type":"https://hefesto.fit/problems/\#(slug)","title":"Refused","status":\#(status)}"#
}

/// A plan of the week of 2026-09-28 with one session, one block and one
/// calibration item.
func planJSON(id: String, plannedSessionId: String) -> String {
    #"""
    {"id":"\#(id)","week_start":"2026-09-28","ruleset_version":"test",
     "input_hash":"sha256:0000000000000000000000000000000000000000000000000000000000000000",
     "stopped":false,"deload":null,
     "sessions":[{"id":"\#(plannedSessionId)","status":"planned","workout_session_id":null,
       "check_in_applied":false,"index":0,"date":"2026-09-29","kind":"full","est_minutes":45,
       "blocks":[{"role":"strength","minutes":null,"paired":false,
         "items":[{"id":"\#(UUIDv7.make())","exercise":"pull-up","exercise_name":"Pull-up","skill":null,
           "stimulus":"strength","kind":"working","sets":3,"reps":6,"hold_s":null,"load_kg":null,
           "assist":"none","reserve":2,"rest_s":120,"calibration":true,"offer":false,"stop_rules":[],
           "intensity":"moderate","role":"support","monitor":false,"reasons":[]}],
         "reasons":[]}],
       "reasons":[]}],
     "rest_days":["2026-09-28"],"exclusions":[],"hints":[],"realism":[],"monitor":[],"loads":[],
     "reasons":[{"rule_id":"WEEK-01","params":[],"sources":[],"evidence":"A","text":"Three sessions.",
       "args":{"n":"3"},"region":null}],
     "disclaimer":"Educational, not medical advice."}
    """#
}

/// A draft as the start answers it: one planned 20 s hold with its target
/// reserve.
func draftJSON(_ id: String, plannedSessionId: String, itemId: String) -> String {
    let blockId = UUIDv7.make()
    return #"""
    {"id":"\#(id)","started_at":"2026-09-29T17:00:00Z","ended_at":null,"timezone":"\#(zurich)",
     "local_date":"2026-09-29","title":"","notes":"","perceived_fatigue":null,"bodyweight_kg":null,
     "status":"draft","is_rest_day":false,"template_id":null,"planned_session_id":"\#(plannedSessionId)",
     "completed_at":null,"updated_at":"2026-09-29T17:00:00Z","server_updated_at":"2026-09-29T17:00:00Z",
     "blocks":[{"id":"\#(blockId)","order_index":0,"kind":"straight","rounds_planned":null,"rounds_done":null,
       "interval_s":null,"notes":"","updated_at":"2026-09-29T17:00:00Z",
       "sets":[{"id":"\#(UUIDv7.make())","block_id":"\#(blockId)","order_index":0,"round_index":null,
         "kind":"working","is_planned":true,"rest_after_planned_s":180,"rest_after_actual_s":null,"rpe":null,
         "rir":null,"sir_s":3,"planned_item_id":"\#(itemId)","completed_at":null,"notes":"",
         "updated_at":"2026-09-29T17:00:00Z","elements":[\#(holdElementJSON(seconds: 20))]}]}]}
    """#
}
