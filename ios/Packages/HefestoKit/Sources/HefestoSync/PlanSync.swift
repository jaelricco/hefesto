import Foundation
import HefestoAPI
import HefestoStore

/// Why the planner refused a request, in terms the UI can explain.
public enum PlanError: Error, Equatable {
    /// The athlete has not answered the onboarding: there is no plan yet.
    case onboardingRequired
    /// Training is stopped (SAFE-02, SAFE-07). The plan's reasons say why
    /// and what lifts the stop.
    case trainingStopped
    /// The planned session is in no active plan: a newer plan replaced it.
    case notFound
    /// The planner cannot plan: its knowledge base is missing or invalid.
    case unavailable
    case refused(status: Int, type: String)
}

/// What a plan refresh found.
public enum PlanRefresh: Equatable, Sendable {
    case updated
    case unchanged
}

extension SyncEngine {
    /// Fetches the plan of the week that holds `day` (YYYY-MM-DD, today in
    /// UTC by default) and keeps it for offline reading (ADR 0021). The kept
    /// plan's `ETag` goes along, so an unchanged plan costs a 304.
    ///
    /// The plan changes when the server applies an event: a completed
    /// session, a pain report, a start. Refresh after every sync and every
    /// start.
    @discardableResult
    public func refreshPlan(on day: String? = nil, now: Date = Date()) async throws -> PlanRefresh {
        let day = day ?? Self.utcDay(now)
        let kept = try db.plan(on: day)
        let out = try await client.getTrainingPlan(query: .init(week: day), headers: .init(ifNoneMatch: kept?.etag))
        switch out {
        case let .ok(r):
            let plan = try r.body.json
            try db.savePlan(CachedPlan(
                weekStart: plan.weekStart, etag: r.headers.eTag,
                payload: try HefestoAPIConfiguration.encoder().encode(plan), fetchedAt: now))
            return .updated
        case .notModified:
            return .unchanged
        case let .conflict(r):
            let error = Self.planError(409, try? r.body.applicationProblemJson)
            // The server has no plan for this athlete: nothing kept applies.
            if error == .onboardingRequired { try db.clearPlans() }
            throw error
        case let .serviceUnavailable(r): throw Self.planError(503, try? r.body.applicationProblemJson)
        case let .notFound(r): throw Self.planError(404, try? r.body.applicationProblemJson)
        case let .badRequest(r): throw Self.refused(400, try? r.body.applicationProblemJson)
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw SyncError.refused(status: status, type: "")
        }
    }

    /// Starts a planned session: the server writes it into the log as a
    /// draft with its planned sets (ADR 0016), and the draft is kept here
    /// like any session the server sends. Returns the log session's id.
    ///
    /// Needs the network. A planned session starts once: a repeat answers
    /// with the session of the first start, whatever `id` it sends, so a
    /// retry is safe. A session already kept here is left as it is, since it
    /// may hold changes not yet pushed; the next sync brings the rest.
    ///
    /// The kept plan still shows the session as planned until the next
    /// `refreshPlan()`.
    @discardableResult
    public func startPlannedSession(
        _ plannedSessionId: String, checkIn: Components.Schemas.CheckIn? = nil, timeZone: TimeZone = .current,
        id: String = UUIDv7.make(), at: Date = Date()
    ) async throws -> String {
        let body = Components.Schemas.PlannedSessionStart(
            id: id, startedAt: at, timezone: timeZone.identifier, checkIn: checkIn)
        let session: Components.Schemas.Session
        switch try await client.startPlannedSession(path: .init(plannedSessionId: plannedSessionId), body: .json(body)) {
        case let .created(r): session = try r.body.json
        case let .ok(r): session = try r.body.json
        case let .conflict(r): throw Self.planError(409, try? r.body.applicationProblemJson)
        case let .notFound(r): throw Self.planError(404, try? r.body.applicationProblemJson)
        case let .serviceUnavailable(r): throw Self.planError(503, try? r.body.applicationProblemJson)
        case let .badRequest(r): throw Self.refused(400, try? r.body.applicationProblemJson)
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .unprocessableContent(r): throw Self.refused(422, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw SyncError.refused(status: status, type: "")
        }
        if try !db.hasSession(id: session.id) {
            try db.replaceSessionTree(id: session.id, with: ServerRows.tree(session))
        }
        return session.id
    }

    /// The planner's problems by their type; anything else stays a refusal.
    static func planError(_ status: Int, _ p: Components.Schemas.Problem?) -> PlanError {
        let type = p?._type ?? ""
        switch type.split(separator: "/").last.map(String.init) ?? "" {
        case "onboarding-required": return .onboardingRequired
        case "training-stopped": return .trainingStopped
        case "planning-unavailable": return .unavailable
        case "not-found": return .notFound
        default: return .refused(status: status, type: type)
        }
    }

    static func utcDay(_ date: Date) -> String {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(identifier: "UTC") ?? .gmt
        let c = calendar.dateComponents([.year, .month, .day], from: date)
        return String(format: "%04d-%02d-%02d", c.year ?? 0, c.month ?? 0, c.day ?? 0)
    }
}

extension CachedPlan {
    /// The plan as the server sent it.
    public func decoded() throws -> Components.Schemas.TrainingPlan {
        try HefestoAPIConfiguration.decoder().decode(Components.Schemas.TrainingPlan.self, from: payload)
    }
}
