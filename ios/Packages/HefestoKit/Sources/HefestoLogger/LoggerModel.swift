import Foundation
import HefestoStore
import Observation

/// The session logger's state and its only way to change a session. Every
/// change goes through the store, which queues it for sync, so the logger
/// works the same with or without a network (ADR 0010).
///
/// A plain set is a set with one element; a combo is the same set with more
/// elements. There is no other path (CLAUDE.md).
@MainActor
@Observable
public final class LoggerModel {
    public private(set) var tree: SessionTree
    /// The rest running since the last completed set, if any.
    public private(set) var rest: RestTimer?
    /// Default rest after a set, when the set does not plan its own.
    public var defaultRestSeconds: Int = 120

    private let db: AppDatabase
    private let now: @Sendable () -> Date

    public init(db: AppDatabase, sessionId: String, now: @escaping @Sendable () -> Date = { Date() }) throws {
        guard let tree = try db.reader.read({ try AppDatabase.sessionTree($0, id: sessionId) }) else {
            throw StoreError.notFound
        }
        self.db = db
        self.tree = tree
        self.now = now
    }

    /// Starts a new draft session today and opens a logger on it.
    public static func startSession(
        db: AppDatabase, timeZone: TimeZone = .current, title: String = "",
        isRestDay: Bool = false, now: @escaping @Sendable () -> Date = { Date() }
    ) throws -> LoggerModel {
        let started = now()
        let session = Session(
            startedAt: started, timezone: timeZone.identifier,
            localDate: Self.localDate(started, in: timeZone), title: title, isRestDay: isRestDay)
        try db.saveSession(session, now: started)
        if !isRestDay {
            try db.saveBlock(Block(sessionId: session.id, orderIndex: 0), now: started)
        }
        return try LoggerModel(db: db, sessionId: session.id, now: now)
    }

    public var session: Session { tree.session }

    // MARK: blocks

    /// Adds a block after the last one and returns its id.
    @discardableResult
    public func addBlock(kind: String = "straight") throws -> String {
        let block = Block(sessionId: session.id, orderIndex: (tree.blocks.map(\.block.orderIndex).max() ?? -1) + 1, kind: kind)
        try db.saveBlock(block, now: now())
        try reload()
        return block.id
    }

    // MARK: sets

    /// Logs a performed set of one element: 8 pull-ups, a 20 s hold.
    @discardableResult
    public func logSet(in blockId: String, element: ElementDraft, restPlannedSeconds: Int? = nil) throws -> String {
        try logCombo(in: blockId, elements: [element], restPlannedSeconds: restPlannedSeconds)
    }

    /// Logs a performed combo: one set, its elements in order.
    @discardableResult
    public func logCombo(in blockId: String, elements: [ElementDraft], restPlannedSeconds: Int? = nil) throws -> String {
        let at = now()
        var entry = SetEntry(
            sessionId: session.id, blockId: blockId, orderIndex: nextSetIndex(in: blockId),
            restAfterPlannedS: restPlannedSeconds, completedAt: at)
        entry = try closeRest(before: entry, at: at)
        let set = SetWithElements(
            entry: entry,
            elements: elements.enumerated().map { $0.element.element(in: entry.id, orderIndex: $0.offset) })
        try db.saveSet(set, now: at)
        rest = RestTimer(startedAt: at, plannedSeconds: restPlannedSeconds ?? defaultRestSeconds)
        try reload()
        return entry.id
    }

    /// Adds an element to an existing set, turning it into a combo (or a
    /// longer one). The set is rewritten whole, as every set write is.
    public func addElement(to setId: String, _ element: ElementDraft) throws {
        guard var set = findSet(setId) else { throw StoreError.notFound }
        set.elements.append(element.element(in: setId, orderIndex: set.elements.count))
        try db.saveSet(set, now: now())
        try reload()
    }

    /// Replaces a set's elements, for correcting a logged set.
    public func editSet(_ setId: String, elements: [SetElement]) throws {
        guard var set = findSet(setId) else { throw StoreError.notFound }
        set.elements = elements
        try db.saveSet(set, now: now())
        try reload()
    }

    public func deleteSet(_ setId: String) throws {
        try db.deleteSet(id: setId, sessionId: session.id, now: now())
        try reload()
    }

    /// Logs again the athlete's last set of this exercise, elements and all:
    /// the one-tap "repeat last set". Returns nil if there is none.
    @discardableResult
    public func repeatLastSet(of exerciseId: String, in blockId: String) throws -> String? {
        guard let last = try db.reader.read({ try AppDatabase.lastSet($0, exerciseId: exerciseId) }) else {
            return nil
        }
        let drafts = last.elements.map(ElementDraft.init(copying:))
        return try logCombo(in: blockId, elements: drafts, restPlannedSeconds: last.entry.restAfterPlannedS)
    }

