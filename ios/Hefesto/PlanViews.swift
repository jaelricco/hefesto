import HefestoPlan
import HefestoSync
import SwiftUI

// The training plan's screens (ADR 0022): today's planned session on Today,
// the week below it, a planned session in full, and the check-in before a
// start. They read HefestoPlan's plain model, never the API's types.

/// The words for the plan's closed vocabularies and its numbers.
enum PlanText {
    static func kind(_ kind: String) -> String {
        switch kind {
        case "light": String(localized: "Light session")
        case "deload": String(localized: "Deload session")
        default: String(localized: "Full session")
        }
    }

    static func role(_ role: String) -> String {
        switch role {
        case "warmup": String(localized: "Warm-up")
        case "balance": String(localized: "Balance")
        case "skill_max": String(localized: "Max effort")
        case "skill_volume": String(localized: "Volume")
        case "strength": String(localized: "Strength")
        case "end": String(localized: "To finish")
        default: role
        }
    }

    /// Sets and target: "3 × 6 reps", "4 × 6 s".
    static func volume(_ item: PlanItem) -> String {
        if let hold = item.holdSeconds { return String(localized: "\(item.sets) × \(hold) s") }
        if let reps = item.reps { return String(localized: "\(item.sets) × \(reps) reps") }
        return String(localized: "\(item.sets) sets")
    }

    /// What should be left at the end of each set.
    static func reserve(_ item: PlanItem) -> String {
        item.isHold
            ? String(localized: "\(item.reserve) s in reserve")
            : String(localized: "\(item.reserve) reps in reserve")
    }

    static func rest(_ seconds: Int) -> String {
        String(localized: "Rest \(Duration.seconds(seconds).formatted(.time(pattern: .minuteSecond)))")
    }

    static func minutes(_ minutes: Double) -> String {
        String(localized: "about \(Int(minutes.rounded())) min")
    }

    static func status(_ status: PlanSession.Status) -> String {
        switch status {
        case .planned: String(localized: "Planned")
        case .started: String(localized: "Started")
        case .completed: String(localized: "Completed")
        }
    }

    /// A plan day (YYYY-MM-DD) as a date at noon here, for formatting.
    static func date(_ day: String) -> Date {
        let parts = day.split(separator: "-").compactMap { Int($0) }
        guard parts.count == 3 else { return .now }
        let c = DateComponents(year: parts[0], month: parts[1], day: parts[2], hour: 12)
        return Calendar.current.date(from: c) ?? .now
    }
}

/// Where a tap on the plan leads: a planned session, looked up live so a
/// newer plan shows at once.
struct PlanRoute: Hashable {
    let sessionId: String
}

// MARK: - Today

/// Today's planned session, or the plan's rest day: what to do now.
struct TodayPlanCard: View {
    let session: PlanSession?
    let onStart: (PlanSession) -> Void
    let onContinue: (PlanSession) -> Void
    let onOpen: (PlanSession) -> Void

    var body: some View {
        if let session {
            sessionCard(session)
        } else {
            VStack(alignment: .leading, spacing: 6) {
                CapsLabel("Today's plan", isHeader: false)
                Text("Rest day").font(.displayTitle).foregroundStyle(Palette.text)
                Text("Your plan rests today. Rest days count towards the streak.")
                    .font(.detailText)
                    .foregroundStyle(Palette.textSecondary)
            }
            .card()
        }
    }

