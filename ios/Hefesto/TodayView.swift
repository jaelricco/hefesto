import HefestoLogger
import HefestoPlan
import HefestoStore
import SwiftUI

/// Today: the plan's session for today, start a session or plan rest, what
/// was kept, the week of the plan, and the sessions logged so far.
struct TodayView: View {
    @Environment(AppModel.self) private var model
    @State private var sessions: [Session] = []
    @State private var pending = 0
    @State private var progress: AthleteProgress?
    @State private var open: OpenSession?
    @State private var error: String?
    @State private var confirmingSignOut = false
    @Environment(\.dynamicTypeSize) private var typeSize
    @State private var path: [PlanRoute] = []
    /// The planned session whose check-in is open.
    @State private var checkingIn: PlanSession?
    /// The log session a start returned, opened once the check-in is gone.
    @State private var started: String?
    @State private var onboarding = false
    /// The body screen, and the sheet it opens on in a demo.
    @State private var showingBody = false
    @State private var showingSettings = false
    @State private var showingHealthData = false
    @State private var bodyOpening: BodySheet?
    @State private var confirmingClearance = false
    @State private var clearanceFailure: String?
    /// Warning signs the planner asked for after the pain before a session
    /// (ADR 0030): asked before the log opens.
    @State private var flagsBeforeStart: RegionRoute?
    /// Why the pain before a session was refused; said before the log opens.
    @State private var failureBeforeStart: String?
    @State private var askingFlags: RegionRoute?
    /// What answering pain questions changed.
    @State private var painOutcome: BodyOutcome?
    /// The morning question about yesterday's session, and the sheet for it.
    @State private var morning: PainQuestion?
    @State private var answeringMorning: PainQuestion?
    /// The log session whose morning question was put aside for the day.
    @AppStorage("fit.hefesto.morningPainLater") private var morningLater = ""

    var body: some View {
        NavigationStack(path: $path) {
            ScrollView {
                VStack(alignment: .leading, spacing: 0) {
                    header
                    Text("Today")
                        .font(.displayLarge)
                        .foregroundStyle(Palette.text)
                        .padding(.horizontal, 20)
                        .accessibilityAddTraits(.isHeader)
                    if let progress { ProgressTiles(progress: progress).padding(.horizontal, 16).padding(.top, 14) }
                    if let morning, morning.sessionId != morningLater {
                        MorningPainCard(question: morning, onAnswer: { answeringMorning = morning },
                                        onLater: { morningLater = morning.sessionId })
                            .padding(.horizontal, 16).padding(.top, 16)
                    }
                    plan.padding(.horizontal, 16).padding(.top, 16)
                    actions.padding(.horizontal, 16).padding(.top, 16)
                    bodyRow.padding(.horizontal, 16).padding(.top, 12)
                    if let week = model.plan.week, !week.stopped {
                        CapsLabel("This week").padding(.horizontal, 20).padding(.top, 28).padding(.bottom, 10)
                        PlanWeekList(week: week, today: model.plan.today) { path.append(PlanRoute(sessionId: $0.id)) }
                            .padding(.horizontal, 16)
                    }
                    CapsLabel("Sessions").padding(.horizontal, 20).padding(.top, 28).padding(.bottom, 10)
                    sessionList.padding(.horizontal, 16).padding(.bottom, 24)
                }
            }
            .themedScreen()
            .toolbar(.hidden, for: .navigationBar)
            .refreshable { await model.syncNow() }
            .navigationDestination(for: PlanRoute.self) { route in
                PlanSessionView(route: route, onStart: { checkingIn = $0 }, onContinue: resume)
            }
            .navigationDestination(isPresented: $showingBody) { BodyView(opening: bodyOpening) }
            .navigationDestination(isPresented: $showingSettings) { PlanSettingsView() }
            .navigationDestination(isPresented: $showingHealthData) { HealthDataView() }
            .confirmationDialog(
                "Did a professional clear you to train?", isPresented: $confirmingClearance, titleVisibility: .visible
            ) {
                Button("Yes, I was cleared") { Task { await clearTraining() } }
                Button("Cancel", role: .cancel) {}
            } message: {
                Text("Confirm only what a doctor told you. Training then starts again, carefully.")
            }
            .fullScreenCover(item: $open) { LoggerView(sessionId: $0.id) }
            .fullScreenCover(isPresented: $onboarding) { OnboardingView() }
            .sheet(item: $checkingIn, onDismiss: openStarted) { session in
                CheckInSheet(
                    session: session,
                    painRegions: model.plan.painRegions(for: session),
                    onStart: { sleep, fatigue, pain in
                        await start(session, sleepHours: sleep, fatigue: fatigue, pain: pain)
                    },
                    onStartOffline: { pain in startOffline(session, pain: pain) },
                    initiallyOffline: model.demo?.screen == .offlinestart)
            }
            .sheet(item: $askingFlags, onDismiss: openStarted) { route in
                RedFlagSheet(regionId: route.id) { painOutcome = $0 }
            }
            .sheet(item: $answeringMorning, onDismiss: findMorning) { question in
                MorningPainSheet(question: question) { reported($0) }
            }
            .alert(
                painOutcome?.waiting == true ? "Kept" : "Sent", isPresented: Binding(get: { painOutcome != nil && askingFlags == nil }, set: { if !$0 { painOutcome = nil } }),
                presenting: painOutcome
            ) { _ in
                Button("OK") { painOutcome = nil }
            } message: { outcome in
                Text(verbatim: BodyText.summary(outcome))
            }
            .alert("Something went wrong", isPresented: .constant(error != nil)) {
                Button("OK") { error = nil }
            } message: {
                Text(verbatim: error ?? "")
            }
            .task { await model.plan.observe() }
            .task {
                guard let demo = model.demo else { return }
                try? model.plan.reload()
                if demo.onboardingPage != nil { onboarding = true }
                switch demo.screen {
                case .body: showingBody = true
                case .pain: bodyOpening = .pain; showingBody = true
                case .symptoms: bodyOpening = .symptoms; showingBody = true
                case .settings: showingSettings = true
                case .consent: showingHealthData = true
                default: break
                }
                if let id = demo.opensSessionId { open = OpenSession(id: id) }
                if let today = model.plan.todaySession {
                    if demo.screen == .checkin || demo.screen == .offlinestart { checkingIn = today }
                    if demo.screen == .plansession { path = [PlanRoute(sessionId: today.id)] }
                }
            }
            .task {
                do {
                    for try await s in model.db.observeRecentSessions() {
                        sessions = s
                        findMorning()
                    }
                } catch {}
            }
            .task {
                do { for try await n in model.db.observePendingOpCount() { pending = n } } catch {}
            }
            .task {
                do { for try await p in model.db.observeProgress() { progress = p } } catch {}
            }
        }
    }

