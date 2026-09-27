# Recherche-Plan: Wissenslücken pro Stream

> Phase 1c. Leitet aus dem PDF-Extrakt (`01_pdf_extract.md`) und der
> Codebasis (`codebase_notes.md`) ab, was die Web-Recherche in Phase 2
> schliessen muss. Jede Lücke ist mit der Stelle verknüpft, an der der
> Algorithmus sie braucht.

## Ausgangslage

Die PDFs (P-01 bis P-04) decken **nur Planche und Maltese** ab, und zwar nur die
**Dosierung einzelner Einheiten** für bereits fortgeschrittene Athleten. Es
fehlen Frequenz, Progressionsregeln, Periodisierung, Deload, Aufwärmen,
Formkriterien, Einstiegskriterien, Verletzungen und alle anderen Skills
(`01_pdf_extract.md` §5). Die Codebasis hat ein präzises Log, eine Unlock-DSL
und eine Content-Pipeline, aber keine belegten Inhalte, kein Nutzerprofil und
keine Übungs-Metadaten für Belastungssteuerung (`codebase_notes.md` §5).

## Gemeinsame Regeln für alle Streams

- Recherche auf Englisch, Dokumente auf Deutsch, Code-Bezeichner auf Englisch.
- Mindestens ~10 unabhängige, hochwertige Quellen; Primärquellen bevorzugt.
- Quellen-IDs mit Stream-Präfix (`A-01`, `B-01` …). PDF-Quellen heissen `P-01`
  bis `P-04` und dürfen von jedem Stream zitiert werden.
- Evidenzstufen: **A** systematische Reviews, Meta-Analysen, RCTs · **B**
  Einzelstudien, sportwissenschaftliche Fachliteratur · **C** etablierte
  Coaching-Literatur, erfahrene Coaches · **D** Foren, Social Media, YouTube
  (nur als Praxisindiz).
- Jede Aussage und jede Zahl trägt eine Quellen-ID. Ohne Evidenz: explizit
  **Praxisheuristik**. Keine erfundenen Studien, Zahlen oder Zitate.
- Widersprüche werden dokumentiert, nicht geglättet.
- Keine Diagnosen. Die App steuert Training und verweist an Fachpersonen.
- Jede Datei endet mit **«Parameter für den Algorithmus»** (Zahl, Quelle,
  Evidenz), **«Widersprüche»**, **«Offene Fragen»** und **«Quellen»** (Tabelle
  im einheitlichen Format, damit `00_sources.md` automatisch zusammengeführt
  werden kann).

## A — Skills & Progressionen → `02_skills_progressions.md`

| Lücke | Wofür der Algorithmus es braucht |
|---|---|
| Vollständige Progressionsleitern für Handstand, L-Sit/V-Sit/Manna, Front Lever, Back Lever, Planche, Human Flag, Maltese, Muscle-up (Stange/Ringe), Swing-/Release-Elemente, One-Arm Pull-up, HSPU, Pistol Squat, gewichtete Grundübungen | Knoten des Skill-Graphen |
| Einstiegsstufen unterhalb der PDF-«Beginner» (Liegestütz, Dips, Support-Halte, Lean, Handgelenk-/Schulterkonditionierung) | Wurzeln des Graphen; Anfänger-Pfade (Persona 1, 5) |
| Voraussetzungen zwischen Skills (z. B. Muscle-up ← Klimmzüge + Dips; Planche ← Pseudo-Liegestütze, Lean) | Kanten `prerequisite`/`recommended` |
| Messbare Unlock-Kriterien je Stufe: Haltezeit oder Wiederholungen, Wiederholungszahl über Tage, Formkriterien (Hüfthöhe, Armstreckung, Protraktion, Körperlinie) | Unlock-DSL (`occurrences`, `min_form_quality`, `within_days`) |
| Wann innerhalb einer Stufe zur nächsten gewechselt wird (z. B. «10–15 s sauber, dann nächste Stufe») | Progressionsregeln |
| Typische Dauer bis zur nächsten Stufe, mit Spannweite und Quelle | `est_weeks_from_prev`; Realismus-Check von Zielen (Persona 5) |
| Häufige Fehler je Stufe | `common_faults`, Formhinweise |
| Carryover zwischen Skills (z. B. Planche ↔ Handstand-Press, Front Lever ↔ Back Lever) | Konfliktauflösung, Synergien bei mehreren Zielen (Persona 2) |
| Relative Schwierigkeit der Stufen (Schwierigkeitsskala, z. B. aus Coaching-Literatur oder Hebelarm-Rechnung) | Einstufung, Dosierung, Ersatzübungen |
| Klärung der PDF-Kürzel: `supi`, `neck band`, Zanetti, Dead Planche, Elevator, «wide» | korrekte Übungsdefinitionen |

