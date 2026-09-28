import Foundation
import HefestoLogger
import HefestoStore

/// A screen a demo run opens on, for design review and the screenshot
/// workflow (.github/workflows/ios-screenshots.yml).
enum DemoScreen: String, CaseIterable {
    case today, logger, composer, finish, map, peek, detail, history, session, stats, celebration
    case signIn = "signin"

    var tab: AppTab {
        switch self {
        case .map, .peek, .detail: .map
        case .history, .session, .stats: .history
        default: .today
        }
    }
}

/// What a demo run seeded, so screens can open on it.
struct DemoRun {
    let screen: DemoScreen
    /// Today's draft, with a set logged 48 s ago and the rest running.
    let logger: LoggerModel?
    let draftSessionId: String
    let pastSessionId: String
    let skillId: String
    let exerciseId: String
    let celebration: Celebration?

    var opensLogger: Bool { [.logger, .composer, .finish].contains(screen) }
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
        try db.replaceExercises([pullUp, scapular, tuck, advTuck, bandTuck, curl, wallHold], contentVersion: "demo")

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

        // The logger screens open on the draft with a combo logged 48 s ago:
        // the rest runs, 1:42 of 2:30 left.
        var logger: LoggerModel?
        if [.logger, .composer, .finish].contains(screen) {
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
            screen: screen, logger: logger, draftSessionId: draft, pastSessionId: past,
            skillId: frontLever.id, exerciseId: tuck.id,
            celebration: screen == .celebration
                ? Celebration(sessionId: draft,
                              unlocked: [.init(levelId: tuckLevel.id, levelName: tuckLevel.name, skillName: frontLever.name)],
                              xp: 45, newlyAvailable: 1, streakDays: 12)
                : nil)
    }
}
#endif
