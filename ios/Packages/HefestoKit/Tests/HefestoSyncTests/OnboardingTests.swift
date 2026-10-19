import Foundation
import HTTPTypes
import Testing
@testable import HefestoAPI
@testable import HefestoPlan
@testable import HefestoStore
@testable import HefestoSync

@MainActor
func onboardingModel(server: ScriptedServer? = nil, db: AppDatabase? = nil) throws -> OnboardingModel {
    let db = try db ?? AppDatabase.inMemory()
    return OnboardingModel(
        sync: server.map { SyncEngine(client: $0.client, db: db) }, catalogue: try testCatalogue(),
        timeZone: TimeZone(identifier: zurich) ?? .gmt, now: { tuesday })
}

func testCatalogue() throws -> OnboardingCatalogue {
    try OnboardingCatalogue(json: Data(catalogueJSON.utf8))
}

/// An adult with consent, two goals, no pull-up yet, rings, a current
/// complaint at the inner elbow and a past one at the wrist.
func filledForm() -> OnboardingForm {
    var f = OnboardingForm()
    f.birthYear = 1995
    f.healthConsent = true
    f.disclaimerAck = true
    f.goals = [.init(skill: "planche", targetLevel: "tuck", targetDate: "2027-06-01"),
               .init(skill: "handstand", targetLevel: "free-10s")]
    f.sessionsPerWeek = 3
    f.sessionMinutes = 45
    f.equipment = ["rings"]
    f.bodyweightKg = 72.5
    f.trainingLevel = "recreational"
    f.trainingAge = "1_to_4_years"
    f.lastRegular = "current_or_lt_3_weeks"
    f.classes = ["push_up_class": "8_12", "pull_up_class": "0", "dead_hang_class": "10_30", "row_class": "6_10",
                 "dip_class": "4_7", "hollow_hold_class": "unknown"]
    f.stages = ["planche": .init(level: "lean", answerClass: "lt_4"), "handstand": .init(level: "none")]
    f.mobility = ["wrist": "yes", "shoulder": "partly"]
    f.dataConfidence = "estimated"
    f.exertion = false
    f.screening = ["SCR-01": false, "SCR-02": true, "SCR-03": false, "SCR-04": false, "SCR-05": false,
                   "SCR-06": false]
    f.bodyMap = ["elbow_inner": .init(current: true), "wrist_back_extension": .init(past12Months: true)]
    var complaint = OnboardingForm.Complaint()
    complaint.painDaily = 1
    complaint.painTraining = 4
    complaint.onset = "gradual"
    complaint.durationWeeks = 8
    complaint.suspectedSerious = "no"
    complaint.assessment = "no"
    complaint.restrictions = ["supination_loaded"]
    f.complaints = ["elbow_inner": complaint]
    f.redFlags = ["elbow_inner": ["RF-01": false, "RF-05": true, "RF-05-displaced": false]]
    return f
}

@Suite @MainActor struct OnboardingTests {
    @Test func theCatalogueReadsAsTheOnboardingAsks() throws {
        let c = try testCatalogue()

        #expect(c.goalSkills.map(\.slug) == ["pull-up", "planche", "handstand"], "a foundation without a milestone is no goal")
        #expect(c.skill("planche")?.levels.map(\.name) == ["Planche lean", "Tuck planche"])
        #expect(c.skill("planche")?.firstMilestone?.slug == "tuck")
        #expect(c.skill("planche")?.holds == true)
        #expect(c.skill("pull-up")?.stageClasses.last?.hi == nil, "an open class has no upper end")
        #expect(c.exertion.id == "RF-10")
        #expect(c.screening.map(\.id) == ["SCR-01", "SCR-02", "SCR-03", "SCR-04", "SCR-05", "SCR-06"])
        #expect(c.minorAgeYears == 18)
        let flag = try #require(c.region("elbow_inner")?.flags.first { $0.id == "RF-05" })
        #expect(flag.followup?.id == "RF-05-displaced")
    }

