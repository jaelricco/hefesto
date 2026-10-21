import XCTest

/// Runs Apple's accessibility audit (VoiceOver labels, contrast, hit regions,
/// Dynamic Type, clipped text) on every screen of the demo mode
/// (Hefesto/DemoMode.swift): seeded local data, no server, no account. Each
/// screen is opened directly by its launch argument, so the walk never
/// depends on how a screen is reached. It runs three times: English at the
/// default text size, and English and German at the largest accessibility
/// size, where German's long strings are rendered for real.
@MainActor
final class AccessibilityAuditTests: XCTestCase {
    /// DemoScreen's raw values, as scripts/ios-screenshots.sh walks them.
    private static let screens = [
        "today", "plansession", "checkin", "offlinestart", "planned", "target",
        "onboarding", "standing", "complaint", "body", "pain", "symptoms", "settings", "consent",
        "logger", "composer", "reserve", "band", "finish", "painafter", "morning", "map", "peek", "detail",
        "history", "session", "stats", "celebration", "signin",
    ]

    private static let largest = ["-UIPreferredContentSizeCategoryName", "UICTContentSizeCategoryAccessibilityXXXL"]

    override func setUp() {
        continueAfterFailure = true // report every screen, not only the first
    }

    func testEveryScreen() throws {
        try auditAll(run: "en", ["-AppleLanguages", "(en)", "-AppleLocale", "en_CH"], judgesSize: false)
    }

    func testEveryScreenAtTheLargestTextSize() throws {
        try auditAll(run: "xxl", ["-AppleLanguages", "(en)", "-AppleLocale", "en_CH"] + Self.largest, judgesSize: true)
    }

    func testEveryScreenInGermanAtTheLargestTextSize() throws {
        try auditAll(run: "de-xxl", ["-AppleLanguages", "(de)", "-AppleLocale", "de_CH"] + Self.largest, judgesSize: true)
    }

