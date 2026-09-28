import HefestoLogger
import HefestoStore
import SwiftUI

/// The session player: dark, high contrast, large targets, one hand. The
/// next set is logged from the bottom of the screen, where the thumb is.
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

    private var isDraft: Bool { logger?.session.status == "draft" }

    var body: some View {
        NavigationStack {
            Group {
                if let logger {
                    content(logger)
                } else {
                    ProgressView()
                }
            }
            .themedScreen()
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Close") { dismiss() }
                }
                ToolbarItem(placement: .principal) {
                    Text(verbatim: logger.map { title($0.session) } ?? "").font(.navTitle)
                }
                if isDraft {
                    ToolbarItem(placement: .confirmationAction) {
                        Button("Finish") { finishing = true }.fontWeight(.semibold)
                    }
                }
            }
            // The rest stays in view however many sets are above or below it.
            .safeAreaInset(edge: .top, spacing: 0) {
                if let logger, let rest = logger.rest {
                    RestCard(rest: rest) { skipRest(logger) }
                        .padding(.horizontal, 20)
                        .padding(.top, 16)
                        .padding(.bottom, 18)
                        .background(Palette.surface, in: .rect(cornerRadius: 20))
                        .padding(.horizontal, 16)
                        .padding(.vertical, 8)
                        .background(Palette.background)
                }
            }
            .safeAreaInset(edge: .bottom) {
                if let logger, isDraft, !logger.session.isRestDay { bottomBar(logger) }
            }
        }
        .preferredColorScheme(.dark)
        .sensoryFeedback(.success, trigger: loggedCount)
        .task {
            do {
                let logger = try model.makeLogger(sessionId: sessionId)
                self.logger = logger
                exercises = try model.db.exercisesById()
                if let demo = model.demo, demo.screen == .composer, let last = logger.tree.blocks.last {
                    composing = Composing(blockId: last.id)
                }
                if model.demo?.screen == .finish { finishing = true }
            } catch {
                self.error = error.localizedDescription
            }
        }
        .sheet(item: $composing) { c in
            SetComposer(db: model.db, defaultRest: logger?.defaultRestSeconds ?? 120) { drafts, rest, reserve in
                log(drafts, rest: rest, reserve: reserve, in: c.blockId)
            }
        }
        .sheet(isPresented: $finishing) {
            FinishSheet { fatigue in finish(fatigue) }
                .presentationDetents([.medium, .large])
        }
        .alert("Something went wrong", isPresented: .constant(error != nil)) {
            Button("OK") { error = nil }
        } message: {
            Text(verbatim: error ?? "")
        }
    }

    @ViewBuilder
    private func content(_ logger: LoggerModel) -> some View {
        // The set the athlete logged last: the list keeps it in view.
        let latest = logger.tree.blocks.flatMap(\.sets)
            .max { ($0.entry.completedAt ?? .distantPast) < ($1.entry.completedAt ?? .distantPast) }?.id
        ScrollViewReader { proxy in
            sets(logger)
                .task(id: latest) {
                    guard let latest else { return }
                    try? await Task.sleep(for: .milliseconds(100))
                    withAnimation(.snappy) { proxy.scrollTo(latest, anchor: .bottom) }
                }
        }
    }

    private func sets(_ logger: LoggerModel) -> some View {
        List {
            ForEach(Array(logger.tree.blocks.enumerated()), id: \.element.id) { i, block in
                Section {
                    ForEach(Array(block.sets.enumerated()), id: \.element.id) { n, set in
                        SetRow(number: n + 1, set: set, exercises: exercises) {
                            repeatSet(set, in: block.id, logger)
                        }
                        .id(set.id)
                        .themedRow()
                        .swipeActions {
                            Button("Delete", systemImage: "trash", role: .destructive) {
                                attempt { try logger.deleteSet(set.id) }
                            }
                        }
                    }
                    // The bottom bar logs into the last block; an earlier
                    // one keeps its own, smaller way in.
                    if isDraft, block.id != logger.tree.blocks.last?.id {
                        Button {
                            composing = Composing(blockId: block.id)
                        } label: {
                            Label("Log a set in this block", systemImage: "plus")
                                .font(.bodyMedium)
                                .foregroundStyle(Palette.ember)
                                .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
                        }
                        .themedRow()
                    }
                    if block.sets.isEmpty {
                        Text("No sets in this block yet.")
                            .font(.detailText)
                            .foregroundStyle(Palette.textSecondary)
                            .frame(minHeight: 44)
                            .themedRow()
                    }
                } header: {
                    CapsLabel("Block \(i + 1)")
                }
            }
        }
        .listStyle(.insetGrouped)
    }

    private func bottomBar(_ logger: LoggerModel) -> some View {
        HStack(spacing: 10) {
            Button {
                attempt { try logger.addBlock() }
            } label: {
                Image(systemName: "square.stack.3d.up")
                    .font(.system(size: 22, weight: .medium))
                    .foregroundStyle(Palette.text)
                    .frame(width: 64, height: 64)
                    .background(Palette.surfaceRaised, in: .rect(cornerRadius: 16))
            }
            .buttonStyle(.plain)
            .accessibilityLabel("Add a block")

            Button {
                if let last = logger.tree.blocks.last { composing = Composing(blockId: last.id) }
            } label: {
                Label("Log a set", systemImage: "plus")
            }
            .buttonStyle(PrimaryButtonStyle())
            .disabled(logger.tree.blocks.isEmpty)
        }
        .padding(.horizontal, 16)
        .padding(.top, 12)
        .padding(.bottom, 8)
        .background(Palette.background.ignoresSafeArea(edges: .bottom))
        .overlay(alignment: .top) { Divider().overlay(Palette.surfaceRaised) }
    }

    private func log(_ drafts: [ElementDraft], rest: Int, reserve: SetReserve, in blockId: String) {
        guard let logger else { return }
        attempt {
            try logger.logCombo(
                in: blockId, elements: drafts, restPlannedSeconds: rest, rir: reserve.rir, sirS: reserve.sirS)
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

/// One logged set: its number, its elements, the numbers large. A combo is
/// one set whose elements are joined by a line, in order.
struct SetRow: View {
    let number: Int
    let set: SetWithElements
    let exercises: [String: Exercise]
    /// Nil where the set is only shown, as in history.
    let onRepeat: (() -> Void)?

    var body: some View {
        HStack(alignment: .center, spacing: 12) {
            HStack(alignment: .center, spacing: 12) {
                Text("\(number)")
                    .font(Typeface.condensed(16, .semibold, relativeTo: .subheadline))
                    .monospacedDigit()
                    .foregroundStyle(Palette.textSecondary)
                    .frame(width: 28, height: 28)
                    .background(Palette.surfaceRaised, in: .circle)
                    .accessibilityLabel(Text("Set \(number)"))
                if set.elements.count > 1 { combo } else if let e = set.elements.first { single(e) }
            }
            .accessibilityElement(children: .combine)
            if let onRepeat {
                Button(action: onRepeat) { Image(systemName: "arrow.clockwise") }
                    .buttonStyle(RoundIconButtonStyle())
                    .accessibilityLabel("Repeat this set")
            }
        }
        .padding(.vertical, 6)
    }

    private func single(_ e: SetElement) -> some View {
        HStack(alignment: .center, spacing: 8) {
            VStack(alignment: .leading, spacing: 2) {
                Text(verbatim: name(e)).font(.rowTitle).foregroundStyle(Palette.text)
                if let extra = ElementFormat.qualifiers(e) {
                    Text(verbatim: extra).font(.fine).foregroundStyle(Palette.textSecondary)
                }
                restLine
            }
            Spacer(minLength: 4)
            metric(e, font: .metric)
        }
    }

    private var combo: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text("Combo")
                .font(.capsSmall)
                .tracking(1.2)
                .textCase(.uppercase)
                .foregroundStyle(Palette.ember)
            VStack(alignment: .leading, spacing: 6) {
                ForEach(set.elements) { e in
                    HStack(alignment: .center, spacing: 10) {
                        Circle().fill(Palette.ember).frame(width: 8, height: 8)
                        VStack(alignment: .leading, spacing: 0) {
                            Text(verbatim: name(e)).font(Typeface.text(16, .semibold, relativeTo: .headline))
                                .foregroundStyle(Palette.text)
                            if let extra = ElementFormat.qualifiers(e) {
                                Text(verbatim: extra).font(.fine).foregroundStyle(Palette.textSecondary)
                            }
                        }
                        Spacer(minLength: 4)
                        metric(e, font: .metricSmall)
                    }
                    .frame(minHeight: 34)
                }
            }
            // The line that joins the parts, from the first dot to the last.
            .background(alignment: .leading) {
                Rectangle().fill(Palette.ember).frame(width: 1.5).padding(.vertical, 17).padding(.leading, 3.25)
            }
            restLine
        }
    }

    @ViewBuilder
    private var restLine: some View {
        if let rest = set.entry.restAfterActualS {
            Text("Rested \(Duration.seconds(rest).formatted(.time(pattern: .minuteSecond)))")
                .font(.fine)
                .monospacedDigit()
                .foregroundStyle(Palette.textSecondary)
        }
    }

    private func metric(_ e: SetElement, font: Font) -> some View {
        HStack(alignment: .firstTextBaseline, spacing: 3) {
            if let m = ElementFormat.metric(e) {
                Text(verbatim: m.value).font(font).monospacedDigit().foregroundStyle(Palette.text)
                Text(verbatim: m.unit).font(.meta).foregroundStyle(Palette.textSecondary)
            }
        }
    }

    private func name(_ e: SetElement) -> String {
        exercises[e.exerciseId]?.name ?? String(localized: "Exercise")
    }
}

