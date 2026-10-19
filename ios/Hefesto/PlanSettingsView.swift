import HefestoPlan
import HefestoSync
import SwiftUI

// The plan's settings after the onboarding (ADR 0029): goals, time, equipment
// and body weight, on the onboarding's own pages. Saving plans the week anew;
// a started session keeps what was done (ADR 0017).

struct PlanSettingsView: View {
    @Environment(AppModel.self) private var model
    @State private var saving = false
    @State private var failure: String?
    @State private var saved = false

    var body: some View {
        let settings = model.settings
        ScrollView {
            VStack(alignment: .leading, spacing: 32) {
                switch settings.status {
                case .loading:
                    ProgressView().frame(maxWidth: .infinity)
                case .offline:
                    note("Your plan's settings change online. Try again when you have a connection.")
                case .needsOnboarding:
                    note("Your plan's settings come with the onboarding. Answer it first.")
                case .unavailable:
                    note("The planner is not available right now. Try again later.")
                case .ready:
                    if let catalogue = settings.catalogue {
                        OnboardingGoalsPage(catalogue: catalogue, model: settings.editor)
                        OnboardingAvailabilityPage(model: settings.editor)
                        OnboardingEquipmentPage(model: settings.editor)
                        BodyweightQuestion(model: settings.editor)
                        outcome(settings)
                    }
                }
            }
            .padding(20)
        }
        .themedScreen()
        .navigationTitle("Plan settings")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .confirmationAction) {
                Button("Save") { Task { await save() } }
                    .fontWeight(.semibold)
                    .disabled(!settings.canSave || saving)
            }
        }
        .task { await settings.load() }
    }

    @ViewBuilder
    private func outcome(_ settings: PlanSettingsModel) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            if saved && !settings.hasChanges {
                Text("Saved. Your plan for the week is new.").font(.rowTitle).foregroundStyle(Palette.text)
            }
            ForEach(Array(settings.realism.enumerated()), id: \.offset) { _, note in
                Text(verbatim: note.text).font(.detailText).foregroundStyle(Palette.text)
            }
            if let failure {
                Text(verbatim: failure).font(.detailText).foregroundStyle(Palette.text)
            }
            Text("Saving plans your week anew. A session you started keeps what you did.")
                .font(.detailText)
                .foregroundStyle(Palette.textSecondary)
        }
    }

    private func note(_ key: LocalizedStringKey) -> some View {
        Text(key).font(.detailText).foregroundStyle(Palette.textSecondary)
    }

    private func save() async {
        saving = true
        failure = nil
        do {
            try await model.settings.save()
            saved = true
        } catch {
            failure = SettingsText.failure(error)
        }
        saving = false
    }
}

/// The body weight, as the onboarding asks it.
struct BodyweightQuestion: View {
    @Bindable var model: OnboardingModel

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            QuestionText("How much do you weigh?")
            HStack(alignment: .firstTextBaseline, spacing: 6) {
                TextField("Body weight", value: $model.form.bodyweightKg, format: .number.precision(.fractionLength(0...1)))
                    .keyboardType(.decimalPad)
                    .font(.metricSmall)
                    .foregroundStyle(Palette.text)
                Text("kg").font(.meta).foregroundStyle(Palette.textSecondary)
            }
            .padding(.horizontal, 14)
            .frame(minHeight: 52)
            .background(Palette.surface, in: .rect(cornerRadius: 12))
        }
    }
}

enum SettingsText {
    static func failure(_ error: any Error) -> String {
        if let e = error as? SettingsError {
            return switch e {
            case .onboardingRequired:
                String(localized: "Your plan's settings come with the onboarding. Answer it first.")
            case .invalid:
                String(localized: "The server could not take these settings. Check them and try again.")
            case .unavailable:
                String(localized: "The planner is not available right now. Try again later.")
            case .refused:
                String(localized: "It could not be sent. Try again later.")
            }
        }
        return String(localized: "Your plan's settings change online. Try again when you have a connection.")
    }
}
