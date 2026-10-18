import Foundation
import Testing
@testable import HefestoAPI
@testable import HefestoStore
@testable import HefestoSync

func bandJSON(_ id: String, owner: String = "mine", brand: String = "Brand A", colour: String) -> String {
    #"{"id":"\#(id)","owner":"\#(owner)","brand":"\#(brand)","colour_label":"\#(colour)","resistance_min_kg":5,"resistance_max_kg":15,"length_cm":null,"thickness_mm":null}"#
}

@Suite struct BandTests {
    @Test func theListIsKeptWithTheAthletesOwnFirst() async throws {
        let db = try AppDatabase.inMemory()
        let mine = UUIDv7.make(), catalogue = UUIDv7.make()
        let server = ScriptedServer { _ in
            (200, #"{"items":[\#(bandJSON(catalogue, owner: "catalogue", brand: "Brand B", colour: "Blue")),\#(bandJSON(mine, colour: "Red"))]}"#)
        }

        try await SyncEngine(client: server.client, db: db).refreshBands()

        let bands = try db.bands()
        #expect(bands.map(\.id) == [mine, catalogue])
        #expect(bands.first?.isMine == true)
        #expect(bands.first?.resistanceMaxKg == 15)
    }

    @Test func aBandNoLongerListedStaysForTheSetsThatNameIt() async throws {
        let db = try AppDatabase.inMemory()
        let red = UUIDv7.make(), blue = UUIDv7.make()
        let lists = Counter()
        let server = ScriptedServer { _ in
            lists.next() == 1
                ? (200, #"{"items":[\#(bandJSON(red, colour: "Red")),\#(bandJSON(blue, colour: "Blue"))]}"#)
                : (200, #"{"items":[\#(bandJSON(blue, colour: "Blue"))]}"#)
        }
        let sync = SyncEngine(client: server.client, db: db)

        try await sync.refreshBands()
        try await sync.refreshBands()

        #expect(try db.bands().map(\.id) == [blue], "no longer offered")
        #expect(try db.bandsById()[red]?.removed == true, "but still named")
    }

    @Test func aNewBandIsCreatedOnlineAndKept() async throws {
        let db = try AppDatabase.inMemory()
        let server = ScriptedServer { r in (201, bandJSON(r.json["id"] as? String ?? "", colour: "Green")) }

        let band = try await SyncEngine(client: server.client, db: db).createBand(
            brand: "Brand A", colourLabel: "Green", resistanceMinKg: 5, resistanceMaxKg: 15)

        let body = try #require(server.requests("createMyBand").first).json
        #expect(body["brand"] as? String == "Brand A")
        #expect(body["colour_label"] as? String == "Green")
        #expect(body["resistance_max_kg"] as? Double == 15)
        #expect(try db.bands().map(\.id) == [band.id])
    }

    @Test func theServerSaysWhyABandWasRefused() async throws {
        let db = try AppDatabase.inMemory()
        let duplicate = SyncEngine(client: ScriptedServer { _ in (409, problemJSON("conflict", status: 409)) }.client, db: db)
        await #expect(throws: BandError.duplicate) {
            try await duplicate.createBand(brand: "A", colourLabel: "Red", resistanceMinKg: 5, resistanceMaxKg: 15)
        }
        let invalid = SyncEngine(client: ScriptedServer { _ in
            (422, #"{"type":"https://hefesto.fit/problems/validation","title":"Invalid","status":422,"errors":{"/resistance_max_kg":"must not be below the minimum"}}"#)
        }.client, db: db)
        await #expect(throws: BandError.invalid(["/resistance_max_kg": "must not be below the minimum"])) {
            try await invalid.createBand(brand: "A", colourLabel: "Red", resistanceMinKg: 15, resistanceMaxKg: 5)
        }
        #expect(try db.bands().isEmpty)
    }

    @Test func aRemovedBandIsNoLongerOffered() async throws {
        let db = try AppDatabase.inMemory()
        let band = Band(brand: "Brand A", colourLabel: "Red", resistanceMinKg: 5, resistanceMaxKg: 15)
        try db.saveBand(band)
        let server = ScriptedServer { _ in (204, "") }

        try await SyncEngine(client: server.client, db: db).deleteBand(id: band.id)

        #expect(server.requests("deleteMyBand").first?.path.hasSuffix(band.id) == true)
        #expect(try db.bands().isEmpty)
        #expect(try db.bandsById()[band.id]?.removed == true)
    }
}
