import Foundation
import Testing
@testable import HefestoAPI
@testable import HefestoPlan
@testable import HefestoStore
@testable import HefestoSync

// The plan's settings (ADR 0029): what the athlete changes is sent, the rest
// goes back as it was.

func profileJSON(sessions: Int = 3, minutes: Int = 60, bodyweight: Double = 75) -> String {
    #"""
    {"birth_year":1995,"is_minor":false,"sessions_per_week":\#(sessions),"session_minutes":\#(minutes),
     "equipment":["floor","pull_up_bar","wall"],"bodyweight_kg":\#(bodyweight),"training_level":"trained",
     "training_months":24,"last_regular_training":"current_or_lt_3_weeks","health_data_consent":true,
     "disclaimer_ack":true,"onboarded_at":"2026-09-28","preferred_days":[1,3,5],"mobility_checks":{"wrist":"partly"},
     "max_added_load_kg":20,"smallest_plate_kg":null}
    """#
}

func goalsJSON(_ goals: [(String, String)] = [("pull-up", "strict-5")]) -> String {
    let items = goals.enumerated().map { i, g in
        #"{"skill":"\#(g.0)","target_level":"\#(g.1)","priority":\#(i + 1),"target_date":null}"#
    }
    return #"{"goals":[\#(items.joined(separator: ","))],"realism":[]}"#
}

@Suite @MainActor struct PlanSettingsTests {
    func settings(_ server: ScriptedServer) throws -> PlanSettingsModel {
        let db = try AppDatabase.inMemory()
        return PlanSettingsModel(sync: SyncEngine(client: server.client, db: db), plan: nil)
    }

    @Test func loadsTheSettingsIntoTheOnboardingForm() async throws {
        let server = ScriptedServer { r in
            switch r.operation {
            case "getTrainingProfile": (200, profileJSON())
            case "getTrainingGoals": (200, goalsJSON())
            default: (200, catalogueJSON)
            }
        }
        let model = try settings(server)

        await model.load()

        #expect(model.status == .ready)
        let form = model.editor.form
        #expect(form.goals.map(\.skill) == ["pull-up"] && form.goals.first?.targetLevel == "strict-5")
        #expect(form.sessionsPerWeek == 3 && form.sessionMinutes == 60 && form.bodyweightKg == 75)
        #expect(form.equipment == ["floor", "pull_up_bar", "wall"])
        #expect(!model.hasChanges && !model.canSave, "nothing changed yet")
    }

    @Test func sendsOnlyWhatChangedAndKeepsTheRest() async throws {
        let server = ScriptedServer { r in
            switch r.operation {
            case "getTrainingProfile": (200, profileJSON())
            case "getTrainingGoals": (200, goalsJSON())
            case "updateTrainingProfile": (200, profileJSON(sessions: 4, bodyweight: 74.5))
            default: (200, catalogueJSON)
            }
        }
        let model = try settings(server)
        await model.load()

        model.editor.form.sessionsPerWeek = 4
        model.editor.form.bodyweightKg = 74.5
        #expect(model.canSave)
        try await model.save()

        #expect(server.requests("updateTrainingGoals").isEmpty, "the goals did not change")
        let sent = try #require(server.requests("updateTrainingProfile").first?.json)
        #expect(sent["sessions_per_week"] as? Int == 4)
        #expect(sent["bodyweight_kg"] as? Double == 74.5)
        #expect(sent["preferred_days"] as? [Int] == [1, 3, 5], "what the screens do not edit goes back")
        #expect((sent["mobility_checks"] as? [String: String])?["wrist"] == "partly")
        #expect(sent["max_added_load_kg"] as? Double == 20)
        #expect(sent["smallest_plate_kg"] == nil)
        #expect(!model.hasChanges)
    }

    @Test func newGoalsGoInTheirOrder() async throws {
        let server = ScriptedServer { r in
            switch r.operation {
            case "getTrainingProfile": (200, profileJSON())
            case "getTrainingGoals": (200, goalsJSON())
            case "updateTrainingGoals": (200, goalsJSON([("planche", "tuck"), ("pull-up", "strict-5")]))
            default: (200, catalogueJSON)
            }
        }
        let model = try settings(server)
        await model.load()

        model.editor.form.goals.insert(OnboardingForm.Goal(skill: "planche", targetLevel: "tuck"), at: 0)
        try await model.save()

        let sent = try #require(server.requests("updateTrainingGoals").first?.json["goals"] as? [[String: Any]])
        #expect(sent.map { $0["skill"] as? String } == ["planche", "pull-up"])
        #expect(sent.map { $0["priority"] as? Int } == [1, 2])
        #expect(server.requests("updateTrainingProfile").isEmpty)
    }

    @Test func signingOutForgetsTheSettings() async throws {
        let server = ScriptedServer { r in
            switch r.operation {
            case "getTrainingProfile": (200, profileJSON())
            case "getTrainingGoals": (200, goalsJSON())
            default: (200, catalogueJSON)
            }
        }
        let model = try settings(server)
        await model.load()
        model.editor.form.sessionsPerWeek = 4

        model.forget()

        #expect(model.status == .loading)
        #expect(model.editor.form == OnboardingForm(), "the next account starts empty")
        #expect(!model.hasChanges)
    }

    @Test func withoutAnOnboardingThereIsNothingToSet() async throws {
        let server = ScriptedServer { r in
            r.operation == "getPlannerCatalogue"
                ? (200, catalogueJSON) : (409, problemJSON("onboarding-required", status: 409))
        }
        let model = try settings(server)

        await model.load()

        #expect(model.status == .needsOnboarding)
    }
}
