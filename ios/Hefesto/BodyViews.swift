import HefestoPlan
import HefestoSync
import SwiftUI

// The body in the app (ADR 0028): the regions the planner keeps a state or a
// constraint for, a pain report, the warning-sign questions of a region,
// exertion symptoms and a professional's clearance. Names, questions, advice
// and reasons are the knowledge base's texts, shown verbatim. The app adds no
// advice of its own and never names a diagnosis.

/// A sheet the body screen opens on, as a demo screen does.
enum BodySheet: Hashable, Identifiable {
    case pain, symptoms
    var id: Self { self }
}

/// The regions with a complaint or a constraint, and the events about them.
struct BodyView: View {
    var opening: BodySheet?

    @Environment(AppModel.self) private var model
    @State private var reportingPain = false
    @State private var reportingSymptoms = false
    @State private var askingFlags: RegionRoute?
    @State private var clearing: BodyRegion?
    @State private var outcome: BodyOutcome?
    @State private var failure: String?

    var body: some View {
        let state = model.body
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                VStack(spacing: 10) {
                    Button { reportingPain = true } label: {
                        Label("Report pain", systemImage: "hand.raised")
                    }
                    .buttonStyle(PrimaryButtonStyle())
                    Button { reportingSymptoms = true } label: {
                        Label("Symptoms during training", systemImage: "heart.text.square")
                    }
                    .buttonStyle(SecondaryButtonStyle())
                }

                if let outcome { OutcomeNotes(outcome: outcome) }
                if let failure {
                    Text(verbatim: failure).font(.detailText).foregroundStyle(Palette.text)
                }

                status(state.status, empty: state.regions.isEmpty)

                ForEach(state.regions) { region in
                    RegionCard(
                        region: region,
                        onFlags: { askingFlags = RegionRoute(id: region.id) },
                        onClearance: { clearing = region })
                }

                if !state.disclaimer.isEmpty {
                    Text(verbatim: state.disclaimer).font(.fine).foregroundStyle(Palette.textSecondary)
                }
            }
            .padding(16)
        }
        .themedScreen()
        .navigationTitle("Body")
        .navigationBarTitleDisplayMode(.inline)
        .task {
            reportingPain = opening == .pain
            reportingSymptoms = opening == .symptoms
            await state.load()
        }
        .refreshable { await state.load() }
        .sheet(isPresented: $reportingPain) {
            PainReportSheet { outcome = $0; askFor($0) }
        }
        .sheet(isPresented: $reportingSymptoms) {
            SymptomsSheet { outcome = $0 }
        }
        .sheet(item: $askingFlags) { route in
            RedFlagSheet(regionId: route.id) { outcome = $0 }
        }
        .confirmationDialog(
            "Did a professional clear this region?",
            isPresented: Binding(get: { clearing != nil }, set: { if !$0 { clearing = nil } }),
            titleVisibility: .visible, presenting: clearing
        ) { region in
            Button("Yes, I was cleared") { Task { await clear(region) } }
            Button("Cancel", role: .cancel) { clearing = nil }
        } message: { _ in
            Text("Confirm only what a doctor or physiotherapist told you. The region then starts its return at the first step.")
        }
    }

    @ViewBuilder
    private func status(_ status: BodyModel.Status, empty: Bool) -> some View {
        switch status {
        case .loading:
            if empty { ProgressView().frame(maxWidth: .infinity) }
        case .offline:
            Text("The body's state needs a connection. Reporting pain or symptoms does too.")
                .font(.detailText).foregroundStyle(Palette.textSecondary)
        case .needsOnboarding:
            Text("The body's state comes with your plan. Answer the onboarding first.")
                .font(.detailText).foregroundStyle(Palette.textSecondary)
        case .unavailable:
            Text("The planner is not available right now. Try again later.")
                .font(.detailText).foregroundStyle(Palette.textSecondary)
        case .ready:
            if empty {
                Text("No region has a complaint or a constraint.")
                    .font(.detailText).foregroundStyle(Palette.textSecondary)
            }
        }
    }

    /// After a pain report, the planner may ask a region's warning signs;
    /// the next sheet opens once the report's is gone.
    private func askFor(_ outcome: BodyOutcome) {
        guard let region = outcome.askRedFlags.first else { return }
        Task {
            try? await Task.sleep(for: .milliseconds(600))
            askingFlags = RegionRoute(id: region)
        }
    }

    private func clear(_ region: BodyRegion) async {
        clearing = nil
        failure = nil
        do {
            outcome = try await model.body.confirmClearance(region: region.id)
        } catch {
            failure = BodyText.failure(error)
        }
    }
}

struct RegionRoute: Identifiable, Hashable {
    let id: String
}

