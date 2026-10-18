import Foundation
import Testing
@testable import HefestoAPI
@testable import HefestoPlan
@testable import HefestoStore
@testable import HefestoSync

// The body in the app (ADR 0028): events go to the server at once, a retry
// keeps its id, and the screens read their own types.

func regionFlagJSON(_ id: String, followup: String = "null") -> String {
    #"""
    {"id":"\#(id)","urgency":"D","action":"lock","question":"Q \#(id)?","advice":"A \#(id).","minor_only":false,
     "followup":\#(followup)}
    """#
}

func regionsJSON(state: String = "rtt_2", constraints: String = "[]", referral: String = "null") -> String {
    let followup = #"{"id":"RF-01b","urgency":"N","action":"stop","question":"Q RF-01b?","advice":"A RF-01b."}"#
    return #"""
    {"regions":[{"region":"elbow_inner","name":"innen am Ellbogen","tracked":true,"state":"\#(state)",
      "entered_via":"onboarding","since":"2026-10-05","start_fraction":0.25,"complaint":true,"prior_injury":false,
      "restrictions":["hang_pull"],"referral":\#(referral),"rest_until":null,"constraints":\#(constraints),
      "red_flags":[\#(regionFlagJSON("RF-01", followup: followup)),\#(regionFlagJSON("RF-10"))]}],
     "disclaimer":"Educational, not medical advice."}
    """#
}

func eventJSON(_ changes: [String] = []) -> String {
    #"{"changes":[\#(changes.joined(separator: ","))],"replayed":false}"#
}

func changeJSON(_ kind: String, region: String? = nil, text: String = "Because.") -> String {
    let region = region.map { #""\#($0)""# } ?? "null"
    return #"""
    {"kind":"\#(kind)","skill":null,"region":\#(region),"session_id":null,"from":null,"to":null,
     "reasons":[{"rule_id":"INJ-06","params":[],"sources":[],"evidence":"B","text":"\#(text)","args":{},"region":\#(region)}]}
    """#
}

@MainActor
func bodyModel(_ server: ScriptedServer) throws -> (BodyModel, AppDatabase) {
    let db = try AppDatabase.inMemory()
    let sync = SyncEngine(client: server.client, db: db)
    return (BodyModel(sync: sync, plan: nil), db)
}

@Suite @MainActor struct BodyTests {
    @Test func theRegionsReadAsTheScreensNeedThem() async throws {
        let server = ScriptedServer { r in
            r.operation == "listMyRegions" ? (200, regionsJSON()) : (200, catalogueJSON)
        }
        let (model, _) = try bodyModel(server)

        await model.load()

        #expect(model.status == .ready)
        let region = try #require(model.region("elbow_inner"))
        #expect(region.rampStage == 2 && !region.locked && !region.clearable)
        #expect(region.restrictions == ["hang_pull"])
        #expect(region.redFlags.map(\.id) == ["RF-01", "RF-10"])
        #expect(region.redFlags.first?.followup?.id == "RF-01b")
        #expect(model.disclaimer == "Educational, not medical advice.")
        #expect(model.catalogue != nil, "a pain report may name any region of the body map")
    }

    @Test func aLockedRegionCanBeCleared() throws {
        let overview = Data(regionsJSON(state: "locked", constraints: #"["region_locked"]"#, referral: #""advise""#).utf8)
        let model = BodyModel(sync: nil, plan: nil, overview: overview)

        let region = try #require(model.region("elbow_inner"))
        #expect(region.locked && region.clearable && region.rampStage == nil)
        #expect(region.referral == "advise")
    }

    @Test func aPainReportSendsWhatTheMomentAsks() async throws {
        let server = ScriptedServer { r in
            switch r.operation {
            case "reportPain": (200, eventJSON([changeJSON("ask_red_flags", region: "elbow_inner", text: "Answer them.")]))
            case "listMyRegions": (200, regionsJSON())
            default: (200, catalogueJSON)
            }
        }
        let (model, _) = try bodyModel(server)
        let at = Date(timeIntervalSince1970: 1_790_676_000)

        let outcome = try await model.reportPain(PainReport(
            region: "elbow_inner", timepoint: "during", nrs: 6, at: at, suddenSharp: true, sessionId: "s-1"))

        let sent = try #require(server.requests("reportPain").first?.json)
        #expect(sent["region"] as? String == "elbow_inner")
        #expect(sent["timepoint"] as? String == "during")
        #expect(sent["nrs"] as? Double == 6)
        #expect(sent["sudden_sharp"] as? Bool == true)
        #expect(sent["lasted_over_1h"] == nil && sent["persisted_over_15min"] == nil)
        #expect(sent["session_id"] as? String == "s-1")
        #expect(outcome.askRedFlags == ["elbow_inner"])
        #expect(outcome.notes.map(\.text) == ["Answer them."])
        #expect(!outcome.trainingStopped)
        #expect(server.requests("listMyRegions").count == 1, "the regions are fetched again")
    }

    @Test func aRetryAfterALostAnswerKeepsItsId() async throws {
        let calls = Counter()
        let server = ScriptedServer { r in
            if r.operation == "reportExertionSymptoms" && calls.next() == 1 { throw LostConnection() }
            return r.operation == "reportExertionSymptoms"
                ? (200, eventJSON([changeJSON("training_stopped")]))
                : (200, regionsJSON())
        }
        let (model, _) = try bodyModel(server)

        await #expect(throws: (any Error).self) { _ = try await model.reportSymptoms() }
        let outcome = try await model.reportSymptoms()

        let ids = server.requests("reportExertionSymptoms").compactMap { $0.json["id"] as? String }
        #expect(ids.count == 2 && ids[0] == ids[1])
        #expect(outcome.trainingStopped)
        _ = try await model.reportSymptoms()
        let third = server.requests("reportExertionSymptoms").last?.json["id"] as? String
        #expect(third != ids[0], "a new report after an answered one is a new event")
    }

    @Test func painNeedsTheConsent() async throws {
        let server = ScriptedServer { _ in (409, problemJSON("consent-required", status: 409)) }
        let (model, _) = try bodyModel(server)

        await #expect(throws: BodyError.consentRequired) {
            _ = try await model.reportPain(PainReport(region: "elbow_inner", timepoint: "daily", nrs: 3, at: Date()))
        }
    }

    @Test func redFlagsAndClearancesGoToTheirRegion() async throws {
        let server = ScriptedServer { r in
            r.operation == "listMyRegions" ? (200, regionsJSON()) : (200, eventJSON())
        }
        let (model, _) = try bodyModel(server)

        _ = try await model.answerRedFlags(region: "elbow_inner", answers: ["RF-01": true, "RF-01b": false, "RF-10": false])
        _ = try await model.confirmClearance(region: "elbow_inner")
        _ = try await model.confirmClearance(region: nil)

        let flags = try #require(server.requests("answerRedFlags").first)
        #expect(flags.path.hasSuffix("/v1/me/regions/elbow_inner/red-flags"))
        let answers = try #require(flags.json["answers"] as? [String: Bool])
        #expect(answers == ["RF-01": true, "RF-01b": false, "RF-10": false])
        #expect(server.requests("confirmRegionClearance").first?.path.hasSuffix("/v1/me/regions/elbow_inner/clearance") == true)
        #expect(server.requests("confirmScreeningClearance").count == 1)
    }

    @Test func aDemoSendsNothing() async throws {
        let model = BodyModel(sync: nil, plan: nil)

        await #expect(throws: BodyModelError.noServer) { _ = try await model.reportSymptoms() }
    }
}
