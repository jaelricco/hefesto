import Foundation
import HefestoPlan
import HefestoSync

/// The onboarding's questions and answer labels in the app's languages
/// (onboarding.md §3). The knowledge base's own texts, like red flags,
/// region names and screening questions, come with the catalogue instead
/// and are shown verbatim.
enum OnboardingText {
    static func equipment(_ key: String) -> String {
        switch key {
        case "floor": String(localized: "Floor")
        case "wall": String(localized: "Wall")
        case "pull_up_bar": String(localized: "Pull-up bar")
        case "low_bar": String(localized: "Low bar")
        case "parallel_bars": String(localized: "Parallel bars")
        case "parallettes": String(localized: "Parallettes")
        case "rings": String(localized: "Rings")
        case "resistance_bands": String(localized: "Resistance bands")
        case "weight_vest": String(localized: "Weight vest")
        case "dumbbells_or_plates": String(localized: "Dumbbells or plates")
        case "dip_belt": String(localized: "Dip belt")
        case "box_or_bench": String(localized: "Box or bench")
        case "gym": String(localized: "Gym")
        case "outdoor_park": String(localized: "Outdoor park")
        default: key
        }
    }

    /// What a choice of equipment brings along, where it brings more.
    static func equipmentDetail(_ key: String) -> String? {
        switch key {
        case "gym": String(localized: "Pull-up bar, dip station, weights and a bench")
        case "outdoor_park": String(localized: "Pull-up bar, low bar and parallel bars")
        default: nil
        }
    }

    /// The equipment to choose from; floor and wall are always there.
    static let equipmentChoices = [
        "pull_up_bar", "low_bar", "parallel_bars", "parallettes", "rings", "resistance_bands", "weight_vest",
        "dumbbells_or_plates", "dip_belt", "box_or_bench", "gym", "outdoor_park",
    ]

    static let trainingLevels: [(key: String, label: String)] = [
        ("sedentary", String(localized: "Hardly active")),
        ("recreational", String(localized: "Active now and then")),
        ("trained", String(localized: "Training regularly")),
        ("highly_trained", String(localized: "Training hard, at a high level")),
    ]

    static let trainingAges: [(key: String, label: String)] = [
        ("lt_6_months", String(localized: "Less than 6 months")),
        ("6_to_12_months", String(localized: "6 to 12 months")),
        ("1_to_4_years", String(localized: "1 to 4 years")),
        ("gt_4_years", String(localized: "More than 4 years")),
    ]

    static let lastRegular: [(key: String, label: String)] = [
        ("current_or_lt_3_weeks", String(localized: "I train now, or did less than 3 weeks ago")),
        ("3_to_6_weeks", String(localized: "3 to 6 weeks ago")),
        ("7_to_16_weeks", String(localized: "7 to 16 weeks ago")),
        ("17_to_26_weeks", String(localized: "17 to 26 weeks ago")),
        ("gt_26_weeks", String(localized: "More than 26 weeks ago")),
        ("never", String(localized: "Never")),
    ]

    static let dataConfidence: [(key: String, label: String)] = [
        ("estimated", String(localized: "Estimated")),
        ("counted_last_4_weeks", String(localized: "Counted in the last 4 weeks")),
        ("filmed", String(localized: "Filmed")),
    ]

    /// The question of a class question, by its key; a question the app
    /// does not know yet is asked by its exercise.
    static func question(_ q: OnboardingCatalogue.ClassQuestion) -> String {
        switch q.key {
        case "push_up_class": String(localized: "How many clean push-ups can you do in one go?")
        case "pull_up_class": String(localized: "How many clean pull-ups? Chin over the bar, arms straight at the bottom.")
        case "dead_hang_class": String(localized: "How long can you hang from the bar?")
        case "row_class": String(localized: "How many rows on a low bar, body inclined?")
        case "dip_class": String(localized: "How many clean dips? Upper arms at least parallel to the floor.")
        case "support_hold_class": String(localized: "How long do you hold yourself up on straight arms?")
        case "hollow_hold_class": String(localized: "How long do you hold the hollow body?")
        default:
            q.holds
                ? String(localized: "How long do you hold it: \(q.exerciseName)?")
                : String(localized: "How many clean repetitions: \(q.exerciseName)?")
        }
    }