    /// The date, and beside it sync and the account. At the accessibility
    /// sizes the date takes a row of its own, so it is never cut.
    private var header: some View {
        Group {
            if typeSize.isAccessibilitySize {
                VStack(alignment: .leading, spacing: 4) {
                    HStack(spacing: 8) { Spacer(); headerControls }
                    dateLabel
                }
            } else {
                HStack(spacing: 8) {
                    dateLabel
                    Spacer()
                    headerControls
                }
            }
        }
        .padding(.leading, 20)
        .padding(.trailing, 12)
        .frame(minHeight: 44)
        // Signing out erases the phone's copy (REVIEW_DECISIONS D-16), so
        // changes not yet sent would be lost.
        .confirmationDialog("Sign out?", isPresented: $confirmingSignOut, titleVisibility: .visible) {
            Button("Sync and sign out") { signOut(syncFirst: true) }
            Button("Sign out and discard", role: .destructive) { signOut(syncFirst: false) }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("\(pending) changes on this phone are not synced yet. Signing out erases everything stored on this phone.")
        }
    }

    private var dateLabel: some View {
        Text(Date.now, format: .dateTime.weekday(.wide).day().month(.wide))
            .font(.capsLabel)
            .tracking(1.2)
            .textCase(.uppercase)
            .foregroundStyle(Palette.textSecondary)
            .fixedSize(horizontal: false, vertical: true)
    }

    private var headerControls: some View {
        HStack(spacing: 8) {
            SyncBadge(pending: pending)
            Menu {
                if model.plan.week != nil {
                    Button("Plan settings", systemImage: "slider.horizontal.3") { showingSettings = true }
                    Button("Health data", systemImage: "lock.shield") { showingHealthData = true }
                }
                Button("Sync now", systemImage: "arrow.triangle.2.circlepath") {
                    Task { await model.syncNow() }
                }
                Button("Sign out", systemImage: "rectangle.portrait.and.arrow.right", role: .destructive) {
                    if pending > 0 { confirmingSignOut = true } else { signOut(syncFirst: false) }
                }
            } label: {
                Image(systemName: "person.crop.circle")
                    .font(.system(size: 24))
                    .foregroundStyle(Palette.textSecondary)
                    .frame(width: 44, height: 44)
            }
            .accessibilityLabel("Account")
        }
    }

