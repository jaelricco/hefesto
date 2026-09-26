import Foundation
import GRDB

// Local rows mirror the sync feed (ADR 0009, ADR 0010). Ids are lowercase
// UUID strings, as the API carries them. `serverSeq` is nil until the row has
// come back from the server once.

public struct Session: Codable, Sendable, Hashable, Identifiable, FetchableRecord, PersistableRecord {
    public static let databaseTableName = "session"
    public var id: String
    public var startedAt: Date
    public var endedAt: Date?
    public var timezone: String
    public var localDate: String // YYYY-MM-DD in `timezone`
    public var title: String
    public var notes: String
    public var perceivedFatigue: Int?
    public var bodyweightKg: Double?
    public var status: String // draft | completed | abandoned
    public var isRestDay: Bool
    public var templateId: String?
    public var completedAt: Date?
    public var updatedAt: Date
    public var serverSeq: Int64?
    public var deletedAt: Date?

    public init(
        id: String = UUIDv7.make(), startedAt: Date, endedAt: Date? = nil, timezone: String,
        localDate: String, title: String = "", notes: String = "", perceivedFatigue: Int? = nil,
        bodyweightKg: Double? = nil, status: String = "draft", isRestDay: Bool = false,
        templateId: String? = nil, completedAt: Date? = nil, updatedAt: Date = Date(),
        serverSeq: Int64? = nil, deletedAt: Date? = nil
    ) {
        self.id = id; self.startedAt = startedAt; self.endedAt = endedAt; self.timezone = timezone
        self.localDate = localDate; self.title = title; self.notes = notes
        self.perceivedFatigue = perceivedFatigue; self.bodyweightKg = bodyweightKg; self.status = status
        self.isRestDay = isRestDay; self.templateId = templateId; self.completedAt = completedAt
        self.updatedAt = updatedAt; self.serverSeq = serverSeq; self.deletedAt = deletedAt
    }
}

public struct Block: Codable, Sendable, Hashable, Identifiable, FetchableRecord, PersistableRecord {
    public static let databaseTableName = "block"
    public var id: String
    public var sessionId: String
    public var orderIndex: Int
    public var kind: String // straight | superset | circuit | emom | amrap | ...
    public var roundsPlanned: Int?
    public var roundsDone: Int?
    public var intervalS: Int?
    public var notes: String
    public var updatedAt: Date
    public var serverSeq: Int64?
    public var deletedAt: Date?

    public init(
        id: String = UUIDv7.make(), sessionId: String, orderIndex: Int, kind: String = "straight",
        roundsPlanned: Int? = nil, roundsDone: Int? = nil, intervalS: Int? = nil, notes: String = "",
        updatedAt: Date = Date(), serverSeq: Int64? = nil, deletedAt: Date? = nil
    ) {
        self.id = id; self.sessionId = sessionId; self.orderIndex = orderIndex; self.kind = kind
        self.roundsPlanned = roundsPlanned; self.roundsDone = roundsDone; self.intervalS = intervalS
        self.notes = notes; self.updatedAt = updatedAt; self.serverSeq = serverSeq; self.deletedAt = deletedAt
    }
}

/// A set entry. Its elements are rows of `SetElement`; a plain set has one,
/// a combo several. There is no other kind of set (CLAUDE.md).
public struct SetEntry: Codable, Sendable, Hashable, Identifiable, FetchableRecord, PersistableRecord {
    public static let databaseTableName = "setEntry"
    public var id: String
    public var sessionId: String
    public var blockId: String
    public var orderIndex: Int
    public var roundIndex: Int?
    public var kind: String // working | warmup | ...
    public var isPlanned: Bool
    public var restAfterPlannedS: Int?
    public var restAfterActualS: Int?
    public var rpe: Double?
    public var rir: Int?
    public var completedAt: Date?
    public var notes: String
    public var updatedAt: Date
    public var serverSeq: Int64?
    public var deletedAt: Date?

    public init(
        id: String = UUIDv7.make(), sessionId: String, blockId: String, orderIndex: Int,
        roundIndex: Int? = nil, kind: String = "working", isPlanned: Bool = false,
        restAfterPlannedS: Int? = nil, restAfterActualS: Int? = nil, rpe: Double? = nil, rir: Int? = nil,
        completedAt: Date? = nil, notes: String = "", updatedAt: Date = Date(),
        serverSeq: Int64? = nil, deletedAt: Date? = nil
    ) {
        self.id = id; self.sessionId = sessionId; self.blockId = blockId; self.orderIndex = orderIndex
        self.roundIndex = roundIndex; self.kind = kind; self.isPlanned = isPlanned
        self.restAfterPlannedS = restAfterPlannedS; self.restAfterActualS = restAfterActualS
        self.rpe = rpe; self.rir = rir; self.completedAt = completedAt; self.notes = notes
        self.updatedAt = updatedAt; self.serverSeq = serverSeq; self.deletedAt = deletedAt
    }
}

