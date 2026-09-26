import Foundation
import GRDB

// The skill map, cached locally so the map, skill detail and celebration read
// the database like every other screen (ADR 0010). Content (skills, levels,
// edges) is replaced whole when its version changes; the athlete's level
// states are replaced on every refresh. Unlocks are never revoked
// (CLAUDE.md): a refresh never moves an unlocked level back.

public struct Skill: Codable, Sendable, Hashable, Identifiable, FetchableRecord, PersistableRecord {
    public static let databaseTableName = "skill"
    public var id: String
    public var slug: String
    public var name: String
    public var family: String
    public var difficultyTier: Int
    public var isMilestone: Bool
    public var status: String // active | draft_placeholder
    public var summary: String
    public var aka: [String]
    public var primaryMuscles: [String]
    public var commonFaults: [String]
    /// Where the skill sits on the map, as authored in content; nil keeps it off the map.
    public var constellation: String?
    public var x: Double?
    public var y: Double?

    public init(
        id: String, slug: String, name: String, family: String, difficultyTier: Int, isMilestone: Bool,
        status: String = "active", summary: String = "", aka: [String] = [], primaryMuscles: [String] = [],
        commonFaults: [String] = [], constellation: String? = nil, x: Double? = nil, y: Double? = nil
    ) {
        self.id = id; self.slug = slug; self.name = name; self.family = family
        self.difficultyTier = difficultyTier; self.isMilestone = isMilestone; self.status = status
        self.summary = summary; self.aka = aka; self.primaryMuscles = primaryMuscles
        self.commonFaults = commonFaults; self.constellation = constellation; self.x = x; self.y = y
    }
}

/// One condition of a level's unlock criteria (brief §6), as the server
/// states it with defaults made explicit.
public struct UnlockCondition: Codable, Sendable, Hashable {
    public var exercise: String // slug
    public var measure: String // reps | hold_seconds | distance_m
    public var op: String // >= | >
    public var value: Double
    public var assistance: String // none | any
    public var occurrences: Int
    public var minFormQuality: Int?
    public var minLoadKg: Double?
    public var maxLoadKg: Double?
    public var withinDays: Int?

    public init(
        exercise: String, measure: String, op: String = ">=", value: Double, assistance: String = "none",
        occurrences: Int = 1, minFormQuality: Int? = nil, minLoadKg: Double? = nil, maxLoadKg: Double? = nil,
        withinDays: Int? = nil
    ) {
        self.exercise = exercise; self.measure = measure; self.op = op; self.value = value
        self.assistance = assistance; self.occurrences = occurrences; self.minFormQuality = minFormQuality
        self.minLoadKg = minLoadKg; self.maxLoadKg = maxLoadKg; self.withinDays = withinDays
    }
}

/// A level's criteria: all of `all`, and at least one of `any` when it is
/// not empty. Both empty means the level can only be self-attested.
public struct UnlockCriteria: Codable, Sendable, Hashable {
    public var all: [UnlockCondition]
    public var any: [UnlockCondition]

    public init(all: [UnlockCondition] = [], any: [UnlockCondition] = []) { self.all = all; self.any = any }

    public var isSelfAttestOnly: Bool { all.isEmpty && any.isEmpty }
}

public struct LevelExercise: Codable, Sendable, Hashable {
    public var exerciseId: String
    public var slug: String
    public var role: String // primary_test | progression | accessory | prehab

    public init(exerciseId: String, slug: String, role: String) {
        self.exerciseId = exerciseId; self.slug = slug; self.role = role
    }
}

public struct SkillLevel: Codable, Sendable, Hashable, Identifiable, FetchableRecord, PersistableRecord {
    public static let databaseTableName = "skillLevel"
    public var id: String
    public var skillId: String
    public var orderIndex: Int
    public var slug: String
    public var name: String
    public var details: String
    public var estWeeksFromPrev: Int?
    public var criteria: UnlockCriteria
    public var exercises: [LevelExercise]

