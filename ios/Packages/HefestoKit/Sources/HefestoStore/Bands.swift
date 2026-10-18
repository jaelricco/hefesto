import Foundation
import GRDB

/// A resistance band a set can be assisted with (ADR 0024): one of the
/// athlete's own or one from the global catalogue. Bands have no sync
/// columns (ADR 0009); the app keeps the server's list and creates bands
/// online.
public struct Band: Codable, Sendable, Hashable, Identifiable, FetchableRecord, PersistableRecord {
    public static let databaseTableName = "band"
    public var id: String
    /// `mine` or `catalogue`.
    public var owner: String
    public var brand: String
    public var colourLabel: String
    /// What the manufacturer states; real assistance varies (PAR-C-61).
    public var resistanceMinKg: Double
    public var resistanceMaxKg: Double
    public var lengthCm: Double?
    public var thicknessMm: Double?
    /// The server no longer lists it. Sets logged with it still name it, so
    /// it is kept, but it is not offered any more.
    public var removed: Bool

    public init(
        id: String = UUIDv7.make(), owner: String = "mine", brand: String, colourLabel: String,
        resistanceMinKg: Double, resistanceMaxKg: Double, lengthCm: Double? = nil, thicknessMm: Double? = nil,
        removed: Bool = false
    ) {
        self.id = id; self.owner = owner; self.brand = brand; self.colourLabel = colourLabel
        self.resistanceMinKg = resistanceMinKg; self.resistanceMaxKg = resistanceMaxKg
        self.lengthCm = lengthCm; self.thicknessMm = thicknessMm; self.removed = removed
    }

    public var isMine: Bool { owner == "mine" }
}

extension AppDatabase {
    static func migrateBands(_ db: Database) throws {
        try db.create(table: "band") { t in
            t.primaryKey("id", .text)
            t.column("owner", .text).notNull()
            t.column("brand", .text).notNull()
            t.column("colourLabel", .text).notNull()
            t.column("resistanceMinKg", .double).notNull()
            t.column("resistanceMaxKg", .double).notNull()
            t.column("lengthCm", .double)
            t.column("thicknessMm", .double)
            t.column("removed", .boolean).notNull().defaults(to: false)
        }
    }

    /// Keeps the server's list of bands. A band it no longer lists stays
    /// for the sets that name it, marked removed.
    public func replaceBands(_ bands: [Band]) throws {
        try writer.write { db in
            let listed = Set(bands.map(\.id))
            for var band in try Band.fetchAll(db) where !listed.contains(band.id) && !band.removed {
                band.removed = true
                try band.update(db)
            }
            for band in bands { try band.save(db) }
        }
    }

    /// Keeps a band the server just created.
    public func saveBand(_ band: Band) throws {
        try writer.write { db in try band.save(db) }
    }

    /// Marks a band removed: it is no longer offered, but sets keep its name.
    public func markBandRemoved(id: String) throws {
        try writer.write { db in
            guard var band = try Band.fetchOne(db, key: id) else { return }
            band.removed = true
            try band.update(db)
        }
    }

    /// The bands to choose from, the athlete's own first, then by brand and
    /// colour.
    static func offeredBands(_ db: Database) throws -> [Band] {
        try Band
            .filter(Column("removed") == false)
            .order(Column("owner").desc, Column("brand").collating(.nocase), Column("colourLabel").collating(.nocase))
            .fetchAll(db)
    }

    public func bands() throws -> [Band] {
        try reader.read { try Self.offeredBands($0) }
    }

    public func observeBands() -> AsyncThrowingStream<[Band], any Error> {
        observe { try Self.offeredBands($0) }
    }

    /// Every band kept here, removed ones included, by id: for naming the
    /// band of a logged set.
    public func bandsById() throws -> [String: Band] {
        try reader.read { db in
            Dictionary(uniqueKeysWithValues: try Band.fetchAll(db).map { ($0.id, $0) })
        }
    }
}
