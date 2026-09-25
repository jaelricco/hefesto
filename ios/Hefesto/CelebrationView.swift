import HefestoAPI
import SwiftUI

/// What a completed session earned, once the server has evaluated it. It
/// celebrates what was achieved and never mentions what was missed (ADR 0003).
struct CelebrationView: View {
    let celebration: Celebration
    @Environment(\.dismiss) private var dismiss

    private var result: Components.Schemas.CompletionResult { celebration.result }
    private var xp: Int { result.xpAwarded.reduce(0) { $0 + $1.amount } }

    var body: some View {
        NavigationStack {
            List {
                Section {
                    VStack(spacing: 8) {
                        Image(systemName: result.unlocked.isEmpty ? "checkmark.seal.fill" : "star.circle.fill")
                            .font(.system(size: 64))
                            .foregroundStyle(result.unlocked.isEmpty ? .green : .yellow)
                            .accessibilityHidden(true)
                        Text(result.unlocked.isEmpty ? LocalizedStringKey("Session saved") : "New skill unlocked")
                            .font(.title2.bold())
                        if xp > 0 { Text("+\(xp) XP").font(.headline).foregroundStyle(.secondary) }
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical)
                }
                .sensoryFeedback(.success, trigger: celebration.id)

                if !result.unlocked.isEmpty {
                    Section("Unlocked") {
                        ForEach(result.unlocked, id: \.levelId) { u in
                            VStack(alignment: .leading) {
                                Text(verbatim: u.levelName).font(.headline)
                                Text(verbatim: u.skillName).font(.subheadline).foregroundStyle(.secondary)
                            }
                        }
                    }
                }
                if !result.newlyAvailable.isEmpty {
                    Section {
                        Text("\(result.newlyAvailable.count) new levels are open to work towards.")
                    }
                }
            }
            .toolbar {
                ToolbarItem(placement: .confirmationAction) { Button("Done") { dismiss() } }
            }
        }
    }
}
