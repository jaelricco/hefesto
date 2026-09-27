import Foundation
import GRDB
import Testing
@testable import HefestoStore

/// Pull-up (two levels) leads to front lever (one level).
func pullSnapshot(version: String = "v1", states: [LevelState]) -> SkillMapSnapshot {
    let pull = Skill(id: "s-pull", slug: "pull-up", name: "Pull-up", family: "pull", difficultyTier: 2,
                     isMilestone: false, constellation: "pull_north", x: 340, y: 360)
    let lever = Skill(id: "s-lever", slug: "front-lever", name: "Front Lever", family: "pull", difficultyTier: 6,
                      isMilestone: true, constellation: "pull_north", x: 340, y: 120)
    return SkillMapSnapshot(
        contentVersion: version,
        skills: [
            SkillWithLevels(skill: pull, levels: [
                SkillLevel(id: "l-pull-5", skillId: "s-pull", orderIndex: 1, slug: "strict-5", name: "Five",
                           criteria: UnlockCriteria(all: [UnlockCondition(exercise: "pull-up", measure: "reps", value: 5)])),
                SkillLevel(id: "l-pull-10", skillId: "s-pull", orderIndex: 2, slug: "strict-10", name: "Ten"),
            ]),
            SkillWithLevels(skill: lever, levels: [
                SkillLevel(id: "l-tuck", skillId: "s-lever", orderIndex: 1, slug: "tuck", name: "Tuck"),
            ]),
        ],
        edges: [
            SkillEdge(fromLevelId: "l-pull-5", toLevelId: "l-pull-10"),
            SkillEdge(fromLevelId: "l-pull-10", toLevelId: "l-tuck"),
            SkillEdge(fromLevelId: "l-pull-5", toLevelId: "l-tuck", relation: "recommended"),
        ],
        states: states)
}

func map(_ db: AppDatabase) throws -> SkillMapData { try db.reader.read { try AppDatabase.skillMap($0) } }

@Suite struct SkillMapStoreTests {
    @Test func theFirstRefreshCountsExistingUnlocksAsSeen() throws {
        let db = try AppDatabase.inMemory()
        try db.replaceSkillMap(pullSnapshot(states: [
            LevelState(levelId: "l-pull-5", state: "unlocked", firstAchievedAt: Date(timeIntervalSince1970: 1)),
            LevelState(levelId: "l-pull-10", state: "available"),
        ]))
        let m = try map(db)
        #expect(m.skills.map(\.skill.slug) == ["front-lever", "pull-up"])
        #expect(m.skills.last?.levels.map(\.slug) == ["strict-5", "strict-10"])
        #expect(m.skills.last?.levels.first?.criteria.all.first?.value == 5)
        #expect(m.unseenUnlocks.isEmpty, "history is not replayed on the first load")
    }

    @Test func aNewUnlockIsUnseenUntilTheMapShowsIt() throws {
        let db = try AppDatabase.inMemory()
        try db.replaceSkillMap(pullSnapshot(states: [LevelState(levelId: "l-pull-5", state: "unlocked")]))
        try db.replaceSkillMap(pullSnapshot(states: [
            LevelState(levelId: "l-pull-5", state: "unlocked"),
            LevelState(levelId: "l-pull-10", state: "unlocked", firstAchievedAt: Date()),
        ]))
        var m = try map(db)
        #expect(m.unseenUnlocks.map(\.levelId) == ["l-pull-10"])
        #expect(m.skillLinks.first { $0.from == "s-pull" && $0.to == "s-lever" }?.newlyLit == true)

        try db.markUnlocksSeen(["l-pull-10"])
        m = try map(db)
        #expect(m.unseenUnlocks.isEmpty)
        #expect(m.skillLinks.first?.newlyLit == false)
        #expect(m.skillLinks.first?.lit == true)
    }

    @Test func unlocksAreNeverRevoked() throws {
        let db = try AppDatabase.inMemory()
        let achieved = Date(timeIntervalSince1970: 1_700_000_000)
        try db.replaceSkillMap(pullSnapshot(states: [
            LevelState(levelId: "l-pull-5", state: "unlocked", firstAchievedAt: achieved, verification: "auto"),
        ]))
        try db.replaceSkillMap(pullSnapshot(states: [
            LevelState(levelId: "l-pull-5", state: "in_progress", bestValue: 3, bestUnit: "reps"),
        ]))
        let s = try #require(try map(db).states["l-pull-5"])
        #expect(s.state == "unlocked")
        #expect(s.firstAchievedAt == achieved)
        #expect(s.bestValue == 3, "current form still updates")
    }

