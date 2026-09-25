import Foundation
import GRDB
import HefestoAPI
import HefestoStore

// Local rows to and from the generated API types. Closed vocabularies are
// stored as their raw strings, and mapped through `init(rawValue:)` so a
// value the spec does not know is an error here, not a guess.

public enum MappingError: Error, Equatable {
    /// A stored value is not in the spec's vocabulary.
    case unknownValue(String)
}

func req<T: RawRepresentable>(_ raw: T.RawValue) throws -> T where T.RawValue == String {
    guard let v = T(rawValue: raw) else { throw MappingError.unknownValue(raw) }
    return v
}

func opt<T: RawRepresentable>(_ raw: String?) throws -> T? where T.RawValue == String {
    guard let raw else { return nil }
    return try req(raw) as T
}

typealias SyncOp = Components.Schemas.SyncOp

enum OpPayload {
    /// The op as it is sent, built from the row's current local state. `nil`
    /// when the row is gone locally, which only a delete can follow.
    static func build(_ op: OutboxOp, _ db: Database) throws -> SyncOp? {
        let kind: SyncOp.OpPayload = try req(op.op.rawValue)
        let entity: SyncOp.EntityPayload = try req(op.entity.rawValue)
        func make(_ data: SyncOp.DataPayload?) -> SyncOp {
            SyncOp(op: kind, entity: entity, id: op.rowId, sessionId: op.sessionId, data: data)
        }
        if op.op == .delete { return make(nil) }

        switch op.entity {
        case .session:
            guard let s = try Session.fetchOne(db, key: op.rowId) else { return nil }
            if op.op == .complete {
                return make(data(complete: .init(
                    completedAt: s.completedAt, endedAt: s.endedAt, perceivedFatigue: s.perceivedFatigue,
                    bodyweightKg: s.bodyweightKg, updatedAt: s.updatedAt)))
            }
            return make(data(session: .init(
                startedAt: s.startedAt, endedAt: s.endedAt, timezone: s.timezone, title: s.title, notes: s.notes,
                perceivedFatigue: s.perceivedFatigue, bodyweightKg: s.bodyweightKg, isRestDay: s.isRestDay,
                templateId: s.templateId,
                // Completion is its own op; a completed session's put leaves status alone.
                status: try opt(s.status == "completed" ? nil : s.status),
                updatedAt: s.updatedAt)))
        case .block:
            guard let b = try Block.fetchOne(db, key: op.rowId) else { return nil }
            return make(data(block: .init(
                orderIndex: b.orderIndex, kind: try req(b.kind), roundsPlanned: b.roundsPlanned,
                roundsDone: b.roundsDone, intervalS: b.intervalS, notes: b.notes, updatedAt: b.updatedAt)))
        case .set:
            guard let e = try SetEntry.fetchOne(db, key: op.rowId) else { return nil }
            let elements = try SetElement.filter(Column("setEntryId") == e.id).order(Column("orderIndex")).fetchAll(db)
            return make(data(set: .init(
                blockId: e.blockId, orderIndex: e.orderIndex, roundIndex: e.roundIndex, kind: try req(e.kind),
                isPlanned: e.isPlanned, restAfterPlannedS: e.restAfterPlannedS, restAfterActualS: e.restAfterActualS,
                rpe: e.rpe, rir: e.rir, completedAt: e.completedAt, notes: e.notes, updatedAt: e.updatedAt,
                elements: try elements.map(element))))
        case .bodyweight:
            guard let b = try Bodyweight.fetchOne(db, key: op.rowId) else { return nil }
            return make(data(bodyweight: .init(
                // The feed does not carry the zone; a pulled entry edited here takes the device's.
                measuredAt: b.measuredAt, timezone: b.timezone.isEmpty ? TimeZone.current.identifier : b.timezone,
                bodyweightKg: b.bodyweightKg, note: b.note,
                updatedAt: b.updatedAt)))
        }
    }

    static func element(_ e: SetElement) throws -> Components.Schemas.SetElementWrite {
        .init(
            id: e.id, exerciseId: e.exerciseId, measure: try req(e.measure), reps: e.reps,
            holdSeconds: e.holdSeconds, distanceM: e.distanceM, tempo: e.tempo, loadKg: e.loadKg,
            isEccentricOnly: e.isEccentricOnly, isPartialRom: e.isPartialRom, romNote: e.romNote,
            formQuality: e.formQuality, failed: e.failed,
            assistance: try e.assistance.map { a in
                Components.Schemas.AssistanceWrite(
                    id: a.id, _type: try req(a.type), bandId: a.bandId, bandCount: a.bandCount,
                    anchor: try opt(a.anchor), estimatedAssistKg: a.estimatedAssistKg, note: a.note)
            },
            mediaIds: e.mediaIds.isEmpty ? nil : e.mediaIds)
    }

    private static func data(
        session: Components.Schemas.SessionPut? = nil, complete: Components.Schemas.SessionComplete? = nil,
        block: Components.Schemas.BlockWrite? = nil, set: Components.Schemas.SetEntryWrite? = nil,
        bodyweight: Components.Schemas.BodyweightWrite? = nil
    ) -> SyncOp.DataPayload {
        .init(value1: session, value2: complete, value3: block, value4: set, value5: bodyweight)
    }
}

// MARK: - Server rows to local rows

