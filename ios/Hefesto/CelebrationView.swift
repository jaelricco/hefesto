import SwiftUI

/// What a completed session earned, once the server has evaluated it. It
/// celebrates what was achieved and never mentions what was missed
/// (ADR 0003). An unlock leads to the map, where its line lights up.
struct CelebrationView: View {
    let celebration: Celebration
    @Environment(AppModel.self) private var model
    @Environment(\.dismiss) private var dismiss
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @State private var shown = false

    private var unlocked: Bool { !celebration.unlocked.isEmpty }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 24) {
                    ZStack {
                        if unlocked && !reduceMotion {
                            ForEach(0..<8, id: \.self) { i in
                                Image(systemName: "sparkle")
                                    .font(.title3)
                                    .foregroundStyle(Palette.gold)
                                    .offset(y: shown ? -86 : -20)
                                    .rotationEffect(.degrees(Double(i) * 45))
                                    .opacity(shown ? 0 : 1)
                            }
                        }
                        Image(systemName: unlocked ? "star.circle.fill" : "checkmark.seal.fill")
                            .font(.system(size: 88))
                            .foregroundStyle(unlocked ? Palette.gold : Palette.ember)
                            .shadow(color: unlocked ? Palette.gold.opacity(0.6) : .clear, radius: shown ? 24 : 0)
                            .scaleEffect(shown || reduceMotion ? 1 : 0.4)
                    }
                    .frame(height: 180)
                    .accessibilityHidden(true)

                    VStack(spacing: 6) {
                        Text(unlocked ? LocalizedStringKey("New skill unlocked") : "Session saved")
                            .font(.displayTitle)
                            .foregroundStyle(Palette.text)
                            .multilineTextAlignment(.center)
                        if celebration.xp > 0 {
                            Text("+\(celebration.xp) XP")
                                .font(.metricSmall)
                                .monospacedDigit()
                                .foregroundStyle(Palette.gold)
                        }
                    }

                    if unlocked {
                        VStack(spacing: 10) {
                            ForEach(celebration.unlocked, id: \.levelId) { u in
                                VStack(spacing: 2) {
                                    Text(verbatim: u.levelName)
                                        .font(Typeface.text(20, .semibold, relativeTo: .title3))
                                        .foregroundStyle(Palette.text)
                                    Text(verbatim: u.skillName)
                                        .font(.capsLabel)
                                        .tracking(1.2)
                                        .textCase(.uppercase)
                                        .foregroundStyle(Palette.gold)
                                }
                                .frame(maxWidth: .infinity)
                                .padding(16)
                                .background(Palette.goldWash, in: .rect(cornerRadius: 16))
                                .accessibilityElement(children: .combine)
                            }
                        }
                    }
                    if celebration.newlyAvailable > 0 {
                        Text("\(celebration.newlyAvailable) new levels are open to work towards.")
                            .font(.detailText)
                            .multilineTextAlignment(.center)
                            .foregroundStyle(Palette.textSecondary)
                    }
                    if celebration.streakDays > 1 {
                        Label("\(celebration.streakDays)-day streak", systemImage: "flame.fill")
                            .font(Typeface.condensed(18, .semibold, relativeTo: .headline))
                            .foregroundStyle(Palette.ember)
                    }

                    if unlocked {
                        Button {
                            model.tab = .map
                            dismiss()
                        } label: {
                            Label("See it on the map", systemImage: "sparkles")
                        }
                        .buttonStyle(PrimaryButtonStyle())
                    }
                }
                .padding(20)
            }
            .themedScreen()
            .toolbar {
                ToolbarItem(placement: .confirmationAction) { Button("Done") { dismiss() } }
            }
        }
        .preferredColorScheme(.dark)
        .sensoryFeedback(.success, trigger: shown)
        .onAppear {
            withAnimation(reduceMotion ? nil : .spring(duration: 0.9, bounce: 0.4)) { shown = true }
        }
    }
}
