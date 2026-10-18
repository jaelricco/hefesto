import Foundation
import HefestoLogger
import HefestoPlan
import HefestoStore

/// A screen a demo run opens on, for design review and the screenshot
/// workflow (.github/workflows/ios-screenshots.yml).
enum DemoScreen: String, CaseIterable {
    // `composer` opens the set composer empty; `reserve` opens it on a hold,
    // so it asks what the athlete had left.
    case today, logger, composer, reserve, finish, map, peek, detail, history, session, stats, celebration
    case signIn = "signin"
    // The plan (ADR 0022): today's session in full, the check-in before its
    // start, the started session in the logger with its targets, and the
    // composer on the next target, a calibration set.
    case plansession, checkin, planned, target
    // The onboarding (ADR 0023): the goals, where the athlete stands, and a
    // current complaint with its warning signs.
    case onboarding, standing, complaint
    // A band-assisted pull-up in the composer (ADR 0024).
    case band

    var tab: AppTab {
        switch self {
        case .map, .peek, .detail: .map
        case .history, .session, .stats: .history
        default: .today
        }
    }

    var opensLogger: Bool { [.logger, .composer, .reserve, .finish, .band].contains(self) }
    var opensPlanned: Bool { [.planned, .target].contains(self) }

    /// The onboarding page the screen opens on, if it opens the onboarding.
    var onboardingPage: OnboardingPage? {
        switch self {
        case .onboarding: .goals
        case .standing: .performance
        case .complaint: .region("elbow_inner")
        default: nil
        }
    }
}

/// What a demo run seeded, so screens can open on it.
struct DemoRun {
    let screen: DemoScreen
    /// Today's draft, with a set logged 48 s ago and the rest running.
    let logger: LoggerModel?
    let draftSessionId: String
    /// Today's planned session as started, with its targets open.
    let plannedSessionId: String
    let pastSessionId: String
    let skillId: String
    /// A hold: the stats screen shows it, and the reserve screen chooses it.
    let exerciseId: String
    /// A band of the athlete's own, and the exercise the band screen assists.
    let bandId: String
    let bandExerciseId: String
    let celebration: Celebration?
    /// The planner's catalogue and the answers so far, for the onboarding.
    let catalogue: OnboardingCatalogue?
    let onboardingForm: OnboardingForm?

    var opensLogger: Bool { screen.opensLogger }
    var onboardingPage: OnboardingPage? { screen.onboardingPage }

    /// The log session the screen opens in the logger, if it opens one.
    var opensSessionId: String? {
        if screen.opensLogger { return draftSessionId }
        if screen.opensPlanned { return plannedSessionId }
        return nil
    }
}

#if DEBUG
// Start a Debug build with `-HefestoDemo YES` (and optionally
// `-HefestoDemoScreen map`) to open it on this data: signed in, offline,
// in memory, nothing sent anywhere. The "Hefesto Demo" scheme does the first.
//
// This is fixture data for looking at the design, like a test's. The three
// real skills and their criteria follow content/skills. Push-up, dip, planche
// and muscle-up, and the prerequisites between them, follow the researched
// planner content (content/training/skills.yaml on the planner branch):
// names and edges only, no criteria. None of it is content, and none of it
// reaches a Release build.
extension DemoRun {
    static var requestedScreen: DemoScreen? {
        let defaults = UserDefaults.standard
        guard defaults.bool(forKey: "HefestoDemo") else { return nil }
        return DemoScreen(rawValue: defaults.string(forKey: "HefestoDemoScreen") ?? "") ?? .today
    }

