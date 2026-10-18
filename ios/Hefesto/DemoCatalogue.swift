#if DEBUG
import Foundation

/// The planner's catalogue for the demo (ADR 0023), in the API's JSON:
/// every skill, level, class question, region and red flag of the knowledge
/// base, with its texts verbatim. Written from content/training on
/// 2026-10-18 (ruleset 0.1.0); a changed knowledge base changes only what the
/// demo shows.
enum DemoCatalogue {
    static var json: String {
        let regions = regionList.map { id, name, flagIds in
            #"{"id":"\#(id)","name":"\#(name)","red_flags":[\#(flagIds.compactMap { flags[$0] }.joined(separator: ","))]}"#
        }
        return #"""
        {"ruleset_version":"0.1.0","disclaimer":"\#(disclaimer)",
         "skills":\#(skills),"exercises":\#(exercises),
         "regions":[\#(regions.joined(separator: ","))],"class_questions":\#(classQuestions),
         "exertion":\#(flags["RF-10"] ?? ""),"screening":\#(screening),"minor_age_years":18}
        """#
    }

    static let disclaimer = "Hefesto passt dein Training an und ersetzt keine ärztliche oder physiotherapeutische Beratung. Die App stellt keine Diagnosen und gibt keine Behandlungsempfehlungen. Bei Warnzeichen empfiehlt sie eine Abklärung durch eine Fachperson. Zeitangaben sind Erfahrungswerte, keine Prognosen."

    static let skills = #"""
    [{"slug":"wrist-conditioning","name":"Handgelenk-Vorbereitung","foundation":true,"levels":[{"slug":"plank","exercise":"plank-hold","threshold":30,"milestone":false}],"stage_classes":[]}
    ,{"slug":"hang-foundation","name":"Hängen und Schulterblatt","foundation":true,"levels":[{"slug":"dead-hang","exercise":"dead-hang","threshold":30,"milestone":false},{"slug":"scapular-pull","exercise":"scapular-pull-up","threshold":8,"milestone":false},{"slug":"arch-hang","exercise":"arch-hang","threshold":8,"milestone":false}],"stage_classes":[]}
    ,{"slug":"pull-up","name":"Klimmzug","foundation":true,"levels":[{"slug":"strict-5","exercise":"pull-up","threshold":5,"milestone":true},{"slug":"strict-10","exercise":"pull-up","threshold":10,"milestone":false},{"slug":"l-sit","exercise":"pull-up-l-sit","threshold":5,"milestone":false}],"stage_classes":[{"key":"1","lo":1,"hi":1,"open":false},{"key":"2_3","lo":2,"hi":3,"open":false},{"key":"4_5","lo":4,"hi":5,"open":false},{"key":"gt_5","lo":5,"hi":null,"open":true}]}
    ,{"slug":"row","name":"Rudern","foundation":true,"levels":[{"slug":"vertical","exercise":"row-vertical","threshold":8,"milestone":false},{"slug":"incline","exercise":"row-incline","threshold":8,"milestone":false},{"slug":"horizontal","exercise":"row-horizontal","threshold":8,"milestone":false},{"slug":"wide","exercise":"row-wide","threshold":8,"milestone":false},{"slug":"archer","exercise":"row-archer","threshold":8,"milestone":false}],"stage_classes":[]}
    ,{"slug":"support-hold","name":"Stütz","foundation":true,"levels":[{"slug":"parallel-bars","exercise":"support-hold-pb","threshold":30,"milestone":false},{"slug":"rings","exercise":"support-hold-rings","threshold":30,"milestone":false},{"slug":"rings-turned-out","exercise":"support-hold-rings-rto","threshold":30,"milestone":false}],"stage_classes":[]}
    ,{"slug":"dip","name":"Dip","foundation":true,"levels":[{"slug":"parallel-bars","exercise":"dip-pb","threshold":8,"milestone":true},{"slug":"rings","exercise":"dip-rings","threshold":8,"milestone":false},{"slug":"rings-l","exercise":"dip-rings-l","threshold":5,"milestone":false}],"stage_classes":[{"key":"1","lo":1,"hi":1,"open":false},{"key":"2_3","lo":2,"hi":3,"open":false},{"key":"4_5","lo":4,"hi":5,"open":false},{"key":"gt_5","lo":5,"hi":null,"open":true}]}
    ,{"slug":"push-up","name":"Liegestütz","foundation":true,"levels":[{"slug":"wall","exercise":"push-up-wall","threshold":8,"milestone":false},{"slug":"incline","exercise":"push-up-incline","threshold":8,"milestone":false},{"slug":"full","exercise":"push-up","threshold":8,"milestone":true},{"slug":"diamond","exercise":"push-up-diamond","threshold":8,"milestone":false},{"slug":"rings","exercise":"push-up-rings","threshold":8,"milestone":false},{"slug":"pseudo-planche","exercise":"pseudo-planche-push-up","threshold":8,"milestone":false}],"stage_classes":[]}
    ,{"slug":"hollow-body","name":"Hollow Body","foundation":true,"levels":[{"slug":"tuck","exercise":"hollow-hold-tuck","threshold":30,"milestone":false},{"slug":"full","exercise":"hollow-hold","threshold":60,"milestone":false}],"stage_classes":[]}
    ,{"slug":"l-sit","name":"L-Sitz","foundation":false,"levels":[{"slug":"tuck","exercise":"l-sit-tuck","threshold":30,"milestone":false},{"slug":"one-leg","exercise":"l-sit-one-leg","threshold":5,"milestone":false},{"slug":"full","exercise":"l-sit","threshold":10,"milestone":true},{"slug":"straddle","exercise":"straddle-l-sit","threshold":10,"milestone":false},{"slug":"rings","exercise":"l-sit-rings-rto","threshold":10,"milestone":false}],"stage_classes":[{"key":"lt_4","lo":0,"hi":4,"open":false},{"key":"4_9","lo":4,"hi":9,"open":false},{"key":"10_19","lo":10,"hi":19,"open":false},{"key":"ge_20","lo":20,"hi":null,"open":true}]}
    ,{"slug":"handstand","name":"Handstand","foundation":false,"levels":[{"slug":"wall","exercise":"wall-handstand-hold","threshold":30,"milestone":false},{"slug":"chest-to-wall","exercise":"handstand-chest-to-wall","threshold":60,"milestone":false},{"slug":"free-10s","exercise":"handstand-freestanding","threshold":10,"milestone":true},{"slug":"free-30s","exercise":"handstand-freestanding","threshold":30,"milestone":false},{"slug":"free-60s","exercise":"handstand-freestanding","threshold":60,"milestone":false}],"stage_classes":[{"key":"lt_4","lo":0,"hi":4,"open":false},{"key":"4_9","lo":4,"hi":9,"open":false},{"key":"10_19","lo":10,"hi":19,"open":false},{"key":"ge_20","lo":20,"hi":null,"open":true}]}
    ,{"slug":"planche","name":"Planche","foundation":false,"levels":[{"slug":"lean","exercise":"planche-lean","threshold":30,"milestone":false},{"slug":"tuck","exercise":"planche-tuck","threshold":10,"milestone":true},{"slug":"advanced-tuck","exercise":"planche-advanced-tuck","threshold":10,"milestone":false},{"slug":"straddle","exercise":"planche-straddle","threshold":3,"milestone":true},{"slug":"half-lay","exercise":"planche-half-lay","threshold":3,"milestone":false},{"slug":"full","exercise":"planche","threshold":3,"milestone":true}],"stage_classes":[{"key":"lt_4","lo":0,"hi":4,"open":false},{"key":"4_9","lo":4,"hi":9,"open":false},{"key":"10_19","lo":10,"hi":19,"open":false},{"key":"ge_20","lo":20,"hi":null,"open":true}]}
    ,{"slug":"front-lever","name":"Front Lever","foundation":false,"levels":[{"slug":"tuck","exercise":"front-lever-tuck","threshold":10,"milestone":true},{"slug":"advanced-tuck","exercise":"front-lever-advanced-tuck","threshold":10,"milestone":false},{"slug":"straddle","exercise":"front-lever-straddle","threshold":3,"milestone":true},{"slug":"one-leg","exercise":"front-lever-one-leg","threshold":5,"milestone":false},{"slug":"full","exercise":"front-lever","threshold":3,"milestone":true}],"stage_classes":[{"key":"lt_4","lo":0,"hi":4,"open":false},{"key":"4_9","lo":4,"hi":9,"open":false},{"key":"10_19","lo":10,"hi":19,"open":false},{"key":"ge_20","lo":20,"hi":null,"open":true}]}
    ,{"slug":"muscle-up","name":"Muscle-up","foundation":false,"levels":[{"slug":"bar-kipping","exercise":"muscle-up-bar-kipping","threshold":1,"milestone":true},{"slug":"bar-strict","exercise":"muscle-up-bar-strict","threshold":1,"milestone":false},{"slug":"l-sit","exercise":"muscle-up-l-sit","threshold":1,"milestone":false}],"stage_classes":[{"key":"1","lo":1,"hi":1,"open":false},{"key":"2_3","lo":2,"hi":3,"open":false},{"key":"4_5","lo":4,"hi":5,"open":false},{"key":"gt_5","lo":5,"hi":null,"open":true}]}
    ]
    """#

    static let exercises = #"""
    [{"slug":"arch-hang","name":"Arch Hang","measure":"reps","requires":["pull_up_bar"],"skill":null,"draft":true}
    ,{"slug":"dead-hang","name":"Dead Hang","measure":"hold_seconds","requires":["pull_up_bar"],"skill":null,"draft":true}
    ,{"slug":"dip-pb","name":"Dip am Barren","measure":"reps","requires":["parallel_bars"],"skill":null,"draft":true}
    ,{"slug":"dip-rings","name":"Dip an Ringen","measure":"reps","requires":["rings"],"skill":null,"draft":true}
    ,{"slug":"dip-rings-l","name":"Dip an Ringen im L-Sitz","measure":"reps","requires":["rings"],"skill":null,"draft":true}
    ,{"slug":"front-lever","name":"Full Front Lever","measure":"hold_seconds","requires":["pull_up_bar"],"skill":null,"draft":true}
    ,{"slug":"front-lever-advanced-tuck","name":"Advanced Tuck Front Lever","measure":"hold_seconds","requires":["pull_up_bar"],"skill":null,"draft":true}
    ,{"slug":"front-lever-one-leg","name":"One-Leg Front Lever","measure":"hold_seconds","requires":["pull_up_bar"],"skill":null,"draft":true}
    ,{"slug":"front-lever-straddle","name":"Straddle Front Lever","measure":"hold_seconds","requires":["pull_up_bar"],"skill":null,"draft":true}
    ,{"slug":"front-lever-tuck","name":"Tuck Front Lever","measure":"hold_seconds","requires":["pull_up_bar"],"skill":null,"draft":true}
    ,{"slug":"handstand-chest-to-wall","name":"Handstand, Bauch zur Wand","measure":"hold_seconds","requires":["wall"],"skill":null,"draft":true}
    ,{"slug":"handstand-freestanding","name":"Freier Handstand","measure":"hold_seconds","requires":["floor"],"skill":null,"draft":true}
    ,{"slug":"hollow-hold","name":"Hollow Hold","measure":"hold_seconds","requires":["floor"],"skill":null,"draft":true}
    ,{"slug":"hollow-hold-tuck","name":"Hollow Hold, gehockt","measure":"hold_seconds","requires":["floor"],"skill":null,"draft":true}
    ,{"slug":"l-sit","name":"L-Sitz","measure":"hold_seconds","requires":["floor"],"skill":null,"draft":true}
    ,{"slug":"l-sit-one-leg","name":"L-Sitz, ein Bein gestreckt","measure":"hold_seconds","requires":["floor"],"skill":null,"draft":true}
    ,{"slug":"l-sit-rings-rto","name":"L-Sitz an Ringen, nach aussen gedreht","measure":"hold_seconds","requires":["rings"],"skill":null,"draft":true}
    ,{"slug":"l-sit-tuck","name":"L-Sitz, gehockt","measure":"hold_seconds","requires":["floor"],"skill":null,"draft":true}
    ,{"slug":"muscle-up-bar-kipping","name":"Muscle-up an der Stange mit Schwung","measure":"reps","requires":["pull_up_bar"],"skill":null,"draft":true}
    ,{"slug":"muscle-up-bar-strict","name":"Strikter Muscle-up an der Stange","measure":"reps","requires":["pull_up_bar"],"skill":null,"draft":true}
    ,{"slug":"muscle-up-l-sit","name":"Muscle-up im L-Sitz","measure":"reps","requires":["pull_up_bar"],"skill":null,"draft":true}
    ,{"slug":"planche","name":"Full Planche","measure":"hold_seconds","requires":["floor"],"skill":null,"draft":true}
    ,{"slug":"planche-advanced-tuck","name":"Advanced Tuck Planche","measure":"hold_seconds","requires":["floor"],"skill":null,"draft":true}
    ,{"slug":"planche-half-lay","name":"Half-Lay Planche","measure":"hold_seconds","requires":["floor"],"skill":null,"draft":true}
    ,{"slug":"planche-lean","name":"Planche Lean","measure":"hold_seconds","requires":["floor"],"skill":null,"draft":true}
    ,{"slug":"planche-straddle","name":"Straddle Planche","measure":"hold_seconds","requires":["floor"],"skill":null,"draft":true}
    ,{"slug":"planche-tuck","name":"Tuck Planche","measure":"hold_seconds","requires":["floor"],"skill":null,"draft":true}
    ,{"slug":"plank-hold","name":"Unterarmstütz mit gestreckten Armen (Plank)","measure":"hold_seconds","requires":["floor"],"skill":null,"draft":true}
    ,{"slug":"pseudo-planche-push-up","name":"Pseudo-Planche-Liegestütz","measure":"reps","requires":["floor"],"skill":null,"draft":true}
    ,{"slug":"pull-up","name":"Klimmzug","measure":"reps","requires":["pull_up_bar"],"skill":null,"draft":true}
    ,{"slug":"pull-up-l-sit","name":"L-Klimmzug","measure":"reps","requires":["pull_up_bar"],"skill":null,"draft":true}
    ,{"slug":"push-up","name":"Liegestütz","measure":"reps","requires":["floor"],"skill":null,"draft":true}
    ,{"slug":"push-up-diamond","name":"Diamant-Liegestütz","measure":"reps","requires":["floor"],"skill":null,"draft":true}
    ,{"slug":"push-up-incline","name":"Liegestütz mit erhöhten Händen","measure":"reps","requires":[],"skill":null,"draft":true}
    ,{"slug":"push-up-rings","name":"Liegestütz an Ringen","measure":"reps","requires":["rings"],"skill":null,"draft":true}
    ,{"slug":"push-up-wall","name":"Liegestütz an der Wand","measure":"reps","requires":["wall"],"skill":null,"draft":true}
    ,{"slug":"row-archer","name":"Archer-Rudern","measure":"reps","requires":["low_bar"],"skill":null,"draft":true}
    ,{"slug":"row-horizontal","name":"Rudern, waagrecht","measure":"reps","requires":["low_bar"],"skill":null,"draft":true}
    ,{"slug":"row-incline","name":"Rudern, schräg","measure":"reps","requires":["low_bar"],"skill":null,"draft":true}
    ,{"slug":"row-vertical","name":"Rudern, fast aufrecht","measure":"reps","requires":["low_bar"],"skill":null,"draft":true}
    ,{"slug":"row-wide","name":"Rudern, breit","measure":"reps","requires":["low_bar"],"skill":null,"draft":true}
    ,{"slug":"scapular-pull-up","name":"Scapula-Pull-up","measure":"reps","requires":["pull_up_bar"],"skill":null,"draft":true}
    ,{"slug":"straddle-l-sit","name":"Gegrätschter L-Sitz","measure":"hold_seconds","requires":["floor"],"skill":null,"draft":true}
    ,{"slug":"support-hold-pb","name":"Stütz am Barren","measure":"hold_seconds","requires":["parallel_bars"],"skill":null,"draft":true}
    ,{"slug":"support-hold-rings","name":"Stütz an Ringen","measure":"hold_seconds","requires":["rings"],"skill":null,"draft":true}
    ,{"slug":"support-hold-rings-rto","name":"Stütz an Ringen, nach aussen gedreht","measure":"hold_seconds","requires":["rings"],"skill":null,"draft":true}
    ,{"slug":"wall-handstand-hold","name":"Handstand an der Wand","measure":"hold_seconds","requires":["wall"],"skill":null,"draft":true}
    ]
    """#

    static let classQuestions = #"""
    [{"key":"push_up_class","exercise":"push-up","classes":[{"key":"0","lo":0,"hi":0,"open":false},{"key":"1_3","lo":1,"hi":3,"open":false},{"key":"4_7","lo":4,"hi":7,"open":false},{"key":"8_12","lo":8,"hi":12,"open":false},{"key":"13_20","lo":13,"hi":20,"open":false},{"key":"21_30","lo":21,"hi":30,"open":false},{"key":"gt_30","lo":30,"hi":null,"open":true}]}
    ,{"key":"pull_up_class","exercise":"pull-up","classes":[{"key":"0","lo":0,"hi":0,"open":false},{"key":"1_3","lo":1,"hi":3,"open":false},{"key":"4_7","lo":4,"hi":7,"open":false},{"key":"8_11","lo":8,"hi":11,"open":false},{"key":"12_15","lo":12,"hi":15,"open":false},{"key":"16_20","lo":16,"hi":20,"open":false},{"key":"gt_20","lo":20,"hi":null,"open":true}]}
    ,{"key":"dead_hang_class","exercise":"dead-hang","classes":[{"key":"lt_10","lo":0,"hi":10,"open":false},{"key":"10_30","lo":10,"hi":30,"open":false},{"key":"30_60","lo":30,"hi":60,"open":false},{"key":"gt_60","lo":60,"hi":null,"open":true}]}
    ,{"key":"row_class","exercise":"row-incline","classes":[{"key":"0","lo":0,"hi":0,"open":false},{"key":"1_5","lo":1,"hi":5,"open":false},{"key":"6_10","lo":6,"hi":10,"open":false},{"key":"11_15","lo":11,"hi":15,"open":false},{"key":"gt_15","lo":15,"hi":null,"open":true}]}
    ,{"key":"dip_class","exercise":"dip-pb","classes":[{"key":"0","lo":0,"hi":0,"open":false},{"key":"1_3","lo":1,"hi":3,"open":false},{"key":"4_7","lo":4,"hi":7,"open":false},{"key":"8_12","lo":8,"hi":12,"open":false},{"key":"13_20","lo":13,"hi":20,"open":false},{"key":"gt_20","lo":20,"hi":null,"open":true}]}
    ,{"key":"support_hold_class","exercise":"support-hold-pb","classes":[{"key":"lt_10","lo":0,"hi":10,"open":false},{"key":"10_30","lo":10,"hi":30,"open":false},{"key":"gt_30","lo":30,"hi":null,"open":true}]}
    ,{"key":"hollow_hold_class","exercise":"hollow-hold","classes":[{"key":"lt_15","lo":0,"hi":15,"open":false},{"key":"15_30","lo":15,"hi":30,"open":false},{"key":"30_60","lo":30,"hi":60,"open":false},{"key":"gt_60","lo":60,"hi":null,"open":true}]}
    ]
    """#

    static let screening = #"""
    [{"id":"SCR-01","question":"Hast du eine bekannte Herz-, Kreislauf-, Stoffwechsel- oder Nierenerkrankung?"}
    ,{"id":"SCR-02","question":"Hat dir eine Ärztin oder ein Arzt geraten, dein Training einzuschränken?"}
    ,{"id":"SCR-03","question":"Nimmst du Medikamente, die deine Belastbarkeit beeinflussen?"}
    ,{"id":"SCR-04","question":"Bist du schwanger?"}
    ,{"id":"SCR-05","question":"Hast du eine andere Erkrankung, die dein Training beeinflusst?"}
    ,{"id":"SCR-06","question":"Hast du ein Knochen- oder Gelenkproblem, das sich durch Training verschlechtern könnte?"}
    ]
    """#

    static let flags: [String: String] = [
        "RF-01": #"{"id":"RF-01","urgency":"D","action":"lock","question":"Gab es einen plötzlichen Knall, Ruck oder stechenden Schmerz mit sofortigem Kraftverlust oder sichtbarer Formveränderung, zum Beispiel ein verschobener Muskelbauch am Oberarm oder an der Brust?","advice":"Lass das in den nächsten Tagen ärztlich abklären. Bis du eine Freigabe bestätigst, plant die App für diese Region keine Belastung.","minor_only":false,"followup":null}"#,
        "RF-02": #"{"id":"RF-02","urgency":"D","action":"lock","question":"Hast du Taubheit, Kribbeln, ausstrahlenden Schmerz oder Schwäche in Arm, Hand oder Bein?","advice":"Lass das in den nächsten Tagen abklären. Bis du eine Freigabe bestätigst, plant die App für diese Region keine Belastung.","minor_only":false,"followup":null}"#,
        "RF-03": #"{"id":"RF-03","urgency":"D","action":"lock","question":"Hast du deutlich Kraft verloren, oder kannst du das Gelenk nicht normal bewegen oder belasten?","advice":"Lass das in den nächsten Tagen abklären. Bis du eine Freigabe bestätigst, plant die App für diese Region keine Belastung.","minor_only":false,"followup":null}"#,
        "RF-04": #"{"id":"RF-04","urgency":"A","action":"rtt0","question":"Hast du Schmerzen in Ruhe oder nachts, unabhängig von der Position?","advice":"Eine Abklärung bei einer Fachperson ist sinnvoll. Bis dahin plant die App für diese Region nur Übungen mit geringer Last.","minor_only":false,"followup":null}"#,
        "RF-05": #"{"id":"RF-05","urgency":"D","action":"lock","question":"Hast du eine deutliche Schwellung, einen grossen Bluterguss oder eine Formveränderung an einem Gelenk?","advice":"Lass das in den nächsten Tagen abklären. Bis du eine Freigabe bestätigst, plant die App für diese Region keine Belastung.","minor_only":false,"followup":{"id":"RF-05-displaced","urgency":"N","action":"stop","question":"Wirkt das Gelenk sichtbar verschoben oder ausgerenkt?","advice":"Bitte such sofort ärztliche Hilfe. Die App plant kein Training, bis du eine Freigabe bestätigst."}}"#,
        "RF-06": #"{"id":"RF-06","urgency":"D","action":"lock","question":"Gab es vor Beginn der Schmerzen einen Sturz, einen Aufprall oder einen anderen Unfall?","advice":"Lass das in den nächsten Tagen abklären. Bis du eine Freigabe bestätigst, plant die App für diese Region keine Belastung.","minor_only":false,"followup":null}"#,
        "RF-07": #"{"id":"RF-07","urgency":"N","action":"stop","question":"Hast du Fieber, ist ein Gelenk gerötet oder überwärmt, oder fühlst du dich allgemein krank?","advice":"Bitte lass das sofort ärztlich abklären. Die App plant kein Training, bis du eine Freigabe bestätigst.","minor_only":false,"followup":null}"#,
        "RF-08": #"{"id":"RF-08","urgency":"N","action":"stop","question":"Hast du zusammen mit den Rückenschmerzen Probleme mit Blase oder Darm, Taubheit im Gesäss- oder Genitalbereich oder sexuelle Funktionsstörungen?","advice":"Bitte such sofort ärztliche Hilfe, das ist ein Notfall. Die App plant kein Training, bis du eine Freigabe bestätigst.","minor_only":false,"followup":null}"#,
        "RF-09": #"{"id":"RF-09","urgency":"D","action":"lock","question":"Hattest du schon einmal eine Krebserkrankung, oder hast du ungewollt Gewicht verloren?","advice":"Lass die Rückenschmerzen in den nächsten Tagen ärztlich abklären. Bis du eine Freigabe bestätigst, plant die App für diese Region keine Belastung.","minor_only":false,"followup":null}"#,
        "RF-10": #"{"id":"RF-10","urgency":"N","action":"stop","question":"Hattest du bei Belastung Brustschmerz, ungewohnte Atemnot, Schwindel oder eine Ohnmacht?","advice":"Bitte lass das sofort ärztlich abklären, bevor du weiter trainierst. Die App plant kein Training, bis du eine Freigabe bestätigst.","minor_only":false,"followup":null}"#,
        "RF-11": #"{"id":"RF-11","urgency":"A","action":"rtt0","question":"Bestehen deine Beschwerden trotz angepasster Belastung seit mehr als 4 Wochen, oder werden sie schlechter?","advice":"Eine Abklärung bei einer Fachperson ist sinnvoll. Bis dahin plant die App für diese Region nur Übungen mit geringer Last.","minor_only":false,"followup":null}"#,
        "RF-12": #"{"id":"RF-12","urgency":"A","action":"rtt0","question":"Hast du seit mehr als 7 Tagen Schmerzen im Handgelenk, wenn du dich darauf stützt?","advice":"Lass das bei einer Fachperson anschauen. Bis dahin pausiert die App Übungen, bei denen du dich auf die Hände stützt.","minor_only":true,"followup":null}"#,
        "RF-13": #"{"id":"RF-13","urgency":"A","action":"rtt0","question":"Hast du seit mehr als 7 Tagen Rückenschmerzen bei Übungen, in denen sich der Rücken nach hinten biegt (zum Beispiel Brücke)?","advice":"Lass das bei einer Fachperson anschauen. Bis dahin pausiert die App Übungen, bei denen sich der Rücken nach hinten biegt.","minor_only":true,"followup":null}"#,
    ]

    static let regionList: [(String, String, [String])] = [
        ("shoulder_front", "vorne an der Schulter", ["RF-01","RF-02","RF-03","RF-04","RF-05","RF-06","RF-07","RF-10"]),
        ("shoulder_top_side", "oben oder seitlich an der Schulter", ["RF-01","RF-02","RF-03","RF-04","RF-05","RF-06","RF-07","RF-10"]),
        ("wrist_back_extension", "am Handrücken oder beim Strecken des Handgelenks", ["RF-01","RF-02","RF-03","RF-04","RF-05","RF-06","RF-07","RF-10","RF-12"]),
        ("wrist_pinky_side", "am Handgelenk auf der Kleinfingerseite", ["RF-01","RF-02","RF-03","RF-04","RF-05","RF-06","RF-07","RF-10","RF-12"]),
        ("elbow_inner", "innen am Ellbogen", ["RF-01","RF-02","RF-03","RF-04","RF-05","RF-06","RF-07","RF-10"]),
        ("elbow_outer", "aussen am Ellbogen", ["RF-01","RF-02","RF-03","RF-04","RF-05","RF-06","RF-07","RF-10"]),
        ("elbow_crease", "in der Ellenbeuge", ["RF-01","RF-02","RF-03","RF-04","RF-05","RF-06","RF-07","RF-10"]),
        ("lower_back", "am unteren Rücken", ["RF-01","RF-02","RF-03","RF-04","RF-05","RF-06","RF-07","RF-08","RF-09","RF-10","RF-13"]),
        ("fingers_forearm_inner", "an Fingern oder Unterarm-Innenseite", ["RF-01","RF-02","RF-03","RF-04","RF-05","RF-06","RF-07","RF-10"]),
        ("chest", "an der Brust", ["RF-01","RF-02","RF-03","RF-04","RF-05","RF-06","RF-07","RF-10"]),
        ("knee", "am Knie", ["RF-01","RF-02","RF-03","RF-04","RF-05","RF-06","RF-07","RF-10"]),
        ("other", "an einer anderen Stelle", ["RF-01","RF-02","RF-03","RF-04","RF-05","RF-06","RF-07","RF-10"]),
    ]
}
#endif