## B — Trainingsmethodik → `03_training_methods.md`

| Lücke | Wofür |
|---|---|
| Periodisierung: linear, wellenförmig, Block; Evidenz für Kraft und Skill | Mesozyklus-Struktur |
| Dosierung Isometrie: Haltezeit, Intensität (% der Maximalhaltezeit), Sätze, kumulierte Haltezeit pro Einheit, Winkelspezifität | Dosierungsregeln für Statics |
| Dosierung dynamisch: Wiederholungsbereiche, Sätze, Nähe zum Versagen für Kraft vs. Hypertrophie | Dosierungsregeln für Kraftübungen |
| RPE/RIR, Autoregulation, Genauigkeit der RIR-Schätzung | Adaption aus Logs, Tagesform |
| Frequenz pro Skill und Muskelgruppe; Greasing the Groove; Ganzkörper vs. Split | Wochenstruktur |
| Interferenz zwischen Skills (z. B. zwei Straight-Arm-Skills), Reihenfolge-Effekte | Konfliktauflösung (Persona 2) |
| Satzpausen je Reiztyp; Pausen zwischen Einheiten | Pausen, Tagesabstände |
| Deload: Auslöser, Häufigkeit, Gestaltung | Deload-Trigger |
| Aufwärmen: Nutzen, Aufbau, Dauer | Warm-up-Block |
| Weighted Calisthenics: Laststeigerung, Übergang von Wiederholungen zu Last | Progressionsregeln |
| Steigerungsraten (Volumen, Intensität) pro Woche | Belastungssteuerung |
| Detraining und Wiedereinstieg: wie schnell Kraft/Skill verloren geht und zurückkommt | Persona 4 (6 Monate Pause) |
| Zeiteffizientes Training, Minimaldosis, Kürzung bei wenig Zeit | Kurze Einheiten (PDF-Einheiten brauchen 60–90 min, P-01–P-04) |
| Die PDF-Dosierungen (2–20 s, 1–7 min Pause) im Licht der Literatur | Validierung der Praxisquelle |

## C — Anatomie & Biomechanik → `04_anatomy.md`

| Lücke | Wofür |
|---|---|
| Primäre/sekundäre Muskeln und Stabilisatoren je Übung (EMG-Studien, wo vorhanden) | Muskel-Metadaten, Ersatzübungen |
| Gelenk- und Sehnenbelastung je Übung (Handgelenk-Extension, Ellbogen-Valgus, Bizepssehne bei gestrecktem Arm, vordere Schulter) | Belastungskategorien pro Gelenk |
| Straight-Arm vs. Bent-Arm: mechanische Unterschiede, welche Strukturen limitieren | eigene Volumenkategorie Straight-Arm |
| Hebelarm-Physik: Drehmoment in Tuck/Adv Tuck/Straddle/Full für Planche und Front Lever, aus anthropometrischen Segmentdaten | relative Schwierigkeit, Dosierung, Ersatzlogik |
| Einfluss von Handposition (supiniert, Parallettes, Ringe) und Bandassistenz auf Last | Ersatzübungen, Assistenz-Äquivalente |
| Körpergewicht und Anthropometrie als Einflussfaktoren | Onboarding (Körpergewicht, Grösse) |

## D — Verletzungen & Prehab → `05_injuries_prehab.md`