/// Help received on an element. Stored as JSON on its element: it is
/// one-to-one, and an element with no assistance stores nothing.
public struct Assistance: Codable, Sendable, Hashable {
    public var id: String
    public var type: String // band | partner | machine | incline | counterweight | foot_support
    public var bandId: String?
    public var bandCount: Int
    public var anchor: String?
    public var estimatedAssistKg: Double?
    public var note: String

    public init(
        id: String = UUIDv7.make(), type: String, bandId: String? = nil, bandCount: Int = 0,
        anchor: String? = nil, estimatedAssistKg: Double? = nil, note: String = ""
    ) {
        self.id = id; self.type = type; self.bandId = bandId; self.bandCount = bandCount
        self.anchor = anchor; self.estimatedAssistKg = estimatedAssistKg; self.note = note
    }
}

public struct SetElement: Codable, Sendable, Hashable, Identifiable, FetchableRecord, PersistableRecord {
    public static let databaseTableName = "setElement"
    public var id: String
    public var setEntryId: String
    public var orderIndex: Int
    public var exerciseId: String
    public var measure: String // reps | hold_seconds | distance_m | none
    public var reps: Int?
    public var holdSeconds: Double?
    public var distanceM: Double?
    public var tempo: String?
    public var loadKg: Double
    public var isEccentricOnly: Bool
    public var isPartialRom: Bool
    public var romNote: String?
    public var formQuality: Int?
    public var failed: Bool
    /// Derived by the server; kept so history reads right before a sync.
    public var assistanceClass: String
    public var assistance: Assistance?
    public var mediaIds: [String]

    public init(
        id: String = UUIDv7.make(), setEntryId: String, orderIndex: Int, exerciseId: String,
        measure: String, reps: Int? = nil, holdSeconds: Double? = nil, distanceM: Double? = nil,
        tempo: String? = nil, loadKg: Double = 0, isEccentricOnly: Bool = false,
        isPartialRom: Bool = false, romNote: String? = nil, formQuality: Int? = nil, failed: Bool = false,
        assistanceClass: String = "unassisted", assistance: Assistance? = nil, mediaIds: [String] = []
    ) {
        self.id = id; self.setEntryId = setEntryId; self.orderIndex = orderIndex; self.exerciseId = exerciseId
        self.measure = measure; self.reps = reps; self.holdSeconds = holdSeconds; self.distanceM = distanceM
        self.tempo = tempo; self.loadKg = loadKg; self.isEccentricOnly = isEccentricOnly
        self.isPartialRom = isPartialRom; self.romNote = romNote; self.formQuality = formQuality
        self.failed = failed; self.assistanceClass = assistanceClass; self.assistance = assistance
        self.mediaIds = mediaIds
    }
}

public struct Bodyweight: Codable, Sendable, Hashable, Identifiable, FetchableRecord, PersistableRecord {
    public static let databaseTableName = "bodyweight"
    public var id: String
    public var measuredAt: Date
    public var timezone: String
    public var localDate: String
    public var bodyweightKg: Double
    public var note: String
    public var updatedAt: Date
    public var serverSeq: Int64?
    public var deletedAt: Date?

    public init(
        id: String = UUIDv7.make(), measuredAt: Date, timezone: String, localDate: String,
        bodyweightKg: Double, note: String = "", updatedAt: Date = Date(),
        serverSeq: Int64? = nil, deletedAt: Date? = nil
    ) {
        self.id = id; self.measuredAt = measuredAt; self.timezone = timezone; self.localDate = localDate
        self.bodyweightKg = bodyweightKg; self.note = note; self.updatedAt = updatedAt
        self.serverSeq = serverSeq; self.deletedAt = deletedAt
    }
}

/// The exercise catalogue, cached for the logger's picker. Content, not the
/// athlete's: it is replaced whole when the content version changes.
public struct Exercise: Codable, Sendable, Hashable, Identifiable, FetchableRecord, PersistableRecord {
    public static let databaseTableName = "exercise"
    public var id: String
    public var slug: String
    public var name: String
    public var family: String
    public var defaultMeasure: String
    public var isBodyweight: Bool

    public init(id: String, slug: String, name: String, family: String, defaultMeasure: String, isBodyweight: Bool) {
        self.id = id; self.slug = slug; self.name = name; self.family = family
        self.defaultMeasure = defaultMeasure; self.isBodyweight = isBodyweight
    }
}

/// A set with its ordered elements, the unit the logger and the sync
/// engine work with.
public struct SetWithElements: Sendable, Hashable, Identifiable {
    public var entry: SetEntry
    public var elements: [SetElement]
    public var id: String { entry.id }

    public init(entry: SetEntry, elements: [SetElement]) {
        self.entry = entry
        self.elements = elements
    }
}
