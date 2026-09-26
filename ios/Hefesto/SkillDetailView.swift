import HefestoStore
import HefestoSync
import SwiftUI

/// A skill: its levels and what each one takes, the athlete's standing, and
/// injury notes that are only ever shown with their disclaimer (CLAUDE.md).
struct SkillDetailView: View {
    let skillId: String
    @Environment(AppModel.self) private var model
    @State private var map: SkillMapData?
    @State private var exercises: [Exercise] = []
    @State private var notes: SkillNotes?
    @State private var attesting: SkillLevel?
    @State private var working = false
    @State private var message: LocalizedStringKey?

    private var skill: SkillWithLevels? { map?.skills.first { $0.id == skillId } }

    var body: some View {
        List {
            if let skill, let map {
                header(skill, map)
                Section("Levels") {
                    ForEach(skill.levels) { level in
                        LevelRow(level: level, state: map.states[level.id], exerciseNames: exerciseNames) {
                            attesting = level
                        }
                    }
                }
                if !skill.skill.commonFaults.isEmpty {
                    Section("Common faults") {
                        ForEach(skill.skill.commonFaults, id: \.self) { Text(verbatim: $0) }
                    }
                }
                injuries(skill)
            }
        }
        .navigationTitle(skill.map { Text(verbatim: $0.skill.name) } ?? Text(verbatim: ""))
        .navigationBarTitleDisplayMode(.inline)
        .confirmationDialog(
            "Mark as achieved?", isPresented: .constant(attesting != nil), titleVisibility: .visible,
            presenting: attesting
        ) { level in
            Button("I can do \(level.name)") { Task { await attest(level) } }
            Button("Cancel", role: .cancel) { attesting = nil }
        } message: { _ in
            Text("For levels you reached before Hefesto, or that the log cannot show. It is marked as self-attested and earns no XP.")
        }
        .alert(message ?? "", isPresented: .constant(message != nil)) {
            Button("OK") { message = nil }
        }
        .task {
            do { for try await m in model.db.observeSkillMap() { map = m } } catch {}
        }
        .task {
            do { for try await e in model.db.observeExercises() { exercises = e } } catch {}
        }
        .task(id: skill?.skill.slug) {
            guard let slug = skill?.skill.slug else { return }
            try? await model.sync.refreshSkillNotes(slug: slug)
            do { for try await n in model.db.observeSkillNotes(slug: slug) { notes = n } } catch {}
        }
    }

    private var exerciseNames: [String: String] {
        Dictionary(exercises.map { ($0.slug, $0.name) }, uniquingKeysWith: { a, _ in a })
    }

    @ViewBuilder
    private func header(_ s: SkillWithLevels, _ map: SkillMapData) -> some View {
        Section {
            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Text(StandingText.label(map.standing(of: s))).font(.subheadline.weight(.semibold))
                        .foregroundStyle(map.standing(of: s) == .locked ? Color.secondary : Color.yellow)
                    Spacer()
                    if s.skill.isMilestone {
                        Label("Milestone", systemImage: "star.circle").font(.caption).foregroundStyle(.yellow)
                    }
                }
                if !s.skill.summary.isEmpty { Text(verbatim: s.skill.summary) }
                Text("Difficulty \(s.skill.difficultyTier) of 10").font(.caption).foregroundStyle(.secondary)
                if s.skill.status == "draft_placeholder" {
                    Text("This skill's content is still being researched.").font(.caption).foregroundStyle(.orange)
                }
            }
        }
    }

    @ViewBuilder
    private func injuries(_ s: SkillWithLevels) -> some View {
        if let notes, !notes.injuries.isEmpty {
            Section {
                // The disclaimer comes first and with every note, as the API sends it.
                Label {
                    Text(verbatim: notes.disclaimer)
                } icon: {
                    Image(systemName: "info.circle")
                }
                .font(.footnote)
                .foregroundStyle(.secondary)
                ForEach(notes.injuries, id: \.name) { injury in
                    DisclosureGroup {
                        VStack(alignment: .leading, spacing: 8) {
                            Text(verbatim: injury.details)
                            if !injury.riskFactors.isEmpty {
                                Text("Risk factors").font(.subheadline.weight(.semibold))
                                ForEach(injury.riskFactors, id: \.self) { Text(verbatim: "• \($0)") }
                            }
                            if !injury.earlySigns.isEmpty {
                                Text("Early signs").font(.subheadline.weight(.semibold))
                                ForEach(injury.earlySigns, id: \.self) { Text(verbatim: "• \($0)") }
                            }
                            if !injury.prehabExerciseSlugs.isEmpty {
                                Text("Prehab").font(.subheadline.weight(.semibold))
                                Text(verbatim: injury.prehabExerciseSlugs.map { exerciseNames[$0] ?? $0 }
                                    .joined(separator: ", "))
                            }
                        }
                        .font(.callout)
                    } label: {
                        VStack(alignment: .leading) {
                            Text(verbatim: injury.name).font(.headline)
                            Text(verbatim: injury.region).font(.caption).foregroundStyle(.secondary)
                        }
                    }
                }
            } header: {
                Text("Injury awareness")
            }
        }
    }

    private func attest(_ level: SkillLevel) async {
        attesting = nil
        working = true
        defer { working = false }
        do {
            try await model.sync.attest(levelId: level.id)
        } catch AttestError.prerequisitesMissing {
            message = "Unlock the levels before this one first."
        } catch {
            message = "Could not reach Hefesto. Check your connection."
        }
    }
}

