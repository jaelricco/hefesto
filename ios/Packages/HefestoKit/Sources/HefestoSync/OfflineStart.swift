import Foundation
import GRDB
import HefestoAPI
import HefestoStore

/// Why a planned session could not start without the server.
public enum OfflineStartError: Error, Equatable {
    /// No plan is kept here for the session's week.
    case noPlan
    /// The session was started elsewhere, and its log session is not kept
    /// here: it takes the server to open it.
    case startedElsewhere
    /// An exercise of the session is not in the catalogue kept here.
    case unknownExercise(String)
}

extension AppDatabase {
    /// Starts a kept planned session without the server (spec §10.5): the
    /// draft is written here as the server writes it at a start (ADR 0016),
    /// and the sync takes it up like any session. The session names its
    /// planned session and each set its plan item; after the push the
    /// planner links the start to its plan and lets the draft follow the
    /// current one (ADR 0017). `day` is the session's date, which finds the
    /// kept plan. Returns the log session's id.
    ///
    /// There is no check-in: lightening a session (ADR 0019) takes the
    /// server. A session started here before is not started again.
    @discardableResult
    public func startPlannedSessionOffline(
        _ plannedSessionId: String, on day: String, timeZone: TimeZone = .current,
        id: String = UUIDv7.make(), at: Date = Date()
    ) throws -> String {
        guard let kept = try plan(on: day) else { throw OfflineStartError.noPlan }
        let plan = try kept.decoded()
        guard !plan.stopped else { throw PlanError.trainingStopped }
        guard let planned = plan.sessions.first(where: { $0.id == plannedSessionId }) else {
            throw PlanError.notFound
        }
        let started = try reader.read { db in
            try Session
                .filter(Column("plannedSessionId") == plannedSessionId && Column("deletedAt") == nil)
                .order(Column("startedAt"))
                .fetchOne(db)
        }
        if let started { return started.id }
        if let logId = planned.workoutSessionId {
            guard try reader.read({ try Session.filter(key: logId).filter(Column("deletedAt") == nil).fetchCount($0) }) > 0
            else { throw OfflineStartError.startedElsewhere }
            return logId
        }
        let exercises = try reader.read { db in
            Dictionary(try Exercise.fetchAll(db).map { ($0.slug, $0) }, uniquingKeysWith: { first, _ in first })
        }
        let session = Session(
            id: id, startedAt: at, timezone: timeZone.identifier, localDate: Self.localDay(at, in: timeZone),
            plannedSessionId: plannedSessionId, updatedAt: at)
        try saveNewSession(try PlanDraft.tree(planned, session: session, exercises: exercises), now: at)
        return id
    }

    static func localDay(_ date: Date, in zone: TimeZone) -> String {
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = zone
        let c = calendar.dateComponents([.year, .month, .day], from: date)
        return String(format: "%04d-%02d-%02d", c.year ?? 0, c.month ?? 0, c.day ?? 0)
    }
}

/// A planned session as the training log records it, built here as the
/// server builds it at a start (spec §10.2; `Materialize` in
/// internal/domain/planning/materialize.go). The planner matches the sets to
/// the plan by their items and counts the sets of each, so both sides must
/// agree: the same blocks, the same sets per item, in the same order.
enum PlanDraft {
    /// The largest reserves the log records: repetitions, and seconds of a
    /// hold.
    static let maxRIR = 10
    static let maxSIR = 60

    static func tree(
        _ planned: Components.Schemas.PlannedSession, session: Session, exercises: [String: Exercise],
        newID: () -> String = { UUIDv7.make() }
    ) throws -> SessionTree {
        var blocks: [BlockWithSets] = []
        for b in planned.blocks {
            // Offers are done only on the athlete's active choice; a block
            // without planned sets is not written.
            let items = b.items.filter { !$0.offer && $0.sets > 0 }
            guard !items.isEmpty else { continue }
            var block = Block(id: newID(), sessionId: session.id, orderIndex: blocks.count)
            var sets: [SetWithElements] = []
            if b.paired && items.count > 1 {
                // An antagonist pair alternates by round.
                let rounds = items.map(\.sets).max() ?? 0
                block.kind = "superset"
                block.roundsPlanned = rounds
                for round in 0..<rounds {
                    for item in items where round < item.sets {
                        sets.append(try set(item, round: round, in: block, exercises: exercises, newID: newID))
                    }
                }
            } else {
                for item in items {
                    for _ in 0..<item.sets {
                        sets.append(try set(item, round: nil, in: block, exercises: exercises, newID: newID))
                    }
                }
            }
            for i in sets.indices { sets[i].entry.orderIndex = i }
            blocks.append(BlockWithSets(block: block, sets: sets))
        }
        return SessionTree(session: session, blocks: blocks)
    }

    /// One planned set of an item: the targets in its one element, the
    /// target reserve and the planned rest on the set. A band stays in the
    /// plan item; the athlete names the band when logging the set.
    static func set(
        _ item: Components.Schemas.PlanItem, round: Int?, in block: Block, exercises: [String: Exercise],
        newID: () -> String
    ) throws -> SetWithElements {
        guard let exercise = exercises[item.exercise] else { throw OfflineStartError.unknownExercise(item.exercise) }
        let loadKg = item.loadKg ?? 0
        var element = SetElement(
            id: "", setEntryId: "", orderIndex: 0, exerciseId: exercise.id, measure: "reps", loadKg: loadKg,
            isEccentricOnly: item.stimulus == "eccentric", assistanceClass: loadKg > 0 ? "loaded" : "unassisted")
        if let hold = item.holdS, hold > 0 {
            element.measure = "hold_seconds"
            element.holdSeconds = Double(hold)
        } else if let reps = item.reps, reps > 0 {
            element.reps = reps
        } else if exercise.defaultMeasure == "hold_seconds" {
            element.measure = "hold_seconds"
        }
        var entry = SetEntry(
            id: newID(), sessionId: block.sessionId, blockId: block.id, orderIndex: 0, roundIndex: round,
            kind: item.kind.rawValue, isPlanned: true, restAfterPlannedS: item.restS, plannedItemId: item.id)
        if element.measure == "hold_seconds" {
            entry.sirS = min(max(item.reserve, 0), maxSIR)
        } else {
            entry.rir = min(max(item.reserve, 0), maxRIR)
        }
        element.id = newID()
        element.setEntryId = entry.id
        return SetWithElements(entry: entry, elements: [element])
    }
}
