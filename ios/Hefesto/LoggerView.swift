import HefestoLogger
import HefestoPlan
import HefestoStore
import SwiftUI

/// The session player: dark, high contrast, large targets, one hand. The
/// next set is logged from the bottom of the screen, where the thumb is.
struct LoggerView: View {
    let sessionId: String
    @Environment(AppModel.self) private var model
    @Environment(\.dynamicTypeSize) private var typeSize
    @Environment(\.dismiss) private var dismiss
    @State private var logger: LoggerModel?
    @State private var exercises: [String: Exercise] = [:]
    @State private var bands: [String: Band] = [:]
    @State private var composing: Composing?
    @State private var finishing = false
    @State private var error: String?
    @State private var loggedCount = 0
    /// Pain or symptoms reported from the session (ADR 0028).
    @State private var bodySheet: BodySheet?
    @State private var bodyOutcome: BodyOutcome?
    @State private var askingFlags: RegionRoute?

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
                    // Bars cap their text; a long press shows it large.
                    Text(verbatim: logger.map { title($0.session) } ?? "").font(.navTitle)
                        .accessibilityShowsLargeContentViewer()
                }
                if isDraft {
                    ToolbarItem(placement: .topBarTrailing) {
                        Menu {
                            Button("Report pain", systemImage: "hand.raised") { bodySheet = .pain }
                            Button("Symptoms during training", systemImage: "heart.text.square") { bodySheet = .symptoms }
                        } label: {
                            Image(systemName: "cross.case")
                        }
                        .accessibilityLabel("Pain and symptoms")
                    }
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
                        .pinnedBar()
                }
            }
            .safeAreaInset(edge: .bottom) {
                if let logger, isDraft, !logger.session.isRestDay { bottomBar(logger).pinnedBar() }
            }
        }
        .preferredColorScheme(.dark)
        .sensoryFeedback(.success, trigger: loggedCount)
        .task {
            do {
                let logger = try model.makeLogger(sessionId: sessionId)
                self.logger = logger
                let exercises = try model.db.exercisesById()
                self.exercises = exercises
                if let demo = model.demo, [.composer, .reserve].contains(demo.screen),
                   let last = logger.tree.blocks.last {
                    composing = Composing(
                        blockId: last.id, exercise: demo.screen == .reserve ? exercises[demo.exerciseId] : nil)
                }
                if let demo = model.demo, demo.screen == .band, let last = logger.tree.blocks.last {
                    composing = Composing(
                        blockId: last.id, exercise: exercises[demo.bandExerciseId], band: demo.bandId)
                }
                if model.demo?.screen == .target, let next = nextPlanned(logger) {
                    composing = Composing(blockId: next.entry.blockId, planned: target(for: next, logger))
                }
                if model.demo?.screen == .finish || model.demo?.screen == .painafter { finishing = true }
            } catch {
                self.error = error.localizedDescription
            }
        }
        .task {
            // Every band, also one just added in the composer, to name it.
            do {
                for try await _ in model.db.observeBands() { bands = (try? model.db.bandsById()) ?? [:] }
            } catch {}
        }
        .sheet(item: $composing) { c in
            SetComposer(
                db: model.db, defaultRest: logger?.defaultRestSeconds ?? 120, exercise: c.exercise, band: c.band,
                planned: c.planned
            ) { drafts, rest, reserve in
                if let planned = c.planned {
                    perform(planned, drafts, reserve: reserve)
                } else {
                    log(drafts, rest: rest, reserve: reserve, in: c.blockId)
                }
            }
        }
        .sheet(isPresented: $finishing) {
            // The pain questions of a planned session (ADR 0030) need the
            // whole screen, as does any sheet at the accessibility sizes.
            let question = model.plan.afterQuestion(logSessionId: sessionId)
            FinishSheet(question: question) { fatigue, pain in finish(fatigue, pain: pain, question: question) }
                .presentationDetents(typeSize.isAccessibilitySize || question != nil ? [.large] : [.medium, .large])
        }
        .sheet(item: $bodySheet) { sheet in
            switch sheet {
            case .pain: PainReportSheet(sessionId: sessionId, initialTimepoint: "during") { reported($0) }
            case .symptoms: SymptomsSheet { reported($0) }
            }
        }
        .sheet(item: $askingFlags) { route in
            RedFlagSheet(regionId: route.id) { bodyOutcome = $0 }
        }
        .alert(
            bodyOutcome?.waiting == true ? "Kept" : "Sent", isPresented: Binding(get: { bodyOutcome != nil }, set: { if !$0 { bodyOutcome = nil } }),
            presenting: bodyOutcome
        ) { _ in
            Button("OK") { bodyOutcome = nil }
        } message: { outcome in
            Text(verbatim: BodyText.summary(outcome))
        }
        .alert("Something went wrong", isPresented: .constant(error != nil)) {
            Button("OK") { error = nil }
        } message: {
            Text(verbatim: error ?? "")
        }
    }

    /// What a report changed. The planner may ask a region's warning signs;
    /// the next sheet opens once this one is gone.
    private func reported(_ outcome: BodyOutcome) {
        guard let region = outcome.askRedFlags.first else {
            bodyOutcome = outcome
            return
        }
        Task {
            try? await Task.sleep(for: .milliseconds(600))
            askingFlags = RegionRoute(id: region)
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
                        Group {
                            if isOpenPlanned(set) {
                                PlannedSetRow(
                                    number: n + 1, set: set, exercises: exercises,
                                    reserveTarget: reserveTarget(set.entry), calibration: isCalibration(set, logger),
                                    band: needsBand(set),
                                    onPerform: isDraft
                                        ? { composing = Composing(blockId: block.id, planned: target(for: set, logger)) }
                                        : nil)
                            } else {
                                SetRow(number: n + 1, set: set, exercises: exercises, bands: bands) {
                                    repeatSet(set, in: block.id, logger)
                                }
                            }
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

    /// While the plan has sets left, the next of them is the main action and
    /// a set of the athlete's own is the small one beside it.
    private func bottomBar(_ logger: LoggerModel) -> some View {
        HStack(spacing: 10) {
            if let next = nextPlanned(logger) {
                Button {
                    if let last = logger.tree.blocks.last { composing = Composing(blockId: last.id) }
                } label: {
                    Image(systemName: "plus")
                        .font(.system(size: 22, weight: .medium))
                        .foregroundStyle(Palette.text)
                        .frame(width: 64, height: 64)
                        .background(Palette.surfaceRaised, in: .rect(cornerRadius: 16))
                }
                .buttonStyle(.plain)
                .accessibilityLabel("Log a set of your own")

                Button {
                    composing = Composing(blockId: next.entry.blockId, planned: target(for: next, logger))
                } label: {
                    Label("Next set", systemImage: "play.fill")
                }
                .buttonStyle(PrimaryButtonStyle())
            } else {
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
            AccessibilityNotification.Announcement(String(localized: "Set logged")).post()
            if let end = logger.rest?.endsAt { RestNotifier.schedule(at: end) }
        }
    }

    /// Performs a planned set with what the athlete did (ADR 0022).
    private func perform(_ planned: PlannedTarget, _ drafts: [ElementDraft], reserve: SetReserve) {
        guard let logger else { return }
        attempt {
            try logger.perform(plannedSet: planned.setId, elements: drafts, rir: reserve.rir, sirS: reserve.sirS)
            loggedCount += 1
            AccessibilityNotification.Announcement(String(localized: "Set logged")).post()
            if let end = logger.rest?.endsAt { RestNotifier.schedule(at: end) }
        }
    }

    private func isOpenPlanned(_ set: SetWithElements) -> Bool {
        set.entry.isPlanned && set.entry.completedAt == nil
    }

    /// The plan asks for a band; the log names the band itself (ADR 0016),
    /// so the composer asks for it (ADR 0024).
    private func needsBand(_ set: SetWithElements) -> Bool {
        item(of: set)?.assist == "band"
    }

    /// The next planned set, in the order of the blocks.
    private func nextPlanned(_ logger: LoggerModel) -> SetWithElements? {
        logger.openPlannedSets.first
    }

    /// The plan's item for a planned set, if this week's plan holds it.
    private func item(of set: SetWithElements) -> PlanItem? {
        set.entry.plannedItemId.flatMap { model.plan.week?.item(id: $0) }
    }

    /// Only the first working set of a calibration item calibrates.
    private func isCalibration(_ set: SetWithElements, _ logger: LoggerModel) -> Bool {
        guard let itemId = set.entry.plannedItemId, set.entry.kind == "working",
              item(of: set)?.calibration == true else { return false }
        let first = logger.tree.blocks.flatMap(\.sets).first {
            $0.entry.plannedItemId == itemId && $0.entry.kind == "working"
        }
        return first?.id == set.id
    }

    /// What the plan wants left at the end of the set.
    private func reserveTarget(_ entry: SetEntry) -> String? {
        if let seconds = entry.sirS { return String(localized: "\(seconds) s in reserve") }
        if let reps = entry.rir { return String(localized: "\(reps) reps in reserve") }
        return nil
    }

    private func target(for set: SetWithElements, _ logger: LoggerModel) -> PlannedTarget {
        let band = needsBand(set)
        return PlannedTarget(
            setId: set.id,
            drafts: set.elements.map { element -> Draft in
                var draft = Draft(target: element, exercise: exercises[element.exerciseId])
                // The plan's band is not in the draft: the athlete names it.
                if band && draft.assistance == "none" { draft.assistance = "band" }
                return draft
            },
            rest: set.entry.restAfterPlannedS, reserveTarget: reserveTarget(set.entry),
            calibration: isCalibration(set, logger), band: band)
    }

    private func repeatSet(_ set: SetWithElements, in blockId: String, _ logger: LoggerModel) {
        guard let first = set.elements.first else { return }
        attempt {
            if try logger.repeatLastSet(of: first.exerciseId, in: blockId) != nil {
                loggedCount += 1
                AccessibilityNotification.Announcement(String(localized: "Set logged")).post()
                if let end = logger.rest?.endsAt { RestNotifier.schedule(at: end) }
            }
        }
    }

    private func skipRest(_ logger: LoggerModel) {
        logger.skipRest()
        RestNotifier.cancel()
    }

    /// Completes the session. The pain answers go once the completion is
    /// synced; warning signs the planner asks for keep the log open.
    private func finish(_ fatigue: Int?, pain: [String: PainAnswer] = [:], question: PainQuestion? = nil) {
        guard let logger else { return }
        attempt {
            try logger.complete(perceivedFatigue: fatigue)
            RestNotifier.cancel()
            finishing = false
            let reports = question?.reports(pain, at: Date()) ?? []
            guard !reports.isEmpty else {
                dismiss()
                Task { await model.syncNow() }
                return
            }
            Task {
                await model.syncNow()
                do {
                    let outcome = try await model.body.reportPain(reports)
                    if !outcome.askRedFlags.isEmpty || outcome.trainingStopped {
                        reported(outcome)
                    } else {
                        dismiss()
                    }
                } catch {
                    // The session is done; the log stays open to say why the
                    // answers were refused.
                    self.error = BodyText.failure(error)
                }
            }
        }
    }

    private func attempt(_ body: () throws -> Void) {
        do { try body() } catch { self.error = error.localizedDescription }
    }

    private func title(_ s: Session) -> String {
        if !s.title.isEmpty { return s.title }
        return s.plannedSessionId == nil ? String(localized: "Session") : String(localized: "Planned session")
    }
}

struct Composing: Identifiable {
    let blockId: String
    /// The exercise the composer opens on, if any.
    var exercise: Exercise?
    /// The planned set it performs, if any.
    var planned: PlannedTarget?
    /// The band the composer opens on, if any.
    var band: String?
    var id: String { planned?.setId ?? blockId }
}

/// A set the plan asked for and the athlete has not done yet: its target in
/// the secondary colour, and the way to do it.
struct PlannedSetRow: View {
    let number: Int
    let set: SetWithElements
    let exercises: [String: Exercise]
    let reserveTarget: String?
    let calibration: Bool
    /// Planned with a band.
    let band: Bool
    /// Nil where the set is only shown, as after the session.
    let onPerform: (() -> Void)?

    var body: some View {
        HStack(alignment: .center, spacing: 12) {
            HStack(alignment: .center, spacing: 12) {
                Text("\(number)")
                    .font(Typeface.condensed(16, .semibold, relativeTo: .subheadline))
                    .monospacedDigit()
                    .foregroundStyle(Palette.textTertiary)
                    .frame(width: 28, height: 28)
                    .overlay(Circle().strokeBorder(Palette.hairline, style: StrokeStyle(lineWidth: 1.5, dash: [3, 3])))
                    .accessibilityLabel(Text("Set \(number)"))
                VStack(alignment: .leading, spacing: 3) {
                    Text(verbatim: names).font(.rowTitle).foregroundStyle(Palette.textSecondary)
                    Text(verbatim: line).font(.fine).foregroundStyle(Palette.textSecondary)
                    if band || calibration {
                        HStack(spacing: 6) {
                            if band { Chip("Band") }
                            if calibration { Chip("Calibration") }
                        }
                    }
                }
                Spacer(minLength: 4)
                if let e = set.elements.first, let m = ElementFormat.metric(e) {
                    HStack(alignment: .firstTextBaseline, spacing: 3) {
                        Text(verbatim: m.value).font(.metric).monospacedDigit().foregroundStyle(Palette.textSecondary)
                        Text(verbatim: m.unit).font(.meta).foregroundStyle(Palette.textTertiary)
                    }
                }
            }
            .accessibilityElement(children: .combine)
            if let onPerform {
                Button(action: onPerform) { Image(systemName: "play.fill") }
                    .buttonStyle(RoundIconButtonStyle())
                    .accessibilityLabel("Do this set")
            }
        }
        .padding(.vertical, 6)
    }

    private var names: String {
        self.set.elements.map { exercises[$0.exerciseId]?.name ?? String(localized: "Exercise") }.joined(separator: " + ")
    }

    private var line: String {
        [String(localized: "Target"), reserveTarget].compactMap { $0 }.joined(separator: " · ")
    }
}

/// One logged set: its number, its elements, the numbers large. A combo is
/// one set whose elements are joined by a line, in order.
struct SetRow: View {
    let number: Int
    let set: SetWithElements
    let exercises: [String: Exercise]
    /// Every band kept here, by id, to name a set's band.
    var bands: [String: Band] = [:]
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
                if let extra = ElementFormat.qualifiers(e, bands: bands) {
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
                            if let extra = ElementFormat.qualifiers(e, bands: bands) {
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

    /// Load, help and failure, when there are any: "+5 kg · Partner". A band
    /// is named when it is kept here.
    static func qualifiers(_ e: SetElement, bands: [String: Band] = [:]) -> String? {
        var parts: [String] = []
        if e.loadKg > 0 { parts.append(String(localized: "+\(e.loadKg.formatted(.number.precision(.fractionLength(0...2)))) kg")) }
        if let a = e.assistance { parts.append(BandText.assistance(a, bands: bands) ?? AssistanceKind.label(a.type)) }
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
                    .accessibilityElement(children: .combine)
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
            .onChange(of: done) { old, new in
                // The haptic's counterpart for VoiceOver.
                if !old && new { AccessibilityNotification.Announcement(String(localized: "Rest done")).post() }
            }
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
    /// The pain questions once a planned session is done (ADR 0030).
    var question: PainQuestion?
    let onFinish: (Int?, [String: PainAnswer]) -> Void
    @State private var fatigue: Int?
    @State private var pain: [String: PainAnswer] = [:]
    @Environment(\.dynamicTypeSize) private var typeSize

    // The question is the bar's large title: at the largest sizes the bar
    // shortens it and shows it in the Large Content Viewer, where a title in
    // the content was cut.
    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    // Fewer columns at the largest text sizes, so numbers are not clipped.
                    LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 8),
                                             count: typeSize.isAccessibilitySize ? 3 : 5), spacing: 8) {
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
                            .accessibilityLabel(Text("\(n) of 10"))
                            .accessibilityAddTraits(fatigue == n ? .isSelected : [])
                        }
                    }
                    Text("Optional. 1 is easy, 10 is everything you had.")
                        .font(.detailText)
                        .foregroundStyle(Palette.textSecondary)
                        .fixedSize(horizontal: false, vertical: true)
                    if let question {
                        PainQuestionView(moment: .afterSession, regions: question.regions, answers: $pain)
                            .padding(.top, 8)
                    }
                    Button {
                        onFinish(fatigue, pain)
                    } label: {
                        Label("Finish session", systemImage: "checkmark")
                    }
                    .buttonStyle(PrimaryButtonStyle())
                }
                .padding(20)
            }
            .themedScreen()
            .navigationTitle("How hard did it feel?")
            .navigationBarTitleDisplayMode(.large)
        }
    }
}
