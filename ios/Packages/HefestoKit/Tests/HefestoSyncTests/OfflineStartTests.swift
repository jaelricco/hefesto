import Foundation
import GRDB
import Testing
@testable import HefestoAPI
@testable import HefestoPlan
@testable import HefestoStore
@testable import HefestoSync

// Starting a planned session without the server (spec §10.5): the draft is
// written here as the server writes it, and the sync carries its links.

/// A plan item as the server sends it.
func itemJSON(
    id: String = UUIDv7.make(), exercise: String, stimulus: String = "strength", kind: String = "working",
    sets: Int, reps: Int? = nil, holdS: Int? = nil, loadKg: Double? = nil, reserve: Int = 2, restS: Int = 90,
    offer: Bool = false
) -> String {
    let reps = reps.map { String($0) } ?? "null", hold = holdS.map { String($0) } ?? "null"
    let load = loadKg.map { String($0) } ?? "null"
    return #"""
    {"id":"\#(id)","exercise":"\#(exercise)","exercise_name":"\#(exercise)","skill":null,
     "stimulus":"\#(stimulus)","kind":"\#(kind)","sets":\#(sets),"reps":\#(reps),"hold_s":\#(hold),
     "load_kg":\#(load),"assist":"none","reserve":\#(reserve),"rest_s":\#(restS),"calibration":false,
     "offer":\#(offer),"stop_rules":[],"intensity":"moderate","role":"support","monitor":false,"reasons":[]}
    """#
}

func blockJSON(role: String = "strength", paired: Bool = false, items: [String]) -> String {
    #"{"role":"\#(role)","minutes":null,"paired":\#(paired),"items":[\#(items.joined(separator: ","))],"reasons":[]}"#
}

/// A plan of the week of 2026-09-28 with one session on Tuesday.
func weekJSON(
    plannedSessionId: String, blocks: [String], stopped: Bool = false, status: String = "planned",
    workoutSessionId: String? = nil
) -> String {
    let workout = workoutSessionId.map { #""\#($0)""# } ?? "null"
    return #"""
    {"id":"\#(UUIDv7.make())","week_start":"2026-09-28","ruleset_version":"test",
     "input_hash":"sha256:0000000000000000000000000000000000000000000000000000000000000000",
     "stopped":\#(stopped),"deload":null,
     "sessions":[{"id":"\#(plannedSessionId)","status":"\#(status)","workout_session_id":\#(workout),
       "check_in_applied":false,"index":0,"date":"2026-09-29","kind":"full","est_minutes":45,
       "blocks":[\#(blocks.joined(separator: ","))],"reasons":[]}],
     "rest_days":[],"exclusions":[],"hints":[],"realism":[],"monitor":[],"loads":[],"reasons":[],
     "disclaimer":"Educational, not medical advice."}
    """#
}

func keepWeek(_ db: AppDatabase, _ json: String) throws {
    try db.savePlan(CachedPlan(weekStart: "2026-09-28", etag: nil, payload: Data(json.utf8), fetchedAt: tuesday))
}

func keepCatalogue(_ db: AppDatabase, holds: [String] = []) throws -> [String: String] {
    let slugs = ["pull-up", "push-up", "tuck-planche"] + holds
    let exercises = slugs.map {
        Exercise(id: UUIDv7.make(), slug: $0, name: $0, family: "test",
                 defaultMeasure: holds.contains($0) || $0 == "tuck-planche" ? "hold_seconds" : "reps", isBodyweight: true)
    }
    try db.replaceExercises(exercises, contentVersion: "test")
    return Dictionary(uniqueKeysWithValues: exercises.map { ($0.slug, $0.id) })
}

@Suite struct PlanDraftTests {
    @Test func buildsTheDraftAsTheServerDoes() throws {
        let db = try AppDatabase.inMemory()
        let ids = try keepCatalogue(db)
        let pull = UUIDv7.make(), hold = UUIDv7.make(), warm = UUIDv7.make(), plain = UUIDv7.make()
        let plannedSessionId = UUIDv7.make()
        try keepWeek(db, weekJSON(plannedSessionId: plannedSessionId, blocks: [
            // An antagonist pair: three rounds, the hold in the first two.
            blockJSON(role: "skill_volume", paired: true, items: [
                itemJSON(id: pull, exercise: "pull-up", sets: 3, reps: 8, reserve: 12, restS: 60),
                itemJSON(id: hold, exercise: "tuck-planche", stimulus: "eccentric", sets: 2, holdS: 20, reserve: 70),
            ]),
            // Offers only: nothing to write.
            blockJSON(role: "skill_max", items: [itemJSON(exercise: "pull-up", sets: 1, reps: 1, offer: true)]),
            blockJSON(items: [
                itemJSON(id: warm, exercise: "push-up", kind: "warmup", sets: 1, reps: 5, loadKg: 10, reserve: -1),
                itemJSON(exercise: "pull-up", sets: 2, reps: 3, offer: true),
                itemJSON(id: plain, exercise: "tuck-planche", sets: 2, reserve: 3, restS: 180),
            ]),
        ]))
        let planned = try #require(try db.plan(on: "2026-09-29")?.decoded().sessions.first)
        let session = Session(startedAt: tuesday, timezone: zurich, localDate: "2026-09-29", plannedSessionId: plannedSessionId)
        let exercises = try fetch(db) { try Exercise.fetchAll($0) }
        let tree = try PlanDraft.tree(
            planned, session: session, exercises: Dictionary(uniqueKeysWithValues: exercises.map { ($0.slug, $0) }))

        #expect(tree.blocks.count == 2, "a block of offers is not written")
        let pair = tree.blocks[0]
        #expect(pair.block.kind == "superset" && pair.block.roundsPlanned == 3 && pair.block.orderIndex == 0)
        #expect(pair.sets.map(\.entry.plannedItemId) == [pull, hold, pull, hold, pull])
        #expect(pair.sets.map(\.entry.roundIndex) == [0, 0, 1, 1, 2])
        #expect(pair.sets.map(\.entry.orderIndex) == [0, 1, 2, 3, 4])
        let rep = pair.sets[0], held = pair.sets[1]
        #expect(rep.entry.rir == 10 && rep.entry.sirS == nil, "the reserve is capped as the log records it")
        #expect(rep.entry.restAfterPlannedS == 60 && rep.entry.isPlanned && rep.entry.kind == "working")
        #expect(rep.elements.count == 1 && rep.elements[0].exerciseId == ids["pull-up"] && rep.elements[0].reps == 8)
        #expect(held.entry.sirS == 60 && held.entry.rir == nil)
        #expect(held.elements[0].measure == "hold_seconds" && held.elements[0].holdSeconds == 20)
        #expect(held.elements[0].isEccentricOnly)

        let straight = tree.blocks[1]
        #expect(straight.block.kind == "straight" && straight.block.roundsPlanned == nil && straight.block.orderIndex == 1)
        #expect(straight.sets.map(\.entry.plannedItemId) == [warm, plain, plain])
        #expect(straight.sets.allSatisfy { $0.entry.roundIndex == nil })
        let warmup = straight.sets[0]
        #expect(warmup.entry.kind == "warmup" && warmup.entry.rir == 0)
        #expect(warmup.elements[0].loadKg == 10 && warmup.elements[0].assistanceClass == "loaded")
        let byCatalogue = straight.sets[1]
        #expect(byCatalogue.elements[0].measure == "hold_seconds", "without a target the catalogue names the measure")
        #expect(byCatalogue.elements[0].holdSeconds == nil && byCatalogue.entry.sirS == 3)
        #expect(Set(tree.blocks.flatMap { $0.sets.flatMap { [$0.entry.id] + $0.elements.map(\.id) } }).count == 16)
    }

    @Test func anExerciseMissingHereStopsTheStart() throws {
        let db = try AppDatabase.inMemory()
        _ = try keepCatalogue(db)
        let plannedSessionId = UUIDv7.make()
        try keepWeek(db, weekJSON(plannedSessionId: plannedSessionId, blocks: [
            blockJSON(items: [itemJSON(exercise: "front-lever", sets: 3, holdS: 5)]),
        ]))

        #expect(throws: OfflineStartError.unknownExercise("front-lever")) {
            try db.startPlannedSessionOffline(plannedSessionId, on: "2026-09-29", at: tuesday)
        }
        #expect(try fetch(db) { try Session.fetchCount($0) } == 0)
    }

    @Test func aStoppedPlanStartsNothing() throws {
        let db = try AppDatabase.inMemory()
        _ = try keepCatalogue(db)
        let plannedSessionId = UUIDv7.make()
        try keepWeek(db, weekJSON(plannedSessionId: plannedSessionId, blocks: [
            blockJSON(items: [itemJSON(exercise: "pull-up", sets: 3, reps: 5)]),
        ], stopped: true))

        #expect(throws: PlanError.trainingStopped) {
            try db.startPlannedSessionOffline(plannedSessionId, on: "2026-09-29", at: tuesday)
        }
    }

    @Test func aSessionStartedElsewhereNeedsTheServer() throws {
        let db = try AppDatabase.inMemory()
        _ = try keepCatalogue(db)
        let plannedSessionId = UUIDv7.make()
        try keepWeek(db, weekJSON(plannedSessionId: plannedSessionId, blocks: [
            blockJSON(items: [itemJSON(exercise: "pull-up", sets: 3, reps: 5)]),
        ], status: "started", workoutSessionId: UUIDv7.make()))

        #expect(throws: OfflineStartError.startedElsewhere) {
            try db.startPlannedSessionOffline(plannedSessionId, on: "2026-09-29", at: tuesday)
        }
    }
}

@Suite struct OfflineStartTests {
    @Test func theSyncCarriesTheSessionWithItsLinks() async throws {
        let db = try AppDatabase.inMemory()
        _ = try keepCatalogue(db)
        let plannedSessionId = UUIDv7.make(), item = UUIDv7.make()
        try keepWeek(db, weekJSON(plannedSessionId: plannedSessionId, blocks: [
            blockJSON(items: [itemJSON(id: item, exercise: "pull-up", sets: 2, reps: 5)]),
        ]))

        let id = try db.startPlannedSessionOffline(
            plannedSessionId, on: "2026-09-29", timeZone: TimeZone(identifier: zurich) ?? .gmt, at: tuesday)
        let again = try db.startPlannedSessionOffline(plannedSessionId, on: "2026-09-29", at: tuesday)
        #expect(again == id, "a session started here before is that session")

        let server = ScriptedServer(applyingEverything)
        _ = try await SyncEngine(client: server.client, db: db).sync()

        let ops = try #require(server.requests("pushChanges").first?.json["ops"] as? [[String: Any]])
        #expect(ops.map { $0["entity"] as? String } == ["session", "block", "set", "set"])
        let session = try #require(ops[0]["data"] as? [String: Any])
        #expect(ops[0]["id"] as? String == id)
        #expect(session["planned_session_id"] as? String == plannedSessionId)
        #expect(session["timezone"] as? String == zurich)
        for op in ops[2...] {
            let set = try #require(op["data"] as? [String: Any])
            #expect(set["planned_item_id"] as? String == item)
            #expect(set["is_planned"] as? Bool == true)
            #expect(set["rir"] as? Int == 2)
            #expect(set["block_id"] as? String == ops[1]["id"] as? String)
        }
    }
}

@Suite @MainActor struct OfflinePlanModelTests {
    @Test func anOfflineStartShowsAsStartedAndThenCompleted() throws {
        let db = try AppDatabase.inMemory()
        _ = try keepCatalogue(db)
        let plannedSessionId = UUIDv7.make()
        try keepPlan(db, plannedSessionId: plannedSessionId)
        let model = planModel(db)
        try model.reload()
        let session = try #require(model.todaySession)

        let id = try model.startOffline(session)

        let started = try #require(model.todaySession)
        #expect(started.status == .started && started.workoutSessionId == id)
        #expect(model.logSession(for: started) == id)
        #expect(try model.startOffline(started) == id)
        let tree = try #require(try fetch(db) { try AppDatabase.sessionTree($0, id: id) })
        #expect(tree.session.plannedSessionId == plannedSessionId && tree.session.localDate == "2026-09-29")
        #expect(tree.blocks.flatMap(\.sets).count == 3)

        try db.completeSession(id: id, at: tuesday)
        try model.reload()
        #expect(model.todaySession?.status == .completed, "a completion here shows before the plan knows it")
    }

    @Test func aStartWithoutAConnectionSaysSo() async throws {
        let db = try AppDatabase.inMemory()
        try keepPlan(db, plannedSessionId: UUIDv7.make())
        let model = planModel(db, server: ScriptedServer { _ in throw LostConnection() })
        try model.reload()
        let session = try #require(model.todaySession)

        await #expect(throws: PlanStartError.offline) {
            _ = try await model.start(session, sleepHours: nil, fatigue: nil)
        }
    }

    @Test func thePlanKeepsTheLogSessionItNames() throws {
        let other = UUIDv7.make(), plannedSessionId = UUIDv7.make()
        let cached = CachedPlan(
            weekStart: "2026-09-28", etag: nil,
            payload: Data(weekJSON(plannedSessionId: plannedSessionId, blocks: [], status: "started",
                                   workoutSessionId: other).utf8),
            fetchedAt: tuesday)

        let week = try PlanWeek(decoding: cached).applying([
            PlannedStart(sessionId: UUIDv7.make(), plannedSessionId: plannedSessionId, completed: false),
        ])

        #expect(week.sessions[0].workoutSessionId == other)
        #expect(week.days[1].session?.workoutSessionId == other)
    }
}
