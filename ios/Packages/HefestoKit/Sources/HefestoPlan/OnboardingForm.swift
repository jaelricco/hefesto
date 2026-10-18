import Foundation
import HefestoAPI

/// The onboarding's answers as the athlete gives them, block by block
/// (onboarding.md §3). Nil is not answered yet. Which questions apply, when
/// a page is complete and the request the server takes are decided here, so
/// the screens only show and fill it.
public struct OnboardingForm: Sendable, Hashable {
    // A — frame
    public var birthYear: Int?
    public var healthConsent: Bool?
    public var disclaimerAck = false

    // B — goals, the first is priority 1
    public var goals: [Goal] = []

    // C — availability
    public var sessionsPerWeek: Int?
    public var sessionMinutes: Int?

    // D — equipment; floor and wall are always there
    public var equipment: Set<String> = []

    // E — body weight and background
    public var bodyweightKg: Double?
    public var trainingLevel: String?
    public var trainingAge: String?
    public var lastRegular: String?
    /// Goal skill → the level held before the break.
    public var preBreak: [String: String] = [:]

    // F — where the athlete stands
    /// Class question key → class key or `unknown`.
    public var classes: [String: String] = [:]
    /// Goal skill → the stage held now.
    public var stages: [String: Stage] = [:]
    /// Self-check → `yes`, `partly` or `no`.
    public var mobility: [String: String] = [:]
    public var dataConfidence: String?

    // G — health
    public var exertion: Bool?
    /// Screening question id → yes. Asked only with consent.
    public var screening: [String: Bool] = [:]
    /// Region → current complaint and injury in the last 12 months.
    public var bodyMap: [String: Mark] = [:]
    /// Region → details, for every current region.
    public var complaints: [String: Complaint] = [:]
    /// Region → red flag id → yes, follow-ups included. The exertion flag is
    /// not asked again: its answer is `exertion`.
    public var redFlags: [String: [String: Bool]] = [:]

    /// Clarification id → yes, after a result with questions.
    public var clarifications: [String: Bool] = [:]

    public init() {}

    public struct Goal: Sendable, Hashable, Identifiable {
        public var skill: String
        public var targetLevel: String
        /// YYYY-MM-DD, optional; only for the realism check.
        public var targetDate: String?
        public var id: String { skill }

        public init(skill: String, targetLevel: String, targetDate: String? = nil) {
            self.skill = skill
            self.targetLevel = targetLevel
            self.targetDate = targetDate
        }
    }

    public struct Stage: Sendable, Hashable {
        /// A level slug, `none` or `unknown`.
        public var level: String
        /// How long or how often, for a level: a stage class or `unknown`.
        public var answerClass: String?

        public init(level: String, answerClass: String? = nil) {
            self.level = level
            self.answerClass = answerClass
        }

        /// Whether the athlete named a level, so its class is asked.
        public var namesLevel: Bool { level != Self.noLevel && level != OnboardingForm.unknown }

        /// The answer for a skill not started yet.
        public static let noLevel = "none"
    }

    public struct Mark: Sendable, Hashable {
        public var current = false
        public var past12Months = false

        public init(current: Bool = false, past12Months: Bool = false) {
            self.current = current
            self.past12Months = past12Months
        }

        public var isEmpty: Bool { !current && !past12Months }
    }

    public struct Complaint: Sendable, Hashable {
        /// NRS 0–10 in daily life.
        public var painDaily: Int?
        /// NRS 0–10 during or after training.
        public var painTraining: Int?
        public var onset: String?
        /// One of `OnboardingForm.durationWeeks`.
        public var durationWeeks: Double?
        public var suspectedSerious: String?
        public var assessment: String?
        public var restrictions: Set<String> = []

        public init() {}

        var isComplete: Bool {
            painDaily != nil && painTraining != nil && onset != nil && durationWeeks != nil
                && suspectedSerious != nil && assessment != nil
        }
    }
}

// MARK: - Vocabularies

