import Foundation
import Testing
@testable import HefestoAPI
@testable import HefestoPlan
@testable import HefestoStore
@testable import HefestoSync

// The questions about pain around a planned session (ADR 0030): the regions
// the plan names, the answers as reports, and what is answered already.

/// 2026-09-30 07:00 UTC, the Wednesday after the fixture session.
let wednesdayMorning = Date(timeIntervalSince1970: 1_790_751_600)

@MainActor
func painPlan(_ db: AppDatabase, at now: Date = tuesday) throws -> (PlanModel, PlanSession) {
    let plannedId = UUIDv7.make()
    try keepWeek(db, weekJSON(plannedSessionId: plannedId, blocks: [], monitor: [("elbow_inner", "innen am Ellbogen")]))
    let model = planModel(db, at: now)
    try model.reload()
    let session = try #require(model.week?.session(id: plannedId))
    return (model, session)
}

/// A log session of the fixture's Tuesday, started from the planned session.
func loggedSession(_ db: AppDatabase, planned: String, completed: Bool = true) throws -> Session {
    let session = Session(
        startedAt: tuesday, timezone: zurich, localDate: "2026-09-29", status: completed ? "completed" : "draft",
        plannedSessionId: planned, completedAt: completed ? tuesday.addingTimeInterval(3600) : nil)
    try db.saveSession(session)
    return session
}

@Suite @MainActor struct PainQuestionTests {
    @Test func theSessionNamesTheRegionsToAskAbout() throws {
        let db = try AppDatabase.inMemory()
        let (model, session) = try painPlan(db)

        #expect(model.painRegions(for: session) == [PlanRegion(id: "elbow_inner", name: "innen am Ellbogen")])
    }

    @Test func aPlanKeptFromBeforeAsksNothing() throws {
        let db = try AppDatabase.inMemory()
        let plannedId = UUIDv7.make()
        try keepWeek(db, weekJSON(plannedSessionId: plannedId, blocks: []))
        let model = planModel(db)
        try model.reload()

        let session = try #require(model.week?.session(id: plannedId))
        #expect(model.painRegions(for: session).isEmpty)
    }

    @Test func theAnswersBecomeReportsOfTheirMoment() {
        let regions = [PlanRegion(id: "elbow_inner", name: "innen am Ellbogen"), PlanRegion(id: "wrist", name: "Handgelenk")]
        let at = tuesday
        let before = PainQuestion(moment: .beforeSession, sessionId: "s-1", regions: regions)
            .reports(["elbow_inner": PainAnswer(now: 2)], at: at)
        #expect(before.map(\.timepoint) == ["before_session"] && before.first?.nrs == 2)
        #expect(before.first?.sessionId == "s-1")

        let after = PainQuestion(moment: .afterSession, sessionId: "s-1", regions: regions)
            .reports(["elbow_inner": PainAnswer(now: 3, during: 4), "wrist": PainAnswer(during: 1)], at: at)
        #expect(after.map(\.timepoint) == ["during", "after", "during"])
        #expect(after.map(\.region) == ["elbow_inner", "elbow_inner", "wrist"])

        let morning = PainQuestion(moment: .nextMorning, sessionId: "s-1", regions: regions)
            .reports(["elbow_inner": PainAnswer(now: 1, lastedOver1h: true)], at: at)
        #expect(morning.map(\.timepoint) == ["next_morning"] && morning.first?.lastedOver1h == true)

        #expect(PainQuestion(moment: .afterSession, sessionId: "s-1", regions: regions).reports([:], at: at).isEmpty,
                "a question not answered sends nothing")
    }

    @Test func afterTheSessionItAsksWhatIsNotAnswered() throws {
        let db = try AppDatabase.inMemory()
        let (model, planned) = try painPlan(db)
        let session = try loggedSession(db, planned: planned.id)

        let question = try #require(model.afterQuestion(logSessionId: session.id))
        #expect(question.moment == .afterSession && question.regions.map(\.id) == ["elbow_inner"])

        try db.keepPainReports([KeptPainReport(region: "elbow_inner", timepoint: "after", nrs: 2, at: tuesday, sessionId: session.id)])
        #expect(model.afterQuestion(logSessionId: session.id) == nil, "answered once is enough")
    }

    @Test func theMorningAfterAsksAboutYesterday() throws {
        let db = try AppDatabase.inMemory()
        let (_, planned) = try painPlan(db)
        let session = try loggedSession(db, planned: planned.id)
        let morning = planModel(db, at: wednesdayMorning)

        let question = try #require(morning.morningQuestion())
        #expect(question.moment == .nextMorning && question.sessionId == session.id)
        #expect(question.regions.map(\.name) == ["innen am Ellbogen"])

        try db.keepPainReports([KeptPainReport(region: "elbow_inner", timepoint: "next_morning", nrs: 0, at: wednesdayMorning, sessionId: session.id)])
        #expect(morning.morningQuestion() == nil, "answered")
    }

    @Test func theMorningQuestionPassesWithItsDay() throws {
        let db = try AppDatabase.inMemory()
        let (_, planned) = try painPlan(db)
        _ = try loggedSession(db, planned: planned.id)

        #expect(planModel(db, at: tuesday).morningQuestion() == nil, "not before the morning after")
        #expect(planModel(db, at: wednesdayMorning.addingTimeInterval(86_400)).morningQuestion() == nil,
                "not asked again a day later")
    }

    @Test func aSessionNotDoneHasNoMorningQuestion() throws {
        let db = try AppDatabase.inMemory()
        let (_, planned) = try painPlan(db)
        _ = try loggedSession(db, planned: planned.id, completed: false)

        #expect(planModel(db, at: wednesdayMorning).morningQuestion() == nil)
    }
}
