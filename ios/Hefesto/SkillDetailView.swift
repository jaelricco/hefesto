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
        ScrollView {
            if let skill, let map {
                VStack(alignment: .leading, spacing: 0) {
                    header(skill, map).padding(.horizontal, 20)
                    if skill.levels.count > 1 {
                        LevelLadder(skill: skill, map: map).padding(.horizontal, 16).padding(.top, 18)
                    }
                    CapsLabel("Levels").padding(.horizontal, 20).padding(.top, 24).padding(.bottom, 10)
                    VStack(spacing: 10) {
                        ForEach(skill.levels) { level in
                            LevelCard(level: level, state: map.states[level.id], exerciseNames: exerciseNames) {
                                attesting = level
                            }
                        }
                    }
                    .padding(.horizontal, 16)
                    if !skill.skill.commonFaults.isEmpty { faults(skill).padding(.top, 24) }
                    injuries.padding(.top, 24)
                }
                .padding(.bottom, 24)
            }
        }
        .themedScreen()
        .navigationTitle(skill.map { Text(verbatim: $0.skill.name) } ?? Text(verbatim: ""))
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            // The name is the page's own title; the bar keeps only the way back.
            ToolbarItem(placement: .principal) { EmptyView() }
        }
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
            await model.refreshSkillNotes(slug: slug)
            do { for try await n in model.db.observeSkillNotes(slug: slug) { notes = n } } catch {}
        }
    }

    private var exerciseNames: [String: String] {
        Dictionary(exercises.map { ($0.slug, $0.name) }, uniquingKeysWith: { a, _ in a })
    }

    @ViewBuilder
    private func header(_ s: SkillWithLevels, _ map: SkillMapData) -> some View {
        let standing = map.standing(of: s)
        VStack(alignment: .leading, spacing: 0) {
            HStack(spacing: 6) {
                if s.skill.isMilestone { Chip("Milestone", tone: .gold) }
                Chip(verbatim: FamilyText.label(s.skill.family))
                Chip(verbatim: StandingText.label(standing), tone: standing == .locked ? .neutral : .ember)
            }
            Text(verbatim: s.skill.name)
                .font(Typeface.condensed(46, .bold, relativeTo: .largeTitle))
                .foregroundStyle(Palette.text)
                .padding(.top, 10)
                .accessibilityAddTraits(.isHeader)
            if !s.skill.summary.isEmpty {
                Text(verbatim: s.skill.summary)
                    .font(Typeface.text(16, relativeTo: .body))
                    .foregroundStyle(Palette.textSecondary)
                    .padding(.top, 6)
            }
            DifficultyMeter(tier: s.skill.difficultyTier).padding(.top, 14)
            if s.skill.status == "draft_placeholder" {
                Text("This skill's content is still being researched.")
                    .font(.fine)
                    .foregroundStyle(Palette.ember)
                    .padding(.top, 10)
            }
        }
    }

    private func faults(_ s: SkillWithLevels) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            CapsLabel("Common faults").padding(.horizontal, 20)
            VStack(alignment: .leading, spacing: 0) {
                ForEach(Array(s.skill.commonFaults.enumerated()), id: \.offset) { i, fault in
                    if i > 0 { Divider().overlay(Palette.hairline).padding(.leading, 38) }
                    HStack(alignment: .firstTextBaseline, spacing: 12) {
                        Text(verbatim: "–").font(.rowTitle).foregroundStyle(Palette.ember).accessibilityHidden(true)
                        Text(verbatim: fault).font(Typeface.text(16, relativeTo: .body)).foregroundStyle(Palette.text)
                    }
                    .padding(.horizontal, 16)
                    .padding(.vertical, 12)
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(Palette.surface, in: .rect(cornerRadius: 16))
            .padding(.horizontal, 16)
        }
    }

    @ViewBuilder
    private var injuries: some View {
        if let notes, !notes.injuries.isEmpty {
            VStack(alignment: .leading, spacing: 10) {
                CapsLabel("Injury awareness").padding(.horizontal, 20)
                VStack(alignment: .leading, spacing: 0) {
                    // The disclaimer comes first and with every note, as the API sends it.
                    Label {
                        Text(verbatim: notes.disclaimer)
                    } icon: {
                        Image(systemName: "info.circle")
                    }
                    .font(.meta)
                    .foregroundStyle(Palette.textSecondary)
                    .padding(16)
                    ForEach(notes.injuries, id: \.name) { injury in
                        Divider().overlay(Palette.hairline)
                        DisclosureGroup {
                            VStack(alignment: .leading, spacing: 8) {
                                if !injury.details.isEmpty { Text(verbatim: injury.details) }
                                if !injury.riskFactors.isEmpty {
                                    Text("Risk factors").font(.rowTitle)
                                    ForEach(injury.riskFactors, id: \.self) { Text(verbatim: "• \($0)") }
                                }
                                if !injury.earlySigns.isEmpty {
                                    Text("Early signs").font(.rowTitle)
                                    ForEach(injury.earlySigns, id: \.self) { Text(verbatim: "• \($0)") }
                                }
                                if !injury.prehabExerciseSlugs.isEmpty {
                                    Text("Prehab").font(.rowTitle)
                                    Text(verbatim: injury.prehabExerciseSlugs.map { exerciseNames[$0] ?? $0 }
                                        .joined(separator: ", "))
                                }
                            }
                            .font(.detailText)
                            .foregroundStyle(Palette.text)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .padding(.top, 8)
                        } label: {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(verbatim: injury.name).font(.rowTitle).foregroundStyle(Palette.text)
                                Text(verbatim: RegionText.label(injury.region)).font(.fine)
                                    .foregroundStyle(Palette.textSecondary)
                            }
                        }
                        .padding(16)
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .background(Palette.surface, in: .rect(cornerRadius: 16))
                .padding(.horizontal, 16)
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

/// The levels in order, as steps: where the athlete is on the way.
struct LevelLadder: View {
    let skill: SkillWithLevels
    let map: SkillMapData

    var body: some View {
        let unlocked = skill.levels.filter { map.state(of: $0.id) == "unlocked" }.count
        VStack(alignment: .leading, spacing: 12) {
            ScrollView(.horizontal, showsIndicators: false) {
                HStack(alignment: .top, spacing: 0) {
                    ForEach(Array(skill.levels.enumerated()), id: \.element.id) { i, level in
                        if i > 0 {
                            Rectangle()
                                .fill(map.state(of: skill.levels[i - 1].id) == "unlocked" ? Palette.gold : Palette.lockedStroke)
                                .frame(width: 32, height: 2)
                                .padding(.top, 13)
                        }
                        VStack(spacing: 6) {
                            step(map.state(of: level.id))
                            Text(verbatim: shortName(level))
                                .font(Typeface.text(14, .semibold, relativeTo: .subheadline))
                                .foregroundStyle(Palette.text)
                                .multilineTextAlignment(.center)
                                .fixedSize(horizontal: false, vertical: true)
                        }
                        .frame(width: 96)
                    }
                }
            }
            Text("\(unlocked) of \(skill.levels.count) levels unlocked")
                .font(.meta)
                .foregroundStyle(Palette.textSecondary)
        }
        .card()
        .accessibilityElement(children: .combine)
    }

    @ViewBuilder
    private func step(_ state: String) -> some View {
        ZStack {
            switch state {
            case "unlocked":
                Circle().fill(Palette.gold)
                Image(systemName: "checkmark").font(.system(size: 13, weight: .bold)).foregroundStyle(Palette.emberInk)
            case "in_progress":
                Circle().strokeBorder(Palette.ember, lineWidth: 2)
                Circle().fill(Palette.ember).padding(8)
            case "available":
                Circle().strokeBorder(Palette.text, lineWidth: 2)
            default:
                Circle().strokeBorder(Palette.lockedStroke, lineWidth: 1.5)
                Image(systemName: "lock.fill").font(.system(size: 11)).foregroundStyle(Palette.lockedLabel)
            }
        }
        .frame(width: 28, height: 28)
    }

    /// "Tuck Front Lever" on the Front Lever page is "Tuck".
    private func shortName(_ level: SkillLevel) -> String {
        let short = level.name.replacingOccurrences(of: skill.skill.name, with: "")
            .trimmingCharacters(in: .whitespaces)
        return short.isEmpty ? level.name : short
    }
}

/// How hard a skill is, as ten bars and in words.
struct DifficultyMeter: View {
    let tier: Int

    var body: some View {
        HStack(spacing: 12) {
            CapsLabel("Difficulty", isHeader: false)
            HStack(spacing: 4) {
                ForEach(1...10, id: \.self) { i in
                    Capsule().fill(i <= tier ? Palette.ember : Palette.hairline).frame(width: 14, height: 6)
                }
            }
            .accessibilityHidden(true)
            Text("\(tier) of 10")
                .font(Typeface.condensed(16, .semibold, relativeTo: .subheadline))
                .monospacedDigit()
                .foregroundStyle(Palette.text)
        }
        .accessibilityElement(children: .combine)
    }
}

/// One level: its state, what unlocks it, and the athlete's best.
struct LevelCard: View {
    let level: SkillLevel
    let state: LevelState?
    let exerciseNames: [String: String]
    let onAttest: () -> Void

    private var status: String { state?.state ?? "locked" }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .center, spacing: 10) {
                Image(systemName: icon).font(.system(size: 20)).foregroundStyle(tint).accessibilityHidden(true)
                Text(verbatim: level.name).font(Typeface.text(18, .semibold, relativeTo: .headline))
                    .foregroundStyle(Palette.text)
                Spacer(minLength: 4)
                Chip(verbatim: LevelText.state(status), tone: status == "unlocked" ? .gold : .neutral)
            }
            if !level.details.isEmpty {
                Text(verbatim: level.details).font(.detailText).foregroundStyle(Palette.textSecondary)
            }
            ForEach(Array(LevelText.criteria(level.criteria, names: exerciseNames).enumerated()), id: \.offset) {
                Text(verbatim: $0.element).font(.detailText).foregroundStyle(Palette.textSecondary)
            }
            if state?.bestValue != nil || (status == "unlocked" && state?.firstAchievedAt != nil) {
                HStack(alignment: .top, spacing: 12) {
                    if let best = state?.bestValue, let unit = state?.bestUnit {
                        stat("Best") {
                            Text(verbatim: LevelText.value(best, unit: unit))
                                .font(.metricSmall)
                                .monospacedDigit()
                                .foregroundStyle(Palette.text)
                        }
                    }
                    if let achieved = state?.firstAchievedAt, status == "unlocked" {
                        stat(state?.verification == "self_attested" ? "Self-attested on" : "Unlocked on") {
                            Text(achieved, format: .dateTime.day().month(.abbreviated).year())
                                .font(.rowTitle)
                                .foregroundStyle(Palette.text)
                                .frame(minHeight: 30)
                        }
                    }
                }
                .padding(.top, 4)
            }
            if status == "available" || status == "in_progress" {
                Button("I can already do this", action: onAttest)
                    .buttonStyle(SecondaryButtonStyle(height: 48))
                    .padding(.top, 4)
            }
        }
        .card()
        .accessibilityElement(children: .combine)
    }

    private func stat<V: View>(_ label: LocalizedStringKey, @ViewBuilder value: () -> V) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            CapsLabel(label, isHeader: false)
            value()
        }
        .frame(maxWidth: .infinity, alignment: .leading)
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
        case "unlocked": Palette.gold
        case "in_progress": Palette.ember
        case "available": Palette.text
        default: Palette.lockedLabel
        }
    }
}

/// A skill family's name, from the vocabulary in content/families.yaml.
enum FamilyText {
    static func label(_ family: String) -> String {
        switch family {
        case "push": String(localized: "Push")
        case "pull": String(localized: "Pull")
        case "core": String(localized: "Core")
        case "legs": String(localized: "Legs")
        case "handstand": String(localized: "Handstand")
        case "dynamic": String(localized: "Dynamic")
        case "mobility": String(localized: "Mobility")
        default: family
        }
    }
}

/// A body region as injury notes name it.
enum RegionText {
    static func label(_ region: String) -> String {
        switch region {
        case "elbow": String(localized: "Elbow")
        case "shoulder": String(localized: "Shoulder")
        case "wrist": String(localized: "Wrist")
        case "lower_back": String(localized: "Lower back")
        case "knee": String(localized: "Knee")
        case "neck": String(localized: "Neck")
        case "hip": String(localized: "Hip")
        default: region
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
