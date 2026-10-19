import Foundation
import HefestoAPI
import HefestoStore
import HefestoSync
import Observation

// The body as the screens read it (ADR 0028): each region with a complaint or
// a constraint, its warning-sign questions and what a professional's
// clearance would change. Names, questions, advice and reasons are the
// knowledge base's texts, shown verbatim; the app adds no advice and never
// names a diagnosis.

/// A body region the planner keeps a state or a constraint for.
public struct BodyRegion: Sendable, Hashable, Identifiable {
    public var id: String
    /// Where on the body, as the body map words it.
    public var name: String
    /// The planner keeps a state for the region; false without the consent
    /// to keep health data. Locks, exclusions and stops hold either way.
    public var tracked: Bool
    /// normal, locked, or a stage of the return ramp, rtt_0 to rtt_5.
    public var state: String?
    public var complaint: Bool
    public var priorInjury: Bool
    /// What a professional said to avoid, as the onboarding recorded it.
    public var restrictions: [String]
    /// soft or advise, when an assessment is recommended.
    public var referral: String?
    /// No load on the region until this day, YYYY-MM-DD.
    public var restUntil: String?
    /// region_locked, region_excluded or plan_stopped.
    public var constraints: [String]
    /// The warning-sign questions asked for the region.
    public var redFlags: [OnboardingCatalogue.Flag]

    /// The stage of the return ramp, 0 to 5, while the region is in it.
    public var rampStage: Int? {
        guard let state, state.hasPrefix("rtt_") else { return nil }
        return Int(state.dropFirst(4))
    }

    public var locked: Bool { state == "locked" || constraints.contains("region_locked") }

    /// A professional's clearance changes something: a locked region or one
    /// in stage 0 re-enters the ramp, and a stop by its warning signs ends
    /// (spec §8.3).
    public var clearable: Bool {
        locked || rampStage == 0 || referral != nil || constraints.contains("plan_stopped")
    }
}

/// What a body event changed, in the knowledge base's words.
public struct BodyOutcome: Sendable, Hashable {
    /// Regions whose warning-sign questions are to be asked now.
    public var askRedFlags: [String]
    /// The event stopped training (SAFE-02).
    public var trainingStopped: Bool
    /// The reasons of the changes, verbatim.
    public var notes: [PlanNote]

    init(_ r: Components.Schemas.PlanEventResult) {
        askRedFlags = r.changes.filter { $0.kind == "ask_red_flags" }.compactMap(\.region)
        trainingStopped = r.changes.contains { $0.kind == "training_stopped" }
        var seen = Set<String>()
        notes = r.changes.flatMap { $0.reasons.map(PlanNote.init) }.filter { seen.insert($0.text).inserted }
    }
}

public enum BodyModelError: Error, Equatable {
    /// The model has no server, as in a demo.
    case noServer
}

/// The body's state and the events the athlete sends about it. Every event
/// needs the network; after one, the plan is fetched again.
@MainActor
@Observable
public final class BodyModel {
    public enum Status: Equatable, Sendable {
        case loading
        case ready
        /// The server could not be reached.
        case offline
        /// There is no plan yet: the onboarding comes first.
        case needsOnboarding
        /// The server's planner is not available.
        case unavailable
    }

    public private(set) var regions: [BodyRegion] = []
    /// Every region of the body map, to report pain in any of them.
    public private(set) var catalogue: OnboardingCatalogue?
    /// The educational disclaimer the API sends with the regions.
    public private(set) var disclaimer = ""
    public private(set) var status: Status = .loading

    private let sync: SyncEngine?
    /// Fetched again after every event the server took.
    private let plan: PlanModel?
    /// The id of each event sent and not yet answered, by what it says: a
    /// retry after a lost answer is the same event (U-40).
    @ObservationIgnored private var pending: [String: String] = [:]

    /// `overview` is the regions in the API's JSON, as a demo keeps them.
    public init(sync: SyncEngine?, plan: PlanModel?, catalogue: OnboardingCatalogue? = nil, overview: Data? = nil) {
        self.sync = sync
        self.plan = plan
        self.catalogue = catalogue
        if let overview, let decoded = try? HefestoAPIConfiguration.decoder()
            .decode(Components.Schemas.RegionOverview.self, from: overview) {
            show(decoded)
        }
    }

    /// Forgets the regions and the events waiting for an answer, on
    /// sign-out. The catalogue and the disclaimer are content and stay.
    public func forget() {
        regions = []
        status = .loading
        pending = [:]
    }

    public func region(_ id: String) -> BodyRegion? {
        regions.first { $0.id == id }
    }

    /// Fetches the regions, and the catalogue once.
    public func load() async {
        guard let sync else { return }
        do {
            if catalogue == nil { catalogue = OnboardingCatalogue(try await sync.plannerCatalogue()) }
            show(try await sync.bodyRegions())
        } catch BodyError.onboardingRequired {
            status = .needsOnboarding
        } catch BodyError.unavailable {
            status = .unavailable
        } catch OnboardingError.unavailable {
            status = .unavailable
        } catch {
            status = regions.isEmpty ? .offline : .ready
        }
    }

    /// Reports exertion symptoms (RF-10): training stops at once.
    public func reportSymptoms() async throws -> BodyOutcome {
        try await send("symptoms") { sync, id in try await sync.reportSymptoms(id: id) }
    }

    public func reportPain(_ report: PainReport) async throws -> BodyOutcome {
        let key = "pain|\(report.region)|\(report.timepoint)|\(report.nrs)|\(report.at.timeIntervalSince1970)"
        return try await send(key) { sync, id in try await sync.reportPain(report, id: id) }
    }

    /// Answers the region's warning-sign questions, follow-ups included.
    public func answerRedFlags(region: String, answers: [String: Bool]) async throws -> BodyOutcome {
        let key = "flags|\(region)|" + answers.sorted { $0.key < $1.key }.map { "\($0.key)=\($0.value)" }.joined(separator: ",")
        return try await send(key) { sync, id in try await sync.answerRedFlags(region: region, answers: answers, id: id) }
    }

    /// Confirms a professional's clearance of a region, or of training as a
    /// whole without one.
    public func confirmClearance(region: String?) async throws -> BodyOutcome {
        try await send("clearance|\(region ?? "")") { sync, id in try await sync.confirmClearance(region: region, id: id) }
    }

    private func send(
        _ key: String,
        _ event: (SyncEngine, String) async throws -> Components.Schemas.PlanEventResult
    ) async throws -> BodyOutcome {
        guard let sync else { throw BodyModelError.noServer }
        let id = pending[key] ?? UUIDv7.make()
        pending[key] = id
        let result = try await event(sync, id)
        pending[key] = nil
        await plan?.refresh()
        await load()
        return BodyOutcome(result)
    }

    private func show(_ overview: Components.Schemas.RegionOverview) {
        regions = overview.regions.map(BodyRegion.init)
        disclaimer = overview.disclaimer
        status = .ready
    }
}

extension BodyRegion {
    init(_ r: Components.Schemas.RegionStatus) {
        id = r.region
        name = r.name
        tracked = r.tracked
        state = r.state
        complaint = r.complaint
        priorInjury = r.priorInjury
        restrictions = r.restrictions
        referral = r.referral
        restUntil = r.restUntil
        constraints = r.constraints
        redFlags = r.redFlags.map(OnboardingCatalogue.Flag.init)
    }
}
