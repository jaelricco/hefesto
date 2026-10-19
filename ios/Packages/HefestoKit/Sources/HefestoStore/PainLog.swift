import Foundation
import GRDB

/// A pain report the athlete gave (spec §8.6), kept on the phone (ADR 0030).
/// It goes to the server at once; one the server could not take yet waits
/// here for the next sync. The kept reports also say which questions about
/// a session are answered.
public struct KeptPainReport: Codable, Sendable, Hashable, Identifiable, FetchableRecord, PersistableRecord {
    public static let databaseTableName = "painReport"
    /// The report's id, also its idempotency key on the server.
    public var id: String
    public var region: String
    /// before_session, warmup, during, after, next_morning or daily.
    public var timepoint: String
    /// 0 to 10.
    public var nrs: Double
    public var at: Date
    public var lastedOver1h: Bool?
    public var persistedOver15min: Bool?
    public var suddenSharp: Bool?
    /// The log session it belongs to, if any.
    public var sessionId: String?
    /// The server has it.
    public var sent: Bool

    public init(
        id: String = UUIDv7.make(), region: String, timepoint: String, nrs: Double, at: Date,
        lastedOver1h: Bool? = nil, persistedOver15min: Bool? = nil, suddenSharp: Bool? = nil,
        sessionId: String? = nil, sent: Bool = false
    ) {
        self.id = id; self.region = region; self.timepoint = timepoint; self.nrs = nrs; self.at = at
        self.lastedOver1h = lastedOver1h; self.persistedOver15min = persistedOver15min
        self.suddenSharp = suddenSharp; self.sessionId = sessionId; self.sent = sent
    }
}

extension AppDatabase {
    static func migratePainLog(_ db: Database) throws {
        try db.create(table: "painReport") { t in
            t.primaryKey("id", .text)
            t.column("region", .text).notNull()
            t.column("timepoint", .text).notNull()
            t.column("nrs", .double).notNull()
            t.column("at", .datetime).notNull()
            t.column("lastedOver1h", .boolean)
            t.column("persistedOver15min", .boolean)
            t.column("suddenSharp", .boolean)
            t.column("sessionId", .text)
            t.column("sent", .boolean).notNull().defaults(to: false)
        }
        try db.create(index: "painReport_session", on: "painReport", columns: ["sessionId", "timepoint"])
    }

    /// Keeps reports before they are sent, so none is lost without a
    /// connection.
    public func keepPainReports(_ reports: [KeptPainReport]) throws {
        try writer.write { db in
            for report in reports { try report.insert(db) }
        }
    }

    /// The reports the server does not have yet, oldest first.
    public func unsentPainReports() throws -> [KeptPainReport] {
        try reader.read { db in
            try KeptPainReport.filter(Column("sent") == false).order(Column("at"), Column("id")).fetchAll(db)
        }
    }

    public func markPainReportSent(id: String) throws {
        try writer.write { db in
            try db.execute(sql: "UPDATE painReport SET sent = 1 WHERE id = ?", arguments: [id])
        }
    }

    /// Forgets a report the server refused for good, as without consent.
    public func dropPainReport(id: String) throws {
        try writer.write { db in _ = try KeptPainReport.deleteOne(db, key: id) }
    }

    /// The reports given about a log session, sent or not.
    public func painReports(sessionId: String) throws -> [KeptPainReport] {
        try reader.read { db in
            try KeptPainReport.filter(Column("sessionId") == sessionId).order(Column("at")).fetchAll(db)
        }
    }

    /// The completed log sessions of a day that were started from a plan,
    /// the last completed first.
    public func completedPlannedSessions(on day: String) throws -> [Session] {
        try reader.read { db in
            try Session.filter(Column("localDate") == day && Column("status") == "completed"
                && Column("plannedSessionId") != nil && Column("deletedAt") == nil)
                .order(Column("completedAt").desc).fetchAll(db)
        }
    }

    /// A live log session.
    public func session(id: String) throws -> Session? {
        try reader.read { db in
            guard let s = try Session.fetchOne(db, key: id), s.deletedAt == nil else { return nil }
            return s
        }
    }
}
