import Foundation
import GRDB

/// The app's local database. The UI reads only from here (ADR 0010); every
/// write to the athlete's rows also queues an outbox op, in one transaction.
public final class AppDatabase: Sendable {
    public let writer: any DatabaseWriter

    public init(_ writer: any DatabaseWriter) throws {
        self.writer = writer
        try Self.migrator.migrate(writer)
    }

    /// An empty in-memory database, for tests and previews.
    public static func inMemory() throws -> AppDatabase {
        try AppDatabase(DatabaseQueue())
    }

    /// The database file at `url`, created if missing.
    public static func onDisk(at url: URL) throws -> AppDatabase {
        try FileManager.default.createDirectory(
            at: url.deletingLastPathComponent(), withIntermediateDirectories: true)
        return try AppDatabase(DatabasePool(path: url.path))
    }

    public var reader: any DatabaseReader { writer }

    static var migrator: DatabaseMigrator {
        var m = DatabaseMigrator()
        #if DEBUG
        m.eraseDatabaseOnSchemaChange = true
        #endif

        m.registerMigration("v1") { db in
            try db.create(table: "session") { t in
                t.primaryKey("id", .text)
                t.column("startedAt", .datetime).notNull()
                t.column("endedAt", .datetime)
                t.column("timezone", .text).notNull()
                t.column("localDate", .text).notNull().indexed()
                t.column("title", .text).notNull()
                t.column("notes", .text).notNull()
                t.column("perceivedFatigue", .integer)
                t.column("bodyweightKg", .double)
                t.column("status", .text).notNull()
                t.column("isRestDay", .boolean).notNull()
                t.column("templateId", .text)
                t.column("completedAt", .datetime)
                t.column("updatedAt", .datetime).notNull()
                t.column("serverSeq", .integer)
                t.column("deletedAt", .datetime)
            }
            try db.create(table: "block") { t in
                t.primaryKey("id", .text)
                t.column("sessionId", .text).notNull().indexed()
                t.column("orderIndex", .integer).notNull()
                t.column("kind", .text).notNull()
                t.column("roundsPlanned", .integer)
                t.column("roundsDone", .integer)
                t.column("intervalS", .integer)
                t.column("notes", .text).notNull()
                t.column("updatedAt", .datetime).notNull()
                t.column("serverSeq", .integer)
                t.column("deletedAt", .datetime)
            }
            try db.create(table: "setEntry") { t in
                t.primaryKey("id", .text)
                t.column("sessionId", .text).notNull().indexed()
                t.column("blockId", .text).notNull().indexed()
                t.column("orderIndex", .integer).notNull()
                t.column("roundIndex", .integer)
                t.column("kind", .text).notNull()
                t.column("isPlanned", .boolean).notNull()
                t.column("restAfterPlannedS", .integer)
                t.column("restAfterActualS", .integer)
                t.column("rpe", .double)
                t.column("rir", .integer)
                t.column("completedAt", .datetime)
                t.column("notes", .text).notNull()
                t.column("updatedAt", .datetime).notNull()
                t.column("serverSeq", .integer)
                t.column("deletedAt", .datetime)
            }
            try db.create(table: "setElement") { t in
                t.primaryKey("id", .text)
                t.column("setEntryId", .text).notNull().indexed()
                t.column("orderIndex", .integer).notNull()
                t.column("exerciseId", .text).notNull().indexed()
                t.column("measure", .text).notNull()
                t.column("reps", .integer)
                t.column("holdSeconds", .double)
                t.column("distanceM", .double)
                t.column("tempo", .text)
                t.column("loadKg", .double).notNull()
                t.column("isEccentricOnly", .boolean).notNull()
                t.column("isPartialRom", .boolean).notNull()
                t.column("romNote", .text)
                t.column("formQuality", .integer)
                t.column("failed", .boolean).notNull()
                t.column("assistanceClass", .text).notNull()
                t.column("assistance", .jsonText)
                t.column("mediaIds", .jsonText).notNull()
            }
            try db.create(table: "bodyweight") { t in
                t.primaryKey("id", .text)
                t.column("measuredAt", .datetime).notNull()
                t.column("timezone", .text).notNull()
                t.column("localDate", .text).notNull()
                t.column("bodyweightKg", .double).notNull()
                t.column("note", .text).notNull()
                t.column("updatedAt", .datetime).notNull()
                t.column("serverSeq", .integer)
                t.column("deletedAt", .datetime)
            }
            try db.create(table: "exercise") { t in
                t.primaryKey("id", .text)
                t.column("slug", .text).notNull().unique()
                t.column("name", .text).notNull()
                t.column("family", .text).notNull()
                t.column("defaultMeasure", .text).notNull()
                t.column("isBodyweight", .boolean).notNull()
            }

            // The outbox. One queued op per row and kind; `batchKey` is set
            // while the op is in a batch sent to the server.
            try db.create(table: "outboxOp") { t in
                t.autoIncrementedPrimaryKey("seq")
                t.column("entity", .text).notNull()
                t.column("op", .text).notNull()
                t.column("rowId", .text).notNull()
                t.column("sessionId", .text)
                t.column("createdAt", .datetime).notNull()
                t.column("batchKey", .text).indexed()
                t.column("payload", .blob)
            }
            try db.execute(sql: """
                CREATE UNIQUE INDEX outboxOp_queued ON outboxOp(entity, rowId, op = 'complete')
                WHERE batchKey IS NULL
                """)

            // Sync bookkeeping: the pull cursor and the batch in flight.
            try db.create(table: "syncState") { t in
                t.primaryKey("id", .integer)
                t.column("cursor", .integer).notNull()
                t.column("batchKey", .text)
                t.column("contentVersion", .text)
            }
            try db.execute(sql: "INSERT INTO syncState (id, cursor) VALUES (1, 0)")

            // Ops the server rejected, kept for the UI to explain.
            try db.create(table: "syncProblem") { t in
                t.autoIncrementedPrimaryKey("id")
                t.column("entity", .text).notNull()
                t.column("op", .text).notNull()
                t.column("rowId", .text).notNull()
                t.column("type", .text).notNull()
                t.column("detail", .text).notNull()
                t.column("createdAt", .datetime).notNull()
            }
        }
        return m
    }
}