    @Test func contentIsReplacedOnlyWhenItsVersionChanges() throws {
        let db = try AppDatabase.inMemory()
        try db.replaceSkillMap(pullSnapshot(states: []))
        var changed = pullSnapshot(states: [])
        changed.skills.removeLast()
        try db.replaceSkillMap(changed) // same version: content kept
        #expect(try map(db).skills.count == 2)
        changed.contentVersion = "v2"
        try db.replaceSkillMap(changed)
        #expect(try map(db).skills.map(\.skill.slug) == ["pull-up"])
        #expect(try map(db).contentVersion == "v2")
    }

    @Test func standingSummarisesTheLevels() throws {
        let db = try AppDatabase.inMemory()
        try db.replaceSkillMap(pullSnapshot(states: [
            LevelState(levelId: "l-pull-5", state: "unlocked"),
            LevelState(levelId: "l-pull-10", state: "in_progress"),
            LevelState(levelId: "l-tuck", state: "locked"),
        ]))
        let m = try map(db)
        let pull = try #require(m.skills.first { $0.skill.slug == "pull-up" })
        let lever = try #require(m.skills.first { $0.skill.slug == "front-lever" })
        #expect(m.standing(of: pull) == .progressing)
        #expect(m.standing(of: lever) == .locked)
        #expect(m.nextLevel(of: pull)?.slug == "strict-10")
        #expect(m.skillLinks.count == 1, "only prerequisites between different skills draw a line")
    }
}

@Suite struct HistoryTests {
    func session(_ date: String, restDay: Bool = false) -> Session {
        Session(startedAt: Date(), timezone: "Europe/Zurich", localDate: date, isRestDay: restDay)
    }

    @Test func weeksStartOnMondayAndFollowISONumbering() {
        let weeks = History.weeks([
            session("2026-09-21"), // Monday, week 39
            session("2026-09-27", restDay: true), // Sunday, still week 39
            session("2026-09-28"), // Monday, week 40
            session("2027-01-01"), // Friday: ISO week 53 of 2026
        ])
        #expect(weeks.map(\.monday) == ["2026-12-28", "2026-09-28", "2026-09-21"])
        #expect(weeks.map(\.week) == [53, 40, 39])
        #expect(weeks[0].year == 2026)
        #expect(weeks[2].sessions.map(\.localDate) == ["2026-09-27", "2026-09-21"])
        #expect(weeks[2].trainingCount == 1, "planned rest is not counted as training")
    }

    func row(_ date: String, set: String, reps: Int? = nil, hold: Double? = nil, load: Double = 0,
             assisted: Bool = false, partial: Bool = false) -> StatRow {
        StatRow(localDate: date, setId: set, reps: reps, holdSeconds: hold, distanceM: nil, loadKg: load,
                assisted: assisted, partial: partial, eccentric: false)
    }

    @Test func bestsCountOnlyCleanRepetitions() {
        let stats = ExerciseStats.compute(exerciseId: "e", rows: [
            row("2026-09-01", set: "a", reps: 6),
            row("2026-09-01", set: "b", reps: 12, assisted: true),
            row("2026-09-03", set: "c", reps: 9, partial: true),
            row("2026-09-05", set: "d", reps: 8, load: 10),
            row("2026-09-05", set: "d", hold: 15), // a combo: same set, second element
        ])
        #expect(stats.days.map(\.localDate) == ["2026-09-01", "2026-09-03", "2026-09-05"])
        #expect(stats.days[0].sets == 2 && stats.days[0].totalReps == 18)
        #expect(stats.days[0].bestReps == 6, "the assisted set is volume, not a best")
        #expect(stats.days[1].bestReps == nil)
        #expect(stats.days[2].sets == 1, "a combo is one set")
        #expect(stats.bestReps == PersonalBest(value: 8, localDate: "2026-09-05"))
        #expect(stats.maxLoadKg == PersonalBest(value: 10, localDate: "2026-09-05"))
        #expect(stats.bestHoldSeconds?.value == 15)
    }

    @Test func statsReadTheAthletesLoggedSets() throws {
        let (db, session, block) = try seeded()
        try db.replaceExercises([Exercise(id: "pull-up-id", slug: "pull-up", name: "Pull-up", family: "pull",
                                          defaultMeasure: "reps", isBodyweight: true)], contentVersion: "v1")
        for (i, reps) in [5, 7].enumerated() {
            let entry = SetEntry(sessionId: session.id, blockId: block.id, orderIndex: i)
            try db.saveSet(SetWithElements(entry: entry, elements: [pullUps(entry, reps: reps)]))
        }
        let stats = try db.reader.read { try AppDatabase.exerciseStats($0, exerciseId: "pull-up-id") }
        #expect(stats.bestReps?.value == 7)
        #expect(stats.days.first?.totalReps == 12)
        let logged = try db.reader.read { try AppDatabase.loggedExercises($0) }
        #expect(logged.map(\.exercise.slug) == ["pull-up"])
        #expect(logged.first?.sets == 2)
    }
}