    @MainActor
    static func seed(_ db: AppDatabase, screen: DemoScreen, now: Date = Date()) throws -> DemoRun {
        let zone = TimeZone.current
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = zone

        func day(_ daysAgo: Int, _ hour: Int, _ minute: Int) -> Date {
            let start = calendar.startOfDay(for: now)
            let date = calendar.date(byAdding: .day, value: -daysAgo, to: start) ?? start
            return calendar.date(bySettingHour: hour, minute: minute, second: 0, of: date) ?? date
        }
        func localDate(_ date: Date) -> String {
            let c = calendar.dateComponents([.year, .month, .day], from: date)
            return String(format: "%04d-%02d-%02d", c.year ?? 0, c.month ?? 0, c.day ?? 0)
        }

        // The catalogue: the exercises in content/exercises.
        let pullUp = Exercise(id: "demo-ex-pull-up", slug: "pull-up", name: "Pull-up", family: "pull",
                              defaultMeasure: "reps", isBodyweight: true)
        let scapular = Exercise(id: "demo-ex-scapular", slug: "scapular-pull-up", name: "Scapular Pull-up",
                                family: "pull", defaultMeasure: "reps", isBodyweight: true)
        let tuck = Exercise(id: "demo-ex-tuck-fl", slug: "front-lever-tuck", name: "Tuck Front Lever",
                            family: "pull", defaultMeasure: "hold_seconds", isBodyweight: true)
        let advTuck = Exercise(id: "demo-ex-adv-tuck-fl", slug: "front-lever-advanced-tuck",
                               name: "Advanced Tuck Front Lever", family: "pull", defaultMeasure: "hold_seconds",
                               isBodyweight: true)
        let bandTuck = Exercise(id: "demo-ex-band-tuck-fl", slug: "band-assisted-tuck-fl",
                                name: "Band-assisted Tuck Front Lever", family: "pull",
                                defaultMeasure: "hold_seconds", isBodyweight: true)
        let curl = Exercise(id: "demo-ex-curl", slug: "pronated-curl-eccentric", name: "Pronated Curl Eccentric",
                            family: "pull", defaultMeasure: "reps", isBodyweight: false)
        let wallHold = Exercise(id: "demo-ex-wall-hs", slug: "wall-handstand-hold", name: "Wall Handstand Hold",
                                family: "handstand", defaultMeasure: "hold_seconds", isBodyweight: true)
        let externalRotation = Exercise(id: "demo-ex-band-er", slug: "band-external-rotation",
                                        name: "Aussenrotation mit Band", family: "mobility", defaultMeasure: "reps",
                                        isBodyweight: false)
        try db.replaceExercises([pullUp, scapular, tuck, advTuck, bandTuck, curl, wallHold, externalRotation],
                                contentVersion: "demo")
        try db.replaceBands(DemoBands.all)

        // The map.
        func skill(_ slug: String, _ name: String, _ family: String, tier: Int, milestone: Bool,
                   x: Double, y: Double, summary: String = "", faults: [String] = []) -> Skill {
            Skill(id: "demo-skill-\(slug)", slug: slug, name: name, family: family, difficultyTier: tier,
                  isMilestone: milestone, summary: summary, commonFaults: faults, constellation: "demo", x: x, y: y)
        }
        let frontLever = skill(
            "front-lever", "Front Lever", "pull", tier: 7, milestone: true, x: 340, y: 120,
            summary: "Waagrechter Halt an der Stange: Körper gestreckt, Arme gerade, Blick nach oben.",
            faults: ["Hips sagging below shoulder line", "Elbows bending to shorten the lever"])
        let pull = skill("pull-up", "Pull-up", "pull", tier: 2, milestone: false, x: 340, y: 360)
        let handstand = skill("handstand", "Handstand", "handstand", tier: 3, milestone: true, x: 80, y: 240)
        let pushUp = skill("push-up", "Push-up", "push", tier: 1, milestone: false, x: 170, y: 380)
        let dip = skill("dip", "Dip", "push", tier: 3, milestone: false, x: 520, y: 400)
        let planche = skill("planche", "Planche", "push", tier: 8, milestone: true, x: 160, y: 120)
        let muscleUp = skill("muscle-up", "Muscle-up", "dynamic", tier: 5, milestone: true, x: 500, y: 220)

        let strict5 = SkillLevel(
            id: "demo-level-strict-5", skillId: pull.id, orderIndex: 1, slug: "strict-5", name: "Five Strict Pull-ups",
            criteria: UnlockCriteria(all: [UnlockCondition(exercise: "pull-up", measure: "reps", value: 5,
                                                           occurrences: 2, withinDays: 30)]))
        let tuckLevel = SkillLevel(
            id: "demo-level-tuck", skillId: frontLever.id, orderIndex: 1, slug: "tuck", name: "Tuck Front Lever",
            estWeeksFromPrev: 6,
            criteria: UnlockCriteria(all: [UnlockCondition(exercise: "front-lever-tuck", measure: "hold_seconds",
                                                           value: 15, occurrences: 2, withinDays: 30)]))
        let advLevel = SkillLevel(
            id: "demo-level-adv-tuck", skillId: frontLever.id, orderIndex: 2, slug: "advanced-tuck",
            name: "Advanced Tuck Front Lever")
        let wall = SkillLevel(id: "demo-level-wall", skillId: handstand.id, orderIndex: 1, slug: "wall",
                              name: "Wall Handstand")
        func only(_ s: Skill) -> SkillLevel {
            SkillLevel(id: "demo-level-\(s.slug)", skillId: s.id, orderIndex: 1, slug: s.slug, name: s.name)
        }
        let pushUpLevel = only(pushUp), dipLevel = only(dip)
        let plancheLevel = only(planche), muscleUpLevel = only(muscleUp)

        try db.replaceSkillMap(SkillMapSnapshot(
            contentVersion: "demo",
            skills: [
                SkillWithLevels(skill: frontLever, levels: [tuckLevel, advLevel]),
                SkillWithLevels(skill: pull, levels: [strict5]),
                SkillWithLevels(skill: handstand, levels: [wall]),
                SkillWithLevels(skill: pushUp, levels: [pushUpLevel]),
                SkillWithLevels(skill: dip, levels: [dipLevel]),
                SkillWithLevels(skill: planche, levels: [plancheLevel]),
                SkillWithLevels(skill: muscleUp, levels: [muscleUpLevel]),
            ],
            edges: [
                SkillEdge(fromLevelId: strict5.id, toLevelId: tuckLevel.id),
                SkillEdge(fromLevelId: tuckLevel.id, toLevelId: advLevel.id),
                SkillEdge(fromLevelId: pushUpLevel.id, toLevelId: plancheLevel.id),
                SkillEdge(fromLevelId: strict5.id, toLevelId: muscleUpLevel.id),
                SkillEdge(fromLevelId: dipLevel.id, toLevelId: muscleUpLevel.id),
            ],
            states: [
                LevelState(levelId: strict5.id, state: "unlocked", bestValue: 8, bestUnit: "reps",
                           firstAchievedAt: day(20, 18, 30), verification: "auto", attemptsCount: 9),
                LevelState(levelId: tuckLevel.id, state: "unlocked", bestValue: 18, bestUnit: "hold_seconds",
                           firstAchievedAt: day(4, 18, 40), verification: "auto", attemptsCount: 7),
                LevelState(levelId: advLevel.id, state: "available", bestValue: 6, bestUnit: "hold_seconds"),
                LevelState(levelId: wall.id, state: "in_progress", bestValue: 30, bestUnit: "hold_seconds"),
                LevelState(levelId: pushUpLevel.id, state: "available", bestValue: 6, bestUnit: "reps"),
                LevelState(levelId: dipLevel.id, state: "available"),
                LevelState(levelId: plancheLevel.id, state: "locked"),
                LevelState(levelId: muscleUpLevel.id, state: "locked"),
            ]),
            now: now)
        try db.saveProgress(AthleteProgress(xpTotal: 340, currentDays: 12, longestDays: 12, freezeCredits: 1))
        // The note as content/skills/front-lever.yaml has it, with the
        // disclaimer as the API sends it (internal/http/dto_progress.go).
        try db.saveSkillNotes(SkillNotes(
            skillSlug: frontLever.slug,
            injuries: [InjuryNote(
                region: "elbow", name: "Medial epicondylalgia", details: "",
                riskFactors: ["Rapid volume jumps", "Full-lay attempts before straddle is solid"],
                prehabExerciseSlugs: ["pronated-curl-eccentric"])],
            disclaimer: "Educational information about training-related injury risk, not medical advice. "
                + "It cannot diagnose or treat anything. If you have pain or an injury, see a qualified clinician."))

        // The log, as if synced: past sessions and today's draft.
        var seq: Int64 = 0
        var sessions: [Session] = [], blocks: [Block] = [], sets: [SetWithElements] = []
        func session(_ start: Date, minutes: Int, title: String = "", restDay: Bool = false,
                     draft: Bool = false, fatigue: Int? = nil, _ blockSets: [[(Exercise, Double, Int?)]]) -> String {
            seq += 1
            let end = start.addingTimeInterval(TimeInterval(minutes * 60))
            let s = Session(
                startedAt: start, endedAt: draft ? nil : end, timezone: zone.identifier, localDate: localDate(start),
                title: title, perceivedFatigue: fatigue, status: draft ? "draft" : "completed", isRestDay: restDay,
                completedAt: draft ? nil : end, updatedAt: end, serverSeq: seq)
            sessions.append(s)
            var at = start
            for (b, entries) in blockSets.enumerated() {
                let block = Block(sessionId: s.id, orderIndex: b, updatedAt: end, serverSeq: seq)
                blocks.append(block)
                for (n, (exercise, amount, rest)) in entries.enumerated() {
                    at = at.addingTimeInterval(TimeInterval(60 + (rest ?? 90)))
                    let entry = SetEntry(sessionId: s.id, blockId: block.id, orderIndex: n, restAfterPlannedS: 120,
                                         restAfterActualS: rest, completedAt: at, updatedAt: end, serverSeq: seq)
                    let isHold = exercise.defaultMeasure == "hold_seconds"
                    sets.append(SetWithElements(entry: entry, elements: [SetElement(
                        setEntryId: entry.id, orderIndex: 0, exerciseId: exercise.id, measure: exercise.defaultMeasure,
                        reps: isHold ? nil : Int(amount), holdSeconds: isHold ? amount : nil, formQuality: 4)]))
                }
            }
            return s.id
        }

        _ = session(day(13, 18, 0), minutes: 50, title: "Zug + Handstand", fatigue: 6,
                    [[(pullUp, 6, 120), (pullUp, 6, 120), (pullUp, 5, 130)], [(wallHold, 20, 90), (wallHold, 25, nil)]])
        _ = session(day(11, 10, 0), minutes: 1, restDay: true, [])
        _ = session(day(9, 17, 30), minutes: 45, fatigue: 7,
                    [[(tuck, 10, 120), (tuck, 12, 120), (tuck, 12, 150)], [(curl, 8, 90), (curl, 8, nil)]])
        _ = session(day(6, 7, 15), minutes: 45, title: "Zug + Handstand", fatigue: 6,
                    [[(pullUp, 7, 120), (pullUp, 6, 125), (pullUp, 6, 130)], [(wallHold, 20, 90), (wallHold, 25, nil)]])
        _ = session(day(4, 18, 5), minutes: 45, fatigue: 7,
                    [[(tuck, 12, 120), (tuck, 15, 150), (tuck, 18, 150)], [(scapular, 10, 90), (scapular, 10, nil)]])
        _ = session(day(3, 9, 0), minutes: 1, restDay: true, [])
        let past = session(day(2, 17, 40), minutes: 55, title: "Zug + Handstand", fatigue: 7,
                           [[(pullUp, 8, 120), (pullUp, 7, 125), (pullUp, 6, 130)],
                            [(wallHold, 25, 90), (wallHold, 30, nil)]])
        let draftStart = now.addingTimeInterval(-20 * 60)
        let draft = session(draftStart, minutes: 0, draft: true, [[(pullUp, 8, 120), (pullUp, 7, 125)], []])
        try db.apply(ServerPage(cursor: seq, sessions: sessions, blocks: blocks, sets: sets))

        // The plan of this week, kept as the server would send it, and today's
        // session started from it, its targets open.
        let plan = DemoPlan(now: now, calendar: calendar, pastSessionId: past)
        seq += 1
        let planned = plan.startedSession(
            warmup: scapular, hold: advTuck, strength: pullUp, finish: externalRotation, zone: zone, now: now, seq: seq)
        if screen.opensPlanned {
            try db.apply(ServerPage(cursor: seq, sessions: [planned.session], blocks: planned.blocks, sets: planned.sets))
        }
        try db.savePlan(CachedPlan(
            weekStart: plan.weekStart, etag: nil,
            payload: Data(plan.json(startedAs: screen.opensPlanned ? planned.session.id : nil).utf8),
            fetchedAt: now))

        // The logger screens open on the draft with a combo logged 48 s ago:
        // the rest runs, 1:42 of 2:30 left.
        var logger: LoggerModel?
        if screen.opensPlanned {
            // The warm-up is done, its last set 48 s ago; the calibration hold is next.
            let clock: @Sendable () -> Date = { Date().addingTimeInterval(-48) }
            let model = try LoggerModel(db: db, sessionId: planned.session.id, now: clock)
            for set in model.openPlannedSets.prefix(2) {
                let drafts = set.elements.map(ElementDraft.init(copying:))
                try model.perform(plannedSet: set.id, elements: drafts, rir: set.entry.rir == nil ? nil : 3,
                                  sirS: set.entry.sirS == nil ? nil : 3)
            }
            logger = model
        }
        if screen.opensLogger {
            let clock: @Sendable () -> Date = { Date().addingTimeInterval(-48) }
            let model = try LoggerModel(db: db, sessionId: draft, now: clock)
            if let lastBlock = model.tree.blocks.last {
                try model.logCombo(in: lastBlock.block.id, elements: [
                    ElementDraft(exerciseId: advTuck.id, measure: "hold_seconds", holdSeconds: 6, formQuality: 4),
                    ElementDraft(exerciseId: tuck.id, measure: "hold_seconds", holdSeconds: 10, formQuality: 4),
                ], restPlannedSeconds: 150)
            }
            logger = model
        }

        return DemoRun(
            screen: screen, logger: logger, draftSessionId: draft, plannedSessionId: planned.session.id,
            pastSessionId: past,
            skillId: frontLever.id, exerciseId: tuck.id, bandId: DemoBands.light.id, bandExerciseId: pullUp.id,
            celebration: screen == .celebration
                ? Celebration(sessionId: draft,
                              unlocked: [.init(levelId: tuckLevel.id, levelName: tuckLevel.name, skillName: frontLever.name)],
                              xp: 45, newlyAvailable: 1, streakDays: 12)
                : nil,
            catalogue: try OnboardingCatalogue(json: Data(DemoCatalogue.json.utf8)),
            onboardingForm: screen.onboardingPage == nil ? nil : DemoOnboarding.form(for: screen))
    }
}