    public init(
        id: String, skillId: String, orderIndex: Int, slug: String, name: String, details: String = "",
        estWeeksFromPrev: Int? = nil, criteria: UnlockCriteria = .init(), exercises: [LevelExercise] = []
    ) {
        self.id = id; self.skillId = skillId; self.orderIndex = orderIndex; self.slug = slug; self.name = name
        self.details = details; self.estWeeksFromPrev = estWeeksFromPrev; self.criteria = criteria
        self.exercises = exercises
    }
}

public struct SkillEdge: Codable, Sendable, Hashable, FetchableRecord, PersistableRecord {
    public static let databaseTableName = "skillEdge"
    public var fromLevelId: String
    public var toLevelId: String
    public var relation: String // prerequisite | recommended | alternative | antagonist
    public var weight: Double

    public init(fromLevelId: String, toLevelId: String, relation: String = "prerequisite", weight: Double = 1) {
        self.fromLevelId = fromLevelId; self.toLevelId = toLevelId; self.relation = relation; self.weight = weight
    }
}

/// The athlete's standing on one level.
public struct LevelState: Codable, Sendable, Hashable, FetchableRecord, PersistableRecord {
    public static let databaseTableName = "levelState"
    public var levelId: String
    public var state: String // locked | available | in_progress | unlocked
    /// The best logged value for the level's headline condition: current form, not achievement.
    public var bestValue: Double?
    public var bestUnit: String?
    public var bestAt: Date?
    public var firstAchievedAt: Date?
    public var verification: String? // auto | self_attested | coach
    /// A hint that an unlocked level has not been practised for a while. It never changes `state`.
    public var staleSince: Date?
    public var attemptsCount: Int
    /// Local only: when the map first showed this unlock. Nil on an unlock
    /// not yet celebrated, which the map animates once.
    public var seenAt: Date?

    public init(
        levelId: String, state: String, bestValue: Double? = nil, bestUnit: String? = nil, bestAt: Date? = nil,
        firstAchievedAt: Date? = nil, verification: String? = nil, staleSince: Date? = nil,
        attemptsCount: Int = 0, seenAt: Date? = nil
    ) {
        self.levelId = levelId; self.state = state; self.bestValue = bestValue; self.bestUnit = bestUnit
        self.bestAt = bestAt; self.firstAchievedAt = firstAchievedAt; self.verification = verification
        self.staleSince = staleSince; self.attemptsCount = attemptsCount; self.seenAt = seenAt
    }

    public var isUnlocked: Bool { state == "unlocked" }
}

/// XP and the training streak. The streak counts planned rest as kept
/// (ADR 0003); the app shows what was kept, never what was missed.
public struct AthleteProgress: Codable, Sendable, Hashable, FetchableRecord, PersistableRecord {
    public static let databaseTableName = "athleteProgress"
    public var id = 1
    public var xpTotal: Int
    public var currentDays: Int
    public var longestDays: Int
    public var freezeCredits: Int
    public var lastCountedDate: String?

    public init(xpTotal: Int, currentDays: Int, longestDays: Int, freezeCredits: Int, lastCountedDate: String? = nil) {
        self.xpTotal = xpTotal; self.currentDays = currentDays; self.longestDays = longestDays
        self.freezeCredits = freezeCredits; self.lastCountedDate = lastCountedDate
    }
}

/// An injury note for a skill. Educational, never medical advice: it is only
/// ever shown with its disclaimer, which the server sends with it (CLAUDE.md).
public struct InjuryNote: Codable, Sendable, Hashable {
    public var region: String
    public var name: String
    public var details: String
    public var riskFactors: [String]
    public var earlySigns: [String]
    public var prehabExerciseSlugs: [String]

