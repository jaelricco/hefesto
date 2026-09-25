import Foundation
import GRDB
import HTTPTypes
import OpenAPIRuntime
import Testing
@testable import HefestoAPI
@testable import HefestoStore
@testable import HefestoSync

/// A server scripted per test: each request is recorded and answered by the
/// handler, or fails as a lost connection when the handler throws.
final class ScriptedServer: ClientTransport, @unchecked Sendable {
    struct Request {
        var operation: String
        var headers: HTTPFields
        var path: String
        var body: Data
        var json: [String: Any] { (try? JSONSerialization.jsonObject(with: body)) as? [String: Any] ?? [:] }
    }

    private let lock = NSLock()
    private var _requests: [Request] = []
    var handler: (Request) throws -> (Int, String)

    init(_ handler: @escaping (Request) throws -> (Int, String)) { self.handler = handler }

    var requests: [Request] { lock.withLock { _requests } }
    func requests(_ operation: String) -> [Request] { requests.filter { $0.operation == operation } }

    func send(_ request: HTTPRequest, body: HTTPBody?, baseURL: URL, operationID: String) async throws
        -> (HTTPResponse, HTTPBody?)
    {
        let data = if let body { try await Data(collecting: body, upTo: 1 << 20) } else { Data() }
        let r = Request(operation: operationID, headers: request.headerFields, path: request.path ?? "", body: data)
        lock.withLock { _requests.append(r) }
        let (status, json) = try lock.withLock { try handler(r) }
        var fields = HTTPFields()
        fields[.contentType] = status >= 400 ? "application/problem+json" : "application/json"
        return (HTTPResponse(status: .init(code: status), headerFields: fields), json.isEmpty ? nil : HTTPBody(json))
    }

    var client: Client {
        Client(serverURL: URL(string: "https://api.test")!, configuration: HefestoAPIConfiguration.configuration,
               transport: self)
    }
}

struct LostConnection: Error {}

final class Counter: @unchecked Sendable {
    private let lock = NSLock()
    private var n = 0
    func next() -> Int { lock.withLock { n += 1; return n } }
}

let zurich = "Europe/Zurich"

func results(_ statuses: [String]) -> String {
    let items = statuses.enumerated().map { i, s in
        s == "rejected"
            ? #"{"index":\#(i),"status":"rejected","problem":{"type":"https://hefesto.fit/problems/validation","title":"Invalid","status":422,"detail":"bad set"}}"#
            : #"{"index":\#(i),"status":"\#(s)"}"#
    }
    return #"{"results":[\#(items.joined(separator: ","))]}"#
}

/// A synchronous read: in an async test `reader.read` picks GRDB's async overload.
func fetch<T>(_ db: AppDatabase, _ value: (Database) throws -> T) throws -> T {
    try db.reader.read(value)
}

let emptyPage = #"{"cursor":0,"has_more":false,"sessions":[],"blocks":[],"sets":[],"bodyweight":[]}"#

/// Answers every push with `applied` for each op and every pull with nothing.
func applyingEverything(_ r: ScriptedServer.Request) -> (Int, String) {
    switch r.operation {
    case "pushChanges":
        let ops = r.json["ops"] as? [Any] ?? []
        return (200, results(Array(repeating: "applied", count: ops.count)))
    default:
        return (200, emptyPage)
    }
}

func seeded() throws -> (AppDatabase, Session, Block) {
    let db = try AppDatabase.inMemory()
    let session = Session(startedAt: Date(), timezone: zurich, localDate: "2026-09-25", title: "Pull day")
    try db.saveSession(session)
    let block = Block(sessionId: session.id, orderIndex: 0)
    try db.saveBlock(block)
    return (db, session, block)
}

func logSet(_ db: AppDatabase, _ session: Session, _ block: Block, reps: Int) throws -> SetEntry {
    let entry = SetEntry(sessionId: session.id, blockId: block.id, orderIndex: 0, completedAt: Date())
    let el = SetElement(setEntryId: entry.id, orderIndex: 0, exerciseId: UUIDv7.make(), measure: "reps", reps: reps)
    try db.saveSet(SetWithElements(entry: entry, elements: [el]))
    return entry
}

