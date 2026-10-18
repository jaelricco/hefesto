import Foundation
import HefestoAPI
import HefestoStore
import HefestoSync

// The plan as the screens read it (ADR 0022): plain values, built from the
// server's plan in one place. A change to the API's plan types stops at the
// mapping below, as with the celebration (ADR 0020).

/// A week of the plan, every day of it: a session or a rest day.
public struct PlanWeek: Sendable, Hashable {
    public var id: String
    /// The Monday the week starts on, YYYY-MM-DD.
    public var weekStart: String
    /// Training is stopped until a clearance (SAFE-02) or for a minor
    /// (SAFE-07). The reasons say why and what lifts it.
    public var stopped: Bool
    /// The kind of deload this week is, if it is one.
    public var deload: String?
    public var sessions: [PlanSession]
    /// The seven days in order, each with its session if it has one.
    public var days: [PlanDay]
    /// Why the week looks as it does, in the knowledge base's words.
    public var reasons: [PlanNote]
    public var hints: [PlanNote]
    /// Exercises left out, and why.
    public var exclusions: [PlanExclusion]
    /// The educational disclaimer the API sends with every plan.
    public var disclaimer: String

    /// A planned session by its id.
    public func session(id: String) -> PlanSession? {
        sessions.first { $0.id == id }
    }

    /// The planned session a log session was started as.
    public func session(startedAs logSessionId: String) -> PlanSession? {
        sessions.first { $0.workoutSessionId == logSessionId }
    }

    /// A plan item by its id, the one a planned set names.
    public func item(id: String) -> PlanItem? {
        for s in sessions {
            for b in s.blocks {
                if let item = b.items.first(where: { $0.id == id }) { return item }
            }
        }
        return nil
    }
}

public struct PlanDay: Sendable, Hashable, Identifiable {
    /// YYYY-MM-DD.
    public var date: String
    /// Nil on a rest day.
    public var session: PlanSession?
    public var id: String { date }
    public var isRestDay: Bool { session == nil }
}

public struct PlanSession: Sendable, Hashable, Identifiable {
    public enum Status: String, Sendable {
        case planned, started, completed
    }

    public var id: String
    public var date: String
    /// full, light or deload.
    public var kind: String
    public var estMinutes: Double
    public var status: Status
    /// The log session it was started as.
    public var workoutSessionId: String?
    /// A tired check-in made it lighter (ADR 0019).
    public var checkInApplied: Bool
    public var blocks: [PlanBlock]
    public var reasons: [PlanNote]

    /// The items the plan asks for, without offers.
    public var plannedItems: [PlanItem] { blocks.flatMap(\.items).filter { !$0.offer } }
    /// Offers: done only on the athlete's active choice (ADAPT-05).
    public var offers: [PlanItem] { blocks.flatMap(\.items).filter(\.offer) }
}

public struct PlanBlock: Sendable, Hashable {
    /// warmup, balance, skill_max, skill_volume, strength or end.
    public var role: String
    /// Time of a general warm-up or balance practice.
    public var minutes: Double?
    /// The items alternate as antagonist pairs.
    public var paired: Bool
    public var items: [PlanItem]
    public var reasons: [PlanNote]
}

public struct PlanItem: Sendable, Hashable, Identifiable {
    public var id: String
    /// The exercise's slug, as the catalogue knows it.
    public var exercise: String
    public var exerciseName: String
    public var sets: Int
    public var reps: Int?
    public var holdSeconds: Int?
    /// Added load.
    public var loadKg: Double?
    /// none or band.
    public var assist: String
    /// Reps or seconds left at the end of each set.
    public var reserve: Int
    public var restSeconds: Int
    /// The first working set calibrates the estimate: its reserve matters.
    public var calibration: Bool
    /// A probe or first attempt, done only on the athlete's active choice.
    public var offer: Bool
    /// light, moderate or hard.
    public var intensity: String
    /// working or warmup.
    public var kind: String
    /// The item loads a region under pain monitoring.
    public var monitor: Bool
    public var reasons: [PlanNote]

    public var isHold: Bool { holdSeconds != nil }
}

/// A reason or hint, as the knowledge base words it.
public struct PlanNote: Sendable, Hashable {
    public var ruleId: String
    public var text: String
    /// The body region it concerns, if any.
    public var region: String?
}

public struct PlanExclusion: Sendable, Hashable {
    public var exerciseName: String
    public var reason: PlanNote
}