    /// "xxl | map | Contrast failed | text 'Fit' @20,72 362x24", on one line:
    /// short, so CI can list every finding in its annotations. Clipped text
    /// also gets the audit's own detail, which says what is cut.
    private static func describe(_ issue: XCUIAccessibilityAuditIssue, run: String, on screen: String) -> String {
        var element = "no element"
        if let e = issue.element {
            let label = e.label.isEmpty ? (e.identifier.isEmpty ? "" : " #\(e.identifier)") : " '\(e.label)'"
            let f = e.frame
            element = "\(kind(e.elementType))\(label) @\(Int(f.minX)),\(Int(f.minY)) \(Int(f.width))x\(Int(f.height))"
        }
        var line = "\(run) | \(screen) | \(issue.compactDescription) | \(element)"
        if issue.auditType == .textClipped {
            line += " | " + issue.detailedDescription.replacingOccurrences(of: "\n", with: " ")
        }
        return line
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

    /// Where a finding is not judged, and why; nil means it counts. Skipped
    /// findings are still printed ("audit skipped"), and CI lists them as
    /// notices, so an exemption never hides anything silently. The pattern
    /// is WWDC23's "Perform accessibility audits for your app".
    /// - Size predictions at the default size (Dynamic Type, clipped text):
    ///   the audit forecasts from the layout as it is, but grids drop columns
    ///   and rows wrap at large sizes. The two runs at the largest size
    ///   render that layout and judge it.
    /// - Dynamic Type or clipped text inside a bar: bars cap their text by
    ///   design and show it in the Large Content Viewer instead. That holds
    ///   for system items only: a custom view put in a bar must add
    ///   `.accessibilityShowsLargeContentViewer()`, or this skips a real gap.
    /// - Clipped text that runs past the edge of its scroll view: it is cut
    ///   by scrolling, not truncated, and scrolls into view.
    /// - Contrast of a disabled control: WCAG 1.4.3 exempts inactive
    ///   components, and a disabled Save must look disabled.
    /// - Contrast of an element that is not fully in view: not hittable,
    ///   partly scrolled out, beside a bar, where iOS fades the content that
    ///   scrolls under it, or partly under a view pinned over the scrolling
    ///   content (`.pinnedBar()`: the rest card, a bottom bar). The pixels
    ///   sampled there are another view's. What a pinned bar holds itself is
    ///   judged.
    /// - A finding that names no element: there is nothing to point at.
    /// - A known false positive, listed below with its evidence.
    private static func exemption(_ issue: XCUIAccessibilityAuditIssue, on screen: String, judgesSize: Bool,
                                  bars: [CGRect], pinned: [CGRect], scrolls: [CGRect]) -> String? {
        let sized = issue.auditType == .dynamicType || issue.auditType == .textClipped
        if sized && !judgesSize { return "a size prediction, judged at the largest size" }
        guard let e = issue.element else { return "names no element" }
        if knownFalsePositives.contains(where: {
            $0.screen == screen && issue.auditType == $0.type && e.label.hasPrefix($0.labelPrefix)
        }) {
            return "a known false positive"
        }
        let frame = e.frame
        let center = CGPoint(x: frame.midX, y: frame.midY)
        let partlyScrolledOut = scrolls.contains { $0.intersects(frame) && !$0.contains(frame) }
        if sized {
            if bars.contains(where: { $0.contains(center) }) { return "in a bar" }
            if issue.auditType == .textClipped && partlyScrolledOut { return "partly scrolled out" }
            return nil
        }
        guard issue.auditType == .contrast else { return nil }
        if !e.isEnabled { return "disabled" }
        if !e.isHittable { return "not hittable" }
        // The strip where iOS fades content scrolling under a bar.
        if bars.contains(where: { $0.insetBy(dx: 0, dy: -44).intersects(frame) }) { return "beside a bar" }
        if pinned.contains(where: { $0.intersects(frame) && !$0.contains(frame) }) { return "partly under a pinned bar" }
        if partlyScrolledOut { return "partly scrolled out" }
        return nil
    }

    /// Findings checked by hand and found not to be real, each matched by
    /// screen, kind and the start of the element's label, so nothing else
    /// slips through. Remove an entry when its screen changes.
    /// - The finish sheet's scale hint, clipped text at the largest size:
    ///   "may be clipped at larger Dynamic Type sizes", though there is no
    ///   larger size. It is whole (362x155 pt), wraps freely in a scroll
    ///   view, and moving it from y 601 to y 230 changed nothing; the same
    ///   font elsewhere passes. The sheet after a planned session
    ///   (`painafter`) holds the same hint.
    private static let knownFalsePositives: [(screen: String, type: XCUIAccessibilityAuditType, labelPrefix: String)] = [
        ("finish", .textClipped, "Optional. 1 "),
        ("painafter", .textClipped, "Optional. 1 "),
    ]

    private func auditAll(run: String, _ arguments: [String], judgesSize: Bool) throws {
        for screen in Self.screens {
            let app = XCUIApplication()
            app.launchArguments = ["-HefestoDemo", "YES", "-HefestoDemoScreen", screen] + arguments
            app.launch()
            XCTAssertTrue(app.wait(for: .runningForeground, timeout: 10), "\(screen) did not open")
            // Let sheets and pushes the demo opens settle before auditing.
            _ = app.windows.firstMatch.waitForExistence(timeout: 5)
            sleep(2)
            let bars = [app.navigationBars, app.tabBars, app.toolbars]
                .flatMap { $0.allElementsBoundByIndex.map(\.frame) }
            let pinned = app.descendants(matching: .any).matching(identifier: "pinned-bar")
                .allElementsBoundByIndex.map(\.frame)
            let scrolls = [app.scrollViews, app.collectionViews, app.tables]
                .flatMap { $0.allElementsBoundByIndex.map(\.frame) }
            XCTContext.runActivity(named: "Accessibility audit: \(run) \(screen)") { _ in
                do {
                    // One failure per issue, naming the screen and the element, so
                    // CI can list them (the default failure names only the type).
                    try app.performAccessibilityAudit { issue in
                        let line = Self.describe(issue, run: run, on: screen)
                        if let reason = Self.exemption(issue, on: screen, judgesSize: judgesSize, bars: bars,
                                                       pinned: pinned, scrolls: scrolls) {
                            print("audit skipped (\(reason)): \(line)")
                        } else {
                            XCTFail(line)
                        }
                        return true
                    }
                } catch {
                    XCTFail("\(run) | \(screen): \(error)")
                }
            }
            app.terminate()
        }
    }
}
