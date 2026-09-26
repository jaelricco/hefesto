import Foundation
import HTTPTypes
import OpenAPIRuntime

/// Adds the access token to every request that needs one, and on a 401
/// refreshes once and retries (ADR 0010).
public struct AuthMiddleware: ClientMiddleware {
    private let auth: AuthService
    /// Operations that carry no access token.
    static let public_: Set<String> = [
        "getHealth", "getReadiness", "register", "login", "signInWithApple", "refreshTokens", "logout",
    ]

    public init(auth: AuthService) { self.auth = auth }

    public func intercept(
        _ request: HTTPRequest, body: HTTPBody?, baseURL: URL, operationID: String,
        next: @Sendable (HTTPRequest, HTTPBody?, URL) async throws -> (HTTPResponse, HTTPBody?)
    ) async throws -> (HTTPResponse, HTTPBody?) {
        guard !Self.public_.contains(operationID) else { return try await next(request, body, baseURL) }

        // The body may be sent twice, so read it once.
        let bytes: Data? = if let body { try await Data(collecting: body, upTo: 8 << 20) } else { nil }
        let token = try await auth.accessToken()
        var first = request
        first.headerFields[.authorization] = "Bearer \(token)"
        let (response, responseBody) = try await next(first, bytes.map { HTTPBody($0) }, baseURL)
        guard response.status == .unauthorized else { return (response, responseBody) }

        let fresh = try await auth.accessTokenAfterUnauthorized(rejected: token)
        var retry = request
        retry.headerFields[.authorization] = "Bearer \(fresh)"
        return try await next(retry, bytes.map { HTTPBody($0) }, baseURL)
    }
}