/// One region: its state, what a professional advised, and what to do.
struct RegionCard: View {
    let region: BodyRegion
    let onFlags: () -> Void
    let onClearance: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(alignment: .firstTextBaseline) {
                Text(verbatim: BodyText.title(region.name)).font(.rowTitle).foregroundStyle(Palette.text)
                Spacer(minLength: 8)
                Chip(verbatim: BodyText.state(region))
            }
            if !region.tracked {
                Text("Without your consent Hefesto keeps no state for this region. Locks and exclusions still hold.")
                    .font(.detailText).foregroundStyle(Palette.textSecondary)
            }
            if let line = BodyText.constraints(region) {
                Text(verbatim: line).font(.detailText).foregroundStyle(Palette.text)
            }
            if let until = region.restUntil {
                Text("No load on this region until \(PlanText.date(until), format: .dateTime.day().month(.wide)).")
                    .font(.detailText).foregroundStyle(Palette.text)
            }
            if let referral = region.referral {
                Text(verbatim: BodyText.referral(referral)).font(.detailText).foregroundStyle(Palette.text)
            }
            if !region.restrictions.isEmpty {
                VStack(alignment: .leading, spacing: 4) {
                    CapsLabel("What a professional advised", isHeader: false)
                    ForEach(region.restrictions, id: \.self) { r in
                        Text(verbatim: BodyText.restriction(r)).font(.detailText).foregroundStyle(Palette.text)
                    }
                }
            }
            HStack(spacing: 16) {
                if !region.redFlags.isEmpty {
                    Button("Warning signs", action: onFlags)
                }
                if region.clearable {
                    Button("I was cleared", action: onClearance)
                }
            }
            .font(.bodyMedium)
            .foregroundStyle(Palette.ember)
            .frame(minHeight: 44)
        }
        .card()
    }
}

/// What an event changed, in the knowledge base's words.
struct OutcomeNotes: View {
    let outcome: BodyOutcome

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            if outcome.trainingStopped {
                Text("Training is paused. Your plan says why.").font(.rowTitle).foregroundStyle(Palette.text)
            }
            ForEach(Array(outcome.notes.enumerated()), id: \.offset) { _, note in
                Text(verbatim: note.text).font(.detailText).foregroundStyle(Palette.text)
            }
            if !outcome.trainingStopped && outcome.notes.isEmpty {
                Text("Sent. Your plan takes it into account.").font(.detailText).foregroundStyle(Palette.textSecondary)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .card()
    }
}

// MARK: - Pain

/// One pain rating: where, when, how strong, and what the planner asks
/// about the moment (spec §8.6). Optional, neutral, without streak effects.
struct PainReportSheet: View {
    /// The log session the pain belongs to, if reported from one.
    var sessionId: String?
    var initialTimepoint: String?
    let onSent: (BodyOutcome) -> Void

    @Environment(AppModel.self) private var model
    @Environment(\.dismiss) private var dismiss
    @State private var region: String?
    @State private var timepoint: String?
    @State private var nrs: String?
    @State private var detail: Bool?
    @State private var sending = false
    @State private var failure: String?

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 24) {
                    PageTitle("Report pain", detail: "Where it hurts, when, and how much. The plan adapts to it; nothing here counts against you.")
                    ChoiceQuestion(
                        question: String(localized: "Where?"),
                        choices: (model.body.catalogue?.regions ?? []).map { Choice(id: $0.id, label: BodyText.title($0.name)) },
                        selection: $region)
                    ChoiceQuestion(
                        question: String(localized: "When?"), choices: BodyText.timepoints, selection: $timepoint, columns: 2)
                    ChoiceQuestion(
                        question: String(localized: "How strong, from 0 (none) to 10 (the worst imaginable)?"),
                        choices: Choice.scale, selection: $nrs, columns: 6)
                    if let question = BodyText.detailQuestion(timepoint) {
                        YesNoQuestion(question: question, answer: $detail)
                    }
                    if let failure {
                        Text(verbatim: failure).font(.detailText).foregroundStyle(Palette.text)
                    }
                    Button {
                        Task { await send() }
                    } label: {
                        if sending {
                            ProgressView().tint(Palette.emberInk)
                        } else {
                            Text("Send report")
                        }
                    }
                    .buttonStyle(PrimaryButtonStyle())
                    .disabled(region == nil || timepoint == nil || nrs == nil || sending)
                }
                .padding(20)
            }
            .themedScreen()
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
            }
            .task {
                if timepoint == nil { timepoint = initialTimepoint }
                if model.body.catalogue == nil { await model.body.load() }
            }
            .onChange(of: timepoint) { detail = nil }
        }
    }

    private func send() async {
        guard let region, let timepoint, let value = nrs.flatMap({ Double($0) }) else { return }
        sending = true
        failure = nil
        var report = PainReport(region: region, timepoint: timepoint, nrs: value, at: Date(), sessionId: sessionId)
        switch timepoint {
        case "warmup": report.persistedOver15min = detail
        case "after": report.lastedOver1h = detail
        case "during": report.suddenSharp = detail
        default: break
        }
        do {
            let outcome = try await model.body.reportPain(report)
            sending = false
            dismiss()
            onSent(outcome)
        } catch {
            sending = false
            failure = BodyText.failure(error)
        }
    }
}

