// The client and types are generated at build time from openapi.yaml (a
// symlink to api/openapi.yaml) by the OpenAPIGenerator plugin. This file only
// configures how they talk to the server.

import Foundation
import HTTPTypes
import OpenAPIRuntime
import OpenAPIURLSession

/// Timestamps as the server writes them: RFC 3339, with fractional seconds
/// when there are any (Go drops trailing zeros, so a whole second has none).
/// Dates are sent with milliseconds, so edits within a second keep their order.
public struct RFC3339DateTranscoder: DateTranscoder, @unchecked Sendable {
    private let lock = NSLock()
    private let fractional: ISO8601DateFormatter
    private let whole: ISO8601DateFormatter

    public init() {
        fractional = ISO8601DateFormatter()
        fractional.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        whole = ISO8601DateFormatter()
        whole.formatOptions = [.withInternetDateTime]
    }

    public func encode(_ date: Date) throws -> String {
        lock.withLock { fractional.string(from: date) }
    }

    public func decode(_ string: String) throws -> Date {
        let date = lock.withLock { fractional.date(from: string) ?? whole.date(from: string) }
        guard let date else {
            throw DecodingError.dataCorrupted(.init(codingPath: [], debugDescription: "Not an RFC 3339 timestamp: \(string)"))
        }
        return date
    }
}

public enum HefestoAPIConfiguration {
    public static let dates = RFC3339DateTranscoder()

    /// The configuration every `Client` of the app uses.
    public static let configuration = Configuration(dateTranscoder: dates, jsonEncodingOptions: [.sortedKeys])

    /// What every client of the app runs before its own middlewares.
    public static let middlewares: [any ClientMiddleware] = [EntityTagMiddleware()]

    /// A client of the API at `serverURL` over URLSession.
    public static func client(serverURL: URL, middlewares: [any ClientMiddleware] = []) -> Client {
        Client(serverURL: serverURL, configuration: configuration, transport: URLSessionTransport(),
               middlewares: Self.middlewares + middlewares)
    }

    /// An encoder that writes generated types exactly as the client does, so
    /// a payload stored for a retry is the payload that was sent.
    public static func encoder() -> JSONEncoder {
        let e = JSONEncoder()
        e.outputFormatting = [.sortedKeys]
        e.dateEncodingStrategy = .custom { date, encoder in
            var c = encoder.singleValueContainer()
            try c.encode(try dates.encode(date))
        }
        return e
    }

    public static func decoder() -> JSONDecoder {
        let d = JSONDecoder()
        d.dateDecodingStrategy = .custom { decoder in
            try dates.decode(try decoder.singleValueContainer().decode(String.self))
        }
        return d
    }
}

/// Sends `If-None-Match` as HTTP defines it. The generated client serialises
/// header parameters like URI components (RFC 6570), so the quotes of an
/// entity tag leave as `%22` and no `ETag` ever matches: every conditional
/// request would cost a full response.
public struct EntityTagMiddleware: ClientMiddleware {
    public init() {}

    public func intercept(
        _ request: HTTPRequest, body: HTTPBody?, baseURL: URL, operationID: String,
        next: @Sendable (HTTPRequest, HTTPBody?, URL) async throws -> (HTTPResponse, HTTPBody?)
    ) async throws -> (HTTPResponse, HTTPBody?) {
        var request = request
        if let tag = request.headerFields[.ifNoneMatch], let decoded = tag.removingPercentEncoding {
            request.headerFields[.ifNoneMatch] = decoded
        }
        return try await next(request, body, baseURL)
    }
}
