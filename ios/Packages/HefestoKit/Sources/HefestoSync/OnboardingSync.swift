import Foundation
import HefestoAPI
import HefestoStore

/// Why the onboarding did not go through, in terms the UI can explain.
public enum OnboardingError: Error, Equatable {
    /// The onboarding runs once, and this athlete has answered it already,
    /// perhaps on another device. The plan is there to fetch.
    case alreadyOnboarded
    /// The server cannot take some answers: messages by JSON pointer.
    case invalid([String: String])
    /// The planner cannot plan: its knowledge base is missing or invalid.
    case unavailable
    case refused(status: Int, type: String)
}

extension SyncEngine {
    /// The planner's catalogue: the skills, regions, red flags, screening
    /// questions and answer classes the onboarding asks with (ADR 0023).
    /// It is not kept: the onboarding needs the network anyway.
    public func plannerCatalogue() async throws -> Components.Schemas.PlannerCatalogue {
        switch try await client.getPlannerCatalogue(headers: .init()) {
        case let .ok(r): return try r.body.json
        case .notModified: throw SyncError.refused(status: 304, type: "")
        case let .serviceUnavailable(r): throw Self.onboardingError(503, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw SyncError.refused(status: status, type: "")
        }
    }

    /// Sends the onboarding's answers. A complete or blocked result carries
    /// the first plan, which is kept here like a fetched one; a result with
    /// questions carries none, and nothing is stored on the server.
    ///
    /// `key` makes a retry of the same answers return the first response;
    /// send new answers with a new key.
    public func submitOnboarding(
        _ answers: Components.Schemas.OnboardingAnswers, key: String, now: Date = Date()
    ) async throws -> Components.Schemas.OnboardingResult {
        let out = try await client.submitOnboarding(headers: .init(idempotencyKey: key), body: .json(answers))
        switch out {
        case let .ok(r):
            let result = try r.body.json
            if let plan = result.plan {
                try db.savePlan(CachedPlan(
                    weekStart: plan.weekStart, etag: nil,
                    payload: try HefestoAPIConfiguration.encoder().encode(plan), fetchedAt: now))
            }
            return result
        case let .conflict(r): throw Self.onboardingError(409, try? r.body.applicationProblemJson)
        case let .unprocessableContent(r):
            throw OnboardingError.invalid((try? r.body.applicationProblemJson)?.errors?.additionalProperties ?? [:])
        case let .serviceUnavailable(r): throw Self.onboardingError(503, try? r.body.applicationProblemJson)
        case let .badRequest(r): throw Self.onboardingError(400, try? r.body.applicationProblemJson)
        case let .unauthorized(r): throw Self.onboardingError(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw OnboardingError.refused(status: status, type: "")
        }
    }

    static func onboardingError(_ status: Int, _ p: Components.Schemas.Problem?) -> OnboardingError {
        let type = p?._type ?? ""
        switch type.split(separator: "/").last.map(String.init) ?? "" {
        case "already-onboarded": return .alreadyOnboarded
        case "planning-unavailable": return .unavailable
        default: return .refused(status: status, type: type)
        }
    }
}