// MARK: - Warning signs

/// A region's warning-sign questions, as the planner asks them. A yes shows
/// its advice at once; every question is answered before sending.
struct RedFlagSheet: View {
    let regionId: String
    let onSent: (BodyOutcome) -> Void

    @Environment(AppModel.self) private var model
    @Environment(\.dismiss) private var dismiss
    @State private var answers: [String: Bool] = [:]
    @State private var sending = false
    @State private var failure: String?

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 24) {
                    if let region = asked {
                        PageTitle("Warning signs", detail: nil)
                        Text(verbatim: BodyText.title(region.name)).font(.rowTitle).foregroundStyle(Palette.textSecondary)
                        ForEach(region.flags) { flag in
                            VStack(alignment: .leading, spacing: 12) {
                                YesNoQuestion(question: flag.question, answer: binding(flag.id))
                                if answers[flag.id] == true {
                                    if let followup = flag.followup {
                                        YesNoQuestion(question: followup.question, answer: binding(followup.id))
                                        Advice(text: answers[followup.id] == true ? followup.advice : flag.advice)
                                    } else {
                                        Advice(text: flag.advice)
                                    }
                                }
                            }
                        }
                        if let failure {
                            Text(verbatim: failure).font(.detailText).foregroundStyle(Palette.text)
                        }
                        Button {
                            Task { await send(region) }
                        } label: {
                            if sending {
                                ProgressView().tint(Palette.emberInk)
                            } else {
                                Text("Send answers")
                            }
                        }
                        .buttonStyle(PrimaryButtonStyle())
                        .disabled(!complete(region) || sending)
                        Text(verbatim: model.body.disclaimer).font(.fine).foregroundStyle(Palette.textSecondary)
                    } else {
                        ProgressView().frame(maxWidth: .infinity)
                    }
                }
                .padding(20)
            }
            .themedScreen()
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
            }
            .task { if model.body.region(regionId) == nil { await model.body.load() } }
        }
    }

    /// The region's questions as the server lists them; for a region it
    /// does not list yet, the body map's, as asked of an adult.
    private var asked: (name: String, flags: [OnboardingCatalogue.Flag])? {
        if let region = model.body.region(regionId) { return (region.name, region.redFlags) }
        guard let region = model.body.catalogue?.region(regionId) else { return nil }
        return (region.name, region.flags.filter { !$0.minorOnly })
    }

    private func binding(_ id: String) -> Binding<Bool?> {
        Binding(get: { answers[id] }, set: { answers[id] = $0 })
    }

    /// Every question answered, and a follow-up exactly when its question
    /// was answered yes.
    private func complete(_ region: (name: String, flags: [OnboardingCatalogue.Flag])) -> Bool {
        region.flags.allSatisfy { flag in
            guard let yes = answers[flag.id] else { return false }
            guard yes, let followup = flag.followup else { return true }
            return answers[followup.id] != nil
        }
    }

    private func send(_ region: (name: String, flags: [OnboardingCatalogue.Flag])) async {
        var sent: [String: Bool] = [:]
        for flag in region.flags {
            sent[flag.id] = answers[flag.id] ?? false
            if sent[flag.id] == true, let followup = flag.followup { sent[followup.id] = answers[followup.id] ?? false }
        }
        sending = true
        failure = nil
        do {
            let outcome = try await model.body.answerRedFlags(region: regionId, answers: sent)
            sending = false
            dismiss()
            onSent(outcome)
        } catch {
            sending = false
            failure = BodyText.failure(error)
        }
    }
}

// MARK: - Exertion symptoms

/// Chest pain, unusual shortness of breath, dizziness or fainting under
/// exertion (RF-10): the knowledge base's question and advice, and the stop.
struct SymptomsSheet: View {
    let onSent: (BodyOutcome) -> Void

