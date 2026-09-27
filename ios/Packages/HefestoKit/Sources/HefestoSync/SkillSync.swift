import Foundation
import HefestoAPI
import HefestoStore

/// Why an attestation was refused, in terms the UI can explain.
public enum AttestError: Error, Equatable {
    /// The level's prerequisites are not all unlocked yet.
    case prerequisitesMissing
    case notFound
    case refused(status: Int)
}

extension SyncEngine {
    /// Refreshes the skill map and the athlete's progress. The map's content
    /// changes rarely; the states change whenever a completion is evaluated,
    /// so this follows every sync that pushed one.
    public func refreshSkillMap() async throws {
        switch try await client.getMySkillMap() {
        case let .ok(r): try db.replaceSkillMap(SkillRows.snapshot(try r.body.json))
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw SyncError.refused(status: status, type: "")
        }
        switch try await client.getMyProgress() {
        case let .ok(r): try db.saveProgress(SkillRows.progress(try r.body.json))
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw SyncError.refused(status: status, type: "")
        }
    }

    /// Fetches a skill's injury notes with their disclaimer, and keeps them
    /// for offline reading.
    public func refreshSkillNotes(slug: String) async throws {
        switch try await client.getSkill(path: .init(slug: slug)) {
        case let .ok(r): try db.saveSkillNotes(SkillRows.notes(slug: slug, try r.body.json))
        case .notFound: return
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw SyncError.refused(status: status, type: "")
        }
    }

    /// Self-attests a level the log cannot prove (a level achieved before
    /// Hefesto, or one with no criteria). It is recorded as self-attested and
    /// earns no XP. Needs the network: the server checks the prerequisites.
    @discardableResult
    public func attest(levelId: String) async throws -> Components.Schemas.AttestResult {
        let result: Components.Schemas.AttestResult
        switch try await client.attestSkillLevel(path: .init(levelId: levelId), body: .json(.init())) {
        case let .ok(r): result = try r.body.json
        case .conflict: throw AttestError.prerequisitesMissing
        case .notFound: throw AttestError.notFound
        case .badRequest: throw AttestError.refused(status: 400)
        case let .unauthorized(r): throw Self.refused(401, try? r.body.applicationProblemJson)
        case let .undocumented(status, _): throw AttestError.refused(status: status)
        }
        try await refreshSkillMap()
        return result
    }
}

/// The server's skill map and progress, mapped to local rows.
enum SkillRows {
    static func snapshot(_ m: Components.Schemas.SkillMap) -> SkillMapSnapshot {
        SkillMapSnapshot(
            contentVersion: m.contentVersion,
            skills: m.skills.map(skill),
            edges: m.edges.map {
                SkillEdge(fromLevelId: $0.fromLevelId, toLevelId: $0.toLevelId, relation: $0.relation.rawValue,
                          weight: $0.weight)
            },
            states: m.states.map {
                LevelState(
                    levelId: $0.levelId, state: $0.state.rawValue, bestValue: $0.bestValue,
                    bestUnit: $0.bestUnit?.rawValue, bestAt: $0.bestAt, firstAchievedAt: $0.firstAchievedAt,
                    verification: $0.verification?.rawValue, staleSince: $0.staleSince,
                    attemptsCount: $0.attemptsCount)
            })
    }

    static func skill(_ s: Components.Schemas.Skill) -> SkillWithLevels {
        SkillWithLevels(
            skill: Skill(
                id: s.id, slug: s.slug, name: s.name, family: s.family, difficultyTier: s.difficultyTier,
                isMilestone: s.isMilestone, status: s.status.rawValue, summary: s.summary, aka: s.aka,
                primaryMuscles: s.primaryMuscles, commonFaults: s.commonFaults,
                constellation: s.map?.constellation, x: s.map?.x, y: s.map?.y),
            levels: s.levels.map { l in
                SkillLevel(
                    id: l.id, skillId: s.id, orderIndex: l.order, slug: l.slug, name: l.name, details: l.description,
                    estWeeksFromPrev: l.estWeeksFromPrev,
                    criteria: UnlockCriteria(
                        all: (l.unlockCriteria.all ?? []).map(condition),
                        any: (l.unlockCriteria.any ?? []).map(condition)),
                    exercises: l.exercises.map {
                        LevelExercise(exerciseId: $0.exerciseId, slug: $0.slug, role: $0.role.rawValue)
                    })
            })
    }

    static func condition(_ c: Components.Schemas.Condition) -> UnlockCondition {
        UnlockCondition(
            exercise: c.exercise, measure: c.measure.rawValue, op: c.op.rawValue, value: c.value,
            assistance: c.assistance.rawValue, occurrences: c.occurrences, minFormQuality: c.minFormQuality,
            minLoadKg: c.minLoadKg, maxLoadKg: c.maxLoadKg, withinDays: c.withinDays)
    }

    static func progress(_ p: Components.Schemas.Progress) -> AthleteProgress {
        AthleteProgress(
            xpTotal: p.xpTotal, currentDays: p.streak.currentDays, longestDays: p.streak.longestDays,
            freezeCredits: p.streak.freezeCredits, lastCountedDate: p.streak.lastCountedDate)
    }

    static func notes(slug: String, _ d: Components.Schemas.SkillDetail) -> SkillNotes {
        SkillNotes(
            skillSlug: slug,
            injuries: d.value2.injuries.map {
                InjuryNote(
                    region: $0.region, name: $0.name, details: $0.description, riskFactors: $0.riskFactors,
                    earlySigns: $0.earlySigns, prehabExerciseSlugs: $0.prehabExercises.map(\.slug))
            },
            disclaimer: d.value2.injuryDisclaimer.text)
    }
}