/// The athlete's own bands in the demo: fixture data, as a test's, not a
/// manufacturer's figures.
enum DemoBands {
    static let light = Band(
        id: "demo-band-light", brand: "Loop", colourLabel: "Rot", resistanceMinKg: 5, resistanceMaxKg: 15)
    static let strong = Band(
        id: "demo-band-strong", brand: "Loop", colourLabel: "Schwarz", resistanceMinKg: 15, resistanceMaxKg: 35)
    static let all = [light, strong]
}

/// The onboarding's answers so far, as far as each demo screen has come.
/// Names follow the knowledge base (content/training).
enum DemoOnboarding {
    static func form(for screen: DemoScreen) -> OnboardingForm {
        var f = OnboardingForm()
        f.birthYear = 1994
        f.healthConsent = true
        f.disclaimerAck = true
        f.goals = [.init(skill: "planche", targetLevel: "tuck", targetDate: nil),
                   .init(skill: "handstand", targetLevel: "free-10s")]
        guard screen != .onboarding else { return f }
        f.sessionsPerWeek = 3
        f.sessionMinutes = 60
        f.equipment = ["pull_up_bar", "parallettes", "rings"]
        f.bodyweightKg = 74
        f.trainingLevel = "recreational"
        f.trainingAge = "1_to_4_years"
        f.lastRegular = "current_or_lt_3_weeks"
        f.classes = ["push_up_class": "13_20", "pull_up_class": "4_7", "dip_class": "4_7", "hollow_hold_class": "15_30"]
        f.stages = ["planche": .init(level: "tuck", answerClass: "4_9"), "handstand": .init(level: "wall")]
        f.mobility = ["wrist": "yes"]
        guard screen == .complaint else { return f }
        f.stages["handstand"]?.answerClass = "10_19"
        f.mobility["shoulder"] = "partly"
        f.dataConfidence = "estimated"
        f.exertion = false
        f.screening = ["SCR-01": false, "SCR-02": false, "SCR-03": false, "SCR-04": false, "SCR-05": false,
                       "SCR-06": false]
        f.bodyMap = ["elbow_inner": .init(current: true)]
        var complaint = OnboardingForm.Complaint()
        complaint.painDaily = 2
        complaint.painTraining = 4
        complaint.onset = "gradual"
        complaint.durationWeeks = 3
        complaint.suspectedSerious = "no"
        complaint.assessment = "no"
        f.complaints = ["elbow_inner": complaint]
        f.redFlags = ["elbow_inner": ["RF-01": false, "RF-02": false, "RF-03": false, "RF-04": true]]
        return f
    }
}

