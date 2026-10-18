import Foundation
import HefestoAPI
import HefestoStore
import HefestoSync
import Observation

/// The plan's settings after the onboarding (ADR 0029): goals, time, equipment
/// and body weight, edited in the onboarding's own form so the screens are
/// the same. What the athlete does not edit here goes back as it was: the
/// server clears optional fields left out.
@MainActor
@Observable
public final class PlanSettingsModel {
    public enum Status: Equatable, Sendable {
        case loading
        case ready
        /// The server could not be reached; settings change online.
        case offline
        /// There is no plan yet: the onboarding comes first.
        case needsOnboarding
        /// The server's planner is not available.
        case unavailable
    }

    public private(set) var status: Status = .loading
    /// The settings being edited. Its catalogue is the planner's.
    public private(set) var editor: OnboardingModel
    /// How long the step to each dated goal usually takes, after a save, in
    /// the knowledge base's words.
    public private(set) var realism: [PlanNote] = []

    private let sync: SyncEngine?
    /// Fetched again after a save: the server plans the week anew.
    private let plan: PlanModel?
    /// The profile as loaded, for the fields the screens do not edit.
    @ObservationIgnored private var kept: Components.Schemas.TrainingProfile?
    /// The form as loaded, to send only what changed.
    @ObservationIgnored private var loaded = OnboardingForm()

    public init(sync: SyncEngine?, plan: PlanModel?, catalogue: OnboardingCatalogue? = nil, form: OnboardingForm? = nil) {
        self.sync = sync
        self.plan = plan
        editor = OnboardingModel(sync: nil, catalogue: catalogue)
        if let form {
            editor.form = form
            loaded = form
            status = .ready
        }
    }

    public var catalogue: OnboardingCatalogue? { editor.catalogue }

    /// Goals, time and body weight make a plan the server takes.
    public var canSave: Bool {
        guard let catalogue, editor.form.isComplete(.goals, catalogue, currentYear: editor.currentYear),
              editor.form.isComplete(.availability, catalogue, currentYear: editor.currentYear),
              let kg = editor.form.bodyweightKg
        else { return false }
        return (30...250).contains(kg) && hasChanges
    }

    public var hasChanges: Bool {
        goalsChanged || profileChanged
    }

    private var goalsChanged: Bool { editor.form.goals != loaded.goals }

    private var profileChanged: Bool {
        editor.form.sessionsPerWeek != loaded.sessionsPerWeek || editor.form.sessionMinutes != loaded.sessionMinutes
            || editor.form.equipment != loaded.equipment || editor.form.bodyweightKg != loaded.bodyweightKg
    }

    /// Fetches the catalogue, the profile and the goals.
    public func load() async {
        guard let sync else { return }
        do {
            let catalogue: OnboardingCatalogue
            if let known = editor.catalogue {
                catalogue = known
            } else {
                catalogue = OnboardingCatalogue(try await sync.plannerCatalogue())
            }
            let profile = try await sync.trainingProfile()
            let goals = try await sync.trainingGoals()
            var form = OnboardingForm()
            form.goals = goals.goals.sorted { $0.priority < $1.priority }.map {
                OnboardingForm.Goal(skill: $0.skill, targetLevel: $0.targetLevel, targetDate: $0.targetDate)
            }
            form.sessionsPerWeek = profile.sessionsPerWeek
            form.sessionMinutes = profile.sessionMinutes
            form.equipment = Set(profile.equipment.map(\.rawValue))
            form.bodyweightKg = profile.bodyweightKg
            let editor = OnboardingModel(sync: nil, catalogue: catalogue)
            editor.form = form
            self.editor = editor
            kept = profile
            loaded = form
            realism = goals.realism.map { PlanNote($0.reason) }
            status = .ready
        } catch SettingsError.onboardingRequired {
            status = .needsOnboarding
        } catch SettingsError.unavailable {
            status = .unavailable
        } catch OnboardingError.unavailable {
            status = .unavailable
        } catch {
            status = .offline
        }
    }

    /// Sends what changed. The server plans the week anew after each part.
    public func save() async throws {
        guard let sync else { throw PlanStartError.noServer }
        let form = editor.form
        if goalsChanged {
            let goals = try await sync.updateTrainingGoals(form.goals.enumerated().map { i, g in
                Components.Schemas.TrainingGoalWrite(
                    skill: g.skill, targetLevel: g.targetLevel, priority: i + 1, targetDate: g.targetDate)
            })
            realism = goals.realism.map { PlanNote($0.reason) }
            loaded.goals = form.goals
        }
        if profileChanged, let sessions = form.sessionsPerWeek, let minutes = form.sessionMinutes,
           let kg = form.bodyweightKg {
            let update = Components.Schemas.TrainingProfileUpdate(
                sessionsPerWeek: sessions, sessionMinutes: minutes,
                equipment: form.equipment.sorted().compactMap(Components.Schemas.Equipment.init(rawValue:)),
                bodyweightKg: kg,
                // Not edited here: sent back as they are, or the server clears them.
                preferredDays: kept?.preferredDays, mobilityChecks: kept?.mobilityChecks,
                maxAddedLoadKg: kept?.maxAddedLoadKg, smallestPlateKg: kept?.smallestPlateKg)
            kept = try await sync.updateTrainingProfile(update)
            loaded.sessionsPerWeek = form.sessionsPerWeek
            loaded.sessionMinutes = form.sessionMinutes
            loaded.equipment = form.equipment
            loaded.bodyweightKg = form.bodyweightKg
        }
        await plan?.refresh()
    }
}
