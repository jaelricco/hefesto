import Foundation
import HefestoAPI
import HefestoStore

/// Why a band could not be added or removed, in terms the UI can explain.
public enum BandError: Error, Equatable {
    /// The athlete has a band of this brand and colour already.
    case duplicate
    /// The server cannot take some fields: messages by JSON pointer.
    case invalid([String: String])
    case refused(status: Int)
}

extension SyncEngine {
    /// Fetches the bands the athlete can use, their own and the catalogue's,
    /// and keeps them here (ADR 0024). Bands are not in the sync feed
    /// (ADR 0009), so this follows every sync.
    public func refreshBands() async throws {
        switch try await client.listBands() {
        case let .ok(r): try db.replaceBands(try r.body.json.items.map(Band.init))
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw SyncError.refused(status: status, type: "")
        }
    }

    /// Adds a band to the athlete's collection and keeps it. Needs the
    /// network: bands are created online.
    @discardableResult
    public func createBand(
        brand: String, colourLabel: String, resistanceMinKg: Double, resistanceMaxKg: Double,
        id: String = UUIDv7.make()
    ) async throws -> Band {
        let body = Components.Schemas.BandCreate(
            id: id, brand: brand, colourLabel: colourLabel, resistanceMinKg: resistanceMinKg,
            resistanceMaxKg: resistanceMaxKg)
        switch try await client.createMyBand(body: .json(body)) {
        case let .created(r):
            let band = Band(try r.body.json)
            try db.saveBand(band)
            return band
        case .conflict: throw BandError.duplicate
        case let .unprocessableContent(r):
            throw BandError.invalid((try? r.body.applicationProblemJson)?.errors?.additionalProperties ?? [:])
        case .badRequest: throw BandError.refused(status: 400)
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw BandError.refused(status: status)
        }
    }

    /// Removes one of the athlete's bands. The server deletes it softly, so
    /// sets logged with it keep it; here it stays for their names.
    public func deleteBand(id: String) async throws {
        switch try await client.deleteMyBand(path: .init(bandId: id)) {
        case .noContent, .notFound: try db.markBandRemoved(id: id)
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw BandError.refused(status: status)
        }
    }
}

extension Band {
    init(_ b: Components.Schemas.Band) {
        self.init(
            id: b.id, owner: b.owner.rawValue, brand: b.brand, colourLabel: b.colourLabel,
            resistanceMinKg: b.resistanceMinKg, resistanceMaxKg: b.resistanceMaxKg, lengthCm: b.lengthCm,
            thicknessMm: b.thicknessMm)
    }
}
