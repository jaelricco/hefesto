import Foundation
#if canImport(Security)
import Security
#endif

/// The tokens of a signed-in device.
public struct Tokens: Codable, Sendable, Hashable {
    public var accessToken: String
    public var accessExpiresAt: Date
    public var refreshToken: String
    public var refreshExpiresAt: Date
    public var userId: String

    public init(accessToken: String, accessExpiresAt: Date, refreshToken: String, refreshExpiresAt: Date, userId: String) {
        self.accessToken = accessToken
        self.accessExpiresAt = accessExpiresAt
        self.refreshToken = refreshToken
        self.refreshExpiresAt = refreshExpiresAt
        self.userId = userId
    }
}

/// Where tokens live between launches.
public protocol TokenStore: Sendable {
    func load() -> Tokens?
    func save(_ tokens: Tokens?)
}

/// Tokens in memory, for tests and previews.
public final class InMemoryTokenStore: TokenStore, @unchecked Sendable {
    private let lock = NSLock()
    private var tokens: Tokens?

    public init(_ tokens: Tokens? = nil) { self.tokens = tokens }

    public func load() -> Tokens? { lock.withLock { tokens } }
    public func save(_ tokens: Tokens?) { lock.withLock { self.tokens = tokens } }
}

#if canImport(Security)
/// Tokens in the Keychain, readable after the first unlock so a background
/// sync can use them, and never synced to other devices.
public struct KeychainTokenStore: TokenStore {
    private let service: String
    private let account = "tokens"

    public init(service: String = "fit.hefesto.ios.auth") { self.service = service }

    public func load() -> Tokens? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]
        var item: CFTypeRef?
        guard SecItemCopyMatching(query as CFDictionary, &item) == errSecSuccess, let data = item as? Data else {
            return nil
        }
        return try? JSONDecoder().decode(Tokens.self, from: data)
    }

    public func save(_ tokens: Tokens?) {
        let base: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
        SecItemDelete(base as CFDictionary)
        guard let tokens, let data = try? JSONEncoder().encode(tokens) else { return }
        var add = base
        add[kSecValueData as String] = data
        add[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly
        SecItemAdd(add as CFDictionary, nil)
    }
}
#endif