    /// Today's planned session, a rest day of the plan, or a stop. Without a
    /// plan, the onboarding that brings one.
    @ViewBuilder
    private var plan: some View {
        if model.plan.status == .needsOnboarding {
            OnboardingCard { onboarding = true }
        } else if let week = model.plan.week {
            if week.stopped {
                PlanStoppedCard(week: week, onClearance: clearance(week))
                if let clearanceFailure {
                    Text(verbatim: clearanceFailure).font(.detailText).foregroundStyle(Palette.text)
                }
            } else {
                TodayPlanCard(
                    session: model.plan.todaySession, onStart: { checkingIn = $0 }, onContinue: resume,
                    onOpen: { path.append(PlanRoute(sessionId: $0.id)) })
            }
        }
    }

    /// The body: report pain or symptoms, and how each region stands.
    private var bodyRow: some View {
        Button {
            bodyOpening = nil
            showingBody = true
        } label: {
            HStack {
                Label("Body: pain and symptoms", systemImage: "figure.arms.open")
                    .font(.bodyMedium)
                    .foregroundStyle(Palette.text)
                Spacer()
                Image(systemName: "chevron.right").foregroundStyle(Palette.textSecondary)
            }
            .frame(minHeight: 44)
            .contentShape(.rect)
        }
        .buttonStyle(.plain)
        .padding(.horizontal, 16)
        .background(Palette.surface, in: .rect(cornerRadius: 16))
    }

    /// The clearance a stop offers; a minor's stop (SAFE-07) is not lifted
    /// by one.
    private func clearance(_ week: PlanWeek) -> (() -> Void)? {
        if week.reasons.contains(where: { $0.ruleId == "SAFE-07" }) { return nil }
        return { confirmingClearance = true }
    }

    /// A professional cleared the athlete: the stop ends (spec §8.3).
    private func clearTraining() async {
        clearanceFailure = nil
        do {
            _ = try await model.body.confirmClearance(region: nil)
        } catch {
            clearanceFailure = BodyText.failure(error)
        }
    }

    /// The plan's session leads while it is open; a free session is then a
    /// second choice, never a second primary action.
    private var planLeads: Bool {
        if model.plan.status == .needsOnboarding { return true }
        guard let week = model.plan.week else { return false }
        if week.stopped { return true }
        return model.plan.todaySession.map { $0.status != .completed } ?? false
    }

    private var actions: some View {
        VStack(spacing: 10) {
            if planLeads {
                Button {
                    start(restDay: false)
                } label: {
                    Label {
                        Text("Free session")
                    } icon: {
                        Image(systemName: "plus").foregroundStyle(Palette.textSecondary)
                    }
                }
                .buttonStyle(SecondaryButtonStyle())
            } else {
                Button {
                    start(restDay: false)
                } label: {
                    Label("Start a session", systemImage: "play.fill")
                }
                .buttonStyle(PrimaryButtonStyle())
            }

            // Planned rest counts as training kept (ADR 0003).
            Button {
                start(restDay: true)
            } label: {
                Label {
                    Text("Log a rest day")
                } icon: {
                    Image(systemName: "moon").foregroundStyle(Palette.textSecondary)
                }
            }
            .buttonStyle(SecondaryButtonStyle())
        }
    }

    @ViewBuilder
    private var sessionList: some View {
        if sessions.isEmpty {
            Text("Your sessions appear here.")
                .font(.detailText)
                .foregroundStyle(Palette.textSecondary)
                .card()
        } else {
            VStack(spacing: 0) {
                ForEach(Array(sessions.enumerated()), id: \.element.id) { i, s in
                    if i > 0 { Divider().overlay(Palette.hairline).padding(.leading, 70) }
                    Button { open = OpenSession(id: s.id) } label: {
                        SessionRow(session: s).padding(.horizontal, 16)
                    }
                    .buttonStyle(.plain)
                }
            }
            .background(Palette.surface, in: .rect(cornerRadius: 16))
        }
    }

    private func signOut(syncFirst: Bool) {
        Task {
            do {
                if syncFirst {
                    guard try await model.syncAndSignOut() else {
                        error = String(localized: "Some changes could not be synced, so you are still signed in. Try again when you are online.")
                        return
                    }
                } else {
                    try await model.signOut()
                }
            } catch {
                self.error = error.localizedDescription
            }
        }
    }

