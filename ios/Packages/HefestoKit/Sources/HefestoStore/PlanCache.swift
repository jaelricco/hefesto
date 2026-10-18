import Foundation
import GRDB

/// A week's training plan as the server sent it (ADR 0021). The plan is the
/// server's read model: it is replaced whole and never edited here, so it is
/// kept as the document it came as, not spread over tables.
public struct CachedPlan: Codable, Sendable, Hashable, FetchableRecord, PersistableRecord {
    public static let databaseTableName = "cachedPlan"
    /// The Monday the plan's week starts on, YYYY-MM-DD.
    public var weekStart: String
    /// The server's `ETag` for the plan, sent back as `If-None-Match`.
    public var etag: String?
    /// The plan's JSON, as the generated client encodes it.
    public var payload: Data
    public var fetchedAt: Date

    public init(weekStart: String, etag: String?, payload: Data, fetchedAt: Date) {
        self.weekStart = weekStart; self.etag = etag; self.payload = payload; self.fetchedAt = fetchedAt
    }
}

/// A log session kept here that was started from a planned session: by the
/// server's start, or here without the server (spec §10.5). The kept plan
/// may not know it yet.
public struct PlannedStart: Sendable, Hashable {
    public var sessionId: String
    public var plannedSessionId: String
    public var completed: Bool

    public init(sessionId: String, plannedSessionId: String, completed: Bool) {
        self.sessionId = sessionId; self.plannedSessionId = plannedSessionId; self.completed = completed
    }
}

/// A week's kept plan with the starts kept here for its days.
public struct KeptPlan: Sendable, Hashable {
    public var plan: CachedPlan?
    public var starts: [PlannedStart]

    public init(plan: CachedPlan?, starts: [PlannedStart]) {
        self.plan = plan; self.starts = starts
    }
}

extension AppDatabase {
    /// Plans this many days older than the newest kept one are dropped.
    static let planRetentionDays = 28

    static func migratePlanCache(_ db: Database) throws {
        try db.create(table: "cachedPlan") { t in
            t.primaryKey("weekStart", .text)
            t.column("etag", .text)
            t.column("payload", .blob).notNull()
            t.column("fetchedAt", .datetime).notNull()
        }
    }

    /// Keeps a week's plan in place of the one before, and drops plans more
    /// than four weeks older than it.
    public func savePlan(_ plan: CachedPlan) throws {
        try writer.write { db in
            try plan.save(db)
            if let oldest = Self.day(plan.weekStart, plus: -Self.planRetentionDays) {
                try CachedPlan.filter(Column("weekStart") < oldest).deleteAll(db)
            }
        }
    }

    /// Forgets every plan: the server has none for this athlete, as before
    /// the onboarding.
    public func clearPlans() throws {
        _ = try writer.write { db in try CachedPlan.deleteAll(db) }
    }

    /// The plan of the week that holds `day` (YYYY-MM-DD), if one is kept.
    public static func plan(_ db: Database, on day: String) throws -> CachedPlan? {
        guard let weekEarliest = Self.day(day, plus: -6) else { return nil }
        return try CachedPlan
            .filter(Column("weekStart") <= day && Column("weekStart") >= weekEarliest)
            .order(Column("weekStart").desc)
            .fetchOne(db)
    }

    public func plan(on day: String) throws -> CachedPlan? {
        try reader.read { try Self.plan($0, on: day) }
    }

    /// Whether a session with this id is kept here at all, deleted or not.
    public func hasSession(id: String) throws -> Bool {
        try reader.read { try Session.exists($0, key: id) }
    }

    public func observePlan(on day: String) -> AsyncThrowingStream<CachedPlan?, any Error> {
        observe { try Self.plan($0, on: day) }
    }

    /// The live sessions kept here that were started from a planned session
    /// on a day of the week starting `weekStart`. As on the server, only a
    /// deleted session no longer counts as a start.
    static func plannedStarts(_ db: Database, weekStart: String) throws -> [PlannedStart] {
        guard let last = Self.day(weekStart, plus: 6) else { return [] }
        return try Row.fetchAll(db, sql: """
            SELECT id, plannedSessionId, status FROM session
            WHERE plannedSessionId IS NOT NULL AND deletedAt IS NULL AND localDate BETWEEN ? AND ?
            ORDER BY startedAt
            """, arguments: [weekStart, last]).map {
            PlannedStart(
                sessionId: $0["id"], plannedSessionId: $0["plannedSessionId"],
                completed: ($0["status"] as String) == "completed")
        }
    }

    static func keptPlan(_ db: Database, on day: String) throws -> KeptPlan {
        let plan = try Self.plan(db, on: day)
        return KeptPlan(plan: plan, starts: try plan.map { try Self.plannedStarts(db, weekStart: $0.weekStart) } ?? [])
    }

    /// The plan of the week that holds `day`, with the starts kept here.
    public func keptPlan(on day: String) throws -> KeptPlan {
        try reader.read { try Self.keptPlan($0, on: day) }
    }

    /// Follows the kept plan and its starts: a start or a completion here
    /// shows before the server's plan does.
    public func observeKeptPlan(on day: String) -> AsyncThrowingStream<KeptPlan, any Error> {
        observe { try Self.keptPlan($0, on: day) }
    }

    /// `day` moved by `days`, both YYYY-MM-DD.
    static func day(_ day: String, plus days: Int) -> String? {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(identifier: "UTC") ?? .gmt
        guard let date = Day.date(day, calendar: calendar),
              let moved = calendar.date(byAdding: .day, value: days, to: date)
        else { return nil }
        return Day.string(moved, calendar: calendar)
    }
}
