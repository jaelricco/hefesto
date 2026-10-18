import Foundation
import HefestoAPI

// What the onboarding asks with (ADR 0023): the planner's catalogue in plain
// values, mapped from the API in one place, like the plan (ADR 0025). Names,
// questions and advice are the knowledge base's texts, shown verbatim.

/// The planner's skills, regions, red flags, screening questions and answer
/// classes, from `GET /v1/planner/catalogue`.
public struct OnboardingCatalogue: Sendable, Hashable {
    /// The educational disclaimer the API sends with the catalogue.
    public var disclaimer: String
    /// Every skill of the planner, in the knowledge base's order.
    public var skills: [Skill]
    /// The body-map regions, in the knowledge base's order.
    public var regions: [Region]
    /// The questions answered with a class, in the knowledge base's order.
    public var classQuestions: [ClassQuestion]
    /// The question on exertion symptoms. Every region asks it too; the
    /// answer is the same.
    public var exertion: Flag
    /// The further screening questions, in the order of their answers.
    public var screening: [Question]
    /// A user counts as a minor while the current year minus the birth year
    /// is at most this (PAR-D-23).
    public var minorAgeYears: Int

    /// The skills an athlete can set as a goal: those with a milestone.
    /// Foundations without one, like hanging, are the way, not a goal.
    public var goalSkills: [Skill] {
        skills.filter { $0.levels.contains(where: \.milestone) }
    }

    public func skill(_ slug: String) -> Skill? {
        skills.first { $0.slug == slug }
    }

    public func region(_ id: String) -> Region? {
        regions.first { $0.id == id }
    }

    public struct Skill: Sendable, Hashable, Identifiable {
        public var slug: String
        public var name: String
        public var foundation: Bool
        /// Easiest first.
        public var levels: [Level]
        /// The classes of the skill's stage question; empty when the
        /// onboarding asks none for it.
        public var stageClasses: [AnswerClass]
        /// Its levels are held for seconds rather than done for repetitions.
        public var holds: Bool
        public var id: String { slug }

        /// The level a new goal aims at: the first milestone.
        public var firstMilestone: Level? {
            levels.first(where: \.milestone) ?? levels.last
        }
    }

    public struct Level: Sendable, Hashable, Identifiable {
        public var slug: String
        /// The name of the exercise that proves the level.
        public var name: String
        public var milestone: Bool
        public var id: String { slug }
    }

    public struct ClassQuestion: Sendable, Hashable, Identifiable {
        /// The key the answer goes under, like `pull_up_class`.
        public var key: String
        public var exerciseName: String
        /// Asked in seconds rather than repetitions.
        public var holds: Bool
        public var classes: [AnswerClass]
        public var id: String { key }
    }

    public struct AnswerClass: Sendable, Hashable, Identifiable {
        public var key: String
        public var lo: Double
        /// Nil for an open class: more than `lo`.
        public var hi: Double?
        public var id: String { key }
    }

    public struct Region: Sendable, Hashable, Identifiable {
        public var id: String
        /// Where on the body, as a phrase: "innen am Ellbogen".
        public var name: String
        /// Every question of the region, those for minors included.
        public var flags: [Flag]
    }

    /// A warning-sign question.
    public struct Flag: Sendable, Hashable, Identifiable {
        public var id: String
        public var question: String
        /// What a yes means for training, and whom to see.
        public var advice: String
        /// N: see a doctor now; D: in the next days; A: an assessment is
        /// advisable.
        public var urgency: String
        public var minorOnly: Bool
        /// Asked after a yes.
        public var followup: Followup?
    }

    public struct Followup: Sendable, Hashable, Identifiable {
        public var id: String
        public var question: String
        public var advice: String
        public var urgency: String
    }

    public struct Question: Sendable, Hashable, Identifiable {
        public var id: String
        public var text: String
    }
}

extension OnboardingCatalogue {
    /// The one mapping from the API's catalogue.
    public init(_ c: Components.Schemas.PlannerCatalogue) {
        var names: [String: String] = [:], holds: [String: Bool] = [:]
        for e in c.exercises {
            names[e.slug] = e.name
            holds[e.slug] = e.measure.rawValue == "hold_seconds"
        }
        disclaimer = c.disclaimer
        skills = c.skills.map { s in
            Skill(
                slug: s.slug, name: s.name, foundation: s.foundation,
                levels: s.levels.map { Level(slug: $0.slug, name: names[$0.exercise] ?? $0.slug, milestone: $0.milestone) },
                stageClasses: s.stageClasses.map(AnswerClass.init),
                holds: s.levels.first.flatMap { holds[$0.exercise] } ?? false)
        }
        regions = c.regions.map { Region(id: $0.id, name: $0.name, flags: $0.redFlags.map(Flag.init)) }
        classQuestions = c.classQuestions.map { q in
            ClassQuestion(
                key: q.key, exerciseName: names[q.exercise] ?? q.exercise, holds: holds[q.exercise] ?? false,
                classes: q.classes.map(AnswerClass.init))
        }
        exertion = Flag(c.exertion)
        screening = c.screening.map { Question(id: $0.id, text: $0.question) }
        minorAgeYears = c.minorAgeYears
    }

    /// The catalogue in the API's JSON, as a demo keeps it.
    public init(json: Data) throws {
        self.init(try HefestoAPIConfiguration.decoder().decode(Components.Schemas.PlannerCatalogue.self, from: json))
    }
}

extension OnboardingCatalogue.AnswerClass {
    init(_ c: Components.Schemas.AnswerClass) {
        key = c.key
        lo = c.lo
        hi = c.open ? nil : c.hi
    }
}

extension OnboardingCatalogue.Flag {
    init(_ f: Components.Schemas.RedFlagQuestion) {
        id = f.id
        question = f.question
        advice = f.advice
        urgency = f.urgency.rawValue
        minorOnly = f.minorOnly
        followup = f.followup.map {
            OnboardingCatalogue.Followup(id: $0.id, question: $0.question, advice: $0.advice, urgency: $0.urgency.rawValue)
        }
    }
}

/// What the server made of the answers.
public struct OnboardingOutcome: Sendable, Hashable {
    public enum Status: String, Sendable {
        /// At most two clarifications are open; nothing is stored yet.
        case needsAnswers = "needs_answers"
        /// A safety answer stops training until a clearance (SAFE-02). The
        /// plan says why.
        case blocked
        /// The start state is stored, and the first plan is kept here.
        case complete
    }

    public var status: Status
    /// The clarifications to answer yes or no, then send again.
    public var questions: [OnboardingCatalogue.Question]
    /// How long the step to each dated goal usually takes, in the
    /// knowledge base's words.
    public var realism: [PlanNote]
    public var hints: [PlanNote]
    public var reasons: [PlanNote]
    public var disclaimer: String

    init(_ r: Components.Schemas.OnboardingResult) {
        // A status this version does not know reads by whether a plan came.
        status = Status(rawValue: r.status.rawValue) ?? (r.plan == nil ? .needsAnswers : .complete)
        questions = r.questions.map { OnboardingCatalogue.Question(id: $0.id, text: $0.text) }
        realism = r.realism.map { PlanNote($0.reason) }
        hints = r.hints.map(PlanNote.init)
        reasons = r.reasons.map(PlanNote.init)
        disclaimer = r.disclaimer
    }
}
