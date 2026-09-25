import HefestoLogger
import HefestoStore
import SwiftUI

/// Today: start a session or plan rest, and the sessions logged so far.
struct TodayView: View {
    @Environment(AppModel.self) private var model
    @State private var sessions: [Session] = []
    @State private var pending = 0
    @State private var open: OpenSession?
    @State private var error: String?

    var body: some View {
        NavigationStack {
            List {
                Section {
                    Button {
                        start(restDay: false)
                    } label: {
                        Label("Start a session", systemImage: "figure.strengthtraining.functional")
                            .font(.title3.weight(.semibold))
                            .frame(maxWidth: .infinity, minHeight: 56)
                    }
                    .buttonStyle(.borderedProminent)
                    .listRowInsets(EdgeInsets())
                    .listRowBackground(Color.clear)

                    // Planned rest counts as training kept (ADR 0003).
                    Button {
                        start(restDay: true)
                    } label: {
                        Label("Log a rest day", systemImage: "bed.double")
                            .frame(maxWidth: .infinity, minHeight: 44)
                    }
                    .buttonStyle(.bordered)
                    .listRowInsets(EdgeInsets())
                    .listRowBackground(Color.clear)
                }

                Section("Sessions") {
                    if sessions.isEmpty {
                        Text("Your sessions appear here.").foregroundStyle(.secondary)
                    }
                    ForEach(sessions) { s in
                        Button { open = OpenSession(id: s.id) } label: { SessionRow(session: s) }
                            .foregroundStyle(.primary)
                    }
                }
            }
            .navigationTitle("Today")
            .toolbar {
                ToolbarItem(placement: .topBarLeading) { SyncBadge(pending: pending) }
                ToolbarItem(placement: .topBarTrailing) {
                    Menu {
                        Button("Sync now", systemImage: "arrow.triangle.2.circlepath") {
                            Task { await model.syncNow() }
                        }
                        Button("Sign out", systemImage: "rectangle.portrait.and.arrow.right", role: .destructive) {
                            Task { await model.signOut() }
                        }
                    } label: {
                        Image(systemName: "person.crop.circle").accessibilityLabel("Account")
                    }
                }
            }
            .refreshable { await model.syncNow() }
            .fullScreenCover(item: $open) { LoggerView(sessionId: $0.id) }
            .alert("Something went wrong", isPresented: .constant(error != nil)) {
                Button("OK") { error = nil }
            } message: {
                Text(verbatim: error ?? "")
            }
            .task {
                do { for try await s in model.db.observeRecentSessions() { sessions = s } } catch {}
            }
            .task {
                do { for try await n in model.db.observePendingOpCount() { pending = n } } catch {}
            }
        }
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

struct SessionRow: View {
    let session: Session

    var body: some View {
        HStack {
            VStack(alignment: .leading, spacing: 2) {
                Text(title).font(.headline)
                Text(session.startedAt, format: .dateTime.weekday(.wide).day().month().hour().minute())
                    .font(.subheadline).foregroundStyle(.secondary)
            }
            Spacer()
            switch session.status {
            case "completed":
                Image(systemName: "checkmark.circle.fill").foregroundStyle(.green).accessibilityLabel("Completed")
            case "abandoned":
                Image(systemName: "xmark.circle").foregroundStyle(.secondary).accessibilityLabel("Abandoned")
            default:
                Text("In progress").font(.caption).foregroundStyle(.orange)
            }
        }
        .accessibilityElement(children: .combine)
    }

    private var title: String {
        if !session.title.isEmpty { return session.title }
        return session.isRestDay ? String(localized: "Rest day") : String(localized: "Session")
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
            .font(.caption)
            .foregroundStyle(.secondary)
        }
    }
}
