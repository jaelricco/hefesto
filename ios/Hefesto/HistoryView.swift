import Charts
import HefestoStore
import SwiftUI

/// What the athlete has done: sessions by week, and each exercise over time.
/// Everything is read from the local database, so it works offline.
struct HistoryView: View {
    enum Mode: Hashable { case sessions, exercises }

    @Environment(AppModel.self) private var model
    @State private var mode = Mode.sessions
    @State private var weeks: [HistoryWeek] = []
    @State private var logged: [LoggedExercise] = []
    @State private var path = NavigationPath()

    var body: some View {
        NavigationStack(path: $path) {
            List {
                Section {
                    Picker("Show", selection: $mode) {
                        Text("Sessions").tag(Mode.sessions)
                        Text("Exercises").tag(Mode.exercises)
                    }
                    .pickerStyle(.segmented)
                    .listRowBackground(Color.clear)
                    .listRowInsets(EdgeInsets())
                }
                switch mode {
                case .sessions: sessions
                case .exercises: exercises
                }
            }
            .listStyle(.insetGrouped)
            .themedScreen()
            .navigationTitle("History")
            .toolbar { ToolbarItem(placement: .topBarTrailing) { ProgressBadge() } }
            .navigationDestination(for: SessionLink.self) { SessionDetailView(sessionId: $0.id) }
            .navigationDestination(for: LoggedExercise.self) { ExerciseStatsView(logged: $0) }
        }
        .task {
            if let demo = model.demo, demo.screen == .session { path.append(SessionLink(id: demo.pastSessionId)) }
        }
        .task {
            do { for try await w in model.db.observeHistory() { weeks = w } } catch {}
        }
        .task {
            do {
                for try await l in model.db.observeLoggedExercises() {
                    logged = l
                    if let demo = model.demo, demo.screen == .stats, path.isEmpty,
                       let item = l.first(where: { $0.exercise.id == demo.exerciseId }) {
                        mode = .exercises
                        path.append(item)
                    }
                }
            } catch {}
        }
    }

    @ViewBuilder
    private var sessions: some View {
        if weeks.isEmpty {
            Text("Your sessions appear here.").font(.detailText).foregroundStyle(Palette.textSecondary).themedRow()
        }
        ForEach(weeks) { week in
            Section {
                ForEach(week.sessions) { s in
                    NavigationLink(value: SessionLink(id: s.id)) { SessionRow(session: s) }
                        .themedRow()
                }
            } header: {
                HStack {
                    CapsLabel("Week \(week.week)")
                    Spacer()
                    Text("\(week.trainingCount) sessions")
                        .font(.capsLabel)
                        .monospacedDigit()
                        .foregroundStyle(Palette.textSecondary)
                }
            }
        }
    }

    @ViewBuilder
    private var exercises: some View {
        if logged.isEmpty {
            Text("Exercises you log appear here.").font(.detailText).foregroundStyle(Palette.textSecondary).themedRow()
        }
        ForEach(logged) { l in
            NavigationLink(value: l) {
                VStack(alignment: .leading, spacing: 2) {
                    Text(verbatim: l.exercise.name).font(.rowTitle).foregroundStyle(Palette.text)
                    Text("\(l.sets) sets").font(.meta).foregroundStyle(Palette.textSecondary)
                }
                .frame(minHeight: 44)
            }
            .themedRow()
        }
    }
}

struct SessionLink: Hashable {
    let id: String
}

/// A logged session, read only. A draft can be reopened in the logger.
struct SessionDetailView: View {
    let sessionId: String
    @Environment(AppModel.self) private var model
    @State private var tree: SessionTree?
    @State private var exercises: [String: Exercise] = [:]
    @State private var logging = false

    var body: some View {
        List {
            if let tree {
                Section {
                    LabeledContent("Date") {
                        Text(tree.session.startedAt, format: .dateTime.weekday(.wide).day().month().year())
                    }
                    .themedRow()
                    if let end = tree.session.endedAt {
                        LabeledContent("Duration") {
                            Text(Duration.seconds(end.timeIntervalSince(tree.session.startedAt))
                                .formatted(.units(allowed: [.hours, .minutes], width: .abbreviated)))
                        }
                        .themedRow()
                    }
                    if let f = tree.session.perceivedFatigue {
                        LabeledContent("Effort") { Text("\(f) of 10") }.themedRow()
                    }
                    if !tree.session.notes.isEmpty { Text(verbatim: tree.session.notes).themedRow() }
                }
                .font(.bodyText)
                if tree.session.status == "draft" {
                    Section {
                        Button {
                            logging = true
                        } label: {
                            Label("Continue this session", systemImage: "play.fill")
                        }
                        .buttonStyle(PrimaryButtonStyle())
                        .listRowInsets(EdgeInsets())
                        .listRowBackground(Color.clear)
                    }
                }
                ForEach(Array(tree.blocks.enumerated()), id: \.element.id) { i, block in
                    Section {
                        ForEach(Array(block.sets.enumerated()), id: \.element.id) { n, set in
                            SetRow(number: n + 1, set: set, exercises: exercises, onRepeat: nil).themedRow()
                        }
                    } header: {
                        CapsLabel("Block \(i + 1)")
                    }
                }
            } else {
                Text("This session was deleted.").font(.detailText).foregroundStyle(Palette.textSecondary).themedRow()
            }
        }
        .listStyle(.insetGrouped)
        .themedScreen()
        .navigationTitle(tree.map { Text(verbatim: $0.session.title.isEmpty
            ? String(localized: $0.session.isRestDay ? "Rest day" : "Session") : $0.session.title) } ?? Text(verbatim: ""))
        .navigationBarTitleDisplayMode(.inline)
        .fullScreenCover(isPresented: $logging) { LoggerView(sessionId: sessionId) }
        .task {
            exercises = (try? model.db.exercisesById()) ?? [:]
            do { for try await t in model.db.observeSessionTree(id: sessionId) { tree = t } } catch {}
        }
    }
}

