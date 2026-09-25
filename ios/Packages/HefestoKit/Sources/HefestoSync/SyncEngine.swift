import Foundation
import GRDB
import HefestoAPI
import HefestoStore

/// Why a sync stopped. Nothing is lost: queued ops stay queued, an unfinished
/// batch is resent under its key, and the cursor only moves forward.
public enum SyncError: Error, Equatable {
    /// The server refused the request as a whole (not one op of it).
    case refused(status: Int, type: String)
}

/// What one sync did, for the UI.
public struct SyncReport: Sendable {
    /// What completing a session did, per session id: the unlock celebration.
    public var completions: [String: Components.Schemas.CompletionResult] = [:]
    public var pushed = 0
    public var pulled = 0
}

/// Runs push-then-pull over the outbox and the local database (ADR 0010).
/// One sync at a time: a call during a sync waits for it and shares its
/// report.
public actor SyncEngine {
    private let client: Client
    private let db: AppDatabase
    private let newKey: @Sendable () -> String
    private var running: Task<SyncReport, any Error>?

    static let batchSize = 200
    static let pageSize = 500

    public init(client: Client, db: AppDatabase, newKey: @escaping @Sendable () -> String = { UUIDv7.make() }) {
        self.client = client
        self.db = db
        self.newKey = newKey
    }

    public func sync() async throws -> SyncReport {
        if let running { return try await running.value }
        let task = Task { try await self.run() }
        running = task
        defer { running = nil }
        return try await task.value
    }

    private func run() async throws -> SyncReport {
        var report = SyncReport()
        try await push(into: &report)
        try await pull(into: &report)
        return report
    }

    // MARK: push

    private func push(into report: inout SyncReport) async throws {
        while true {
            let (key, ops) = try nextBatch()
            guard !ops.isEmpty else { return }
            let sendable = ops.filter { $0.payload != nil }
            let decoder = HefestoAPIConfiguration.decoder()
            let body = try sendable.map { try decoder.decode(SyncOp.self, from: $0.payload!) }

            var outcomes: [Int64: OpOutcome] = [:]
            if !body.isEmpty {
                let out = try await client.pushChanges(headers: .init(idempotencyKey: key), body: .json(.init(ops: body)))
                let results: [Components.Schemas.SyncOpResult]
                switch out {
                case let .ok(r): results = try r.body.json.results
                case let .badRequest(r): throw Self.refused(400, try? r.body.applicationProblemJson)
                case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
                case let .conflict(r): throw Self.refused(409, try? r.body.applicationProblemJson)
                case let .unprocessableContent(r): throw Self.refused(422, try? r.body.applicationProblemJson)
                case let .undocumented(status, _): throw SyncError.refused(status: status, type: "")
                }
                for r in results where sendable.indices.contains(r.index) {
                    guard let seq = sendable[r.index].seq else { continue }
                    switch r.status.rawValue {
                    case "superseded":
                        outcomes[seq] = .superseded
                    case "rejected":
                        outcomes[seq] = .rejected(
                            type: r.problem?._type ?? "", detail: r.problem?.detail ?? r.problem?.title ?? "")
                    default:
                        outcomes[seq] = .applied
                        if let c = r.completion { report.completions[sendable[r.index].rowId] = c }
                    }
                }
            }
            let superseded = try db.finishBatch(key: key, outcomes: outcomes)
            report.pushed += body.count
            try await refetch(superseded)
        }
    }

    /// The batch in flight, resent as it was; or the next queued ops, frozen
    /// into a new batch with their current payloads. An op whose row is gone
    /// locally joins with no payload and is settled without being sent.
    private func nextBatch() throws -> (String, [OutboxOp]) {
        if let inFlight = try db.batchInFlight(), !inFlight.ops.isEmpty { return (inFlight.key, inFlight.ops) }
        let queued = try db.queuedOps(limit: Self.batchSize)
        guard !queued.isEmpty else { return ("", []) }
        let key = newKey()
        let payloads: [(seq: Int64, payload: Data?)] = try db.reader.read { conn in
            let encoder = HefestoAPIConfiguration.encoder()
            var out: [(seq: Int64, payload: Data?)] = []
            for op in queued {
                guard let seq = op.seq else { continue }
                out.append((seq: seq, payload: try OpPayload.build(op, conn).map { try encoder.encode($0) }))
            }
            return out
        }
        try db.startBatch(key: key, ops: payloads)
        let batch = try db.batchInFlight()
        return (key, batch?.ops ?? [])
    }

    /// Takes the server's copy of rows it refused to overwrite: the feed may
    /// already be past them, so they are fetched, not waited for.
    private func refetch(_ superseded: [OutboxOp]) async throws {
        var sessions: [String] = []
        for op in superseded {
            switch op.entity {
            case .bodyweight:
                try db.markBodyweightDeleted(id: op.rowId)
            case .session:
                if !sessions.contains(op.rowId) { sessions.append(op.rowId) }
            case .block, .set:
                if let s = op.sessionId, !sessions.contains(s) { sessions.append(s) }
            }
        }
        for id in sessions {
            switch try await client.getSession(path: .init(sessionId: id)) {
            case let .ok(r): try db.replaceSessionTree(id: id, with: ServerRows.tree(try r.body.json))
            case .notFound: try db.replaceSessionTree(id: id, with: nil)
            case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
            case let .undocumented(status, _): throw SyncError.refused(status: status, type: "")
            }
        }
    }

    // MARK: pull

    private func pull(into report: inout SyncReport) async throws {
        while true {
            let cursor = try db.cursor()
            let out = try await client.pullChanges(query: .init(cursor: cursor, limit: Self.pageSize))
            let page: Components.Schemas.SyncPage
            switch out {
            case let .ok(r): page = try r.body.json
            case let .badRequest(r): throw Self.refused(400, try? r.body.applicationProblemJson)
            case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
            case let .undocumented(status, _): throw SyncError.refused(status: status, type: "")
            }
            try db.apply(ServerRows.page(page))
            report.pulled += page.sessions.count + page.blocks.count + page.sets.count + page.bodyweight.count
            guard page.hasMore, page.cursor > cursor else { return }
        }
    }

    // MARK: content

    /// Refreshes the cached exercise catalogue when its content version
    /// changed. Content is not the athlete's, so it is not part of `sync()`.
    public func refreshExercises() async throws {
        let version = try db.contentVersion()
        let out = try await client.listExercises(query: .init(), headers: .init(ifNoneMatch: version.map { "\"\($0)\"" }))
        switch out {
        case let .ok(r):
            let list = try r.body.json
            try db.replaceExercises(list.items.map(ServerRows.exercise), contentVersion: list.contentVersion)
        case .notModified:
            return
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw SyncError.refused(status: status, type: "")
        }
    }

    private static func refused(_ status: Int, _ p: Components.Schemas.Problem?) -> SyncError {
        .refused(status: status, type: p?._type ?? "")
    }
}