    public init(region: String, name: String, details: String, riskFactors: [String] = [],
                earlySigns: [String] = [], prehabExerciseSlugs: [String] = []) {
        self.region = region; self.name = name; self.details = details; self.riskFactors = riskFactors
        self.earlySigns = earlySigns; self.prehabExerciseSlugs = prehabExerciseSlugs
    }
}

/// A skill's injury notes and the disclaimer they are shown with, as fetched.
public struct SkillNotes: Codable, Sendable, Hashable, FetchableRecord, PersistableRecord {
    public static let databaseTableName = "skillNotes"
    public var skillSlug: String
    public var injuries: [InjuryNote]
    public var disclaimer: String
    public var fetchedAt: Date

    public init(skillSlug: String, injuries: [InjuryNote], disclaimer: String, fetchedAt: Date = Date()) {
        self.skillSlug = skillSlug; self.injuries = injuries; self.disclaimer = disclaimer; self.fetchedAt = fetchedAt
    }
}

// MARK: - What the map reads

public struct SkillWithLevels: Sendable, Hashable, Identifiable {
    public var skill: Skill
    public var levels: [SkillLevel]
    public var id: String { skill.id }

    public init(skill: Skill, levels: [SkillLevel]) { self.skill = skill; self.levels = levels }
}

/// Where the athlete stands on a skill as a whole, for drawing its node.
public enum SkillStanding: String, Sendable, Hashable, CaseIterable {
    /// No level is open yet.
    case locked
    /// A level is open to work on; none is unlocked.
    case available
    /// Some levels are unlocked, not all.
    case progressing
    /// Every level is unlocked.
    case mastered
}

public struct SkillMapData: Sendable, Hashable {
    public var contentVersion: String?
    public var skills: [SkillWithLevels]
    public var edges: [SkillEdge]
    public var states: [String: LevelState]

    public init(contentVersion: String?, skills: [SkillWithLevels], edges: [SkillEdge], states: [String: LevelState]) {
        self.contentVersion = contentVersion; self.skills = skills; self.edges = edges; self.states = states
    }

    public func state(of levelId: String) -> String { states[levelId]?.state ?? "locked" }

    public func standing(of skill: SkillWithLevels) -> SkillStanding {
        let levels = skill.levels.map { state(of: $0.id) }
        let unlocked = levels.filter { $0 == "unlocked" }.count
        if !levels.isEmpty, unlocked == levels.count { return .mastered }
        if unlocked > 0 { return .progressing }
        if levels.contains(where: { $0 == "available" || $0 == "in_progress" }) { return .available }
        return .locked
    }

    /// The level to work on next: the first one not unlocked, in order.
    public func nextLevel(of skill: SkillWithLevels) -> SkillLevel? {
        skill.levels.first { state(of: $0.id) != "unlocked" }
    }

    /// The map's lines between skills: one per pair of skills joined by a
    /// prerequisite between any of their levels, from the earlier skill.
    public var skillLinks: [SkillLink] {
        var skillOfLevel: [String: String] = [:]
        for s in skills { for l in s.levels { skillOfLevel[l.id] = s.id } }
        var seen = Set<SkillLink>()
        var out: [SkillLink] = []
        for e in edges where e.relation == "prerequisite" {
            guard let from = skillOfLevel[e.fromLevelId], let to = skillOfLevel[e.toLevelId], from != to else { continue }
            let link = SkillLink(from: from, to: to, lit: state(of: e.fromLevelId) == "unlocked",
                                 newlyLit: states[e.fromLevelId].map { $0.isUnlocked && $0.seenAt == nil } ?? false)
            let key = SkillLink(from: from, to: to, lit: false, newlyLit: false)
            if seen.insert(key).inserted {
                out.append(link)
            } else if let i = out.firstIndex(where: { $0.from == from && $0.to == to }) {
                out[i].lit = out[i].lit || link.lit
                out[i].newlyLit = out[i].newlyLit || link.newlyLit
            }
        }
        return out
    }

