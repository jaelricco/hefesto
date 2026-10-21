import Foundation
import HefestoPlan
import HefestoSync
import SwiftUI

// The consent to keep health data (ADR 0031): its text, the athlete's
// decisions, a withdrawal and a new grant. The text is the app's and has a
// version that goes with every decision. A changed word is a new version,
// archived in docs/consent/; ios/scripts/check-strings.py holds the archive,
// the catalogue and this file together.

/// The consent text's version and the language it is shown in.
enum HealthConsentText {
    /// Change it with any word of `ConsentTextView`, and archive the new text.
    static let version = "1"

    /// The text as the athlete reads it now.
    static var current: ConsentText {
        let shown = Bundle.main.preferredLocalizations.first ?? "en"
        let tag = shown.range(of: "^[a-z]{2,3}(-[A-Za-z0-9]{2,8})?$", options: .regularExpression) != nil ? shown : "en"
        return ConsentText(version: version, locale: tag)
    }
}

/// The consent text: the question and what it means. The onboarding and the
/// health data screen show the same words.
struct ConsentTextView: View {
    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            QuestionText("May Hefesto keep your health answers?")
            Text("Complaints and health answers let the planner work around them. Without your consent the safety questions are still asked, but only used for this plan and not kept, and the planner plans conservatively.")
                .font(.detailText)
                .foregroundStyle(Palette.textSecondary)
                .fixedSize(horizontal: false, vertical: true)
        }
    }
}

/// The consent: what it covers, what the athlete decided and when, and a
/// withdrawal or a new grant. Both need a connection.
struct HealthDataView: View {
    @Environment(AppModel.self) private var model
    @State private var confirmingWithdrawal = false
    @State private var granting = false
    @State private var sending = false
    @State private var outcome: BodyOutcome?
    @State private var failure: String?

    var body: some View {
        let consent = model.consent
        ScrollView {
            VStack(alignment: .leading, spacing: 24) {
                status(consent.status, loaded: consent.granted != nil)
                if let granted = consent.granted {
                    decision(granted, latest: consent.latest)
                    ConsentTextView()
                    VStack(alignment: .leading, spacing: 8) {
                        CapsLabel("Health data")
                        Text("Complaints and the regions they concern, pain reports, the state of each region in its return, and your answers to the safety questions.")
                            .font(.detailText)
                            .foregroundStyle(Palette.text)
                            .fixedSize(horizontal: false, vertical: true)
                    }
                    if let outcome { OutcomeNotes(outcome: outcome) }
                    if granted, outcome != nil, model.body.regions.contains(where: { $0.rampStage == 0 }) {
                        stageZero
                    }
                    if let failure {
                        Text(verbatim: failure).font(.detailText).foregroundStyle(Palette.text)
                    }
                    action(granted)
                    record(consent.decisions)
                }
            }
            .padding(16)
        }
        .themedScreen()
        .navigationTitle("Health data")
        .navigationBarTitleDisplayMode(.inline)
        .task { await consent.load() }
        .refreshable { await consent.load() }
        .confirmationDialog("Withdraw your consent?", isPresented: $confirmingWithdrawal, titleVisibility: .visible) {
            Button("Withdraw and delete", role: .destructive) { Task { await withdraw() } }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("Hefesto deletes your complaints, pain reports, the state of each region and your answers to the safety questions, on the server and on this phone. Locked or excluded regions and a stop of training stay: they protect you. Your plan then plans more cautiously.")
        }
        .sheet(isPresented: $granting) {
            GrantConsentSheet { outcome = $0 }
        }
    }

    @ViewBuilder
    private func status(_ status: ConsentModel.Status, loaded: Bool) -> some View {
        switch status {
        case .loading:
            if !loaded { ProgressView().frame(maxWidth: .infinity) }
        case .offline:
            note("Your consent and its record need a connection. Changing it does too.")
        case .needsOnboarding:
            note("You decide on the consent in the onboarding. Answer it first.")
        case .unavailable:
            note("The planner is not available right now. Try again later.")
        case .ready:
            EmptyView()
        }
    }

    private func decision(_ granted: Bool, latest: ConsentDecision?) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(granted ? "Hefesto keeps your health answers." : "Hefesto does not keep your health answers.")
                .font(.rowTitle)
                .foregroundStyle(Palette.text)
                .fixedSize(horizontal: false, vertical: true)
            if let latest {
                Text(verbatim: ConsentWording.line(latest))
                    .font(.detailText)
                    .foregroundStyle(Palette.textSecondary)
                    .fixedSize(horizontal: false, vertical: true)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .card()
    }

    @ViewBuilder
    private func action(_ granted: Bool) -> some View {
        if granted {
            Button { confirmingWithdrawal = true } label: {
                if sending { ProgressView() } else { Text("Withdraw consent") }
            }
            .buttonStyle(SecondaryButtonStyle())
            .disabled(sending)
        } else {
            Button { granting = true } label: { Text("Give consent") }
                .buttonStyle(PrimaryButtonStyle())
        }
    }

