import Foundation
import HefestoAPI
import HefestoStore

// Pain reports kept on the phone until the server has them (ADR 0030). A
// report goes at once, as every event about the body (ADR 0028); one that
// cannot go yet waits for the next sync, with its id, so a retry is the same
// report (U-40).

/// What sending pain reports came to.
public struct PainSendResult: Sendable {
    /// The ids of the reports just kept, in the order given.
    public var kept: [String] = []
    /// The server's answer to each report it took now.
    public var results: [Components.Schemas.PlanEventResult] = []
    /// Reports the server refused for good, by id; they are dropped.
    public var refused: [String: BodyError] = [:]
    /// Reports kept here for the next sync.
    public var waiting = 0
}

extension SyncEngine {
    /// Keeps the reports, then sends every report the server does not have
    /// yet, oldest first.
    public func sendPain(_ reports: [PainReport]) async throws -> PainSendResult {
        let kept = reports.map {
            KeptPainReport(region: $0.region, timepoint: $0.timepoint, nrs: $0.nrs, at: $0.at,
                           lastedOver1h: $0.lastedOver1h, persistedOver15min: $0.persistedOver15min,
                           suddenSharp: $0.suddenSharp, sessionId: $0.sessionId)
        }
        try db.keepPainReports(kept)
        var out = try await flushPainReports()
        out.kept = kept.map(\.id)
        return out
    }

    /// Sends every kept report the server does not have yet, oldest first,
    /// and stops at the first that cannot go now. A report the server
    /// refuses for good, as without consent, is dropped.
    public func flushPainReports() async throws -> PainSendResult {
        var out = PainSendResult()
        let unsent = try db.unsentPainReports()
        for (i, kept) in unsent.enumerated() {
            let report = PainReport(
                region: kept.region, timepoint: kept.timepoint, nrs: kept.nrs, at: kept.at,
                lastedOver1h: kept.lastedOver1h, persistedOver15min: kept.persistedOver15min,
                suddenSharp: kept.suddenSharp, sessionId: kept.sessionId)
            do {
                out.results.append(try await reportPain(report, id: kept.id))
                try db.markPainReportSent(id: kept.id)
            } catch let error as BodyError where Self.isFinal(error) {
                out.refused[kept.id] = error
                try db.dropPainReport(id: kept.id)
            } catch {
                out.waiting = unsent.count - i
                return out
            }
        }
        return out
    }

    /// The server will never take the report: retrying cannot help.
    static func isFinal(_ error: BodyError) -> Bool {
        switch error {
        case .consentRequired, .onboardingRequired, .unknownRegion, .invalid: true
        case .unavailable: false
        case let .refused(status, _): (400..<500).contains(status) && ![401, 408, 429].contains(status)
        }
    }
}
