import Foundation
import GRDB

// What views observe. Each stream yields the current value, then a new one
// after every commit that changes it, whoever wrote it: the logger or sync.
// Views get plain values and never import GRDB.

extension AppDatabase {
    public func observeRecentSessions(limit: Int = 50) -> AsyncThrowingStream<[Session], any Error> {
        observe { try Self.recentSessions($0, limit: limit) }
    }

    public func observeSessionTree(id: String) -> AsyncThrowingStream<SessionTree?, any Error> {
        observe { try Self.sessionTree($0, id: id) }
    }

    public func observeExercises(matching query: String = "") -> AsyncThrowingStream<[Exercise], any Error> {
        observe { try Self.exercises($0, matching: query) }
    }

    public func observePendingOpCount() -> AsyncThrowingStream<Int, any Error> {
        observe { try Self.pendingOpCount($0) }
    }

    /// Exercises by id, for naming the elements of logged sets.
    public func exercisesById() throws -> [String: Exercise] {
        try reader.read { db in
            Dictionary(uniqueKeysWithValues: try Exercise.fetchAll(db).map { ($0.id, $0) })
        }
    }

    func observe<T: Sendable>(_ fetch: @escaping @Sendable (Database) throws -> T) -> AsyncThrowingStream<T, any Error> {
        AsyncThrowingStream { continuation in
            let cancellable = ValueObservation.tracking(fetch).start(
                in: reader,
                onError: { continuation.finish(throwing: $0) },
                onChange: { continuation.yield($0) })
            continuation.onTermination = { _ in cancellable.cancel() }
        }
    }
}
