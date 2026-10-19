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
        "today", "plansession", "checkin", "offlinestart", "planned", "target",
        "onboarding", "standing", "complaint", "body", "pain", "symptoms", "settings",
        "logger", "composer", "reserve", "band", "finish", "map", "peek", "detail",
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

    /// "map | Contrast failed | text 'Fit' @20,72", on one line: short, so CI
    /// can list every finding in its annotations.
    private static func describe(_ issue: XCUIAccessibilityAuditIssue, on screen: String) -> String {
        var element = "no element"
        if let e = issue.element {
            let label = e.label.isEmpty ? (e.identifier.isEmpty ? "" : " #\(e.identifier)") : " '\(e.label)'"
            element = "\(kind(e.elementType))\(label) @\(Int(e.frame.minX)),\(Int(e.frame.minY))"
        }
        return "\(screen) | \(issue.compactDescription) | \(element)"
    }

    private static func kind(_ type: XCUIElement.ElementType) -> String {
        switch type {
        case .staticText: "text"
        case .button: "button"
        case .image: "image"
        case .textField: "field"
        case .switch: "switch"
        case .progressIndicator: "progress"
        case .cell: "cell"
        case .other: "other"
        default: "type \(type.rawValue)"
        }
    }

    /// Findings the app cannot act on, skipped for a stated reason (the
    /// pattern of WWDC23's "Perform accessibility audits for your app"):
    /// - Dynamic Type inside a navigation bar: bars cap their text by design
    ///   and show it in the Large Content Viewer instead. That holds for
    ///   system items only: a custom view put in a bar must add
    ///   `.accessibilityShowsLargeContentViewer()`, or this skips a real gap.
    /// - The contrast of a disabled control: WCAG 1.4.3 exempts inactive
    ///   components, and a disabled Save must look disabled.
    /// - The contrast of an element that is not hittable: scrolled under a
    ///   bar or off screen, so the pixels sampled at its frame are another
    ///   view's.
    private static func isExempt(_ issue: XCUIAccessibilityAuditIssue, bars: [CGRect]) -> Bool {
        guard let e = issue.element else { return false }
        switch issue.auditType {
        case .dynamicType:
            let center = CGPoint(x: e.frame.midX, y: e.frame.midY)
            return bars.contains { $0.contains(center) }
        case .contrast:
            return !e.isEnabled || !e.isHittable
        default:
            return false
        }
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
            let bars = app.navigationBars.allElementsBoundByIndex.map(\.frame)
            XCTContext.runActivity(named: "Accessibility audit: \(screen)") { _ in
                do {
                    // One failure per issue, naming the screen and the element, so
                    // CI can list them (the default failure names only the type).
                    try app.performAccessibilityAudit { issue in
                        if !Self.isExempt(issue, bars: bars) { XCTFail(Self.describe(issue, on: screen)) }
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