enum ElementFormat {
    /// The number to read at a glance and its unit: "8" "reps", "12" "s".
    static func metric(_ e: SetElement) -> (value: String, unit: String)? {
        switch e.measure {
        case "reps": e.reps.map { (value: "\($0)", unit: String(localized: "reps")) }
        case "hold_seconds": e.holdSeconds.map { (value: "\(Int($0))", unit: "s") }
        case "distance_m": e.distanceM.map { (value: "\(Int($0))", unit: "m") }
        default: nil
        }
    }

    /// Load, help and failure, when there are any: "+5 kg · Partner".
    static func qualifiers(_ e: SetElement) -> String? {
        var parts: [String] = []
        if e.loadKg > 0 { parts.append(String(localized: "+\(e.loadKg.formatted(.number.precision(.fractionLength(0...2)))) kg")) }
        if let a = e.assistance { parts.append(AssistanceKind.label(a.type)) }
        if e.failed { parts.append(String(localized: "to failure")) }
        return parts.isEmpty ? nil : parts.joined(separator: " · ")
    }

    static func summary(_ e: SetElement) -> String {
        var parts: [String] = []
        switch e.measure {
        case "reps": if let r = e.reps { parts.append(String(localized: "\(r) reps")) }
        case "hold_seconds": if let s = e.holdSeconds { parts.append(String(localized: "\(Int(s)) s hold")) }
        case "distance_m": if let d = e.distanceM { parts.append(String(localized: "\(Int(d)) m")) }
        default: break
        }
        if let q = qualifiers(e) { parts.append(q) }
        return parts.joined(separator: " · ")
    }
}

