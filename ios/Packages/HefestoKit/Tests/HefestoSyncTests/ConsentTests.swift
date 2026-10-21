import Foundation
import Testing
@testable import HefestoAPI
@testable import HefestoPlan
@testable import HefestoStore
@testable import HefestoSync

// The consent to keep health data after the onboarding (ADR 0031): the
// record reads as the screen needs it, every decision names its text, a
// withdrawal forgets the pain reports kept here, and a grant asks the
// screening again.

func consentJSON(granted: Bool, records: [String]) -> String {
    #"{"granted":\#(granted),"records":[\#(records.joined(separator: ","))]}"#
}

func consentRecordJSON(granted: Bool, source: String = "change", text: String? = #"{"version":"1","locale":"de"}"#,
                       at: String = "2026-10-21T08:00:00Z") -> String {
    let text = text.map { #","text":\#($0)"# } ?? ""
    return #"{"granted":\#(granted),"source":"\#(source)","at":"\#(at)"\#(text)}"#
}

let consentTextDE = ConsentText(version: "1", locale: "de")

@MainActor
func consentModel(_ server: ScriptedServer, db: AppDatabase? = nil) throws -> (ConsentModel, AppDatabase) {
    let db = try db ?? AppDatabase.inMemory()
    let sync = SyncEngine(client: server.client, db: db)
    return (ConsentModel(sync: sync, plan: nil, body: nil), db)
}

@Suite @MainActor struct ConsentTests {
    @Test func theRecordReadsNewestFirst() async throws {
        let server = ScriptedServer { r in
            r.operation == "getHealthConsent"
                ? (200, consentJSON(granted: false, records: [
                    consentRecordJSON(granted: false, at: "2026-10-21T08:00:00Z"),
                    consentRecordJSON(granted: true, source: "onboarding", text: nil, at: "2026-09-28T07:00:00Z"),
                ]))
                : (200, catalogueJSON)
        }
        let (model, _) = try consentModel(server)

        await model.load()

        #expect(model.status == .ready && model.granted == false)
        #expect(model.decisions.map(\.granted) == [false, true])
        #expect(model.latest?.text == consentTextDE && !(model.latest?.inOnboarding ?? true))
        #expect(model.decisions.last?.inOnboarding == true && model.decisions.last?.text == nil,
                "a decision from before ADR 0031 names no text")
        #expect(model.catalogue?.screening.count == 6, "a grant asks the screening again")
    }

    @Test func withoutTheOnboardingThereIsNoConsentYet() async throws {
        let server = ScriptedServer { r in
            r.operation == "getHealthConsent"
                ? (409, #"{"type":"https://hefesto.fit/problems/onboarding-required","title":"Onboarding required","status":409}"#)
                : (200, catalogueJSON)
        }
        let (model, _) = try consentModel(server)

        await model.load()

        #expect(model.status == .needsOnboarding && model.granted == nil)
    }

    @Test func aWithdrawalNamesItsTextAndForgetsThePainKeptHere() async throws {
        let server = ScriptedServer { r in
            switch r.operation {
            case "changeHealthConsent": (200, eventJSON([changeJSON("consent_changed", text: "Health data deleted.")]))
            case "getHealthConsent": (200, consentJSON(granted: false, records: [consentRecordJSON(granted: false)]))
            default: (200, catalogueJSON)
            }
        }
        let db = try AppDatabase.inMemory()
        try db.keepPainReports([
            KeptPainReport(region: "elbow_inner", timepoint: "after", nrs: 2, at: Date(), sessionId: "s-1", sent: true),
            KeptPainReport(region: "elbow_inner", timepoint: "next_morning", nrs: 1, at: Date(), sessionId: "s-1"),
        ])
        let (model, _) = try consentModel(server, db: db)

        let outcome = try await model.withdraw(text: consentTextDE)

        let body = try #require(server.requests("changeHealthConsent").first).json
        #expect(body["granted"] as? Bool == false)
        #expect(body["text"] as? [String: String] == ["version": "1", "locale": "de"])
        #expect(body["screening"] == nil && body["past_injuries"] == nil)
        #expect(outcome.notes.map(\.text) == ["Health data deleted."])
        #expect(try db.painReports(sessionId: "s-1").isEmpty, "health data on the phone goes too")
        #expect(try db.pendingOpCount() == 0)
        #expect(model.granted == false && model.status == .ready, "read again after the change")
    }

    @Test func aRefusedWithdrawalKeepsThePainReports() async throws {
        let server = ScriptedServer { r in
            r.operation == "changeHealthConsent"
                ? (503, #"{"type":"https://hefesto.fit/problems/planning-unavailable","title":"Unavailable","status":503}"#)
                : (200, catalogueJSON)
        }
        let db = try AppDatabase.inMemory()
        try db.keepPainReports([KeptPainReport(region: "elbow_inner", timepoint: "after", nrs: 2, at: Date(), sessionId: "s-1")])
        let (model, _) = try consentModel(server, db: db)

        await #expect(throws: BodyError.unavailable) { _ = try await model.withdraw(text: consentTextDE) }

        #expect(try db.painReports(sessionId: "s-1").count == 1)
    }

    @Test func aGrantAsksTheScreeningAndThePastInjuries() async throws {
        let calls = Counter()
        let server = ScriptedServer { r in
            switch r.operation {
            case "changeHealthConsent":
                if calls.next() == 1 { throw LostConnection() }
                return (200, eventJSON([changeJSON("consent_changed")]))
            case "getHealthConsent": return (200, consentJSON(granted: true, records: [consentRecordJSON(granted: true)]))
            default: return (200, catalogueJSON)
            }
        }
        let (model, _) = try consentModel(server)
        await model.load()

        #expect(!model.canGrant)
        await #expect(throws: ConsentModelError.incomplete) { _ = try await model.grant(text: consentTextDE) }
        for (i, q) in try #require(model.catalogue).screening.enumerated() { model.screening[q.id] = i == 1 }
        model.pastInjuries = ["wrist_back_extension", "nowhere"]
        #expect(model.canGrant)

        await #expect(throws: (any Error).self) { _ = try await model.grant(text: consentTextDE) }
        _ = try await model.grant(text: consentTextDE)

        let sent = server.requests("changeHealthConsent").map(\.json)
        #expect(sent.count == 2 && sent[0]["id"] as? String == sent[1]["id"] as? String, "the retry is the same change")
        #expect(sent[1]["granted"] as? Bool == true)
        #expect(sent[1]["screening"] as? [Bool] == [false, true, false, false, false, false], "in the catalogue's order")
        #expect(sent[1]["past_injuries"] as? [String] == ["wrist_back_extension"], "only regions of the body map")
        #expect(model.screening.isEmpty && model.pastInjuries.isEmpty, "the answers are not kept")
        #expect(model.granted == true)
    }

    @Test func theOnboardingNamesTheConsentText() async throws {
        let db = try AppDatabase.inMemory()
        let server = ScriptedServer { _ in (200, onboardingResultJSON(status: "complete", plan: true)) }
        let model = try onboardingModel(server: server, db: db)
        model.form = filledForm()

        _ = try await model.submit(consentText: consentTextDE)

        let body = try #require(server.requests("submitOnboarding").first).json
        #expect(body["health_data_consent_text"] as? [String: String] == ["version": "1", "locale": "de"])
    }

    @Test func signingOutForgetsTheConsent() throws {
        let record = Data(consentJSON(granted: true, records: [consentRecordJSON(granted: true)]).utf8)
        let model = ConsentModel(sync: nil, plan: nil, body: nil, record: record)
        model.screening = ["SCR-01": true]
        #expect(model.granted == true && model.decisions.count == 1)

        model.forget()

        #expect(model.granted == nil && model.decisions.isEmpty && model.screening.isEmpty && model.status == .loading)
    }
}