    private func sessionCard(_ s: PlanSession) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack {
                CapsLabel("Today's plan", isHeader: false)
                Spacer()
                if s.checkInApplied { Chip("Lighter today") }
            }
            VStack(alignment: .leading, spacing: 2) {
                Text(verbatim: PlanText.kind(s.kind)).font(.displayTitle).foregroundStyle(Palette.text)
                Text(verbatim: PlanText.minutes(s.estMinutes)).font(.meta).foregroundStyle(Palette.textSecondary)
            }
            VStack(alignment: .leading, spacing: 6) {
                ForEach(s.plannedItems.prefix(4)) { item in
                    HStack(alignment: .firstTextBaseline) {
                        Text(verbatim: item.exerciseName).font(.bodyMedium).foregroundStyle(Palette.text)
                        Spacer(minLength: 8)
                        Text(verbatim: PlanText.volume(item)).font(.meta).monospacedDigit()
                            .foregroundStyle(Palette.textSecondary)
                    }
                }
                if s.plannedItems.count > 4 {
                    Text("and \(s.plannedItems.count - 4) more").font(.meta).foregroundStyle(Palette.textSecondary)
                }
            }
            .accessibilityElement(children: .combine)
            action(s)
            Button("See the whole session") { onOpen(s) }
                .font(.bodyMedium)
                .foregroundStyle(Palette.textSecondary)
                .frame(maxWidth: .infinity, minHeight: 44)
        }
        .card()
    }

    @ViewBuilder
    private func action(_ s: PlanSession) -> some View {
        switch s.status {
        case .planned:
            Button { onStart(s) } label: { Label("Start", systemImage: "play.fill") }
                .buttonStyle(PrimaryButtonStyle())
        case .started:
            Button { onContinue(s) } label: { Label("Continue", systemImage: "play.fill") }
                .buttonStyle(PrimaryButtonStyle())
        case .completed:
            Label("Done for today", systemImage: "checkmark")
                .font(.bodyMedium)
                .foregroundStyle(Palette.gold)
                .frame(maxWidth: .infinity, minHeight: 44, alignment: .leading)
        }
    }
}

/// No plan yet: the onboarding that brings one leads on Today.
struct OnboardingCard: View {
    let onStart: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            CapsLabel("Your plan", isHeader: false)
            Text("Get your training plan").font(.displayTitle).foregroundStyle(Palette.text)
            Text("A few questions, about five minutes. Hefesto then plans your week around your goals, your time and your equipment.")
                .font(.detailText)
                .foregroundStyle(Palette.textSecondary)
            Button(action: onStart) {
                Label("Answer the questions", systemImage: "sparkles")
            }
            .buttonStyle(PrimaryButtonStyle())
        }
        .card()
    }
}

/// The week of the plan, day by day. A rest day is a day of the plan like
/// any other (ADR 0003).
struct PlanWeekList: View {
    let week: PlanWeek
    let today: String
    let onOpen: (PlanSession) -> Void

    var body: some View {
        VStack(spacing: 0) {
            ForEach(Array(week.days.enumerated()), id: \.element.id) { i, day in
                if i > 0 { Divider().overlay(Palette.hairline).padding(.leading, 70) }
                if let session = day.session {
                    Button { onOpen(session) } label: { row(day, session) }
                        .buttonStyle(.plain)
                } else {
                    row(day, nil)
                }
            }
        }
        .background(Palette.surface, in: .rect(cornerRadius: 16))
    }

    private func row(_ day: PlanDay, _ session: PlanSession?) -> some View {
        HStack(spacing: 14) {
            DateBlock(date: PlanText.date(day.date))
            VStack(alignment: .leading, spacing: 2) {
                Text(verbatim: session.map { PlanText.kind($0.kind) } ?? String(localized: "Rest day"))
                    .font(.rowTitle)
                    .foregroundStyle(session == nil ? Palette.textSecondary : Palette.text)
                if let session {
                    Text(verbatim: "\(PlanText.minutes(session.estMinutes)) · \(PlanText.status(session.status))")
                        .font(.meta)
                        .foregroundStyle(Palette.textSecondary)
                }
            }
            Spacer(minLength: 8)
            if day.date == today { Chip("Today") }
            if let session {
                if session.status == .completed {
                    Image(systemName: "checkmark").font(.body.weight(.semibold)).foregroundStyle(Palette.gold)
                        .accessibilityLabel("Completed")
                } else {
                    Image(systemName: "chevron.right").foregroundStyle(Palette.textTertiary)
                }
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 10)
        .frame(minHeight: 64)
        .contentShape(.rect)
        .accessibilityElement(children: .combine)
    }
}

/// Training is stopped: the plan says why and what lifts the stop, in the
/// knowledge base's words. The app adds no advice of its own.
struct PlanStoppedCard: View {
    let week: PlanWeek
    /// Confirms a professional's clearance (ADR 0028); nil when none can
    /// lift the stop, as for a minor (SAFE-07).
    var onClearance: (() -> Void)?

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            CapsLabel("Your plan", isHeader: false)
            Text("Training is paused").font(.displayTitle).foregroundStyle(Palette.text)
            ForEach(Array(week.reasons.enumerated()), id: \.offset) { _, note in
                Text(verbatim: note.text).font(.detailText).foregroundStyle(Palette.text)
            }
            if let onClearance {
                Button("A professional cleared me", action: onClearance)
                    .font(.bodyMedium)
                    .foregroundStyle(Palette.ember)
                    .frame(minHeight: 44)
            }
            Text(verbatim: week.disclaimer).font(.fine).foregroundStyle(Palette.textSecondary)
        }
        .card()
    }
}

