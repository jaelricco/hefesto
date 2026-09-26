import Foundation
import GRDB

// Local writes. Each changes the athlete's rows and queues its op in the same
// transaction, so the database and the outbox never disagree (ADR 0010).
// Writes stamp `updatedAt`, the athlete's clock that decides between two
// copies of a set (ADR 0009).

extension AppDatabase {
    /// Creates or updates a session.
    public func saveSession(_ session: Session, now: Date = Date()) throws {
        try writer.write { db in
            var s = session
            s.updatedAt = now
            try s.save(db)
            try Self.enqueue(db, .session, .put, rowId: s.id, sessionId: nil)
        }
    }

    /// Completes a draft session. The server evaluates unlocks when the op
    /// arrives; the sync engine reports them back.
    public func completeSession(
        id: String, at completedAt: Date = Date(), perceivedFatigue: Int? = nil, now: Date = Date()
    ) throws {
        try writer.write { db in
            guard var s = try Session.fetchOne(db, key: id), s.deletedAt == nil else {
                throw StoreError.notFound
            }
            guard s.status == "draft" else { return } // completion is final, and idempotent
            s.status = "completed"
            s.completedAt = completedAt
            s.endedAt = s.endedAt ?? completedAt
            if let f = perceivedFatigue { s.perceivedFatigue = f }
            s.updatedAt = now
            try s.update(db)
            try Self.enqueue(db, .session, .complete, rowId: id, sessionId: nil)
        }
    }

    /// Deletes a session and everything in it. Deletion is final.
    public func deleteSession(id: String, now: Date = Date()) throws {
        try writer.write { db in
            try db.execute(sql: "UPDATE session SET deletedAt = ?, updatedAt = ? WHERE id = ?", arguments: [now, now, id])
            try db.execute(sql: "UPDATE block SET deletedAt = ? WHERE sessionId = ? AND deletedAt IS NULL", arguments: [now, id])
            try db.execute(sql: "UPDATE setEntry SET deletedAt = ? WHERE sessionId = ? AND deletedAt IS NULL", arguments: [now, id])
            try Self.enqueue(db, .session, .delete, rowId: id, sessionId: nil)
        }
    }

    public func saveBlock(_ block: Block, now: Date = Date()) throws {
        try writer.write { db in
            var b = block
            b.updatedAt = now
            try b.save(db)
            try Self.enqueue(db, .block, .put, rowId: b.id, sessionId: b.sessionId)
        }
    }

    public func deleteBlock(id: String, sessionId: String, now: Date = Date()) throws {
        try writer.write { db in
            try db.execute(sql: "UPDATE block SET deletedAt = ?, updatedAt = ? WHERE id = ?", arguments: [now, now, id])
            try db.execute(sql: "UPDATE setEntry SET deletedAt = ? WHERE blockId = ? AND deletedAt IS NULL", arguments: [now, id])
            try Self.enqueue(db, .block, .delete, rowId: id, sessionId: sessionId)
        }
    }

    /// Writes a set with its complete, ordered list of elements: one for a
    /// plain set, several for a combo. Elements no longer listed are removed.
    public func saveSet(_ set: SetWithElements, now: Date = Date()) throws {
        guard !set.elements.isEmpty else { throw StoreError.emptySet }
        try writer.write { db in
            var entry = set.entry
            entry.updatedAt = now
            try entry.save(db)
            let keep = set.elements.map(\.id)
            try SetElement
                .filter(Column("setEntryId") == entry.id && !keep.contains(Column("id")))
                .deleteAll(db)
            for (i, el) in set.elements.enumerated() {
                var e = el
                e.setEntryId = entry.id
                e.orderIndex = i
                try e.save(db)
            }
            try Self.enqueue(db, .set, .put, rowId: entry.id, sessionId: entry.sessionId)
        }
    }

    public func deleteSet(id: String, sessionId: String, now: Date = Date()) throws {
        try writer.write { db in
            try db.execute(sql: "UPDATE setEntry SET deletedAt = ?, updatedAt = ? WHERE id = ?", arguments: [now, now, id])
            try Self.enqueue(db, .set, .delete, rowId: id, sessionId: sessionId)
        }
    }

    public func saveBodyweight(_ entry: Bodyweight, now: Date = Date()) throws {
        try writer.write { db in
            var b = entry
            b.updatedAt = now
            try b.save(db)
            try Self.enqueue(db, .bodyweight, .put, rowId: b.id, sessionId: nil)
        }
    }

    public func deleteBodyweight(id: String, now: Date = Date()) throws {
        try writer.write { db in
            try db.execute(sql: "UPDATE bodyweight SET deletedAt = ?, updatedAt = ? WHERE id = ?", arguments: [now, now, id])
            try Self.enqueue(db, .bodyweight, .delete, rowId: id, sessionId: nil)
        }
    }

    /// Replaces the cached exercise catalogue.
    public func replaceExercises(_ exercises: [Exercise], contentVersion: String) throws {
        try writer.write { db in
            _ = try Exercise.deleteAll(db)
            for e in exercises { try e.insert(db) }
            try db.execute(sql: "UPDATE syncState SET contentVersion = ? WHERE id = 1", arguments: [contentVersion])
        }
    }
}

public enum StoreError: Error, Equatable {
    case notFound
    /// A set has at least one element; a combo has several.
    case emptySet
}
