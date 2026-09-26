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

    var body: some View {
        NavigationStack {
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
            .navigationTitle("History")
            .toolbar { ToolbarItem(placement: .topBarLeading) { ProgressBadge() } }
            .navigationDestination(for: SessionLink.self) { SessionDetailView(sessionId: $0.id) }
            .navigationDestination(for: LoggedExercise.self) { ExerciseStatsView(logged: $0) }
        }
        .task {
            do { for try await w in model.db.observeHistory() { weeks = w } } catch {}
        }
        .task {
            do { for try await l in model.db.observeLoggedExercises() { logged = l } } catch {}
        }
    }

    @ViewBuilder
    private var sessions: some View {
        if weeks.isEmpty {
            Text("Your sessions appear here.").foregroundStyle(.secondary)
        }
        ForEach(weeks) { week in
            Section {
                ForEach(week.sessions) { s in
                    NavigationLink(value: SessionLink(id: s.id)) { SessionRow(session: s) }
                }
            } header: {
                HStack {
                    Text("Week \(week.week)")
                    Spacer()
                    Text("\(week.trainingCount) sessions").monospacedDigit()
                }
            }
        }
    }

    @ViewBuilder
    private var exercises: some View {
        if logged.isEmpty {
            Text("Exercises you log appear here.").foregroundStyle(.secondary)
        }
        ForEach(logged) { l in
            NavigationLink(value: l) {
                VStack(alignment: .leading) {
                    Text(verbatim: l.exercise.name).font(.headline)
                    Text("\(l.sets) sets").font(.caption).foregroundStyle(.secondary)
                }
                .frame(minHeight: 44)
            }
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
                    if let end = tree.session.endedAt {
                        LabeledContent("Duration") {
                            Text(Duration.seconds(end.timeIntervalSince(tree.session.startedAt))
                                .formatted(.units(allowed: [.hours, .minutes], width: .abbreviated)))
                        }
                    }
                    if let f = tree.session.perceivedFatigue {
                        LabeledContent("Effort") { Text("\(f) of 10") }
                    }
                    if !tree.session.notes.isEmpty { Text(verbatim: tree.session.notes) }
                    if tree.session.status == "draft" {
                        Button("Continue this session", systemImage: "play.fill") { logging = true }
                            .frame(minHeight: 44)
                    }
                }
                ForEach(Array(tree.blocks.enumerated()), id: \.element.id) { i, block in
                    Section("Block \(i + 1)") {
                        ForEach(Array(block.sets.enumerated()), id: \.element.id) { n, set in
                            SetRow(number: n + 1, set: set, exercises: exercises, onRepeat: nil)
                        }
                    }
                }
            } else {
                Text("This session was deleted.").foregroundStyle(.secondary)
            }
        }
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
                Section("Bests") {
                    best("Most reps", stats.bestReps, unit: "reps")
                    best("Longest hold", stats.bestHoldSeconds, unit: "hold_seconds")
                    best("Longest distance", stats.bestDistanceM, unit: "distance_m")
                    best("Most added load", stats.maxLoadKg, unit: "kg")
                    Text("Bests count full, unassisted repetitions.").font(.caption).foregroundStyle(.secondary)
                }
                Section(chartTitle) {
                    Chart(stats.days) { day in
                        if let y = chartValue(day) {
                            BarMark(x: .value("Day", Day.date(day.localDate) ?? .distantPast, unit: .day),
                                    y: .value("Value", y))
                                .foregroundStyle(.yellow.gradient)
                        }
                    }
                    .frame(height: 200)
                    .accessibilityLabel(Text(chartTitle))
                }
                Section("Days") {
                    ForEach(stats.days.reversed()) { day in
                        HStack {
                            Text(Day.date(day.localDate) ?? .distantPast, format: .dateTime.day().month().year())
                            Spacer()
                            Text("\(day.sets) sets").foregroundStyle(.secondary)
                            if let v = chartValue(day) {
                                Text(verbatim: LevelText.value(v, unit: chartUnit)).monospacedDigit()
                            }
                        }
                        .accessibilityElement(children: .combine)
                    }
                }
            } else {
                Text("No sets logged yet.").foregroundStyle(.secondary)
            }
        }
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
            LabeledContent(title) {
                VStack(alignment: .trailing) {
                    Text(verbatim: unit == "kg"
                         ? String(localized: "+\(b.value.formatted()) kg")
                         : LevelText.value(b.value, unit: unit))
                        .font(.headline.monospacedDigit())
                    Text(Day.date(b.localDate) ?? .distantPast, format: .dateTime.day().month().year())
                        .font(.caption).foregroundStyle(.secondary)
                }
            }
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
