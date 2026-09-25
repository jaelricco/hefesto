import Foundation
import GRDB

// Reads for the UI. Views observe these through GRDB's ValueObservation, so
// they update as the logger and the sync engine write.

extension AppDatabase {
    /// Live sessions, newest first.
    public static func recentSessions(_ db: Database, limit: Int = 50) throws -> [Session] {
        try Session.filter(Column("deletedAt") == nil)
            .order(Column("startedAt").desc).limit(limit).fetchAll(db)
    }

    /// A session with its live blocks and sets in order, each set with its
    /// ordered elements.
    public static func sessionTree(_ db: Database, id: String) throws -> SessionTree? {
        guard let session = try Session.fetchOne(db, key: id), session.deletedAt == nil else { return nil }
        let blocks = try Block.filter(Column("sessionId") == id && Column("deletedAt") == nil)
            .order(Column("orderIndex")).fetchAll(db)
        let entries = try SetEntry.filter(Column("sessionId") == id && Column("deletedAt") == nil)
            .order(Column("orderIndex")).fetchAll(db)
        let elements = try SetElement.filter(entries.map(\.id).contains(Column("setEntryId")))
            .order(Column("orderIndex")).fetchAll(db)
        let bySet = Dictionary(grouping: elements, by: \.setEntryId)
        let byBlock = Dictionary(grouping: entries, by: \.blockId)
        return SessionTree(
            session: session,
            blocks: blocks.map { b in
                BlockWithSets(block: b, sets: (byBlock[b.id] ?? []).map {
                    SetWithElements(entry: $0, elements: bySet[$0.id] ?? [])
                })
            })
    }

    /// The athlete's most recent performed set containing `exerciseId`, for
    /// "repeat last set".
    public static func lastSet(_ db: Database, exerciseId: String) throws -> SetWithElements? {
        let sql = """
            SELECT se.* FROM setEntry se
            JOIN setElement el ON el.setEntryId = se.id
            WHERE el.exerciseId = ? AND se.deletedAt IS NULL AND NOT se.isPlanned
            ORDER BY COALESCE(se.completedAt, se.updatedAt) DESC
            LIMIT 1
            """
        guard let entry = try SetEntry.fetchOne(db, sql: sql, arguments: [exerciseId]) else { return nil }
        let elements = try SetElement.filter(Column("setEntryId") == entry.id).order(Column("orderIndex")).fetchAll(db)
        return SetWithElements(entry: entry, elements: elements)
    }

    public static func exercises(_ db: Database, matching query: String = "") throws -> [Exercise] {
        var request = Exercise.order(Column("name"))
        if !query.isEmpty {
            request = request.filter(Column("name").like("%\(query)%") || Column("slug").like("%\(query)%"))
        }
        return try request.fetchAll(db)
    }

    /// Ops waiting to be sent, for the sync indicator.
    public static func pendingOpCount(_ db: Database) throws -> Int {
        try OutboxOp.fetchCount(db)
    }
}

public struct BlockWithSets: Sendable, Hashable, Identifiable {
    public var block: Block
    public var sets: [SetWithElements]
    public var id: String { block.id }

    public init(block: Block, sets: [SetWithElements]) { self.block = block; self.sets = sets }
}

public struct SessionTree: Sendable, Hashable {
    public var session: Session
    public var blocks: [BlockWithSets]

    public init(session: Session, blocks: [BlockWithSets]) { self.session = session; self.blocks = blocks }
}
