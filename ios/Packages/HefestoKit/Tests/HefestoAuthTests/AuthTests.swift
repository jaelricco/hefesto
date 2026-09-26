import Foundation
import HTTPTypes
import OpenAPIRuntime
import Testing
@testable import HefestoAPI
@testable import HefestoAuth

/// A server scripted per test, answering by operation id and recording each
/// request. Refreshes are slow on purpose, so concurrent callers overlap.
final class AuthServer: ClientTransport, @unchecked Sendable {
    private let lock = NSLock()
    private var log: [(op: String, auth: String?)] = []
    var validToken = "access-1"
    var refreshStatus = 200
    var issued = 1

    func calls(_ op: String) -> [(op: String, auth: String?)] { lock.withLock { log.filter { $0.op == op } } }

    func send(_ request: HTTPRequest, body: HTTPBody?, baseURL: URL, operationID: String) async throws
        -> (HTTPResponse, HTTPBody?)
    {
        if let body { _ = try await Data(collecting: body, upTo: 1 << 20) }
        lock.withLock { log.append((operationID, request.headerFields[.authorization])) }
        switch operationID {
        case "refreshTokens":
            try await Task.sleep(for: .milliseconds(50))
            let (status, token): (Int, String) = lock.withLock {
                guard refreshStatus == 200 else { return (refreshStatus, "") }
                issued += 1
                validToken = "access-\(issued)"
                return (200, validToken)
            }
            if status != 200 { return Self.problem(status) }
            return Self.json(200, authJSON(access: token, refresh: "refresh-\(token)"))
        case "login":
            return Self.json(200, authJSON(access: lock.withLock { validToken }, refresh: "refresh-1"))
        default:
            let ok = lock.withLock { request.headerFields[.authorization] == "Bearer \(validToken)" }
            return ok ? Self.json(200, userJSON) : Self.problem(401)
        }
    }

    static func json(_ status: Int, _ body: String) -> (HTTPResponse, HTTPBody?) {
        (HTTPResponse(status: .init(code: status), headerFields: [.contentType: "application/json"]), HTTPBody(body))
    }

    static func problem(_ status: Int) -> (HTTPResponse, HTTPBody?) {
        (HTTPResponse(status: .init(code: status), headerFields: [.contentType: "application/problem+json"]),
         HTTPBody(#"{"type":"about:blank","title":"No","status":\#(status)}"#))
    }
}

let userJSON = #"""
    {"id":"0190f0e0-0000-7000-8000-000000000001","email":"a@hefesto.fit","email_verified":false,
     "display_name":"A","locale":"en","unit_system":"metric","week_start":1,"timezone":"Europe/Zurich",
     "status":"active","deletion_requested_at":null,"created_at":"2026-09-01T00:00:00Z"}
    """#

func authJSON(access: String, refresh: String) -> String {
    #"""
    {"access_token":"\#(access)","token_type":"Bearer","expires_in":900,"refresh_token":"\#(refresh)",
     "refresh_expires_at":"2026-12-01T00:00:00Z","user":\#(userJSON)}
    """#
}

final class Clock: @unchecked Sendable {
    private let lock = NSLock()
    private var t = Date(timeIntervalSince1970: 1_790_000_000)
    var now: Date { lock.withLock { t } }
    func advance(_ s: TimeInterval) { lock.withLock { t += s } }
}

struct Fixture {
    let server = AuthServer()
    let store = InMemoryTokenStore()
    let clock = Clock()
    let auth: AuthService
    let api: Client

    init() {
        let url = URL(string: "https://api.test")!
        let config = HefestoAPIConfiguration.configuration
        let clock = self.clock
        auth = AuthService(
            client: Client(serverURL: url, configuration: config, transport: server),
            store: store, deviceId: "0190f0e0-0000-7000-8000-00000000000d", now: { clock.now })
        api = Client(serverURL: url, configuration: config, transport: server,
                     middlewares: [AuthMiddleware(auth: auth)])
    }
}

@Suite struct AuthServiceTests {
    @Test func loginKeepsTheTokens() async throws {
        let f = Fixture()
        try await f.auth.login(email: "a@hefesto.fit", password: "correct horse")
        #expect(f.store.load()?.accessToken == "access-1")
        #expect(f.store.load()?.userId == "0190f0e0-0000-7000-8000-000000000001")
        #expect(f.store.load()?.accessExpiresAt == f.clock.now.addingTimeInterval(900))
        #expect(await f.auth.isSignedIn)
    }

    @Test func aFreshTokenIsUsedAsIs() async throws {
        let f = Fixture()
        try await f.auth.login(email: "a@hefesto.fit", password: "correct horse")
        #expect(try await f.auth.accessToken() == "access-1")
        #expect(f.server.calls("refreshTokens").isEmpty)
    }

    @Test func concurrentCallersNearExpiryShareOneRefresh() async throws {
        let f = Fixture()
        try await f.auth.login(email: "a@hefesto.fit", password: "correct horse")
        f.clock.advance(900 - 30) // inside the refresh margin

        let tokens = try await withThrowingTaskGroup(of: String.self) { group in
            for _ in 0..<5 { group.addTask { try await f.auth.accessToken() } }
            return try await group.reduce(into: []) { $0.append($1) }
        }
        #expect(Set(tokens) == ["access-2"])
        #expect(f.server.calls("refreshTokens").count == 1)
        #expect(f.store.load()?.refreshToken == "refresh-access-2", "the rotated refresh token is kept")
    }

    @Test func aDeadRefreshTokenSignsTheDeviceOut() async throws {
        let f = Fixture()
        try await f.auth.login(email: "a@hefesto.fit", password: "correct horse")
        f.clock.advance(900)
        f.server.refreshStatus = 401

        await #expect(throws: AuthError.signedOut) { try await f.auth.accessToken() }
        #expect(f.store.load() == nil)
        #expect(await !f.auth.isSignedIn)
    }
}

@Suite struct AuthMiddlewareTests {
    @Test func requestsCarryTheAccessToken() async throws {
        let f = Fixture()
        try await f.auth.login(email: "a@hefesto.fit", password: "correct horse")
        _ = try await f.api.getMe()
        #expect(f.server.calls("getMe").map(\.auth) == ["Bearer access-1"])
        #expect(f.server.calls("login").map(\.auth) == [nil], "sign-in carries no token")
    }

    @Test func aRefusedTokenIsRefreshedOnceAndTheRequestRetried() async throws {
        let f = Fixture()
        try await f.auth.login(email: "a@hefesto.fit", password: "correct horse")
        f.server.validToken = "revoked-elsewhere" // the server no longer takes access-1

        _ = try await f.api.getMe()
        #expect(f.server.calls("getMe").map(\.auth) == ["Bearer access-1", "Bearer access-2"])
        #expect(f.server.calls("refreshTokens").count == 1)
    }
}