/// This week's plan for the demo: fixture data in the API's JSON, as the
/// server would send it, with the knowledge base's names and texts
/// (content/training). Today has a full session, two days ago the one the
/// history shows as done, two days on a light one; the other days rest.
struct DemoPlan {
    let weekStart: String
    let today: String
    let days: [String]
    let pastSessionId: String
    let todayId = "demo-planned-today"

    static let warmupItem = "demo-item-scapula"
    static let holdItem = "demo-item-adv-tuck"
    static let strengthItem = "demo-item-pull-up"
    static let finishItem = "demo-item-rotation"

    init(now: Date, calendar: Calendar, pastSessionId: String) {
        var iso = Calendar(identifier: .iso8601)
        iso.timeZone = calendar.timeZone
        let start = iso.dateInterval(of: .weekOfYear, for: now)?.start ?? now
        days = (0..<7).map { Self.day(iso.date(byAdding: .day, value: $0, to: start) ?? start, iso) }
        weekStart = days[0]
        today = Self.day(now, iso)
        self.pastSessionId = pastSessionId
    }

    static func day(_ date: Date, _ calendar: Calendar) -> String {
        let c = calendar.dateComponents([.year, .month, .day], from: date)
        return String(format: "%04d-%02d-%02d", c.year ?? 0, c.month ?? 0, c.day ?? 0)
    }