// MARK: - A planned session

/// A planned session in full: every block and item with its targets, and
/// why the plan chose them.
struct PlanSessionView: View {
    let route: PlanRoute
    let onStart: (PlanSession) -> Void
    let onContinue: (PlanSession) -> Void
    @Environment(AppModel.self) private var model

    var body: some View {
        Group {
            if let session = model.plan.week?.session(id: route.sessionId) {
                content(session)
            } else {
                ContentUnavailableView(
                    "This session is no longer in your plan", systemImage: "calendar.badge.exclamationmark",
                    description: Text("Your plan changed. Today shows the new one."))
            }
        }
        .themedScreen()
        .navigationBarTitleDisplayMode(.inline)
    }

    private func content(_ s: PlanSession) -> some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 0) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(PlanText.date(s.date), format: .dateTime.weekday(.wide).day().month(.wide))
                        .font(.capsLabel)
                        .tracking(1.2)
                        .textCase(.uppercase)
                        .foregroundStyle(Palette.textSecondary)
                    Text(verbatim: PlanText.kind(s.kind))
                        .font(.displayLarge)
                        .foregroundStyle(Palette.text)
                        .accessibilityAddTraits(.isHeader)
                    HStack(spacing: 8) {
                        Text(verbatim: PlanText.minutes(s.estMinutes)).font(.meta)
                            .foregroundStyle(Palette.textSecondary)
                        Chip(verbatim: PlanText.status(s.status))
                        if s.checkInApplied { Chip("Lighter today") }
                    }
                }
                .padding(.horizontal, 20)
                .padding(.top, 8)

                if s.checkInApplied {
                    Text("After your check-in, the hardest block is technique work today.")
                        .font(.detailText)
                        .foregroundStyle(Palette.textSecondary)
                        .padding(.horizontal, 20)
                        .padding(.top, 10)
                }

                ForEach(Array(s.blocks.enumerated()), id: \.offset) { _, block in
                    blockView(block)
                }

                if !s.reasons.isEmpty {
                    CapsLabel("Why this session").padding(.horizontal, 20).padding(.top, 28).padding(.bottom, 10)
                    VStack(alignment: .leading, spacing: 10) {
                        ForEach(Array(s.reasons.enumerated()), id: \.offset) { _, note in
                            Text(verbatim: note.text).font(.detailText).foregroundStyle(Palette.text)
                        }
                    }
                    .card()
                    .padding(.horizontal, 16)
                }

                if let week = model.plan.week {
                    Text(verbatim: week.disclaimer)
                        .font(.fine)
                        .foregroundStyle(Palette.textSecondary)
                        .padding(.horizontal, 20)
                        .padding(.top, 20)
                }

                action(s).padding(.horizontal, 16).padding(.vertical, 24)
            }
        }
    }

    private func blockView(_ block: PlanBlock) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(alignment: .firstTextBaseline) {
                CapsLabel(verbatim: PlanText.role(block.role))
                if let minutes = block.minutes {
                    Text(verbatim: PlanText.minutes(minutes)).font(.fine).foregroundStyle(Palette.textSecondary)
                }
                if block.paired { Chip("Alternate") }
            }
            .padding(.horizontal, 20)
            .padding(.top, 28)
            .padding(.bottom, 6)
            if let why = block.reasons.first {
                Text(verbatim: why.text)
                    .font(.fine)
                    .foregroundStyle(Palette.textSecondary)
                    .padding(.horizontal, 20)
                    .padding(.bottom, 10)
            }
            if !block.items.isEmpty {
                VStack(spacing: 0) {
                    ForEach(Array(block.items.enumerated()), id: \.element.id) { i, item in
                        if i > 0 { Divider().overlay(Palette.hairline).padding(.leading, 16) }
                        PlanItemRow(item: item)
                    }
                }
                .background(Palette.surface, in: .rect(cornerRadius: 16))
                .padding(.horizontal, 16)
            }
        }
    }

    @ViewBuilder
    private func action(_ s: PlanSession) -> some View {
        if s.date == model.plan.today {
            switch s.status {
            case .planned:
                Button { onStart(s) } label: { Label("Start", systemImage: "play.fill") }
                    .buttonStyle(PrimaryButtonStyle())
            case .started:
                Button { onContinue(s) } label: { Label("Continue", systemImage: "play.fill") }
                    .buttonStyle(PrimaryButtonStyle())
            case .completed:
                EmptyView()
            }
        }
    }
}

