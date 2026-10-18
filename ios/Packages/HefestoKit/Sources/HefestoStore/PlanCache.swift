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