    /// The plan, with today's session started as `logSessionId` if given.
    func json(startedAs logSessionId: String?) -> String {
        guard let t = days.firstIndex(of: today) else { return "" }
        var sessions: [String] = [], rest: [String] = []
        for (i, day) in days.enumerated() {
            switch i - t {
            case 0: sessions.append(todaySession(startedAs: logSessionId))
            case -2: sessions.append(session(
                "demo-planned-past", day, index: i, kind: "full", minutes: 50, status: "completed",
                workout: pastSessionId, blocks: [block("strength", [item(
                    "demo-item-past", "pull-up", "Klimmzug", sets: 3, reps: 6, hold: nil, reserve: 2, rest: 120)],
                    why: Self.strengthWhy)]))
            case 2: sessions.append(session(
                "demo-planned-light", day, index: i, kind: "light", minutes: 30, status: "planned", workout: nil,
                blocks: [
                    block("balance", [item(
                        "demo-item-wall", "wall-handstand-hold", "Handstand an der Wand", sets: 3, reps: nil, hold: 20,
                        reserve: 5, rest: 90)], why: Self.balanceWhy),
                    block("end", [item(
                        "demo-item-rotation-light", "band-external-rotation", "Aussenrotation mit Band", sets: 2,
                        reps: 15, hold: nil, reserve: 3, rest: 60, intensity: "light")], why: Self.finishWhy),
                ], why: [Self.lightWhy]))
            default: rest.append("\"\(day)\"")
            }
        }
        return """
        {"id":"demo-plan","week_start":"\(weekStart)","ruleset_version":"demo",
         "input_hash":"sha256:0000000000000000000000000000000000000000000000000000000000000000",
         "stopped":false,"deload":null,"sessions":[\(sessions.joined(separator: ","))],
         "rest_days":[\(rest.joined(separator: ","))],"exclusions":[],"hints":[],"realism":[],"monitor":[],
         "loads":[],"reasons":[],"disclaimer":"\(Self.disclaimer)"}
        """
    }

