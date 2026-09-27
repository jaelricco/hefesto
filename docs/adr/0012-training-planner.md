# ADR 0012 — Trainingsplaner: deterministischer Kern, Wissensbasis als Daten

- Status: proposed (Checkpoint nach Stufe 4 des Planer-Tracks)
- Date: 2026-09-27
- Deciders: Jaelricco

Dieses ADR ist auf Deutsch, wie alle Dokumente des Planer-Tracks
(`docs/research/`, `docs/algorithm/`).

## Kontext

Hefesto soll aus einem kurzen Onboarding und den Logs individuelle
Trainingspläne erzeugen und anpassen. Die Recherche (`docs/research/02`–`07`,
572 Quellen) und die Synthese (`08_synthesis.md`) zeigen: Die Struktur eines
Planers (Reihenfolge, Frequenz, Autoregulation, Belastungsdeckel,
Unsicherheit der Selbstauskunft) ist gut belegt; die Zahlen für
Straight-Arm-Skills sind überwiegend Praxis oder Übertragung. Am Checkpoint
nach Stufe 3 wurden alle Vorschläge angenommen (ENT-1 bis ENT-10 in `08`
§6.2, OE-1 bis OE-9 in `onboarding.md` §10). Die Spezifikation steht in
`docs/algorithm/spec.md`.

Randbedingungen aus dem Projekt: `internal/domain` ohne I/O
(`arch_test.go`), OpenAPI als Quelle der Wahrheit (ADR 0007), Unlocks nie
zurücknehmen und keine Mechanik, die Pausen bestraft (ADR 0003, ADR 0008),
Inhalte nur aus Recherche (`CONTENT_AUTHORING.md`), keine medizinischen
Aussagen.

## Entscheidungen

### 1. Deterministischer, regelbasierter Kern

Der Planer ist ein Regel- und Constraint-System: gierige Auswahl mit festen
Prioritäten, danach harte Prüfung aller Deckel und Abstände mit
deterministischer Kürzung. Gleiche Eingaben ergeben byte-gleich denselben
Plan (`input_hash`, `ruleset_version`, `content_version_id`). Keine
Optimierung und kein Lernverfahren im Kern, weil sie dünn belegte Zahlen als
Zielfunktion behandeln und schwer erklärbar sind. Ein Sprachmodell darf
später höchstens Begründungstexte umformulieren oder Fragen zum Plan
beantworten; es erzeugt und ändert keine Zahl (spec §1.1, EXPL-06).

### 2. Schichten

- `internal/domain/planning`: reine Funktionen (`Generate`, `Adapt`,
  `ValidateKnowledge`, `Materialize`) über einem unveränderlichen Snapshot.
- `internal/planning`: Anwendungsdienst mit Ports (Snapshot, Pläne,
  Entscheidungsprotokoll, Wissensbasis, Uhr, IDs).
- `internal/store`: Adapter für die Ports; `internal/http`: Handler,
  OpenAPI zuerst.

### 3. Wissensbasis als versionierte Daten, Logik im Code

Parameter, Leitern, Lastprofile, Beschwerde-Matrix, Red Flags,
Einheitsvorlagen, Prehab, Regelkatalog und Quellen liegen als YAML unter
`content/training/`; Übungen bekommen einen optionalen `training:`-Block. Die
Logik jeder Regel steht in einer getesteten Go-Funktion; Werte, Texte und
Belege liest sie nur aus dem Katalog. Eine eigene Regelsprache in YAML wurde
verworfen: eine zweite Programmiersprache ohne Typprüfung. Validiert wird in
CI (`contentlint -strict`), beim Import und beim Laden in der API; ist die
Wissensbasis ungültig, antworten nur die Planungs-Endpunkte mit `503`.

### 4. Jede Zahl ist belegt oder als Heuristik markiert

Werte tragen die Parameter-IDs der Recherche (`PAR-A-…` bis `PAR-F-…`) oder
sind als `PAR-S-…` mit Begründung festgelegt (spec Anhang B). Jede
Entscheidung im Plan trägt Regel-, Parameter- und Quellen-IDs (`Reason`);
die UI zeigt sie mit Evidenzstufe.

### 5. Dosierung aus der Kapazität, nicht aus dem Unlock-Status