    @Test func onlyTheQuestionsThatApplyAreAsked() throws {
        let c = try testCatalogue()
        var f = OnboardingForm()
        f.classes = ["pull_up_class": "4_7", "dip_class": "4_7"]
        #expect(f.classQuestions(c).map(\.key) == ["push_up_class", "pull_up_class", "dip_class"])

        f.classes["pull_up_class"] = "0"
        #expect(f.classQuestions(c).map(\.key).contains("dead_hang_class"))
        #expect(!f.classQuestions(c).map(\.key).contains("row_class"), "rows need a low bar or rings")
        f.equipment = ["outdoor_park"]
        #expect(f.classQuestions(c).map(\.key).contains("row_class"), "a park has a low bar")

        f.classes["dip_class"] = "0"
        #expect(f.classQuestions(c).map(\.key).contains("support_hold_class"))

        f.goals = [.init(skill: "planche", targetLevel: "tuck"), .init(skill: "handstand", targetLevel: "free-10s")]
        #expect(f.classQuestions(c).map(\.key).contains("hollow_hold_class"))
        #expect(f.mobilityChecks == ["wrist", "shoulder"])
        #expect(f.stageSkills(c).map(\.slug) == ["planche", "handstand"])
        #expect(!f.asksPreBreak)
        f.lastRegular = "17_to_26_weeks"
        #expect(f.asksPreBreak)
    }