    /// A class as the athlete reads it: "4–7", "under 10 s", "more than 30".
    static func answerClass(_ c: OnboardingCatalogue.AnswerClass, holds: Bool) -> String {
        let lo = Int(c.lo)
        guard let hi = c.hi.map({ Int($0) }) else {
            return holds ? String(localized: "more than \(lo) s") : String(localized: "more than \(lo)")
        }
        if holds { return lo == 0 ? String(localized: "under \(hi) s") : String(localized: "\(lo)–\(hi) s") }
        return lo == hi ? "\(lo)" : String(localized: "\(lo)–\(hi)")
    }

    static func mobility(_ check: String) -> String {
        switch check {
        case "wrist":
            String(localized: "Can you put your palms flat on the floor and, with straight arms, shift your weight forward without pain?")
        case "shoulder":
            String(localized: "Do your straight arms reach up beside your ears without arching your lower back?")
        case "ankle":
            String(localized: "In a lunge with your toes a hand's width from a wall, does your knee touch the wall without the heel lifting?")
        case "compression":
            String(localized: "Sitting with straight legs, can you reach your toes?")
        default: check
        }
    }

    static let mobilityAnswers: [(key: String, label: String)] = [
        ("yes", String(localized: "Yes")), ("partly", String(localized: "Partly")), ("no", String(localized: "No")),
    ]

    static let onsets: [(key: String, label: String)] = [
        ("sudden", String(localized: "Suddenly")), ("gradual", String(localized: "Gradually")),
    ]

    static func duration(_ weeks: Double) -> String {
        switch weeks {
        case ..<2: String(localized: "Less than 2 weeks")
        case ..<4: String(localized: "2 to 4 weeks")
        case ...12: String(localized: "4 to 12 weeks")
        default: String(localized: "More than 12 weeks")
        }
    }

    static let suspicions: [(key: String, label: String)] = [
        ("yes", String(localized: "Yes")), ("no", String(localized: "No")), ("unsure", String(localized: "Not sure")),
    ]

    static let assessments: [(key: String, label: String)] = [
        ("no", String(localized: "No")),
        ("yes_overuse_or_tendon", String(localized: "Yes: overuse or a tendon")),
        ("yes_tear_or_suspected_tear", String(localized: "Yes: a tear, or the suspicion of one")),
        ("yes_other", String(localized: "Yes: something else")),
        ("in_progress", String(localized: "It is being looked into")),
    ]

    static let restrictions: [(key: String, label: String)] = [
        ("support_straight_arm", String(localized: "Supporting on straight arms")),
        ("hang_pull", String(localized: "Hanging and pulling")),
        ("overhead", String(localized: "Arms overhead")),
        ("wrist_extension_loaded", String(localized: "Weight on a bent-back wrist")),
        ("supination_loaded", String(localized: "Palms up under load")),
        ("spine_extension", String(localized: "Arching the back")),
    ]

    /// Why the answers did not go through, in the athlete's terms.
    static func failure(_ error: any Error) -> String {
        if let e = error as? OnboardingError {
            return switch e {
            case .alreadyOnboarded:
                String(localized: "You have answered these questions before. Your plan is on its way.")
            case .invalid:
                String(localized: "Some answers could not be taken. Check them and try again.")
            case .unavailable:
                String(localized: "The planner is not available right now. Try again later.")
            case .refused:
                String(localized: "The answers could not be sent. Try again later.")
            }
        }
        if let e = error as? OnboardingSubmitError {
            return switch e {
            case .noServer: String(localized: "The demo sends no answers.")
            case .incomplete: String(localized: "A question is still open.")
            }
        }
        return String(localized: "Hefesto could not reach the server. Your answers are kept; try again.")
    }
}