extension OnboardingForm {
    /// The `answer` every class and stage question takes besides its classes.
    public static let unknown = "unknown"
    public static let sessionMinutesChoices = [20, 30, 45, 60, 75, 90]
    /// How long a complaint has lasted, as the weeks sent for each choice:
    /// under 2, 2–4, 4–12 and over 12 weeks (onboarding.md §3.7). Each is
    /// inside its class, so a referral after more than 4 weeks (PAR-D-19)
    /// follows the class.
    public static let durationWeeks: [Double] = [1, 3, 8, 13]
    public static let onsets = ["sudden", "gradual"]
    public static let suspicions = ["yes", "no", "unsure"]
    public static let assessments = ["no", "yes_overuse_or_tendon", "yes_tear_or_suspected_tear", "yes_other", "in_progress"]
    public static let mobilityAnswers = ["yes", "partly", "no"]
}

// MARK: - Which questions apply (onboarding.md §3.6)

/// The conditions of onboarding.md §3.6, with the knowledge base's skill
/// slugs. The catalogue does not carry them; a skill renamed there drops
/// its question here, which leaves the planner's conservative default.
enum OnboardingRules {
    /// Answer classes that are asked only after another answer.
    static let afterNoPullUp: Set<String> = ["dead_hang_class", "row_class"]
    /// Rows need a low bar or rings; a park has a low bar.
    static let rowEquipment: Set<String> = ["low_bar", "rings", "outdoor_park"]
    /// Goals that bring the hollow-body question.
    static let hollowGoals: Set<String> = ["planche", "front-lever", "l-sit"]
    /// The mobility self-checks and the goals that bring them; at most two
    /// per goal.
    static let mobility: [(check: String, goals: Set<String>)] = [
        ("wrist", ["handstand", "planche"]),
        ("shoulder", ["handstand"]),
        ("compression", ["l-sit"]),
    ]
    /// Breaks long enough to ask what came before them.
    static let longBreaks: Set<String> = ["7_to_16_weeks", "17_to_26_weeks", "gt_26_weeks"]
}

extension OnboardingForm {
    /// Whether the athlete counts as a minor (PAR-D-23), so the minor-only
    /// red flags are asked.
    public func isMinor(_ c: OnboardingCatalogue, currentYear: Int) -> Bool {
        guard let birthYear else { return false }
        return currentYear - birthYear <= c.minorAgeYears
    }

    /// The class questions that apply, in the catalogue's order.
    public func classQuestions(_ c: OnboardingCatalogue) -> [OnboardingCatalogue.ClassQuestion] {
        c.classQuestions.filter { asks($0.key) }
    }

    func asks(_ key: String) -> Bool {
        switch key {
        case "dead_hang_class":
            classes["pull_up_class"] == "0"
        case "row_class":
            classes["pull_up_class"] == "0" && !equipment.isDisjoint(with: OnboardingRules.rowEquipment)
        case "support_hold_class":
            classes["dip_class"] == "0"
        case "hollow_hold_class":
            goals.contains { OnboardingRules.hollowGoals.contains($0.skill) }
        default:
            true
        }
    }

    /// The goal skills whose stage is asked.
    public func stageSkills(_ c: OnboardingCatalogue) -> [OnboardingCatalogue.Skill] {
        goals.compactMap { c.skill($0.skill) }.filter { !$0.stageClasses.isEmpty }
    }

    /// The mobility self-checks the goals bring, in a fixed order.
    public var mobilityChecks: [String] {
        OnboardingRules.mobility.filter { rule in goals.contains { rule.goals.contains($0.skill) } }.map(\.check)
    }

    /// After a long break, what came before it is asked for each goal.
    public var asksPreBreak: Bool {
        lastRegular.map { OnboardingRules.longBreaks.contains($0) } ?? false
    }

    /// The regions with a current complaint, in the catalogue's order.
    public func currentRegions(_ c: OnboardingCatalogue) -> [OnboardingCatalogue.Region] {
        c.regions.filter { bodyMap[$0.id]?.current == true }
    }

    /// The red flags to ask for a region: without the exertion question,
    /// answered once for all, and the minor-only ones only of a minor.
    public func flags(
        for region: OnboardingCatalogue.Region, _ c: OnboardingCatalogue, currentYear: Int
    ) -> [OnboardingCatalogue.Flag] {
        let minor = isMinor(c, currentYear: currentYear)
        return region.flags.filter { $0.id != c.exertion.id && (minor || !$0.minorOnly) }
    }
}

// MARK: - Pages