    /// Unlocks the map has not celebrated yet, in the order they happened.
    public var unseenUnlocks: [LevelState] {
        states.values.filter { $0.isUnlocked && $0.seenAt == nil }
            .sorted { ($0.firstAchievedAt ?? .distantPast) < ($1.firstAchievedAt ?? .distantPast) }
    }
}

/// A line on the map from one skill to one that builds on it.
public struct SkillLink: Sendable, Hashable {
    public var from: String
    public var to: String
    /// The earlier skill's level on this line is unlocked.
    public var lit: Bool
    /// ... and that unlock has not been shown yet: the map animates it once.
    public var newlyLit: Bool
}

/// The server's skill map, mapped to local rows.
public struct SkillMapSnapshot: Sendable {
    public var contentVersion: String
    public var skills: [SkillWithLevels]
    public var edges: [SkillEdge]
    public var states: [LevelState]

    public init(contentVersion: String, skills: [SkillWithLevels], edges: [SkillEdge], states: [LevelState]) {
        self.contentVersion = contentVersion; self.skills = skills; self.edges = edges; self.states = states
    }
}

extension AppDatabase {
    static func migrateSkillMap(_ db: Database) throws {
        try db.create(table: "skill") { t in
            t.primaryKey("id", .text)
            t.column("slug", .text).notNull().unique()
            t.column("name", .text).notNull()
            t.column("family", .text).notNull()
            t.column("difficultyTier", .integer).notNull()
            t.column("isMilestone", .boolean).notNull()
            t.column("status", .text).notNull()
            t.column("summary", .text).notNull()
            t.column("aka", .jsonText).notNull()
            t.column("primaryMuscles", .jsonText).notNull()
            t.column("commonFaults", .jsonText).notNull()
            t.column("constellation", .text)
            t.column("x", .double)
            t.column("y", .double)
        }
        try db.create(table: "skillLevel") { t in
            t.primaryKey("id", .text)
            t.column("skillId", .text).notNull().indexed()
            t.column("orderIndex", .integer).notNull()
            t.column("slug", .text).notNull()
            t.column("name", .text).notNull()
            t.column("details", .text).notNull()
            t.column("estWeeksFromPrev", .integer)
            t.column("criteria", .jsonText).notNull()
            t.column("exercises", .jsonText).notNull()
        }
        try db.create(table: "skillEdge") { t in
            t.column("fromLevelId", .text).notNull()
            t.column("toLevelId", .text).notNull()
            t.column("relation", .text).notNull()
            t.column("weight", .double).notNull()
            t.primaryKey(["fromLevelId", "toLevelId", "relation"])
        }
        try db.create(table: "levelState") { t in
            t.primaryKey("levelId", .text)
            t.column("state", .text).notNull()
            t.column("bestValue", .double)
            t.column("bestUnit", .text)
            t.column("bestAt", .datetime)
            t.column("firstAchievedAt", .datetime)
            t.column("verification", .text)
            t.column("staleSince", .datetime)
            t.column("attemptsCount", .integer).notNull()
            t.column("seenAt", .datetime)
        }
        try db.create(table: "athleteProgress") { t in
            t.primaryKey("id", .integer)
            t.column("xpTotal", .integer).notNull()
            t.column("currentDays", .integer).notNull()
            t.column("longestDays", .integer).notNull()
            t.column("freezeCredits", .integer).notNull()
            t.column("lastCountedDate", .text)
        }
        try db.create(table: "skillNotes") { t in
            t.primaryKey("skillSlug", .text)
            t.column("injuries", .jsonText).notNull()
            t.column("disclaimer", .text).notNull()
            t.column("fetchedAt", .datetime).notNull()
        }
        try db.alter(table: "syncState") { t in
            t.add(column: "skillContentVersion", .text)
        }
    }

    // MARK: writes

