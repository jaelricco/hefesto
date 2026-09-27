#if DEBUG
import Foundation
import HefestoAPI
import HefestoAuth
import HefestoLogger
import HefestoStore
import HefestoSync

/// Local data for the UI tests' accessibility audit: the app opens signed in,
/// on an in-memory database, and never talks to a server. It exists only in
/// Debug builds and only when launched with `-uiTestFixture`.
///
/// These are test fixtures, not content: names only, no coaching data.
enum UITestFixture {
    static var isRequested: Bool { ProcessInfo.processInfo.arguments.contains("-uiTestFixture") }

    @MainActor
    static func model() throws -> AppModel {
        let db = try AppDatabase.inMemory()
        try seed(db)
        let nowhere = URL(string: "http://127.0.0.1:9")!
        let auth = AuthService(client: HefestoAPIConfiguration.client(serverURL: nowhere),
                               store: InMemoryTokenStore(), deviceId: "ui-test")
        return AppModel(db: db, auth: auth,
                        sync: SyncEngine(client: HefestoAPIConfiguration.client(serverURL: nowhere), db: db),
                        offline: true)
    }

    @MainActor
    static func seed(_ db: AppDatabase) throws {
        let pull = Exercise(id: "e-pull", slug: "pull-up", name: "Pull-up", family: "pull",
                            defaultMeasure: "reps", isBodyweight: true)
        let tuck = Exercise(id: "e-tuck", slug: "tuck-hold", name: "Tuck hold", family: "pull",
                            defaultMeasure: "hold_seconds", isBodyweight: true)
        try db.replaceExercises([pull, tuck], contentVersion: "ui-test")

        let skillA = Skill(id: "s-a", slug: "skill-a", name: "Pull-up", family: "pull", difficultyTier: 2,
                           isMilestone: true, constellation: "north", x: 100, y: 300)
        let skillB = Skill(id: "s-b", slug: "skill-b", name: "Tuck hold", family: "pull", difficultyTier: 4,
                           isMilestone: true, constellation: "north", x: 260, y: 120)
        let skillC = Skill(id: "s-c", slug: "skill-c", name: "Handstand", family: "handstand", difficultyTier: 3,
                           isMilestone: false, constellation: "west", x: 20, y: 80)
        let criteria = UnlockCriteria(all: [UnlockCondition(exercise: "pull-up", measure: "reps", value: 1)])
        try db.replaceSkillMap(SkillMapSnapshot(
            contentVersion: "ui-test",
            skills: [
                SkillWithLevels(skill: skillA, levels: [
                    SkillLevel(id: "l-a1", skillId: "s-a", orderIndex: 1, slug: "one", name: "Level one", criteria: criteria,
                               exercises: [LevelExercise(exerciseId: "e-pull", slug: "pull-up", role: "primary_test")]),
                    SkillLevel(id: "l-a2", skillId: "s-a", orderIndex: 2, slug: "two", name: "Level two"),
                ]),
                SkillWithLevels(skill: skillB, levels: [
                    SkillLevel(id: "l-b1", skillId: "s-b", orderIndex: 1, slug: "one", name: "Level one"),
                ]),
                SkillWithLevels(skill: skillC, levels: [
                    SkillLevel(id: "l-c1", skillId: "s-c", orderIndex: 1, slug: "one", name: "Level one"),
                ]),
            ],
            edges: [SkillEdge(fromLevelId: "l-a1", toLevelId: "l-a2"), SkillEdge(fromLevelId: "l-a2", toLevelId: "l-b1")],
            states: [
                LevelState(levelId: "l-a1", state: "unlocked", bestValue: 3, bestUnit: "reps",
                           firstAchievedAt: Date(timeIntervalSinceNow: -86_400 * 3), verification: "auto"),
                LevelState(levelId: "l-a2", state: "in_progress"),
                LevelState(levelId: "l-c1", state: "available"),
            ]))
        try db.saveProgress(AthleteProgress(xpTotal: 120, currentDays: 3, longestDays: 5, freezeCredits: 1))

        // One completed session with a straight set and a combo.
        let logger = try LoggerModel.startSession(db: db, title: "")
        guard let block = logger.tree.blocks.first?.id else { return }
        try logger.logSet(in: block, element: ElementDraft(exerciseId: "e-pull", measure: "reps", reps: 3))
        try logger.logCombo(in: block, elements: [
            ElementDraft(exerciseId: "e-tuck", measure: "hold_seconds", holdSeconds: 5),
            ElementDraft(exerciseId: "e-pull", measure: "reps", reps: 2),
        ])
        try logger.complete(perceivedFatigue: 6)
    }
}
#endif
