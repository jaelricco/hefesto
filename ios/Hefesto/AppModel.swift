import Foundation
import HefestoAPI
import HefestoAuth
import HefestoLogger
import HefestoPlan
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
    /// The training plan the screens show (ADR 0022).
    let plan: PlanModel
    /// The onboarding's answers, kept while the app runs, so closing it
    /// loses nothing (ADR 0023).
    let onboarding: OnboardingModel
    /// The body's regions and the events about them (ADR 0028).
    let body: BodyModel
    /// The plan's settings after the onboarding (ADR 0029).
    let settings: PlanSettingsModel

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
        // A local: an observed property cannot be read before every stored
        // property is set.
        let demo: DemoRun?
        #if DEBUG
        if let screen = DemoRun.requestedScreen {
            db = try AppDatabase.inMemory()
            demo = try DemoRun.seed(db, screen: screen)
        } else {
            db = try Self.openDatabase()
            demo = nil
        }
        #else
        db = try Self.openDatabase()
        demo = nil
        #endif
        self.demo = demo

        auth = AuthService(
            client: HefestoAPIConfiguration.client(serverURL: baseURL),
            store: KeychainTokenStore(), deviceId: Self.deviceId,
            appVersion: Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String)
        let api = HefestoAPIConfiguration.client(serverURL: baseURL, middlewares: [AuthMiddleware(auth: auth)])
        sync = SyncEngine(client: api, db: db)
        // A demo shows the plan it seeded and never reaches the network.
        let plan = PlanModel(db: db, sync: demo == nil ? sync : nil)
        self.plan = plan
        onboarding = OnboardingModel(sync: demo == nil ? sync : nil, catalogue: demo?.catalogue)
        body = BodyModel(
            sync: demo == nil ? sync : nil, plan: plan, catalogue: demo?.catalogue,
            overview: demo?.bodyOverview.map { Data($0.utf8) })
        settings = PlanSettingsModel(
            sync: demo == nil ? sync : nil, plan: plan, catalogue: demo?.catalogue, form: demo?.settingsForm)
        if let form = demo?.onboardingForm { onboarding.form = form }
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
        if demo == nil {
            await auth.logout()
            await auth.useDevice(id: Self.newDeviceId())
            forgetAthlete()
        }
        isSignedIn = false
    }

    /// Forgets what the screens hold of the athlete in memory: answers, body
    /// regions, settings and the week shown. Wherever the local database is
    /// erased for another account (REVIEW_DECISIONS D-16), this goes with it.
    private func forgetAthlete() {
        plan.forget()
        onboarding.forget()
        body.forget()
        settings.forget()
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
            // Bands are not in the sync feed (ADR 0009).
            try? await sync.refreshBands()
            // States change when a completion is evaluated; content rarely.
            try? await sync.refreshSkillMap()
            // The plan changes with every event the sync carried.
            await plan.refresh()
            isOffline = false
            if let done = report.completions.first {
                celebration = Celebration(sessionId: done.key, result: done.value)
            }
        } catch {
            // A refresh the server refused signs the device out; anything
            // else is the network, and everything stays queued.
            let wasSignedIn = isSignedIn
            isSignedIn = await auth.isSignedIn
            isOffline = isSignedIn
            if wasSignedIn && !isSignedIn { await auth.useDevice(id: Self.newDeviceId()) }
        }
    }

    /// Adds a band to the athlete's collection, online. A demo keeps it here
    /// only.
    func createBand(brand: String, colourLabel: String, minKg: Double, maxKg: Double) async throws -> Band {
        guard demo == nil else {
            let band = Band(brand: brand, colourLabel: colourLabel, resistanceMinKg: minKg, resistanceMaxKg: maxKg)
            try db.saveBand(band)
            return band
        }
        return try await sync.createBand(
            brand: brand, colourLabel: colourLabel, resistanceMinKg: minKg, resistanceMaxKg: maxKg)
    }

    /// Removes one of the athlete's bands; sets logged with it keep it.
    func deleteBand(id: String) async throws {
        guard demo == nil else { return try db.markBandRemoved(id: id) }
        try await sync.deleteBand(id: id)
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

    /// This install's id for the signed-in account. Not a secret: it names
    /// the device's tokens so signing out here leaves other devices signed in.
    /// The server binds it to one account, so a sign-out starts a new one.
    static var deviceId: String {
        UserDefaults.standard.string(forKey: deviceIdKey) ?? newDeviceId()
    }

    static func newDeviceId() -> String {
        let id = UUIDv7.make()
        UserDefaults.standard.set(id, forKey: deviceIdKey)
        return id
    }

    private static let deviceIdKey = "fit.hefesto.deviceId"
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
