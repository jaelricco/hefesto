import SwiftUI
import UIKit

// Design direction B, "Glut" (ADR 0020): one dark, warm world for the whole
// app, because the logger and the map are dark anyway (brief §10). Ember is
// for actions, gold only for what the athlete achieved. Barlow Condensed
// carries titles and numbers, Barlow the text. Every size scales with
// Dynamic Type.

/// The colours. Contrast is at least 4.5:1 for text on the surfaces it sits on.
enum Palette {
    static var background: Color { Color(hex: 0x15120F) }
    /// The map's sky, a shade darker than the rest.
    static var sky: Color { Color(hex: 0x120F0C) }
    static var surface: Color { Color(hex: 0x211D19) }
    static var surfaceRaised: Color { Color(hex: 0x2B2621) }
    static var hairline: Color { Color(hex: 0x3A332C) }

    static var text: Color { Color(hex: 0xF3EEE8) }
    static var textSecondary: Color { Color(hex: 0xA89F95) }
    static var textTertiary: Color { Color(hex: 0x948A80) }

    /// Actions: the one thing to press on a screen.
    static var ember: Color { Color(hex: 0xFF7A33) }
    /// Text and icons on ember.
    static var emberInk: Color { Color(hex: 0x1A0E06) }
    /// Achievement only: unlocks, XP, bests.
    static var gold: Color { Color(hex: 0xF2C14E) }
    /// Behind gold text, for chips and unlock cards.
    static var goldWash: Color { Color(hex: 0x2E2716) }

    static var lockedStroke: Color { Color(hex: 0x5E544A) }
    static var lockedLabel: Color { Color(hex: 0x8A8178) }
}

extension Color {
    init(hex: UInt32) {
        self.init(
            .sRGB,
            red: Double((hex >> 16) & 0xFF) / 255,
            green: Double((hex >> 8) & 0xFF) / 255,
            blue: Double(hex & 0xFF) / 255)
    }
}

/// The bundled typefaces (Fonts/, SIL Open Font License). A face that fails
/// to load falls back to the system font, never to nothing.
enum Typeface {
    enum Weight { case regular, medium, semibold, bold }

    static func text(_ size: CGFloat, _ weight: Weight = .regular, relativeTo style: Font.TextStyle) -> Font {
        let name = switch weight {
        case .regular: "Barlow-Regular"
        case .medium: "Barlow-Medium"
        case .semibold, .bold: "Barlow-SemiBold"
        }
        return .custom(name, size: size, relativeTo: style)
    }

    static func condensed(_ size: CGFloat, _ weight: Weight = .semibold, relativeTo style: Font.TextStyle) -> Font {
        let name = switch weight {
        case .regular, .medium: "BarlowCondensed-Medium"
        case .semibold: "BarlowCondensed-SemiBold"
        case .bold: "BarlowCondensed-Bold"
        }
        return .custom(name, size: size, relativeTo: style)
    }

    static func uiCondensed(_ size: CGFloat, bold: Bool, relativeTo style: UIFont.TextStyle) -> UIFont {
        let base = UIFont(name: bold ? "BarlowCondensed-Bold" : "BarlowCondensed-SemiBold", size: size)
            ?? .systemFont(ofSize: size, weight: bold ? .bold : .semibold)
        return UIFontMetrics(forTextStyle: style).scaledFont(for: base)
    }
}

extension Font {
    /// A screen's own title: "Heute", "Front Lever".
    static var displayLarge: Font { Typeface.condensed(48, .bold, relativeTo: .largeTitle) }
    static var displayTitle: Font { Typeface.condensed(34, .bold, relativeTo: .title) }
    static var navTitle: Font { Typeface.condensed(20, .semibold, relativeTo: .headline) }
    /// Section labels, set in capitals with tracking.
    static var capsLabel: Font { Typeface.condensed(14, .semibold, relativeTo: .footnote) }
    static var capsSmall: Font { Typeface.condensed(12, .semibold, relativeTo: .caption) }
    /// Reps, seconds, XP: the numbers the athlete reads at a glance.
    static var metric: Font { Typeface.condensed(36, .bold, relativeTo: .title) }
    static var metricSmall: Font { Typeface.condensed(26, .bold, relativeTo: .title3) }
    static var timerDigits: Font { Typeface.condensed(84, .bold, relativeTo: .largeTitle) }
    /// The label of a primary action.
    static var actionLabel: Font { Typeface.condensed(24, .semibold, relativeTo: .title3) }

    static var rowTitle: Font { Typeface.text(17, .semibold, relativeTo: .headline) }
    static var bodyText: Font { Typeface.text(17, relativeTo: .body) }
    static var bodyMedium: Font { Typeface.text(17, .medium, relativeTo: .body) }
    static var detailText: Font { Typeface.text(15, relativeTo: .callout) }
    static var meta: Font { Typeface.text(14, relativeTo: .subheadline) }
    static var fine: Font { Typeface.text(13, relativeTo: .footnote) }
}

// MARK: - Building blocks

/// A section label: capitals, tracked, secondary.
struct CapsLabel: View {
    let text: Text
    /// A section's label is a heading for VoiceOver; a label inside a card is not.
    var isHeader = true

