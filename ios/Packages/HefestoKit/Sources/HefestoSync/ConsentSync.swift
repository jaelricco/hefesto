import Foundation
import HefestoAPI
import HefestoStore

// The consent to keep health data after the onboarding (ADR 0031): the
// decision and its record, a withdrawal and a new grant. Each goes to the
// server at once, with the version and language of the text the athlete
// read; nothing waits in the outbox.

extension SyncEngine {
    /// Whether the athlete consents now, and their decisions, newest first.
    public func healthConsent() async throws -> Components.Schemas.HealthConsent {
        switch try await client.getHealthConsent() {
        case let .ok(r): return try r.body.json
        case let .conflict(r): throw Self.bodyError(409, try? r.body.applicationProblemJson)
        case let .serviceUnavailable(r): throw Self.bodyError(503, try? r.body.applicationProblemJson)
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw BodyError.refused(status: status, type: "")
        }
    }

    /// Grants or withdraws the consent. A grant answers the six screening
    /// questions again and names the regions injured in the last 12 months.
    /// A withdrawal deletes the health data on the server, and the pain
    /// reports kept here go with it. `id` makes a retry answer like the
    /// first time.
    public func changeHealthConsent(
        granted: Bool, screening: [Bool]? = nil, pastInjuries: [String]? = nil,
        text: Components.Schemas.ConsentText?, id: String
    ) async throws -> Components.Schemas.PlanEventResult {
        let body = Components.Schemas.HealthConsentChange(
            id: id, granted: granted, screening: screening, pastInjuries: pastInjuries, text: text)
        let result: Components.Schemas.PlanEventResult
        switch try await client.changeHealthConsent(body: .json(body)) {
        case let .ok(r): result = try r.body.json
        case let .conflict(r): throw Self.bodyError(409, try? r.body.applicationProblemJson)
        case let .unprocessableContent(r): throw Self.bodyInvalid(try? r.body.applicationProblemJson)
        case let .serviceUnavailable(r): throw Self.bodyError(503, try? r.body.applicationProblemJson)
        case let .badRequest(r): throw Self.bodyError(400, try? r.body.applicationProblemJson)
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw BodyError.refused(status: status, type: "")
        }
        if !granted { try db.forgetPainReports() }
        return result
    }
}