    @Test func aRegionAsksItsFlagsButNotTheExertionAgain() throws {
        let c = try testCatalogue()
        var f = filledForm()
        f.bodyMap["wrist_back_extension"] = .init(current: true)
        let wrist = try #require(c.region("wrist_back_extension"))

        #expect(f.pages(c) == [.frame, .goals, .availability, .equipment, .background, .performance, .health,
                               .region("elbow_inner"), .region("wrist_back_extension"), .finish])
        #expect(f.flags(for: wrist, c, currentYear: 2026).map(\.id) == ["RF-01"], "an adult, and RF-10 is asked once")
        f.birthYear = 2010
        #expect(f.flags(for: wrist, c, currentYear: 2026).map(\.id) == ["RF-01", "RF-12"], "a minor")
    }

    @Test func aPageIsCompleteWhenEveryQuestionThatAppliesIsAnswered() throws {
        let c = try testCatalogue()
        var f = filledForm()
        for page in f.pages(c) {
            #expect(f.isComplete(page, c, currentYear: 2026), "\(page)")
        }
        f.redFlags["elbow_inner"]?["RF-05-displaced"] = nil
        #expect(!f.isComplete(.region("elbow_inner"), c, currentYear: 2026), "a follow-up after a yes")
        f.stages["planche"]?.answerClass = nil
        #expect(!f.isComplete(.performance, c, currentYear: 2026), "how long, for a named level")
        f.screening["SCR-04"] = nil
        #expect(!f.isComplete(.health, c, currentYear: 2026))
        f.healthConsent = false
        #expect(f.isComplete(.health, c, currentYear: 2026), "no screening without consent")
        #expect(f.request(c, currentYear: 2026) == nil, "the elbow is still open")
    }

    @Test func theRequestCarriesWhatAppliesAndNothingElse() async throws {
        let db = try AppDatabase.inMemory()
        let server = ScriptedServer { _ in (200, onboardingResultJSON(status: "complete", plan: true)) }
        let model = try onboardingModel(server: server, db: db)
        model.form = filledForm()
        // Answers that no longer apply stay behind.
        model.form.classes["support_hold_class"] = "gt_30"
        model.form.preBreak = ["planche": "tuck"]

        let outcome = try await model.submit()

        #expect(outcome.status == .complete)
        let body = try #require(server.requests("submitOnboarding").first).json
        #expect((body["goals"] as? [[String: Any]])?.map { $0["priority"] as? Int } == [1, 2])
        #expect((body["goals"] as? [[String: Any]])?.first?["target_date"] as? String == "2027-06-01")
        let classes = try #require(body["classes"] as? [String: String])
        #expect(classes == ["push_up_class": "8_12", "pull_up_class": "0", "dead_hang_class": "10_30",
                            "row_class": "6_10", "dip_class": "4_7", "hollow_hold_class": "unknown"])
        #expect(body["pre_break_level"] == nil, "no break")
        let stages = try #require(body["skill_stages"] as? [String: [String: Any]])
        #expect(stages["planche"]?["level"] as? String == "lean" && stages["planche"]?["class"] as? String == "lt_4")
        #expect(stages["handstand"]?["class"] == nil, "no class without a level")
        #expect(body["mobility_checks"] as? [String: String] == ["wrist": "yes", "shoulder": "partly"])
        #expect(body["screening"] as? [Bool] == [false, true, false, false, false, false])
        let map = try #require(body["body_map"] as? [String: [String: Bool]])
        #expect(map["elbow_inner"] == ["current": true, "past_12_months": false])
        #expect(map["wrist_back_extension"] == ["current": false, "past_12_months": true])
        let flags = try #require(body["red_flags"] as? [String: [String: Bool]])
        #expect(flags["elbow_inner"] == ["RF-01": false, "RF-05": true, "RF-05-displaced": false, "RF-10": false])
        let complaint = try #require((body["complaints"] as? [String: [String: Any]])?["elbow_inner"])
        #expect(complaint["duration_weeks"] as? Double == 8)
        #expect(complaint["restrictions"] as? [String] == ["supination_loaded"])
        #expect(try db.plan(on: "2026-09-29") != nil, "the first plan is kept")
    }

    @Test func withoutConsentTheScreeningAndPastInjuriesStayUnasked() async throws {
        let server = ScriptedServer { _ in (200, onboardingResultJSON(status: "complete", plan: true)) }
        let model = try onboardingModel(server: server)
        model.form = filledForm()
        model.form.healthConsent = false

        try await model.submit()

        let body = try #require(server.requests("submitOnboarding").first).json
        #expect(body["screening"] as? [Bool] == [])
        let map = try #require(body["body_map"] as? [String: [String: Bool]])
        #expect(map.keys.sorted() == ["elbow_inner"], "a past injury is not asked without consent")
        #expect(body["red_flags"] != nil, "the safety questions are asked anyway")
    }

    @Test func questionsComeBackAndTheirAnswersGoWithTheNextRequest() async throws {
        let server = ScriptedServer { r in
            r.json["clarifications"] == nil
                ? (200, onboardingResultJSON(status: "needs_answers", plan: false))
                : (200, onboardingResultJSON(status: "complete", plan: true))
        }
        let model = try onboardingModel(server: server)
        model.form = filledForm()

        let first = try await model.submit()
        #expect(first.status == .needsAnswers)
        #expect(first.questions.map(\.id) == ["R-2/planche"])
        model.form.clarifications["R-2/planche"] = false
        let second = try await model.submit()

        #expect(second.status == .complete)
        let requests = server.requests("submitOnboarding")
        #expect(requests[1].json["clarifications"] as? [String: Bool] == ["R-2/planche": false])
        let key = HTTPField.Name("Idempotency-Key")!
        #expect(requests[0].headers[key] != requests[1].headers[key], "new answers, a new key")
    }

    @Test func aRetryOfTheSameAnswersKeepsItsKey() async throws {
        let calls = Counter()
        let server = ScriptedServer { _ in
            if calls.next() == 1 { throw LostConnection() }
            return (200, onboardingResultJSON(status: "complete", plan: true))
        }
        let model = try onboardingModel(server: server)
        model.form = filledForm()

        await #expect(throws: (any Error).self) { try await model.submit() }
        try await model.submit()

        let requests = server.requests("submitOnboarding")
        let key = HTTPField.Name("Idempotency-Key")!
        #expect(requests.count == 2)
        #expect(requests[0].headers[key] != nil && requests[0].headers[key] == requests[1].headers[key])
    }

    @Test func signingOutForgetsTheAnswersAndTheirKey() async throws {
        let server = ScriptedServer { _ in (200, onboardingResultJSON(status: "complete", plan: true)) }
        let model = try onboardingModel(server: server)
        model.form = filledForm()
        try await model.submit()

        model.forget()

        #expect(model.form == OnboardingForm() && model.outcome == nil, "the next account starts empty")
        #expect(model.catalogue != nil, "the catalogue is content, not the athlete's")
        model.form = filledForm()
        try await model.submit()
        let requests = server.requests("submitOnboarding")
        let key = HTTPField.Name("Idempotency-Key")!
        #expect(requests.count == 2 && requests[0].headers[key] != requests[1].headers[key])
    }

    @Test func theServerSaysWhyItRefused() async throws {
        let model = try onboardingModel(server: ScriptedServer { _ in (409, problemJSON("already-onboarded", status: 409)) })
        model.form = filledForm()
        await #expect(throws: OnboardingError.alreadyOnboarded) { try await model.submit() }

        let invalid = try onboardingModel(server: ScriptedServer { _ in
            (422, #"{"type":"https://hefesto.fit/problems/validation","title":"Invalid","status":422,"errors":{"/bodyweight_kg":"must be 30–250"}}"#)
        })
        invalid.form = filledForm()
        await #expect(throws: OnboardingError.invalid(["/bodyweight_kg": "must be 30–250"])) { try await invalid.submit() }
    }

    @Test func anIncompleteFormOrADemoSendsNothing() async throws {
        let server = ScriptedServer { _ in (200, onboardingResultJSON(status: "complete", plan: true)) }
        let model = try onboardingModel(server: server)
        await #expect(throws: OnboardingSubmitError.incomplete) { try await model.submit() }
        #expect(server.requests.isEmpty)

        let demo = try onboardingModel()
        demo.form = filledForm()
        await #expect(throws: OnboardingSubmitError.noServer) { try await demo.submit() }
        try await demo.load()
        #expect(demo.catalogue != nil, "a demo brings its own catalogue")
    }
}

