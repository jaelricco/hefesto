import HefestoLogger
import HefestoStore
import SwiftUI

/// The session player: dark, high contrast, large targets, one hand.
struct LoggerView: View {
    let sessionId: String
    @Environment(AppModel.self) private var model
    @Environment(\.dismiss) private var dismiss
    @State private var logger: LoggerModel?
    @State private var exercises: [String: Exercise] = [:]
    @State private var composing: Composing?
    @State private var finishing = false
    @State private var error: String?
    @State private var loggedCount = 0

    var body: some View {
        NavigationStack {
            Group {
                if let logger {
                    content(logger)
                } else {
                    ProgressView()
                }
            }
            .navigationTitle(logger.map { title($0.session) } ?? "")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Close") { dismiss() }
                }
                if logger?.session.status == "draft" {
                    ToolbarItem(placement: .confirmationAction) {
                        Button("Finish") { finishing = true }.bold()
                    }
                }
            }
        }
        .preferredColorScheme(.dark)
        .sensoryFeedback(.success, trigger: loggedCount)
        .task {
            do {
                logger = try LoggerModel(db: model.db, sessionId: sessionId)
                exercises = try model.db.exercisesById()
            } catch {
                self.error = error.localizedDescription
            }
        }
        .sheet(item: $composing) { c in
            SetComposer(db: model.db, defaultRest: logger?.defaultRestSeconds ?? 120) { drafts, rest in
                log(drafts, rest: rest, in: c.blockId)
            }
        }
        .sheet(isPresented: $finishing) {
            FinishSheet { fatigue in finish(fatigue) }
                .presentationDetents([.medium])
        }
        .alert("Something went wrong", isPresented: .constant(error != nil)) {
            Button("OK") { error = nil }
        } message: {
            Text(verbatim: error ?? "")
        }
    }

    @ViewBuilder
    private func content(_ logger: LoggerModel) -> some View {
        List {
            if let rest = logger.rest {
                Section { RestBanner(rest: rest) { skipRest(logger) } }
            }
            ForEach(Array(logger.tree.blocks.enumerated()), id: \.element.id) { i, block in
                Section {
                    ForEach(Array(block.sets.enumerated()), id: \.element.id) { n, set in
                        SetRow(number: n + 1, set: set, exercises: exercises) {
                            repeatSet(set, in: block.id, logger)
                        }
                        .swipeActions {
                            Button("Delete", systemImage: "trash", role: .destructive) {
                                attempt { try logger.deleteSet(set.id) }
                            }
                        }
                    }
                    if logger.session.status == "draft" {
                        Button {
                            composing = Composing(blockId: block.id)
                        } label: {
                            Label("Log a set", systemImage: "plus.circle.fill")
                                .font(.title3.weight(.semibold))
                                .frame(maxWidth: .infinity, minHeight: 56)
                        }
                        .buttonStyle(.borderedProminent)
                        .listRowInsets(EdgeInsets(top: 8, leading: 16, bottom: 8, trailing: 16))
                    }
                } header: {
                    Text("Block \(i + 1)")
                }
            }
            if logger.session.status == "draft", !logger.session.isRestDay {
                Section {
                    Button("Add a block", systemImage: "square.stack.3d.up") {
                        attempt { try logger.addBlock() }
                    }
                    .frame(minHeight: 44)
                }
            }
        }
    }

    private func log(_ drafts: [ElementDraft], rest: Int, in blockId: String) {
        guard let logger else { return }
        attempt {
            try logger.logCombo(in: blockId, elements: drafts, restPlannedSeconds: rest)
            loggedCount += 1
            if let end = logger.rest?.endsAt { RestNotifier.schedule(at: end) }
        }
    }

    private func repeatSet(_ set: SetWithElements, in blockId: String, _ logger: LoggerModel) {
        guard let first = set.elements.first else { return }
        attempt {
            if try logger.repeatLastSet(of: first.exerciseId, in: blockId) != nil {
                loggedCount += 1
                if let end = logger.rest?.endsAt { RestNotifier.schedule(at: end) }
            }
        }
    }

    private func skipRest(_ logger: LoggerModel) {
        logger.skipRest()
        RestNotifier.cancel()
    }

    private func finish(_ fatigue: Int?) {
        guard let logger else { return }
        attempt {
            try logger.complete(perceivedFatigue: fatigue)
            RestNotifier.cancel()
            finishing = false
            dismiss()
            Task { await model.syncNow() }
        }
    }

    private func attempt(_ body: () throws -> Void) {
        do { try body() } catch { self.error = error.localizedDescription }
    }

    private func title(_ s: Session) -> String {
        s.title.isEmpty ? String(localized: "Session") : s.title
    }
}

