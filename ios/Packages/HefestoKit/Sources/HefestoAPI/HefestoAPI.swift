// The client and types are generated at build time from openapi.yaml (a
// symlink to api/openapi.yaml) by the OpenAPIGenerator plugin. This file only
// configures how they talk to the server.

import Foundation
import OpenAPIRuntime

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
