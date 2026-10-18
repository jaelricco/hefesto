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

    /// "map | Contrast failed | <description> | button 'Fit' id=… at (x, y, w, h)", on one line.
    private static func describe(_ issue: XCUIAccessibilityAuditIssue, on screen: String) -> String {
        var element = "no element"
        if let e = issue.element {
            let label = e.label.isEmpty ? "" : " '\(e.label)'"
            let id = e.identifier.isEmpty ? "" : " id=\(e.identifier)"
            let f = e.frame
            element = "\(e.elementType)\(label)\(id) at (\(Int(f.minX)), \(Int(f.minY)), \(Int(f.width)), \(Int(f.height)))"
        }
        return [screen, issue.compactDescription, issue.detailedDescription, element]
            .map { $0.replacingOccurrences(of: "\n", with: " ") }
            .joined(separator: " | ")
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
                    // One failure per issue, naming the screen and the element, so
                    // CI can list them (the default failure names only the type).
                    try app.performAccessibilityAudit { issue in
                        XCTFail(Self.describe(issue, on: screen))
                        return true
                    }
                } catch {
                    XCTFail("\(screen): \(error)")
                }
            }
            app.terminate()
        }
    }
}