struct Composing: Identifiable {
    let blockId: String
    var id: String { blockId }
}

/// One logged set: its elements in order, a combo reading as one line each.
struct SetRow: View {
    let number: Int
    let set: SetWithElements
    let exercises: [String: Exercise]
    let onRepeat: () -> Void

    var body: some View {
        HStack(alignment: .firstTextBaseline, spacing: 12) {
            Text("\(number)")
                .font(.title2.monospacedDigit().bold())
                .frame(minWidth: 28)
                .accessibilityLabel(Text("Set \(number)"))
            VStack(alignment: .leading, spacing: 4) {
                ForEach(set.elements) { e in
                    VStack(alignment: .leading, spacing: 0) {
                        Text(verbatim: exercises[e.exerciseId]?.name ?? String(localized: "Exercise"))
                            .font(.headline)
                        Text(verbatim: ElementFormat.summary(e)).font(.body.monospacedDigit())
                    }
                }
                if let rest = set.entry.restAfterActualS {
                    Text("Rested \(Duration.seconds(rest).formatted(.time(pattern: .minuteSecond)))")
                        .font(.caption).foregroundStyle(.secondary)
                }
            }
            Spacer()
            Button(action: onRepeat) {
                Image(systemName: "arrow.clockwise.circle.fill").font(.title)
            }
            .buttonStyle(.plain)
            .frame(minWidth: 56, minHeight: 56)
            .accessibilityLabel("Repeat this set")
        }
        .padding(.vertical, 4)
    }
}

enum ElementFormat {
    static func summary(_ e: SetElement) -> String {
        var parts: [String] = []
        switch e.measure {
        case "reps": if let r = e.reps { parts.append(String(localized: "\(r) reps")) }
        case "hold_seconds": if let s = e.holdSeconds { parts.append(String(localized: "\(Int(s)) s hold")) }
        case "distance_m": if let d = e.distanceM { parts.append(String(localized: "\(Int(d)) m")) }
        default: break
        }
        if e.loadKg > 0 { parts.append(String(localized: "+\(e.loadKg.formatted(.number.precision(.fractionLength(0...2)))) kg")) }
        if let a = e.assistance { parts.append(AssistanceKind.label(a.type)) }
        if e.failed { parts.append(String(localized: "to failure")) }
        return parts.joined(separator: " · ")
    }
}

/// The rest since the last set, computed from the wall clock each second.
struct RestBanner: View {
    let rest: RestTimer
    let onSkip: () -> Void

    var body: some View {
        TimelineView(.periodic(from: rest.startedAt, by: 1)) { context in
            let done = rest.isDone(at: context.date)
            HStack {
                VStack(alignment: .leading) {
                    Text(done ? LocalizedStringKey("Rest done") : "Rest").font(.headline)
                    Text(clock(context.date))
                        .font(.system(size: 48, weight: .bold, design: .rounded).monospacedDigit())
                        .foregroundStyle(done ? .green : .primary)
                        .contentTransition(.numericText())
                }
                Spacer()
                Button("Skip", action: onSkip)
                    .buttonStyle(.bordered)
                    .frame(minHeight: 56)
            }
            .sensoryFeedback(.impact(weight: .heavy), trigger: done) { old, new in !old && new }
            .accessibilityElement(children: .combine)
        }
    }

    private func clock(_ now: Date) -> String {
        let seconds = rest.remaining(at: now) ?? rest.elapsed(at: now)
        return Duration.seconds(seconds).formatted(.time(pattern: .minuteSecond))
    }
}

/// Ends a session, with how hard it felt if the athlete wants to say.
struct FinishSheet: View {
    let onFinish: (Int?) -> Void
    @State private var fatigue: Int?

    var body: some View {
        NavigationStack {
            VStack(spacing: 20) {
                Text("How hard did it feel?").font(.title3.weight(.semibold))
                LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 8), count: 5), spacing: 8) {
                    ForEach(1...10, id: \.self) { n in
                        Button {
                            fatigue = fatigue == n ? nil : n
                        } label: {
                            Text("\(n)").font(.title3.monospacedDigit().bold()).frame(maxWidth: .infinity, minHeight: 48)
                        }
                        .buttonStyle(.bordered)
                        .tint(fatigue == n ? .accentColor : .secondary)
                        .accessibilityAddTraits(fatigue == n ? .isSelected : [])
                    }
                }
                Text("Optional. 1 is easy, 10 is everything you had.").font(.footnote).foregroundStyle(.secondary)
                Button {
                    onFinish(fatigue)
                } label: {
                    Text("Finish session").font(.title3.weight(.semibold)).frame(maxWidth: .infinity, minHeight: 56)
                }
                .buttonStyle(.borderedProminent)
            }
            .padding()
        }
    }
}
