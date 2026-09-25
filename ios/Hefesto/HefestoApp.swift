import SwiftUI

@main
struct HefestoApp: App {
    @State private var model: AppModel?
    @State private var failure: String?
    @Environment(\.scenePhase) private var scenePhase

    var body: some Scene {
        WindowGroup {
            Group {
                if let model {
                    RootView().environment(model)
                } else if let failure {
                    ContentUnavailableView("Hefesto could not open its data", systemImage: "externaldrive.badge.xmark",
                                           description: Text(verbatim: failure))
                } else {
                    ProgressView()
                }
            }
            .task {
                guard model == nil else { return }
                do {
                    let m = try AppModel()
                    model = m
                    await m.start()
                } catch {
                    failure = error.localizedDescription
                }
            }
            .onChange(of: scenePhase) { _, phase in
                if phase == .active, let model { Task { await model.syncNow() } }
            }
        }
    }
}

struct RootView: View {
    @Environment(AppModel.self) private var model

    var body: some View {
        @Bindable var model = model
        Group {
            if model.isSignedIn {
                TodayView()
            } else {
                SignInView()
            }
        }
        .sheet(item: $model.celebration) { CelebrationView(celebration: $0) }
    }
}
