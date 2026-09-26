import Foundation
import GRDB

/// One page of the server's change feed, already mapped to local rows.
public struct ServerPage: Sendable {
    public var cursor: Int64
    public var sessions: [Session]
    public var blocks: [Block]
    public var sets: [SetWithElements]
    public var bodyweight: [Bodyweight]

    public init(cursor: Int64, sessions: [Session] = [], blocks: [Block] = [],
                sets: [SetWithElements] = [], bodyweight: [Bodyweight] = []) {
        self.cursor = cursor; self.sessions = sessions; self.blocks = blocks
        self.sets = sets; self.bodyweight = bodyweight
    }
}

extension AppDatabase {
    /// Applies a page and advances the cursor, in one transaction, parents
    /// first. A row with a queued or in-flight op is left alone: the local
    /// edit is newer and still on its way (ADR 0010).
    public func apply(_ page: ServerPage) throws {
        try writer.write { db in
            let pending = try Self.pendingRows(db)
            for s in page.sessions where !pending.contains(.init(.session, s.id)) { try s.save(db) }
            for b in page.blocks where !pending.contains(.init(.block, b.id)) { try b.save(db) }
            for set in page.sets where !pending.contains(.init(.set, set.entry.id)) {
                try Self.replaceSet(db, set)
            }
            for b in page.bodyweight where !pending.contains(.init(.bodyweight, b.id)) { try b.save(db) }
            try db.execute(sql: "UPDATE syncState SET cursor = MAX(cursor, ?) WHERE id = 1", arguments: [page.cursor])
        }
    }

    /// Replaces a session's local tree with the server's copy, after the
    /// server superseded a local write to it. `nil` means the server has no
    /// live session: it was deleted, so it is deleted here too.
    public func replaceSessionTree(id: String, with tree: SessionTree?, now: Date = Date()) throws {
        try writer.write { db in
            guard let tree else {
                try db.execute(sql: "UPDATE session SET deletedAt = COALESCE(deletedAt, ?) WHERE id = ?", arguments: [now, id])
                try db.execute(sql: "UPDATE block SET deletedAt = COALESCE(deletedAt, ?) WHERE sessionId = ?", arguments: [now, id])
                try db.execute(sql: "UPDATE setEntry SET deletedAt = COALESCE(deletedAt, ?) WHERE sessionId = ?", arguments: [now, id])
                return
            }
            try tree.session.save(db)
            let liveBlocks = Set(tree.blocks.map(\.block.id))
            let liveSets = Set(tree.blocks.flatMap { $0.sets.map(\.entry.id) })
            // What the server no longer has is gone.
            for b in try Block.filter(Column("sessionId") == id && Column("deletedAt") == nil).fetchAll(db)
            where !liveBlocks.contains(b.id) {
                try db.execute(sql: "UPDATE block SET deletedAt = ? WHERE id = ?", arguments: [now, b.id])
            }
            for s in try SetEntry.filter(Column("sessionId") == id && Column("deletedAt") == nil).fetchAll(db)
            where !liveSets.contains(s.id) {
                try db.execute(sql: "UPDATE setEntry SET deletedAt = ? WHERE id = ?", arguments: [now, s.id])
            }
            for b in tree.blocks {
                try b.block.save(db)
                for set in b.sets { try Self.replaceSet(db, set) }
            }
        }
    }

    /// Marks a bodyweight entry deleted, after the server superseded a write
    /// to it (the only way it can: the entry was deleted elsewhere).
    public func markBodyweightDeleted(id: String, now: Date = Date()) throws {
        try writer.write { db in
            try db.execute(sql: "UPDATE bodyweight SET deletedAt = COALESCE(deletedAt, ?) WHERE id = ?", arguments: [now, id])
        }
    }

    static func replaceSet(_ db: Database, _ set: SetWithElements) throws {
        try set.entry.save(db)
        let keep = set.elements.map(\.id)
        try SetElement.filter(Column("setEntryId") == set.entry.id && !keep.contains(Column("id"))).deleteAll(db)
        for e in set.elements { try e.save(db) }
    }

    struct RowRef: Hashable {
        let entity: OutboxEntity
        let rowId: String
        init(_ entity: OutboxEntity, _ rowId: String) { self.entity = entity; self.rowId = rowId }
    }

    static func pendingRows(_ db: Database) throws -> Set<RowRef> {
        Set(try OutboxOp.fetchAll(db).map { RowRef($0.entity, $0.rowId) })
    }
}
