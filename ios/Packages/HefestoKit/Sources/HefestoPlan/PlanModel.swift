import Foundation
import HefestoAPI
import HefestoStore
import HefestoSync
import Observation

/// The plan the screens show and the one way they start a planned session
/// (ADR 0022). It reads the plan kept in the local database, so it shows the
/// week offline; refreshing and starting need the network.
@MainActor
@Observable
public final class PlanModel {
    public enum Status: Equatable, Sendable {
        /// The plan kept here, if any, is what there is to show.
        case ready
        /// There is no plan: the onboarding comes first.
        case needsOnboarding
        /// The server's planner is not available.
        case unavailable
    }

    /// This week's plan, nil until one is kept here.
    public private(set) var week: PlanWeek?
    public private(set) var status: Status = .ready

    private let db: AppDatabase
    /// Nil where the model must never reach the network, as in a demo.
    private let sync: SyncEngine?
    private let timeZone: TimeZone
    private let now: @Sendable () -> Date

    public init(
        db: AppDatabase, sync: SyncEngine?, timeZone: TimeZone = .current,
        now: @escaping @Sendable () -> Date = { Date() }
    ) {
        self.db = db
        self.sync = sync
        self.timeZone = timeZone
        self.now = now
    }

    /// Today, YYYY-MM-DD in the athlete's time zone, as the plan dates its
    /// sessions.
    public var today: String {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = timeZone
        let c = calendar.dateComponents([.year, .month, .day], from: now())
        return String(format: "%04d-%02d-%02d", c.year ?? 0, c.month ?? 0, c.day ?? 0)
    }

    /// Today's planned session, if the plan has one.
    public var todaySession: PlanSession? {
        week?.days.first { $0.date == today }?.session
    }

    /// Follows the plan kept for this week until the task is cancelled.
    public func observe() async {
        do {
            for try await cached in db.observePlan(on: today) { show(cached) }
        } catch {
            // The database stopped the observation; the last week stays.
        }
    }

    /// Reads the plan kept for this week once.
    public func reload() throws {
        show(try db.plan(on: today))
    }

    /// Fetches this week's plan. What the planner says becomes the status;
    /// a network failure leaves the plan kept here as it is.
    public func refresh() async {
        guard let sync else { return }
        do {
            try await sync.refreshPlan(on: today)
            status = .ready
            try reload()
        } catch PlanError.onboardingRequired {
            status = .needsOnboarding
            week = nil
        } catch PlanError.unavailable {
            status = .unavailable
        } catch {
            // Offline, or refused for a reason a later refresh can fix.
        }
    }

    /// The log session to open for a planned session that was started, if
    /// it is kept here.
    public func logSession(for session: PlanSession) -> String? {
        guard let id = session.workoutSessionId, (try? db.hasSession(id: id)) == true else { return nil }
        return id
    }

    /// Starts a planned session with the check-in the athlete gave, if any,
    /// and returns the log session to open. The server writes the draft
    /// (ADR 0016) and lightens it after a tired check-in (ADR 0019); the
    /// answers go only into this request.
    public func start(_ session: PlanSession, sleepHours: Double?, fatigue: Int?) async throws -> String {
        guard let sync else { throw PlanStartError.noServer }
        var checkIn: Components.Schemas.CheckIn?
        if sleepHours != nil || fatigue != nil {
            checkIn = Components.Schemas.CheckIn(sleepHours: sleepHours, fatigue: fatigue)
        }
        let id: String
        do {
            id = try await sync.startPlannedSession(session.id, checkIn: checkIn, timeZone: timeZone, at: now())
        } catch PlanError.trainingStopped {
            // The plan says why; show it before saying no.
            await refresh()
            throw PlanError.trainingStopped
        } catch PlanError.notFound {
            // A newer plan replaced the session.
            await refresh()
            throw PlanError.notFound
        }
        await refresh()
        return id
    }

    private func show(_ cached: CachedPlan?) {
        guard let cached else {
            week = nil
            return
        }
        do {
            week = try PlanWeek(decoding: cached)
        } catch {
            // A plan this version cannot read is as good as none; the next
            // refresh replaces it.
            week = nil
        }
    }
}

public enum PlanStartError: Error, Equatable {
    /// The model has no server to start on, as in a demo.
    case noServer
}