@Suite struct PushTests {
    @Test func sendsQueuedOpsParentsFirstUnderOneKey() async throws {
        let (db, session, block) = try seeded()
        _ = try logSet(db, session, block, reps: 8)
        let server = ScriptedServer(applyingEverything)
        let engine = SyncEngine(client: server.client, db: db, newKey: { "key-0001" })

        let report = try await engine.sync()

        let push = try #require(server.requests("pushChanges").first)
        #expect(push.headers[HTTPField.Name("Idempotency-Key")!] == "key-0001")
        let ops = try #require(push.json["ops"] as? [[String: Any]])
        #expect(ops.map { $0["entity"] as? String } == ["session", "block", "set"])
        #expect(ops.map { $0["op"] as? String } == ["put", "put", "put"])
        let sessionData = try #require(ops[0]["data"] as? [String: Any])
        #expect(sessionData["title"] as? String == "Pull day")
        #expect(sessionData["timezone"] as? String == zurich)
        let setData = try #require(ops[2]["data"] as? [String: Any])
        #expect(ops[2]["session_id"] as? String == session.id)
        let elements = try #require(setData["elements"] as? [[String: Any]])
        #expect(elements.count == 1, "a plain set is one set with one element")
        #expect(elements[0]["reps"] as? Int == 8)
        #expect(report.pushed == 3)
        #expect(try db.queuedOps().isEmpty)
        #expect(try db.batchInFlight()?.key == nil)
    }

    @Test func aLostResponseResendsTheSameBytesUnderTheSameKey() async throws {
        let (db, session, _) = try seeded()
        let keys = Counter()
        let server = ScriptedServer { _ in throw LostConnection() }
        let engine = SyncEngine(client: server.client, db: db, newKey: { "key-000\(keys.next())" })

        await #expect(throws: (any Error).self) { try await engine.sync() }
        #expect(try db.batchInFlight()?.key == "key-0001")

        // An edit made meanwhile waits for the next batch; the retry does not change.
        var s = try #require(try fetch(db) { try Session.fetchOne($0, key: session.id) })
        s.title = "Renamed"
        try db.saveSession(s)

        server.handler = applyingEverything
        _ = try await engine.sync()

        let pushes = server.requests("pushChanges")
        #expect(pushes.count == 3, "the lost push, its retry, then the edit")
        #expect(pushes[1].body == pushes[0].body)
        #expect(pushes[1].headers[HTTPField.Name("Idempotency-Key")!] == "key-0001")
        #expect(pushes[2].headers[HTTPField.Name("Idempotency-Key")!] == "key-0002")
        let renamed = try #require((pushes[2].json["ops"] as? [[String: Any]])?.first?["data"] as? [String: Any])
        #expect(renamed["title"] as? String == "Renamed")
        #expect(try db.queuedOps().isEmpty)
    }

    @Test func aRejectedOpIsKeptAsAProblem() async throws {
        let (db, session, block) = try seeded()
        let entry = try logSet(db, session, block, reps: 5)
        let server = ScriptedServer { r in
            r.operation == "pushChanges" ? (200, results(["applied", "applied", "rejected"])) : (200, emptyPage)
        }
        _ = try await SyncEngine(client: server.client, db: db).sync()

        let problems = try db.syncProblems()
        #expect(problems.count == 1)
        #expect(problems.first?.rowId == entry.id)
        #expect(problems.first?.detail == "bad set")
        #expect(try db.queuedOps().isEmpty)
    }

    @Test func aSupersededSetTakesTheServersCopyOfItsSession() async throws {
        let (db, session, block) = try seeded()
        let entry = try logSet(db, session, block, reps: 5)
        let server = ScriptedServer { r in
            switch r.operation {
            case "pushChanges": return (200, results(["applied", "applied", "superseded"]))
            case "getSession": return (200, sessionJSON(session, block, setId: entry.id, reps: 9))
            default: return (200, emptyPage)
            }
        }
        _ = try await SyncEngine(client: server.client, db: db).sync()

        #expect(server.requests("getSession").count == 1)
        let tree = try #require(try fetch(db) { try AppDatabase.sessionTree($0, id: session.id) })
        #expect(tree.blocks.first?.sets.first?.elements.first?.reps == 9, "the server's copy wins")
    }

