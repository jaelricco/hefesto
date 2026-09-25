import Foundation
import HefestoAPI
import OpenAPIRuntime

/// Why signing in or refreshing failed, in terms the UI can explain.
public enum AuthError: Error, Equatable, Sendable {
    case invalidCredentials
    case emailTaken
    case invalid(String)
    case rateLimited
    case forbidden
    /// The session ended (refresh token expired, revoked or reused): sign in again.
    case signedOut
    case server(Int)
}

/// Signs the device in and keeps a valid access token (ADR 0010). One actor,
/// so concurrent requests that need a refresh wait for the same one.
public actor AuthService {
    private let client: Client // without the auth middleware: these calls carry no token
    private let store: any TokenStore
    private let device: Components.Schemas.Device
    private let now: @Sendable () -> Date
    private var refreshing: Task<Tokens, any Error>?

    /// Refresh this long before the access token expires.
    static let refreshMargin: TimeInterval = 60

    public init(client: Client, store: any TokenStore, deviceId: String, appVersion: String? = nil,
                now: @escaping @Sendable () -> Date = { Date() }) {
        self.client = client
        self.store = store
        self.device = .init(id: deviceId, platform: .ios, appVersion: appVersion)
        self.now = now
    }

    public var isSignedIn: Bool { store.load() != nil }
    public var userId: String? { store.load()?.userId }

    public func register(email: String, password: String, displayName: String?, timezone: String, locale: String?) async throws {
        let out = try await client.register(body: .json(.init(
            email: email, password: password, displayName: displayName, locale: locale,
            timezone: timezone, device: device)))
        switch out {
        case let .created(r): try keep(r.body.json)
        case .conflict: throw AuthError.emailTaken
        case let .unprocessableContent(r): throw AuthError.invalid(Self.detail(try? r.body.applicationProblemJson))
        case .tooManyRequests: throw AuthError.rateLimited
        case let .undocumented(status, _): throw AuthError.server(status)
        default: throw AuthError.server(0)
        }
    }

    public func login(email: String, password: String) async throws {
        let out = try await client.login(body: .json(.init(email: email, password: password, device: device)))
        switch out {
        case let .ok(r): try keep(r.body.json)
        case .unauthorized: throw AuthError.invalidCredentials
        case .forbidden: throw AuthError.forbidden
        case .tooManyRequests: throw AuthError.rateLimited
        case let .undocumented(status, _): throw AuthError.server(status)
        default: throw AuthError.server(0)
        }
    }

    public func signInWithApple(identityToken: String, rawNonce: String?, displayName: String?) async throws {
        let out = try await client.signInWithApple(body: .json(.init(
            identityToken: identityToken, nonce: rawNonce, displayName: displayName, device: device)))
        switch out {
        case let .ok(r): try keep(r.body.json)
        case let .created(r): try keep(r.body.json) // first sign-in created the account
        case .unauthorized: throw AuthError.invalidCredentials
        case .forbidden: throw AuthError.forbidden
        case .tooManyRequests: throw AuthError.rateLimited
        case let .undocumented(status, _): throw AuthError.server(status)
        default: throw AuthError.server(0)
        }
    }

    /// Signs this device out: the server revokes its tokens, and they are
    /// forgotten here whatever the network says.
    public func logout() async {
        if let t = store.load() {
            _ = try? await client.logout(body: .json(.init(refreshToken: t.refreshToken)))
        }
        store.save(nil)
    }

    /// A valid access token, refreshed first if it expires within a minute.
    public func accessToken() async throws -> String {
        guard let t = store.load() else { throw AuthError.signedOut }
        if t.accessExpiresAt.timeIntervalSince(now()) > Self.refreshMargin { return t.accessToken }
        return try await refresh().accessToken
    }

    /// Refreshes after the server refused an access token.
    public func accessTokenAfterUnauthorized(rejected: String) async throws -> String {
        if let t = store.load(), t.accessToken != rejected { return t.accessToken } // another request refreshed already
        return try await refresh().accessToken
    }

    /// One refresh at a time; callers during a refresh share its result.
    private func refresh() async throws -> Tokens {
        if let refreshing { return try await refreshing.value }
        guard let current = store.load() else { throw AuthError.signedOut }
        let task = Task { [client] () throws -> Tokens in
            let out = try await client.refreshTokens(body: .json(.init(refreshToken: current.refreshToken)))
            switch out {
            case let .ok(r): return try Self.tokens(r.body.json, now: self.now())
            case .unauthorized, .forbidden: throw AuthError.signedOut
            case .tooManyRequests: throw AuthError.rateLimited
            case let .undocumented(status, _): throw AuthError.server(status)
            default: throw AuthError.server(0)
            }
        }
        refreshing = task
        defer { refreshing = nil }
        do {
            let fresh = try await task.value
            store.save(fresh)
            return fresh
        } catch AuthError.signedOut {
            store.save(nil)
            throw AuthError.signedOut
        }
    }

    private func keep(_ r: Components.Schemas.AuthResponse) throws {
        store.save(Self.tokens(r, now: now()))
    }

    static func tokens(_ r: Components.Schemas.AuthResponse, now: Date) -> Tokens {
        Tokens(
            accessToken: r.accessToken,
            accessExpiresAt: now.addingTimeInterval(TimeInterval(r.expiresIn)),
            refreshToken: r.refreshToken,
            refreshExpiresAt: r.refreshExpiresAt,
            userId: r.user.id)
    }

    static func detail(_ p: Components.Schemas.Problem?) -> String {
        guard let p else { return "" }
        if let errors = p.errors?.additionalProperties, !errors.isEmpty {
            return errors.sorted { $0.key < $1.key }.map { "\($0.key): \($0.value)" }.joined(separator: "\n")
        }
        return p.detail ?? p.title
    }
}