    // MARK: rest

    /// Ends the rest without logging a set.
    public func skipRest() {
        rest = nil
    }

    // MARK: finishing

    /// Completes the session. Unlocks arrive when the completion syncs.
    public func complete(perceivedFatigue: Int? = nil) throws {
        try db.completeSession(id: session.id, at: now(), perceivedFatigue: perceivedFatigue, now: now())
        rest = nil
        try reload()
    }

    public func reload() throws {
        guard let tree = try db.reader.read({ try AppDatabase.sessionTree($0, id: session.id) }) else {
            throw StoreError.notFound
        }
        self.tree = tree
    }

    // MARK: private

    private func findSet(_ id: String) -> SetWithElements? {
        tree.blocks.lazy.flatMap(\.sets).first { $0.id == id }
    }

    private func nextSetIndex(in blockId: String) -> Int {
        (tree.blocks.first { $0.id == blockId }?.sets.map(\.entry.orderIndex).max() ?? -1) + 1
    }

    /// Records the actual rest on the previous set when the next one starts.
    private func closeRest(before entry: SetEntry, at: Date) throws -> SetEntry {
        guard let rest, let previous = tree.blocks.flatMap(\.sets).max(by: {
            ($0.entry.completedAt ?? .distantPast) < ($1.entry.completedAt ?? .distantPast)
        }) else { return entry }
        var updated = previous
        updated.entry.restAfterActualS = rest.elapsed(at: at)
        try db.saveSet(updated, now: at)
        return entry
    }

    static func localDate(_ date: Date, in zone: TimeZone) -> String {
        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = zone
        let c = cal.dateComponents([.year, .month, .day], from: date)
        return String(format: "%04d-%02d-%02d", c.year ?? 0, c.month ?? 0, c.day ?? 0)
    }
}

/// What the athlete enters for one element of a set.
public struct ElementDraft: Sendable, Hashable {
    public var exerciseId: String
    public var measure: String
    public var reps: Int?
    public var holdSeconds: Double?
    public var distanceM: Double?
    public var loadKg: Double
    public var assistance: Assistance?
    public var formQuality: Int?
    public var failed: Bool
    public var isPartialRom: Bool
    public var isEccentricOnly: Bool

    public init(
        exerciseId: String, measure: String, reps: Int? = nil, holdSeconds: Double? = nil,
        distanceM: Double? = nil, loadKg: Double = 0, assistance: Assistance? = nil,
        formQuality: Int? = nil, failed: Bool = false, isPartialRom: Bool = false, isEccentricOnly: Bool = false
    ) {
        self.exerciseId = exerciseId; self.measure = measure; self.reps = reps
        self.holdSeconds = holdSeconds; self.distanceM = distanceM; self.loadKg = loadKg
        self.assistance = assistance; self.formQuality = formQuality; self.failed = failed
        self.isPartialRom = isPartialRom; self.isEccentricOnly = isEccentricOnly
    }

    /// A copy of a logged element, for repeating a set. Its assistance gets a
    /// new id: it is a new row.
    public init(copying e: SetElement) {
        self.init(
            exerciseId: e.exerciseId, measure: e.measure, reps: e.reps, holdSeconds: e.holdSeconds,
            distanceM: e.distanceM, loadKg: e.loadKg,
            assistance: e.assistance.map { var a = $0; a.id = UUIDv7.make(); return a },
            formQuality: nil, failed: false, isPartialRom: e.isPartialRom, isEccentricOnly: e.isEccentricOnly)
    }

    func element(in setId: String, orderIndex: Int) -> SetElement {
        SetElement(
            setEntryId: setId, orderIndex: orderIndex, exerciseId: exerciseId, measure: measure,
            reps: reps, holdSeconds: holdSeconds, distanceM: distanceM, loadKg: loadKg,
            isEccentricOnly: isEccentricOnly, isPartialRom: isPartialRom, formQuality: formQuality,
            failed: failed, assistanceClass: Self.assistanceClass(assistance: assistance, loadKg: loadKg),
            assistance: assistance)
    }

    /// The class the server will derive, so history reads right before a sync.
    static func assistanceClass(assistance: Assistance?, loadKg: Double) -> String {
        if assistance != nil { return "assisted" }
        return loadKg > 0 ? "loaded" : "unassisted"
    }
}