/// One page of the onboarding: a block, or the details of a region with a
/// current complaint.
public enum OnboardingPage: Hashable, Sendable {
    case frame, goals, availability, equipment, background, performance, health
    case region(String)
    case finish
}

extension OnboardingForm {
    /// The pages in order; a region page for every current complaint.
    public func pages(_ c: OnboardingCatalogue) -> [OnboardingPage] {
        [.frame, .goals, .availability, .equipment, .background, .performance, .health]
            + currentRegions(c).map { .region($0.id) } + [.finish]
    }

    /// Whether every question of a page that applies is answered.
    public func isComplete(_ page: OnboardingPage, _ c: OnboardingCatalogue, currentYear: Int) -> Bool {
        switch page {
        case .frame:
            guard let birthYear else { return false }
            return (1920...currentYear).contains(birthYear) && healthConsent != nil && disclaimerAck
        case .goals:
            return (1...3).contains(goals.count)
                && goals.allSatisfy { g in c.skill(g.skill)?.levels.contains { $0.slug == g.targetLevel } == true }
        case .availability:
            return sessionsPerWeek != nil && sessionMinutes != nil
        case .equipment:
            return true
        case .background:
            guard let bodyweightKg else { return false }
            return (30...250).contains(bodyweightKg) && trainingLevel != nil && trainingAge != nil && lastRegular != nil
        case .performance:
            return classQuestions(c).allSatisfy { classes[$0.key] != nil }
                && stageSkills(c).allSatisfy { s in
                    guard let stage = stages[s.slug] else { return false }
                    return !stage.namesLevel || stage.answerClass != nil
                }
                && mobilityChecks.allSatisfy { mobility[$0] != nil }
                && dataConfidence != nil
        case .health:
            return exertion != nil && (healthConsent != true || c.screening.allSatisfy { screening[$0.id] != nil })
        case let .region(id):
            guard let region = c.region(id) else { return true }
            let answers = redFlags[id] ?? [:]
            return complaints[id]?.isComplete == true
                && flags(for: region, c, currentYear: currentYear).allSatisfy { flag in
                    guard let yes = answers[flag.id] else { return false }
                    return !yes || flag.followup.map { answers[$0.id] != nil } ?? true
                }
        case .finish:
            return true
        }
    }
}

// MARK: - The request