enum ServerRows {
    static func page(_ p: Components.Schemas.SyncPage) -> ServerPage {
        ServerPage(
            cursor: p.cursor,
            sessions: p.sessions.map(session),
            blocks: p.blocks.map(block),
            sets: p.sets.map(set),
            bodyweight: p.bodyweight.map(bodyweight))
    }

    static func session(_ s: Components.Schemas.SyncSession) -> Session {
        Session(
            id: s.id, startedAt: s.startedAt, endedAt: s.endedAt, timezone: s.timezone, localDate: s.localDate,
            title: s.title, notes: s.notes, perceivedFatigue: s.perceivedFatigue, bodyweightKg: s.bodyweightKg,
            status: s.status.rawValue, isRestDay: s.isRestDay, templateId: s.templateId,
            completedAt: s.completedAt, updatedAt: s.updatedAt, serverSeq: s.serverSeq, deletedAt: s.deletedAt)
    }

    static func block(_ b: Components.Schemas.SyncBlock) -> Block {
        Block(
            id: b.id, sessionId: b.sessionId, orderIndex: b.orderIndex, kind: b.kind.rawValue,
            roundsPlanned: b.roundsPlanned, roundsDone: b.roundsDone, intervalS: b.intervalS, notes: b.notes,
            updatedAt: b.updatedAt, serverSeq: b.serverSeq, deletedAt: b.deletedAt)
    }

    static func set(_ s: Components.Schemas.SyncSet) -> SetWithElements {
        SetWithElements(
            entry: SetEntry(
                id: s.id, sessionId: s.sessionId, blockId: s.blockId, orderIndex: s.orderIndex,
                roundIndex: s.roundIndex, kind: s.kind.rawValue, isPlanned: s.isPlanned,
                restAfterPlannedS: s.restAfterPlannedS, restAfterActualS: s.restAfterActualS, rpe: s.rpe,
                rir: s.rir, completedAt: s.completedAt, notes: s.notes, updatedAt: s.updatedAt,
                serverSeq: s.serverSeq, deletedAt: s.deletedAt),
            elements: s.elements.map { element($0, setEntryId: s.id) })
    }

    static func element(_ e: Components.Schemas.SetElement, setEntryId: String) -> SetElement {
        SetElement(
            id: e.id, setEntryId: setEntryId, orderIndex: e.orderIndex, exerciseId: e.exerciseId,
            measure: e.measure.rawValue, reps: e.reps, holdSeconds: e.holdSeconds, distanceM: e.distanceM,
            tempo: e.tempo, loadKg: e.loadKg, isEccentricOnly: e.isEccentricOnly, isPartialRom: e.isPartialRom,
            romNote: e.romNote, formQuality: e.formQuality, failed: e.failed,
            assistanceClass: e.assistanceClass.rawValue,
            assistance: e.assistance.map {
                Assistance(
                    id: $0.id, type: $0._type.rawValue, bandId: $0.bandId, bandCount: $0.bandCount,
                    anchor: $0.anchor, estimatedAssistKg: $0.estimatedAssistKg, note: $0.note)
            },
            mediaIds: e.mediaIds)
    }

    static func bodyweight(_ b: Components.Schemas.SyncBodyweight) -> Bodyweight {
        Bodyweight(
            id: b.id, measuredAt: b.measuredAt, timezone: "", localDate: b.localDate, bodyweightKg: b.bodyweightKg,
            note: b.note, updatedAt: b.updatedAt, serverSeq: b.serverSeq, deletedAt: b.deletedAt)
    }

    /// A session as `GET /v1/sessions/{id}` returns it, with its live tree.
    static func tree(_ s: Components.Schemas.Session) -> SessionTree {
        let session = Session(
            id: s.id, startedAt: s.startedAt, endedAt: s.endedAt, timezone: s.timezone, localDate: s.localDate,
            title: s.title, notes: s.notes, perceivedFatigue: s.perceivedFatigue, bodyweightKg: s.bodyweightKg,
            status: s.status.rawValue, isRestDay: s.isRestDay, templateId: s.templateId,
            completedAt: s.completedAt, updatedAt: s.updatedAt, serverSeq: nil, deletedAt: nil)
        return SessionTree(session: session, blocks: s.blocks.map { b in
            BlockWithSets(
                block: Block(
                    id: b.id, sessionId: s.id, orderIndex: b.orderIndex, kind: b.kind.rawValue,
                    roundsPlanned: b.roundsPlanned, roundsDone: b.roundsDone, intervalS: b.intervalS,
                    notes: b.notes, updatedAt: b.updatedAt, serverSeq: nil, deletedAt: nil),
                sets: b.sets.map { e in
                    SetWithElements(
                        entry: SetEntry(
                            id: e.id, sessionId: s.id, blockId: e.blockId, orderIndex: e.orderIndex,
                            roundIndex: e.roundIndex, kind: e.kind.rawValue, isPlanned: e.isPlanned,
                            restAfterPlannedS: e.restAfterPlannedS, restAfterActualS: e.restAfterActualS,
                            rpe: e.rpe, rir: e.rir, completedAt: e.completedAt, notes: e.notes,
                            updatedAt: e.updatedAt, serverSeq: nil, deletedAt: nil),
                        elements: e.elements.map { element($0, setEntryId: e.id) })
                })
        })
    }

    static func exercise(_ e: Components.Schemas.Exercise) -> Exercise {
        Exercise(
            id: e.id, slug: e.slug, name: e.name, family: e.family, defaultMeasure: e.defaultMeasure.rawValue,
            isBodyweight: e.isBodyweight)
    }
}