    /// Starts a planned session after its check-in. What is waiting to sync
    /// goes first, so the plan adapts to the latest log. Says why when the
    /// start did not happen; without a connection the athlete may start
    /// without the server. A demo never has one.
    private func start(
        _ session: PlanSession, sleepHours: Double?, fatigue: Int?, pain: [String: PainAnswer] = [:]
    ) async -> StartFailure? {
        failureBeforeStart = nil
        await model.syncNow()
        do {
            let id = try await model.plan.start(session, sleepHours: sleepHours, fatigue: fatigue)
            started = id
            // The pain before the session goes with it (ADR 0030); warning
            // signs the planner asks for come before the log.
            if let outcome = await reportBefore(session, logSessionId: id, pain: pain) {
                flagsBeforeStart = outcome.askRedFlags.first.map(RegionRoute.init)
            }
            checkingIn = nil
            return nil
        } catch PlanStartError.offline, PlanStartError.noServer {
            return .offline
        } catch {
            return .message(PlanStartMessage.text(error))
        }
    }

    /// Starts a planned session here, as planned and without the check-in
    /// (ADR 0027). The next sync takes it to the server, and the pain
    /// answers with it.
    private func startOffline(_ session: PlanSession, pain: [String: PainAnswer] = [:]) -> String? {
        do {
            let id = try model.plan.startOffline(session)
            started = id
            Task { await reportBefore(session, logSessionId: id, pain: pain) }
            checkingIn = nil
            return nil
        } catch {
            return PlanStartMessage.text(error)
        }
    }

    /// Sends the pain before a session; reports that cannot go yet wait for
    /// the next sync. A refusal is said before the log opens.
    @discardableResult
    private func reportBefore(_ session: PlanSession, logSessionId: String, pain: [String: PainAnswer]) async -> BodyOutcome? {
        guard let question = model.plan.beforeQuestion(for: session, logSessionId: logSessionId) else { return nil }
        let reports = question.reports(pain, at: Date())
        guard !reports.isEmpty else { return nil }
        do {
            return try await model.body.reportPain(reports)
        } catch {
            failureBeforeStart = BodyText.failure(error)
            return nil
        }
    }

    /// The morning question, unless answered or yesterday had none.
    private func findMorning() {
        morning = model.demo == nil || model.demo?.screen == .morning ? model.plan.morningQuestion() : nil
    }

    /// What the morning answers changed. The planner may ask a region's
    /// warning signs; their sheet opens once this one is gone.
    private func reported(_ outcome: BodyOutcome) {
        guard let region = outcome.askRedFlags.first else {
            painOutcome = outcome
            return
        }
        Task {
            try? await Task.sleep(for: .milliseconds(600))
            askingFlags = RegionRoute(id: region)
        }
    }

    /// Opens a started session's log, starting it again on the server if
    /// it is not here yet: the server answers with the same session.
    private func resume(_ session: PlanSession) {
        if let id = model.plan.logSession(for: session) {
            open = OpenSession(id: id)
            return
        }
        Task {
            switch await start(session, sleepHours: nil, fatigue: nil) {
            case nil: openStarted()
            case .offline?: error = PlanStartMessage.text(PlanStartError.offline)
            case let .message(text)?: error = text
            }
        }
    }

    /// Opens the log of a session just started, once the warning signs
    /// asked before it are answered; not if they stopped training.
    private func openStarted() {
        if let flags = flagsBeforeStart {
            flagsBeforeStart = nil
            askingFlags = flags
            return
        }
        guard let id = started else { return }
        started = nil
        // The session is started; Today resumes it once the refusal is read.
        if let failure = failureBeforeStart {
            failureBeforeStart = nil
            error = failure
            return
        }
        // A stop says why in the alert; the advice of a yes is in the sheet.
        if painOutcome?.trainingStopped == true { return }
        painOutcome = nil
        open = OpenSession(id: id)
    }

    private func start(restDay: Bool) {
        do {
            let logger = try LoggerModel.startSession(db: model.db, isRestDay: restDay)
            if restDay {
                try logger.complete()
                Task { await model.syncNow() }
            } else {
                open = OpenSession(id: logger.session.id)
            }
        } catch {
            self.error = error.localizedDescription
        }
    }
}

struct OpenSession: Identifiable, Hashable {
    let id: String
}

/// XP and the streak as the athlete kept them. A streak of nothing is not
/// shown: the app never points at what was missed (ADR 0003).
struct ProgressTiles: View {
    let progress: AthleteProgress