extension OnboardingForm {
    /// The answers as the API takes them, or nil while a page is incomplete.
    /// Only what applies goes out: an answer to a question that no longer
    /// applies, as after a changed goal, stays behind.
    public func request(_ c: OnboardingCatalogue, currentYear: Int) -> Components.Schemas.OnboardingAnswers? {
        guard pages(c).allSatisfy({ isComplete($0, c, currentYear: currentYear) }),
              let birthYear, let healthConsent, let sessionsPerWeek, let sessionMinutes, let bodyweightKg,
              let trainingLevel = trainingLevel.flatMap(Components.Schemas.TrainingLevel.init(rawValue:)),
              let trainingAge = trainingAge.flatMap(Components.Schemas.TrainingAge.init(rawValue:)),
              let lastRegular = lastRegular.flatMap(Components.Schemas.LastRegularTraining.init(rawValue:)),
              let dataConfidence = dataConfidence.flatMap(Components.Schemas.DataConfidence.init(rawValue:)),
              let exertion
        else { return nil }

        let askedClasses = Set(classQuestions(c).map(\.key))
        var answers = Components.Schemas.OnboardingAnswers(
            birthYear: birthYear,
            healthDataConsent: healthConsent,
            disclaimerAck: disclaimerAck,
            goals: goals.enumerated().map { i, g in
                Components.Schemas.TrainingGoalWrite(
                    skill: g.skill, targetLevel: g.targetLevel, priority: i + 1, targetDate: g.targetDate)
            },
            sessionsPerWeek: sessionsPerWeek,
            sessionMinutes: sessionMinutes,
            equipment: equipment.sorted().compactMap(Components.Schemas.Equipment.init(rawValue:)),
            bodyweightKg: bodyweightKg,
            trainingLevel: trainingLevel,
            calisthenicsTrainingAge: trainingAge,
            lastRegularTraining: lastRegular,
            classes: .init(additionalProperties: classes.filter { askedClasses.contains($0.key) }),
            dataConfidence: dataConfidence,
            exertionSymptoms: exertion,
            // Without consent the screening is not asked (onboarding.md §3.1).
            screening: healthConsent ? c.screening.map { screening[$0.id] ?? false } : [])

        if asksPreBreak {
            let goalSkills = Set(goals.map(\.skill))
            let before = preBreak.filter { goalSkills.contains($0.key) }
            if !before.isEmpty { answers.preBreakLevel = .init(additionalProperties: before) }
        }

        var stageAnswers: [String: Components.Schemas.StageAnswer] = [:]
        for skill in stageSkills(c) {
            guard let stage = stages[skill.slug] else { continue }
            stageAnswers[skill.slug] = Components.Schemas.StageAnswer(
                level: stage.level, _class: stage.namesLevel ? stage.answerClass : nil)
        }
        if !stageAnswers.isEmpty { answers.skillStages = .init(additionalProperties: stageAnswers) }

        let checks = mobilityChecks
        if !checks.isEmpty {
            func answer(_ check: String) -> Components.Schemas.MobilityAnswer? {
                guard checks.contains(check) else { return nil }
                return mobility[check].flatMap(Components.Schemas.MobilityAnswer.init(rawValue:))
            }
            answers.mobilityChecks = Components.Schemas.MobilityChecks(
                wrist: answer("wrist"), shoulder: answer("shoulder"), ankle: answer("ankle"),
                compression: answer("compression"))
        }

        // The body map; without consent past injuries are not asked.
        var map: [String: Components.Schemas.BodyMapEntry] = [:]
        for region in c.regions {
            guard let mark = bodyMap[region.id] else { continue }
            let past = healthConsent && mark.past12Months
            if mark.current || past {
                map[region.id] = Components.Schemas.BodyMapEntry(current: mark.current, past12Months: past)
            }
        }
        if !map.isEmpty { answers.bodyMap = .init(additionalProperties: map) }

        var details: [String: Components.Schemas.Complaint] = [:]
        var flagAnswers: [String: Components.Schemas.RedFlagAnswerMap] = [:]
        for region in currentRegions(c) {
            guard let complaint = complaints[region.id].flatMap({ $0.request() }) else { return nil }
            details[region.id] = complaint
            let given = redFlags[region.id] ?? [:]
            // Every region asks the exertion question too; it was answered
            // once for all.
            var asked: [String: Bool] = [c.exertion.id: exertion]
            for flag in flags(for: region, c, currentYear: currentYear) {
                let yes = given[flag.id] ?? false
                asked[flag.id] = yes
                if yes, let followup = flag.followup { asked[followup.id] = given[followup.id] ?? false }
            }
            if let followup = c.exertion.followup, exertion { asked[followup.id] = given[followup.id] ?? false }
            flagAnswers[region.id] = Components.Schemas.RedFlagAnswerMap(additionalProperties: asked)
        }
        if !details.isEmpty {
            answers.complaints = .init(additionalProperties: details)
            answers.redFlags = .init(additionalProperties: flagAnswers)
        }

        if !clarifications.isEmpty { answers.clarifications = .init(additionalProperties: clarifications) }
        return answers
    }
}

extension OnboardingForm.Complaint {
    func request() -> Components.Schemas.Complaint? {
        guard let painDaily, let painTraining, let durationWeeks,
              let onset = raw(onset, \Components.Schemas.Complaint.onset),
              let suspected = raw(suspectedSerious, \Components.Schemas.Complaint.suspectedSerious),
              let assessment = raw(assessment, \Components.Schemas.Complaint.professionalAssessment)
        else { return nil }
        let tags = restrictions.sorted().compactMap(Components.Schemas.Restriction.init(rawValue:))
        return Components.Schemas.Complaint(
            painDaily: Double(painDaily), painTraining: Double(painTraining), onset: onset,
            durationWeeks: durationWeeks, suspectedSerious: suspected, professionalAssessment: assessment,
            restrictions: tags.isEmpty ? nil : tags)
    }
}

/// A closed-vocabulary value of an API type by its raw value, typed by the
/// property it goes into.
private func raw<Root, Value: RawRepresentable>(
    _ value: String?, _: KeyPath<Root, Value>
) -> Value? where Value.RawValue == String {
    value.flatMap(Value.init(rawValue:))
}