/// The rest since the last set, computed from the wall clock each second:
/// the time left in large numbers, and how much of the planned rest is gone.
struct RestCard: View {
    let rest: RestTimer
    let onSkip: () -> Void

    var body: some View {
        TimelineView(.periodic(from: rest.startedAt, by: 1)) { context in
            let done = rest.isDone(at: context.date)
            VStack(alignment: .leading, spacing: 10) {
                HStack(alignment: .bottom) {
                    VStack(alignment: .leading, spacing: 0) {
                        CapsLabel(done ? "Rest done" : "Rest", isHeader: false)
                        Text(clock(context.date))
                            .font(.timerDigits)
                            .monospacedDigit()
                            .foregroundStyle(done ? Palette.ember : Palette.text)
                            .contentTransition(.numericText())
                            .lineLimit(1)
                            .minimumScaleFactor(0.6)
                    }
                    Spacer(minLength: 8)
                    Button(action: onSkip) {
                        Text("Skip")
                            .font(.bodyMedium)
                            .foregroundStyle(Palette.text)
                            .padding(.horizontal, 16)
                            .frame(minHeight: 44)
                            .background(Palette.surfaceRaised, in: .rect(cornerRadius: 12))
                    }
                    .buttonStyle(.plain)
                    .padding(.bottom, 10)
                }
                if let planned = rest.plannedSeconds {
                    Capsule()
                        .fill(Palette.hairline)
                        .frame(height: 6)
                        .overlay(alignment: .leading) {
                            Capsule().fill(Palette.ember)
                                .scaleEffect(x: fraction(at: context.date, of: planned), y: 1, anchor: .leading)
                        }
                        .accessibilityHidden(true)
                    Text("of \(Duration.seconds(planned).formatted(.time(pattern: .minuteSecond)))")
                        .font(.fine)
                        .monospacedDigit()
                        .foregroundStyle(Palette.textSecondary)
                        .frame(maxWidth: .infinity, alignment: .trailing)
                }
            }
            .sensoryFeedback(.impact(weight: .heavy), trigger: done) { old, new in !old && new }
            .accessibilityElement(children: .combine)
        }
    }

    private func clock(_ now: Date) -> String {
        let seconds = rest.remaining(at: now) ?? rest.elapsed(at: now)
        return Duration.seconds(seconds).formatted(.time(pattern: .minuteSecond))
    }

    private func fraction(at now: Date, of planned: Int) -> CGFloat {
        min(1, CGFloat(rest.elapsed(at: now)) / CGFloat(max(1, planned)))
    }
}

/// Ends a session, with how hard it felt if the athlete wants to say.
struct FinishSheet: View {
    let onFinish: (Int?) -> Void
    @State private var fatigue: Int?

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    Text("How hard did it feel?").font(.displayTitle).foregroundStyle(Palette.text)
                    LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 8), count: 5), spacing: 8) {
                        ForEach(1...10, id: \.self) { n in
                            Button {
                                fatigue = fatigue == n ? nil : n
                            } label: {
                                Text("\(n)")
                                    .font(.metricSmall)
                                    .monospacedDigit()
                                    .foregroundStyle(fatigue == n ? Palette.emberInk : Palette.text)
                                    .frame(maxWidth: .infinity, minHeight: 52)
                                    .background(fatigue == n ? Palette.ember : Palette.surface, in: .rect(cornerRadius: 12))
                            }
                            .buttonStyle(.plain)
                            .accessibilityAddTraits(fatigue == n ? .isSelected : [])
                        }
                    }
                    Text("Optional. 1 is easy, 10 is everything you had.")
                        .font(.detailText)
                        .foregroundStyle(Palette.textSecondary)
                    Button {
                        onFinish(fatigue)
                    } label: {
                        Label("Finish session", systemImage: "checkmark")
                    }
                    .buttonStyle(PrimaryButtonStyle())
                }
                .padding(20)
            }
            .themedScreen()
        }
    }
}