    init(_ key: LocalizedStringKey, isHeader: Bool = true) { text = Text(key); self.isHeader = isHeader }
    init(verbatim: String, isHeader: Bool = true) { text = Text(verbatim: verbatim); self.isHeader = isHeader }

    var body: some View {
        text
            .font(.capsLabel)
            .tracking(1.2)
            .textCase(.uppercase)
            .foregroundStyle(Palette.textSecondary)
            .accessibilityAddTraits(isHeader ? .isHeader : [])
    }
}

/// A small rounded label, gold for achievement, neutral otherwise.
struct Chip: View {
    enum Tone { case gold, neutral, ember }
    let text: Text
    var tone: Tone = .neutral

    init(_ key: LocalizedStringKey, tone: Tone = .neutral) { text = Text(key); self.tone = tone }
    init(verbatim: String, tone: Tone = .neutral) { text = Text(verbatim: verbatim); self.tone = tone }

    var body: some View {
        text
            .font(.capsSmall)
            .tracking(0.8)
            .textCase(.uppercase)
            .foregroundStyle(tone == .gold ? Palette.gold : tone == .ember ? Palette.ember : Palette.textSecondary)
            .padding(.horizontal, 9)
            .padding(.vertical, 3)
            .background(tone == .gold ? Palette.goldWash : Palette.surfaceRaised, in: .capsule)
    }
}

extension View {
    /// A card: the warm surface with a 16 pt radius.
    func card(padding: CGFloat = 16) -> some View {
        self.padding(padding)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(Palette.surface, in: .rect(cornerRadius: 16))
    }

    /// A screen on the dark ground, for lists and forms as much as scroll views.
    func themedScreen() -> some View {
        self.scrollContentBackground(.hidden)
            .background(Palette.background.ignoresSafeArea())
    }

    /// A list or form row on the card surface.
    func themedRow() -> some View {
        self.listRowBackground(Palette.surface)
            .listRowSeparatorTint(Palette.hairline)
    }
}

/// The one action to take on a screen: ember, large, in reach of the thumb.
struct PrimaryButtonStyle: ButtonStyle {
    var height: CGFloat = 64
    @Environment(\.isEnabled) private var isEnabled

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.actionLabel)
            .labelStyle(.titleAndIcon)
            .foregroundStyle(isEnabled ? Palette.emberInk : Palette.textTertiary)
            .frame(maxWidth: .infinity, minHeight: height)
            .padding(.horizontal, 16)
            .background(isEnabled ? Palette.ember : Palette.surfaceRaised, in: .rect(cornerRadius: 16))
            .opacity(configuration.isPressed ? 0.8 : 1)
            .contentShape(.rect(cornerRadius: 16))
    }
}

/// A second action next to the primary one.
struct SecondaryButtonStyle: ButtonStyle {
    var height: CGFloat = 52

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.bodyMedium)
            .foregroundStyle(Palette.text)
            .frame(maxWidth: .infinity, minHeight: height)
            .padding(.horizontal, 16)
            .background(Palette.surface, in: .rect(cornerRadius: 16))
            .overlay(RoundedRectangle(cornerRadius: 16).strokeBorder(Palette.hairline))
            .opacity(configuration.isPressed ? 0.75 : 1)
            .contentShape(.rect(cornerRadius: 16))
    }
}

/// A round icon button, 48 pt: repeat a set, skip.
struct RoundIconButtonStyle: ButtonStyle {
    var size: CGFloat = 48

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: 20, weight: .semibold))
            .foregroundStyle(Palette.ember)
            .frame(width: size, height: size)
            .background(Palette.surfaceRaised, in: .circle)
            .opacity(configuration.isPressed ? 0.7 : 1)
            .contentShape(.circle)
    }
}

// MARK: - System bars

enum Appearance {
    /// Titles in Barlow Condensed. Before iOS 26 the tab bar also gets the
    /// warm ground; from iOS 26 it is Liquid Glass and keeps the system look.
    @MainActor
    static func apply() {
        let title = UIColor(Palette.text)
        let nav = UINavigationBar.appearance()
        nav.titleTextAttributes = [
            .font: Typeface.uiCondensed(20, bold: false, relativeTo: .headline), .foregroundColor: title,
        ]
        nav.largeTitleTextAttributes = [
            .font: Typeface.uiCondensed(40, bold: true, relativeTo: .largeTitle), .foregroundColor: title,
        ]

        if #unavailable(iOS 26) {
            let tabs = UITabBarAppearance()
            tabs.configureWithOpaqueBackground()
            tabs.backgroundColor = UIColor(Palette.background)
            tabs.shadowColor = UIColor(Palette.surfaceRaised)
            let item = UITabBarItemAppearance()
            item.normal.iconColor = UIColor(Palette.textTertiary)
            item.normal.titleTextAttributes = [.foregroundColor: UIColor(Palette.textTertiary)]
            item.selected.iconColor = UIColor(Palette.ember)
            item.selected.titleTextAttributes = [.foregroundColor: UIColor(Palette.ember)]
            tabs.stackedLayoutAppearance = item
            tabs.inlineLayoutAppearance = item
            tabs.compactInlineLayoutAppearance = item
            UITabBar.appearance().standardAppearance = tabs
            UITabBar.appearance().scrollEdgeAppearance = tabs
        }
    }
}
