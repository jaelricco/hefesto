import Foundation

/// The rest between two sets. A value, not a ticking object: time left is
/// computed from the wall clock, so it is right after the app was in the
/// background or the phone was locked (ADR 0010).
public struct RestTimer: Sendable, Hashable, Codable {
    /// When the set that began this rest was completed.
    public var startedAt: Date
    /// The planned rest, in seconds; nil when none was planned.
    public var plannedSeconds: Int?

    public init(startedAt: Date, plannedSeconds: Int?) {
        self.startedAt = startedAt
        self.plannedSeconds = plannedSeconds
    }

    /// Seconds rested so far, never negative.
    public func elapsed(at now: Date) -> Int {
        max(0, Int(now.timeIntervalSince(startedAt).rounded(.down)))
    }

    /// Seconds of planned rest left, never negative; nil with no plan.
    public func remaining(at now: Date) -> Int? {
        plannedSeconds.map { max(0, $0 - elapsed(at: now)) }
    }

    /// Whether the planned rest is over.
    public func isDone(at now: Date) -> Bool {
        remaining(at: now) == 0
    }

    /// When the planned rest ends, for a local notification.
    public var endsAt: Date? {
        plannedSeconds.map { startedAt.addingTimeInterval(TimeInterval($0)) }
    }
}
