import Foundation
import HefestoAPI
import HefestoStore
import HefestoSync
import Observation

// The consent to keep health data after the onboarding (ADR 0031): what the
// athlete decided and when, a withdrawal, and a new grant with the questions
// that were not kept without it (ADR 0015). The text the athlete decides
// with is the app's; it goes with every decision by version and language.

/// The consent text the athlete decided with: its version, which the app
/// keeps with the text, and the language it was shown in.
public struct ConsentText: Sendable, Hashable {
    public var version: String
    public var locale: String

    public init(version: String, locale: String) {
        self.version = version
        self.locale = locale
    }
}

/// One decision on the consent, as the server recorded it.
public struct ConsentDecision: Sendable, Hashable {
    public var granted: Bool
    /// Given in the onboarding, not changed afterwards.
    public var inOnboarding: Bool
    /// Nil when the app that sent it did not name its text.
    public var text: ConsentText?
    public var at: Date
}

/// The consent and the record of the athlete's decisions. A change needs
/// the network; after one, the plan and the body's regions are fetched again.
@MainActor
@Observable
public final class ConsentModel {
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

    public private(set) var status: Status = .loading
    /// Nil until loaded.
    public private(set) var granted: Bool?
    /// The decisions, newest first.
    public private(set) var decisions: [ConsentDecision] = []
    /// The screening questions and the body map a grant asks again.
    public private(set) var catalogue: OnboardingCatalogue?
    /// A grant's answers: screening question id → yes.
    public var screening: [String: Bool] = [:]
    /// A grant's regions injured in the last 12 months.
    public var pastInjuries: Set<String> = []

    private let sync: SyncEngine?
    private let plan: PlanModel?
    private let body: BodyModel?
    /// The id of each change sent and not yet answered, by what it says: a
    /// retry after a lost answer is the same change (U-40).
    @ObservationIgnored private var pending: [String: String] = [:]

    /// `record` is the consent in the API's JSON, as a demo keeps it.
    public init(
        sync: SyncEngine?, plan: PlanModel?, body: BodyModel?, catalogue: OnboardingCatalogue? = nil,
        record: Data? = nil
    ) {
        self.sync = sync
        self.plan = plan
        self.body = body
        self.catalogue = catalogue
        if let record, let decoded = try? HefestoAPIConfiguration.decoder()
            .decode(Components.Schemas.HealthConsent.self, from: record) {
            show(decoded)
        }
    }

    /// The last decision, as the screen states it.
    public var latest: ConsentDecision? { decisions.first }

    /// Every screening question has its answer: a grant can go.
    public var canGrant: Bool {
        guard let catalogue else { return false }
        return catalogue.screening.allSatisfy { screening[$0.id] != nil }
    }

    /// Forgets the consent and a grant's answers, on sign-out. The catalogue
    /// is content and stays.
    public func forget() {
        status = .loading
        granted = nil
        decisions = []
        clearAnswers()
        pending = [:]
    }

    /// Fetches the consent, and the catalogue once.
    public func load() async {
        guard let sync else { return }
        do {
            if catalogue == nil { catalogue = OnboardingCatalogue(try await sync.plannerCatalogue()) }
            show(try await sync.healthConsent())
        } catch BodyError.onboardingRequired {
            status = .needsOnboarding
        } catch BodyError.unavailable {
            status = .unavailable
        } catch OnboardingError.unavailable {
            status = .unavailable
        } catch {
            status = granted == nil ? .offline : .ready
        }
    }

    /// Withdraws the consent: the server deletes the health data and keeps
    /// what protects the athlete; the pain reports kept here go too.
    public func withdraw(text: ConsentText) async throws -> BodyOutcome {
        try await change("withdraw|\(text.version)") { sync, id in
            try await sync.changeHealthConsent(granted: false, text: .init(text), id: id)
        }
    }

    /// Grants the consent with the screening answers and the regions
    /// injured in the last 12 months.
    public func grant(text: ConsentText) async throws -> BodyOutcome {
        guard let catalogue, canGrant else { throw ConsentModelError.incomplete }
        let answers = catalogue.screening.map { screening[$0.id] ?? false }
        let injuries = catalogue.regions.map(\.id).filter(pastInjuries.contains)
        let key = "grant|\(text.version)|" + answers.map { $0 ? "1" : "0" }.joined() + "|" + injuries.joined(separator: ",")
        let outcome = try await change(key) { sync, id in
            try await sync.changeHealthConsent(
                granted: true, screening: answers, pastInjuries: injuries, text: .init(text), id: id)
        }
        clearAnswers()
        return outcome
    }

    private func change(
        _ key: String,
        _ event: (SyncEngine, String) async throws -> Components.Schemas.PlanEventResult
    ) async throws -> BodyOutcome {
        guard let sync else { throw BodyModelError.noServer }
        let id = pending[key] ?? UUIDv7.make()
        pending[key] = id
        let result = try await event(sync, id)
        pending[key] = nil
        await plan?.refresh()
        await body?.load()
        await load()
        return BodyOutcome(result)
    }

    private func clearAnswers() {
        screening = [:]
        pastInjuries = []
    }

    private func show(_ c: Components.Schemas.HealthConsent) {
        granted = c.granted
        decisions = c.records.map(ConsentDecision.init)
        status = .ready
    }
}

public enum ConsentModelError: Error, Equatable {
    /// A screening question is still open.
    case incomplete
}

extension ConsentDecision {
    init(_ r: Components.Schemas.ConsentRecord) {
        granted = r.granted
        inOnboarding = r.source == .onboarding
        text = r.text.map { ConsentText(version: $0.version, locale: $0.locale) }
        at = r.at
    }
}

extension Components.Schemas.ConsentText {
    init(_ t: ConsentText) {
        self.init(version: t.version, locale: t.locale)
    }
}