/// One level: its state, what unlocks it, and the athlete's best.
struct LevelRow: View {
    let level: SkillLevel
    let state: LevelState?
    let exerciseNames: [String: String]
    let onAttest: () -> Void

    private var status: String { state?.state ?? "locked" }

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline) {
                Image(systemName: icon).foregroundStyle(tint).accessibilityHidden(true)
                Text(verbatim: level.name).font(.headline)
                Spacer()
                Text(LevelText.state(status)).font(.caption).foregroundStyle(tint)
            }
            if !level.details.isEmpty { Text(verbatim: level.details).font(.callout).foregroundStyle(.secondary) }
            ForEach(Array(LevelText.criteria(level.criteria, names: exerciseNames).enumerated()), id: \.offset) {
                Text(verbatim: $0.element).font(.callout)
            }
            if let best = state?.bestValue, let unit = state?.bestUnit {
                Text("Best: \(LevelText.value(best, unit: unit))").font(.caption.monospacedDigit())
            }
            if let achieved = state?.firstAchievedAt, status == "unlocked" {
                Text(state?.verification == "self_attested"
                     ? LocalizedStringKey("Self-attested on \(achieved.formatted(date: .abbreviated, time: .omitted))")
                     : "Unlocked on \(achieved.formatted(date: .abbreviated, time: .omitted))")
                    .font(.caption).foregroundStyle(.secondary)
            }
            if status == "available" || status == "in_progress" {
                Button("I can already do this", action: onAttest)
                    .font(.callout)
                    .frame(minHeight: 44)
            }
        }
        .padding(.vertical, 4)
        .accessibilityElement(children: .combine)
    }

    private var icon: String {
        switch status {
        case "unlocked": "checkmark.seal.fill"
        case "in_progress": "circle.lefthalf.filled"
        case "available": "circle"
        default: "lock.fill"
        }
    }

    private var tint: Color {
        switch status {
        case "unlocked": .yellow
        case "in_progress": .orange
        case "available": .primary
        default: .secondary
        }
    }
}

/// Unlock criteria in words, from the DSL the server sends (brief §6).
enum LevelText {
    static func state(_ s: String) -> String {
        switch s {
        case "unlocked": String(localized: "Unlocked")
        case "in_progress": String(localized: "In progress")
        case "available": String(localized: "Open")
        default: String(localized: "Locked")
        }
    }

    static func value(_ v: Double, unit: String) -> String {
        let n = v.formatted(.number.precision(.fractionLength(0...1)))
        switch unit {
        case "reps": return String(localized: "\(n) reps")
        case "hold_seconds": return String(localized: "\(n) s")
        case "distance_m": return String(localized: "\(n) m")
        default: return n
        }
    }

    static func criteria(_ c: UnlockCriteria, names: [String: String]) -> [String] {
        if c.isSelfAttestOnly { return [String(localized: "Self-attested: the log cannot show this one.")] }
        var lines = c.all.map { condition($0, names: names) }
        if !c.any.isEmpty {
            lines.append(String(localized: "One of:"))
            lines += c.any.map { "  " + condition($0, names: names) }
        }
        return lines
    }

    static func condition(_ c: UnlockCondition, names: [String: String]) -> String {
        let exercise = names[c.exercise] ?? c.exercise
        let amount = value(c.value, unit: c.measure)
        var parts = [c.op == ">" ? String(localized: "More than \(amount) of \(exercise)")
                                 : String(localized: "\(amount) of \(exercise)")]
        if c.assistance == "none" { parts.append(String(localized: "unassisted")) }
        if let q = c.minFormQuality { parts.append(String(localized: "form \(q)+")) }
        if let kg = c.minLoadKg { parts.append(String(localized: "at least \(kg.formatted()) kg added")) }
        if let kg = c.maxLoadKg { parts.append(String(localized: "at most \(kg.formatted()) kg added")) }
        if c.occurrences > 1 {
            if let days = c.withinDays {
                parts.append(String(localized: "\(c.occurrences) times within \(days) days"))
            } else {
                parts.append(String(localized: "\(c.occurrences) times"))
            }
        }
        return parts.joined(separator: ", ")
    }
}