    @Environment(AppModel.self) private var model
    @Environment(\.dismiss) private var dismiss
    @State private var sending = false
    @State private var failure: String?

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    PageTitle("Symptoms during training", detail: nil)
                    if let exertion = model.body.catalogue?.exertion {
                        QuestionText(verbatim: exertion.question)
                        Advice(text: exertion.advice)
                    } else {
                        ProgressView().frame(maxWidth: .infinity)
                    }
                    Text("Reporting pauses your training at once, until you confirm that a professional has cleared you.")
                        .font(.detailText).foregroundStyle(Palette.textSecondary)
                    if let failure {
                        Text(verbatim: failure).font(.detailText).foregroundStyle(Palette.text)
                    }
                    Button(role: .destructive) {
                        Task { await send() }
                    } label: {
                        if sending {
                            ProgressView().tint(Palette.emberInk)
                        } else {
                            Text("Report and pause training")
                        }
                    }
                    .buttonStyle(PrimaryButtonStyle())
                    .disabled(sending)
                }
                .padding(20)
            }
            .themedScreen()
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
            }
            .task { if model.body.catalogue == nil { await model.body.load() } }
        }
    }

    private func send() async {
        sending = true
        failure = nil
        do {
            let outcome = try await model.body.reportSymptoms()
            sending = false
            dismiss()
            onSent(outcome)
        } catch {
            sending = false
            failure = BodyText.failure(error, symptoms: true)
        }
    }
}

// MARK: - Words

/// The body's states and events in the athlete's words. The region names,
/// questions and advice come from the knowledge base.
enum BodyText {
    /// "innen am Ellbogen" as a title: "Innen am Ellbogen".
    static func title(_ name: String) -> String {
        name.prefix(1).uppercased() + name.dropFirst()
    }

    static func state(_ region: BodyRegion) -> String {
        if region.locked { return String(localized: "Paused") }
        if let stage = region.rampStage { return String(localized: "Return, step \(stage) of 5") }
        if region.state == "normal" { return String(localized: "No limits") }
        return String(localized: "Not tracked")
    }

    static func constraints(_ region: BodyRegion) -> String? {
        if region.constraints.contains("plan_stopped") {
            return String(localized: "Its warning signs pause all training.")
        }
        if region.locked { return String(localized: "The plan leaves this region out until you confirm a clearance.") }
        if region.constraints.contains("region_excluded") {
            return String(localized: "The plan leaves exercises for this region out.")
        }
        if region.rampStage == 0 { return String(localized: "The plan leaves this region out for now.") }
        return nil
    }

    /// A professional's restriction, as the onboarding asked it.
    static func restriction(_ key: String) -> String {
        OnboardingText.restrictions.first { $0.key == key }?.label ?? key
    }

    static func referral(_ referral: String) -> String {
        referral == "advise"
            ? String(localized: "An assessment by a professional is recommended.")
            : String(localized: "An assessment by a professional can help.")
    }

    static let timepoints: [Choice] = [
        Choice(id: "before_session", label: String(localized: "Before training")),
        Choice(id: "warmup", label: String(localized: "In the warm-up")),
        Choice(id: "during", label: String(localized: "During training")),
        Choice(id: "after", label: String(localized: "After training")),
        Choice(id: "next_morning", label: String(localized: "This morning")),
        Choice(id: "daily", label: String(localized: "In everyday life")),
    ]

    /// What the planner asks about the moment (spec §8.6).
    static func detailQuestion(_ timepoint: String?) -> String? {
        switch timepoint {
        case "warmup": String(localized: "Has it lasted longer than 15 minutes?")
        case "after": String(localized: "Did it last longer than an hour?")
        case "during": String(localized: "Did it come suddenly and sharp?")
        default: nil
        }
    }

    /// An outcome in a few lines, for an alert.
    static func summary(_ outcome: BodyOutcome) -> String {
        var lines = outcome.notes.map(\.text)
        if outcome.trainingStopped { lines.insert(String(localized: "Training is paused. Your plan says why."), at: 0) }
        return lines.isEmpty ? String(localized: "Sent. Your plan takes it into account.") : lines.joined(separator: "\n\n")
    }

    static func failure(_ error: any Error, symptoms: Bool = false) -> String {
        if let e = error as? BodyError {
            return switch e {
            case .consentRequired:
                String(localized: "Pain reports are health data. Without your consent to keep health data, Hefesto does not take them.")
            case .onboardingRequired:
                String(localized: "This needs your plan. Answer the onboarding first.")
            case .unknownRegion, .invalid:
                String(localized: "The report could not be taken. Check your answers.")
            case .unavailable:
                String(localized: "The planner is not available right now. Try again later.")
            case .refused:
                String(localized: "It could not be sent. Try again later.")
            }
        }
        if symptoms {
            return String(localized: "No connection. Stop training now and follow the advice above; send the report when you are online.")
        }
        return String(localized: "No connection. Reports need one; try again when you are online.")
    }
}
