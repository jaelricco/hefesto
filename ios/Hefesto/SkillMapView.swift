import HefestoStore
import SwiftUI

/// The skill map as a constellation (brief §3.5, §10). Positions come from
/// content, never from a layout algorithm. Locked skills are dim, unlocked
/// ones glow, and a line lights up once, animated, the first time the map
/// shows an unlock. A tap shows a skill below the map; a second tap, or the
/// card, opens it.
struct SkillMapView: View {
    @Environment(AppModel.self) private var model
    @State private var map: SkillMapData?
    /// The skill shown in the card below the map.
    @State private var peek: String?
    /// The skill whose detail is open.
    @State private var opened: String?
    @State private var zoom: CGFloat = 1
    @State private var pinch: CGFloat = 1
    @State private var offset: CGSize = .zero
    @State private var drag: CGSize = .zero
    /// When the unlock animation started; nil when there is nothing new.
    @State private var revealStart: Date?

    static let revealSeconds = 1.6

    var body: some View {
        NavigationStack {
            Group {
                if let map, !placed(map).isEmpty {
                    constellation(map)
                } else if map == nil {
                    ProgressView()
                } else {
                    ContentUnavailableView(
                        "The map is on its way", systemImage: "sparkles",
                        description: Text("Skills appear after the first sync."))
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .background(Palette.sky.ignoresSafeArea())
            .safeAreaInset(edge: .top, spacing: 0) { header }
            .overlay(alignment: .bottom) {
                if let map, let id = peek, let skill = map.skills.first(where: { $0.id == id }) {
                    PeekCard(skill: skill, map: map) { opened = id }
                        .padding(.horizontal, 12)
                        .padding(.bottom, 12)
                        .transition(.move(edge: .bottom).combined(with: .opacity))
                }
            }
            .toolbar(.hidden, for: .navigationBar)
            .navigationDestination(item: $opened) { SkillDetailView(skillId: $0) }
        }
        .preferredColorScheme(.dark)
        .task {
            if let demo = model.demo {
                if demo.screen == .peek { peek = demo.skillId }
                if demo.screen == .detail { opened = demo.skillId }
            }
        }
        .task {
            do {
                for try await m in model.db.observeSkillMap() {
                    map = m
                    if revealStart == nil, !m.unseenUnlocks.isEmpty { revealStart = .now }
                }
            } catch {}
        }
        .task(id: revealStart) {
            // Once the lines have lit, the unlocks count as shown.
            guard revealStart != nil, let ids = map?.unseenUnlocks.map(\.levelId) else { return }
            try? await Task.sleep(for: .seconds(Self.revealSeconds + 0.4))
            try? model.db.markUnlocksSeen(ids)
            revealStart = nil
        }
    }

    private var header: some View {
        HStack(spacing: 8) {
            Text("Skills")
                .font(.displayTitle)
                .foregroundStyle(Palette.text)
                .accessibilityAddTraits(.isHeader)
            Spacer()
            ProgressBadge(style: .chips)
            Button {
                withAnimation(.snappy) { zoom = 1; offset = .zero }
            } label: {
                Image(systemName: "arrow.up.left.and.down.right.magnifyingglass")
                    .font(.system(size: 17, weight: .medium))
                    .foregroundStyle(Palette.text)
                    .frame(width: 44, height: 44)
                    .background(Palette.surface, in: .circle)
                    .overlay(Circle().strokeBorder(Palette.hairline))
            }
            .buttonStyle(.plain)
            .accessibilityLabel("Fit")
        }
        .padding(.leading, 20)
        .padding(.trailing, 12)
        .padding(.vertical, 4)
    }

    // MARK: drawing

    private func constellation(_ map: SkillMapData) -> some View {
        GeometryReader { geo in
            let layout = MapLayout(skills: placed(map), size: geo.size, zoom: zoom * pinch,
                                   offset: CGSize(width: offset.width + drag.width, height: offset.height + drag.height))
            ZStack {
                TimelineView(.animation(paused: revealStart == nil)) { timeline in
                    let reveal = revealStart.map { min(1, timeline.date.timeIntervalSince($0) / Self.revealSeconds) } ?? 1
                    Canvas { ctx, size in
                        drawStars(ctx, size: size)
                        drawLinks(ctx, map: map, layout: layout, reveal: reveal)
                        drawNodes(ctx, map: map, layout: layout, reveal: reveal)
                    }
                }
                .accessibilityHidden(true)

                // One real button per skill: the tap target, and what VoiceOver reads.
                ForEach(placed(map)) { s in
                    let p = layout.point(s.skill)
                    Button { select(s.id) } label: {
                        Color.clear.frame(width: 56, height: 56).contentShape(Circle())
                    }
                    .position(p)
                    .accessibilityLabel(Text(verbatim: s.skill.name))
                    .accessibilityValue(Text(StandingText.label(map.standing(of: s))))
                    .accessibilityHint(nextLevelHint(map, s))
                    .accessibilityAddTraits(peek == s.id ? .isSelected : [])
                }

                ForEach(placed(map)) { s in
                    let locked = map.standing(of: s) == .locked
                    Text(verbatim: s.skill.name)
                        .font(Typeface.condensed(13, .semibold, relativeTo: .caption))
                        .tracking(1)
                        .textCase(.uppercase)
                        .foregroundStyle(locked ? Palette.lockedLabel : Palette.text)
                        .fixedSize()
                        .position(x: layout.point(s.skill).x,
                                  y: layout.point(s.skill).y + layout.radius(s.skill) + (peek == s.id ? 22 : 14))
                        .allowsHitTesting(false)
                        .accessibilityHidden(true)
                }
            }
            .contentShape(Rectangle())
            .onTapGesture { withAnimation(.snappy) { peek = nil } }
            .gesture(
                SimultaneousGesture(
                    MagnifyGesture()
                        .onChanged { pinch = $0.magnification }
                        .onEnded { v in zoom = min(4, max(0.6, zoom * v.magnification)); pinch = 1 },
                    DragGesture()
                        .onChanged { drag = $0.translation }
                        .onEnded { v in
                            offset.width += v.translation.width
                            offset.height += v.translation.height
                            drag = .zero
                        }))
        }
    }

    private func select(_ id: String) {
        if peek == id {
            opened = id
        } else {
            withAnimation(.snappy) { peek = id }
        }
    }

    private func placed(_ map: SkillMapData) -> [SkillWithLevels] {
        map.skills.filter { $0.skill.x != nil && $0.skill.y != nil }
    }

    private func drawStars(_ ctx: GraphicsContext, size: CGSize) {
        // A fixed field, so the sky does not shimmer between frames.
        var rng = SeededRandom(seed: 7)
        for _ in 0..<140 {
            let p = CGPoint(x: rng.next() * size.width, y: rng.next() * size.height)
            let r = 0.4 + rng.next() * 0.9
            ctx.fill(Path(ellipseIn: CGRect(x: p.x - r, y: p.y - r, width: r * 2, height: r * 2)),
                     with: .color(Palette.text.opacity(0.15 + rng.next() * 0.25)))
        }
    }

    private func drawLinks(_ ctx: GraphicsContext, map: SkillMapData, layout: MapLayout, reveal: Double) {
        let byId = Dictionary(uniqueKeysWithValues: placed(map).map { ($0.id, $0.skill) })
        for link in map.skillLinks {
            guard let from = byId[link.from], let to = byId[link.to] else { continue }
            let a = layout.point(from), b = layout.point(to)
            var line = Path()
            line.move(to: a)
            line.addLine(to: b)
            ctx.stroke(line, with: .color(Palette.text.opacity(0.16)), style: StrokeStyle(lineWidth: 1, dash: [3, 4]))
            guard link.lit else { continue }
            // A newly lit line draws from the unlocked skill outwards.
            let t = link.newlyLit ? reveal : 1
            var lit = Path()
            lit.move(to: a)
            lit.addLine(to: CGPoint(x: a.x + (b.x - a.x) * t, y: a.y + (b.y - a.y) * t))
            var glow = ctx
            glow.addFilter(.blur(radius: 3))
            glow.stroke(lit, with: .color(Palette.gold.opacity(0.55)), lineWidth: 4)
            ctx.stroke(lit, with: .color(Palette.gold.opacity(0.95)), lineWidth: 1.5)
        }
    }

    private func drawNodes(_ ctx: GraphicsContext, map: SkillMapData, layout: MapLayout, reveal: Double) {
        let unseenSkills = Set(map.unseenUnlocks.compactMap { state in
            map.skills.first { $0.levels.contains { $0.id == state.levelId } }?.id
        })
        for s in placed(map) {
            let p = layout.point(s.skill)
            let r = layout.radius(s.skill)
            let rect = CGRect(x: p.x - r, y: p.y - r, width: r * 2, height: r * 2)
            let standing = map.standing(of: s)
            let pulse = unseenSkills.contains(s.id) ? 1 + 0.6 * sin(reveal * .pi) : 1
            switch standing {
            case .locked:
                ctx.fill(Path(ellipseIn: rect), with: .color(Palette.surfaceRaised))
                ctx.stroke(Path(ellipseIn: rect), with: .color(Palette.lockedStroke), lineWidth: 1)
            case .available:
                ctx.fill(Path(ellipseIn: rect), with: .color(Palette.surface))
                ctx.stroke(Path(ellipseIn: rect), with: .color(Palette.text), lineWidth: 1.5)
            case .progressing, .mastered:
                let color = standing == .mastered ? Palette.gold : Palette.ember
                var glow = ctx
                glow.addFilter(.blur(radius: r * 0.8 * pulse))
                glow.fill(Path(ellipseIn: rect.insetBy(dx: -r * 0.4, dy: -r * 0.4)), with: .color(color.opacity(0.6)))
                ctx.fill(Path(ellipseIn: rect), with: .color(color))
            }
            if standing == .progressing {
                // How far along the skill's levels the athlete is.
                let done = Double(s.levels.filter { map.state(of: $0.id) == "unlocked" }.count)
                var arc = Path()
                arc.addArc(center: p, radius: r + 4, startAngle: .degrees(-90),
                           endAngle: .degrees(-90 + 360 * done / Double(max(1, s.levels.count))), clockwise: false)
                ctx.stroke(arc, with: .color(Palette.gold), style: StrokeStyle(lineWidth: 2.5, lineCap: .round))
            }
            if peek == s.id {
                let ring = rect.insetBy(dx: -12, dy: -12)
                ctx.stroke(Path(ellipseIn: ring), with: .color(Palette.text.opacity(0.55)),
                           style: StrokeStyle(lineWidth: 1.2, dash: [2, 3]))
            }
        }
    }

    private func nextLevelHint(_ map: SkillMapData, _ s: SkillWithLevels) -> Text {
        guard let next = map.nextLevel(of: s) else { return Text("Every level unlocked") }
        return Text("Next: \(next.name)")
    }
}

/// The skill a tap picked, below the map: where the athlete stands and what
/// comes next. The whole card opens the skill.
struct PeekCard: View {
    let skill: SkillWithLevels
    let map: SkillMapData
    let onOpen: () -> Void

    var body: some View {
        let standing = map.standing(of: skill)
        let unlocked = skill.levels.filter { map.state(of: $0.id) == "unlocked" }.count
        Button(action: onOpen) {
            HStack(spacing: 14) {
                NodeGlyph(standing: standing, fraction: Double(unlocked) / Double(max(1, skill.levels.count)))
                VStack(alignment: .leading, spacing: 2) {
                    Text(verbatim: skill.skill.name)
                        .font(Typeface.condensed(20, .bold, relativeTo: .title3))
                        .tracking(0.6)
                        .textCase(.uppercase)
                        .foregroundStyle(Palette.text)
                    Text("\(StandingText.label(standing)) · \(unlocked) of \(skill.levels.count) levels")
                        .font(.meta)
                        .foregroundStyle(Palette.textSecondary)
                    if let next = map.nextLevel(of: skill) {
                        Text("Next: \(next.name)")
                            .font(.meta)
                            .foregroundStyle(Palette.text)
                            .lineLimit(1)
                    }
                }
                Spacer(minLength: 4)
                Image(systemName: "chevron.right")
                    .font(.body.weight(.semibold))
                    .foregroundStyle(Palette.ember)
                    .frame(width: 44, height: 44)
            }
            .padding(.leading, 16)
            .padding(.trailing, 8)
            .padding(.vertical, 14)
            .background(Palette.surface, in: .rect(cornerRadius: 18))
            .overlay(RoundedRectangle(cornerRadius: 18).strokeBorder(Palette.hairline))
            .contentShape(.rect(cornerRadius: 18))
        }
        .buttonStyle(.plain)
        .accessibilityHint("Opens the skill")
    }
}

/// A skill's node, small: for the card and the skill page.
struct NodeGlyph: View {
    let standing: SkillStanding
    /// Unlocked levels over all levels, for the arc.
    let fraction: Double
    var size: CGFloat = 36

    var body: some View {
        ZStack {
            switch standing {
            case .locked:
                Circle().fill(Palette.surfaceRaised).overlay(Circle().strokeBorder(Palette.lockedStroke))
                    .padding(size * 0.22)
            case .available:
                Circle().fill(Palette.surface).overlay(Circle().strokeBorder(Palette.text, lineWidth: 1.5))
                    .padding(size * 0.22)
            case .progressing:
                Circle().fill(Palette.ember).padding(size * 0.22)
                Circle().trim(from: 0, to: fraction)
                    .stroke(Palette.gold, style: StrokeStyle(lineWidth: 2.5, lineCap: .round))
                    .rotationEffect(.degrees(-90))
                    .padding(size * 0.08)
            case .mastered:
                Circle().fill(Palette.gold).padding(size * 0.22)
                    .shadow(color: Palette.gold.opacity(0.6), radius: 6)
            }
        }
        .frame(width: size, height: size)
        .accessibilityHidden(true)
    }
}

/// Maps content coordinates onto the screen: fit the constellation with a
/// margin, then apply the athlete's zoom and pan.
struct MapLayout {
    let minX: Double, minY: Double, scale: Double, origin: CGPoint
    let zoom: CGFloat, offset: CGSize, center: CGPoint

    init(skills: [SkillWithLevels], size: CGSize, zoom: CGFloat, offset: CGSize) {
        let xs = skills.compactMap(\.skill.x), ys = skills.compactMap(\.skill.y)
        let minX = xs.min() ?? 0, maxX = xs.max() ?? 1, minY = ys.min() ?? 0, maxY = ys.max() ?? 1
        let margin: CGFloat = 60
        let w = max(1, maxX - minX), h = max(1, maxY - minY)
        let scale = min((size.width - margin * 2) / w, (size.height - margin * 2) / h)
        self.minX = minX; self.minY = minY; self.scale = min(scale, 3)
        origin = CGPoint(x: (size.width - w * self.scale) / 2, y: (size.height - h * self.scale) / 2)
        self.zoom = zoom; self.offset = offset
        center = CGPoint(x: size.width / 2, y: size.height / 2)
    }

    func point(_ s: Skill) -> CGPoint {
        let base = CGPoint(x: origin.x + ((s.x ?? 0) - minX) * scale, y: origin.y + ((s.y ?? 0) - minY) * scale)
        return CGPoint(x: center.x + (base.x - center.x) * zoom + offset.width,
                       y: center.y + (base.y - center.y) * zoom + offset.height)
    }

    func radius(_ s: Skill) -> CGFloat { (s.isMilestone ? 14 : 9) * min(max(zoom, 0.8), 1.6) }
}

/// A small deterministic generator for the star field.
struct SeededRandom {
    private var state: UInt64
    init(seed: UInt64) { state = seed }
    mutating func next() -> Double {
        state = state &* 6364136223846793005 &+ 1442695040888963407
        return Double(state >> 11) / Double(1 << 53)
    }
}

enum StandingText {
    static func label(_ s: SkillStanding) -> String {
        switch s {
        case .locked: String(localized: "Locked")
        case .available: String(localized: "Open to work on")
        case .progressing: String(localized: "In progress")
        case .mastered: String(localized: "Every level unlocked")
        }
    }
}

/// XP and the streak. It shows what the athlete kept, never what they missed.
struct ProgressBadge: View {
    enum Style { case chips, plain }
    var style: Style = .plain
    @Environment(AppModel.self) private var model
    @State private var progress: AthleteProgress?

    var body: some View {
        Group {
            if let progress {
                HStack(spacing: style == .chips ? 8 : 10) {
                    item(Text("\(progress.xpTotal) XP"), icon: "star.fill", iconTint: Palette.gold, textTint: Palette.gold)
                    if progress.currentDays > 0 {
                        item(Text("\(progress.currentDays)"), icon: "flame.fill", iconTint: Palette.ember,
                             textTint: style == .chips ? Palette.text : Palette.ember)
                            .accessibilityElement(children: .ignore)
                            .accessibilityLabel(Text("\(progress.currentDays)-day streak"))
                    }
                }
            }
        }
        .task {
            do { for try await p in model.db.observeProgress() { progress = p } } catch {}
        }
    }

    @ViewBuilder
    private func item(_ title: Text, icon: String, iconTint: Color, textTint: Color) -> some View {
        let label = HStack(spacing: 5) {
            Image(systemName: icon).font(.system(size: 12, weight: .semibold)).foregroundStyle(iconTint)
            title.foregroundStyle(textTint)
        }
        switch style {
        case .chips:
            label
                .font(Typeface.condensed(16, .semibold, relativeTo: .subheadline))
                .padding(.horizontal, 10)
                .frame(minHeight: 32)
                .background(Palette.surface, in: .capsule)
                .overlay(Capsule().strokeBorder(Palette.hairline))
        case .plain:
            label.font(Typeface.condensed(15, .semibold, relativeTo: .caption))
        }
    }
}
