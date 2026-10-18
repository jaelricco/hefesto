import Foundation
import HefestoAPI
import HefestoAuth
import HefestoLogger
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
    var tab: AppTab = .today

    /// Set only in a Debug build started as a demo (DemoMode.swift): a seeded
    /// in-memory database, signed in, never on the network.
    private(set) var demo: DemoRun?

    private var pathMonitor: NWPathMonitor?

    init() throws {
        let baseURL = Self.apiBaseURL
        #if DEBUG
        if let screen = DemoRun.requestedScreen {
            db = try AppDatabase.inMemory()
            demo = try DemoRun.seed(db, screen: screen)
        } else {
            db = try Self.openDatabase()
        }
        #else
        db = try Self.openDatabase()
        #endif

        auth = AuthService(
            client: HefestoAPIConfiguration.client(serverURL: baseURL),
            store: KeychainTokenStore(), deviceId: Self.deviceId,
            appVersion: Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String)
        let api = HefestoAPIConfiguration.client(serverURL: baseURL, middlewares: [AuthMiddleware(auth: auth)])
        sync = SyncEngine(client: api, db: db)
    }

    private static func openDatabase() throws -> AppDatabase {
        let dir = try FileManager.default.url(
            for: .applicationSupportDirectory, in: .userDomainMask, appropriateFor: nil, create: true)
        return try AppDatabase.onDisk(at: dir.appending(path: "hefesto.sqlite"))
    }

    func start() async {
        if let demo {
            isSignedIn = demo.screen != .signIn
            tab = demo.screen.tab
            celebration = demo.celebration
            return
        }
        isSignedIn = await auth.isSignedIn
        watchNetwork()
        await syncNow()
    }

    /// A logger on a session. A demo hands out its prepared one, with the
    /// rest already running.
    func makeLogger(sessionId: String) throws -> LoggerModel {
        if let logger = demo?.logger, logger.session.id == sessionId { return logger }
        return try LoggerModel(db: db, sessionId: sessionId)
    }

    // MARK: auth

    func signedIn() async {
        guard demo == nil else { isSignedIn = true; return }
        isSignedIn = await auth.isSignedIn
        await syncNow()
    }

    func signOut() async {
        if demo == nil { await auth.logout() }
        isSignedIn = false
    }

    // MARK: sync

    /// One push-then-pull, and the catalogue if it changed. Failures leave
    /// everything queued; the next trigger tries again.
    func syncNow() async {
        guard isSignedIn, demo == nil else { return }
        isSyncing = true
        defer { isSyncing = false }
        do {
            let report = try await sync.sync()
            try? await sync.refreshExercises()
            // States change when a completion is evaluated; content rarely.
            try? await sync.refreshSkillMap()
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

    /// Fetches a skill's injury notes for offline reading; a demo has them already.
    func refreshSkillNotes(slug: String) async {
        guard demo == nil else { return }
        try? await sync.refreshSkillNotes(slug: slug)
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

enum AppTab: Hashable {
    case today, history, map
}

/// What a completed session earned, ready to show. Built from the server's
/// answer here, so the celebration does not depend on the API's types.
struct Celebration: Identifiable {
    struct Unlock: Hashable {
        let levelId: String
        let levelName: String
        let skillName: String
    }

    let sessionId: String
    let unlocked: [Unlock]
    let xp: Int
    let newlyAvailable: Int
    let streakDays: Int
    var id: String { sessionId }
}

extension Celebration {
    init(sessionId: String, result: Components.Schemas.CompletionResult) {
        self.init(
            sessionId: sessionId,
            unlocked: result.unlocked.map { Unlock(levelId: $0.levelId, levelName: $0.levelName, skillName: $0.skillName) },
            xp: result.xpAwarded.reduce(0) { $0 + $1.amount },
            newlyAvailable: result.newlyAvailable.count,
            streakDays: result.streak.currentDays)
    }
}