    private func todaySession(startedAs logSessionId: String?) -> String {
        session(
            todayId, today, index: days.firstIndex(of: today) ?? 0, kind: "full", minutes: 45,
            status: logSessionId == nil ? "planned" : "started", workout: logSessionId,
            blocks: [
                block("warmup", [item(
                    Self.warmupItem, "scapular-pull-up", "Scapula-Pull-up", sets: 2, reps: 8, hold: nil, reserve: 3,
                    rest: 60, kind: "warmup", intensity: "light")], minutes: 8, why: Self.warmupWhy),
                block("skill_max", [
                    item(Self.holdItem, "front-lever-advanced-tuck", "Advanced Tuck Front Lever", sets: 4, reps: nil,
                         hold: 6, reserve: 2, rest: 150, calibration: true, intensity: "hard",
                         why: Self.calibrationWhy),
                    item("demo-item-straddle", "front-lever-straddle", "Straddle Front Lever", sets: 1, reps: nil,
                         hold: 3, reserve: 1, rest: 180, offer: true, intensity: "hard"),
                ], why: Self.holdWhy),
                block("strength", [item(
                    Self.strengthItem, "pull-up", "Klimmzug", sets: 3, reps: 6, hold: nil, reserve: 2, rest: 120)],
                    why: Self.strengthWhy),
                block("end", [item(
                    Self.finishItem, "band-external-rotation", "Aussenrotation mit Band", sets: 2, reps: 15, hold: nil,
                    reserve: 3, rest: 60, intensity: "light", monitor: true)], why: Self.finishWhy),
            ],
            why: [Self.orderWhy])
    }

