import Foundation
import HefestoAPI
import HefestoStore

// The plan's settings after the onboarding (ADR 0029): the goals, the time a
// week, the equipment and the body weight. Each change goes to the server at
// once; the server generates the week's plan anew (WEEK-08).

/// Why a settings change did not go through, in terms the UI can explain.
public enum SettingsError: Error, Equatable {
    /// The athlete has not answered the onboarding yet.
    case onboardingRequired
    /// The server cannot take some answers: messages by JSON pointer.
    case invalid([String: String])
    /// The planner cannot plan: its knowledge base is missing or invalid.
    case unavailable
    case refused(status: Int, type: String)
}

extension SyncEngine {
    public func trainingProfile() async throws -> Components.Schemas.TrainingProfile {
        switch try await client.getTrainingProfile() {
        case let .ok(r): return try r.body.json
        case let .conflict(r): throw Self.settingsError(409, try? r.body.applicationProblemJson)
        case let .serviceUnavailable(r): throw Self.settingsError(503, try? r.body.applicationProblemJson)
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw SettingsError.refused(status: status, type: "")
        }
    }

    /// Replaces the part of the profile that may change after the
    /// onboarding. Optional fields left out are cleared, so send back what
    /// is not changed.
    public func updateTrainingProfile(
        _ update: Components.Schemas.TrainingProfileUpdate
    ) async throws -> Components.Schemas.TrainingProfile {
        switch try await client.updateTrainingProfile(body: .json(update)) {
        case let .ok(r): return try r.body.json
        case let .conflict(r): throw Self.settingsError(409, try? r.body.applicationProblemJson)
        case let .unprocessableContent(r):
            throw SettingsError.invalid((try? r.body.applicationProblemJson)?.errors?.additionalProperties ?? [:])
        case let .serviceUnavailable(r): throw Self.settingsError(503, try? r.body.applicationProblemJson)
        case let .badRequest(r): throw Self.settingsError(400, try? r.body.applicationProblemJson)
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw SettingsError.refused(status: status, type: "")
        }
    }

    public func trainingGoals() async throws -> Components.Schemas.TrainingGoalList {
        switch try await client.getTrainingGoals() {
        case let .ok(r): return try r.body.json
        case let .conflict(r): throw Self.settingsError(409, try? r.body.applicationProblemJson)
        case let .serviceUnavailable(r): throw Self.settingsError(503, try? r.body.applicationProblemJson)
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw SettingsError.refused(status: status, type: "")
        }
    }

    /// Replaces the goals: one to three, the first with priority 1.
    public func updateTrainingGoals(
        _ goals: [Components.Schemas.TrainingGoalWrite]
    ) async throws -> Components.Schemas.TrainingGoalList {
        switch try await client.updateTrainingGoals(body: .json(.init(goals: goals))) {
        case let .ok(r): return try r.body.json
        case let .conflict(r): throw Self.settingsError(409, try? r.body.applicationProblemJson)
        case let .unprocessableContent(r):
            throw SettingsError.invalid((try? r.body.applicationProblemJson)?.errors?.additionalProperties ?? [:])
        case let .serviceUnavailable(r): throw Self.settingsError(503, try? r.body.applicationProblemJson)
        case let .badRequest(r): throw Self.settingsError(400, try? r.body.applicationProblemJson)
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw SettingsError.refused(status: status, type: "")
        }
    }

    static func settingsError(_ status: Int, _ p: Components.Schemas.Problem?) -> SettingsError {
        let type = p?._type ?? ""
        switch type.split(separator: "/").last.map(String.init) ?? "" {
        case "onboarding-required": return .onboardingRequired
        case "planning-unavailable": return .unavailable
        default: return .refused(status: status, type: type)
        }
    }
}