    /// Takes the server's skill map. Content is replaced when its version
    /// changed; level states always. A level unlocked here stays unlocked,
    /// and keeps when it was first shown. On the first refresh every unlock
    /// counts as already seen, so the map does not replay the athlete's
    /// whole history.
    public func replaceSkillMap(_ snapshot: SkillMapSnapshot, now: Date = Date()) throws {
        try writer.write { db in
            let current = try String.fetchOne(db, sql: "SELECT skillContentVersion FROM syncState WHERE id = 1")
            if current != snapshot.contentVersion {
                _ = try SkillEdge.deleteAll(db)
                _ = try SkillLevel.deleteAll(db)
                _ = try Skill.deleteAll(db)
                for s in snapshot.skills {
                    try s.skill.insert(db)
                    for l in s.levels { try l.insert(db) }
                }
                for e in snapshot.edges { try e.insert(db) }
                try db.execute(
                    sql: "UPDATE syncState SET skillContentVersion = ? WHERE id = 1",
                    arguments: [snapshot.contentVersion])
            }

            let existing = Dictionary(uniqueKeysWithValues: try LevelState.fetchAll(db).map { ($0.levelId, $0) })
            let firstRefresh = existing.isEmpty
            for var incoming in snapshot.states {
                let old = existing[incoming.levelId]
                if let old, old.isUnlocked, !incoming.isUnlocked {
                    // Unlocks are never revoked: keep the achievement, take the rest.
                    incoming.state = old.state
                    incoming.firstAchievedAt = incoming.firstAchievedAt ?? old.firstAchievedAt
                    incoming.verification = incoming.verification ?? old.verification
                }
                incoming.seenAt = old?.seenAt ?? (firstRefresh && incoming.isUnlocked ? now : nil)
                try incoming.save(db)
            }
        }
    }

    /// Records that the map has shown these unlocks.
    public func markUnlocksSeen(_ levelIds: [String], at now: Date = Date()) throws {
        guard !levelIds.isEmpty else { return }
        try writer.write { db in
            try LevelState.filter(levelIds.contains(Column("levelId")) && Column("seenAt") == nil)
                .updateAll(db, Column("seenAt").set(to: now))
        }
    }

    public func saveProgress(_ progress: AthleteProgress) throws {
        try writer.write { db in try progress.save(db) }
    }

    public func saveSkillNotes(_ notes: SkillNotes) throws {
        try writer.write { db in try notes.save(db) }
    }

    // MARK: reads

    public static func skillMap(_ db: Database) throws -> SkillMapData {
        let skills = try Skill.order(Column("name")).fetchAll(db)
        let levels = Dictionary(grouping: try SkillLevel.order(Column("orderIndex")).fetchAll(db), by: \.skillId)
        return SkillMapData(
            contentVersion: try String.fetchOne(db, sql: "SELECT skillContentVersion FROM syncState WHERE id = 1"),
            skills: skills.map { SkillWithLevels(skill: $0, levels: levels[$0.id] ?? []) },
            edges: try SkillEdge.fetchAll(db),
            states: Dictionary(uniqueKeysWithValues: try LevelState.fetchAll(db).map { ($0.levelId, $0) }))
    }

    public static func progress(_ db: Database) throws -> AthleteProgress? {
        try AthleteProgress.fetchOne(db, key: 1)
    }

    public static func skillNotes(_ db: Database, slug: String) throws -> SkillNotes? {
        try SkillNotes.fetchOne(db, key: slug)
    }

    public func observeSkillMap() -> AsyncThrowingStream<SkillMapData, any Error> {
        observe { try Self.skillMap($0) }
    }

    public func observeProgress() -> AsyncThrowingStream<AthleteProgress?, any Error> {
        observe { try Self.progress($0) }
    }

    public func observeSkillNotes(slug: String) -> AsyncThrowingStream<SkillNotes?, any Error> {
        observe { try Self.skillNotes($0, slug: slug) }
    }
}
