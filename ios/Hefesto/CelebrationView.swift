import HefestoAPI
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

    private var result: Components.Schemas.CompletionResult { celebration.result }
    private var xp: Int { result.xpAwarded.reduce(0) { $0 + $1.amount } }
    private var unlocked: Bool { !result.unlocked.isEmpty }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 24) {
                    ZStack {
                        if unlocked && !reduceMotion {
                            ForEach(0..<8, id: \.self) { i in
                                Image(systemName: "sparkle")
                                    .font(.title3)
                                    .foregroundStyle(.yellow)
                                    .offset(y: shown ? -86 : -20)
                                    .rotationEffect(.degrees(Double(i) * 45))
                                    .opacity(shown ? 0 : 1)
                            }
                        }
                        Image(systemName: unlocked ? "star.circle.fill" : "checkmark.seal.fill")
                            .font(.system(size: 88))
                            .foregroundStyle(unlocked ? Color.yellow : Color.green)
                            .shadow(color: unlocked ? .yellow.opacity(0.7) : .clear, radius: shown ? 24 : 0)
                            .scaleEffect(shown || reduceMotion ? 1 : 0.4)
                    }
                    .frame(height: 180)
                    .accessibilityHidden(true)

                    VStack(spacing: 6) {
                        Text(unlocked ? LocalizedStringKey("New skill unlocked") : "Session saved")
                            .font(.title.bold())
                        if xp > 0 { Text("+\(xp) XP").font(.headline).foregroundStyle(.secondary) }
                    }

                    if unlocked {
                        VStack(spacing: 12) {
                            ForEach(result.unlocked, id: \.levelId) { u in
                                VStack(spacing: 2) {
                                    Text(verbatim: u.levelName).font(.title3.weight(.semibold))
                                    Text(verbatim: u.skillName).foregroundStyle(.secondary)
                                }
                                .frame(maxWidth: .infinity)
                                .padding()
                                .background(.yellow.opacity(0.12), in: .rect(cornerRadius: 16))
                                .accessibilityElement(children: .combine)
                            }
                        }
                    }
                    if !result.newlyAvailable.isEmpty {
                        Text("\(result.newlyAvailable.count) new levels are open to work towards.")
                            .multilineTextAlignment(.center)
                            .foregroundStyle(.secondary)
                    }
                    if result.streak.currentDays > 1 {
                        Label("\(result.streak.currentDays)-day streak", systemImage: "flame.fill")
                            .foregroundStyle(.orange)
                    }

                    if unlocked {
                        Button {
                            model.tab = .map
                            dismiss()
                        } label: {
                            Text("See it on the map").font(.headline).frame(maxWidth: .infinity, minHeight: 50)
                        }
                        .buttonStyle(.borderedProminent)
                    }
                }
                .padding()
            }
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
