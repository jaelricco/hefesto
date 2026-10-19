import Foundation
import HefestoAPI
import HefestoStore

// The body's side of the planner in the app (ADR 0028): exertion symptoms,
// pain reports, the state of each region, its warning-sign questions and a
// professional's clearance. Every event goes to the server at once; it is
// health data and safety, so nothing waits in the outbox.

/// Why a body event did not go through, in terms the UI can explain.
public enum BodyError: Error, Equatable {
    /// Pain reports are health data: without the consent to keep health
    /// data the server does not take them.
    case consentRequired
    /// The athlete has not answered the onboarding yet.
    case onboardingRequired
    /// The region is not one of the planner's.
    case unknownRegion
    /// The server cannot take some answers: messages by JSON pointer.
    case invalid([String: String])
    /// The planner cannot plan: its knowledge base is missing or invalid.
    case unavailable
    case refused(status: Int, type: String)
}

/// One pain rating as the athlete gives it (spec §8.6).
public struct PainReport: Sendable, Hashable {
    public var region: String
    /// before_session, warmup, during, after, next_morning or daily.
    public var timepoint: String
    /// 0 to 10.
    public var nrs: Double
    public var at: Date
    /// After the session: the pain lasted longer than an hour.
    public var lastedOver1h: Bool?
    /// In the warm-up: the pain stayed longer than 15 minutes.
    public var persistedOver15min: Bool?
    /// During training: the pain came suddenly and sharp.
    public var suddenSharp: Bool?
    /// The log session it belongs to, if any.
    public var sessionId: String?

    public init(
        region: String, timepoint: String, nrs: Double, at: Date, lastedOver1h: Bool? = nil,
        persistedOver15min: Bool? = nil, suddenSharp: Bool? = nil, sessionId: String? = nil
    ) {
        self.region = region; self.timepoint = timepoint; self.nrs = nrs; self.at = at
        self.lastedOver1h = lastedOver1h; self.persistedOver15min = persistedOver15min
        self.suddenSharp = suddenSharp; self.sessionId = sessionId
    }
}

extension SyncEngine {
    /// Reports chest pain, unusual shortness of breath, dizziness or
    /// fainting under exertion (RF-10). Training stops at once (SAFE-02)
    /// until the athlete confirms a professional's clearance. `id` makes a
    /// retry answer like the first time.
    public func reportSymptoms(id: String) async throws -> Components.Schemas.PlanEventResult {
        switch try await client.reportExertionSymptoms(body: .json(.init(id: id))) {
        case let .ok(r): return try r.body.json
        case let .conflict(r): throw Self.bodyError(409, try? r.body.applicationProblemJson)
        case let .unprocessableContent(r): throw Self.bodyInvalid(try? r.body.applicationProblemJson)
        case let .serviceUnavailable(r): throw Self.bodyError(503, try? r.body.applicationProblemJson)
        case let .badRequest(r): throw Self.bodyError(400, try? r.body.applicationProblemJson)
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw BodyError.refused(status: status, type: "")
        }
    }

    /// Reports one pain rating. The planner may ask the region's
    /// warning-sign questions (`ask_red_flags`) or change its ramp.
    public func reportPain(_ report: PainReport, id: String) async throws -> Components.Schemas.PlanEventResult {
        guard let timepoint = Components.Schemas.PainTimepoint(rawValue: report.timepoint) else {
            throw BodyError.invalid(["/timepoint": report.timepoint])
        }
        let body = Components.Schemas.PainReportCreate(
            id: id, region: report.region, timepoint: timepoint, nrs: report.nrs, at: report.at,
            lastedOver1h: report.lastedOver1h, persistedOver15min: report.persistedOver15min,
            suddenSharp: report.suddenSharp, sessionId: report.sessionId)
        switch try await client.reportPain(body: .json(body)) {
        case let .ok(r): return try r.body.json
        case let .conflict(r): throw Self.bodyError(409, try? r.body.applicationProblemJson)
        case let .unprocessableContent(r): throw Self.bodyInvalid(try? r.body.applicationProblemJson)
        case let .serviceUnavailable(r): throw Self.bodyError(503, try? r.body.applicationProblemJson)
        case let .badRequest(r): throw Self.bodyError(400, try? r.body.applicationProblemJson)
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw BodyError.refused(status: status, type: "")
        }
    }