// MARK: - server payloads

func onboardingResultJSON(status: String, plan: Bool) -> String {
    let questions = status == "needs_answers"
        ? #"[{"id":"R-2/planche","rule":"R-2","skill":"planche","text":"Can you hold the planche lean for 10 s?"}]"#
        : "[]"
    let firstPlan = plan ? #","plan":\#(planJSON(id: UUIDv7.make(), plannedSessionId: UUIDv7.make()))"# : ""
    return #"""
    {"status":"\#(status)","questions":\#(questions),"realism":[],"hints":[],"reasons":[],
     "disclaimer":"Educational, not medical advice."\#(firstPlan)}
    """#
}

private func flagJSON(_ id: String, minorOnly: Bool = false, followup: String = "null") -> String {
    #"{"id":"\#(id)","urgency":"D","action":"lock","question":"Question \#(id)?","advice":"Advice \#(id).","minor_only":\#(minorOnly),"followup":\#(followup)}"#
}

private let displaced = #"{"id":"RF-05-displaced","urgency":"N","action":"stop","question":"Displaced?","advice":"Now."}"#

private func exerciseJSON(_ slug: String, _ name: String, holds: Bool) -> String {
    #"{"slug":"\#(slug)","name":"\#(name)","measure":"\#(holds ? "hold_seconds" : "reps")","requires":[],"skill":null,"draft":true}"#
}

private func classJSON(_ key: String, _ lo: Int, _ hi: Int?) -> String {
    #"{"key":"\#(key)","lo":\#(lo),"hi":\#(hi.map(String.init) ?? "null"),"open":\#(hi == nil)}"#
}

private func questionJSON(_ key: String, _ exercise: String, _ classes: [String]) -> String {
    #"{"key":"\#(key)","exercise":"\#(exercise)","classes":[\#(classes.joined(separator: ","))]}"#
}

/// A catalogue with the shape of the knowledge base's, cut down to what the
/// tests ask about.
let catalogueJSON: String = {
    let reps = [classJSON("0", 0, 0), classJSON("1_3", 1, 3), classJSON("gt_30", 30, nil)]
    let holds = [classJSON("lt_10", 0, 10), classJSON("gt_30", 30, nil)]
    let skills = [
        #"{"slug":"hang-foundation","name":"Hang","foundation":true,"levels":[{"slug":"dead-hang","exercise":"dead-hang","threshold":30,"milestone":false}],"stage_classes":[]}"#,
        #"{"slug":"pull-up","name":"Pull-up","foundation":true,"levels":[{"slug":"strict-5","exercise":"pull-up","threshold":5,"milestone":true}],"stage_classes":[\#(classJSON("1", 1, 1)),\#(classJSON("gt_5", 5, nil))]}"#,
        #"{"slug":"planche","name":"Planche","foundation":false,"levels":[{"slug":"lean","exercise":"planche-lean","threshold":20,"milestone":false},{"slug":"tuck","exercise":"planche-tuck","threshold":10,"milestone":true}],"stage_classes":[\#(classJSON("lt_4", 0, 4)),\#(classJSON("ge_20", 20, nil))]}"#,
        #"{"slug":"handstand","name":"Handstand","foundation":false,"levels":[{"slug":"wall","exercise":"wall-handstand-hold","threshold":30,"milestone":false},{"slug":"free-10s","exercise":"handstand-freestanding","threshold":10,"milestone":true}],"stage_classes":[\#(classJSON("lt_4", 0, 4))]}"#,
    ]
    let exercises = [
        exerciseJSON("dead-hang", "Dead hang", holds: true), exerciseJSON("pull-up", "Pull-up", holds: false),
        exerciseJSON("planche-lean", "Planche lean", holds: true), exerciseJSON("planche-tuck", "Tuck planche", holds: true),
        exerciseJSON("wall-handstand-hold", "Wall handstand", holds: true),
        exerciseJSON("handstand-freestanding", "Handstand", holds: true), exerciseJSON("push-up", "Push-up", holds: false),
        exerciseJSON("dip-pb", "Dip", holds: false), exerciseJSON("support-hold-pb", "Support hold", holds: true),
        exerciseJSON("row-incline", "Incline row", holds: false), exerciseJSON("hollow-hold", "Hollow hold", holds: true),
    ]
    let regions = [
        #"{"id":"elbow_inner","name":"inside the elbow","red_flags":[\#(flagJSON("RF-01")),\#(flagJSON("RF-05", followup: displaced)),\#(flagJSON("RF-10"))]}"#,
        #"{"id":"wrist_back_extension","name":"back of the wrist","red_flags":[\#(flagJSON("RF-01")),\#(flagJSON("RF-10")),\#(flagJSON("RF-12", minorOnly: true))]}"#,
    ]
    let questions = [
        questionJSON("push_up_class", "push-up", reps), questionJSON("pull_up_class", "pull-up", reps),
        questionJSON("dead_hang_class", "dead-hang", holds), questionJSON("row_class", "row-incline", reps),
        questionJSON("dip_class", "dip-pb", reps), questionJSON("support_hold_class", "support-hold-pb", holds),
        questionJSON("hollow_hold_class", "hollow-hold", holds),
    ]
    let screening = (1...6).map { #"{"id":"SCR-0\#($0)","question":"Screening \#($0)?"}"# }
    return #"""
    {"ruleset_version":"test","disclaimer":"Educational, not medical advice.",
     "skills":[\#(skills.joined(separator: ","))],"exercises":[\#(exercises.joined(separator: ","))],
     "regions":[\#(regions.joined(separator: ","))],"class_questions":[\#(questions.joined(separator: ","))],
     "exertion":\#(flagJSON("RF-10")),"screening":[\#(screening.joined(separator: ","))],"minor_age_years":18}
    """#
}()
