import XCTest

/// Runs Apple's accessibility audit (VoiceOver labels, contrast, hit regions,
/// Dynamic Type, clipped text) on every screen of the demo mode
/// (Hefesto/DemoMode.swift): seeded local data, no server, no account. Each
/// screen is opened directly by its launch argument, so the walk never
/// depends on how a screen is reached. The same pass runs in English at the
/// default text size, at the largest accessibility size, and in German,
/// where strings are longest.
@MainActor
final class AccessibilityAuditTests: XCTestCase {
    /// DemoScreen's raw values, as scripts/ios-screenshots.sh walks them.
    private static let screens = [
        "today", "logger", "composer", "reserve", "finish", "map", "peek", "detail",
        "history", "session", "stats", "celebration", "signin",
    ]

    override func setUp() {
        continueAfterFailure = true // report every screen, not only the first
    }

    func testEveryScreen() throws {
        try auditAll(["-AppleLanguages", "(en)", "-AppleLocale", "en_CH"])
    }

    func testEveryScreenAtTheLargestTextSize() throws {
        try auditAll(["-AppleLanguages", "(en)", "-AppleLocale", "en_CH",
                      "-UIPreferredContentSizeCategoryName", "UICTContentSizeCategoryAccessibilityXXXL"])
    }

    func testEveryScreenInGerman() throws {
        try auditAll(["-AppleLanguages", "(de)", "-AppleLocale", "de_CH"])
    }

    private func auditAll(_ arguments: [String]) throws {
        for screen in Self.screens {
            let app = XCUIApplication()
            app.launchArguments = ["-HefestoDemo", "YES", "-HefestoDemoScreen", screen] + arguments
            app.launch()
            XCTAssertTrue(app.wait(for: .runningForeground, timeout: 10), "\(screen) did not open")
            // Let sheets and pushes the demo opens settle before auditing.
            _ = app.windows.firstMatch.waitForExistence(timeout: 5)
            sleep(2)
            XCTContext.runActivity(named: "Accessibility audit: \(screen)") { _ in
                do {
                    try app.performAccessibilityAudit()
                } catch {
                    XCTFail("\(screen): \(error)")
                }
            }
            app.terminate()
        }
    }
}