    @Test func aSupersededWriteToADeletedSessionDeletesItHere() async throws {
        let (db, session, _) = try seeded()
        let server = ScriptedServer { r in
            switch r.operation {
            case "pushChanges": return (200, results(["superseded", "superseded"]))
            case "getSession":
                return (404, #"{"type":"https://hefesto.fit/problems/not-found","title":"Not found","status":404}"#)
            default: return (200, emptyPage)
            }
        }
        _ = try await SyncEngine(client: server.client, db: db).sync()

        #expect(server.requests("getSession").count == 1, "one fetch per session")
        #expect(try fetch(db) { try AppDatabase.sessionTree($0, id: session.id) } == nil)
    }

    @Test func completionReportsWhatItUnlocked() async throws {
        let (db, session, block) = try seeded()
        _ = try logSet(db, session, block, reps: 10)
        try db.completeSession(id: session.id, perceivedFatigue: 7)
        let server = ScriptedServer { r in
            guard r.operation == "pushChanges" else { return (200, emptyPage) }
            let ops = r.json["ops"] as? [[String: Any]] ?? []
            let items = ops.enumerated().map { i, op in
                op["op"] as? String == "complete"
                    ? #"{"index":\#(i),"status":"applied","completion":\#(completionJSON(session))}"#
                    : #"{"index":\#(i),"status":"applied"}"#
            }
            return (200, #"{"results":[\#(items.joined(separator: ","))]}"#)
        }
        let report = try await SyncEngine(client: server.client, db: db).sync()

        let ops = try #require(server.requests("pushChanges").first?.json["ops"] as? [[String: Any]])
        #expect(ops.last?["op"] as? String == "complete")
        #expect((ops.last?["data"] as? [String: Any])?["perceived_fatigue"] as? Int == 7)
        let completion = try #require(report.completions[session.id])
        #expect(completion.unlocked.map(\.levelName) == ["10 strict pull-ups"])
    }
}

@Suite struct PullTests {
    @Test func appliesPagesUntilTheFeedIsDrained() async throws {
        let db = try AppDatabase.inMemory()
        let sessionId = UUIDv7.make()
        let server = ScriptedServer { r in
            if r.path.contains("cursor=0") {
                return (200, #"{"cursor":5,"has_more":true,"sessions":[\#(syncSessionJSON(sessionId, seq: 5))],"blocks":[],"sets":[],"bodyweight":[]}"#)
            }
            return (200, #"{"cursor":7,"has_more":false,"sessions":[],"blocks":[],"sets":[],"bodyweight":[\#(bodyweightJSON(seq: 7))]}"#)
        }
        let report = try await SyncEngine(client: server.client, db: db).sync()

        #expect(server.requests("pullChanges").count == 2)
        #expect(try db.cursor() == 7)
        #expect(report.pulled == 2)
        let s = try #require(try fetch(db) { try Session.fetchOne($0, key: sessionId) })
        #expect(s.serverSeq == 5)
        #expect(s.startedAt.timeIntervalSince1970 == 1_789_997_600.25, "fractional seconds survive")
    }

    @Test func refreshesTheCatalogueOnlyWhenItChanged() async throws {
        let db = try AppDatabase.inMemory()
        let server = ScriptedServer { r in
            if r.headers[.ifNoneMatch] == #""v1""# { return (304, "") }
            return (200, #"{"content_version":"v1","items":[\#(exerciseJSON)]}"#)
        }
        let engine = SyncEngine(client: server.client, db: db)
        try await engine.refreshExercises()
        try await engine.refreshExercises()

        #expect(server.requests.count == 2)
        #expect(try db.contentVersion() == "v1")
        #expect(try fetch(db) { try AppDatabase.exercises($0) }.map(\.slug) == ["pull-up"])
    }
}

// MARK: - server payloads

func sessionJSON(_ s: Session, _ b: Block, setId: String, reps: Int) -> String {
    #"""
    {"id":"\#(s.id)","started_at":"2026-09-25T10:00:00Z","ended_at":null,"timezone":"\#(zurich)",
     "local_date":"2026-09-25","title":"Pull day","notes":"","perceived_fatigue":null,"bodyweight_kg":null,
     "status":"draft","is_rest_day":false,"template_id":null,"completed_at":null,
     "updated_at":"2026-09-25T10:00:00Z","server_updated_at":"2026-09-25T10:00:00Z",
     "blocks":[{"id":"\#(b.id)","order_index":0,"kind":"straight","rounds_planned":null,"rounds_done":null,
       "interval_s":null,"notes":"","updated_at":"2026-09-25T10:00:00Z",
       "sets":[{"id":"\#(setId)","block_id":"\#(b.id)","order_index":0,"round_index":null,"kind":"working",
         "is_planned":false,"rest_after_planned_s":null,"rest_after_actual_s":null,"rpe":null,"rir":null,
         "completed_at":"2026-09-25T10:05:00.5Z","notes":"","updated_at":"2026-09-25T10:05:00.5Z",
         "elements":[\#(elementJSON(reps: reps))]}]}]}
    """#
}

func elementJSON(reps: Int) -> String {
    #"""
    {"id":"\#(UUIDv7.make())","order_index":0,"exercise_id":"\#(UUIDv7.make())","measure":"reps","reps":\#(reps),
     "hold_seconds":null,"distance_m":null,"tempo":null,"load_kg":0,"is_eccentric_only":false,
     "is_partial_rom":false,"rom_note":null,"form_quality":null,"failed":false,
     "assistance_class":"unassisted","assistance":null,"media_ids":[]}
    """#
}

func syncSessionJSON(_ id: String, seq: Int) -> String {
    #"""
    {"id":"\#(id)","started_at":"2026-09-21T13:33:20.25Z","ended_at":null,"timezone":"\#(zurich)",
     "local_date":"2026-09-21","title":"","notes":"","perceived_fatigue":null,"bodyweight_kg":null,
     "status":"draft","is_rest_day":false,"template_id":null,"completed_at":null,
     "updated_at":"2026-09-21T13:33:20.25Z","server_updated_at":"2026-09-21T13:33:21Z",
     "server_seq":\#(seq),"deleted_at":null}
    """#
}

func bodyweightJSON(seq: Int) -> String {
    #"""
    {"id":"\#(UUIDv7.make())","measured_at":"2026-09-21T07:00:00Z","local_date":"2026-09-21",
     "bodyweight_kg":72.5,"note":"","updated_at":"2026-09-21T07:00:00Z","server_seq":\#(seq),"deleted_at":null}
    """#
}

func completionJSON(_ s: Session) -> String {
    #"""
    {"session":\#(sessionJSON(s, Block(sessionId: s.id, orderIndex: 0), setId: UUIDv7.make(), reps: 10)),
     "already_completed":false,
     "unlocked":[{"level_id":"\#(UUIDv7.make())","skill_slug":"pull-up","skill_name":"Pull-up",
       "level_slug":"ten-strict","level_name":"10 strict pull-ups","verification":"auto",
       "occurred_at":"2026-09-25T10:30:00Z","evidence_set_entry_id":null}],
     "newly_available":[],"xp_awarded":[{"source":"session_completed","amount":10}],"xp_total":10,
     "streak":{"current_days":1,"longest_days":1,"freeze_credits":0,"last_counted_date":"2026-09-25"}}
    """#
}

let exerciseJSON = #"""
    {"id":"\#(UUIDv7.make())","slug":"pull-up","name":"Pull-up","aka":[],"family":"pull","default_measure":"reps",
     "load_semantics":"added","is_bodyweight":true,"unilateral":false,"tempo_applicable":true,"equipment":["bar"],
     "summary":"","cues":[],"common_faults":[],"status":"active"}
    """#
