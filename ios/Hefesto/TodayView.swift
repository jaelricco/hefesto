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
    @State private var path: [PlanRoute] = []
    /// The planned session whose check-in is open.
    @State private var checkingIn: PlanSession?
    /// The log session a start returned, opened once the check-in is gone.
    @State private var started: String?
    @State private var onboarding = false

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
                    plan.padding(.horizontal, 16).padding(.top, 16)
                    actions.padding(.horizontal, 16).padding(.top, 16)
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
            .fullScreenCover(item: $open) { LoggerView(sessionId: $0.id) }
            .fullScreenCover(isPresented: $onboarding) { OnboardingView() }
            .sheet(item: $checkingIn, onDismiss: openStarted) { session in
                CheckInSheet(
                    session: session,
                    onStart: { sleep, fatigue in await start(session, sleepHours: sleep, fatigue: fatigue) },
                    onStartOffline: { startOffline(session) },
                    initiallyOffline: model.demo?.screen == .offlinestart)
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
                if let id = demo.opensSessionId { open = OpenSession(id: id) }
                if let today = model.plan.todaySession {
                    if demo.screen == .checkin || demo.screen == .offlinestart { checkingIn = today }
                    if demo.screen == .plansession { path = [PlanRoute(sessionId: today.id)] }
                }
            }
            .task {
                do { for try await s in model.db.observeRecentSessions() { sessions = s } } catch {}
            }
            .task {
                do { for try await n in model.db.observePendingOpCount() { pending = n } } catch {}
            }
            .task {
                do { for try await p in model.db.observeProgress() { progress = p } } catch {}
            }
        }
    }

    private var header: some View {
        HStack(spacing: 8) {
            Text(Date.now, format: .dateTime.weekday(.wide).day().month(.wide))
                .font(.capsLabel)
                .tracking(1.2)
                .textCase(.uppercase)
                .foregroundStyle(Palette.textSecondary)
            Spacer()
            SyncBadge(pending: pending)
            Menu {
                Button("Sync now", systemImage: "arrow.triangle.2.circlepath") {
                    Task { await model.syncNow() }
                }
                Button("Sign out", systemImage: "rectangle.portrait.and.arrow.right", role: .destructive) {
                    Task { await model.signOut() }
                }
            } label: {
                Image(systemName: "person.crop.circle")
                    .font(.system(size: 24))
                    .foregroundStyle(Palette.textSecondary)
                    .frame(width: 44, height: 44)
            }
            .accessibilityLabel("Account")
        }
        .padding(.leading, 20)
        .padding(.trailing, 12)
        .frame(minHeight: 44)
    }

    /// Today's planned session, a rest day of the plan, or a stop. Without a
    /// plan, the onboarding that brings one.
    @ViewBuilder
    private var plan: some View {
        if model.plan.status == .needsOnboarding {
            OnboardingCard { onboarding = true }
        } else if let week = model.plan.week {
            if week.stopped {
                PlanStoppedCard(week: week)
            } else {
                TodayPlanCard(
                    session: model.plan.todaySession, onStart: { checkingIn = $0 }, onContinue: resume,
                    onOpen: { path.append(PlanRoute(sessionId: $0.id)) })
            }
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

    /// Starts a planned session after its check-in. What is waiting to sync
    /// goes first, so the plan adapts to the latest log. Says why when the
    /// start did not happen; without a connection the athlete may start
    /// without the server. A demo never has one.
    private func start(_ session: PlanSession, sleepHours: Double?, fatigue: Int?) async -> StartFailure? {
        await model.syncNow()
        do {
            started = try await model.plan.start(session, sleepHours: sleepHours, fatigue: fatigue)
            checkingIn = nil
            return nil
        } catch PlanStartError.offline, PlanStartError.noServer {
            return .offline
        } catch {
            return .message(PlanStartMessage.text(error))
        }
    }

    /// Starts a planned session here, as planned and without the check-in
    /// (ADR 0025). The next sync takes it to the server.
    private func startOffline(_ session: PlanSession) -> String? {
        do {
            started = try model.plan.startOffline(session)
            checkingIn = nil
            return nil
        } catch {
            return PlanStartMessage.text(error)
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

    private func openStarted() {
        guard let id = started else { return }
        started = nil
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