    var body: some View {
        HStack(spacing: 10) {
            tile(icon: "star.fill", tint: Palette.gold, value: progress.xpTotal, unit: "XP", valueTint: Palette.gold) {
                Text("Total")
            }
            if progress.currentDays > 0 {
                tile(icon: "flame.fill", tint: Palette.ember, value: progress.currentDays,
                     unit: progress.currentDays == 1 ? "day" : "days", valueTint: Palette.text) {
                    Text("Rest days count")
                }
                .accessibilityElement(children: .ignore)
                .accessibilityLabel(Text("\(progress.currentDays)-day streak"))
                .accessibilityHint(Text("Rest days count"))
            }
        }
    }

    private func tile(icon: String, tint: Color, value: Int, unit: LocalizedStringKey, valueTint: Color,
                      @ViewBuilder caption: () -> Text) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            HStack(alignment: .firstTextBaseline, spacing: 6) {
                Image(systemName: icon).font(.system(size: 15, weight: .semibold)).foregroundStyle(tint)
                Text(value, format: .number).font(.metric).monospacedDigit().foregroundStyle(valueTint)
                Text(unit).font(Typeface.text(14, .semibold, relativeTo: .subheadline))
                    .foregroundStyle(Palette.textSecondary)
            }
            caption().font(.fine).foregroundStyle(Palette.textSecondary)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Palette.surface, in: .rect(cornerRadius: 16))
        .accessibilityElement(children: .combine)
    }
}

/// A day in a list: the date's number over its weekday.
struct DateBlock: View {
    let date: Date

    var body: some View {
        VStack(spacing: 0) {
            Text(date, format: .dateTime.day())
                .font(.metricSmall)
                .monospacedDigit()
                .foregroundStyle(Palette.text)
            Text(date, format: .dateTime.weekday(.short))
                .font(.capsSmall)
                .tracking(1)
                .textCase(.uppercase)
                .foregroundStyle(Palette.textSecondary)
        }
        .frame(width: 40)
    }
}

struct SessionRow: View {
    let session: Session

    var body: some View {
        HStack(spacing: 14) {
            DateBlock(date: session.startedAt)
            VStack(alignment: .leading, spacing: 2) {
                Text(title).font(.rowTitle).foregroundStyle(Palette.text)
                Text(detail).font(.meta).foregroundStyle(Palette.textSecondary)
            }
            Spacer(minLength: 8)
            status
        }
        .padding(.vertical, 10)
        .frame(minHeight: 64)
        .contentShape(.rect)
        .accessibilityElement(children: .combine)
    }

    @ViewBuilder
    private var status: some View {
        if session.isRestDay {
            Image(systemName: "moon").foregroundStyle(Palette.textSecondary).accessibilityLabel("Rest day")
        } else {
            switch session.status {
            case "completed":
                Image(systemName: "checkmark").font(.body.weight(.semibold)).foregroundStyle(Palette.gold)
                    .accessibilityLabel("Completed")
            case "abandoned":
                Image(systemName: "xmark").foregroundStyle(Palette.textTertiary).accessibilityLabel("Abandoned")
            default:
                Circle().fill(Palette.ember).frame(width: 10, height: 10).accessibilityLabel("In progress")
            }
        }
    }

    private var title: String {
        if !session.title.isEmpty { return session.title }
        return session.isRestDay ? String(localized: "Rest day") : String(localized: "Session")
    }

    private var detail: String {
        // Planned rest keeps the streak (ADR 0003); say so where it is logged.
        if session.isRestDay { return String(localized: "Counts towards the streak") }
        let time = session.startedAt.formatted(date: .omitted, time: .shortened)
        switch session.status {
        case "completed": return String(localized: "\(time) · completed")
        case "abandoned": return String(localized: "\(time) · abandoned")
        default: return String(localized: "\(time) · in progress")
        }
    }
}

/// How many changes are waiting to sync; nothing when everything is sent.
struct SyncBadge: View {
    @Environment(AppModel.self) private var model
    let pending: Int

    var body: some View {
        if model.isSyncing {
            ProgressView().accessibilityLabel("Syncing")
        } else if pending > 0 {
            Label {
                Text("\(pending) to sync")
            } icon: {
                Image(systemName: model.isOffline ? "icloud.slash" : "icloud.and.arrow.up")
            }
            .labelStyle(.titleAndIcon)
            .font(.fine)
            .foregroundStyle(Palette.textSecondary)
        }
    }
}