    /// Today's session as the start writes it into the log (ADR 0016).
    func startedSession(
        warmup: Exercise, hold: Exercise, strength: Exercise, finish: Exercise, zone: TimeZone, now: Date, seq: Int64
    ) -> (session: Session, blocks: [Block], sets: [SetWithElements]) {
        let started = now.addingTimeInterval(-12 * 60)
        let session = Session(
            id: "demo-session-planned", startedAt: started, timezone: zone.identifier, localDate: today,
            plannedSessionId: todayId, updatedAt: started, serverSeq: seq)
        var blocks: [Block] = [], sets: [SetWithElements] = []
        let plan: [(Exercise, String, Int, Int?, Double?, Int, Int, String)] = [
            (warmup, Self.warmupItem, 2, 8, nil, 3, 60, "warmup"),
            (hold, Self.holdItem, 4, nil, 6, 2, 150, "working"),
            (strength, Self.strengthItem, 3, 6, nil, 2, 120, "working"),
            (finish, Self.finishItem, 2, 15, nil, 3, 60, "working"),
        ]
        for (b, (exercise, itemId, count, reps, holdSeconds, reserve, rest, kind)) in plan.enumerated() {
            let block = Block(id: "demo-planned-block-\(b)", sessionId: session.id, orderIndex: b, updatedAt: started,
                              serverSeq: seq)
            blocks.append(block)
            for n in 0..<count {
                let entry = SetEntry(
                    id: "demo-planned-set-\(b)-\(n)", sessionId: session.id, blockId: block.id, orderIndex: n,
                    kind: kind, isPlanned: true, restAfterPlannedS: rest,
                    rir: holdSeconds == nil ? reserve : nil, sirS: holdSeconds == nil ? nil : reserve,
                    plannedItemId: itemId, updatedAt: started, serverSeq: seq)
                sets.append(SetWithElements(entry: entry, elements: [SetElement(
                    setEntryId: entry.id, orderIndex: 0, exerciseId: exercise.id, measure: exercise.defaultMeasure,
                    reps: reps, holdSeconds: holdSeconds)]))
            }
        }
        return (session, blocks, sets)
    }

