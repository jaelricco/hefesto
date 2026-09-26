import Foundation
import GRDB

// History and per-exercise stats, computed from the local database. The
// device holds every set the athlete logged (sync pulls them all), so stats
// work offline and need no endpoint of their own (ADR 0011).

/// One ISO week of sessions, Monday first.
public struct HistoryWeek: Sendable, Hashable, Identifiable {
    /// The Monday that starts the week, as YYYY-MM-DD.
    public var monday: String
    public var year: Int
    public var week: Int
    public var sessions: [Session]
    public var id: String { monday }

    /// Sessions that were training, not planned rest.
    public var trainingCount: Int { sessions.filter { !$0.isRestDay }.count }
}

public enum History {
    /// Groups sessions by the ISO week of their local date, newest week first
    /// and newest session first within a week.
    public static func weeks(_ sessions: [Session]) -> [HistoryWeek] {
        var calendar = Calendar(identifier: .iso8601)
        calendar.timeZone = TimeZone(identifier: "UTC")!
        var byMonday: [String: HistoryWeek] = [:]
        for s in sessions {
            guard let day = Day.date(s.localDate, calendar: calendar),
                  let interval = calendar.dateInterval(of: .weekOfYear, for: day)
            else { continue }
            let monday = Day.string(interval.start, calendar: calendar)
            let c = calendar.dateComponents([.yearForWeekOfYear, .weekOfYear], from: day)
            byMonday[monday, default: HistoryWeek(
                monday: monday, year: c.yearForWeekOfYear ?? 0, week: c.weekOfYear ?? 0, sessions: [])
            ].sessions.append(s)
        }
        return byMonday.values
            .map { w in
                var w = w
                w.sessions.sort { ($0.localDate, $0.startedAt) > ($1.localDate, $1.startedAt) }
                return w
            }
            .sorted { $0.monday > $1.monday }
    }
}

/// One day's work on an exercise.
public struct ExerciseDay: Sendable, Hashable, Identifiable {
    public var localDate: String
    public var sets: Int
    public var totalReps: Int
    public var bestReps: Int?
    public var bestHoldSeconds: Double?
    public var bestDistanceM: Double?
    public var maxLoadKg: Double?
    public var id: String { localDate }
}

/// A best effort and when it happened.
public struct PersonalBest: Sendable, Hashable {
    public var value: Double
    public var localDate: String
}

/// What the athlete has done with one exercise over time.
public struct ExerciseStats: Sendable, Hashable {
    public var exerciseId: String
    /// Oldest first, for a chart.
    public var days: [ExerciseDay]
    /// Bests count only full, unassisted repetitions: no partial range, no
    /// eccentric-only, no assistance. Added load still counts.
    public var bestReps: PersonalBest?
    public var bestHoldSeconds: PersonalBest?
    public var bestDistanceM: PersonalBest?
    public var maxLoadKg: PersonalBest?

    public var isEmpty: Bool { days.isEmpty }
}

/// An exercise the athlete has logged, for choosing whose stats to show.
public struct LoggedExercise: Sendable, Hashable, Identifiable {
    public var exercise: Exercise
    public var sets: Int
    public var lastDate: String
    public var id: String { exercise.id }
}

extension AppDatabase {
    /// Live sessions for history, newest first.
    public static func historySessions(_ db: Database) throws -> [Session] {
        try Session.filter(Column("deletedAt") == nil && Column("status") != "abandoned")
            .order(Column("localDate").desc, Column("startedAt").desc).fetchAll(db)
    }

    public func observeHistory() -> AsyncThrowingStream<[HistoryWeek], any Error> {
        observe { History.weeks(try Self.historySessions($0)) }
    }

    /// Exercises with at least one performed set, most recently used first.
    public static func loggedExercises(_ db: Database) throws -> [LoggedExercise] {
        let rows = try Row.fetchAll(db, sql: """
            SELECT el.exerciseId AS exerciseId, COUNT(DISTINCT se.id) AS sets, MAX(s.localDate) AS lastDate
            FROM setElement el
            JOIN setEntry se ON se.id = el.setEntryId
            JOIN session s ON s.id = se.sessionId
            WHERE se.deletedAt IS NULL AND s.deletedAt IS NULL AND NOT se.isPlanned
            GROUP BY el.exerciseId
            ORDER BY lastDate DESC
            """)
        let exercises = Dictionary(uniqueKeysWithValues: try Exercise.fetchAll(db).map { ($0.id, $0) })
        return rows.compactMap { row in
            let id: String = row["exerciseId"]
            guard let e = exercises[id] else { return nil }
            return LoggedExercise(exercise: e, sets: row["sets"], lastDate: row["lastDate"])
        }
    }

