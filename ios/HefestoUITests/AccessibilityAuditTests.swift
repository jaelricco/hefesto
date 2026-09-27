import XCTest

/// Runs Apple's accessibility audit (VoiceOver labels, contrast, hit regions,
/// Dynamic Type, clipped text) on every main screen. The app runs on its
/// offline fixture (UITestFixture.swift), so no server is needed. The same
/// pass runs at the default text size, at the largest accessibility size, and
/// in German, where strings are longest.
@MainActor
final class AccessibilityAuditTests: XCTestCase {
    override func setUp() {
        continueAfterFailure = false
    }

    func testEveryScreen() throws {
        try walk(launch())
    }

    func testEveryScreenAtTheLargestTextSize() throws {
        try walk(launch(["-UIPreferredContentSizeCategoryName", "UICTContentSizeCategoryAccessibilityXXXL"]))
    }

    func testEveryScreenInGerman() throws {
        try walk(launch(["-AppleLanguages", "(de)", "-AppleLocale", "de_CH"]))
    }

    private func launch(_ arguments: [String] = []) -> XCUIApplication {
        let app = XCUIApplication()
        // The map, not the list, whatever an earlier run left in the defaults.
        app.launchArguments = ["-uiTestFixture", "-skills.showList", "NO"] + arguments
        app.launch()
        return app
    }

    /// Today, the logger, history and a session, stats, the map, a skill, and
    /// the skill list. Tabs are found by position so the walk works in every
    /// language.
    private func walk(_ app: XCUIApplication) throws {
        let tabs = app.tabBars.firstMatch
        XCTAssertTrue(tabs.waitForExistence(timeout: 10), "the fixture should open signed in")

        // Today, then a new session in the logger.
        try audit(app, "Today")
        app.buttons["start-session"].tap()
        XCTAssertTrue(app.buttons["log-set"].waitForExistence(timeout: 5))
        try audit(app, "Logger")
        app.buttons["close-logger"].tap()

        // History: sessions, one session, then exercises and one exercise's stats.
        tabs.buttons.element(boundBy: 1).tap()
        let session = app.buttons["history-session"].firstMatch
        XCTAssertTrue(session.waitForExistence(timeout: 5))
        try audit(app, "History")
        session.tap()
        try audit(app, "Session detail")
        app.navigationBars.buttons.element(boundBy: 0).tap()
        app.segmentedControls["history-mode"].buttons.element(boundBy: 1).tap()
        let exercise = app.buttons["history-exercise"].firstMatch
        XCTAssertTrue(exercise.waitForExistence(timeout: 5))
        exercise.tap()
        try audit(app, "Exercise stats")
        app.navigationBars.buttons.element(boundBy: 0).tap()

        // Skills: the map, a skill, then the list.
        tabs.buttons.element(boundBy: 2).tap()
        let node = app.buttons["skill-node-skill-a"]
        XCTAssertTrue(node.waitForExistence(timeout: 5))
        try audit(app, "Skill map")
        node.tap()
        try audit(app, "Skill detail")
        app.navigationBars.buttons.element(boundBy: 0).tap()
        app.buttons["skills-view-toggle"].tap()
        try audit(app, "Skill list")
    }

    private func audit(_ app: XCUIApplication, _ screen: String) throws {
        try XCTContext.runActivity(named: "Accessibility audit: \(screen)") { _ in
            try app.performAccessibilityAudit()
        }
    }
}