/// One exercise over time: bests and a chart of each day's work.
struct ExerciseStatsView: View {
    let logged: LoggedExercise
    @Environment(AppModel.self) private var model
    @State private var stats: ExerciseStats?

    private var measure: String { logged.exercise.defaultMeasure }

    var body: some View {
        List {
            if let stats, !stats.isEmpty {
                Section {
                    best("Most reps", stats.bestReps, unit: "reps")
                    best("Longest hold", stats.bestHoldSeconds, unit: "hold_seconds")
                    best("Longest distance", stats.bestDistanceM, unit: "distance_m")
                    best("Most added load", stats.maxLoadKg, unit: "kg")
                } header: {
                    CapsLabel("Bests")
                } footer: {
                    Text("Bests count full, unassisted repetitions.").font(.fine).foregroundStyle(Palette.textSecondary)
                }
                Section {
                    Chart(stats.days) { day in
                        if let y = chartValue(day) {
                            BarMark(x: .value("Day", Day.date(day.localDate) ?? .distantPast, unit: .day),
                                    y: .value("Value", y))
                                .foregroundStyle(Palette.ember)
                                .cornerRadius(3)
                        }
                    }
                    .chartXAxis {
                        AxisMarks(values: .automatic) { _ in
                            AxisGridLine().foregroundStyle(Palette.hairline)
                            AxisValueLabel().foregroundStyle(Palette.textSecondary)
                        }
                    }
                    .chartYAxis {
                        AxisMarks { _ in
                            AxisGridLine().foregroundStyle(Palette.hairline)
                            AxisValueLabel().foregroundStyle(Palette.textSecondary)
                        }
                    }
                    .frame(height: 200)
                    .padding(.vertical, 8)
                    .accessibilityLabel(Text(chartTitle))
                    .themedRow()
                } header: {
                    CapsLabel(chartTitle)
                }
                Section {
                    ForEach(stats.days.reversed()) { day in
                        HStack {
                            Text(Day.date(day.localDate) ?? .distantPast, format: .dateTime.day().month().year())
                                .font(.bodyText)
                            Spacer()
                            Text("\(day.sets) sets").font(.meta).foregroundStyle(Palette.textSecondary)
                            if let v = chartValue(day) {
                                Text(verbatim: LevelText.value(v, unit: chartUnit))
                                    .font(Typeface.condensed(20, .bold, relativeTo: .headline))
                                    .monospacedDigit()
                                    .frame(minWidth: 56, alignment: .trailing)
                            }
                        }
                        .accessibilityElement(children: .combine)
                        .themedRow()
                    }
                } header: {
                    CapsLabel("Days")
                }
            } else {
                Text("No sets logged yet.").font(.detailText).foregroundStyle(Palette.textSecondary).themedRow()
            }
        }
        .listStyle(.insetGrouped)
        .themedScreen()
        .navigationTitle(Text(verbatim: logged.exercise.name))
        .task {
            do {
                for try await s in model.db.observeExerciseStats(exerciseId: logged.exercise.id) { stats = s }
            } catch {}
        }
    }

    @ViewBuilder
    private func best(_ title: LocalizedStringKey, _ b: PersonalBest?, unit: String) -> some View {
        if let b {
            LabeledContent {
                VStack(alignment: .trailing, spacing: 0) {
                    Text(verbatim: unit == "kg"
                         ? String(localized: "+\(b.value.formatted()) kg")
                         : LevelText.value(b.value, unit: unit))
                        .font(.metricSmall)
                        .monospacedDigit()
                        .foregroundStyle(Palette.gold)
                    Text(Day.date(b.localDate) ?? .distantPast, format: .dateTime.day().month().year())
                        .font(.fine).foregroundStyle(Palette.textSecondary)
                }
            } label: {
                Text(title).font(.bodyText)
            }
            .themedRow()
        }
    }

    private var chartUnit: String {
        switch measure {
        case "hold_seconds", "distance_m": measure
        default: "reps"
        }
    }

    private var chartTitle: LocalizedStringKey {
        switch measure {
        case "hold_seconds": "Longest hold per day"
        case "distance_m": "Longest distance per day"
        default: "Reps per day"
        }
    }

    private func chartValue(_ day: ExerciseDay) -> Double? {
        switch measure {
        case "hold_seconds": day.bestHoldSeconds
        case "distance_m": day.bestDistanceM
        default: day.totalReps > 0 ? Double(day.totalReps) : nil
        }
    }
}

enum Day {
    /// A local date (YYYY-MM-DD) as a date for display, at noon so no time
    /// zone moves it to another day.
    static func date(_ s: String) -> Date? {
        let parts = s.split(separator: "-").compactMap { Int($0) }
        guard parts.count == 3 else { return nil }
        return Calendar(identifier: .gregorian).date(
            from: DateComponents(year: parts[0], month: parts[1], day: parts[2], hour: 12))
    }
}