    /// A region excluded until the grant starts its return at stage 0; green
    /// everyday pain and negative warning signs lead out of it (ADR 0015).
    private var stageZero: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("A region left out until now starts its return at step 0. To leave it, report its everyday pain under Body and answer its warning signs.")
                .font(.detailText)
                .foregroundStyle(Palette.text)
                .fixedSize(horizontal: false, vertical: true)
            NavigationLink { BodyView(opening: .pain) } label: {
                Label("Report pain", systemImage: "hand.raised")
            }
            .buttonStyle(SecondaryButtonStyle())
        }
        .card()
    }

    private func record(_ decisions: [ConsentDecision]) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            CapsLabel("Your decisions")
            ForEach(Array(decisions.enumerated()), id: \.offset) { _, d in
                VStack(alignment: .leading, spacing: 2) {
                    Text(d.granted ? "Given" : "Withdrawn").font(.bodyMedium).foregroundStyle(Palette.text)
                    Text(verbatim: ConsentWording.detail(d))
                        .font(.fine)
                        .foregroundStyle(Palette.textSecondary)
                        .fixedSize(horizontal: false, vertical: true)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .accessibilityElement(children: .combine)
            }
            if decisions.isEmpty {
                Text("No decision is recorded yet. The record starts with this version of the app.")
                    .font(.detailText)
                    .foregroundStyle(Palette.textSecondary)
            }
        }
    }

    private func note(_ key: LocalizedStringKey) -> some View {
        Text(key).font(.detailText).foregroundStyle(Palette.textSecondary)
    }

    private func withdraw() async {
        sending = true
        failure = nil
        do {
            outcome = try await model.consent.withdraw(text: HealthConsentText.current)
        } catch {
            failure = ConsentWording.failure(error)
        }
        sending = false
    }
}

/// A new grant asks again what was not kept without the consent: the safety
/// questions and the injuries of the last 12 months (ADR 0015).
struct GrantConsentSheet: View {
    let onGranted: (BodyOutcome) -> Void

    @Environment(AppModel.self) private var model
    @Environment(\.dismiss) private var dismiss
    @State private var sending = false
    @State private var failure: String?

    var body: some View {
        @Bindable var consent = model.consent
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 28) {
                    PageTitle(
                        "Give consent",
                        detail: "What was not kept without it is asked again: the safety questions, and injuries in the last 12 months.")
                    ConsentTextView()
                    if let catalogue = consent.catalogue {
                        VStack(alignment: .leading, spacing: 20) {
                            ForEach(catalogue.screening) { q in
                                YesNoQuestion(
                                    question: q.text,
                                    answer: Binding(get: { consent.screening[q.id] }, set: { consent.screening[q.id] = $0 }))
                            }
                        }
                        VStack(alignment: .leading, spacing: 10) {
                            QuestionText("Did an injury stop your training in the last 12 months?")
                            ForEach(catalogue.regions) { region in
                                ChoiceButton(label: region.name, selected: consent.pastInjuries.contains(region.id)) {
                                    if consent.pastInjuries.remove(region.id) == nil { consent.pastInjuries.insert(region.id) }
                                }
                            }
                        }
                    } else {
                        ProgressView().frame(maxWidth: .infinity)
                    }
                    if let failure {
                        Text(verbatim: failure).font(.detailText).foregroundStyle(Palette.text)
                    }
                    Button { Task { await grant() } } label: {
                        if sending { ProgressView().tint(Palette.emberInk) } else { Text("Give consent") }
                    }
                    .buttonStyle(PrimaryButtonStyle())
                    .disabled(!consent.canGrant || sending)
                }
                .padding(20)
            }
            .themedScreen()
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
            }
        }
    }

    private func grant() async {
        sending = true
        failure = nil
        do {
            let outcome = try await model.consent.grant(text: HealthConsentText.current)
            sending = false
            dismiss()
            onGranted(outcome)
        } catch {
            sending = false
            failure = ConsentWording.failure(error)
        }
    }
}

/// The consent's decisions and failures in words.
enum ConsentWording {
    /// The last decision, as the screen states it.
    static func line(_ d: ConsentDecision) -> String {
        let day = d.at.formatted(date: .long, time: .omitted)
        let said = d.granted ? String(localized: "Given on \(day).") : String(localized: "Withdrawn on \(day).")
        guard let text = d.text else { return said }
        return said + " " + String(localized: "Text version \(text.version).")
    }

    /// One decision of the record: when, where, and to which text.
    static func detail(_ d: ConsentDecision) -> String {
        var parts = [d.at.formatted(date: .long, time: .shortened)]
        if d.inOnboarding { parts.append(String(localized: "in the onboarding")) }
        if let text = d.text { parts.append(String(localized: "text version \(text.version)")) }
        return parts.joined(separator: " · ")
    }

    static func failure(_ error: any Error) -> String {
        guard let e = error as? BodyError else {
            return String(localized: "No connection. Changing the consent needs one; try again when you are online.")
        }
        return switch e {
        case .onboardingRequired: String(localized: "This needs your plan. Answer the onboarding first.")
        case .invalid, .unknownRegion, .consentRequired: String(localized: "The answers could not be taken. Check them.")
        case .unavailable: String(localized: "The planner is not available right now. Try again later.")
        case .refused: String(localized: "It could not be sent. Try again later.")
        }
    }
}