    public func observeLoggedExercises() -> AsyncThrowingStream<[LoggedExercise], any Error> {
        observe { try Self.loggedExercises($0) }
    }

    public static func exerciseStats(_ db: Database, exerciseId: String) throws -> ExerciseStats {
        let rows = try Row.fetchAll(db, sql: """
            SELECT s.localDate AS localDate, se.id AS setId, el.reps AS reps, el.holdSeconds AS holdSeconds,
                   el.distanceM AS distanceM, el.loadKg AS loadKg, el.assistance IS NOT NULL AS assisted,
                   el.isPartialRom AS partial, el.isEccentricOnly AS eccentric
            FROM setElement el
            JOIN setEntry se ON se.id = el.setEntryId
            JOIN session s ON s.id = se.sessionId
            WHERE el.exerciseId = ? AND se.deletedAt IS NULL AND s.deletedAt IS NULL AND NOT se.isPlanned
            ORDER BY s.localDate
            """, arguments: [exerciseId])
        return ExerciseStats.compute(exerciseId: exerciseId, rows: rows.map {
            StatRow(localDate: $0["localDate"], setId: $0["setId"], reps: $0["reps"], holdSeconds: $0["holdSeconds"],
                    distanceM: $0["distanceM"], loadKg: $0["loadKg"], assisted: $0["assisted"],
                    partial: $0["partial"], eccentric: $0["eccentric"])
        })
    }

    public func observeExerciseStats(exerciseId: String) -> AsyncThrowingStream<ExerciseStats, any Error> {
        observe { try Self.exerciseStats($0, exerciseId: exerciseId) }
    }
}

/// One logged element of the exercise, flattened for the stats.
struct StatRow {
    var localDate: String
    var setId: String
    var reps: Int?
    var holdSeconds: Double?
    var distanceM: Double?
    var loadKg: Double
    var assisted: Bool
    var partial: Bool
    var eccentric: Bool

    /// Whether this element counts towards a best.
    var isClean: Bool { !assisted && !partial && !eccentric }
}

extension ExerciseStats {
    static func compute(exerciseId: String, rows: [StatRow]) -> ExerciseStats {
        var stats = ExerciseStats(exerciseId: exerciseId, days: [])
        func better(_ best: PersonalBest?, _ value: Double?, _ date: String) -> PersonalBest? {
            guard let value, value > 0 else { return best }
            guard let best else { return PersonalBest(value: value, localDate: date) }
            return value > best.value ? PersonalBest(value: value, localDate: date) : best
        }
        for (date, dayRows) in Dictionary(grouping: rows, by: \.localDate).sorted(by: { $0.key < $1.key }) {
            let clean = dayRows.filter(\.isClean)
            stats.days.append(ExerciseDay(
                localDate: date,
                sets: Set(dayRows.map(\.setId)).count,
                totalReps: dayRows.compactMap(\.reps).reduce(0, +),
                bestReps: clean.compactMap(\.reps).max(),
                bestHoldSeconds: clean.compactMap(\.holdSeconds).max(),
                bestDistanceM: clean.compactMap(\.distanceM).max(),
                maxLoadKg: clean.map(\.loadKg).filter { $0 > 0 }.max()))
            for r in clean {
                stats.bestReps = better(stats.bestReps, r.reps.map(Double.init), date)
                stats.bestHoldSeconds = better(stats.bestHoldSeconds, r.holdSeconds, date)
                stats.bestDistanceM = better(stats.bestDistanceM, r.distanceM, date)
                stats.maxLoadKg = better(stats.maxLoadKg, r.loadKg, date)
            }
        }
        return stats
    }
}

enum Day {
    static func date(_ s: String, calendar: Calendar) -> Date? {
        let parts = s.split(separator: "-").compactMap { Int($0) }
        guard parts.count == 3 else { return nil }
        return calendar.date(from: DateComponents(year: parts[0], month: parts[1], day: parts[2]))
    }

    static func string(_ d: Date, calendar: Calendar) -> String {
        let c = calendar.dateComponents([.year, .month, .day], from: d)
        return String(format: "%04d-%02d-%02d", c.year ?? 0, c.month ?? 0, c.day ?? 0)
    }
}
