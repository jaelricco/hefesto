import Foundation
import HefestoStore
import HefestoSync

// The questions about pain around a planned session (spec §8.6, ADR 0030):
// the pain now before it, during it and after it once it is done, and the
// morning after. Each asks about the regions in the ramp the session names.
// They are neutral and voluntary: a question not answered passes with its
// moment, and the ramp waits for a report (`PAR-S-34`).

/// When a pain question is asked.
public enum PainMoment: Sendable, Hashable {
    /// Before the session, with the check-in: the pain now.
    case beforeSession
    /// Once it is done: the pain during it, and now.
    case afterSession
    /// The morning after: the pain now, and whether the pain after the
    /// session lasted longer than an hour.
    case nextMorning
}

/// A question about the pain in some regions, around one session.
public struct PainQuestion: Sendable, Hashable, Identifiable {
    public var moment: PainMoment
    /// The log session the answers belong to.
    public var sessionId: String
    public var regions: [PlanRegion]

    public var id: String { "\(sessionId)|\(moment)" }
}

/// What the athlete answers about one region. Every answer is optional.
public struct PainAnswer: Sendable, Hashable {
    /// The pain now, 0 to 10: before the session, after it or in the morning.
    public var now: Int?
    /// The pain during the session, 0 to 10.
    public var during: Int?
    /// The pain after the session lasted longer than an hour.
    public var lastedOver1h: Bool?

    public init(now: Int? = nil, during: Int? = nil, lastedOver1h: Bool? = nil) {
        self.now = now
        self.during = during
        self.lastedOver1h = lastedOver1h
    }
}

extension PainQuestion {
    /// The reports the answers make, one per value given. A region without
    /// an answer sends nothing. The morning's «longer than an hour» goes with
    /// its report: the planner counts it for the session whichever of the
    /// session's reports carries it.
    public func reports(_ answers: [String: PainAnswer], at: Date) -> [PainReport] {
        regions.flatMap { region -> [PainReport] in
            guard let a = answers[region.id] else { return [] }
            func report(_ timepoint: String, _ nrs: Int?, lastedOver1h: Bool? = nil) -> PainReport? {
                nrs.map {
                    PainReport(region: region.id, timepoint: timepoint, nrs: Double($0), at: at,
                               lastedOver1h: lastedOver1h, sessionId: sessionId)
                }
            }
            return switch moment {
            case .beforeSession: [report("before_session", a.now)].compactMap { $0 }
            case .afterSession: [report("during", a.during), report("after", a.now)].compactMap { $0 }
            case .nextMorning: [report("next_morning", a.now, lastedOver1h: a.lastedOver1h)].compactMap { $0 }
            }
        }
    }
}

extension PlanModel {
    /// The regions to ask about before a planned session starts.
    public func painRegions(for session: PlanSession) -> [PlanRegion] {
        week?.painRegions(for: session) ?? []
    }

    /// The question before a session, once its start returned the log
    /// session the answers belong to.
    public func beforeQuestion(for session: PlanSession, logSessionId: String) -> PainQuestion? {
        question(.beforeSession, logSessionId: logSessionId, regions: painRegions(for: session))
    }

    /// The question once a log session started from the plan is done: the
    /// regions not yet answered for it.
    public func afterQuestion(logSessionId: String) -> PainQuestion? {
        guard let session = try? db.session(id: logSessionId),
              let regions = regions(of: session) else { return nil }
        return question(.afterSession, logSessionId: logSessionId, regions: regions)
    }

    /// The morning question about yesterday's last session from the plan,
    /// while some region is not answered for this morning. A day later it
    /// is gone: the question is not asked again.
    public func morningQuestion() -> PainQuestion? {
        guard let yesterday = DayString.shifted(today, by: -1),
              let session = (try? db.completedPlannedSessions(on: yesterday))?.first,
              let regions = regions(of: session) else { return nil }
        return question(.nextMorning, logSessionId: session.id, regions: regions)
    }

    /// The regions the planned session of a log session names.
    private func regions(of session: Session) -> [PlanRegion]? {
        guard let planned = session.plannedSessionId,
              let cached = try? db.keptPlan(on: session.localDate).plan,
              let week = try? PlanWeek(decoding: cached),
              let ps = week.session(id: planned) else { return nil }
        return week.painRegions(for: ps)
    }

    /// The question, without the regions already answered for it.
    private func question(_ moment: PainMoment, logSessionId: String, regions: [PlanRegion]) -> PainQuestion? {
        let timepoint = switch moment {
        case .beforeSession: "before_session"
        case .afterSession: "after"
        case .nextMorning: "next_morning"
        }
        let answered = Set(((try? db.painReports(sessionId: logSessionId)) ?? [])
            .filter { $0.timepoint == timepoint || moment == .afterSession && $0.timepoint == "during" }
            .map(\.region))
        let open = regions.filter { !answered.contains($0.id) }
        return open.isEmpty ? nil : PainQuestion(moment: moment, sessionId: logSessionId, regions: open)
    }
}