/// One item of a planned session: the exercise, its sets and targets, and
/// what is special about it.
struct PlanItemRow: View {
    let item: PlanItem

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline) {
                Text(verbatim: item.exerciseName).font(.rowTitle).foregroundStyle(Palette.text)
                Spacer(minLength: 8)
                Text(verbatim: PlanText.volume(item)).font(.metricSmall).monospacedDigit()
                    .foregroundStyle(item.offer ? Palette.textSecondary : Palette.text)
            }
            Text(verbatim: details)
                .font(.meta)
                .monospacedDigit()
                .foregroundStyle(Palette.textSecondary)
            if !chips.isEmpty {
                HStack(spacing: 6) {
                    ForEach(chips, id: \.self) { Chip(verbatim: $0) }
                }
            }
            if let why = item.reasons.first {
                Text(verbatim: why.text)
                    .font(.fine)
                    .foregroundStyle(Palette.textSecondary)
                    .lineLimit(3)
            }
        }
        .padding(16)
        .frame(maxWidth: .infinity, alignment: .leading)
        .accessibilityElement(children: .combine)
    }

    private var details: String {
        var parts = [PlanText.reserve(item), PlanText.rest(item.restSeconds)]
        if let load = item.loadKg, load > 0 {
            parts.insert(String(localized: "+\(load.formatted(.number.precision(.fractionLength(0...1)))) kg"), at: 0)
        }
        return parts.joined(separator: " · ")
    }

    private var chips: [String] {
        var out: [String] = []
        if item.offer { out.append(String(localized: "Optional")) }
        if item.calibration { out.append(String(localized: "Calibration")) }
        if item.assist == "band" { out.append(String(localized: "Band")) }
        if item.kind == "warmup" { out.append(String(localized: "Warm-up set")) }
        if item.monitor { out.append(String(localized: "Watch for pain")) }
        return out
    }
}

// MARK: - Check-in

/// Two optional questions before a planned session starts (ADR 0019). After
/// short sleep or high fatigue, the server makes the session lighter. The
/// answers go only into the start request; nothing here keeps them.
///
/// Without a connection the sheet says so and offers the start without the
/// server (ADR 0027): the session starts as planned, without the check-in.
struct CheckInSheet: View {
    let session: PlanSession
    /// Starts the session; says why when it could not.
    let onStart: (Double?, Int?) async -> StartFailure?
    /// Starts the session without the server; returns a message when it
    /// could not.
    let onStartOffline: () -> String?
    /// Opens on the offline choice, as a demo screen does.
    var initiallyOffline = false

    @Environment(\.dismiss) private var dismiss
    @State private var sleep: Int?
    @State private var fatigue: Int?
    @State private var starting = false
    @State private var failure: String?
    @State private var offline = false

