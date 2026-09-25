import Foundation
import GRDB

/// What an op writes. Matches the `entity` of a sync push op (ADR 0009).
public enum OutboxEntity: String, Codable, Sendable, CaseIterable {
    case session, block, set, bodyweight
}

/// The kind of write. Matches the `op` of a sync push op.
public enum OutboxKind: String, Codable, Sendable, CaseIterable {
    case put, delete, complete
}

/// A queued write. It names the row, not its contents: the payload is read
/// from the row when the op is sent, so the newest local state goes out.
/// `payload` is set once the op joins a batch, and is what a retry resends.
public struct OutboxOp: Codable, Sendable, Hashable, FetchableRecord, MutablePersistableRecord {
    public static let databaseTableName = "outboxOp"
    public var seq: Int64?
    public var entity: OutboxEntity
    public var op: OutboxKind
    public var rowId: String
    public var sessionId: String?
    public var createdAt: Date
    public var batchKey: String?
    public var payload: Data?

    public mutating func didInsert(_ inserted: InsertionSuccess) {
        seq = inserted.rowID
    }
}

/// How the server settled an op of a batch.
public enum OpOutcome: Sendable, Hashable {
    /// Applied; the op is done.
    case applied
    /// The server's copy wins (a deleted row, or an older set). The sync
    /// engine fetches the server's copy, since the feed may be past it.
    case superseded
    /// Invalid as sent; kept as a problem for the UI.
    case rejected(type: String, detail: String)
}

extension AppDatabase {
    /// Queues a write, coalescing with a queued op on the same row: a newer
    /// put replaces a queued put in place, so parents stay ahead of
    /// children, and a delete replaces a queued put. A `complete` is queued
    /// on its own, after the puts it follows.
    static func enqueue(_ db: Database, _ entity: OutboxEntity, _ op: OutboxKind, rowId: String, sessionId: String?) throws {
        let isComplete = op == .complete
        let queued = try OutboxOp
            .filter(Column("entity") == entity.rawValue && Column("rowId") == rowId && Column("batchKey") == nil)
            .fetchAll(db)
            .first { ($0.op == .complete) == isComplete }
        if var existing = queued {
            existing.op = op
            try existing.update(db)
            return
        }
        var new = OutboxOp(
            seq: nil, entity: entity, op: op, rowId: rowId, sessionId: sessionId,
            createdAt: Date(), batchKey: nil, payload: nil)
        try new.insert(db)
    }

    /// The oldest queued ops not yet in a batch.
    public func queuedOps(limit: Int = 200) throws -> [OutboxOp] {
        try reader.read { db in
            try OutboxOp.filter(Column("batchKey") == nil).order(Column("seq")).limit(limit).fetchAll(db)
        }
    }

    /// The batch in flight, if a previous push did not finish: its key and
    /// ops in order, with the payloads that were sent.
    public func batchInFlight() throws -> (key: String, ops: [OutboxOp])? {
        try reader.read { db in
            guard let key = try String.fetchOne(db, sql: "SELECT batchKey FROM syncState WHERE id = 1") else {
                return nil
            }
            let ops = try OutboxOp.filter(Column("batchKey") == key).order(Column("seq")).fetchAll(db)
            return (key, ops)
        }
    }

    /// Freezes ops into a batch under `key`, with the payload each will send.
    /// Until the batch is finished, a retry sends exactly these bytes under
    /// the same key.
    public func startBatch(key: String, ops: [(seq: Int64, payload: Data?)]) throws {
        try writer.write { db in
            for op in ops {
                try db.execute(
                    sql: "UPDATE outboxOp SET batchKey = ?, payload = ? WHERE seq = ?",
                    arguments: [key, op.payload, op.seq])
            }
            try db.execute(sql: "UPDATE syncState SET batchKey = ? WHERE id = 1", arguments: [key])
        }
    }

    /// Settles a batch with an outcome per op, and returns the superseded
    /// ops, whose rows the sync engine must refetch.
    @discardableResult
    public func finishBatch(key: String, outcomes: [Int64: OpOutcome]) throws -> [OutboxOp] {
        try writer.write { db in
            var superseded: [OutboxOp] = []
            let ops = try OutboxOp.filter(Column("batchKey") == key).fetchAll(db)
            for op in ops {
                guard let seq = op.seq else { continue }
                switch outcomes[seq] ?? .applied {
                case .applied:
                    break
                case .superseded:
                    superseded.append(op)
                case let .rejected(type, detail):
                    try db.execute(
                        sql: """
                            INSERT INTO syncProblem (entity, op, rowId, type, detail, createdAt)
                            VALUES (?, ?, ?, ?, ?, ?)
                            """,
                        arguments: [op.entity.rawValue, op.op.rawValue, op.rowId, type, detail, Date()])
                }
                _ = try OutboxOp.deleteOne(db, key: seq)
            }
            try db.execute(sql: "UPDATE syncState SET batchKey = NULL WHERE id = 1")
            return superseded
        }
    }

    /// The pull cursor: everything up to it has been applied.
    public func cursor() throws -> Int64 {
        try reader.read { db in
            try Int64.fetchOne(db, sql: "SELECT cursor FROM syncState WHERE id = 1") ?? 0
        }
    }

    /// Problems the server reported on rejected ops, newest first.
    public func syncProblems() throws -> [(entity: String, rowId: String, type: String, detail: String)] {
        try reader.read { db in
            try Row.fetchAll(db, sql: "SELECT * FROM syncProblem ORDER BY id DESC").map {
                ($0["entity"], $0["rowId"], $0["type"], $0["detail"])
            }
        }
    }
}