    // MARK: JSON pieces

    private func session(
        _ id: String, _ day: String, index: Int, kind: String, minutes: Int, status: String, workout: String?,
        blocks: [String], why: [String] = []
    ) -> String {
        """
        {"id":"\(id)","status":"\(status)","workout_session_id":\(workout.map { "\"\($0)\"" } ?? "null"),
         "check_in_applied":false,"index":\(index),"date":"\(day)","kind":"\(kind)","est_minutes":\(minutes),
         "blocks":[\(blocks.joined(separator: ","))],"reasons":[\(why.joined(separator: ","))]}
        """
    }

    private func block(_ role: String, _ items: [String], minutes: Int? = nil, why: String) -> String {
        """
        {"role":"\(role)","minutes":\(minutes.map(String.init) ?? "null"),"paired":false,
         "items":[\(items.joined(separator: ","))],"reasons":[\(why)]}
        """
    }

    private func item(
        _ id: String, _ slug: String, _ name: String, sets: Int, reps: Int?, hold: Int?, reserve: Int, rest: Int,
        kind: String = "working", calibration: Bool = false, offer: Bool = false, intensity: String = "moderate",
        monitor: Bool = false, why: String? = nil
    ) -> String {
        """
        {"id":"\(id)","exercise":"\(slug)","exercise_name":"\(name)","skill":null,"stimulus":"strength",
         "kind":"\(kind)","sets":\(sets),"reps":\(reps.map(String.init) ?? "null"),
         "hold_s":\(hold.map(String.init) ?? "null"),"load_kg":null,"assist":"none","reserve":\(reserve),
         "rest_s":\(rest),"calibration":\(calibration),"offer":\(offer),"stop_rules":[],"intensity":"\(intensity)",
         "role":"goal","monitor":\(monitor),"reasons":[\(why ?? "")]}
        """
    }

    private static func reason(_ rule: String, _ text: String) -> String {
        """
        {"rule_id":"\(rule)","params":[],"sources":[],"evidence":"H","text":"\(text)","args":{},"region":null}
        """
    }

    // The knowledge base's texts (content/training/rules.yaml, body.yaml).
    static let disclaimer = DemoCatalogue.disclaimer
    static let orderWhy = reason("SESS-02", "Reihenfolge: Aufwärmen, Balance und Technik, dann die schwersten Halte, "
        + "danach Volumen, Kraft und zuletzt belastende Prehab. Schweres kommt, solange du frisch bist.")
    static let warmupWhy = reason("SESS-03", "Aufwärmen mit allgemeinem Teil, Prehab-Aktivierung und leichten "
        + "Rampensätzen vor dem ersten schweren Block.")
    static let balanceWhy = reason("SESS-04", "Balance übst du früh in der Einheit, weil Ermüdung die Kontrolle "
        + "verschlechtert.")
    static let holdWhy = reason("SESS-05", "Alle Sätze eines schweren Halts hintereinander, mit langen Pausen, damit "
        + "jeder Satz in guter Qualität gelingt.")
    static let strengthWhy = reason("SESS-07", "Kraftübungen für den Weg zum Ziel und zum Ausgleich; Gegenspieler "
        + "können abwechselnd trainiert werden.")
    static let finishWhy = reason("SESS-08", "Belastende Prehab und Rumpfarbeit kommen ans Ende, damit sie die schweren "
        + "Übungen nicht vorermüden.")
    static let lightWhy = reason("SESS-09", "Leichte Einheit: Balance, Technik und Prehab ohne harte Reize, damit du "
        + "dich von den vollen Einheiten erholst.")
    static let calibrationWhy = reason("SEL-08", "Die ersten Einheiten liegen eine Stufe unter deiner Angabe, bis Logs "
        + "sie bestätigen.")
}
#endif
