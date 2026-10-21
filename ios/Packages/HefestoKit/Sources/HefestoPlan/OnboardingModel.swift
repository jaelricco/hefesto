import Foundation
import HefestoAPI
import HefestoStore
import HefestoSync
import Observation

/// The onboarding the screens walk through (ADR 0023): the catalogue it
/// asks with, the answers so far and what the server made of them. It needs
/// the network twice, for the catalogue and for the answers; a complete or
/// blocked result keeps the first plan, which the plan model then shows.
@MainActor
@Observable
public final class OnboardingModel {
    /// Nil until it is loaded.
    public private(set) var catalogue: OnboardingCatalogue?
    public var form = OnboardingForm()
    /// The last result; questions in it are answered in `form.clarifications`.
    public private(set) var outcome: OnboardingOutcome?

    /// Nil where the model must never reach the network, as in a demo.
    private let sync: SyncEngine?
    private let timeZone: TimeZone
    private let now: @Sendable () -> Date
    /// The answers last sent and their key, so a retry after a lost
    /// response returns the first result rather than a second one.
    @ObservationIgnored
    private var sent: (answers: Components.Schemas.OnboardingAnswers, key: String)?

    public init(
        sync: SyncEngine?, catalogue: OnboardingCatalogue? = nil, timeZone: TimeZone = .current,
        now: @escaping @Sendable () -> Date = { Date() }
    ) {
        self.sync = sync
        self.catalogue = catalogue
        self.timeZone = timeZone
        self.now = now
    }

    /// The year now, for the birth year and the minor check.
    public var currentYear: Int {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = timeZone
        return calendar.component(.year, from: now())
    }

    /// Today, YYYY-MM-DD, the earliest date a goal can have.
    public var today: String {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = timeZone
        let c = calendar.dateComponents([.year, .month, .day], from: now())
        return String(format: "%04d-%02d-%02d", c.year ?? 0, c.month ?? 0, c.day ?? 0)
    }

    /// Forgets the answers and the result, on sign-out: the next account
    /// starts empty. The catalogue is content, not the athlete's, and stays.
    public func forget() {
        form = OnboardingForm()
        outcome = nil
        sent = nil
    }

    /// Fetches the catalogue once.
    public func load() async throws {
        guard catalogue == nil else { return }
        guard let sync else { throw OnboardingSubmitError.noServer }
        catalogue = OnboardingCatalogue(try await sync.plannerCatalogue())
    }

    public var pages: [OnboardingPage] {
        catalogue.map { form.pages($0) } ?? []
    }

    public func isComplete(_ page: OnboardingPage) -> Bool {
        guard let catalogue else { return false }
        return form.isComplete(page, catalogue, currentYear: currentYear)
    }

    /// Sends the answers and returns what the server made of them. Questions
    /// in the result are answered in `form.clarifications`, then sent again.
    /// `consentText` is the text the consent was decided with (ADR 0031).
    @discardableResult
    public func submit(consentText: ConsentText? = nil) async throws -> OnboardingOutcome {
        guard let catalogue, var answers = form.request(catalogue, currentYear: currentYear) else {
            throw OnboardingSubmitError.incomplete
        }
        answers.healthDataConsentText = consentText.map(Components.Schemas.ConsentText.init)
        guard let sync else { throw OnboardingSubmitError.noServer }
        // The same answers again are a retry and keep their key.
        let key = sent.flatMap { $0.answers == answers ? $0.key : nil } ?? UUIDv7.make()
        sent = (answers, key)
        let result = try await sync.submitOnboarding(answers, key: key, now: now())
        let outcome = OnboardingOutcome(result)
        self.outcome = outcome
        return outcome
    }
}

public enum OnboardingSubmitError: Error, Equatable {
    /// The model has no server, as in a demo.
    case noServer
    /// A page still has a question open.
    case incomplete
}