| Lücke | Wofür |
|---|---|
| Epidemiologie: Calisthenics/Street Workout, Turnen, Klettern als Analogie | Priorisierung der Risiken |
| Mechanismen und Risikofaktoren: mediale/laterale Epikondylopathie, distale und lange Bizepssehne, Handgelenk (dorsales Impingement, TFCC), Schulter, Finger/Unterarm | Verletzungsflags, Kontraindikationen |
| Adaptationszeiten Sehne/Bindegewebe vs. Muskel | Steigerungsraten, Volumengrenzen |
| Load-Management: akut:chronisch, Schmerz-Monitoring-Modell, Kritik daran | Belastungssteuerung |
| Return-to-Training: allgemeine, belegte Prinzipien (nicht medizinisch) | Wiedereinstiegslogik (Persona 3, 4) |
| Kontraindikationen je Übung und Beschwerdebild | Ausschlussregeln |
| Red Flags mit ärztlicher Abklärung (z. B. Kraftverlust, Taubheit, Deformität nach «Knall», Nachtschmerz) | Red-Flag-Logik |
| Prehab: welche präventiven Übungen Evidenz haben | Prehab-Block |

## E — CNS & motorisches Lernen → `06_cns_motor_learning.md`

| Lücke | Wofür |
|---|---|
| Neuronale Adaptationen (Rekrutierung, Rate Coding, intermuskuläre Koordination, kortikal) und ihr Zeitverlauf | Erwartung an frühe Fortschritte, Kalibrierung |
| Stadien motorischen Lernens, Spezifität, Übungsvariabilität, contextual interference | Übungsauswahl, Variation |
| Geblockte vs. verteilte Übung; Frequenz vs. Volumen beim Skillerwerb | Frequenz pro Skill |
| Schlaf und Konsolidierung | Tagesabstände, Hinweise |
| Zentrale vs. periphere Ermüdung; Erholungszeitverlauf nach schwerem Krafttraining | Pausen zwischen Einheiten |
| Kritische Einordnung «CNS-Fatigue» | Vermeidung von Mythen in Regeln und Begründungen |
| Konsequenzen: Reihenfolge in der Einheit, Frequenz, Frische bei Skillarbeit | Session-Aufbau (PDF-Muster F-1 prüfen) |

## F — Leistungsdiagnostik → `07_assessment.md`

| Lücke | Wofür |
|---|---|
| Validität und Reliabilität von Maximalwiederholungs-Tests (Liegestütz, Klimmzug, Dip), Haltetests, Mobilitätstests | Onboarding-Tests, optionaler Testtag |
| Standardisierte Protokolle mit Formkriterien | Testtag-Anleitung |
| Prädiktoren für Skill-Bereitschaft (z. B. Klimmzugzahl → Muscle-up) | Einstufung, Voraussetzungen |
| Genauigkeit der Selbsteinschätzung (Wiederholungen, Niveau, RIR) | Konfidenzwerte, Nachkalibrierung |
| Schätzung von Maximalkraft aus Wiederholungen; Körpergewichtsanteile je Übung | Umrechnung zwischen Übungen |
| Kurzfragebögen zu Trainingsalter und Beschwerden (z. B. Screening-Fragen) | Pflichtteil unter 5 min |

## Verknüpfung mit den Personas (Phase 5)

| Persona | Braucht vor allem |
|---|---|
| 1 Anfänger, Outdoor-Park, 2×/Woche, Muscle-up | A (Muscle-up-Leiter, Einstieg), B (Frequenz 2×), F (Einstufung) |
| 2 Fortgeschritten, Gym, 4×/Woche, Planche + Front Lever | A (Carryover), B (Interferenz, Split), C (Straight-Arm-Last), D (Volumengrenzen) |
| 3 Mediale Ellbogenbeschwerden, Ziel Planche | C (Ellbogenlast), D (Kontraindikation, Wiedereinstieg, Red Flags) |
| 4 Wiedereinsteiger nach 6 Monaten | B (Detraining/Retraining), D (Sehnen-Readaptation), F (Nachkalibrierung) |
| 5 Full Planche in 8 Wochen als Anfänger | A (Dauer je Stufe), E (Lernzeitverlauf) |
| 6 Widersprüchliche Angaben | F (Selbsteinschätzung, Plausibilitätsregeln) |