extension PlanWeek {
    /// The plan a cached document holds.
    public init(decoding cached: CachedPlan) throws {
        self.init(try cached.decoded())
    }

    /// The one mapping from the API's plan.
    init(_ p: Components.Schemas.TrainingPlan) {
        let sessions = p.sessions.map(PlanSession.init)
        id = p.id
        weekStart = p.weekStart
        stopped = p.stopped
        deload = p.deload
        self.sessions = sessions
        days = Self.days(from: p.weekStart, sessions: sessions)
        reasons = p.reasons.map(PlanNote.init)
        hints = p.hints.map(PlanNote.init)
        exclusions = p.exclusions.map { PlanExclusion(exerciseName: $0.exerciseName, reason: PlanNote($0.reason)) }
        disclaimer = p.disclaimer
    }

    /// The week as it stands here: a start or a completion kept here that
    /// the kept plan does not show yet, as after a start without the server
    /// (spec §10.5). A session the plan gives to another log session keeps
    /// it; the plan has the last word once it knows.
    func applying(_ starts: [PlannedStart]) -> PlanWeek {
        guard !starts.isEmpty else { return self }
        var week = self
        week.sessions = sessions.map { (s: PlanSession) -> PlanSession in
            let start: PlannedStart?
            if let id = s.workoutSessionId {
                start = starts.first { $0.sessionId == id }
            } else {
                start = starts.first { $0.plannedSessionId == s.id }
            }
            guard let start else { return s }
            var out = s
            out.workoutSessionId = start.sessionId
            if start.completed {
                out.status = .completed
            } else if s.status == .planned {
                out.status = .started
            }
            return out
        }
        week.days = Self.days(from: weekStart, sessions: week.sessions)
        return week
    }

    static func days(from weekStart: String, sessions: [PlanSession]) -> [PlanDay] {
        (0..<7).compactMap { offset in
            guard let date = DayString.shifted(weekStart, by: offset) else { return nil }
            return PlanDay(date: date, session: sessions.first { $0.date == date })
        }
    }
}

extension PlanSession {
    init(_ s: Components.Schemas.PlannedSession) {
        id = s.id
        date = s.date
        kind = s.kind.rawValue
        estMinutes = s.estMinutes
        // A status the app does not know yet reads as planned: the server
        // still answers a start of it.
        status = Status(rawValue: s.status) ?? .planned
        workoutSessionId = s.workoutSessionId
        checkInApplied = s.checkInApplied
        blocks = s.blocks.map(PlanBlock.init)
        reasons = s.reasons.map(PlanNote.init)
    }
}

extension PlanBlock {
    init(_ b: Components.Schemas.PlanBlock) {
        role = b.role.rawValue
        minutes = b.minutes
        paired = b.paired
        items = b.items.map(PlanItem.init)
        reasons = b.reasons.map(PlanNote.init)
    }
}

extension PlanItem {
    init(_ i: Components.Schemas.PlanItem) {
        id = i.id
        exercise = i.exercise
        exerciseName = i.exerciseName
        sets = i.sets
        reps = i.reps
        holdSeconds = i.holdS
        loadKg = i.loadKg
        assist = i.assist.rawValue
        reserve = i.reserve
        restSeconds = i.restS
        calibration = i.calibration
        offer = i.offer
        intensity = i.intensity.rawValue
        kind = i.kind.rawValue
        monitor = i.monitor
        reasons = i.reasons.map(PlanNote.init)
    }
}

extension PlanNote {
    init(_ r: Components.Schemas.PlanReason) {
        ruleId = r.ruleId
        text = r.text
        region = r.region
    }
}

/// Calendar days as the API writes them, YYYY-MM-DD.
enum DayString {
    /// `day` moved by `days`, counted in UTC so a time zone change cannot
    /// skip or repeat a day.
    static func shifted(_ day: String, by days: Int) -> String? {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(identifier: "UTC") ?? .gmt
        let parts = day.split(separator: "-").compactMap { Int($0) }
        guard parts.count == 3,
              let date = calendar.date(from: DateComponents(year: parts[0], month: parts[1], day: parts[2])),
              let moved = calendar.date(byAdding: .day, value: days, to: date)
        else { return nil }
        let c = calendar.dateComponents([.year, .month, .day], from: moved)
        return String(format: "%04d-%02d-%02d", c.year ?? 0, c.month ?? 0, c.day ?? 0)
    }
}