    /// Hours of sleep to choose from; the ends stand for less and more.
    static let sleepChoices = [4, 5, 6, 7, 8, 9]

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Before you start").font(.displayTitle).foregroundStyle(Palette.text)
                        Text("Two quick questions. Both are optional.")
                            .font(.detailText)
                            .foregroundStyle(Palette.textSecondary)
                    }

                    VStack(alignment: .leading, spacing: 10) {
                        CapsLabel("Sleep last night")
                        LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 8), count: 3), spacing: 8) {
                            ForEach(Self.sleepChoices, id: \.self) { hours in
                                choice(label: sleepLabel(hours), selected: sleep == hours) {
                                    sleep = sleep == hours ? nil : hours
                                }
                            }
                        }
                    }

                    VStack(alignment: .leading, spacing: 10) {
                        CapsLabel("How tired are you?")
                        LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 8), count: 5), spacing: 8) {
                            ForEach(1...10, id: \.self) { n in
                                choice(label: "\(n)", selected: fatigue == n) {
                                    fatigue = fatigue == n ? nil : n
                                }
                            }
                        }
                        Text("1 is fresh, 10 is exhausted.")
                            .font(.detailText)
                            .foregroundStyle(Palette.textSecondary)
                    }

                    Text("After 6 hours of sleep or less, or a tiredness of 8 or more, the hardest block becomes technique work today. Your answers are not stored.")
                        .font(.detailText)
                        .foregroundStyle(Palette.textSecondary)

                    if let failure {
                        Label {
                            Text(verbatim: failure)
                        } icon: {
                            Image(systemName: "exclamationmark.circle")
                        }
                        .font(.detailText)
                        .foregroundStyle(Palette.text)
                    }

                    if offline {
                        offlineChoice
                    } else {
                        Button {
                            Task { await start() }
                        } label: {
                            if starting {
                                ProgressView().tint(Palette.emberInk)
                            } else {
                                Label("Start session", systemImage: "play.fill")
                            }
                        }
                        .buttonStyle(PrimaryButtonStyle())
                        .disabled(starting)
                    }
                }
                .padding(20)
            }
            .themedScreen()
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
            }
            .onAppear { offline = offline || initiallyOffline }
        }
    }

    /// No connection: the session can start here as planned, and the plan
    /// catches up at the next sync. The check-in needs the server.
    private var offlineChoice: some View {
        VStack(alignment: .leading, spacing: 12) {
            Label {
                Text("No connection. Without one, the session starts as planned: making it lighter after short sleep or high tiredness needs the server. Your plan takes the session up at the next sync.")
            } icon: {
                Image(systemName: "wifi.slash")
            }
            .font(.detailText)
            .foregroundStyle(Palette.text)
            Button {
                failure = onStartOffline()
            } label: {
                Label("Start without connection", systemImage: "play.fill")
            }
            .buttonStyle(PrimaryButtonStyle())
            Button {
                Task { await start() }
            } label: {
                if starting {
                    ProgressView().tint(Palette.textSecondary)
                } else {
                    Text("Try again")
                }
            }
            .font(.bodyMedium)
            .foregroundStyle(Palette.textSecondary)
            .frame(maxWidth: .infinity, minHeight: 44)
            .disabled(starting)
        }
    }

    private func start() async {
        starting = true
        failure = nil
        switch await onStart(sleep.map { Double($0) }, fatigue) {
        case nil: offline = false
        case .offline?: offline = true
        case let .message(text)?:
            offline = false
            failure = text
        }
        starting = false
    }

    private func sleepLabel(_ hours: Int) -> String {
        if hours == Self.sleepChoices.first { return String(localized: "\(hours) h or less") }
        if hours == Self.sleepChoices.last { return String(localized: "\(hours) h or more") }
        return String(localized: "\(hours) h")
    }

    private func choice(label: String, selected: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Text(verbatim: label)
                .font(.metricSmall)
                .monospacedDigit()
                .lineLimit(1)
                .minimumScaleFactor(0.6)
                .foregroundStyle(selected ? Palette.emberInk : Palette.text)
                .frame(maxWidth: .infinity, minHeight: 52)
                .padding(.horizontal, 4)
                .background(selected ? Palette.ember : Palette.surface, in: .rect(cornerRadius: 12))
        }
        .buttonStyle(.plain)
        .accessibilityAddTraits(selected ? .isSelected : [])
    }
}

/// How a start in the check-in sheet failed.
enum StartFailure: Equatable {
    /// Why, in the athlete's words.
    case message(String)
    /// No connection: the session can start without the server.
    case offline
}

/// Why a start did not happen, in the athlete's terms.
enum PlanStartMessage {
    static func text(_ error: any Error) -> String {
        if let e = error as? OfflineStartError {
            return switch e {
            case .noPlan:
                String(localized: "This device keeps no plan for the day. Starting needs a connection.")
            case .startedElsewhere:
                String(localized: "This session was started on another device. Continuing it here needs a connection.")
            case .unknownExercise:
                String(localized: "An exercise of this session is not on this device yet. Starting it needs a connection.")
            }
        }
        if (error as? PlanStartError) == .offline {
            return String(localized: "No connection. Continuing this session here needs one.")
        }
        return switch error as? PlanError {
        case .trainingStopped:
            String(localized: "Training is paused. Your plan says why.")
        case .notFound:
            String(localized: "Your plan has changed. Have a look at today's session again.")
        case .unavailable:
            String(localized: "The planner is not available right now. Try again later.")
        case .onboardingRequired:
            String(localized: "Your plan needs the onboarding first.")
        case .refused, nil:
            String(localized: "The session could not start. Try again later.")
        }
    }
}
