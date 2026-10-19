import HefestoPlan
import HefestoSync
import SwiftUI

// The questions about pain around a planned session (ADR 0030): before it in
// the check-in, once it is done in the finish sheet, and the morning after on
// Today. Every answer is optional; nothing here counts against the athlete.

/// The questions of one moment, for each region the plan names.
struct PainQuestionView: View {
    let moment: PainMoment
    let regions: [PlanRegion]
    @Binding var answers: [String: PainAnswer]

    var body: some View {
        VStack(alignment: .leading, spacing: 20) {
            ForEach(regions) { region in
                VStack(alignment: .leading, spacing: 14) {
                    CapsLabel(verbatim: BodyText.title(region.name), isHeader: false)
                    switch moment {
                    case .beforeSession:
                        scale(String(localized: "Pain right now"), region, \.now)
                    case .afterSession:
                        scale(String(localized: "Pain during the session"), region, \.during)
                        scale(String(localized: "Pain right now"), region, \.now)
                    case .nextMorning:
                        scale(String(localized: "Pain this morning"), region, \.now)
                        YesNoQuestion(
                            question: String(localized: "Did the pain after the session last longer than an hour?"),
                            answer: Binding(
                                get: { answers[region.id]?.lastedOver1h },
                                set: { answers[region.id, default: PainAnswer()].lastedOver1h = $0 }))
                    }
                }
            }
            Text("0 is no pain, 10 the worst imaginable. Optional: your plan adapts the load of the region to it.")
                .font(.detailText)
                .foregroundStyle(Palette.textSecondary)
                .fixedSize(horizontal: false, vertical: true)
        }
    }

    private func scale(_ question: String, _ region: PlanRegion, _ value: WritableKeyPath<PainAnswer, Int?>) -> some View {
        ChoiceQuestion(
            question: question, choices: Choice.scale,
            selection: Binding(
                get: { answers[region.id]?[keyPath: value].map(String.init) },
                set: { answers[region.id, default: PainAnswer()][keyPath: value] = $0.flatMap { Int($0) } }),
            columns: 6)
    }
}

/// The morning after a planned session: a card on Today that asks about the
/// regions of yesterday's session, until answered or put aside for the day.
struct MorningPainCard: View {
    let question: PainQuestion
    let onAnswer: () -> Void
    let onLater: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("How is it this morning?").font(.rowTitle).foregroundStyle(Palette.text)
            Text(verbatim: MorningPainCard.detail(question))
                .font(.detailText)
                .foregroundStyle(Palette.textSecondary)
                .fixedSize(horizontal: false, vertical: true)
            Button("Answer", action: onAnswer).buttonStyle(PrimaryButtonStyle())
            Button("Not today", action: onLater)
                .font(.bodyMedium)
                .foregroundStyle(Palette.textSecondary)
                .frame(maxWidth: .infinity, minHeight: 44)
        }
        .card()
    }

    static func detail(_ question: PainQuestion) -> String {
        let names = question.regions.map(\.name).formatted(.list(type: .and))
        return String(localized: "After yesterday's session: how does it feel \(names)? Your plan uses the answer to decide whether the load there can grow.")
    }
}

/// The morning question, answered.
struct MorningPainSheet: View {
    let question: PainQuestion
    let onSent: (BodyOutcome) -> Void

    @Environment(AppModel.self) private var model
    @Environment(\.dismiss) private var dismiss
    @State private var answers: [String: PainAnswer] = [:]
    @State private var sending = false
    @State private var failure: String?

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 24) {
                    PageTitle("This morning", detail: "About yesterday's session. Every answer is optional.")
                    PainQuestionView(moment: .nextMorning, regions: question.regions, answers: $answers)
                    if let failure {
                        Text(verbatim: failure).font(.detailText).foregroundStyle(Palette.text)
                    }
                    Button {
                        Task { await send() }
                    } label: {
                        if sending {
                            ProgressView().tint(Palette.emberInk)
                        } else {
                            Text("Send")
                        }
                    }
                    .buttonStyle(PrimaryButtonStyle())
                    .disabled(question.reports(answers, at: Date()).isEmpty || sending)
                }
                .padding(20)
            }
            .themedScreen()
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
            }
        }
    }

    private func send() async {
        sending = true
        failure = nil
        do {
            let outcome = try await model.body.reportPain(question.reports(answers, at: Date()))
            sending = false
            dismiss()
            onSent(outcome)
        } catch {
            sending = false
            failure = BodyText.failure(error)
        }
    }
}