    /// Every region with a complaint or a constraint, with its state, its
    /// restrictions and the warning-sign questions asked for it. Not kept:
    /// the state changes with every event.
    public func bodyRegions() async throws -> Components.Schemas.RegionOverview {
        switch try await client.listMyRegions() {
        case let .ok(r): return try r.body.json
        case let .conflict(r): throw Self.bodyError(409, try? r.body.applicationProblemJson)
        case let .serviceUnavailable(r): throw Self.bodyError(503, try? r.body.applicationProblemJson)
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw BodyError.refused(status: status, type: "")
        }
    }

    /// Answers a region's warning-sign questions: every question asked,
    /// and a follow-up exactly when its question was answered yes.
    public func answerRedFlags(
        region: String, answers: [String: Bool], id: String
    ) async throws -> Components.Schemas.PlanEventResult {
        let body = Components.Schemas.RedFlagAnswers(id: id, answers: .init(additionalProperties: answers))
        switch try await client.answerRedFlags(path: .init(region: region), body: .json(body)) {
        case let .ok(r): return try r.body.json
        case let .notFound(r): throw Self.bodyError(404, try? r.body.applicationProblemJson)
        case let .conflict(r): throw Self.bodyError(409, try? r.body.applicationProblemJson)
        case let .unprocessableContent(r): throw Self.bodyInvalid(try? r.body.applicationProblemJson)
        case let .serviceUnavailable(r): throw Self.bodyError(503, try? r.body.applicationProblemJson)
        case let .badRequest(r): throw Self.bodyError(400, try? r.body.applicationProblemJson)
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw BodyError.refused(status: status, type: "")
        }
    }

    /// Confirms that a professional cleared a region, or, without a region,
    /// training as a whole: the screening, a stop after exertion symptoms
    /// and every locked region (spec §8.3).
    public func confirmClearance(region: String?, id: String) async throws -> Components.Schemas.PlanEventResult {
        let body = Components.Schemas.PlanEventCreate(id: id)
        guard let region else {
            switch try await client.confirmScreeningClearance(body: .json(body)) {
            case let .ok(r): return try r.body.json
            case let .conflict(r): throw Self.bodyError(409, try? r.body.applicationProblemJson)
            case let .unprocessableContent(r): throw Self.bodyInvalid(try? r.body.applicationProblemJson)
            case let .serviceUnavailable(r): throw Self.bodyError(503, try? r.body.applicationProblemJson)
            case let .badRequest(r): throw Self.bodyError(400, try? r.body.applicationProblemJson)
            case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
            case let .undocumented(status, _): throw BodyError.refused(status: status, type: "")
            }
        }
        switch try await client.confirmRegionClearance(path: .init(region: region), body: .json(body)) {
        case let .ok(r): return try r.body.json
        case let .notFound(r): throw Self.bodyError(404, try? r.body.applicationProblemJson)
        case let .conflict(r): throw Self.bodyError(409, try? r.body.applicationProblemJson)
        case let .unprocessableContent(r): throw Self.bodyInvalid(try? r.body.applicationProblemJson)
        case let .serviceUnavailable(r): throw Self.bodyError(503, try? r.body.applicationProblemJson)
        case let .badRequest(r): throw Self.bodyError(400, try? r.body.applicationProblemJson)
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw BodyError.refused(status: status, type: "")
        }
    }

    /// The planner's problems by their type; anything else stays a refusal.
    static func bodyError(_ status: Int, _ p: Components.Schemas.Problem?) -> BodyError {
        let type = p?._type ?? ""
        switch type.split(separator: "/").last.map(String.init) ?? "" {
        case "consent-required": return .consentRequired
        case "onboarding-required": return .onboardingRequired
        case "planning-unavailable": return .unavailable
        case "not-found": return .unknownRegion
        default: return .refused(status: status, type: type)
        }
    }

    static func bodyInvalid(_ p: Components.Schemas.Problem?) -> BodyError {
        .invalid(p?.errors?.additionalProperties ?? [:])
    }
}
