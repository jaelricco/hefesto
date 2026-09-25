import Foundation
import HefestoAPI
import HefestoAuth
import HefestoStore
import HefestoSync
import Network
import Observation

/// The app's composition root: it builds the database, the API client, auth
/// and sync once, and holds the state every screen shares. The logic lives
/// in HefestoKit; this only wires it and runs syncs (ADR 0010).
@MainActor
@Observable
final class AppModel {
    let db: AppDatabase
    let auth: AuthService
    let sync: SyncEngine

    private(set) var isSignedIn = false
    private(set) var isSyncing = false
    /// The last sync failed; the athlete's data is safe and will go later.
    private(set) var isOffline = false
    /// A completion the server evaluated, shown once as the celebration.
    var celebration: Celebration?

    private var pathMonitor: NWPathMonitor?

    init() throws {
        let baseURL = Self.apiBaseURL
        let dir = try FileManager.default.url(
            for: .applicationSupportDirectory, in: .userDomainMask, appropriateFor: nil, create: true)
        db = try AppDatabase.onDisk(at: dir.appending(path: "hefesto.sqlite"))

        auth = AuthService(
            client: HefestoAPIConfiguration.client(serverURL: baseURL),
            store: KeychainTokenStore(), deviceId: Self.deviceId,
            appVersion: Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String)
        let api = HefestoAPIConfiguration.client(serverURL: baseURL, middlewares: [AuthMiddleware(auth: auth)])
        sync = SyncEngine(client: api, db: db)
    }

    func start() async {
        isSignedIn = await auth.isSignedIn
        watchNetwork()
        await syncNow()
    }

    // MARK: auth

    func signedIn() async {
        isSignedIn = await auth.isSignedIn
        await syncNow()
    }

    func signOut() async {
        await auth.logout()
        isSignedIn = false
    }

    // MARK: sync

    /// One push-then-pull, and the catalogue if it changed. Failures leave
    /// everything queued; the next trigger tries again.
    func syncNow() async {
        guard isSignedIn else { return }
        isSyncing = true
        defer { isSyncing = false }
        do {
            let report = try await sync.sync()
            try? await sync.refreshExercises()
            isOffline = false
            if let done = report.completions.first {
                celebration = Celebration(sessionId: done.key, result: done.value)
            }
        } catch {
            // A refresh the server refused signs the device out; anything
            // else is the network, and everything stays queued.
            isSignedIn = await auth.isSignedIn
            isOffline = isSignedIn
        }
    }

    private func watchNetwork() {
        guard pathMonitor == nil else { return }
        let monitor = NWPathMonitor()
        // Called on the monitor's queue, so explicitly not main-actor isolated.
        monitor.pathUpdateHandler = { @Sendable [weak self] path in
            guard path.status == .satisfied else { return }
            Task { @MainActor in
                guard let self, self.isOffline else { return }
                await self.syncNow()
            }
        }
        monitor.start(queue: DispatchQueue(label: "fit.hefesto.network"))
        pathMonitor = monitor
    }

    // MARK: configuration

    static var apiBaseURL: URL {
        let raw = Bundle.main.object(forInfoDictionaryKey: "HefestoAPIBaseURL") as? String ?? ""
        return URL(string: raw) ?? URL(string: "http://localhost:8080")!
    }

    /// A stable id for this install. Not a secret: it names the device's
    /// tokens so signing out here leaves other devices signed in.
    static var deviceId: String {
        let key = "fit.hefesto.deviceId"
        if let id = UserDefaults.standard.string(forKey: key) { return id }
        let id = UUIDv7.make()
        UserDefaults.standard.set(id, forKey: key)
        return id
    }
}

struct Celebration: Identifiable {
    let sessionId: String
    let result: Components.Schemas.CompletionResult
    var id: String { sessionId }
}