Kapazitäten werden als Schätzung mit Unsicherheit geführt (eindimensionaler
Filter, Heuristik) und konservativ dosiert (μ, μ − 0.5σ, μ − σ je
Konfidenz). Der Planer schreibt nie Unlocks, XP oder Streaks; die
Unlock-Engine bleibt allein zuständig (ADR 0008). Kalibriert wird
submaximal (RIR/SIR 2–3), Maximaltests nur freiwillig und unter
Sicherheitsbedingungen.

### 6. Belastungssteuerung über Strukturkonten

Last wird je Struktur (Handgelenk, Ellbogen, Bizepssehne, Schulter …)
satzbasiert gezählt, getrennt nach gestrecktem und gebeugtem Arm. Wochendeckel
gegen das 3-Wochen-Mittel, Einheitsdeckel gegen das 30-Tage-Maximum,
Mindestabstände zwischen harten Reizen derselben Struktur. Kein
ACWR-Kriterium.

### 7. Verletzungslogik ist Laststeuerung (ENT-2)

Beschwerden steuern Ausschluss, Modifikation und eine Rampe der
Trainingslast; Red Flags führen zu Sperre oder Stopp und zum Verweis an
Fachpersonen. Die Rampe ist keine Rehabilitation und kein Rückkehr-Rat;
`CONTENT_AUTHORING.md` sagt das jetzt ausdrücklich. Keine Diagnosen, keine
Heil- oder Schutzversprechen; der Disclaimer steht im API-Payload.

### 8. Einbindung ins Log

Eine geplante Einheit wird beim Start als `draft`-Session mit
`is_planned`-Sätzen im bestehenden Log-Modell angelegt; jeder Satz-Eintrag
verweist über `set_entries.planned_item_id` auf sein Plan-Item. Der Abschluss
bleibt der bestehende Endpunkt und liefert zusätzlich `plan_changes[]`. Der
Planer schreibt weder Unlocks noch XP noch `user_training_days`: Geplante
Ruhetage werden über das bestehende Ruhetag-Loggen erfasst, Deload-Tage beim
Abschluss der Einheit (ADR 0008 unverändert). Schema-Änderungen sind additiv:
neue Tabellen, `workout_sessions.planned_session_id`, `set_entries.sir_s` und
`set_entries.planned_item_id`.

### 9. Übernommene Checkpoint-Entscheidungen

API client-neutral (ENT-1); Rampe als Laststeuerung (ENT-2); Minderjährige
und Gesundheitsdaten erst nach rechtlicher Prüfung, Gesundheitsdaten nur mit
eigener Einwilligung und getrennt gespeichert (ENT-3, ENT-4, OE-1, OE-2);
DSL-Erweiterungen `min_distinct_days` und `min_load_pct_bw` additiv mit Golden
Files (ENT-5); implizite Vorstufe bleibt (ENT-6); Schmerzberichte als eigene
Tabelle, Check-in optional (ENT-7, OE-8); eigener Screening-Wortlaut (ENT-8,
OE-3); gewichtete Stufen übergangsweise mit 75-kg-Referenz (ENT-9);
Wissensbasis bleibt `draft_placeholder` bis zur fachlichen Abnahme (ENT-10);
keine Sammel-Selbstbestätigung (OE-5); submaximale Kalibrierung statt Testtag
in v1 (OE-6).

## Folgen

- Pläne sind nachvollziehbar, testbar und reproduzierbar; jede Zahl ist
  auffindbar und aus eigenen Logs später nachschärfbar.
- Viele Werte sind Heuristik; sie sind bewusst konservativ und markiert. Die
  Einstiegsrampe ist dadurch langsam (spec §7.4); die Alternative steht als
  ENT-S-1 zur Entscheidung.
- Die Wissensbasis wird ein eigener Pflegeaufwand mit fachlicher Abnahme.
- Neue Tabellen mit Gesundheitsdaten brauchen vor dem öffentlichen Betrieb
  eine rechtliche Prüfung (Einwilligung, Aufbewahrung, Löschung,
  Minderjährige, Screening-Wortlaut).
- Sicherheitsauflagen (Trainingsstopp, gesperrte Regionen) werden als minimale
  Auflage ohne Antworten gespeichert, damit sie auch ohne Einwilligung für
  Gesundheitsdaten wirken; das ist Teil der rechtlichen Prüfung (ENT-S-7).
- Ein unabhängiger Review der Spezifikation (3 kritische, 19 wichtige, 21
  kleinere Befunde) ist eingearbeitet.
- Offene Entscheidungen: spec §14 (ENT-S-1 bis ENT-S-9).
