import SwiftUI

// The app target: SwiftUI views only (ADR 0010). The screens land later in Phase 5.
@main
struct HefestoApp: App {
    var body: some Scene {
        WindowGroup { Text(verbatim: "Hefesto") }
    }
}
