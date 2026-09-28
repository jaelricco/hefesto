# Spezifikation: Trainingsplan-Algorithmus

> Phase 4. Legt fest, wie der Planer aus dem Start-Zustand des Onboardings
> (`onboarding.md`) und den Logs deterministisch und erklärbar Trainingspläne
> erzeugt und anpasst. Grundlage sind die Recherche (`docs/research/02`–`07`),
> die Synthese (`08_synthesis.md`) und die am Checkpoint nach Phase 3
> angenommenen Entscheidungen (ENT-1 bis ENT-10, OE-1 bis OE-9).
>
> **Kennzeichnung von Zahlen.** Jede Zahl verweist auf eine Parameter-ID der
> Recherche (`PAR-A-…` bis `PAR-F-…`), auf eine Quelle (`A-…` bis `F-…`, `P-…`)
> oder auf einen abgestimmten Wert der Synthese (`08` §4). Werte, die diese
> Spezifikation selbst festlegt, heissen `PAR-S-nn`, sind **Heuristik** und
> tragen ihre Begründung in Anhang B. Regeln heissen nach ihrem Bereich
> (`SAFE-`, `GOAL-`, `WEEK-`, `SESS-`, `SEL-`, `DOSE-`, `LOAD-`, `ADAPT-`,
> `INJ-`, `EXPL-`, `KB-`) und stehen gesammelt in Anhang A.
>
> **Keine Diagnosen.** Der Planer passt Trainingslast an und verweist bei
> Warnzeichen an Fachpersonen. Er nennt keine Verdachtsdiagnose, verspricht
> keine Heilung und keinen Schutz ohne Evidenz (ADR 0003, `CONTENT_AUTHORING.md`,
> PAR-D-38, PAR-D-39).

## Inhalt

0. Überblick und Begriffe
1. Ansatz und Architektur
2. Wissensbasis als Daten
3. Skill-Graph
4. Nutzerzustand
5. Plangenerierung
6. Adaption aus Logs
7. Belastungssteuerung
8. Verletzungslogik
9. Erklärbarkeit
10. Schnittstellen
11. Performance
12. Tests und Personas
13. Umsetzung, Migration und Rollout
14. Entscheidungen des Checkpoints nach Phase 4
15. Umsetzung in Phase 5: Entscheidungen, Abweichungen, Lücken
- Anhang A: Regelkatalog
- Anhang B: Spezifikations-Parameter (`PAR-S`)
- Anhang C: Index der verwendeten Forschungsparameter

## 0. Überblick und Begriffe

### 0.1 Was der Planer tut

1. Er prüft, ob überhaupt geplant werden darf (Sicherheits-Gate, §5.2).
2. Er leitet aus den Zielen die zu trainierenden Leitern ab (§3.4, §5.3).
3. Er verteilt die Reize auf die Woche (§5.4) und baut jede Einheit (§5.5).
4. Er wählt Übungen nach Equipment, Beschwerden und Können (§5.6) und
   dosiert sie (§5.7).
5. Er prüft alle Belastungsdeckel und kürzt deterministisch (§7).
6. Nach jeder Einheit, jedem Schmerzbericht und an jeder Wochengrenze passt
   er Zustand und Plan an (§6, §8).
7. Jede Entscheidung trägt Regel-ID, Parameter-IDs und Quellen-IDs (§9).

### 0.2 Was der Planer nicht tut

- Keine Diagnose, keine Behandlung, keine Aussage über Heilung (§8.1).
- Kein Maximaltest ohne aktive Wahl und Sicherheitsbedingungen
  (`onboarding.md` §4.3); Standard ist die submaximale Kalibrierung.
- Keine XP, keine Unlocks: Freischaltungen entstehen nur aus Logs über die
  bestehende Unlock-Engine (ADR 0008). Der Planer liest den Unlock-Status nur
  zur Anzeige; er dosiert nach der Kapazitätsschätzung (§4.3).
- Keine Mechanik, die Pausen bestraft (ADR 0003): geplante Ruhe, Deloads und
  Lastreduktionen zählen im Streak; keine Texte über Rückstand.
- Keine Zahl aus einem Sprachmodell (§1.1).

### 0.3 Begriffe

| Begriff | Bedeutung |
|---|---|
| Skill, Stufe (Level) | Knoten der Skill-Karte; eine Stufe ist ein **Leistungsnachweis** mit Unlock-Kriterien (`02` §2.1). |
| Übung (Exercise) | Was geloggt wird. Eine Übung kann Primärtest einer Stufe, Progression oder Zubringer sein. |
| Leiter (Ladder) | Geordnete Folge von Übungen **einer Bewegung** nach Schwierigkeit, z. B. Planche: Lean → Tuck → Adv Tuck → Straddle → … Die Leiter ist die Trainingssicht; die Stufen sind die Nachweissicht (§3.3). |
| Arbeitssprosse | Die Sprosse einer Leiter, an der der Planer den Hauptreiz setzt; aus der Kapazität bestimmt (§5.6), nicht aus dem Unlock-Status. |
| Reiz (Stimulus) | Ein Block einer Übung mit Zweck: `skill_max`, `skill_volume`, `balance`, `strength`, `hypertrophy`, `conditioning`, `eccentric`, `technique`, `prehab`, `mobility`, `warmup_ramp`. |
| Intensitätsklasse | `hard` (hart), `moderate` (mittel), `light` (leicht) je Reiz; steuert Abstände (§7.5). |
| Struktur | Gelenk-/Sehnengruppe mit eigenem Lastkonto: `wrist`, `elbow_medial`, `elbow_lateral`, `biceps_distal`, `biceps_long_head_anterior_shoulder`, `shoulder_overhead`, `shoulder_extension`, `lumbar`, `knee`, `fingers_forearm` (`04` Tabelle 2). |
| Region | Ort auf der Körperkarte des Onboardings (`onboarding.md` §3.7); jede Region verweist auf Strukturen (§4.6). |
| Belastungseinheit (BE) | Struktur-Last eines Satzes nach §4.5. |
| Rampe (RTT) | Stufen 0–5 der Laststeuerung nach Beschwerden oder Pausen (`05` §6.2). Trainingssteuerung, keine Rehabilitation (§8.1). |
| Kapazität | Schätzung der frischen Maximalleistung je Übung, Messgrösse und Assistenz als Normalverteilung (μ, σ) (§4.3). |
| Dosiswert | Der Wert, aus dem dosiert wird: μ − k·σ nach Konfidenzklasse (§4.8). |

## 1. Ansatz und Architektur

### 1.1 Deterministischer, regelbasierter Kern

Der Planer ist ein **regel- und constraint-basiertes System**: Eingaben sind
ein Zustands-Snapshot, die Wissensbasis und ein Stichtag; die Ausgabe ist ein
Plan mit Begründungen. Dieselben Eingaben ergeben byte-gleich denselben Plan.

| Anforderung | Umsetzung |
|---|---|
| Reproduzierbar | Reine Funktionen; keine Uhr, kein Zufall, keine Map-Iteration ohne Sortierung; Zeit kommt als Parameter (`now`, `week_start`). Jeder Plan speichert `ruleset_version`, `content_version_id` und `input_hash` (SHA-256 des kanonischen JSON-Snapshots). |
| Testbar | Tabellengetriebene Tests je Regel, Eigenschaftstests über alle Personas (§12). |
| Erklärbar | Jede Entscheidung erzeugt ein `Reason`-Objekt mit Regel-, Parameter- und Quellen-IDs (§9). |
| Sicher | Deckel und Ausschlüsse sind harte Constraints, die nach der Auswahl geprüft werden (§7); ein Plan, der sie verletzt, wird nie ausgegeben. |

**Warum keine Optimierung oder kein Lernverfahren im Kern.** Die Recherche
belegt die Struktur (Reihenfolge, Frequenz, Autoregulation, Deckel) gut, die
Zahlen für Straight-Arm-Skills kaum (`08` §1). Ein Optimierer würde dünn
belegte Zahlen als Zielfunktion behandeln und seine Entscheidungen schwer
erklärbar machen. Ein greedy Verfahren mit festen Prioritäten und
nachgelagerter Constraint-Prüfung ist erklärbar, deterministisch und schnell
(§11).

**Optionale KI-Schicht.** Ein Sprachmodell darf später höchstens
(a) Begründungstexte umformulieren oder übersetzen und (b) Fragen zum Plan
beantworten, jeweils mit dem Plan und seinen `Reason`-Objekten als einziger
Grundlage. Es erzeugt, ändert oder dosiert nichts; seine Ausgabe wird nicht
zurückgeschrieben. Nicht Teil von v1 (Regel `EXPL-06`).

### 1.2 Pakete und Datenfluss

```
content/training/*.yaml ──┐                    (Wissensbasis, versioniert)
content/exercises/*.yaml ─┼─ internal/content (Laden, Schema, Semantik) ──┐
content/skills/*.yaml ────┘   cmd/contentlint (CI), cmd/seed (Import)     │
                                                                          ▼
                                     planner_knowledge (Snapshot, JSONB, je content_version)
                                                                          │
internal/store/planning.go ── implementiert die Ports ──┐                 │
                                                        ▼                 ▼
                         internal/planning  (Anwendungsdienst: Ports, Ablauf, Transaktionen)
                                                        │
                                                        ▼
                         internal/domain/planning  (rein, ohne I/O; arch_test.go)
                           kb.go        Typen der Wissensbasis + Validierung
                           state.go     Nutzerzustand
                           estimate.go  Kapazität (Filter), Konfidenz
                           load.go      Belastungseinheiten, Deckel, Abstände
                           injury.go    Regionen-Automat, Matrix, Red Flags
                           generate.go  Wochenplan und Einheiten
                           adapt.go     Anpassung aus Logs und Schmerzberichten
                           explain.go   Reasons, Regelkatalog
                                                              │
internal/http/handlers_plan.go ── ruft internal/planning; OpenAPI-first (ADR 0007)
```

- `internal/domain/planning` importiert nur die Standardbibliothek,
  `github.com/google/uuid` und `internal/domain/progress` (für die Unlock-DSL).
  `internal/domain/arch_test.go` bekommt `internal/planning` und
  `internal/content` als zusätzliche verbotene Importe (der Test prüft nur
  direkte Importe; `internal/content` benutzt `os`). Der Kern bekommt einen
  fertigen Zustands-Snapshot und gibt Pläne, Zustandsänderungen und
  Begründungen zurück; er liest und schreibt nichts selbst.
- `internal/planning` definiert die Ports (Interfaces) für Zustand, Logs,
  Pläne, Wissensbasis, Uhr und ID-Erzeugung (§10.1), ruft den Kern auf und
  schreibt die Ergebnisse. `internal/store` implementiert die Ports; Tests
  nutzen In-Memory-Adapter.
- `internal/content` lädt die Wissensbasis aus Dateien wie heute die Skills
  (`content.Load`), prüft Schema und Semantik (§2.6) und liefert die Typen
  des Kerns.

### 1.3 Determinismus im Detail

| Regel | Inhalt |
|---|---|
| Sortierung | Jede Auswahl endet mit einem vollständigen Tie-Break: Priorität, dann Leiterrang, dann Slug (lexikografisch). |
| Zahlen | Rechnung in `float64`; jedes Produkt, das in eine Summe eingeht, wird ausdrücklich als `float64(a*b)` geschrieben, damit der Compiler auf arm64 keine FMA-Anweisung erzeugt (sonst unterscheiden sich Ergebnisse zwischen Architekturen). Ausgabewerte werden am Ende einmal gerundet nach `PAR-S-18`: Wdh., Sätze und Haltezeiten abgerundet, Pausen auf 15 s, Laststeigerungen auf die kleinste verfügbare Scheibe **auf**gerundet (PAR-B-32). |
| Zeit | Der Kern kennt nur `now` und die Zeitzone des Users (IANA, wie `workout_sessions.timezone`); Wochen beginnen am Montag (ISO, wie ADR 0011). |
| Hash | `input_hash` = SHA-256 über die kanonische JSON-Form (RFC 8785, JCS) von Snapshot, `now`, `week_start`, `ruleset_version` und `content_version_id`. |
| IDs | UUIDv7 werden ausserhalb des Kerns erzeugt und als Generator-Interface injiziert; Tests nutzen einen deterministischen Generator. |
| Versionen | Ein Plan ist an `ruleset_version` und `content_version_id` gebunden. Ändert sich eine davon, wird der laufende Plan zu Wochenbeginn neu erzeugt, nie mitten in der Woche (`WEEK-08`). |

## 2. Wissensbasis als Daten

### 2.1 Dateien

Alle Inhalte liegen im Repository unter `content/`, getrennt vom Code, und
folgen `CONTENT_AUTHORING.md`: keine erfundenen Zahlen, `status:
draft_placeholder` bis zur Abnahme durch einen Menschen (ENT-10).

| Datei | Inhalt | Herkunft |
|---|---|---|
| `content/training/manifest.yaml` | `ruleset_version` (SemVer), Liste der Dateien | – |
| `content/training/parameters.yaml` | Jeder vom Planer benutzte Parameter: ID, Schlüssel, Wert(e), Einheit, Quellen-IDs, Evidenz, `heuristic: true/false`, Begründung | `PAR-*` der Streams, `PAR-S` (Anhang B) |
| `content/training/rules.yaml` | Regelkatalog: ID, Titel, Textschlüssel, Parameter-IDs, Quellen-IDs, Evidenz | Anhang A |
| `content/training/sources.yaml` | Quellenverzeichnis: ID, Titel, Autoren, Jahr, URL/DOI, Typ, Evidenz | erzeugt aus `docs/research/00_sources.md` |
| `content/training/structures.yaml` | Strukturen, Lastgruppe für den Wochendeckel (§7.2), Regionen-Zuordnung | `04` Tabelle 2, `05` §8, `PAR-S-15` |
| `content/training/load_profiles.yaml` | Ordinalprofil 0–3 je Lastfamilie × Struktur, Referenzstufe je Familie, Modifikatoren | `04` §3.8 Tabelle 2, PAR-C-27, 40, 44, 47 |
| `content/training/ladders.yaml` | Leitern: geordnete Übungen, Momentverhältnis bzw. KG-Anteil je Sprosse, OG-Ordinal, Kennzeichen `foundation` für Grundlagenleitern (§3.4) | `02` §3–7, PAR-A-25–38, PAR-C-03–10, 18–22 |
| `content/training/complaint_matrix.yaml` | Region × Beschwerdefamilie → `X`/`M`/`S`/`–` mit Modifikationsliste | `05` §8 |
| `content/training/red_flags.yaml` | RF-01 bis RF-13: Textschlüssel, Regionen, Bedingungen (`is_minor`), Dringlichkeit, Aktion | `05` §9, `onboarding.md` §3.7 |
| `content/training/session_templates.yaml` | Vorlagen je Einheitslänge, Aufwärmdauer, Kürzungsreihenfolge | PAR-B-64–69, `onboarding.md` §3.3 |
| `content/training/prehab.yaml` | Prehab je Region: Übungen, Evidenzlabel, Dosierung | `05` §10, PAR-D-01, 37, 38 |
| `content/training/readiness_hints.yaml` | Weiche Bereitschaftshinweise je Stufe | PAR-A-52–54, 69, PAR-F-62–64 |
| `content/training/onboarding.yaml` | Fragen, Antwortklassen, Klassenwerte, Plausibilitätsregeln | `onboarding.md` §3, §5 |
| `content/exercises/*.yaml` | neuer optionaler Block `training:` (§2.2) | `02`, `04` |
| `content/skills/*.yaml` | unverändert; Kriterien ggf. mit den DSL-Erweiterungen (§3.2) | `02` §2.4 |

### 2.2 Übungs-Metadaten (`training:`-Block)

Der Block ist additiv; Übungen ohne ihn plant der Planer nicht.

| Feld | Typ | Bedeutung | Grundlage |
|---|---|---|---|
| `pattern` | Vokabular: `vertical_pull`, `horizontal_pull`, `vertical_push`, `horizontal_push`, `straight_arm_push`, `straight_arm_pull`, `handbalance`, `core_flexion`, `core_extension`, `legs`, `grip_hang`, `mobility`, `prehab` | Bewegungsmuster; Kapazitätsgruppen, Antagonisten, Ersatz | `02` §4–7 |
| `straight_arm` | `none` · `straight_arm` · `straight_arm_axial` | Kategorie nach PAR-C-49; `axial` zählt für Handgelenk und Überkopf, nicht für Ellbogen/Bizeps | PAR-C-49 |
| `direction` | `push` · `pull` · `none` | Zugrichtung für Paarung und gleiche Richtung | PAR-B-48, PAR-B-81 |
| `load_family` | Slug aus `load_profiles.yaml` | Profilzeile in `04` Tabelle 2 | PAR-C-44, 45 |
| `load_modifiers` | Liste: `neutral_grip`, `rings`, `wide_or_supinated_grip` | −1 Handgelenk / +1 Bizeps und vordere Schulter / +1 Überkopf, gekappt 0–3 | PAR-C-27, 47, 40 |
| `complaint_family` | eine der neun Spalten der Matrix `05` §8 oder `none` | Verknüpfung zur Beschwerde-Matrix | `05` §8 |
| `supinated_straight_arm` | Bool | eigene Risikokategorie | PAR-D-41, `08` §3 |
| `limiting_factor` | `strength` · `balance` · `mixed` | steuert Frequenz (§5.4) | PAR-E-28 |
| `stimulus_types` | Teilmenge der Reiztypen (§0.3) | wofür die Übung eingesetzt werden darf | `03` §3–4, `06` §8 |
| `measure` | wie `default_measure` | Messgrösse für Kapazität und Dosierung | – |
| `requires` | Menge aus dem Equipment-Vokabular (`onboarding.md` §3.4) | alle nötig; Varianten auf anderem Gerät sind eigene Übungen | H-5 (`08` §3) |
| `ladder`, `rung` | Leiter-Slug, Rang | Position in der Leiter | `ladders.yaml` |
| `og_ordinal` | Zahl | Schwierigkeit auf der OG-Skala; Stufengrenzen PAR-A-23 | PAR-A-22–38 |
| `torque_ratio` bzw. `bw_fraction` | Zahl | Last relativ zur Referenz der Familie | PAR-C-03–10, 18–22; PAR-F-08–11, 57, 58 |
| `assistable` | Bool | Band-Assistenz möglich | PAR-C-13, 61, 62 |
| `loadable` | Bool | Zusatzlast möglich | PAR-A-67, PAR-B-32 |
| `gtg_allowed` | Bool | nur Bent-Arm-Grundübungen und Balance | PAR-B-80, PAR-E-27, `08` Rang 20 |
| `prehab_region` | Region oder leer | für den Prehab-Block | `05` §10 |

### 2.3 Parameter-Katalog

```yaml
- id: PAR-D-09
  key: max_weekly_load_increase_straight_arm_pct
  value: 10
  unit: percent_vs_3_week_mean
  sources: [D-17, D-18, D-27, D-33, D-36]
  evidence: heuristic
  rationale_key: par.d09.rationale      # Text in docs/research/05, Anmerkung
- id: PAR-S-02
  key: static_target_total_hold_s
  value: 60
  unit: s
  sources: [A-63, B-09]
  derived_from: [PAR-A-64, PAR-B-09]
  evidence: heuristic
  rationale_key: par.s02.rationale
```

Der Code liest Werte **nur** über den Katalog (`kb.Param("PAR-D-09")`), nie als
Literal. Eine Parameter-ID, die der Code benutzt, aber der Katalog nicht kennt,
lässt die Validierung scheitern (`KB-03`).

### 2.4 Regel-Katalog

```yaml
- id: LOAD-02
  title_key: rule.load02.title          # «Wochendeckel je Struktur»
  text_key: rule.load02.text            # mit Platzhaltern {structure}, {cap_pct}
  params: [PAR-D-09, PAR-D-10, PAR-D-11, PAR-D-02, PAR-D-23, PAR-S-14, PAR-S-15]
  evidence: heuristic
```

Die **Logik** der Regeln steht im Go-Code des Kerns, jede Regel in einer
eigenen, getesteten Funktion; ihre **Parameter, Texte und Belege** stehen im
Katalog. Eine eigene Regelsprache in YAML wäre eine zweite
Programmiersprache ohne Typprüfung und schwerer zu testen; die Trennung
«Logik im Code, Wissen in Daten» hält beides prüfbar (Entscheidung in
ADR 0012). Die Quellen-IDs einer Regel ergeben sich aus ihren Parametern plus
eigenen `sources`.

### 2.5 Versionierung

- `ruleset_version` (SemVer) im Manifest. **Major**: andere Regel-Logik oder
  entfernte Parameter; **Minor**: neue Regeln, Parameter oder Inhalte;
  **Patch**: Texte, Korrekturen ohne Wirkung auf Pläne.
- Der Import (`cmd/seed`) schreibt die validierte Wissensbasis als ein
  kanonisches JSON-Dokument in `planner_knowledge` (§4.9), gebunden an die
  bestehende `content_versions`-Zeile (Checksumme über alle Dateien, wie heute).
- Die API lädt beim Start und bei neuer Content-Version den jüngsten Snapshot
  und validiert ihn erneut (§2.6).
- Jeder Plan speichert `ruleset_version`, `content_version_id` und
  `input_hash`; ein alter Plan lässt sich mit dem Snapshot seiner Version
  nachrechnen.

### 2.6 Validierung

`cmd/contentlint` (CI, `-strict`) und der Import prüfen, die API prüft beim
Laden erneut. Befunde folgen dem bestehenden `content.Issue`-Modell.

| ID | Prüfung | Schwere |
|---|---|---|
| KB-01 | JSON-Schema jeder Datei (`content/schema/training/*.schema.json`) | Fehler |
| KB-02 | Referenzen: jede Übung, Leiter, Struktur, Region, Familie, Regel, Parameter- und Quellen-ID existiert | Fehler |
| KB-03 | Jeder im Code benutzte Parameter existiert im Katalog; jeder Katalog-Parameter hat Quellen oder `heuristic: true` mit Begründung | Fehler |
| KB-04 | Zyklenfreiheit: Skill-Graph inklusive impliziter Vorstufen (wie `progress.topoSort`) und jede Leiter | Fehler |
| KB-05 | Leitern sind monoton im `og_ordinal` (fällt nicht mit steigendem Rang). `torque_ratio` und `bw_fraction` dürfen dokumentiert abweichen, weil das OG-Level nicht linear im Moment ist (PAR-C-17; z. B. One-Leg 0.88 nach Straddle 0.91, PAR-C-05, PAR-C-07) | Fehler bzw. Warnung ohne Dokumentation |
| KB-06 | Matrix vollständig: jede Region × Beschwerdefamilie hat einen Eintrag | Fehler |
| KB-07 | Jede Übung mit `training:`-Block hat `load_family`, `pattern`, `straight_arm`, `complaint_family`, `requires` | Fehler |
| KB-08 | Jede Stufe eines Ziel-Skills ist über ihren Primärtest einer Leitersprosse zugeordnet | Fehler |
| KB-09 | Wertebereiche: Anteile in (0, 1], Ratings 0–3, Pausen > 0, Vorlagen decken alle Minutenoptionen | Fehler |
| KB-10 | Jede Red-Flag hat Dringlichkeit N/D/A und Aktion; jede Region hat mindestens RF-01–RF-07 und RF-10 | Fehler |
| KB-11 | Texte: jeder Textschlüssel existiert in jeder ausgelieferten Sprache | Warnung (CI: Fehler) |
| KB-12 | Sprachregeln (§9.3): verbotene Begriffe in Texten | Fehler |
| KB-13 | `status: draft_placeholder` in Inhalten, die ein Plan benutzt | Warnung; in Produktion Fehler, solange ENT-10 nicht erfüllt ist |

Scheitert die Validierung beim API-Start, läuft die API weiter, aber alle
Planungs-Endpunkte antworten mit `503` und dem Problemtyp
`planning-unavailable` (§10.4). Logging, Sync und Skill-Karte sind nicht
betroffen. Kein `panic` (CLAUDE.md).

## 3. Skill-Graph

### 3.1 Knoten, Kanten, Gewichte

- **Knoten** sind Stufen (`skill_levels`); die Reihenfolge innerhalb eines
  Skills ist eine implizite `prerequisite`-Kante zur vorigen Stufe
  (`internal/domain/progress/states.go`). Das bleibt so (ENT-6). Geräte- und
  Variantenstufen, die keine harte Kette bilden (Stange vs. Ringe, Maltese-Halt
  vs. -Press), sind eigene Skills mit `recommended`-Kanten (`02` §2.5).
- **Kanten** (`skill_edges`): `prerequisite` (Gewicht 1.0, hart), `recommended`
  (0.3–0.7), `alternative` (0.5), `antagonist` (0.5) (PAR-A-62). Kraftbaselines
  sind nie `prerequisite` (PAR-A-43, PAR-C-38, `08` Rang 10).
- Die Kanten aus `02` §8 sind die Startmenge. Der Graph ist inklusive der
  impliziten Ketten zyklenfrei (`02` §8, `KB-04`).

### 3.2 Unlock-Kriterien

Die DSL (`progress.Criteria`) bleibt. Die Vorlage für Stufentypen kommt aus
`02` §2.4, die Schwellen aus `08` §4 (Rang 16):

| Stufentyp | Kriterium | Parameter |
|---|---|---|
| Dynamische Grundübung | `reps ≥ 8 · none · occ 3 · 7 d` | PAR-A-03, 18, 20 |
| Grund-Isometrie | `hold ≥ 30 s · none · occ 3 · 7 d` | PAR-A-07, 18, 20 |
| Zwischenstufe Hebel-Static | `hold ≥ 10 s · none · form ≥ 4 · occ 2 · 28 d` | PAR-A-10, 17, 18, 19 |
| Endstufe | `hold ≥ 3 s · none · form ≥ 4 · occ 2 · 28 d` | PAR-A-11, 17, 18, 19 |
| Press/Kraftskill | `reps ≥ 1 · none · form ≥ 4 · occ 2 · 28 d` | `02` §2.4 (H-UNL) |
| Gewichtete Stufe | `reps ≥ 1 · min_load_pct_bw X · occ 2 · 28 d` | PAR-A-33, 34 (Umrechnung unten) |
| Schwung/Release | nur Selbstbestätigung | PAR-A-60 |

Formwerte bilden Winkelabweichungen ab (5: ≤ 5°, 4: > 5–15°, 3: > 15–30°, 2:
> 30–45°, 1: > 45°; PAR-A-16); Armbeugung > 15° begrenzt die Form auf ≤ 3
(PAR-A-78). Assistierte, partielle, exzentrische und gescheiterte Elemente
zählen nie (PAR-A-21).

**DSL-Erweiterungen (ENT-5), additiv:**

| Feld | Semantik | Datenbedarf | Rückwärtskompatibilität |
|---|---|---|---|
| `min_distinct_days` (int ≥ 1) | Die qualifizierenden Beobachtungen müssen auf mindestens so vielen verschiedenen lokalen Kalendertagen liegen (`workout_sessions.local_date`). Behebt, dass `occurrences` Sätze zählt, nicht Tage (`02` §2.5). | `Observation.LocalDate` (neu) | fehlt das Feld, gilt 1 (heutiges Verhalten) |
| `min_load_pct_bw` (Zahl > 0) | Zusatzlast ≥ Wert/100 × Körpergewicht zum Zeitpunkt der Einheit (`workout_sessions.bodyweight_kg`, sonst jüngster Eintrag in `user_bodyweight_log` davor). Ohne bekanntes Körpergewicht qualifiziert die Beobachtung nicht (konservativ); das Ergebnis nennt den Grund. | `Observation.BodyweightKg` (neu, optional) | fehlt das Feld, keine Wirkung |

Umrechnung der OG-Gesamtlasten (PAR-A-33, 34: Vielfache des KG als
Systemmasse) in Zusatzlast: `min_load_pct_bw = (Vielfaches − 1) × 100`, z. B.
gewichteter Klimmzug Level 4 = 1.18 × KG → 18 %. Bis die Erweiterung
ausgerollt ist, gilt übergangsweise die absolute Last mit 75-kg-Referenz
(PAR-A-67) oder nur Selbstbestätigung (ENT-9).

Vorgehen nach ADR 0008: zuerst Golden Files mit Beispielen für beide Felder
(Tage an der Grenze, fehlendes Körpergewicht, Zeitzonen-Tageswechsel), dann
Schema (`criteria.schema.json`), dann Evaluator. Bestehende Unlocks werden nie
zurückgenommen, auch wenn Kriterien strenger werden (CLAUDE.md).

### 3.3 Leitern und Stufen

Eine Leiter ordnet Übungen einer Bewegung (`ladders.yaml`); die Stufen eines
Skills verweisen über ihre Primärtest-Übung auf eine Sprosse (`KB-08`).
Zwischen zwei Stufen können weitere Sprossen liegen (z. B. Band- oder
Lean-Varianten), die trainiert, aber nie freigeschaltet werden (`02` §2.1).
Die Schwierigkeit einer Sprosse steht als `og_ordinal` (PAR-A-22–38) und, wo
berechenbar, als Momentverhältnis (Planche, Front Lever: Tuck ≈ 0.60, Adv
Tuck ≈ 0.75, One-Leg ≈ 0.88, Half-Lay ≈ 0.93, Straddle 0.85–0.96, Full 1.0;
PAR-C-03–08) oder als KG-Anteil (Liegestütz-Leiter 0.41–0.74; PAR-C-18–22).
Planche ist bei gleicher Stufe ≈ 1.39× schwerer als der Front Lever (PAR-C-16);
das nutzt nur die Ersatzlogik, nicht die Dosierung.

### 3.4 Pfad zu einem Ziel (`GOAL-02`)

Eingabe: Ziel-Stufe `t`, Zustand. Ausgabe: die Menge der **aktiven Leitern**
mit Arbeitsstufe und Rolle.

1. Bestimme die Vorfahren von `t` über `prerequisite`-Kanten und implizite
   Vorstufen (topologisch, wie `progress.topoSort`).
2. Ein Vorfahr gilt **für die Planung** als erfüllt, wenn er freigeschaltet ist
   **oder** der Dosiswert (§4.8) seines Kriteriums den Schwellenwert erreicht
   (z. B. `pull-up/strict-5`: Dosiswert Klimmzug ≥ 5 Wdh.). Das ist eine
   Planungsentscheidung, kein Unlock (ADR 0008); die Skill-Karte zeigt weiter
   den Unlock-Status.
3. **Zubringer-Leitern** (Rolle `feeder`, Priorität des Ziels): jede Leiter
   eines unerfüllten Vorfahren, deren harte Vorfahren erfüllt sind. Leitern
   mit dem Kennzeichen `foundation` (die Wurzeln aus `02` §4: Handgelenk,
   Hang, Körperspannung, Stütz, Liegestütz, Dip, Rudern, Klimmzug,
   Kompression, Scapula) werden auch dann aktiv, wenn nur andere
   Grundlagenleitern fehlen: Ihre unteren Sprossen (Exzentrik, Band,
   Regression) sind Übungen, keine Stufen (`02` §2.1), und Grundlagen werden
   in der Praxis parallel aufgebaut (A-44).
4. Die Leiter von `t` selbst wird erst aktiv (Rolle `goal`), wenn alle harten
   Vorfahren ihrer ersten unerfüllten Stufe erfüllt sind; ihre Arbeitssprosse
   folgt §5.6.
5. `recommended`-Kanten mit Gewicht ≥ 0.5 (`PAR-S-37`) zu unerfüllten Stufen
   erzeugen **Zubringer-Übungen** im Kraft-/Zubringerblock (Rolle `support`),
   nie Sperren (PAR-A-62, PAR-F-42).

Beispiel Persona 1 (Muscle-up, 0–3 Klimmzüge, 0 Dips): Vorfahren
`pull-up/strict-5` und `dip/parallel-bars` (hart, `02` §8) und deren
Vorfahren (`hang-foundation/arch-hang`, `support-hold/parallel-bars`) sind
unerfüllt → Zubringer-Leitern Hang, Klimmzug (ab Exzentrik), Stütz und Dip,
alle als Grundlagenleitern parallel; die Muscle-up-Leiter wird erst aktiv,
wenn Klimmzug und Dip Dosiswerte ≥ 5 haben. Das folgt aus den harten Kanten
`pull-up/strict-5` und `dip/parallel-bars` → Muscle-up (`02` §8, PAR-A-69);
die höheren Coaching-Schwellen bleiben weiche Hinweise (`08` §5).

### 3.5 Weiche Bereitschaftshinweise (`GOAL-04`)

Hinweise aus `readiness_hints.yaml` (z. B. Front Lever: 10 Klimmzüge, 30 s
Totehang, 60 s Hollow; Planche: 3 × 20 Liegestütze, 60 s Hollow; Muscle-up:
5 + 5 als Minimum, Coaching-Spanne 8–18 Klimmzüge und 8–25 Dips) werden
angezeigt, **sperren aber nie** (PAR-F-42 `soft`; PAR-A-52, 53, 69,
PAR-F-62–64; `08` §5). Sie werden als Erfahrungswerte gekennzeichnet
(Evidenz C/D).

### 3.6 Realismus-Check (`GOAL-05`)

Aus `onboarding.md` §6, hier als Formel:

- Aktuelle und Ziel-Stufe auf OG-Ordinalen `o_a < o_t` (PAR-A-22–38).
- Für jeden Schritt `o → o+1` das Wochenband `[lo(o+1), hi(o+1)]` aus
  PAR-A-45 (Ziel-Level ≤ 4: 2–8; 5–8: 4–13; 9–12: 8–26; ≥ 13: 13–52 Wochen).
- **Untergrenze** `U = Σ lo` (PAR-A-51). **Angezeigte Spanne**
  `[Σ (lo+hi)/2, Σ hi]` — die obere Hälfte der Bänder, weil PAR-A-45 am
  optimistischen Ende der Coaching-Angaben liegt (`02` §3.6, `08` Rang 19).
- Liegt `target_date` vor `now + U`: Spanne anzeigen und die nächste Stufe
  des Ziel-Skills auf dem Pfad als Etappenziel vorschlagen, mit ihrer eigenen
  Spanne. Das Ziel wird nie abgelehnt; der Plan richtet sich nach der
  Arbeitsstufe, nicht nach dem Datum (O-8).
- Prior-Faktoren für Grösse und Geschlecht (PAR-C-32, PAR-C-54) sind in v1
  **abgeschaltet** (`PAR-S-19`): Dass die relative Hebelanforderung die Lernzeit
  proportional verlängert, ist nicht belegt.
- Text ohne Wertung (ADR 0003 §3), mit Kennzeichnung «Erfahrungswerte aus dem
  Coaching, keine Prognose».

Rechenbeispiel Persona 5 (Anfänger, Full Planche in 8 Wochen): OG 0 → 11
(PAR-A-25) ergibt `U` = 4 × 2 + 4 × 4 + 3 × 8 = 48 Wochen (PAR-A-51) und eine
angezeigte Spanne von 105–162 Wochen; Coaching nennt 24–36 Monate (PAR-A-47).
Die Tuck Planche (OG 5) hat `U` = 4 × 2 + 4 = 12 Wochen und eine angezeigte
Spanne von 28.5–45 Wochen. Das liegt über der Coaching-Angabe «Tuck nach 2–6
Monaten» (PAR-A-47): Für frühe Stufen ist die obere Bandhälfte eher
pessimistisch. Die Bänder werden aus eigenen Logs nachgeschärft (`08` §6.1
Nr. 5).

## 4. Nutzerzustand

### 4.1 Übersicht

| Teil | Inhalt | Quelle | Aktualisiert bei |
|---|---|---|---|
| Profil | Verfügbarkeit, Minuten, Equipment, Körpergewicht, `is_minor`, Trainingsniveau, Trainingsalter, Pausenklasse, Mobilitäts-Checks, Einwilligungen | Onboarding Blöcke A–F, «Profil verfeinern» | Profiländerung |
| Ziele | Skill, Ziel-Stufe, Priorität 1–3, Datum, Etappenziel | Block B, §3.6 | Zieländerung |
| Kapazitäten | je Übung × Messgrösse × Assistenz: μ, σ, Herkunft, Zeitpunkt, Anzahl Beobachtungen | §4.3 | jede Einheit |
| Leiterstand | je aktiver Leiter: Arbeitssprosse, Status `claimed`/`calibrated`, seit wann, Prüfsprosse, Anzahl Einheiten an der Sprosse | §4.4 | jede Einheit |
| Belastungshistorie | Belastungseinheiten je Lastkonto, Tag und Woche; 30-Tage-Maximum je Struktur | §4.5, aus Logs berechnet | jede Einheit |
| Regionen (Toleranz) | Zustand je Region, Rampenstufe, Referenzvolumen, Vorverletzung, Einschränkungen, Schmerzverlauf | §4.6, Block G, Schmerzberichte | Schmerzbericht, Red Flag, Freigabe, Pause |
| Screening | `clear` · `clearance_recommended` · `stop` | Block G | Onboarding, Freigabe |
| Trainingsphase | Mesozyklus-Woche, letzter Deload (Datum, Art), Pausenklasse je Konto, fällige Kalibrierungen | §4.7 | Wochenwechsel, Einheit |
| Konfidenz | Klasse je Kapazität | §4.8 | mit der Kapazität |

Der Kern erhält alles zusammen als **Snapshot** (unveränderlicher Wert). Der
Snapshot ist der einzige Eingang ausser Wissensbasis und Stichtag; sein
kanonisches JSON ergibt den `input_hash` (§1.3).

### 4.2 Profil, Ziele, Equipment, Verfügbarkeit

Die Felder stehen in `onboarding.md` §3 und §7. Abgeleitet werden:

| Ableitung | Regel | Grundlage |
|---|---|---|
| `is_minor` | laufendes Jahr − `birth_year` ≤ 18 (nur das Jahr ist bekannt; die Regel zählt im Zweifel als minderjährig) | PAR-D-23 |
| Erfahrungsklasse | `novice`: Trainingsalter `lt_6_months` oder Niveau `sedentary`; `advanced`: `gt_4_years` und `highly_trained`; sonst `intermediate`. Das Trainingsalter wächst ab dem Onboarding um jeden Monat mit ≥ 4 geloggten Einheiten (untere Klassengrenze als Start) | `PAR-S-20`; Grenzen PAR-B-01 (6 Monate), PAR-F-48 |
| Periodisierungsmodell | `novice` → lineare Doppelprogression; sonst wochenweise wellenförmig | PAR-B-01, PAR-B-02 |
| Risikofenster | fortgeschriebenes Trainingsalter 6–48 Monate | PAR-D-04 |
| Equipment-Menge | Auswahl plus implizite Geräte (`outdoor_park`, `gym`) | `onboarding.md` §3.4 |
| Volle Einheiten je Woche | höchstens 3 (`novice`), 4 (`intermediate`), 5 (`advanced`); weitere Tage werden leichte Einheiten oder geplante Ruhe | PAR-B-36 (Obergrenzen der Spannen), `onboarding.md` §3.3 |

Die Erfahrungsklasse steuert nur Wochenstruktur und Periodisierung. Die
Einstufung je Skill kommt aus der Kapazität (O-2, PAR-F-43, PAR-F-70).

### 4.3 Kapazität mit Unsicherheit

**Schlüssel:** (Übung, Messgrösse, Assistenz). Assistenz ist `none`,
`band:<Band-ID>` oder `load` (Zusatzlast). Assistierte Kapazitäten sind
getrennte Schätzungen; sie zählen nie für die unassistierte Kapazität
(`onboarding.md` §5.4, PAR-A-21).

**Startwerte** aus dem Onboarding nach `onboarding.md` §5.2 (Klassenmitte
× 0.95 für erinnerte Wdh., PAR-F-55; untere Klassengrenze für Halte; σ nach
PAR-F-20, PAR-F-21, PAR-F-56; Umrechnungen mit σ ≥ 0.35 μ, PAR-F-26;
Verbreiterung × 1.25 in den dort genannten Fällen). Für nach unten offene
Halteklassen («< 4 s», «< 10 s», «< 15 s») ist μ die halbe Obergrenze; für
alle Halte gilt σ ≥ 3 s, für Wdh. σ ≥ 2 Wdh. (PAR-F-20), und jeder
Beobachtungsfehler r ≥ 1 s bzw. 1 Wdh. (`PAR-S-38`). Eine Kapazität von 0
(z. B. 0 Klimmzüge) ist gültig; ihre Konfidenzklasse ist «niedrig».

**Abgeleitete Startwerte für nie beobachtete Sprossen** (`PAR-S-39`). Eine
Umrechnung zwischen Hebelstufen über Haltezeit-Intensitäts-Modelle ist
ausgeschlossen (`08` §4). Der Planer nutzt nur die sichere Richtung:
- Eine **leichtere** Sprosse oder die **Band-Variante** derselben Sprosse
  bekommt μ = μ der schwereren bzw. unassistierten Sprosse (Untergrenze),
  σ = max(3 s, 0.35 μ) (PAR-F-26), Herkunft `derived`; der erste Satz ist ein
  Kalibrierungssatz. Ist auch diese Kapazität 0, beginnt der Kalibrierungssatz
  mit 5 s (Untergrenze von PAR-B-10) bzw. 3 Wdh.
- Eine **schwerere** Sprosse hat keinen Startwert; sie wird erst über
  Prüfversuche (ADAPT-05) beobachtet und ist bis dahin nie Arbeitssprosse.

**Fortschreibung** als eindimensionaler Kalman-Filter (`07` §8.1–8.3;
**Heuristik**, kein für Trainingsdaten validiertes Verfahren):

1. *Vorhersage* seit der letzten Beobachtung (Δw Wochen):
   σ² ← σ² + q² · Δw mit q = 0.5 Wdh. je Woche (PAR-F-28) bzw. q = 0.05 · μ
   s je Woche für Halte (`PAR-S-04`). μ bleibt.
2. *Beobachtung* x mit Fehler r:

| Beobachtung | x | r | Grundlage |
|---|---|---|---|
Die Zeilen gelten **in dieser Reihenfolge**; die erste zutreffende gewinnt.

| Beobachtung | x | r | Grundlage |
|---|---|---|---|
| Form < 3, assistiert (für den `none`-Schlüssel), partiell, nur exzentrisch | keine Beobachtung | – | PAR-F-41; `07` §8.2 |
| nicht der erste Arbeitssatz der Übung in der Einheit (auch wenn gescheitert) | nur **Untergrenze** b | – | PAR-E-31, E-28 |
| Test (`kind = test`) oder Satz mit `failed = true` | geschaffte Wdh. bzw. Sekunden | Wdh.: 2.0; ≤ 5 Wdh.: 1.0. Halt: Anteil von μ wie unten | PAR-F-01, 02, 16, 22 |
| Log-Satz, Wdh., RIR ≤ 3 und ≤ 12 Wdh. | Wdh. + RIR (ohne Bias-Korrektur) | 2.0 Wdh. | PAR-F-23, 24, 25 |
| Log-Satz, Wdh., RIR ≤ 3 und > 12 Wdh. | nur Untergrenze b = Wdh. + RIR | – | PAR-F-23 (gilt nur bis 12 Wdh.) |
| Log-Satz, Halt, SIR ≤ max(3 s, 0.5 × Halt) | Halt + SIR | Anteil von μ: Skill-Statics 0.25 (`PAR-S-03`), Gleichgewicht 0.25, Rumpfbeuger 0.40, Ausdauerhalte 0.15 | PAR-F-16, PAR-F-68, `PAR-S-31` |
| RIR > 3 bzw. SIR über der Grenze der Zeile davor | nur Untergrenze b = Wdh. + 3 bzw. Halt + max(3 s, 0.5 × Halt): die Reserve zählt bis zu ihrer glaubwürdigen Grenze | – | PAR-F-24, `PAR-S-31` |
| alle übrigen (ohne RIR/SIR) | nur Untergrenze b = Wdh. bzw. Halt | – | PAR-F-24 |

3. *Update*: K = σ² / (σ² + r²); μ ← μ + K · (x − μ); σ² ← (1 − K) · σ².
   Eine Untergrenze b wirkt nur, wenn b > μ: dann wie eine Beobachtung x = b
   mit dem Fehler r der Übungsart (`PAR-S-31`).
4. *Widerspruch* (`ADAPT-03`): Ist |x − μ| > 2 · √(σ² + r²) (PAR-F-32), wird
   nicht überschrieben: μ ← min(μ, x), σ ← max(σ, r) (PAR-F-33), und der
   nächste erste Arbeitssatz wird ein Kalibrierungssatz. Bestätigt die nächste
   Beobachtung die Abweichung in dieselbe Richtung, wird die Schätzung auf das
   Mittel beider Beobachtungen mit σ = r zurückgesetzt (`PAR-S-21`).
5. Nur der **erste Arbeitssatz** einer Übung in einer Einheit ist eine volle
   Beobachtung; spätere Sätze sind ermüdet und liefern nur Untergrenzen
   (PAR-E-31, E-28). Mit der SIR-Grenze max(3 s, 0.5 × Halt) sind auch die
   kurzen Sätze an der Grenze des Arbeitsfensters (h = d − 2 bei d < 6 s) und
   Kalibrierungssätze mit SIR 2–3 volle Beobachtungen; die Schätzung friert
   dort nicht ein.

**Messung von Reserve bei Halten.** Das Log kennt heute nur `rir` (0–10,
Wiederholungen). Für Halte braucht der Planer Sekunden in Reserve (SIR,
`03` §5.3). Vorschlag: neue, optionale Spalte `set_entries.sir_s` (§4.9,
ENT-S-4).

### 4.4 Leiterstand

Je aktiver Leiter: Arbeitssprosse, Status `claimed` (aus dem Onboarding, noch
nicht kalibriert) oder `calibrated` (mindestens eine volle Beobachtung an der
Sprosse), Datum des Sprossenwechsels, optionale Prüfsprosse (§6.3), Zahl der
Einheiten an der Sprosse. Ein Sprossenwechsel ändert nie einen Unlock (ADR
0008).

### 4.5 Belastungshistorie und Belastungseinheiten (`LOAD-01`)

Die Last eines geloggten oder geplanten Satzelements `e` auf eine Struktur
`s` ist

```
u(e, s) = w_kind(e) · min(3, r_eff(e, s) · k(e)) / 3
```

| Grösse | Definition | Grundlage |
|---|---|---|
| `r_eff(e, s)` | Profilwert 0–3 der Lastfamilie der Übung (`04` Tabelle 2), plus Modifikatoren: Neutralgriff/Parallettes −1 Handgelenk, Ringe +1 Bizeps und vordere Schulter bei Stützübungen, weiter Griff oder Untergriff +1 Überkopf; gekappt auf 0–3 | PAR-C-27, 40, 44, 45, 47 |
| `k(e)` | Statik: Momentverhältnis der Sprosse ÷ Momentverhältnis der Referenzsprosse der Familie (die Tabelle gilt für die schwerste übliche Stufe); dynamisch: KG-Anteil ÷ Referenz-KG-Anteil × (KG + Zusatzlast) / KG; ohne Daten 1.0 | PAR-C-03–10, 18–22, 44; `04` §3.8 |
| `w_kind(e)` | 1.0 für `working`, `test`, `backoff`, `cluster`, `drop`; 0.5 für `warmup` (Rampensätze haben höchstens die halbe Satzhaltezeit bzw. Ziel-Wdh., SESS-03) | `PAR-S-08` |
| Assistenz | zählt voll (keine Minderung durch das Band) | PAR-B-79, H-3 (`08` §3) |
| Kombination | jedes Element eines `set_entry` trägt seine Last | CLAUDE.md (ein Codepfad) |

Die Last ist **satzbasiert** wie in `05` §5.4 («Summe der Arbeitssätze ×
Strukturgewicht»); die Haltedauer geht nicht ein. Innerhalb des Arbeitsfensters
von 4–20 s (`08` §4) ist das vertretbar, weil die Satzzahl begrenzt ist
(§7.6); als Schwäche vermerkt (§14).

**Bewusste Abweichung:** PAR-D-31 gilt laut `05` für das Volumen, «nicht für
die Stufe». Hier geht die Sprosse über k in die Last ein; ein Sprossenwechsel
zählt damit als Laststeigerung und kann über die Deckel Sätze kosten. Das ist
strenger als `05` und setzt S-2 um (neue Stufen sind nicht sofort voll
belastbar, PAR-D-06).

**Lastkonten.** Für den Wochendeckel wird jede Struktur ausser `wrist` in zwei
Konten geführt: `s/SA` (Beiträge von Übungen mit `straight_arm` oder
`straight_arm_axial`) und `s/BA` (alle übrigen); `wrist` hat ein Konto
(`PAR-S-15`). So gilt der strengere Deckel für gestreckte Arme, ohne
Bent-Arm-Arbeit an derselben Struktur mit auszubremsen (§7.2).

**Aggregate** (aus den Logs der letzten 6 Wochen berechnet, PAR-B-73):

| Aggregat | Definition | Grundlage |
|---|---|---|
| `W(a, w)` | Summe von u über alle Sätze des Kontos `a` in ISO-Woche `w` | `05` §5.4 |
| `R(a)` | Mittel von `W` über die letzten bis zu 3 abgeschlossenen Wochen **mit Last > 0, ohne geplante, Stagnations- und Ermüdungs-Deloads**, innerhalb der letzten 6 Wochen; Wochen mit Schmerz-Deload zählen (§7.2) | PAR-D-09–11 (3-Wochen-Mittel), PAR-B-73, `PAR-S-14` |
| `M(s)` | grösste Einheitslast der Struktur `s` (Summe beider Konten) in den letzten 30 Tagen | PAR-D-31 |
| neu | `R(a)` bzw. `M(s)` = 0 oder nicht bestimmbar | PAR-D-12 |

Geplante Deload-Wochen fallen aus `R` heraus, weil ein Deload eine gewollte,
zeitweise Senkung ist (`08` Rang 12) und sonst jeden folgenden Deckel senken
würde. Ein Schmerz-Deload dagegen senkt die Last, weil die vorherige Last nicht
vertragen wurde; er bleibt in `R` (§7.2).
Wochen ohne Last fallen heraus, weil Pausen eigene Regeln haben (§6.11); eine
einzelne ausgelassene Woche soll nicht wie ein Wiedereinstieg wirken
(PAR-B-59: bis 2 Wochen Pause 90–100 % Volumen).

### 4.6 Regionen, Beschwerden und Einschränkungen

Je Region (`onboarding.md` §3.7):

| Feld | Werte | Grundlage |
|---|---|---|
| `state` | `normal` · `rtt_0` … `rtt_5` · `locked` | `05` §6.2, PAR-D-21 |
| `entered_via` | `onboarding` · `pain_report` · `red_flag` · `break` · `clearance` | – |
| `ramp_accounts` | Lastkonten, für die die Rampe gilt: bei Beschwerde und Red Flag alle Konten der Strukturen der Region; bei `break` nur die Straight-Arm-Konten und `wrist` | §6.11, PAR-D-29 |
| `start_fraction`, `ramp_step` | Startanteil 0.5 (leichte Beschwerde) oder 0.25 (nach Verweis, Sehnenbefund oder Pause ≥ 4 Wochen); aktueller Schritt der Reihe 0.25 → 0.5 → 0.75 → 1.0 | PAR-D-24, PAR-D-25, PAR-D-33, PAR-D-29 |
| `reference_volume` | `R(a)` der Rampenkonten vor der Beschwerde bzw. Pause, **nur wenn geloggt**; ohne geloggte Historie gibt es keine Referenz, und die Rampe wirkt nur als zusätzliche Obergrenze auf LOAD-02/LOAD-04 (§7.2) | `05` §6.2 («nur auf früher toleriertes Niveau») |
| `prior_injury` | Verletzung in den letzten 12 Monaten | PAR-F-47, PAR-D-02 |
| `restrictions` | Bewegungskategorien einer Fachperson | `onboarding.md` §3.7 |
| Schmerzverlauf | Berichte (NRS 0–10) je Zeitpunkt, Regelverletzungen der letzten 14 Tage, Beginn der Beschwerde | PAR-D-13–20, 32 |

Zuordnung Region → Strukturen (`structures.yaml`, `PAR-S-15`):

| Region | Strukturen |
|---|---|
| `shoulder_front` | `biceps_long_head_anterior_shoulder`, `shoulder_extension` |
| `shoulder_top_side` | `shoulder_overhead` |
| `elbow_inner` | `elbow_medial` |
| `elbow_outer` | `elbow_lateral` |
| `elbow_crease` | `biceps_distal` |
| `wrist_back_extension`, `wrist_pinky_side` | `wrist` |
| `chest` | `biceps_long_head_anterior_shoulder` (ENT-S-9; Zuordnung und Matrixzeile wie `shoulder_front`, **Heuristik** bis zur fachlichen Prüfung) |
| `fingers_forearm_inner` | `fingers_forearm` |
| `lower_back` | `lumbar` |
| `knee` | `knee` |
| `other` | keine Struktur. Beschwerde ohne Red Flag: der User schliesst Übungen selbst aus. Red Flag N/D oder Selbsteinschätzung «ernst»: Planung pausiert, bis der User betroffene Übungen ausgeschlossen oder eine Freigabe bestätigt hat (INJ-02) |

### 4.7 Trainingsphase

Mesozyklus-Woche (1–6), Datum und Art des letzten Deloads (`planned`,
`stagnation`, `fatigue`, `pain`), Pausendauer je Lastkonto (Tage seit der
letzten Einheit mit Last > 0; ohne Logs aus `last_regular_training`, §6.11),
fällige Kalibrierungen je Kapazität, nicht verbrauchter Satz-Spielraum je
Konto (`PAR-S-35`, §7.2).

**Minimale Planungsauflagen** (`planning_constraints`): Stopps und
Regionen-Ausschlüsse, die aus Sicherheitsfragen folgen, werden als Auflage
ohne Antworten und ohne Schmerzwerte gespeichert (Art, Region, Zeitpunkt,
aufgehoben am). So wirken sie auch ohne Einwilligung für Gesundheitsdaten
weiter (SAFE-02, SAFE-04); ob das ohne Einwilligung zulässig ist, ist Teil der
rechtlichen Prüfung (ENT-4, ENT-S-7).

### 4.8 Konfidenz und Dosiswert

| Klasse | σ / μ | Dosiswert d | Zusatz | Grundlage |
|---|---|---|---|---|
| hoch | σ/μ < 0.15 | μ | – | PAR-F-30, PAR-F-31 |
| mittel | 0.15 ≤ σ/μ < 0.30 | μ − 0.5 σ | – | PAR-F-30, PAR-F-31 |
| niedrig | σ/μ ≥ 0.30 oder μ ≤ 0 | μ − σ | erster Arbeitssatz ist ein Kalibrierungssatz mit RIR/SIR 2–3 | PAR-F-30, PAR-F-31; `onboarding.md` §4.1 |

d wird bei 0 abgeschnitten. Die Klasse erscheint in den Begründungen («Dein
Wert ist noch geschätzt, deshalb beginnen wir vorsichtig»).

### 4.9 Persistenz (Skizze für die Migration)

Alle Tabellen folgen CLAUDE.md: UUIDv7 aus Go, `text` + benannter `CHECK`,
`user_id` auf jeder Zeile, Kinder über `(id, user_id)`, Sync-Spalten, wo der
Client offline schreibt. Die DDL entsteht als eigene, additive Migration
(`make migrate-new name=planning`); diese Skizze legt nur Inhalte fest.

| Tabelle | Wichtige Spalten | Schreibt | Sync | Einwilligung |
|---|---|---|---|---|
| `user_training_profiles` | `user_id` (PK), `birth_year`, `sessions_per_week` (1–7), `session_minutes` (20, 30, 45, 60, 75, 90), `training_level`, `training_age`, `last_regular_training`, `pre_break_level` (jsonb), `data_confidence`, `mobility` (jsonb), `height_cm`, `sex`, `preferred_days` (smallint[] 1–7), `secondary_focus`, `health_data_consent`, `consent_at`, `disclaimer_ack_at`, `onboarding_completed_at` | Client/API | ja | – |
| `user_goals` | `skill_id`, `target_level_id`, `priority` (1–3, eindeutig je User für aktive Ziele, `DEFERRABLE`), `target_date`, `milestone_level_id`, `status` (`active`, `achieved`, `dropped`) | Client/API | ja | – |
| `user_equipment` | (`user_id`, `equipment_key`) | Client/API | ja | – |
| `user_bands` | Bezeichnung, Herstellerangabe kg, `estimated_assist_kg` | Client/API | ja | – |
| `user_region_status` | (`user_id`, `region`), `state`, `since`, `entered_via`, `start_fraction`, `reference_volume` (jsonb), `prior_injury`, `restrictions` (text[] mit CHECK) | Server | nur lesen | ja |
| `user_pain_reports` | `region`, `timepoint` (`before_session`, `warmup`, `during`, `after`, `next_morning`, `daily`), `nrs` (0–10), `lasted_over_1h` (bei `after`), `persisted_over_15min` (bei `warmup`), `sudden_sharp` (Bool), `session_id` (optional), `reported_at`, `local_date` | Client | ja | ja |
| `user_red_flag_answers` | `region`, `flag_id` (`RF-01` … `RF-13`), `answer`, `answered_at` | Client/API | ja | ja (ohne Einwilligung nicht gespeichert, `onboarding.md` §3.1) |
| `planning_constraints` | `kind` (`plan_stopped`, `region_locked`, `region_excluded`), `region`, `created_at`, `cleared_at`; keine Antworten, keine Werte | Server | nur lesen | nein (Auflage, ENT-S-7) |
| `user_screening` | Frage, Antwort, Zeitpunkt, `clearance_confirmed_at` | API | nein | ja |
| `user_capacity_estimates` | (`user_id`, `exercise_id`, `measure`, `assistance_key`), `mu`, `sigma`, `origin`, `observed_at`, `n_obs`, `pending_contradiction` (jsonb) | Server | nur lesen | – |
| `user_ladder_states` | (`user_id`, `ladder`), `rung_exercise_id`, `status`, `since`, `probe_exercise_id`, `exposures` | Server | nur lesen | – |
| `training_plans` | `week_start`, `ruleset_version`, `content_version_id`, `input_hash`, `status` (`active`, `superseded`, `expired`); höchstens ein aktiver Plan je User und Woche | Server | nur lesen | – |
| `planned_sessions` | `plan_id`, `order_index` (`DEFERRABLE`), `scheduled_date`, `earliest_start`, `kind` (`full`, `light`, `deload`), `est_minutes`, `payload` (jsonb nach OpenAPI-Schema `PlannedSession`), `status` (`planned`, `started`, `completed`, `expired`), `workout_session_id` | Server | nur lesen | – |
| `plan_decisions` | `occurred_at`, `trigger`, `source_id` (z. B. Session-ID; eindeutig mit `trigger` für Idempotenz), `rule_id`, `payload` (jsonb: vorher/nachher, Reasons), `plan_id` | Server | nur lesen | – |
| `planner_knowledge` | `content_version_id` (PK), `ruleset_version`, `document` (jsonb), `checksum` | Seed | – | – |
| `workout_sessions` | **neu**: `planned_session_id` (optional, FK über `(id, user_id)`) | Client/API | ja | – |
| `set_entries` | **neu**: `sir_s` (optional, 0–60), `planned_item_id` (optional; Verweis auf das Plan-Item, §10.2) | Client | ja | – |

Gesundheitsangaben (Tabellen mit «ja» in der Spalte Einwilligung) werden beim
Widerruf der Einwilligung gelöscht und mit dem Konto kaskadiert. Aufbewahrung
und Rechtsgrundlage sind rechtlich zu klären (ENT-4, OE-2); bis dahin gilt:
nur speichern, was eine Planentscheidung braucht.

## 5. Plangenerierung

### 5.1 Ablauf

```
Generate(kb, snapshot, week_start) -> Plan
  1. SAFE   Sicherheits-Gate                         §5.2
  2. GOAL   Ziele -> aktive Leitern, Zubringer       §5.3
  3. WEEK   Tage, Einheitsarten, Reizverteilung      §5.4
  4. SESS   Blöcke und Reihenfolge je Einheit        §5.5
  5. SEL    Übung je Platz                           §5.6
  6. DOSE   Sätze, Wdh./Halt, Reserve, Tempo, Pause  §5.7
  7. LOAD   Deckel und Abstände prüfen, kürzen       §7
  8. TIME   Zeitbudget prüfen, kürzen                §5.8
  9. EXPL   Begründungen, input_hash                 §9
```

Schritte 7 und 8 laufen in einer Schleife, bis beide erfüllt sind; jede
Kürzung entfernt mindestens einen Satz oder senkt eine Sprosse, daher endet
die Schleife (§11).

### 5.2 Sicherheits-Gate

| Regel | Bedingung | Folge | Grundlage |
|---|---|---|---|
| SAFE-01 | Onboarding unvollständig oder Hinweis nicht bestätigt | kein Plan; Problem `onboarding-required` | `onboarding.md` §3.1 |
| SAFE-02 | Belastungssymptome «ja» (im Onboarding oder jederzeit über «Symptome beim Training melden») oder aktive Red Flag mit Dringlichkeit N (RF-05 bei sichtbar verschobenem Gelenk, RF-07, RF-08, RF-10) | kein Plan; ärztliche Abklärung empfehlen; die betroffene Region zusätzlich `locked`; Auflage `plan_stopped`; Plan erst nach bestätigter Freigabe; Problem `training-stopped` | S-1 (`08` §2.2), PAR-F-45, PAR-D-21, `05` §9 |
| SAFE-03 | Screening mit «ja» ohne bestätigte Freigabe | Plan ohne Tests und Prüfversuche; Abklärung empfehlen | `onboarding.md` §3.7, F-41 |
| SAFE-04 | keine Einwilligung für Gesundheitsdaten | Plan ohne Tests und Prüfversuche; alle Steigerungsdeckel × 0.5; eine gemeldete aktuelle Beschwerde schliesst die Region wie `rtt_0` aus (Auflage `region_excluded`), weil ohne Einwilligung kein Schmerz-Monitoring möglich ist; Hinweis darauf | `onboarding.md` §3.1, PAR-D-02 |
| SAFE-05 | Region `locked` | Übungen mit `r_eff` ≥ 1 auf einer Struktur der Region sind ausgeschlossen | PAR-D-21, `PAR-S-22` |
| SAFE-06 | Region `rtt_0` | Übungen mit `r_eff` ≥ 2 auf einer Struktur der Region sind ausgeschlossen | `05` §6.2, `PAR-S-22` |
| SAFE-07 | `is_minor` | kein Plan, solange nicht entschieden ist, ob Minderjährige zugelassen werden (ENT-3, OE-1); danach §8.8 | ENT-3 |

«Maximalversuch» meint in dieser Spezifikation Tests (`kind = test`) und
Prüfversuche einer Stufe (`onboarding.md` §4.3). Die Sätze des Maximalblocks
(§5.7) enden immer mit Reserve (PAR-B-27, PAR-E-18) und sind keine
Maximalversuche in diesem Sinn.

### 5.3 Ziele, Leitern und Zubringer

| Regel | Inhalt | Grundlage |
|---|---|---|
| GOAL-01 | Ziele nach Priorität; höchstens 3 | `onboarding.md` §3.2 |
| GOAL-02 | Pfad je Ziel → aktive Leitern mit Rolle `goal`, `feeder` oder `support` (§3.4) | `02` §8, PAR-A-62 |
| GOAL-03 | Reicht die Woche nicht für alle Ziele in Mindestfrequenz (WEEK-06), bekommt das Ziel mit der niedrigsten Priorität eine Erhaltungsdosis: 1 Einheit/Woche, 1–2 Sätze, gleiche Sprosse; der User erfährt warum | R-9 (`onboarding.md` §5.5), PAR-B-63, PAR-B-34 |
| GOAL-04 | Weiche Bereitschaftshinweise anzeigen, nie sperren | PAR-F-42, §3.5 |
| GOAL-05 | Realismus-Check bei Datum | §3.6 |
| GOAL-06 | Gegenspieler: Enthält die Woche nur Druck- oder nur Zug-Skills, kommt je Einheit ein Bent-Arm-Paar der Gegenrichtung in den Kraftblock | Vorlagen mit Antagonisten-Paaren PAR-B-64–66; `02` §8 (`antagonist`) |

### 5.4 Wochenstruktur

**Tage (WEEK-01).** Bevorzugte Tage, sonst das Standardmuster für n
Einheiten (`PAR-S-06`): 1: Mi · 2: Mo, Do · 3: Mo, Mi, Fr · 4: Mo, Di, Do, Sa
· 5: Mo, Di, Mi, Fr, Sa · 6: Mo–Sa · 7: Mo–So. Die Muster maximieren den
kleinsten Abstand zwischen Einheiten.

**Einheitsarten (WEEK-02).** Aus den n Tagen werden so viele `full`-Einheiten
wie die Erfahrungsklasse erlaubt (§4.2); gewählt wird die Teilmenge mit dem
grössten kleinsten Abstand (bei Gleichstand die früheste). Übrige Tage werden
`light`-Einheiten, wenn ein Balance-Skill geplant ist oder Mobilitätsbedarf
besteht, sonst **geplante Ruhe**. Der Planer schreibt `user_training_days`
nicht selbst (ADR 0008 bleibt unverändert): Die App zeigt den Tag als
geplanten Ruhetag und bietet das bestehende Ruhetag-Loggen mit einem Tipp an;
so hält er den Streak (ADR 0003 §1). Ob geplante Ruhetage automatisch zählen
sollen, ist ENT-S-8.

**Split (WEEK-03).** Bis 3 volle Einheiten Ganzkörper; ab 4 ergibt sich die
Aufteilung aus den Abstandsregeln (§7.5): Straight-Arm-Skills und die
Bent-Arm-Arbeit an denselben Strukturen landen auf denselben Tagen, die übrigen
Tage tragen Beine, Rumpf, Balance und leichte Technik (PAR-B-37; volumengleich
ist die Aufteilung ohne Einfluss, B-49).

**Frequenz je Leiter (WEEK-04).**

| Leiter | Ziel-Einheiten je Woche | Grundlage |
|---|---|---|
| `limiting_factor = strength` oder `mixed` | `novice` 2, sonst 3; höchstens 4 | PAR-B-34, PAR-E-11, `08` §4 |
| `limiting_factor = balance` | 4 (auch in `light`-Einheiten) | PAR-E-12, `08` §4 |
| Zubringer (Bent-Arm) | wie Kraft: `novice` 2, sonst 3 | PAR-B-34 |
| Erhaltung (GOAL-03) | 1 | PAR-B-63 |

**Verteilung (WEEK-05), deterministisch und gierig.**

1. Einheiten der Verteilung sortieren: Priorität, Rolle (`goal` vor `feeder`
   vor `support`), Slug.
2. Für jede Einheit und jede benötigte Exposition den Tag wählen, der
   a) eine `full`-Einheit ist (harte und mittlere Reize nur dort; Balance und
   Technik auch in `light`),
   b) die Abstände aller Strukturen mit `r_eff` ≥ 2 der Arbeitssprosse einhält
   (§7.5),
   c) höchstens zwei Straight-Arm-Skills gleicher Richtung und je Skill einen
   Maximalblock enthält (PAR-B-48),
   d) bevorzugt schon einen Straight-Arm-Skill der Gegenrichtung trägt
   (Paarung, PAR-B-81),
   e) den kleinsten Abstand zu den übrigen Expositionen derselben Leiter
   maximiert,
   f) bei Gleichstand der früheste ist.
3. Findet sich kein Tag, bekommt die Leiter weniger Expositionen; fällt sie auf
   0, greift GOAL-03 (WEEK-06).

**Deload-Woche (WEEK-07).** In Mesozyklus-Woche 6 oder nach einem Auslöser
(§6.9) sind alle Einheiten `deload`: gleiche Tage, Übungen und Frequenz;
Sätze × 0.6 (abgerundet, mindestens 1 — Ausnahme von `PAR-S-18`, weil die
Übungen gleich bleiben, PAR-B-54); Reserve +2 (RIR bzw. SIR); Sprosse halten
(eine leichter nur beim Schmerz-Deload) (PAR-B-50–54, PAR-B-03). Eine im
Deload abgeschlossene Einheit wird beim Abschluss als Deload-Tag erfasst
(`user_training_days.deload`); der Streak zählt sie (ADR 0003).

**Neu erzeugen (WEEK-08).** Zu Wochenbeginn, bei Profil- oder Zieländerung
(ab der nächsten nicht begonnenen Einheit), nach Freigabe einer Region und bei
neuer `ruleset_version` bzw. Content-Version (zu Wochenbeginn). Abgeschlossene
Einheiten bleiben unverändert.

### 5.5 Aufbau einer Einheit

**Vorlage (SESS-01)** nach `session_minutes` (PAR-B-64–67; `onboarding.md`
§3.3):

| Minuten | Aufwärmen | Blöcke | Arbeitssätze | Grundlage |
|---|---|---|---|---|
| 20 | 5 min | Primärblock + 1 Paar | – | Minimaldosis, **Heuristik** (`onboarding.md` §3.3) |
| 30 | 5 min | Primärblock 2–3 Sätze; 1–2 Antagonisten-Paare × 2–3 Runden | 8–12 | PAR-B-64 |
| 45 | 7 min | Skill 5 min; Primär 3–4; 2 Paare; optional 1 Ergänzung | 12–16 | PAR-B-65 |
| 60 | 10 min | Skill 8–10 min; Primär 3–5; Sekundär 3–4; 1 Paar; 1–2 Ergänzung/Prehab | 15–20 | PAR-B-66 |
| 75 | 10 min | wie 60, plus eine Volumenposition | 15–20 | Interpolation (**Heuristik**) |
| 90+ | 12–15 min | 1 Maximalübung (2–3) + 2 Volumenübungen (je 5) + 1–2 Zubringer (2–3); Prehab | 12–18 | PAR-B-67 |

Punktwerte innerhalb der Spannen: Aufwärmen 12 min bei 90+; Minutenangaben
der Vorlagen sind Startwerte für die Zeitschätzung (§5.8). Ist ein
Balance-Skill ein **Ziel**, gilt für den Balanceblock der abgestimmte Wert aus
`08` §4 (SESS-04) statt des «Skill»-Platzes der Vorlage; das Zeitbudget kürzt
ihn nach §5.8 bis auf 5 min.

**Reihenfolge (SESS-02).** Aufwärmen (mit Prehab-Aktivierung und Rampensätzen)
→ Balance-/Technikblock (ermüdungsarm) → Maximalblöcke der Straight-Arm-Skills
nach Priorität → Volumenblöcke → Kraftblock (Zubringer, Bent-Arm-Grundübungen,
Antagonisten-Paare) → belastende Prehab- und Konditionsübungen
(PAR-B-46, PAR-E-01, PAR-E-02, `08` §4 Zeile «Reihenfolge Balance-Block»). Bei
gleicher Priorität (z. B. zwei Zubringer eines Ziels) wechselt die Reihenfolge
von Einheit zu Einheit (PAR-E-03).

*Auflösung eines Widerspruchs:* `08` Rang 2 und PAR-E-01 stellen Prehab ans
Ende, Rang 9 und `05` §10 ins Aufwärmen (OSTRC, PAR-D-37). Der Planer trennt:
**Aktivierung** (leichte Übungen der Prehab-Liste, z. B. Aussenrotation mit
Band) kommt ins Aufwärmen wie im geprüften Programm [D-75]; **belastende**
Prehab (z. B. Unterarm-Exzentrik) kommt ans Ende, damit sie den Maximalblock
nicht vorermüdet (E-28).

**Aufwärmen (SESS-03).**

| Teil | Regel | Grundlage |
|---|---|---|
| Dauer | nach Vorlage; nie unter 5 min | PAR-B-69, PAR-B-68 |
| Allgemein | 5–10 min Anteil (bei 5 min Gesamtdauer 3 min, `PAR-S-37`), steigend | PAR-E-47 (**Heuristik**) |
| Prehab-Aktivierung | in 3 Einheiten je Woche (bei weniger Einheiten in jeder); Regionen in der Rangfolge Schulter > Handgelenk > Ellbogen = Rücken, Regionen mit Vorverletzung zuerst; Evidenzlabel je Region | PAR-D-37, PAR-D-01, PAR-D-38, `onboarding.md` §3.7 |
| Handgelenk-Vorbereitung | wenn eine Übung mit `r_eff(wrist)` ≥ 2 geplant ist | `02` §4.1, A-39 (C) |
| Mobilität | nach den Mobilitäts-Checks (§5.6, SEL-05) | `onboarding.md` §3.6, PAR-F-53 |
| Statisches Dehnen | < 60 s je Muskelgruppe | PAR-B-71, PAR-E-45 |
| Verboten | Maximalversuche, maximale Isometrie zur Potenzierung | PAR-D-05, PAR-E-46 |
| Rampensätze | 2 vor dem ersten Maximalblock (3 bei OG ≥ 10): zwei Sprossen unter der Arbeitssprosse, dann eine Sprosse darunter (oder mit Band), jeweils mit der halben Satzhaltezeit bzw. der Hälfte der Ziel-Wdh.; `kind = warmup`, Last-Gewicht 0.5 (`PAR-S-08`) | PAR-B-70, PAR-E-26 (Übertragung von %1RM auf Sprossen: **Heuristik**) |

**Balanceblock (SESS-04).** 11–15 min (Punktwert 12 min); in Einheiten unter
45 min 5–10 min (Punktwert 6 min) (`08` §4, PAR-E-35, PAR-B-35). Sätze à
21–40 s inklusive Versuche (Punktwert 30 s, PAR-E-36), Pause mindestens so
lang wie der Versuch (PAR-E-38), 30–90 s (PAR-B-74). Die Zahl der Sätze ergibt sich aus der Blockdauer. Balance zählt
voll in den Handgelenk-Deckel (`08` §4 Zeile «Greasing the Groove»).

**Maximal- und Volumenblöcke (SESS-05, SESS-06).** Je Straight-Arm-Leiter der
Einheit ein Maximalblock mit allen Sätzen hintereinander (PAR-E-32); nur wenn
das Zeitbudget sonst nicht reicht, werden Skills der Gegenrichtung als Paar
abgewechselt (PAR-B-81, §7.7); danach die Volumenblöcke in derselben
Reihenfolge. Dosierung §5.7.

**Kraftblock (SESS-07).** Zubringer- und Unterstützungsleitern, danach
Antagonisten-Paare als Supersätze mit 120 s zwischen den abwechselnden Sätzen
(PAR-B-45). Supersätze sind chronisch nicht schlechter (B-101).

**Ende (SESS-08).** Belastende Prehab, Konditionsstatik (z. B. Leans für
Anfänger, die noch nicht in der Leiter arbeiten), Rumpf.

**Leichte Einheit (SESS-09).** Balanceblock, Mobilität, Prehab-Aktivierung,
Technik an einer leichteren Sprosse (§5.7, `technique`). Keine harten oder
mittleren Reize (§7.5).

**Hypertrophie als Nebenziel (SESS-10).** Nur bei `secondary_focus =
hypertrophy` und freier Zeit: zusätzliche Sätze für Bent-Arm-Grundübungen bis
10 Wochensätze je Muskel (Start; Spanne 10–20, PAR-B-21), höchstens ≈ 11
fraktionale Sätze je Muskel und Einheit (PAR-B-75); Muskelrollen primär 1.0,
sekundär 0.5, Stabilisator 0.25 (PAR-C-46); assistierte Sätze nach PAR-B-79.
Die Deckel (§7) gelten unverändert.

**Greasing the Groove (SESS-11, optional, in v1 aus).** Nur Übungen mit
`gtg_allowed` (Bent-Arm-Grundübungen; Balance nur innerhalb des
Handgelenk-Deckels); je Satz ≤ 50 % der Maximal-Wdh. und RIR ≥ 2; nie
Straight-Arm-Statics; zählt in die Belastungseinheiten, nicht als
Hypertrophie-Satz (PAR-B-80, PAR-E-27, PAR-E-48, PAR-E-49, `08` Rang 20).

### 5.6 Übungsauswahl

**Kandidaten (SEL-01).** Sprossen der aktiven Leiter und ihre `alternative`-
Kanten, jeweils nur Übungen mit `training:`-Block und ohne `status: retired`.

**Filter, in dieser Reihenfolge:**

| Regel | Filter | Grundlage |
|---|---|---|
| SEL-02 | Equipment: `requires` ⊆ Equipment-Menge | `onboarding.md` §3.4 |
| SEL-03 | Regionen: SAFE-05/06; Matrix `X` für Regionen mit aktueller Beschwerde; `M` → Modifikation (SEL-10); `S` → erlaubt mit Schmerz-Monitoring (§8.6); Einschränkungen einer Fachperson über die `restriction_tags` der Übung | `05` §8, `onboarding.md` §3.7 |
| SEL-04 | Supinierte Straight-Arm-Varianten nur, wenn die Arbeitssprosse der Leiter mindestens OG 6 (Intermediate) erreicht; bei Ellenbeugen-Beschwerde ausgeschlossen, bis die Region `normal` ist; Einstieg als neue Belastungsart mit 50 % in Woche 1 | PAR-D-41, PAR-A-23, PAR-D-12, `08` §3 |
| SEL-05 | Mobilität: Handgelenk `no`/`partly` → Neutralgriff-Varianten (Parallettes, Fäuste) bevorzugen; Schulter → Brust-zur-Wand-Varianten im Handstand; Sprunggelenk → erhöhte Ferse bei Pistol-Regressionen; Kompression → Kompressionsleiter | `onboarding.md` §3.6, PAR-C-27, PAR-F-53 |
| SEL-06 | Minderjährige: Deckel nach §7.2; RF-12/RF-13 aktiv | PAR-D-23, `05` §9 |

Die `restriction_tags` sind eine eigene Angabe je Übung im `training:`-Block
(`support_straight_arm`, `hang_pull`, `overhead`, `wrist_extension_loaded`,
`supination_loaded`, `spine_extension`), damit die Zuordnung zu den Kategorien
aus dem Onboarding nicht aus Profilwerten erraten wird.

**Arbeitssprosse bei Halten (SEL-07).** Die höchste Sprosse mit Dosiswert
d ≥ 4 s, höchstens bis zur angebotenen Prüfsprosse (§6.3). Hat keine
unassistierte Sprosse d ≥ 4 s: die Band-Variante der niedrigsten Sprosse der
Ziel-Leiter, wenn `resistance_bands` vorhanden und die Übung `assistable` ist
(PAR-C-13, PAR-C-61), sonst die nächstniedrigere Sprosse bis zur Wurzel.
Grenze 4 s aus `08` §4 (Satzhaltezeit ≥ 2 s plus Reserve ≥ 2 s; PAR-B-05,
PAR-B-07).

**Einstieg nach dem Onboarding (SEL-08).** In den ersten zwei Einheiten eines
Musters ist die Arbeitssprosse das Minimum aus dem Ergebnis von SEL-07 und
der Sprosse unter der angegebenen Stufe (bzw. der angegebenen Stufe mit Band);
es wird also nie zweimal abgestuft. Der erste Arbeitssatz ist ein
Kalibrierungssatz. Ein Prüfversuch der angegebenen Stufe nur nach
`onboarding.md` §4.3.

**Arbeitssprosse bei Wiederholungen (SEL-09).** Die höchste Sprosse, an der
`floor(d) − RIR_Ziel` in den Wiederholungsbereich der Periodisierung fällt
(§5.7). Liegt es an der obersten Sprosse über 12 (untere Grenze von
PAR-B-23): Zusatzlast,
wenn `loadable` und Gewichte vorhanden, sonst bleibt die Sprosse mit höherem
Bereich. Ist an der Wurzel d < 1 (z. B. 0 Klimmzüge): exzentrische Variante
(PAR-B-16, PAR-B-77), Band-Variante falls vorhanden, dazu die Zubringer der
Leiter (Rudern, Hang; `02` §4.2, §4.7).

**Ersatz (SEL-10).** Fällt eine Übung weg oder ist sie `M`, in dieser
Reihenfolge:

1. gleiche Sprosse mit einem Modifikator, der die betroffene Struktur senkt
   (Parallettes, Fäuste, Neutralgriff; Equipment nötig) (PAR-C-27);
2. eine Sprosse tiefer in derselben Leiter;
3. eine `alternative`-Kante (`02` §8);
4. eine Übung desselben `pattern` mit niedrigerem `r_eff` auf der betroffenen
   Struktur;
5. streichen, mit Begründung.

`M` bedeutet ausserdem, dass die Familie unter den Rampenanteil der Region
fällt (§8.5); der Anteil wird je Konto genau einmal angewandt.

**Übungen ohne Matrix-Familie.** Hat eine Übung `complaint_family: none`, aber
`r_eff` ≥ 2 auf einer Struktur einer Region mit Beschwerde, gilt sie als `M`.

**Variation (SEL-11).** Gibt es gleichwertige Varianten derselben Sprosse,
wechseln sie zwischen den Einheiten; die Arbeitssprosse bleibt in jeder
Skill-Einheit (PAR-E-34, PAR-E-24).

### 5.7 Dosierung

Überall gilt: kein Satz bis zum Versagen bei Straight-Arm-Halten und
Skill-Versuchen (PAR-B-27, PAR-E-18); Tests sind ausgenommen. h = Satzhaltezeit,
d = Dosiswert (§4.8).

| Reiz | Sprosse | Sätze | Wdh. / Halt | Reserve | Pause | Grundlage |
|---|---|---|---|---|---|---|
| `skill_max`, Halt (DOSE-01) | Arbeitssprosse | `clamp(round(60 / h), 2, 5)` | h = min(0.70 · d, d − 2), ≥ 2 s; Anstieg ≤ +2 s je Woche | ≥ 2 s (folgt aus h) | 300 s (≥ 180); OG ≥ 14: 420 s | `08` §4; `PAR-S-01`, `PAR-S-02`, `PAR-S-05`; PAR-B-08, PAR-B-33, PAR-E-04, PAR-E-05, PAR-E-08 |
| `skill_max`, Wdh. (DOSE-02; Press, HSPU, Muscle-up, einarmiger Klimmzug) | Arbeitssprosse | 3 (2–5) | floor(d) − 1; bei d < 2 leichtere Sprosse, Band oder Exzentrik (SEL-09) | RIR ≥ 1 | 300 s | PAR-E-08, PAR-E-18, PAR-E-04 |
| `skill_volume` (DOSE-03) | Arbeitssprosse − 1; Band an der Arbeitssprosse nur, wenn es keine tiefere Sprosse gibt oder sie ausgeschlossen ist | `clamp(round(60 / h), 3, 5)` | h = min(0.70 · d', d' − 2), auf 5–20 s begrenzt; unter 5 s entfällt der Block | wie oben | 240 s (180–300) | PAR-B-10–12, PAR-E-06, `08` §4 |
| `conditioning` (DOSE-04; Leans, Stütz, Rumpfhalte, Sprossen mit d > 30 s) | Arbeitssprosse | 3, bei > 90 s Gesamtzeit 2 | h = min(0.70 · d, d − 2), auf 10–30 s begrenzt | wie oben | 120 s (120–180) | PAR-B-76, PAR-E-07, `08` §4 |
| `strength`, `novice` (DOSE-05) | nach SEL-09 | 3 | 5–8, lineare Doppelprogression (§6.4) | RIR ≥ 1, Ziel 2 | 120 s | PAR-A-01–05, PAR-B-01, PAR-B-24, PAR-B-42 (untere Grenze wegen Zeitbudget, `PAR-S-29`) |
| `strength`, trainiert (DOSE-06) | nach SEL-09 | 3 | wellenförmig je Exposition: schwer 3–6, mittel 6–10, leicht 10–15; Ziel = min(Obergrenze, floor(d) − 2) | RIR 2 | schwer 180 s, sonst 120 s | PAR-B-02, PAR-B-17–19, PAR-B-24, PAR-B-42, PAR-B-43, `PAR-S-13`, `PAR-S-29` |
| `eccentric` (DOSE-07) | exzentrische Variante | 2 (bis 3) | 3 Cluster-Wdh. à 3 s zu Beginn, steigend bis 7–10 s (ADAPT-10) | – | 180 s | PAR-B-16, PAR-B-77 |
| `balance` (DOSE-08) | Handstand-Leiter | nach Blockdauer | 21–40 s Satzdauer | – | ≥ Versuchsdauer, 30–90 s | PAR-E-35, 36, 38, PAR-B-74 |
| `technique` (DOSE-09) | Arbeitssprosse − 1 | 3 Versuche (bis 5) | h = min(0.5 · d, 10 s) | ≥ 50 % | ≥ Versuchsdauer, 30–90 s | `PAR-S-26`; PAR-E-14, PAR-E-38, PAR-B-74 |
| `prehab` (DOSE-10) | Prehab-Liste der Region | 2 | 15 Wdh. (12–20) | RIR 2 (1–3) | 90 s (60–120) | PAR-B-78, PAR-B-44, PAR-D-37 |
| `accessory` (DOSE-12; Zubringer, Unterstützung, Antagonisten) | Unterstützungsleiter | 3 | 12 Wdh. (12–20) | RIR 2 | 120 s; im Paar 120 s zwischen den abwechselnden Sätzen | PAR-B-78, `08` §4 (Zubringer 120 s), PAR-E-07, PAR-B-45 |
| `weighted` (DOSE-11) | oberste Sprosse + Zusatzlast | 3 | Bereich wie `strength` | RIR 2 | wie `strength` | PAR-B-23, PAR-B-32 |

Erläuterungen:

- **Satzzahl im Maximalblock.** `round(60 / h)` trifft die «Sweet Spots» der
  OG-Tabelle (Maximalhalt 10 s → 5 × 7 s; 20 s → 4 × 14 s; 30 s → 3 × 21 s
  statt 3 × 20 s; PAR-A-64). Ab d ≈ 10 s liegt die Gesamtzeit in 30–90 s
  (PAR-B-09); darunter ist sie kürzer (d = 6 s: 5 × 4 s = 20 s), weil
  höchstens 5 Sätze geplant werden (PAR-B-08) — bewusst vorsichtig. Die Zahl
  60 ist `PAR-S-02`.
- **Punktwerte.** Wo eine Quelle eine Spanne nennt, steht der Punktwert in der
  Tabelle; die Spanne in Klammern ist der Rahmen für die Autoregulation
  (`PAR-S-36`). Die Wahl folgt der Regel aus `08` §4: bei Sicherheitsgrössen
  die vorsichtigere Seite, sonst die Mitte.
- **Maximalhalt < 4 s** ergibt h < 2 s; dann gilt SEL-07 (leichtere Sprosse
  oder Band).
- **Wellenförmig (DOSE-06).** Bei 2 Expositionen je Woche schwer und mittel,
  bei 3 schwer, leicht, mittel in dieser Reihenfolge (`PAR-S-13`, Beispiel
  15/10/5 in B-115).
- **Zusatzlast (DOSE-11).** Einstieg, wenn alle Sätze an der obersten Sprosse
  12 saubere Wdh. erreichen (untere Grenze von PAR-B-23): Last = 2.5 % der
  bewegten Masse (KG + Zusatzlast), auf die kleinste verfügbare Scheibe
  **auf**gerundet, mindestens eine kleinste Scheibe (PAR-B-32), höchstens
  `max_added_load_kg` (`onboarding.md` §3.8). Ohne Angabe der Scheiben gilt
  1.25 kg (`PAR-S-37`).
- **Tempo.** Vorgegeben nur für Exzentrik (PAR-B-16). Für Halte im Maximalblock
  zeigt die App den Hinweis «Spannung schnell aufbauen, dann halten»
  (PAR-E-25; Hinweis, keine Zahl). Für dynamische Grundübungen gibt die
  Recherche nur den Praxiswert «10X0» [A-44] (Evidenz D); der Planer schreibt
  kein Tempo vor.
- **Intensität** wird über die Sprosse, das Band, die Zusatzlast und die
  Reserve gesteuert, nicht über % MVC: Die Modelle Haltezeit → Intensität
  widersprechen sich im Kurzzeitbereich und dienen nur Erklärtexten (`08` §4,
  PAR-A-63, PAR-B-14).
- **Kalibrierungssatz.** Erster Arbeitssatz einer Hauptübung bei niedriger
  Konfidenz, nach einem Widerspruch, nach Pausen (§6.11) und alle 4–6 Wochen
  (PAR-B-29): Ziel RIR/SIR 2–3; die App fragt die Reserve ausdrücklich ab
  (`onboarding.md` §4.1). Grenze: Ein Satz mit RIR 2–3 kalibriert die
  **Kapazität**, nicht die Genauigkeit der RIR-Schätzung, die PAR-B-29 mit
  einem Testsatz prüfen will. Ohne freiwilligen Test (§4.3 in
  `onboarding.md`) verlässt sich der Planer dafür auf den Fehler r = 2 Wdh.
  im Filter (§4.3).
- **Deload** überschreibt die Tabelle: Sätze × 0.6, Reserve +2 (WEEK-07).

### 5.8 Zeitbudget

Geschätzte Dauer = Aufwärmen + Σ Blöcke; ein Block = Σ Sätze × (Arbeitszeit +
Pause) − letzte Pause + 30 s Wechsel je Übung; Arbeitszeit = Haltezeit bzw.
3 s je Wiederholung (`PAR-S-11`). Liegt die Schätzung über `session_minutes`,
kürzt der Planer in dieser Reihenfolge (PAR-B-68):

1. Ergänzungen streichen;
2. Volumensätze auf 2–3;
3. Antagonisten-Paare zu Supersätzen, dann streichen;
4. Pausen der Nicht-Maximal-Reize auf die Untergrenze ihrer Spanne
   (Volumen 180 s; Kraft 120 s bei `novice`, sonst 180 s; Zubringer 120 s;
   PAR-E-06, PAR-B-42, `08` §4);
5. Balanceblock auf 5 min (`08` §4);
6. Volumenblock des Skills mit der niedrigsten Priorität streichen, dann
   dessen Maximalblock (Erhaltung, GOAL-03).

Nie gekürzt werden das Aufwärmen unter 5 min und der Primärblock des Ziels mit
Priorität 1 (PAR-B-68).

### 5.9 Stoppregeln in der Einheit

Der Plan liefert je Maximal- und Volumenblock Stoppregeln, die der Client
während der Einheit anzeigt; die Logs zeigen später, ob sie gegriffen haben.

| Regel | Auslöser | Folge | Grundlage |
|---|---|---|---|
| DOSE-20 | Form ≤ Form des ersten Arbeitssatzes − 1 oder < 3 | Block beenden; weiter mit Regression bzw. Kraftblock | PAR-E-15 |
| DOSE-21 | 2 Fehlversuche in Folge an derselben Sprosse | Block beenden; eine Sprosse leichter oder assistiert | PAR-E-16 |
| DOSE-22 | Leistung > 20 % unter dem besten Satz der Einheit | Block beenden | PAR-E-17 |
| DOSE-23 | Schmerz > 5/10 in einer beobachteten Region oder plötzlicher, stechender Schmerz in **irgendeiner** Region | Übung beenden; Schmerzbericht mit Red-Flag-Fragen (§8.2); bei einer N-Antwort Einheit beenden (auch offline, §10.5) | PAR-D-15, `05` §9 |

### 5.10 Ausgabe

Ein Plan ist eine Woche mit geordneten Einheiten; jede Einheit hat die Form des
Log-Modells (Blöcke → Satz-Einträge → Elemente), damit das Starten eine
strukturelle Kopie ist (wie Vorlagen, `00005_templates.sql`). Eine
Kombination bleibt ein Satz-Eintrag mit mehreren Elementen (CLAUDE.md).

```json
{
  "id": "0192…", "week_start": "2026-10-05", "ruleset_version": "1.0.0",
  "content_version_id": "0191…", "input_hash": "sha256:…",
  "sessions": [{
    "id": "0192…", "order_index": 0, "scheduled_date": "2026-10-05",
    "kind": "full", "est_minutes": 58,
    "blocks": [{
      "order_index": 2, "role": "skill_max", "kind": "straight",
      "entries": [{
        "id": "0192…", "order_index": 0, "kind": "working", "is_calibration": true,
        "rest_after_planned_s": 300, "target_rir": null, "target_sir_s": 3,
        "elements": [{ "exercise": "planche-tuck", "measure": "hold_seconds",
                       "target_hold_seconds": 7, "target_load_kg": 0, "assistance": null }],
        "stop_rules": ["DOSE-20", "DOSE-21", "DOSE-22", "DOSE-23"],
        "reasons": [{ "rule_id": "DOSE-01", "params": ["PAR-S-01", "PAR-S-02", "PAR-B-08"],
                      "sources": ["A-63", "B-09", "B-113"], "evidence": "heuristic",
                      "text_key": "reason.dose01", "args": { "hold_s": 7, "max_s": 10 } }]
      }],
      "reasons": [ … ]
    }],
    "reasons": [ … ]
  }],
  "exclusions": [{ "exercise": "planche-rings-tuck", "region": "elbow_inner",
                   "rule_id": "SEL-03", "matrix": "X", "sources": ["D-63"] }],
  "hints": [ … ], "realism": [ … ],
  "disclaimer": { "key": "planner.disclaimer",
                  "text": "Hefesto plant Training. Es ersetzt keine ärztliche oder physiotherapeutische Abklärung." }
}
```

## 6. Adaption aus Logs

### 6.1 Auslöser

| Auslöser | Wirkung | Idempotenz |
|---|---|---|
| Einheit abgeschlossen (`POST /v1/sessions/{id}/complete` oder Sync) | Kapazitäten, Leiterstand, Belastungshistorie, Regionen, Plateau- und Deload-Prüfung; `plan_changes` | einmal je Session (`plan_decisions` eindeutig über `trigger` + `source_id`) |
| Schmerzbericht | Schmerzregeln, Rampe, Verweise (§8.6) | einmal je Bericht |
| Check-in beim Start (optional) | Anpassung nur dieser Einheit (ADAPT-17) | je Start |
| Wochenwechsel | neuer Wochenplan; Pausen- und Deload-Prüfung | einmal je Woche |
| Profil-, Ziel-, Equipmentänderung | Neuerzeugung ab der nächsten nicht begonnenen Einheit (WEEK-08) | je Änderung |
| Freigabe einer Region oder des Screenings | Zustandswechsel (§8.3) | je Freigabe |
| neue Content- oder Regelversion | Neuerzeugung zu Wochenbeginn | je Version |

Die Adaption läuft nach dem Commit der auslösenden Transaktion. Scheitert sie,
bleibt der Abschluss der Einheit gültig; die Adaption wird beim nächsten
Planabruf nachgeholt (Idempotenz über `plan_decisions`).

### 6.2 Kapazität

ADAPT-01 bis ADAPT-03 wenden §4.3 auf jede qualifizierende Beobachtung an:
Update, erster Arbeitssatz als volle Beobachtung, Widerspruchsbehandlung. Die
Nachkalibrierung ändert nie den Unlock-Status (ADR 0008).

### 6.3 Progression bei Halten

| Regel | Inhalt | Grundlage |
|---|---|---|
| ADAPT-04 | Innerhalb der Sprosse folgt h dem aktualisierten Dosiswert, höchstens +2 s je Satz und Woche | PAR-B-33 |
| ADAPT-05 | **Prüfsprosse anbieten**, wenn der erste Arbeitssatz der Arbeitssprosse in 2 aufeinanderfolgenden Einheiten x = Halt + SIR ≥ 20 s mit Form ≥ 4 ergibt: Der Plan enthält dann ein **Angebot** für 2 kurze Versuche (≤ 5 s) an der nächsten Sprosse am Anfang des Maximalblocks. Die Versuche finden nur statt, wenn der User das Angebot aktiv annimmt (keine Voreinstellung) | PAR-B-05, PAR-B-30, PAR-A-78; kurze Probehalte an der nächsten Stufe [A-35]; `PAR-S-24`; `onboarding.md` §4.3 |
| ADAPT-06 | **Wechseln**, sobald die Prüfversuche für die nächste Sprosse einen Dosiswert ≥ 4 s ergeben (eine schwerere Sprosse hat vorher keinen Wert, §4.3); spätestens, wenn x an der Arbeitssprosse ≥ 30 s erreicht (dann mit Band, falls d < 4 s). Höchstens ein Sprossenwechsel je Leiter und Woche. Die alte Sprosse wandert in den Volumen- bzw. Konditionsblock | `08` §4 («Stufenfenster»), PAR-A-65, PAR-B-57 |
| ADAPT-06a | Prüfversuche und erste konzentrische Versuche (ADAPT-10) nur, wenn die Region `normal` ist, keine Rampe auf einem betroffenen Konto läuft, nicht in Woche 1 einer neuen Belastungsart, und SAFE-03/04 nicht greifen | `onboarding.md` §4.3, PAR-D-12, PAR-D-29 |

Die 2 × 5 s der Prüfversuche zählen voll ins Straight-Arm-Budget (§7.6).

### 6.4 Progression bei Wiederholungen

| Regel | Inhalt | Grundlage |
|---|---|---|
| ADAPT-07 | **`novice`, lineare Doppelprogression**: Start mit max(5, min(8, floor(d) − 2)) Wdh. × 3; hat eine Einheit alle Sätze mit RIR ≥ 1 geschafft, +1 Wdh. je Satz; bei 3 × 8 nächste Sprosse mit 3 × 5. Liegt der Dosiswert der nächsten Sprosse unter 6 (ohne Beobachtung: abgeleiteter Wert nach §4.3), bleibt die alte Sprosse (bis 12 Wdh., untere Grenze von PAR-B-23) und die neue erscheint einmal je Woche als Kalibrierungssatz am Blockanfang | PAR-A-01–05, PAR-B-01, PAR-B-23, `PAR-S-33` |
| ADAPT-08 | **Trainiert, wellenförmig**: Ziele je Expositionsklasse aus dem Dosiswert (DOSE-06); die Sprosse einer Klasse steigt erst, wenn deren letzte 2 Expositionen alle Sätze an der Obergrenze bei Ziel-RIR hatten | PAR-B-02, PAR-B-30 |
| ADAPT-09 | **Zusatzlast**: Einstieg nach DOSE-11; Steigerung um 2.5 % der bewegten Masse, wenn die Obergrenze in 2 Einheiten erreicht ist | PAR-B-23, PAR-B-30, PAR-B-32 |
| ADAPT-10 | **Exzentrik → erste Wiederholung**: +1 s je Wdh. und Einheit (`PAR-S-37`) bis 7–10 s; bei 3 × 3 Cluster-Wdh. à 7–10 s einen konzentrischen Versuch im nächsten Maximalblock **anbieten** (aktive Annahme, ADAPT-06a) | PAR-B-16, PAR-B-77, PAR-A-66 |

### 6.5 Balance-Skills

Handstand-Leitern steigen über die Haltezeit der Stufe (z. B. frei 10 s, dann
60 s vor dem einarmigen Handstand, PAR-A-55). Stagnation wird frühestens nach
16 Einheiten an einer Sprosse beurteilt (PAR-E-37). Balance-Tage folgen dem
Handgelenk-Deckel (§7.2).

### 6.6 Autoregulation (ADAPT-11)

In den ersten 4 Wochen gehen RIR/SIR nur in die Kapazität ein; danach steuern
sie die Ziele (PAR-B-29). Weicht die gemeldete Reserve in 2 aufeinanderfolgenden
Einheiten um ≥ 2 vom Ziel ab, ändert der Planer das Ziel um ±1 Wdh. bzw.
±1 s (Punktwert aus ±1–2); verlässt das neue Ziel den Bereich der Sprosse,
wechselt er die Sprosse nach SEL-07/SEL-09 (PAR-B-31). RIR wird im Mittel um ≈ 1 Wdh.
unterschätzt (PAR-B-28); der Planer korrigiert das nicht (PAR-F-25), sondern
setzt alle 4–6 Wochen einen Kalibrierungssatz (PAR-B-29).

### 6.7 Regression (ADAPT-12)

Eine Sprosse tiefer, sofort und ohne Wochenlimit (PAR-B-57 begrenzt nur
Steigerungen), wenn

- der Dosiswert der Arbeitssprosse unter 4 s fällt (Halte) bzw. der
  Wiederholungsbereich an der Sprosse nicht mehr erreichbar ist (SEL-07,
  SEL-09),
- DOSE-20 oder DOSE-21 in 2 aufeinanderfolgenden Einheiten gegriffen hat
  (PAR-E-15, PAR-E-16; `onboarding.md` §5.4),
- eine Schmerzregel verletzt ist (PAR-D-18, §8.6) oder
- eine Pausenrampe es verlangt (§6.11).

### 6.8 Plateau (ADAPT-13)

- **Erkennung** (`PAR-S-12`): Eine Kraft-Leiter hat ein Plateau, wenn sie seit
  mindestens 4 Wochen an derselben Sprosse arbeitet (`PAR-S-17`; neuronale
  Frühphase 3–5 Wochen, PAR-E-20) und der erste Arbeitssatz (x = Leistung +
  Reserve) in den letzten 2 Einheiten nicht über dem Wert der Einheit davor
  lag («stagniert oder fällt in ≥ 2 aufeinanderfolgenden Einheiten»,
  PAR-B-49 a). Balance-Leitern: erst nach 16 Einheiten (PAR-E-37).
- **Antwort, in dieser Reihenfolge:** (1) Deload der nächsten Woche, wenn das
  Plateau die Ziel-Leiter mit Priorität 1 betrifft («Hauptübung», PAR-B-49 a)
  und der letzte Deload ≥ 3 Wochen zurückliegt (`PAR-S-17`); (2) sonst und
  bei allen anderen Leitern Variation
  zwischen den Einheiten (PAR-E-34) und eine Unterstützungsübung aus einer
  `recommended`-Kante mit Gewicht ≥ 0.3 (z. B. gewichteter Klimmzug für den
  Front Lever, `02` §8); (3) nie mehr Sätze über die Deckel hinaus.
- Einzelne erste Sätze sind verrauscht (Test-SEM 2 Wdh., PAR-F-01). Ein
  fälschlich erkanntes Plateau kostet eine Deload-Woche; das ist in Kauf
  genommen, weil ein Deload keine Pause ist und im Streak zählt (`08` Rang 12,
  ADR 0003).
- Texte sagen, dass Fortschritt in Stufen verläuft, nie, dass jemand
  zurückliegt (ADR 0003).

### 6.9 Deload (ADAPT-14)

| Art | Auslöser | Umfang | Grundlage |
|---|---|---|---|
| geplant | Mesozyklus-Woche 6 (nach 5 Aufbauwochen) | ganze Woche nach WEEK-07 | PAR-B-03, PAR-B-50 |
| Stagnation | ADAPT-13 | nächste Woche nach WEEK-07 | PAR-B-49 a |
| Ermüdung | `perceived_fatigue` ≥ 8 in ≥ 3 der letzten 5 Einheiten | nächste Woche nach WEEK-07 | PAR-B-49 b |
| Höchstdauer | 8 Aufbauwochen ohne Deload | nächste Woche | PAR-B-49 d |
| Schmerz | Schmerzregel verletzt | ab der nächsten Einheit für 7 Tage: betroffene Strukturen 1 Sprosse leichter, Volumen −30 % | PAR-B-49 c, PAR-D-18 |

Geplante, Stagnations-, Ermüdungs- und Höchstdauer-Deloads setzen den
Mesozyklus zurück und fallen aus dem Referenzmittel der Deckel heraus (§4.5).
Der Schmerz-Deload wirkt nur auf die betroffenen Strukturen, setzt den
Mesozyklus nicht zurück und bleibt im Referenzmittel; danach gilt für diese
Konten bis zur ersten grünen Woche (§8.6) ein Deckel von 1.0 × dem
Referenzmittel vor der Verletzung der Schmerzregel, erst dann wieder LOAD-02
(`PAR-S-40`; «reduzieren und halten», PAR-D-18, `05` §5.4). Alle Deloads
werden beim Abschluss der Einheiten als Deload-Tage erfasst und zählen im
Streak (ADR 0003).

### 6.10 Verpasste Einheiten (ADAPT-15)

- Die Einheiten einer Woche sind eine **geordnete Warteschlange** mit
  Vorschlagsdatum, kein Kalenderzwang (`PAR-S-16`). Wer am Dienstag statt am
  Montag trainiert, bekommt die nächste Einheit der Schlange.
- Beim Start prüft der Planer die Abstände (§7.5) gegen die tatsächlich
  geloggten Einheiten. Verletzt die nächste Einheit einen Abstand, tauscht er
  sie gegen eine passende spätere Einheit der Woche; gibt es keine, stuft er
  die betroffenen harten Reize zu Technik (DOSE-09) herab. Beides mit
  Begründung.
- Verpasste Einheiten werden **nicht nachgeholt** oder verdoppelt; das würde
  den Einheitsdeckel verletzen (PAR-D-31). Am Wochenende verfallen offene
  Einheiten still (`status = expired`); es gibt keinen Hinweis auf Rückstand
  (ADR 0003).
- Längere Lücken behandelt §6.11.

### 6.11 Pausen und Wiedereinstieg (ADAPT-16)

Pausendauer je Lastkonto = Tage seit der letzten Einheit mit Last > 0 auf dem
Konto. Die Rampe bezieht sich auf das Referenzmittel vor der Pause.

| Pause | Bent-Arm, Beine, Rumpf | Straight-Arm- und Handgelenk-Konten | Grundlage |
|---|---|---|---|
| < 8 Tage | normal (LOAD-02) | normal | – |
| 8–14 Tage | Woche 1: höchstens 100 % von R, gleiche Sprosse | wie links | PAR-B-59 (bis 2 Wochen 90–100 %, obere Grenze) |
| 15–20 Tage | Woche 1: 90 %, gleiche Sprosse | wie links | PAR-B-59 (untere Grenze), B-84 |
| 21–48 Tage | Woche 1: 70 %, Sprosse nach Kalibrierungssatz; danach je Woche × 1.1 bis zur Referenz | ab 28 Tagen: Rampe (§8.6) ab Stufe 1 mit 25 % | PAR-B-60 (untere Grenzen), PAR-D-29, PAR-D-33 |
| 49–118 Tage | Woche 1: 60 %, eine Sprosse leichter oder nach Kalibrierung; je Woche × 1.1 | Rampe ab Stufe 1 mit 25 % | PAR-B-61, PAR-D-29, PAR-D-33 |
| ≥ 119 Tage | Woche 1: 50 %, 2 Sprossen leichter (vorsichtige Seite von «1–2»), danach Sprosse nach Kalibrierung und SEL-07; je Woche × 1.1; die höchsten Straight-Arm-Sprossen erst nach 4 Wochen Basis | Rampe ab Stufe 1 mit 25 % | PAR-B-62, PAR-D-29, PAR-D-33 |

- Wo Stream B eine Spanne nennt (70–80 %, +10–15 %), nimmt der Planer die
  untere Grenze (Sicherheitsgrösse, `08` §4). Mit × 1.1 je Woche ist die
  Referenz in Woche 5, 7 bzw. 9 erreicht (Woche 1 mitgezählt): in den
  Rückkehrfenstern von PAR-B-61 und PAR-B-62 (4–8, 6–12 Wochen), eine Woche
  länger als das Fenster von PAR-B-60 (2–4 Wochen); bewusst vorsichtig.
- Für Straight-Arm- und Handgelenk-Konten gewinnt die strengere D-Regel
  (`08` §4 Zeile «Wiedereinstieg»), weil die Sehnensteifigkeit schneller
  zurückgeht als die Kraft [D-18, D-19].
- «Retest» in PAR-B-60–62 heisst hier **Kalibrierungssatz** (RIR/SIR 2–3); ein
  Maximaltest nur nach `onboarding.md` §4.3 und nie für Straight-Arm- oder
  Handgelenk-Tests vor dem Ende der Rampe.
- Die Unsicherheit wächst mit der Pause (Vorhersageschritt, §4.3); ab 7 Wochen
  zusätzlich σ × 1.25 (`onboarding.md` §5.2).
- Eine Rampe nach einer Pause ohne Beschwerde (`entered_via = break`) wertet
  fehlende Schmerzberichte als schmerzfrei; bei einer Beschwerde braucht jede
  Stufe Berichte (§8.6). Die Pausen-Rampe gilt nur für die Straight-Arm-Konten
  und `wrist` der Region (`ramp_accounts`, §4.6); die Bent-Arm-Konten folgen
  der linken Spalte.
- Die Prozentwerte beziehen sich auf die geloggte Referenz vor der Pause,
  ohne geloggte Referenz auf das Zielvolumen. Für Straight-Arm- und
  Handgelenk-Konten ersetzt die Rampe den Wochendeckel in beiden Fällen, die
  Stufen wechseln zu Wochenbeginn (§15.2 U-13, Review); Bent-Arm-Konten folgen
  LOAD-02 bzw. LOAD-04 mit den Faktoren oben als Obergrenze.

**Ohne Logs: Pause aus dem Onboarding** (`PAR-S-41`; `onboarding.md` §3.5):

| `last_regular_training` | behandelt wie | Straight-Arm- und Handgelenk-Konten |
|---|---|---|
| `current_or_lt_3_weeks` | keine Pause | normal (LOAD-04) |
| `3_to_6_weeks` | 21–48 Tage | Rampe ab Stufe 1 mit 25 % (die Klasse beginnt bei 3 Wochen; die strengere Regel gilt, `onboarding.md` §3.5) |
| `7_to_16_weeks` | 49–118 Tage | Rampe ab Stufe 1 mit 25 % |
| `17_to_26_weeks`, `gt_26_weeks` | ≥ 119 Tage | Rampe ab Stufe 1 mit 25 % |
| `never` | neue Belastungsart (LOAD-04) | neue Belastungsart |

`pre_break_level` ist die **angegebene Stufe** für SEL-08 (Arbeitssprosse
darunter bzw. nach der Tabelle oben) und die **Obergrenze** der Sprosse
während der Rampe; der Startwert der Dosierung bleibt die Kapazität mit
σ × 1.25 (`onboarding.md` §3.5, §5.2).

### 6.12 Check-in (ADAPT-17, optional)

Beim Start kann der User Schlaf (Stunden) und Tagesform (1–10) angeben (ENT-7).
Schlaf ≤ 6 h (PAR-E-41) oder Tagesform-Ermüdung ≥ 8 (`PAR-S-28`) ersetzt die
Maximalblöcke dieser Einheit durch Technik (PAR-E-43, PAR-E-19); Kraftübungen
bleiben (Oberkörperkraft war unter Schlafmangel unbeeinflusst, PAR-E-42). Der
Check-in hat keine Folgen für Streak, XP oder Fortschritt und wird ohne
Einwilligung nicht gespeichert.

### 6.13 Unlocks und Profiländerungen (ADAPT-18)

Die Unlock-Engine läuft unverändert beim Abschluss (ADR 0008). Ein neuer Unlock
oder eine Kapazität, die eine Voraussetzung erfüllt, aktiviert Leitern nach
GOAL-02 ab der nächsten Einheit. Eine selbst bestätigte Stufe zählt für die
Planung als erfüllt; dosiert wird aber aus der Kapazität, die ohne Logs mit dem
«weiss nicht»-Prior startet (`onboarding.md` §5.2).

### 6.14 Ausgabe der Adaption

Jede Änderung wird ein Eintrag in `plan_decisions` und erscheint in der Antwort
des Abschlusses als `plan_changes[]`:

```json
{ "kind": "rung_up", "ladder": "planche", "from": "planche-tuck",
  "to": "planche-advanced-tuck", "effective_from": "next_session",
  "reasons": [{ "rule_id": "ADAPT-06", "params": ["PAR-A-65", "PAR-B-57"],
                "sources": ["A-63", "A-44", "B-12", "B-32"], "text_key": "reason.adapt06",
                "args": { "next_dose_s": 4.6 } }] }
```

Arten: `rung_up`, `rung_down`, `probe_offered`, `target_changed`,
`calibration_due`, `deload_scheduled`, `volume_capped`, `region_state`,
`referral_suggested`, `ramp_started`, `ramp_step`, `ladder_activated`,
`maintenance`.

## 7. Belastungssteuerung

### 7.1 Konten und Steigerungsraten (`PAR-S-15`)

| Konto | Wochensteigerung c | Grundlage |
|---|---|---|
| `wrist` | +10 % | PAR-D-11 |
| alle `…/SA`-Konten | +10 % | PAR-D-09; für Strukturen ausserhalb der in PAR-D-09 genannten (Bizepssehne, Ellbogen medial, vordere Schulter) gilt der strengere Wert nach `08` §4 («Sicherheitsgrösse → strengerer Wert») und S-2 |
| alle `…/BA`-Konten | +20 % | PAR-D-10 |

### 7.2 Wochendeckel (LOAD-02)

```
W_geplant(a) ≤ R(a) · (1 + c(a) · f(a))
```

f(a) ist das Minimum der zutreffenden Faktoren (sie multiplizieren sich
nicht): 0.5 bei Vorverletzung der Region in den letzten 12 Monaten
(PAR-D-02), 0.5 für Handgelenk- und Straight-Arm-Konten bei Minderjährigen
(PAR-D-23), 0.5 ohne Einwilligung (SAFE-04), 0.75 im Risikofenster
6–48 Monate Trainingsalter (PAR-D-04, Betrag `PAR-S-25`, vorläufig bis zur
Entscheidung ENT-S-5), sonst 1. Nach einem Schmerz-Deload gilt für die
betroffenen Konten bis zur ersten grünen Woche W ≤ 1.0 × R vor der Verletzung
der Schmerzregel (§6.9, `PAR-S-40`).

**Rampen.** Hat ein Rampenkonto eine **geloggte** Referenz vor der Beschwerde
bzw. Pause, ersetzen die Rampenanteile den Wochendeckel, weil nur auf früher
toleriertes Niveau zurückgekehrt wird (`05` §6.2). Ohne geloggte Referenz
(z. B. eine Beschwerde schon im Onboarding) gibt es kein toleriertes
Niveau: Dann gilt LOAD-02 bzw. LOAD-04 unverändert, und der Rampenanteil ×
Zielvolumen wirkt **zusätzlich** als Obergrenze. Eine Region mit Beschwerde
wächst so nie schneller als ein beschwerdefreies neues Konto. Die
Pausenrampe ohne Beschwerde ersetzt den Deckel dagegen auch ohne geloggte
Referenz (§6.11, §15.2 U-13).

**Ganze Sätze** (`PAR-S-35`). Der Deckel ist eine Zahl in Belastungseinheiten,
geplant werden ganze Sätze. Der Anteil eines Satzes, der beim Abrunden
übrig bleibt, wird je Konto als Spielraum in die nächste Woche übertragen;
erreicht der Spielraum einen ganzen Satz, wird dieser Satz geplant (höchstens
+1 Satz je Übung und Woche, PAR-B-55). So wächst auch ein kleines Volumen im
Mittel mit der Rate des Deckels, statt beim Abrunden stehen zu bleiben; eine
einzelne Woche kann dabei bis zu einem Satz über dem rechnerischen Deckel
liegen.

### 7.3 Einheitsdeckel (LOAD-03)

```
S_geplant(s, Einheit) ≤ M(s) · 1.10
```

je Struktur (beide Konten zusammen) (PAR-D-31, `08` §4). In der Rampe bezieht
sich M auf das geloggte beschwerdefreie Maximum vor der Beschwerde (`05`
§6.2); gibt es keins, gilt LOAD-04.

**Ganze Sätze** (`PAR-S-48`, Review). Geplant werden ganze Sätze. Der Deckel
lässt deshalb je Einheit und Struktur mindestens M plus den kleinsten
geplanten Satz dieser Struktur in der Einheit zu (bei M = 0: mindestens diesen
einen Satz). Ohne diese Regel wächst eine Einheit mit einem oder zwei Sätzen
nie, weil schon ein zusätzlicher Satz mehr als 10 % ist. Das Wochenwachstum
begrenzt weiter LOAD-02.

### 7.4 Neue Belastungsart (LOAD-04)

Ist R(a) = 0 (keine Last in den letzten 6 Wochen), gilt für die erste Woche
W ≤ 0.5 × Zielvolumen; ist M(s) = 0, gilt je Einheit ≤ 0.5 × Ziel-Einheitslast
(PAR-D-12). Ab der zweiten Woche gilt LOAD-02 mit R über die vorhandenen
Wochen (`PAR-S-14`). Damit startet jeder neue User mit halbem Volumen
(`onboarding.md` §3.5).

**Folge, die der Checkpoint entscheiden soll (ENT-S-1).** Weil R ein
nachlaufendes Mittel ist, wächst ein Straight-Arm-Konto von 50 % des
Zielvolumens effektiv nur um ≈ 4–6 % je Woche (Woche 2: 55 %, Woche 3: 58 %,
Woche 4: 60 %, Woche 5: 63 %) und erreicht 100 % nach ≈ 15 Wochen, im
Risikofenster (f = 0.75) nach ≈ 20 Wochen; ein Bent-Arm-Konto nach ≈ 8 bzw.
10 Wochen (eigene Rechnung ohne Deload-Wochen, die den Weg um je eine Woche
verlängern).
Die Intensität (Sprosse) ist davon nicht betroffen, nur die Satzzahl; Kraft
sättigt mit dem Volumen früh (`08` §3 H-6), deshalb ist der Verlust klein.
Für Trainierte, die gerade regelmässig trainieren, ist das trotzdem sehr
vorsichtig; dafür gilt LOAD-04b (ENT-S-1). Die Rechnung gilt für ganze
Sätze nur dank des Spielraum-Übertrags (`PAR-S-35`); ohne ihn blieben kleine
Straight-Arm-Volumen beim Abrunden dauerhaft stehen.

**Einstieg für aktuell Trainierende (LOAD-04b, ENT-S-1).** Ein Konto bekommt
statt LOAD-04 die Stufen 50 % → 75 % → 100 % des Zielvolumens in
wöchentlichen Schritten, wenn alle Bedingungen gelten:
`last_regular_training = current_or_lt_3_weeks`; der User hat für eine Übung,
die das Konto belastet, einen Leistungsstand > 0 angegeben (nicht «weiss
nicht»); die Regionen des Kontos haben weder eine aktuelle Beschwerde noch
eine Verletzung in den letzten 12 Monaten; der User ist nicht minderjährig.
Jeder Schritt setzt voraus, dass in der Vorwoche keine Schmerzregel verletzt
wurde; sonst bleibt der Schritt stehen. Nach 100 % gilt LOAD-02 gegen die dann
geloggten Wochen. Die Schritte folgen der Reihe PAR-D-25 (ab 0.5, wie
PAR-D-12), mit derselben Begründung wie die Rampe in `05` §6.2: Rückkehr auf
ein Niveau, das der User aktuell trägt (`PAR-S-43`). Der Einheitsdeckel
LOAD-03 gilt in diesen drei Wochen gegen das Maximum der Vorwoche × 1.5, weil
die Schritte selbst +50 % bzw. +33 % betragen.

### 7.5 Abstände (LOAD-05)

| Klasse | Reize | Mindestabstand zur nächsten harten oder mittleren Belastung derselben Struktur | Grundlage |
|---|---|---|---|
| hart | `skill_max`, `skill_volume` an Straight-Arm-Sprossen, `test`, Prüfversuche, `eccentric`, jeder Satz mit Ziel-RIR ≤ 1 | 48 h; in der Rampe 72 h | PAR-D-08, PAR-D-34, PAR-B-38, PAR-E-13; `skill_volume` als hart: `PAR-S-30` |
| mittel | `strength`, `weighted`, Hypertrophie (RIR ≥ 2) | 24 h | PAR-B-38, PAR-E-14 |
| mittel (Technik) | `technique` an einer Straight-Arm-Sprosse mit `r_eff` ≥ 2; in einer Region mit Rampe hart | 24 h (Rampe 72 h) | `08` §4 («24 h nach submaximaler Technik»), PAR-E-14, PAR-D-34 |
| leicht | `balance`, übrige `technique`, `conditioning` ohne Straight-Arm, `prehab`, `mobility`, Aufwärmen | kein Mindestabstand ausser einer Einheit je Tag und Struktur | PAR-D-03 |

Geprüft werden nur Strukturen mit `r_eff` ≥ 2 in beiden Einheiten
(`PAR-S-07`): Ein Profilwert 1 bedeutet geringe Last (PAR-C-45). Massgeblich
ist die höchste Klasse der Einheit auf der Struktur.

### 7.6 Straight-Arm-Budget je Einheit (LOAD-06)

Satzzahl aller Übungen mit `straight_arm = straight_arm` (ohne `axial`; ohne
Aufwärmsätze; Band-Sätze zählen voll, PAR-B-79):

| Höchste Arbeitssprosse der Einheit | Budget | Grundlage |
|---|---|---|
| OG ≤ 5 (Beginner) | 8 | PAR-B-47 (untere Grenze der Spanne 8–12), PAR-A-23 |
| OG 6–9 (Intermediate) | 12 | PAR-B-47 |
| OG ≥ 10 (Advanced, Elite) | 18 | PAR-B-47, PAR-E-09 |

Die Zuordnung der Spanne von PAR-B-47 zu den OG-Bändern ist `PAR-S-23`. Das
Budget wird nach Priorität im Verhältnis 3 : 2 : 1 verteilt, mit mindestens 2
Sätzen je Maximalblock (`PAR-S-09`; PAR-B-08 Untergrenze 2); Reste beim
Runden gehen nach dem Verfahren der grössten Reste, bei Gleichstand an die
höhere Priorität. Reicht das Budget
nicht für alle Maximalblöcke, fällt der Block mit der niedrigsten Priorität in
die Erhaltung (GOAL-03).

### 7.7 Konflikte zwischen Skills (LOAD-07, LOAD-08)

| Konflikt | Regel | Grundlage |
|---|---|---|
| Zwei Straight-Arm-Skills gleicher Richtung (z. B. Planche + Maltese) | höchstens 2 je Einheit, gemeinsames Budget, Priorität zuerst | PAR-B-48 |
| Gegenrichtung (Planche + Front Lever) | Standard: zwei Maximalblöcke nacheinander (geblockt, PAR-E-32). Nur wenn das Zeitbudget sonst nicht reicht: als Paar abwechselnd; zwischen A- und B-Satz max(120 s, Pause des Skills / 2), sodass zwischen zwei Sätzen desselben Skills seine volle Pause liegt (300 s) | PAR-E-32, PAR-B-81, PAR-B-45, PAR-E-04, `PAR-S-10` |
| Gemeinsame Strukturen (Planche und Front Lever belasten beide `elbow_medial`, `biceps_distal`; `04` Tabelle 2) | Sie landen auf **denselben** Tagen, nicht auf aufeinanderfolgenden: Die Abstandsregel (§7.5) verbietet Planche am Montag und Front Lever am Dienstag | §7.5; «keine Sperrfrist zwischen zwei verschiedenen Skills» (PAR-B-81) gilt nur, solange keine Struktur verletzt wird |
| Handstand + Planche/HSPU | beide belasten das Handgelenk mit 3; Balance ist `leicht` (kein Abstand), zählt aber voll in den Handgelenk-Deckel | `04` Tabelle 2, `08` §4 (GtG), PAR-D-11 |
| Front Lever + Back Lever | beide `shoulder_extension` 3: gleiche Tage; Budget gemeinsam | `04` Tabelle 2 |
| Muscle-up + Klimmzug/Dip | Muscle-up-Versuche im Maximalblock, Klimmzug/Dip im Kraftblock derselben Einheit | PAR-B-46 |

*Auflösung eines Widerspruchs:* PAR-E-32 verlangt, alle Maximalversuche eines
Skills hintereinander zu legen; PAR-B-81 paart Skills der Gegenrichtung
abwechselnd mit der Pause PAR-B-45 (120 s). Beide sind Heuristik (`08` Rang 21
bzw. `03` §7). Der Planer blockt standardmässig und paart nur zur Zeitersparnis.
Beim Paaren bleibt die Pause je Skill nach PAR-E-04 erhalten (`PAR-S-10`: 150 s
zwischen den abwechselnden Sätzen ergeben 300 s je Skill), damit die
Leistungsgrösse dem besser belegten Wert folgt (`08` §4).

**Beispiel Persona 2** (Planche und Front Lever, 4 Einheiten, keine bevorzugten
Tage → Mo, Di, Do, Sa): Beide Skills kommen auf Mo, Do, Sa (Abstände 72 h,
48 h, 48 h), geblockt nacheinander, gepaart nur bei Zeitmangel; Di trägt
Beine, Rumpf, Balance und leichte Technik ohne Straight-Arm-Sprossen. So
bekommt jeder Skill 3 Einheiten (PAR-B-34 für Trainierte), ohne dass eine
Struktur innerhalb von 48 h zweimal hart belastet wird.

### 7.8 Schutz neuer Sprossen (LOAD-09)

Nach einem Wechsel auf eine neue Straight-Arm-Sprosse gilt diese ≈ 12 Wochen
nicht als voll belastbar (PAR-D-06): In dieser Zeit wird sie nur im
Maximalblock trainiert, das Volumen bleibt an der Sprosse darunter, und es gibt
keinen Prüfversuch der übernächsten Sprosse (`PAR-S-27`). Die höhere
Belastungseinheit der neuen Sprosse (§4.5) läuft ohnehin durch die Deckel.

### 7.9 Kürzen bei Deckelüberschreitung (LOAD-10)

Verletzt ein Plan einen Deckel oder das Budget, kürzt der Planer
deterministisch, jeweils zuerst bei der niedrigsten Priorität und neu prüfend
nach jedem Schritt:

1. Unterstützungs- und Ergänzungssätze, die das Konto belasten;
2. Balance-, Technik-, Konditions- und belastende Prehab-Sätze;
3. Volumensätze (bis der Block entfällt);
4. Kraftsätze auf 1 (§15.2 U-15);
5. Maximalsätze auf 1, danach Angebote (ADAPT-05, ADAPT-10);
6. Arbeitssprosse eine tiefer (senkt k);
7. Exposition der Einheit streichen;
8. Erhaltung (GOAL-03).

Jeder Schritt senkt die Last mindestens eines Kontos; die Schleife endet, weil
Satzzahl und Sprossen nach unten begrenzt sind (§11). Jede Kürzung erzeugt
einen Reason mit dem Deckel, seinem Wert und der Ursache.

### 7.10 Keine ACWR-Sperre (LOAD-11)

Das Verhältnis akuter zu chronischer Last wird nicht als Kriterium verwendet
(PAR-B-58, PAR-D-30). Die Session-RPE-Last (Ermüdung × Minuten, PAR-B-72) und
Monotonie/Strain im 6-Wochen-Fenster (PAR-B-73) werden nur für spätere
Auswertungen protokolliert, nicht angezeigt und nicht als Sperre benutzt.

## 8. Verletzungslogik

### 8.1 Grundsatz

Der Planer steuert **Trainingslast**, nicht Heilung. Er fragt Symptome und Orte
ab, nennt keine Verdachtsdiagnose und verweist bei Warnzeichen an
Fachpersonen. Die Rampe (§8.6) ist Laststeuerung nach Beschwerden oder Pausen,
keine Rehabilitation und kein Rückkehr-Rat im medizinischen Sinn; diese
Unterscheidung steht jetzt auch in `CONTENT_AUTHORING.md` (ENT-2, ADR 0012).
Jede Antwort mit Beschwerde-, Rampen- oder Red-Flag-Inhalt trägt den
Disclaimer im API-Payload (CLAUDE.md).

### 8.2 Red Flags (INJ-01, INJ-02)

**Wann gefragt:** im Onboarding für jede Region mit aktueller Beschwerde; bei
**jedem** Schmerzbericht über 5/10 während/nach, über 2/10 im Alltag
(PAR-D-15, PAR-D-14), mit `sudden_sharp` oder für eine neue Region
(«bei jeder Beschwerdemeldung», S-1, `05` §9); RF-11 automatisch über die
Fristen (§8.7). Belastungssymptome (RF-10) kann der User jederzeit über
«Symptome beim Training melden» angeben, unabhängig von einer Region. Die
Fragen, Dringlichkeiten und N-Aktionen liegen im Plan-Payload, damit der
Client sie offline auswerten und eine Einheit stoppen kann (§10.5). **Welche:** ortsabhängig nach `onboarding.md` §3.7
(RF-01 bis RF-07 und RF-10 überall; RF-08, RF-09 nur Rücken; RF-12 nur
Handgelenk und minderjährig; RF-13 nur Rücken und minderjährig).

| Dringlichkeit | Flags | Aktion |
|---|---|---|
| N (sofort) | RF-05 (Gelenk sichtbar verschoben), RF-07, RF-08, RF-10 | **Training insgesamt stoppen** (SAFE-02) und Region `locked`; sofortige ärztliche Abklärung empfehlen; Plan erst nach bestätigter Freigabe |
| D (in den nächsten Tagen) | RF-01, RF-02, RF-03, RF-05 (Schwellung, Bluterguss), RF-06, RF-09 | Region `locked`, zeitnahe Abklärung empfehlen, Freigabe nötig (PAR-D-21) |
| A (Abklärung empfehlen) | RF-04, RF-11, RF-12, RF-13 | Region `rtt_0`; Hinweis auf Fachperson; RF-12/RF-13: Stütz- bzw. Extensionselemente pausieren |

Eigene Einschätzung «ernsthafte Verletzung» (`suspected_serious = yes`) oder
Befund «Riss oder Verdacht» wirken wie eine D-Flag; `unsure` bzw. eine
laufende Abklärung setzen `rtt_0` mit Hinweis (`onboarding.md` §3.7).

**Region `other`.** Sie hat keine Strukturen, auf die ein Ausschluss wirken
könnte (z. B. Brust bei RF-01). Eine N- oder D-Flag bzw. «ernst» in `other`
pausiert deshalb die Planung (Auflage `plan_stopped`), bis der User die
betroffenen Übungen ausgeschlossen oder die Freigabe bestätigt hat (INJ-02).
Eine eigene Region «Brust» ist ENT-S-9.

### 8.3 Zustandsautomat je Region (INJ-03)

| Von | Ereignis | Nach | Grundlage |
|---|---|---|---|
| `normal` | Beschwerde ohne Red Flag, Alltagsschmerz ≤ 2 | `rtt_1` mit Start 0.5 (0.25 nach Sehnenbefund/Verweis) | PAR-D-14, PAR-D-24, PAR-D-33 |
| `normal` | Beschwerde ohne Red Flag, Alltagsschmerz > 2 | `rtt_0` | PAR-D-14 |
| `normal` | Pause ≥ 28 Tage auf Straight-Arm-/Handgelenk-Konten der Region | `rtt_1` mit 0.25 (`entered_via = break`) | PAR-D-29, PAR-D-33 |
| beliebig | Red Flag N | `locked` und SAFE-02 (Training gestoppt) | PAR-D-21, `05` §9 (RF-05 «Region sperren»), `onboarding.md` §3.7 |
| beliebig | Red Flag D oder Selbsteinschätzung «ernst» | `locked` | PAR-D-21 |
| beliebig | Red Flag A oder Beschwerde > 28 Tage ohne Besserung (RF-11) | `rtt_0` | `05` §9, PAR-D-19 |
| `locked` | User bestätigt Freigabe durch eine Fachperson (`POST /v1/me/regions/{region}/clearance`; nach einem N-Stopp hebt `POST /v1/me/screening/clearance` den Stopp auf und setzt gesperrte Regionen auf `rtt_1`) | `rtt_1` mit 0.25 | `05` §6.2 (Stufe 0 nach Verweis), PAR-D-33 |
| `rtt_0` | Alltagsschmerz ≤ 2 und Red-Flag-Fragen negativ | `rtt_1` | `05` §6.2 |
| `rtt_n` (1–4) | PAR-D-26 erfüllt (≥ 2 Einheiten der Stufe ohne Beschwerden während, nach und am Folgetag; Schmerzregeln eingehalten; ≥ 7 Tage seit dem letzten Wechsel) | nächster Volumenschritt; ist der Schritt 0.75 der Stufe 2 erreicht und erfüllt, `rtt_{n+1}` | PAR-D-25, PAR-D-26 |
| `rtt_n` (1–5) | Soreness am Folgetag (`PAR-S-47`) oder Schmerz > 1 h danach (`lasted_over_1h`) | Stufe bzw. Schritt wiederholen (Zähler und 7-Tage-Frist beginnen neu), 1 Tag Pause der Region | PAR-D-28, `PAR-S-47` |
| `rtt_n` (1–5) | Schmerz im Aufwärmen, der > 15 min anhält (`persisted_over_15min`) | einen Volumenschritt bzw. eine Stufe zurück (in `rtt_1` auf den nächstkleineren Anteil, mindestens 0.25), 2 Tage Pause der Region | PAR-D-28 |
| `rtt_n` (1–5) | Schmerzregel PAR-D-15, PAR-D-16 oder PAR-D-17 verletzt | Schmerz-Deload (§6.9); Zähler und 7-Tage-Frist des Schritts beginnen neu; bei Soreness zusätzlich die Zeilen darüber | PAR-D-18, PAR-D-26 |
| `rtt_5` | 2 Wochen ohne Regelverletzung | `normal` (normale Deckel gegen die dann gültige Referenz) | **Heuristik** (`PAR-S-32`) |

### 8.4 Matrix anwenden (INJ-04)

Für jede Region mit aktueller Beschwerde (`rtt_1`–`rtt_4`) bestimmt die Zelle
Region × `complaint_family` der Übung die Aktion (`05` §8): **X** nicht planen;
**M** modifizieren (§8.5); **S** erlaubt, solange die Schmerzregeln halten
(§8.6); **–** nicht betroffen. In `rtt_5` gelten M-Zellen als S; X-Zellen
bleiben X, bis die Region `normal` ist, und kommen dann als neue
Belastungsart (LOAD-04, 50 %) zurück (**Heuristik**, `PAR-S-32`). Die Matrix-Aktionen sind **Heuristik**; die Quelle einer Zelle belegt
den Mechanismus, nicht die Aktion (`05` §8). `knee` wirkt nur auf die
Bein-Familie (Tiefe begrenzen, Pistol-Stufen M), `other` hat keine
automatische Regel (`onboarding.md` §3.7).

### 8.5 Modifikation und Ersatz (INJ-05)

`M` heisst, in dieser Reihenfolge (SEL-10): Modifikator, der die betroffene
Struktur senkt (Parallettes, Fäuste, Neutralgriff; PAR-C-27); eine Sprosse
tiefer; kürzere Halte (Technik-Dosierung DOSE-09); Volumen der Familie ×
Rampenanteil der Region. Supinierte Straight-Arm-Varianten sind bei
Ellenbeugen-Beschwerde immer X (PAR-D-41).

### 8.6 Rampe und Schmerz-Monitoring (INJ-06, INJ-07)

| Stufe | Training der Region | Volumen (Anteil der Referenz) | Grundlage |
|---|---|---|---|
| 0 | nicht geplant (SAFE-06) | 0 | `05` §6.2 |
| 1 | Regression; bei Handgelenk und Ellbogen zuerst Hang/Zug | Startanteil 0.5 bzw. 0.25 | PAR-D-24, PAR-D-27, PAR-D-33 |
| 2 | Regression; Stütz ohne Impact | die auf den Startanteil folgenden Schritte der Reihe 0.25 → 0.5 → 0.75 (`ramp_step`), je Schritt ≥ 7 Tage und PAR-D-26 | PAR-D-25, PAR-D-26 |
| 3 | Regression | 1.0 | `05` §6.2 |
| 4 | Arbeitssprosse submaximal (Technik-Dosierung, Band erlaubt) | 1.0 | `05` §6.2 |
| 5 | Maximalblöcke und Progression; Prüfversuche erst ab `normal` (ADAPT-06a) | normale Deckel | `05` §6.2, `onboarding.md` §4.3 |

Der Rampenanteil gilt für die `ramp_accounts` der Region (§4.6) und wird je
Konto genau einmal angewandt, auch wenn eine Übung zusätzlich eine M-Zelle
hat.

Harte Reize der Region liegen in der Rampe ≥ 72 h auseinander (PAR-D-34).

**Schmerz-Monitoring.** Beobachtet werden Regionen in `rtt_1`–`rtt_5` und
Regionen, deren S-Zellen in der Einheit vorkommen. Vor der Einheit fragt die
App kurz nach dem aktuellen Wert (`before_session`), im Aufwärmen nur, wenn
Schmerz auftritt (`warmup`, mit «hält länger als 15 min an»), nach der Einheit
«während» und «danach» (mit «hielt länger als 1 h an»), am nächsten Morgen
«heute früh»; täglicher Alltagsschmerz optional (PAR-D-13). Die Fragen sind
neutral, freiwillig und ohne Streak-Folgen.

- **Neue Beschwerde** (Übergang aus `normal`): ein Bericht über 2/10 oder
  Berichte über 0 an 2 verschiedenen Tagen innerhalb von 7 Tagen (PAR-D-14;
  Zählweise `PAR-S-42`).
- **Morgenregel:** «heute früh» ist nicht höher als `before_session`; fehlt
  dieser Wert, gilt der letzte Morgen- bzw. Alltagswert vor der Einheit
  (PAR-D-16).
- **Wochentrend steigend:** Mittel der Werte «danach» der laufenden ISO-Woche
  ≥ 1 Punkt über dem Mittel der Vorwoche (PAR-D-17; Schwelle `PAR-S-42`).

| Regel | Bedingung | Folge | Grundlage |
|---|---|---|---|
| grün | alle Werte ≤ 2 und keine Soreness (`PAR-S-47`) | Progression erlaubt; zählt für PAR-D-26 | PAR-D-14, `PAR-S-47` |
| akzeptabel | während/nach ≤ 5, am Morgen nicht höher als vor der Einheit, Wochentrend nicht steigend | weiter ohne Progression | PAR-D-15, PAR-D-16, PAR-D-17 |
| verletzt | eine der Bedingungen verfehlt | ab der nächsten Einheit: betroffene Strukturen 1 Sprosse leichter, −30 % Volumen, 7 Tage (als Deload erfasst); in der Rampe zusätzlich §8.3 | PAR-D-18, PAR-D-28 |
| kein Bericht | Region mit Beschwerde | keine Progression der Rampe; Plan läuft weiter | `PAR-S-34` (konservativ, ohne Strafe) |

Ohne Red Flags wird nie komplett pausiert; die Last wird reduziert
(PAR-D-40). Die App verspricht keine Linderung durch bestimmte Übungen
(PAR-D-39).

### 8.7 Verweise (INJ-08)

| Bedingung | Hinweis | Dringlichkeit | Grundlage |
|---|---|---|---|
| Beschwerde ≥ 14 Tage ohne Besserung trotz Lastanpassung | freundlicher Hinweis auf eine Fachperson | A | PAR-D-32 |
| ≥ 3 Schmerzregel-Verletzungen in 14 Tagen | Fachperson empfehlen | A | PAR-D-20 |
| Beschwerde ≥ 28 Tage ohne Besserung oder schlechter (RF-11) | Fachperson empfehlen; Region `rtt_0` | A | PAR-D-19, `05` §9 |
| Dauer > 4 Wochen schon im Onboarding | Hinweis im Onboarding | A | `onboarding.md` §3.7 |

«Ohne Besserung» heisst: Alltagsschmerz > 2 (PAR-D-14) oder erneute
Regelverletzung in der Frist.

### 8.8 Minderjährige (INJ-09)

Ob Minderjährige zugelassen werden, ist rechtlich offen (ENT-3, OE-1). Falls
ja: Deckelfaktor 0.5 für Handgelenk und Straight-Arm (PAR-D-23, PAR-D-02),
RF-12 und RF-13 aktiv, Verweisfristen 7 Tage für Handgelenkschmerz bei Stütz
und Rückenschmerz bei Extension (PAR-D-22, PAR-D-42).

### 8.9 Vorgaben von Fachpersonen (INJ-10)

Einschränkungen aus `complaints[].restrictions` sind harte Ausschlüsse über die
`restriction_tags` der Übungen, bis der User sie aufhebt; sie gehen jeder
App-Regel vor (`onboarding.md` §3.7).

## 9. Erklärbarkeit

### 9.1 Das Reason-Objekt (EXPL-01, EXPL-02)

Jede Entscheidung, die der User sieht, trägt mindestens einen Reason:

| Feld | Inhalt |
|---|---|
| `rule_id` | Regel aus Anhang A |
| `params` | benutzte Parameter-IDs (`PAR-…`) |
| `sources` | Vereinigung der Quellen der Parameter und der Regel |
| `evidence` | Evidenz der Regel: `A`, `B`, `C`, `D` oder `heuristic` |
| `text_key`, `args` | lokalisierter Text mit Platzhaltern |
| `text` | fertiger Text in der Sprache der Anfrage (`Accept-Language`) |

Reasons hängen an: Plan (Wochenstruktur, Deload, Ziele), Einheit (Art, Tag,
Tausch), Block (Reihenfolge, Paarung), Satz-Eintrag (Sprosse, Dosis, Pause,
Kalibrierung), Ausschluss (`exclusions[]`), Hinweis (`hints[]`),
Realismus-Check (`realism[]`) und Änderung (`plan_changes[]`). Jede ID in einem
Reason existiert in der Wissensbasis der Planversion (Validierung beim Erzeugen;
Eigenschaftstest §12.1).

### 9.2 Anzeige

- **«Warum?»** an jedem Element: Regeltext mit den eingesetzten Werten, z. B.
  «7 s je Satz: etwa 70 % deiner geschätzten Maximalhaltezeit von 10 s, damit
  2–3 s Reserve bleiben. Lange Pausen, weil die Qualität der Halte zählt.»
- **«Belege»**: Liste der Quellen mit Evidenzstufe in Worten (A «Studienlage
  stark», B «Einzelstudien», C «Coaching-Literatur», D «Praxisberichte»,
  Heuristik «Faustregel dieser App, wird aus Daten nachgeschärft»); Titel und
  Link aus `GET /v1/planner/sources`.
- **Keine Diagnose durch Belege (EXPL-07).** Reasons, die an eine Region mit
  Beschwerde oder Red Flag gebunden sind (SEL-03, INJ-*, SAFE-02/05/06),
  zeigen dem User nur die Evidenzstufe, keine Quellentitel: Titel wie die von
  D-64, D-68 oder D-89 nennen Erkrankungen und wären neben einer Beschwerde
  eine Verdachtsdiagnose (`05` §8, §9). Texte nennen die Region in den Worten
  der Körperkarte («innen am Ellbogen»), nie die Struktur dahinter
  (`biceps_distal` usw.). Die vollständigen IDs bleiben in `plan_decisions`
  und in der Entscheidungsspur für den User selbst.
- **«Was hat sich geändert?»** nach jeder Einheit aus `plan_changes[]`.
- **«Nicht im Plan»** aus `exclusions[]`, z. B. «Planche an Ringen ist
  pausiert, solange du Beschwerden innen am Ellbogen meldest.»
- Zeitangaben stets als Erfahrungswerte aus dem Coaching, nicht als Prognose
  (GOAL-05).

### 9.3 Sprachregeln (EXPL-04, KB-12)

| Verboten | Grund |
|---|---|
| Diagnosebegriffe und Verdachtsdiagnosen (z. B. Namen von Sehnen- oder Gelenkerkrankungen) in Texten an den User, auch in angezeigten Quellentiteln und Platzhaltern | keine Diagnosen (`05` §8, §9); EXPL-07 |
| «heilt», «lindert», «schützt vor» | PAR-D-38, PAR-D-39; Prehab-Texte nutzen das Evidenzlabel der Region |
| «ZNS-/CNS-Ermüdung» als Begründung | PAR-E-29 |
| «Schlaf verstärkt den Lerneffekt» | PAR-E-44 |
| «Rückstand», «aufholen», «verpasst», Countdown-Formulierungen | ADR 0003 |
| Zeitangaben ohne Kennzeichnung als Erfahrungswert | GOAL-05, `08` Rang 19 |

Die Prüfung läuft über eine Liste verbotener Wendungen je Sprache in
`content/training/` und in CI (KB-12).

### 9.4 Änderungsprotokoll und Nachvollzug (EXPL-03, EXPL-05)

- `plan_decisions` speichert jede Änderung mit Auslöser, Regel, Vorher/Nachher
  und Reasons. `GET /v1/me/plan/decisions` zeigt sie dem User.
- `GET /v1/me/plan?explain=trace` liefert dem User selbst die vollständige
  Entscheidungsspur (Kandidaten, Filter, verworfene Optionen mit Regel-ID); für
  Support und Tests. Nie für andere User.
- Ein gespeicherter Plan ist mit seinem `planner_knowledge`-Snapshot und dem
  Eingangs-Snapshot nachrechenbar (`input_hash`).

### 9.5 Optionale Sprachmodell-Schicht (EXPL-06)

Nicht in v1. Später höchstens zum Umformulieren und Übersetzen von
Reason-Texten und zum Beantworten von Fragen mit Plan und Reasons als einziger
Grundlage; sie erzeugt oder ändert keine Zahl und keine Übung (§1.1).

## 10. Schnittstellen

### 10.1 Ports und Kernfunktionen

```go
// internal/domain/planning (rein)
func ValidateKnowledge(kb Knowledge) []Issue
func Generate(kb Knowledge, s Snapshot, week Week) (Plan, error)
func Adapt(kb Knowledge, s Snapshot, ev Event) (Delta, []Change, error)
func Materialize(p PlannedSession, ids IDSource) (SessionDraft, error)

// internal/planning (Anwendungsdienst; implementiert von internal/store)
type SnapshotReader interface {
    Snapshot(ctx context.Context, userID uuid.UUID, asOf time.Time) (planning.Snapshot, error)
}
type PlanStore interface {
    ActivePlan(ctx context.Context, userID uuid.UUID, week planning.Week) (planning.Plan, bool, error)
    SavePlan(ctx context.Context, p planning.Plan) error
    ApplyDelta(ctx context.Context, userID uuid.UUID, d planning.Delta) error
}
type DecisionLog interface {
    Seen(ctx context.Context, userID uuid.UUID, trigger string, sourceID uuid.UUID) (bool, error)
    Record(ctx context.Context, userID uuid.UUID, cs []planning.Change) error
}
type KnowledgeSource interface {
    Current(ctx context.Context) (planning.Knowledge, error)
}
type Clock interface{ Now() time.Time }
type IDSource interface{ New() uuid.UUID }
```

Fehler werden mit `%w` gewickelt; der Kern gibt Fehler nur für ungültige
Eingaben zurück (z. B. Snapshot verweist auf unbekannte Übung), nie `panic`.

### 10.2 Einbindung ins Log-System

- **Start** (`POST /v1/me/plan/sessions/{id}/start`): Der Dienst erzeugt aus
  dem Plan eine `draft`-Session in den bestehenden Tabellen: Blöcke,
  Satz-Einträge mit `is_planned = true`, `kind`, `rest_after_planned_s`,
  Elemente mit den Zielwerten in `reps` bzw. `hold_seconds`, `load_kg` und
  Assistenz; `workout_sessions.planned_session_id` verweist auf die geplante
  Einheit. Jeder Satz-Eintrag bekommt eine neue UUIDv7 und in
  `set_entries.planned_item_id` die Item-ID des Plans; so ordnet die Adaption
  Ist zu Soll zu, auch wenn zwei Geräte dieselbe Einheit offline starten.
  Der Start ist idempotent (bestehender Draft wird zurückgegeben).
- **Ausführen** läuft unverändert über die Log-API bzw. den Sync (ADR 0007,
  0009): Aus einem geplanten Satz wird ein ausgeführter (`is_planned = false`,
  `completed_at`, Istwerte). Zielwerte bleiben im Plan erhalten.
- **Abschluss** (`POST /v1/sessions/{id}/complete`, unverändert idempotent):
  Nach dem Commit ruft der Dienst `Adapt`; die Antwort bekommt das optionale,
  additive Feld `plan_changes[]`. Über den Sync abgeschlossene Einheiten lösen
  denselben Pfad aus.
- **Schmerzberichte** sind Client-Zeilen (`user_pain_reports`) mit Sync; nach
  dem Commit wertet der Dienst sie aus (§8.6).
- **Geplante Ruhe und Deload.** Der Planer schreibt `user_training_days` nicht
  selbst. Ein geplanter Ruhetag wird über das bestehende Ruhetag-Loggen
  erfasst (ein Tipp in der App); eine im Deload abgeschlossene Einheit setzt
  beim Abschluss `deload` (ADR 0003, ADR 0008 unverändert; ENT-S-8).

### 10.3 Endpunkte (Tag `planning`, OpenAPI zuerst)

| Methode und Pfad | Zweck | Anmerkungen |
|---|---|---|
| `GET`, `PUT /v1/me/training-profile` | Profil, Verfügbarkeit, Equipment, Körpergewicht, Einwilligungen | `PUT` ersetzt ganz; Validierung nach `onboarding.md` §3 |
| `POST /v1/me/onboarding` | alle Antworten auf einmal; Ergebnis `needs_answers` (Rückfragen, höchstens 2), `blocked` (SAFE-02) oder `complete` (Start-Zustand, Realismus, Hinweise, erster Plan) | idempotent über `Idempotency-Key` wie die übrige API |
| `GET`, `PUT /v1/me/goals` | Ziele mit Priorität, Ziel-Stufe, Datum, Etappenziel | Änderung löst WEEK-08 aus |
| `GET /v1/me/plan` | Plan der laufenden Woche (`?week=YYYY-MM-DD`, `?explain=trace`) | `ETag`; erzeugt den Plan, falls keiner existiert |
| `POST /v1/me/plan/regenerate` | Plan ab der nächsten nicht begonnenen Einheit neu erzeugen | idempotent; gleiche Eingaben → gleicher Plan |
| `GET /v1/me/plan/sessions/{id}` | eine geplante Einheit | – |
| `POST /v1/me/plan/sessions/{id}/start` | Draft-Session erzeugen (§10.2); optional Check-in (ADAPT-17) | gibt `WorkoutSession` im bestehenden Schema zurück |
| `GET /v1/me/plan/decisions` | Änderungsprotokoll | Cursor-Paginierung |
| `POST /v1/me/pain-reports`, `GET /v1/me/pain-reports` | Schmerzberichte (auch über Sync) | nur mit Einwilligung gespeichert |
| `POST /v1/me/symptoms` | «Symptome beim Training melden» (Belastungssymptome, RF-10) | wirkt sofort (SAFE-02); ohne Einwilligung nur als Auflage gespeichert |
| `GET /v1/me/regions` | Zustand je Region mit Texten und Disclaimer | – |
| `POST /v1/me/regions/{region}/red-flags` | Antworten auf die Red-Flag-Fragen | ohne Einwilligung nur flüchtig ausgewertet |
| `POST /v1/me/regions/{region}/clearance`, `POST /v1/me/screening/clearance` | User bestätigt die Freigabe durch eine Fachperson | Zustandswechsel §8.3 |
| `GET /v1/me/capacity` | Kapazitäten mit Konfidenzklasse | für «Profil verfeinern» |
| `GET /v1/planner/rules`, `GET /v1/planner/sources`, `GET /v1/planner/parameters` | Katalog für die Erklärungen | öffentlich lesbar, versioniert über `ETag` |

Alle Anfragekörper werden gegen `api/openapi.yaml` validiert; Contract-Tests
prüfen Routen und Antworten (ADR 0007). Planungs-Antworten tragen
`ruleset_version` und `content_version_id`.

### 10.4 Fehler (application/problem+json)

| Typ | Status | Wann |
|---|---|---|
| `planning-unavailable` | 503 | Wissensbasis fehlt oder ist ungültig (§2.6) |
| `onboarding-required` | 409 | SAFE-01 |
| `training-stopped` | 409 | SAFE-02; der Körper nennt die empfohlene Abklärung ohne Diagnose |
| `validation` (bestehend) | 422 | ungültige Profil- oder Onboarding-Daten |

### 10.5 Offline und Sync

- Der Client hält den Plan der laufenden Woche im lokalen Speicher (`ETag`),
  wie die Skill-Karte (ADR 0011). Server-Tabellen des Planers sind für den
  Client nur lesbar.
- Offline startet der Client eine geplante Einheit selbst nach demselben
  Materialisierungsschema (§10.2, neue IDs, `planned_item_id`); der Sync lädt
  sie hoch, der Server adaptiert nach dem Commit.
- Schmerzberichte und Red-Flag-Antworten werden offline erfasst und
  synchronisiert. Der Plan-Payload enthält die Red-Flag-Fragen der Regionen
  mit Dringlichkeit und Aktion; eine N-Antwort beendet die Einheit auch
  offline, bevor der Server davon weiss.

### 10.6 Clients (ENT-1)

Die API bleibt client-neutral. Swift-DTOs werden aus `api/openapi.yaml`
erzeugt (CLAUDE.md); ein Svelte-Client (rung.fit) bekommt seine Typen ebenso
generiert, nie von Hand. Ob rung.fit ein neuer Produktname ist, bleibt zu
klären (ENT-1).

## 11. Performance

### 11.1 Grössen

| Grösse | Richtwert |
|---|---|
| Wissensbasis | ≈ 40 Skills, ≈ 200 Stufen, ≈ 300 Übungen, ≈ 30 Leitern, 10 Strukturen, 11 Regionen, 9 Beschwerdefamilien, ≈ 300 Parameter (davon 262 aus der Recherche und 42 `PAR-S`), ≈ 120 Regeln, 572 Quellen |
| Snapshot | ≤ 42 Tage Logs (PAR-B-73): ≤ ≈ 40 Einheiten × 40 Satz-Elemente ≈ 1 600 Elemente; ≤ ≈ 100 Kapazitäten |
| Plan | ≤ 7 Einheiten, ≤ ≈ 40 Sätze je Einheit (P ≤ 280) |

### 11.2 Aufwand

| Schritt | Komplexität | Grössenordnung |
|---|---|---|
| Wissensbasis validieren (beim Laden) | O(V + E) für Referenzen und Zyklen; O(R × F) Matrix | < 10 ms, einmal je Version |
| Belastungshistorie | O(Elemente × Strukturen) | ≈ 16 000 Operationen |
| Pfade | O(V + E) je Ziel | vernachlässigbar |
| Einheitsarten | Teilmengen der Tage, ≤ 2⁷ | vernachlässigbar |
| Verteilung | O(Leitern × Expositionen × Tage × Strukturen) | ≈ 12 × 4 × 7 × 10 |
| Auswahl und Dosierung | O(Plätze × Kandidaten × Filter) | ≈ 40 × 30 × 6 |
| Deckel und Kürzen | je Iteration O(P × Strukturen), höchstens O(P) Iterationen → O(P² × Strukturen) | ≤ ≈ 800 000 Operationen im schlechtesten Fall |
| Adaption je Einheit | O(Sätze × Strukturen) | vernachlässigbar |

### 11.3 Ziele und Messung

- `Generate` p95 < 50 ms, `Adapt` p95 < 10 ms auf dem Produktionsserver, ohne
  Datenbankzeit; Speicher je Aufruf < 5 MB.
- Datenbank: ≈ 8 indizierte Abfragen je Snapshot (alle über `user_id`, die
  Logs über den bestehenden Index `(user_id, local_date)`); Pläne werden
  gespeichert und nur bei Ereignissen neu erzeugt (§6.1).
- Phase 5 liefert Go-Benchmarks je Persona (`BenchmarkGenerate`); eine
  Regression über 2× gegenüber dem Referenzwert lässt den Test scheitern.

## 12. Tests und Personas

### 12.1 Eigenschaften, die für jeden Plan gelten

Geprüft über alle Personas und über zusätzlich erzeugte gültige Snapshots
(fester Seed, damit die Tests selbst deterministisch sind):

| Nr. | Eigenschaft |
|---|---|
| I-1 | Determinismus: zweimal erzeugt → byte-gleich; `input_hash` stabil |
| I-2 | Keine Übung mit Matrix-X für eine Region mit Beschwerde; nichts mit `r_eff` ≥ 1 auf einer gesperrten Region |
| I-3 | Wochendeckel, Einheitsdeckel und Straight-Arm-Budget eingehalten |
| I-4 | Abstände nach §7.5 eingehalten |
| I-5 | Kein Straight-Arm- oder Skill-Satz ohne Reserve; Tests und Prüfversuche nur, wenn §6.3/`onboarding.md` §4.3 es erlauben |
| I-6 | Jedes Element hat mindestens einen Reason; alle IDs existieren |
| I-7 | Kein Text verletzt §9.3 |
| I-8 | Geschätzte Dauer ≤ `session_minutes`, oder die Einheit ist auf Aufwärmen + Primärblock gekürzt und begründet |
| I-9 | Equipment: jede Übung ist mit dem Equipment des Users ausführbar |
| I-10 | Der Planer schreibt nie Unlocks, XP, Streaks oder `user_training_days`; Deload- und Ruhetage sind im Plan markiert |
| I-11 | Keine Region mit Beschwerde wächst schneller als ein beschwerdefreies neues Konto (§7.2) |

### 12.2 Tabellengetriebene Tests je Regel

Jede Regel in Anhang A hat eine Testtabelle mit Grenzfällen, z. B. DOSE-01:
d = 3.9 s → Sprosse tiefer; d = 4 s → h = 2 s (5 Sätze); d = 10 → 5 × 7 s;
d = 20 → 4 × 14 s; d = 30 → 3 × 21 s. LOAD-02: R = 0, Woche 1; Deload-Woche im Fenster;
Vorverletzung und Minderjährigkeit gleichzeitig (Faktor 0.5, nicht 0.25).

### 12.3 Golden Files

Je Persona ein erwarteter Plan (JSON) unter `testdata/`. Eine Änderung am
Golden File ist eine bewusste Entscheidung im Review und braucht einen
Changelog-Eintrag in `ruleset_version`.

### 12.4 Personas

| Persona | Eingaben (Kurzform) | Erwartete Eigenschaften |
|---|---|---|
| 1 Anfänger, Outdoor-Park, 2×/Woche, Muscle-up | `outdoor_park`, 2 × 45 min, 0–3 Klimmzüge, 0 Dips, Liegestütz 8–12, `lt_6_months` | 2 volle Ganzkörper-Einheiten (Mo, Do); Zubringer-Leitern Klimmzug (Exzentrik, Rudern am niedrigen Holm, Hang), Dip (Stütz, Exzentrik), Liegestütz; keine Band-Übungen (kein Band); kein Muscle-up-Block, Hinweis 5 + 5 als Minimum (§3.4, §3.5); Straight-Arm-Budget 8; Woche 1 mit 50 % (LOAD-04); lineare Doppelprogression |
| 2 Fortgeschritten, Gym, 4×/Woche, Planche + Front Lever | `gym`, 4 × 60–90 min, Tuck/Adv-Tuck-Stufen mit Halteklassen, `1_to_4_years` | Planche und Front Lever auf 3 Tagen mit ≥ 48 h Abstand, geblockt (gepaart nur bei Zeitmangel, §7.7); vierter Tag ohne Straight-Arm-Sprossen und ohne harte Zugreize (Beine, Rumpf, Balance, leichte Technik); Budget 12 bzw. 18 nach OG-Band; Pausen 300 s; wellenförmige Kraftarbeit; Kalibrierungssätze in den ersten Einheiten; Volumen startet bei 50 % (LOAD-04, ENT-S-1) |
| 3 Fortgeschritten, mediale Ellbogenbeschwerden, Ziel Planche | wie 2, Region `elbow_inner` aktuell, Alltagsschmerz 1–2, Training 3–4/10, keine Red Flag | Region `rtt_1` (Start 0.5, ohne geloggte Referenz zusätzlich unter LOAD-04); Planche-Familie M (Regression), Ringe-Straight-Arm X, Klimmzug M (Neutralgriff); 72 h zwischen harten Reizen der Region; Schmerz-Monitoring aktiv; keine Tests und Prüfversuche an der Region; Texte und Belege ohne Diagnose (EXPL-07) |
| 4 Wiedereinsteiger nach 6 Monaten | `17_to_26_weeks`, `pre_break_level` Adv Tuck Planche, 10 Klimmzüge | Pause aus dem Onboarding (§6.11); Straight-Arm- und Handgelenk-Konten in der Rampe ab Stufe 1 mit 25 % des Zielvolumens, die Rampe ersetzt den Wochendeckel (§15.2 U-13); Bent-Arm 50 %, 2 Sprossen unter der Angabe, dann nach Kalibrierung; σ × 1.25; keine Straight-Arm-Tests vor Ende der Rampe; Sprosse höchstens bis Adv Tuck während der Rampe |
| 5 Anfänger, Full Planche in 8 Wochen | Ziel Full Planche mit Datum, Stufe `none`, Liegestütz 4–7 | Realismus-Check: Untergrenze 48 Wochen, Spanne 105–162 Wochen, Etappenziel die Tuck Planche mit eigener Spanne (die Lean hat kein OG-Level, §15.2 U-19; Coaching nennt für sie 0–2 Monate, PAR-A-47); Plan ab den Wurzeln (Liegestütz, Stütz, Handgelenk, Hollow); kein Planche-Maximalblock; neutrale Texte |
| 6 Widersprüchliche Angaben | z. B. `sedentary`, 0 Liegestütze, aber Straddle Planche und Full Front Lever | höchstens 2 Rückfragen; danach Stufe nach R-2/R-3 (plausible Vorstufe), σ × 1.25, niedrige Konfidenz → Kalibrierungssätze; keine stille Übernahme des höheren Werts |

### 12.5 Szenarien über mehrere Wochen

| Szenario | Erwartung |
|---|---|
| Persona 3 meldet nach Einheit 2 Schmerz 6/10 | Red-Flag-Fragen; ohne Red Flag Rückschritt nach den Soreness Rules der Rampe (PAR-D-28), als Schmerz-Deload erfasst; dritte Verletzung in 14 Tagen → Verweis-Hinweis |
| Persona 4 über 8 Wochen ohne Beschwerden | Rampe 0.25 → 0.5 → 0.75 → 1.0, je Stufe ≥ 7 Tage; danach normale Deckel |
| Plateau an einer Sprosse | Deload der Folgewoche, danach Variation; kein zusätzliches Volumen über die Deckel |
| Zwei Einheiten an aufeinanderfolgenden Tagen | Tausch oder Herabstufung beim Start mit Begründung |
| Eine Woche ohne Training | kein Rückstandshinweis; Deckel gegen die Wochen mit Last |
| Red Flag RF-10 im Schmerzbericht | Training gestoppt, Plan erst nach Freigabe |
| Muscle-up-Leiter wird aktiv, sobald ihre Voraussetzungen erreicht sind (5 Klimmzüge, 8 Dips: die Schwellen der Stufen `pull-up/strict-5` und `dip/parallel-bars` aus `02`; die Faustregel «5 + 5» ist der Hinweis GOAL-04) | Leiter mit Rolle `goal` im Plan, mit Begründung |

Phase 5 dokumentiert die erzeugten Pläne je Persona und ihre Plausibilität in
`docs/algorithm/personas.md`.

## 13. Umsetzung, Migration und Rollout

### 13.1 Reihenfolge

1. **Phase 5 (Vorschlag, ENT-S-2):** `internal/domain/planning` (Kern),
   Wissensbasis-Dateien für die Personas (ENT-S-3), Laden und Validieren in
   `internal/content` und `cmd/contentlint`, `internal/planning` mit Ports und
   In-Memory-Adaptern, Tests nach §12, Persona-Dokumentation, unabhängiges
   Review.
2. **Danach:** OpenAPI-Erweiterung (zuerst), Migration, Store-Adapter,
   HTTP-Handler, DSL-Erweiterungen mit Golden Files, Client-Anbindung.

### 13.2 Migration

- Nur additive Änderungen: neue Tabellen, drei neue optionale Spalten
  (`workout_sessions.planned_session_id`, `set_entries.sir_s`,
  `set_entries.planned_item_id`). Keine
  bestehende Spalte wird umbenannt oder gelöscht; die laufende API bleibt
  während eines Deploys kompatibel (CLAUDE.md, «Expand, deploy, contract»).
- Inhalte kommen nie über Migrationen, sondern über den Seed (CLAUDE.md).
- `sqlc` nach der Schemaänderung neu erzeugen (`make sqlc`).

### 13.3 Rollout

- Schalter `PLANNER_ENABLED` (Konfiguration, Standard aus); Endpunkte liefern
  sonst `404`.
- Interner Test mit den Personas und echten Logs der Entwickler.
- Öffentlich erst, wenn (a) die Wissensbasis fachlich abgenommen ist
  (`draft_placeholder` entfernt, ENT-10), (b) die Rechtsfragen zu
  Gesundheitsdaten, Minderjährigen und dem Screening-Wortlaut geklärt sind
  (ENT-3, ENT-4, ENT-8; OE-1 bis OE-3).
- Beobachtung nur über serverseitige Logs (`slog` mit Request-ID): Laufzeit,
  Fehler, Anzahl je Regel-ID. Keine Gesundheitswerte in Logs, kein
  Analytics-SDK (CLAUDE.md).

### 13.4 Datenschutz

Gesundheitsangaben nur mit Einwilligung, in eigenen Tabellen, gelöscht beim
Widerruf und mit dem Konto (§4.9). Sicherheitsfragen ohne Einwilligung werden
flüchtig ausgewertet und nicht gespeichert (ENT-4, rechtlich zu prüfen).

## 14. Entscheidungen des Checkpoints nach Phase 4

Die Entscheidungen des Checkpoints nach Phase 3 sind umgesetzt (ENT-1 bis
ENT-10, OE-1 bis OE-9; Stellen in ADR 0012). Die folgenden Fragen dieser
Spezifikation hat der Checkpoint nach Phase 4 (27.09.2026) entschieden: **alle
Vorschläge angenommen.** Die Umsetzung steht an den genannten Stellen
(ENT-S-1: §7.4 LOAD-04b; ENT-S-9: §4.6, §8.4).

| Nr. | Frage | Optionen | Entscheidung (angenommen) |
|---|---|---|---|
| ENT-S-1 | Einstiegsrampe für User, die gerade trainieren (§7.4) | (a) wie spezifiziert: jede Belastungsart startet mit 50 % und wächst über die Deckel (Straight-Arm ≈ 15 Wochen bis 100 %); (b) für Konten, zu denen der User eine aktuelle Stufe angibt, bei `last_regular_training = current_or_lt_3_weeks`, ohne Beschwerde und Vorverletzung: 50 % → 75 % → 100 % in wöchentlichen Schritten, jeweils nur ohne Schmerzregel-Verletzung; danach normale Deckel | **(b)**: Wer die Belastung aktuell trägt, kehrt auf toleriertes Niveau zurück (dieselbe Logik wie `05` §6.2); neue Belastungsarten bleiben bei (a) |
| ENT-S-2 | Umfang von Phase 5 | (a) Kern, Wissensbasis, Validierung, Ports mit In-Memory-Adaptern, Tests; (b) zusätzlich Migration, Store, HTTP und OpenAPI | **(a)**; API und Schema nach einem eigenen Review, OpenAPI zuerst (ADR 0007) |
| ENT-S-3 | Inhalt der Wissensbasis in Phase 5 | (a) nur die Leitern, die die Personas brauchen (Klimmzug, Rudern, Hang, Dip, Stütz, Liegestütz, Planche, Front Lever, Handstand, Muscle-up, Hollow, L-Sit, Prehab); (b) alle Skills aus `02` | **(a)**, alle Einträge `draft_placeholder` bis zur fachlichen Abnahme (ENT-10) |
| ENT-S-4 | Reserve bei Halten loggen | (a) neue Spalte `set_entries.sir_s`; (b) `rir` für Halte als Sekunden deuten | **(a)**: eindeutige Semantik, `rir` bleibt Wiederholungen |
| ENT-S-5 | Risikofenster 6–48 Monate (PAR-D-04) | (a) Deckelfaktor 0.75 (`PAR-S-25`); (b) kein zusätzlicher Faktor, weil die Deckel schon konservativ sind | **(a)**, aber nur zusammen mit ENT-S-1 (b); sonst (b) |
| ENT-S-6 | Belastungseinheit ohne Haltedauer (§4.5) | (a) satzbasiert wie `05` §5.4; (b) Sätze × Haltedauer relativ zur Maximalhaltezeit | **(a)** für v1; mit Logs prüfen |
| ENT-S-7 | Sicherheitsauflagen ohne Einwilligung (§4.7) | (a) minimale Auflagen (Stopp, Regionen-Ausschluss) ohne Antworten und Werte speichern; (b) ohne Einwilligung kein Plan, sobald eine Sicherheitsfrage greift (Alternative aus OE-2) | **(a)**, vorbehaltlich der rechtlichen Prüfung (ENT-4); sonst (b) |
| ENT-S-8 | Geplante Ruhetage und der Streak (§5.4) | (a) Ruhetag mit einem Tipp über das bestehende Loggen (ADR 0008 unverändert); (b) geplante Ruhetage zählen am Tagesende automatisch (ADR 0008 ändern) | **(a)** für v1 |
| ENT-S-9 | Region «Brust» auf der Körperkarte | (a) hinzufügen (RF-01 nennt die Brust; heute nur über `other`); (b) bei `other` bleiben | **(a)**, mit Zuordnung zu `biceps_long_head_anterior_shoulder` und Dip-/Liegestütz-Familien nach fachlicher Prüfung |

## 15. Umsetzung in Phase 5: Entscheidungen, Abweichungen, Lücken

### 15.1 Stand

Umgesetzt nach ENT-S-2 (a): der reine Kern `internal/domain/planning`, die
Wissensbasis `content/training/` mit JSON-Schemas (`content/schema/training/`),
Loader und Validierung (`internal/content`, `cmd/contentlint`) sowie der
Anwendungsdienst `internal/planning` mit den Ports aus §10.1 und
In-Memory-Adaptern. Nicht umgesetzt: Migration, Store, HTTP-Endpunkte und
OpenAPI (nach eigenem Review, ADR 0007), `Materialize`.

Tests: sechs Personas als Golden Files (`internal/domain/planning/testdata/`),
die Eigenschaften I-1 bis I-11 für jeden erzeugten Plan, zwölf simulierte
Wochen je Persona, die Szenarien aus §12.5, Tabellentests je Regel, ein Test
je Validierungsprüfung KB-01 bis KB-12, ein Abgleich jedes
Forschungsparameters mit seiner Tabellenzeile in `docs/research/` und
Benchmarks (`Generate` 4–8 ms und 1–2 MB, `Adapt` ≈ 25 µs, `Build` ≈ 2.5 ms).
Die Pläne und ihre Plausibilität stehen in `personas.md`.

### 15.2 Entscheidungen und Abweichungen

Die Personas und die Simulation haben Lücken und Widersprüche dieser
Spezifikation gezeigt. Die Umsetzung entscheidet sie wie folgt; die mit
**Review** markierten Punkte ändern Verhalten, das der Checkpoint bestätigen
sollte.

| Nr. | Stelle | Umsetzung | Grund |
|---|---|---|---|
| U-1 | §2.1 | Neun Dateien (`manifest`, `parameters`, `rules`, `sources`, `body`, `exercises`, `skills`, `sessions`, `onboarding`); Übungen und Leitern in `content/training/` statt im `training:`-Block von `content/exercises/` | neue Übungen in `content/exercises/` wären bis zum Inhalts-Import Waisen und liessen `contentlint -strict` scheitern |
| U-2 | §2.3 | Jeder Parameter trägt `text`, den Wortlaut der Recherche-Tabelle, und `value`/`values`, die gelesene Zahl in der Einheit des Planers (z. B. 10 % → 0.10). Ein Test prüft Schlüssel, Wortlaut und Quellen gegen `docs/research/`. `definition: true` für Skalenbezüge ohne Quelle (PAR-C-08, PAR-C-22) | keine Zahl ohne nachprüfbare Herkunft |
| U-3 | §2.6 KB-13 | `contentlint` meldet den Entwurfsstatus als Hinweis, nicht als Warnung; die API lehnt Entwürfe im Produktionsmodus ab | der übrige Inhaltsbaum ist ebenfalls Entwurf und wird nicht angemahnt; ENT-10 bleibt Voraussetzung für Produktion |
| U-4 | §4.3, `PAR-S-31` | Eine Reserve über der Grenze zählt als Untergrenze bis zur Grenze: b = Halt + max(3 s, 0.5 × Halt) bzw. Wdh. + 3 | sonst ergeben Prüfversuche (≤ 5 s, `PAR-S-24`) nie einen Dosiswert ≥ 4 s, und ADAPT-06 ist unerreichbar |
| U-5 | §4.3 | Die erste Beobachtung einer Übung aktualisiert den abgeleiteten Startwert, den auch die Planung benutzt (`PAR-S-39`); eine Untergrenze ohne jeden Startwert wird Startwert mit σ = 0.35 μ (PAR-F-26) | Planung und Adaption rechnen mit derselben Schätzung |
| U-6 | §5.4 WEEK-02 | Ein Tag, an dem ausser Aufwärmen und Prehab nichts übrig bleibt, wird geplanter Ruhetag; leichte Leitern bevorzugen weniger belegte Tage | keine «Einheiten» nur aus Prehab |
| U-7 | §5.5 | Prehab höchstens zwei verschiedene Programme (Schulter, Handgelenk nach PAR-D-01), an PAR-D-37 Einheiten je Woche | zwei Schulterregionen teilen ein Programm |
| U-8 | §5.6 SEL-09 | Zwischen d < 1 (Exzentrik) und dem Wiederholungsbereich: konzentrisch unter dem Bereich, sobald eine Wdh. mit Ziel-RIR möglich ist (d − RIR ≥ 1); findet keine Sprosse einen Wert, beginnt die Kalibrierung eine Sprosse unter der angegebenen, nicht an der Wurzel | die Lücke war nicht geregelt; wer 1–6 Klimmzüge schafft, übte sonst nur Negative |
| U-9 | §5.7 DOSE-04 | Konditionshalt ohne Schätzung: Kalibrierungssatz 10 s (untere Grenze PAR-B-76) mit 2 s Reserve | sonst 5 s ohne Reserve |
| U-10 | §6.3 | Angebote (ADAPT-05, ADAPT-10) auch für Zubringer; bei einer Band-Sprosse ist das Angebot dieselbe Sprosse ohne Band; Angebote gehören zum Maximalblock und werden in LOAD-10 Schritt 5 zuletzt gestrichen | sonst kam kein Band-Nutzer und kein Anfänger je an ein Angebot |
| U-11 | §6.4 ADAPT-10 | Der erste konzentrische Versuch ist ein Kalibrierungssatz (so viele saubere Wdh. wie die Reserve erlaubt) | ein einzelner Versuch liefert nur eine Untergrenze |
| U-12 | §6.8 | Deload-Einheiten sind kein Plateau-Beleg; der Abstand von `PAR-S-17` zählt ab der letzten Deload-Einheit | sonst folgte auf jeden Deload ein zweiter |
| U-13 | §6.11, §7.2 | **Review.** Die Pausenrampe ersetzt für Straight-Arm- und Handgelenk-Konten den Wochendeckel auch ohne geloggte Referenz (wie LOAD-04b); Einheitsdeckel in der Rampe 1.5 × Maximum; die Stufen wechseln zu Wochenbeginn | mit LOAD-02 auf dem nachlaufenden Mittel blieb ein Wiedereinsteiger monatelang bei ≈ 1 Straight-Arm-Satz je Woche; §12.5 erwartet 0.25 → 1.0 in ≥ 7-Tage-Schritten. Der Stand vor der Pause (`pre_break_level`) ist toleriertes Niveau wie bei ENT-S-1 |
| U-14 | §7.2 | Ein Deckel trägt die Regel, die tatsächlich bindet; der Spielraum-Übertrag (`PAR-S-35`) gilt für LOAD-02 | vorher benannte die Pausenregel auch Deckel, die LOAD-02 setzte, und der Übertrag fiel weg |
| U-15 | §7.9 LOAD-10 | Untergrenze 1 Satz je Arbeitsübung bis Schritt 7 (statt 2); innerhalb eines Schritts zuerst die Übung mit den meisten Sätzen; Schritt 7 streicht nach Priorität | mit Untergrenze 2 erreichte der halbe Einstieg (LOAD-04) die 50 % nur durch Streichen ganzer Übungen |
| U-16 | §8.6 | Eine Einheit zählt für die Rampe einer Beschwerde, wenn der Morgenbericht grün ist; ohne jeden Basiswert wird die Morgenregel nicht geprüft, nur die Schwellen | PAR-D-26 fragt nach dem Folgetag; ein fehlender Basiswert machte aus 1/10 eine Verletzung der Schmerzregel |
| U-17 | §8.7 INJ-08 | Die dritte Verletzung in 14 Tagen erzeugt immer den Verweis-Hinweis, auch wenn ein Timer schon riet | Hinweis zum Anlass |
| U-18 | §8.3 | Ein Stopp durch eine Red Flag gehört zur Region; ihre Freigabe hebt ihn auf; Belastungssymptome ohne Region brauchen die allgemeine Freigabe | vorher blieb der Plan nach der Freigabe gestoppt |
| U-19 | §3.6 | Etappenziel ist die erste offene Stufe mit höherem OG-Level (die Lean hat keins); für Persona 5 die Tuck Planche | ohne OG-Stufe gibt es kein Band nach PAR-A-45 |
| U-20 | `onboarding.md` §5.5 | Offene oder verneinte Rückfragen lösen konservativ auf: R-3 eine Stufe tiefer, danach R-2 bis zur höchsten Stufe mit plausiblen Vorstufen (≥ `PAR-S-46` × Schwelle) | die Regel verlangte die Auflösung, ohne sie zu beziffern |
| U-21 | §10.1 | `Adapt` gibt den ganzen neuen Snapshot zurück (kein Delta); `SnapshotStore` speichert ihn; ohne `Materialize` und `IDSource` | einfacher und ausreichend, solange es keinen Store gibt |
| U-22 | §12.3, §11.3 | Golden Files als lesbarer Text; die Benchmark-Grenze ist kein Test (Zeitmessung in CI schwankt) | Reviewbarkeit |
| U-23 | §5.10 | Der Plan listet je Lastkonto Zielvolumen, geplantes Volumen, Deckel und bindende Regel (`loads`) | Erklärbarkeit; I-3 und I-11 werden damit prüfbar |

### 15.3 Nicht umgesetzt

GOAL-03 (Erhaltungsdosis; es gibt nur den Hinweis WEEK-06), SEL-05
(Mobilitätsantworten ändern die Auswahl noch nicht), SEL-10 über «eine Sprosse
tiefer» und Band hinaus (die Wissensbasis hat keine Griff- oder
Gerätevarianten), SEL-11, SESS-10, SESS-11, ADAPT-08 als eigene Progression je
Expositionsklasse (DOSE-06 dosiert wellenförmig, die Progression läuft über die
Kapazität), ADAPT-09, ADAPT-11, ADAPT-17, der Mobilitätsblock und Texte in
weiteren Sprachen (KB-11). Für die Ellbogen-Regionen gibt es kein Prehab: Die
Recherche nennt Programme, aber keine übertragbare Übung (`05` §10).
Minderjährige bekommen keinen Plan (SAFE-07); INJ-09 ist deshalb nicht aktiv.

### 15.4 Befunde für den Review

- **Woche 1 ist kurz.** Mit dem halben Einstieg (LOAD-04, LOAD-04b) dauern die
  ersten Einheiten 11–30 min bei 45–60 min Budget. Gewollt vorsichtig, nutzt
  aber die Zeit kaum; Kandidat für Technik- oder Mobilitätsarbeit.
- **Abgeleitete Startwerte sind Untergrenzen.** Wer eine Tuck Planche 10–19 s
  hält, beginnt mit 2 × 4 s Lean als Kalibrierung (SEL-08 plus `PAR-S-39`); die
  erste Einheit korrigiert den Wert.
- **Die Plateau-Definition ist streng.** Mit `PAR-S-12` (zwei Einheiten ohne
  Zuwachs) plant die Simulation etwa alle 4–5 Wochen einen Stagnations-Deload,
  weil Leistung in Wochenschritten statt je Einheit wächst.
- **Kleine Straight-Arm-Volumina wachsen langsam.** Ausserhalb der Rampen
  wächst ein Konto mit 1–2 Sätzen unter LOAD-02 um einen Satz in etwa 10–13
  Wochen (c = 10 %, im Risikofenster f = 0.75).

## Anhang A: Regelkatalog

Evidenz: A–D nach `00_sources.md`; H = Heuristik (Begründung im Abschnitt).

| Regel | Inhalt | Abschnitt | Parameter | Evidenz |
|---|---|---|---|---|
| SAFE-01 | Onboarding und Hinweis vor jedem Plan | §5.2 | – | Projektvorgabe |
| SAFE-02 | Belastungssymptome oder N-Red-Flag → kein Plan bis Freigabe | §5.2, §8.2 | PAR-F-45, PAR-D-21 | B |
| SAFE-03 | Screening «ja» → keine Tests und Prüfversuche bis Freigabe | §5.2 | – | B (F-41) / H |
| SAFE-04 | Ohne Einwilligung konservativ | §5.2 | PAR-D-02 | H |
| SAFE-05 | Gesperrte Region: nichts mit Last ≥ 1 | §5.2 | PAR-D-21, PAR-S-22 | H |
| SAFE-06 | Rampenstufe 0: nichts mit Last ≥ 2 | §5.2 | PAR-S-22 | H |
| SAFE-07 | Minderjährige: kein Plan bis zur Entscheidung ENT-3 | §5.2 | – | Projektvorgabe |
| GOAL-01 | Bis 3 Ziele nach Priorität | §5.3 | – | H |
| GOAL-02 | Pfad, Zubringer, Unterstützung | §3.4 | PAR-A-62 | C/H |
| GOAL-03 | Erhaltungsdosis bei Zeitmangel | §5.3 | PAR-B-63, PAR-B-34 | B |
| GOAL-04 | Bereitschaftshinweise nie als Sperre | §3.5 | PAR-F-42, PAR-A-52, 53, 69, PAR-F-62–64 | C/D |
| GOAL-05 | Realismus-Check | §3.6 | PAR-A-45, 47, 51, PAR-S-19 | D/H |
| GOAL-06 | Gegenspieler im Kraftblock | §5.3 | PAR-B-64–66 | H |
| WEEK-01 | Tage | §5.4 | PAR-S-06 | H |
| WEEK-02 | volle und leichte Einheiten, geplante Ruhe | §5.4 | PAR-B-36, PAR-S-20 | B/H |
| WEEK-03 | Split aus Abständen | §5.4 | PAR-B-37 | H (A für Volumengleichheit) |
| WEEK-04 | Frequenz je Leiter | §5.4 | PAR-B-34, PAR-E-11, PAR-E-12 | A (Übertragung)/H |
| WEEK-05 | Verteilung | §5.4 | PAR-B-48, PAR-B-81 | H |
| WEEK-06 | Zu wenig Tage → Erhaltung | §5.4 | PAR-B-63 | B |
| WEEK-07 | Deload-Woche | §5.4 | PAR-B-03, PAR-B-50–54 | B |
| WEEK-08 | Neu erzeugen | §5.4 | – | – |
| SESS-01 | Vorlage nach Minuten | §5.5 | PAR-B-64–67 | H/C |
| SESS-02 | Reihenfolge | §5.5 | PAR-B-46, PAR-E-01–03 | A |
| SESS-03 | Aufwärmen, Prehab-Aktivierung, Rampensätze | §5.5 | PAR-B-69–71, PAR-D-01, 05, 37, 38, PAR-E-26, 45–47 | A/B/H |
| SESS-04 | Balanceblock | §5.5 | PAR-E-35, 36, 38, PAR-B-35, PAR-B-74 | A (Analogie)/H |
| SESS-05 | Maximalblöcke geblockt, Paare | §5.5 | PAR-E-32, PAR-B-81 | H |
| SESS-06 | Volumenblöcke | §5.5 | PAR-B-10–12 | B/C |
| SESS-07 | Kraftblock, Supersätze | §5.5 | PAR-B-45 | B |
| SESS-08 | Belastende Prehab am Ende | §5.5 | PAR-E-01 | H |
| SESS-09 | Leichte Einheit | §5.5 | PAR-E-12, PAR-E-14 | H |
| SESS-10 | Hypertrophie als Nebenziel | §5.5 | PAR-B-21, 75, 79, PAR-C-46 | A/B/H |
| SESS-11 | Greasing the Groove (optional, v1 aus) | §5.5 | PAR-B-80, PAR-E-27, 48, 49 | C/D/H |
| SEL-01 | Kandidaten | §5.6 | – | – |
| SEL-02 | Equipment | §5.6 | – | – |
| SEL-03 | Regionen, Matrix, Vorgaben | §5.6, §8.4 | `05` §8 | H |
| SEL-04 | Supinierte Straight-Arm-Varianten | §5.6 | PAR-D-41, PAR-D-12, PAR-A-23 | B/H |
| SEL-05 | Mobilität steuert Auswahl | §5.6 | PAR-F-53, PAR-C-27 | H |
| SEL-06 | Minderjährige | §5.6 | PAR-D-23 | H |
| SEL-07 | Arbeitssprosse Halte (d ≥ 4 s) | §5.6 | PAR-B-05, PAR-B-07, PAR-C-13, PAR-C-61 | H |
| SEL-08 | Einstieg eine Sprosse tiefer | §5.6 | – | H (`onboarding.md` §5.4) |
| SEL-09 | Arbeitssprosse Wdh. | §5.6 | PAR-B-23, PAR-B-16, PAR-B-77 | H/C |
| SEL-10 | Ersatzreihenfolge | §5.6 | PAR-C-27 | H |
| SEL-11 | Variation zwischen Einheiten | §5.6 | PAR-E-24, PAR-E-34 | A/B |
| SEL-12 | Unplausible Angabe | §5.6, `onboarding.md` §5.5 | PAR-S-44, PAR-S-46 | H |
| DOSE-01 | Maximalblock Halt | §5.7 | PAR-S-01, 02, 05, PAR-B-08, 33, PAR-E-04, 05, 08 | C/H |
| DOSE-02 | Maximalblock Wdh. | §5.7 | PAR-E-04, 08, 18 | C/H |
| DOSE-03 | Volumenblock | §5.7 | PAR-B-10–12, PAR-E-06 | B/C |
| DOSE-04 | Konditionsstatik | §5.7 | PAR-B-76, PAR-E-07 | C |
| DOSE-05 | Kraft, Anfänger | §5.7 | PAR-A-01–05, PAR-B-01, 24, 42, PAR-S-29 | A/D |
| DOSE-06 | Kraft, wellenförmig | §5.7 | PAR-B-02, 17–19, 24, 42, 43, PAR-S-13, PAR-S-29 | A/H |
| DOSE-07 | Exzentrik | §5.7 | PAR-B-16, PAR-B-77 | C |
| DOSE-08 | Balance | §5.7 | PAR-E-35, 36, 38, PAR-B-74 | A (Analogie)/H |
| DOSE-09 | Technik | §5.7 | PAR-S-26, PAR-E-14, PAR-E-38 | H |
| DOSE-10 | Prehab | §5.7 | PAR-B-78, PAR-B-44, PAR-D-37 | B |
| DOSE-12 | Zubringer, Unterstützung, Antagonisten | §5.7 | PAR-B-78, PAR-B-45, PAR-E-07 | B |
| DOSE-11 | Zusatzlast | §5.7 | PAR-B-23, PAR-B-32 | B |
| DOSE-20–23 | Stoppregeln | §5.9 | PAR-E-15–17, PAR-D-15 | H/A (Reha) |
| LOAD-01 | Belastungseinheit | §4.5 | PAR-C-27, 40, 44, 45, 47, PAR-B-79, PAR-S-08, PAR-S-15 | H |
| LOAD-02 | Wochendeckel, Rampen, ganze Sätze | §7.2 | PAR-D-02, 04, 09–11, 23, PAR-B-55, PAR-S-14, 15, 25, 35, 40 | H |
| LOAD-03 | Einheitsdeckel | §7.3 | PAR-D-31, PAR-S-48 | B (Analogie) |
| LOAD-04 | Neue Belastungsart | §7.4 | PAR-D-12 | H |
| LOAD-04b | Einstieg für aktuell Trainierende | §7.4 | PAR-D-12, PAR-D-25, PAR-S-43 | H |
| LOAD-05 | Abstände | §7.5 | PAR-D-03, 08, 34, PAR-B-38, PAR-E-13, 14, PAR-S-07, PAR-S-30 | B/H |
| LOAD-06 | Straight-Arm-Budget | §7.6 | PAR-B-47, PAR-A-23, PAR-S-09, PAR-S-23 | H |
| LOAD-07 | Gleiche Richtung | §7.7 | PAR-B-48 | H |
| LOAD-08 | Paarung Gegenrichtung | §7.7 | PAR-B-81, PAR-B-45, PAR-E-04, PAR-E-33, PAR-S-10 | H |
| LOAD-09 | Schutz neuer Sprossen | §7.8 | PAR-D-06, PAR-S-27 | A/B (Zeitverlauf)/H |
| LOAD-10 | Kürzen | §7.9 | – | H |
| LOAD-11 | Keine ACWR-Sperre | §7.10 | PAR-B-58, PAR-D-30, PAR-B-72, 73 | A/B |
| ADAPT-01–03 | Kapazität, erster Satz, Widerspruch, abgeleitete Startwerte | §4.3, §6.2 | PAR-F-01, 02, 16, 20–26, 28, 32, 33, 41, 55, 68, PAR-E-31, PAR-S-03, 04, 21, 31, 38, 39, 44, 45 | B/H |
| ADAPT-04 | Haltezeit wächst höchstens +2 s je Woche | §6.3 | PAR-B-33 | H |
| ADAPT-05 | Prüfsprosse anbieten | §6.3 | PAR-B-05, PAR-B-30, PAR-A-78, PAR-S-24 | B/C/H |
| ADAPT-06 | Sprosse wechseln | §6.3 | PAR-A-65, PAR-B-57 | C/H |
| ADAPT-06a | Bedingungen für Prüfversuche | §6.3 | PAR-D-12, PAR-D-29 | H |
| ADAPT-07 | Lineare Doppelprogression | §6.4 | PAR-A-01–05, PAR-B-01, PAR-B-23, PAR-S-33 | A/D/H |
| ADAPT-08 | Wellenförmige Progression | §6.4 | PAR-B-02, PAR-B-30 | A/B |
| ADAPT-09 | Zusatzlast steigern | §6.4 | PAR-B-23, 30, 32 | B |
| ADAPT-10 | Exzentrik zur ersten Wdh. | §6.4 | PAR-B-16, 77, PAR-A-66 | C |
| ADAPT-11 | Autoregulation | §6.6 | PAR-B-28, 29, 31, PAR-F-25 | A/H |
| ADAPT-12 | Regression | §6.7 | PAR-E-15, 16, PAR-D-18 | H |
| ADAPT-13 | Plateau | §6.8 | PAR-B-49, PAR-E-20, 34, 37, PAR-S-12, PAR-S-17 | B/H |
| ADAPT-14 | Deload geplant und ausgelöst | §6.9 | PAR-B-03, 49–54, PAR-D-18, PAR-S-40 | B/H |
| ADAPT-15 | Verpasste Einheiten | §6.10 | PAR-D-31, PAR-S-16 | H |
| ADAPT-16 | Pausen und Wiedereinstieg, Pause aus dem Onboarding | §6.11 | PAR-B-59–62, PAR-D-29, 33, PAR-S-41 | B/H |
| ADAPT-17 | Check-in | §6.12 | PAR-E-19, 41–43, PAR-S-28 | A/H |
| ADAPT-18 | Unlocks und Profiländerungen | §6.13 | – | – |
| INJ-01, INJ-02 | Red Flags: wann, welche, Aktion | §8.2 | PAR-D-21, RF-01–RF-13 | B/H |
| INJ-03 | Zustandsautomat | §8.3 | PAR-D-14, 19, 21, 24, 26, 28, 29, 33, PAR-S-32 | B/H |
| INJ-04 | Matrix | §8.4 | `05` §8 | H |
| INJ-05 | Modifikation und Ersatz | §8.5 | PAR-C-27, PAR-D-41 | B/H |
| INJ-06 | Rampe | §8.6 | PAR-D-24–27, 33, 34, PAR-S-47 | B/H |
| INJ-07 | Schmerz-Monitoring | §8.6 | PAR-D-13–18, 28, 39, 40, PAR-S-34, PAR-S-42, PAR-S-47 | A (Reha)/H |
| INJ-08 | Verweise | §8.7 | PAR-D-19, 20, 32 | A/B/H |
| INJ-09 | Minderjährige | §8.8 | PAR-D-22, 23, 42 | H |
| INJ-10 | Vorgaben von Fachpersonen | §8.9 | – | H |
| EXPL-01–07 | Reasons, IDs, Protokoll, Sprachregeln, Spur, Sprachmodell, keine Diagnose durch Belege | §9 | PAR-D-38, 39, PAR-E-29, 44 | – |
| KB-01–13 | Validierung der Wissensbasis | §2.6 | – | – |

## Anhang B: Spezifikations-Parameter (`PAR-S`)

Alle Werte sind **Heuristik**. Die Spalte «Anker» nennt, woran der Wert
festgemacht ist.

| ID | Schlüssel | Wert | Anker und Begründung |
|---|---|---|---|
| PAR-S-01 | `set_hold_fraction_default` | 0.70 | Mitte der abgestimmten Spanne 0.65–0.75 (`08` §4; PAR-A-64, PAR-B-06) |
| PAR-S-02 | `static_target_total_hold_s` | 60 | reproduziert die OG-«Sweet Spots» (PAR-A-64) und liegt in 30–90 s (PAR-B-09) bzw. 40–90 s (PAR-B-12) |
| PAR-S-03 | `skill_static_obs_noise_frac` | 0.25 | zwischen Ausdauer- (0.15) und Rumpfhalten (0.40), gleich Gleichgewichtshalten (PAR-F-16); keine Reliabilitätsdaten für Hebel-Statics (`08` §6.1 Nr. 3) |
| PAR-S-04 | `hold_capacity_drift_frac_per_week` | 0.05 × μ | Halte-Gegenstück zu PAR-F-28 (0.5 Wdh. je Woche ≈ 5 % einer Kapazität von 10 Wdh.) |
| PAR-S-05 | `heaviest_element_min_og_ordinal` | 14 | PAR-E-05 nennt die Maltese und «vergleichbare schwerste Elemente»; OG 14 = Full Planche an Ringen, darüber Inverted Cross, Maltese (`02` §3.4) |
| PAR-S-06 | `default_day_patterns` | §5.4 | grösster kleinster Abstand; PAR-D-08 und PAR-E-13 verlangen 48 h zwischen harten Reizen |
| PAR-S-07 | `spacing_min_rating` | 2 | Profilwert 1 = geringe Last (PAR-C-45); sonst würde fast jede Übung jede andere sperren |
| PAR-S-08 | `warmup_set_load_weight` | 0.5; Rampensätze mit höchstens der halben Satzhaltezeit bzw. Ziel-Wdh. | Rampensätze sind submaximal (PAR-B-70), laufen aber an Sprossen mit hohem Moment (vgl. `PAR-S-30`); halbe Dauer und halbes Gewicht als vorsichtiger Kompromiss |
| PAR-S-09 | `straight_arm_budget_priority_weights` | 3 : 2 : 1, mindestens 2 Sätze je Maximalblock | Priorität bestimmt die Zuteilung knapper Ressourcen (`onboarding.md` §3.2, PAR-B-81); 2 = Untergrenze von PAR-B-08 |
| PAR-S-10 | `paired_block_gap_s` | max(PAR-B-45, Pause des Skills / 2) | vereint PAR-B-81 und PAR-E-33 (§7.7) |
| PAR-S-11 | `duration_model` | 3 s je Wdh.; 30 s Wechsel je Übung | grobe Zeitschätzung für die Kürzungsregel; im Usability-Test messen |
| PAR-S-12 | `plateau_definition` | erster Arbeitssatz in 2 Einheiten nicht über dem Wert davor | Umsetzung von PAR-B-49 a am ersten, frischen Satz (PAR-E-31) |
| PAR-S-13 | `undulating_rep_ranges` | schwer 3–6, mittel 6–10, leicht 10–15; Reihenfolge schwer, leicht, mittel | Bereiche aus PAR-B-17 und PAR-B-19, Beispiel 15/10/5 (B-115); Reihenfolge trennt die schweren Tage |
| PAR-S-14 | `reference_week_mean` | bis zu 3 Wochen mit Last > 0, ohne Deload, im 6-Wochen-Fenster | PAR-D-09–11 (3-Wochen-Mittel), PAR-B-73 (6 Wochen), PAR-B-59 (kurze Pausen ohne Rampe) |
| PAR-S-15 | `load_accounts_and_rates` | §4.5, §7.1: alle SA-Konten und `wrist` +10 %, alle BA-Konten +20 % | PAR-D-09–11; Konten trennen gestreckte und gebeugte Arme, damit der strengere Deckel nur die Straight-Arm-Last bremst; für SA-Strukturen ausserhalb von PAR-D-09 gilt der strengere Wert (`08` §4) |
| PAR-S-16 | `session_queue` | Warteschlange je Woche, kein Nachholen, stilles Verfallen | ADR 0003; Nachholen würde PAR-D-31 verletzen |
| PAR-S-17 | `plateau_guards` | ≥ 4 Wochen an der Sprosse; ≥ 3 Wochen zwischen Stagnations-Deloads | neuronale Frühphase 3–5 Wochen (PAR-E-20); verhindert Deload-Ketten durch Messrauschen (PAR-F-01) |
| PAR-S-18 | `rounding` | Wdh., Sätze und Sekunden abrunden; Pausen auf 15 s; Laststeigerung auf die kleinste Scheibe aufrunden, mindestens eine Scheibe (PAR-B-32); Übung mit < 1 Satz entfällt, ausser im Deload (mindestens 1, WEEK-07) | Abrunden ist die sichere Richtung (S-2, `08` §2.2); die Laststeigerung rundet PAR-B-32 selbst auf, sonst stiege die Last nie |
| PAR-S-19 | `realism_prior_scaling_enabled` | false | dass relative Hebelanforderung (PAR-C-32, PAR-C-54) die Lernzeit proportional verlängert, ist nicht belegt |
| PAR-S-20 | `experience_class_mapping` | §4.2 | Grenze 6 Monate aus PAR-B-01; `highly_trained` aus PAR-F-48 |
| PAR-S-21 | `contradiction_confirmation` | zweite Abweichung in dieselbe Richtung → Mittel beider Beobachtungen, σ = r | verhindert, dass PAR-F-33 eine echte Veränderung dauerhaft blockiert |
| PAR-S-22 | `exclusion_rating_thresholds` | `locked`: ≥ 1; `rtt_0`: ≥ 2 | Red Flag: jede Last meiden (PAR-D-21); Rampenstufe 0 ohne Red Flag: gezielte Last meiden, geringe Nebenlast erlauben (`05` §6.2 «nicht geplant») |
| PAR-S-23 | `straight_arm_budget_by_og_band` | 8 / 12 / 18 | Spanne 8–12 und 18 aus PAR-B-47 auf die OG-Bänder (PAR-A-23) gelegt |
| PAR-S-24 | `probe_attempts` | 2 Versuche à ≤ 5 s | GMB testet die nächste Stufe mit 3–5-s-Halten (A-35, C); 2 statt bis zu 8 Sätze wegen des Budgets (PAR-B-47) |
| PAR-S-25 | `risk_window_cap_factor` | 0.75 | PAR-D-04 verlangt vorsichtigere Steigerung ohne Betrag; zwischen 1 und PAR-D-02 (0.5); ENT-S-5 |
| PAR-S-26 | `technique_dose` | h = min(0.5 · d, 10 s), 3–5 Versuche | submaximale Technik (PAR-E-14) mit grosser Reserve; Obergrenze hält sie `leicht` |
| PAR-S-27 | `new_rung_protection` | Volumen an der alten Sprosse, keine übernächste Prüfsprosse, für PAR-D-06 Wochen | Umsetzung von S-2 (`08` §2.2) ohne zusätzliche Zahl |
| PAR-S-28 | `checkin_fatigue_threshold` | ≥ 8 | gleiche Schwelle wie PAR-B-49 b |
| PAR-S-29 | `rest_lower_bound_choice` | Kraft: Anfänger 120 s, Trainierte schwer 180 s | untere Grenzen von PAR-B-42 wegen des Zeitbudgets (PAR-B-68), wie `08` §4 bei Leans |
| PAR-S-30 | `volume_block_counts_as_hard` | true | Volumen an der Sprosse unter der Arbeitssprosse hat noch ≈ 80 % des Moments (PAR-C-03–07); Sehnenlast hängt an der Lasthöhe (C-65, S-2) |
| PAR-S-31 | `lower_bound_observation` | wirkt nur, wenn b > μ, dann wie x = b; Halte zählen voll nur mit SIR ≤ max(3 s, 0.5 × Halt); grössere Reserven zählen als Untergrenze bis zu dieser Grenze (Wdh.: bis RIR 3) | Sätze mit RIR > 3 sind nur Untergrenzen (PAR-F-24); die Halte-Grenze ist das Gegenstück (grosse Reserve = ungenaue Schätzung, F-08) |
| PAR-S-32 | `rtt5_to_normal_weeks` | 2 Wochen ohne Regelverletzung; X-Zellen erst ab `normal`, dann als neue Belastungsart | Stufe 5 hat keine Weiter-Bedingung in `05` §6.2; 2 Wochen = zwei Durchgänge von PAR-D-26 |
| PAR-S-33 | `novice_next_rung_min_dose_reps` | 6 | 5 Wdh. (PAR-A-04) + 1 RIR (PAR-B-24) |
| PAR-S-34 | `missing_pain_report_policy` | keine Rampen-Progression, sonst keine Folge | konservativ ohne Strafe (ADR 0003); PAR-D-26 verlangt beschwerdefreie Einheiten |
| PAR-S-35 | `set_headroom_carry` | Rest-Spielraum je Konto in die Folgewoche übertragen; höchstens +1 Satz je Übung und Woche | verhindert, dass ganze Sätze bei kleinen Volumen nie wachsen; +1 Satz aus PAR-B-55 |
| PAR-S-36 | `point_values` | Punktwerte der Tabelle in §5.7, SESS-01, SESS-04 | Spannen der Quellen brauchen für einen deterministischen Plan einen Wert; Wahl nach `08` §4 (Sicherheit vorsichtig, sonst Mitte) |
| PAR-S-37 | `misc_small_values` | 3 min allgemeines Aufwärmen bei 5 min Gesamtdauer; Kantengewicht ≥ 0.5 für Unterstützungsübungen; +1 s je Exzentrik-Wdh. und Einheit; 1.25 kg kleinste Scheibe ohne Angabe | 3 min: Rest für Rampensätze; 0.5 = Mitte der `recommended`-Gewichte (PAR-A-62); +1 s führt in ≈ 4–7 Einheiten von 3 auf 7–10 s (PAR-B-16); 1.25 kg = übliche kleinste Hantelscheibe |
| PAR-S-38 | `estimate_floors` | offene Halteklassen μ = halbe Obergrenze; σ ≥ max(1 s, 0.15 · μ) bzw. 2 Wdh.; r ≥ 1 s bzw. 1 Wdh. | verhindert σ = 0 und undefinierte Konfidenz bei Nullwerten; 2 Wdh. aus PAR-F-20. **Review:** Der Halte-Boden ist anteilig (0.15 = Grenze «hoch» in PAR-F-30); ein fester Boden von 3 s machte jeden Halt unter 10 s dauerhaft zu «niedriger Konfidenz», und Sprossen mit kurzen Halten wurden nie Arbeitssprosse |
| PAR-S-39 | `derived_rung_prior` | leichtere Sprosse oder Band: μ = μ der schwereren bzw. unassistierten, σ = max(3 s, 0.35 μ); schwerere Sprosse: kein Wert bis zu Prüfversuchen | nur die sichere Richtung (leichter ≥ schwerer), keine Umrechnung über Intensitätsmodelle (`08` §4); σ aus PAR-F-26 |
| PAR-S-40 | `post_pain_deload_cap` | bis zur ersten grünen Woche Deckel 1.0 × Referenz vor der Verletzung der Schmerzregel | «reduzieren und halten» (PAR-D-18, `05` §5.4); verhindert den Sprung auf R × 1.1 direkt nach dem Schmerz-Deload |
| PAR-S-41 | `onboarding_break_mapping` | Tabelle in §6.11 | Klassen aus `onboarding.md` §3.5 auf die Bänder von PAR-B-59–62 und PAR-D-29 gelegt |
| PAR-S-42 | `pain_entry_and_trend` | neue Beschwerde: ein Wert > 2 oder Werte > 0 an 2 Tagen in 7 Tagen; Trend steigend: Wochenmittel «danach» ≥ 1 Punkt über der Vorwoche | Grenze 2 aus PAR-D-14; Zählweise und 1-Punkt-Schwelle sind Heuristik, damit PAR-D-17 berechenbar wird |
| PAR-S-43 | `entry_ramp_current_trainers` | 0.5 → 0.75 → 1.0 wöchentlich; Einheitsdeckel in diesen Wochen 1.5 × Vorwochenmaximum | ENT-S-1 (b); Schritte aus PAR-D-25; 1.5 = grösster Schritt (0.5 → 0.75), damit der Einheitsdeckel die Rampe nicht blockiert |
| PAR-S-44 | `plausibility_widening_factor` | σ × 1.25 | `onboarding.md` §5.2 (Verbreiterung bei widersprüchlichen Angaben, R-2, und nach Pausen ab 7 Wochen); Faktor ist dort als Heuristik festgelegt |
| PAR-S-45 | `unknown_answer_sigma_frac` | σ ≥ 0.5 × μ | `onboarding.md` §5.2, Zeile «weiss nicht»: Populations-Prior mit breiter Unsicherheit |
| PAR-S-46 | `prerequisite_plausibility_fraction` | 0.5 | R-2 (`onboarding.md` §5.5): eine Vorstufe gilt als plausibel, wenn ihre Schätzung mindestens die Hälfte ihrer Schwelle erreicht; die Hälfte entspricht dem Startanteil neuer Belastung (PAR-D-12) und lässt Messrauschen der Selbstauskunft (PAR-F-20: 30 %) Platz |
| PAR-S-47 | `soreness_definition` | «Soreness» in PAR-D-26 und PAR-D-28: ein Wert während, nach oder am Folgetag über dem Wert vor der Einheit (Basiswert nach PAR-D-16; ohne Basiswert 0) sowie jede Angabe «hielt länger als 1 h an» oder «hält länger als 15 min an»; eine Einheit zählt für die Rampe nur ohne Soreness und mit allen Werten ≤ PAR-D-14 | **Review.** PAR-D-28 nennt «Schmerz am Folgetag» ohne Schwelle; wörtlich genommen käme, wer mit stabilem Grundschmerz trainiert, nie aus der Rampe. Der Vergleich mit dem Wert vor der Einheit folgt PAR-D-16 und ist strenger als PAR-D-14 allein (`08` §4: strengerer Wert) |
| PAR-S-48 | `session_cap_whole_set` | LOAD-03 lässt je Einheit und Struktur mindestens M plus den kleinsten geplanten Satz dieser Struktur in der Einheit zu (bei M = 0 diesen Satz) | **Review.** PAR-D-31 (10 %) gilt für Volumen; mit ganzen Sätzen wäre bei 1–2 Sätzen je Einheit jede Steigerung > 10 %, und das Volumen bliebe beim halbierten Einstieg stehen. PAR-B-55 erlaubt +1 Satz je Übung; das Wochenwachstum begrenzt weiter LOAD-02 |

## Anhang C: Index der verwendeten Forschungsparameter

Automatisch aus den Parametertabellen der Streams erzeugt. «§» nennt die
Abschnitte dieser Spezifikation, in denen der Parameter genannt ist (auch
über Spannen wie «PAR-B-64–67»); «A» steht für den Regelkatalog. Wert und
Evidenz stehen gekürzt; massgeblich ist die Stream-Datei.

| Parameter | Schlüssel | Wert (gekürzt) | Evidenz | § |
|---|---|---|---|---|
| PAR-A-01 | `dynamic_work_sets` | 3 Sätze | D | 5, 6, A |
| PAR-A-02 | `dynamic_work_rep_range` | 5–8 Wdh. | D | 5, 6, A |
| PAR-A-03 | `dynamic_advance_threshold_reps` | 8 (in allen 3 Sätzen, gute Form) Wdh. | D | 3, 5, 6, A |
| PAR-A-04 | `dynamic_restart_reps_after_advance` | 5 Wdh. | D | 5, 6, A |
| PAR-A-05 | `dynamic_rep_increment_per_session` | +1 Wdh. pro Satz und Einheit | D | 5, 6, A |
| PAR-A-07 | `basic_iso_advance_hold_s` | 30 (in allen 3 Sätzen) s | D | 3 |
| PAR-A-10 | `lever_intermediate_unlock_hold_s` | 10 s | Heuristik | 3 |
| PAR-A-11 | `terminal_hold_unlock_s` | 3 s | C | 3 |
| PAR-A-16 | `form_quality_angle_map` | 5: ≤ 5° · 4: > 5–15° · 3: > 15–30° · 2: > 30–45° · 1: > 45° ° | Heuristik | 3 |
| PAR-A-17 | `unlock_min_form_quality_statics` | 4 1–5 | Heuristik | 3 |
| PAR-A-18 | `unlock_occurrences_default` | 2 (dynamisch ≈ 3×8: 3) Vorkommen | Heuristik | 3 |
| PAR-A-19 | `unlock_within_days_statics` | 28 Tage | Heuristik | 3 |
| PAR-A-20 | `unlock_within_days_dynamic` | 7 Tage | Heuristik | 3 |
| PAR-A-21 | `counts_for_unlock` | nur unassistiert, voller Weg, nicht exzentrisch, nicht gescheitert; K… | Heuristik | 3, 4 |
| PAR-A-22 | `difficulty_tier_from_og_level` | ⌈OG-Level × 10 / 16⌉, höchstens 10; Wurzeln = 1 Tier 1–10 | Heuristik | 2, 3 |
| PAR-A-23 | `og_level_bands` | Beginner 1–5 · Intermediate 6–9 · Advanced 10–13 · Elite 14–16 OG-Lev… | C | 2, 3, 5, 7, A |
| PAR-A-24 | `og_fig_quartiles` | Basic 1–4 · A 5–8 · B 9–12 · C 13–16 OG-Level | C | 2, 3 |
| PAR-A-25 | `og_level_planche_floor` | Frog 3 · SA Frog 4 · Tuck 5 · Adv Tuck 6 · Straddle 8 · Half-Lay/One-… | C | 2, 3 |
| PAR-A-26 | `og_level_planche_rings` | Frog 4 · SA Frog 5 · Tuck 6 · Adv Tuck 8 · Straddle 10 · Half-Lay 12… | C | 2, 3 |
| PAR-A-27 | `rings_level_offset` | Planche +1…+3; Liegestütz +3 (1 → 4); Dip +1 (3 → 4); L-Sit +2 (3 → 5… | C | 2, 3 |
| PAR-A-28 | `og_level_front_lever` | Tuck 4 · Adv Tuck 5 · Straddle 6 · Half-Lay/One-Leg 7 · Full 8 OG-Lev… | C | 2, 3 |
| PAR-A-29 | `og_level_back_lever` | German Hang 1 · Skin the Cat 2 · Tuck 3 · Adv Tuck 4 · Straddle 5 · H… | C | 2, 3 |
| PAR-A-30 | `og_level_handstand_hspu` | Wand-HS 1–3 · frei 4–5 · Ring-HS 7 · One-Arm-HS 10; Pike-HeSPU 1 · Bo… | C | 2, 3 |
| PAR-A-31 | `og_level_l_v_manna` | Tuck-L 1 · 1-Bein 2 · L 3 · Straddle-L 4 · RTO-L 5 · V45 6 · V75 7 ·… | C | 2, 3 |
| PAR-A-32 | `og_level_pull_oac` | Sprung 1 · exz. 2 · Klimmzug 3 · L 4 · Pullover 5; Ringe: L 4 · breit… | C | 2, 3 |
| PAR-A-33 | `og_weighted_pull_up_total_bw` | L3 1,00 · L4 1,18 · L5 1,35 · L6 1,50 · L7 1,65 · L8 1,78 · L9 1,90 ·… | C | 2, 3 |
| PAR-A-34 | `og_weighted_dip_total_bw` | L3 1,00 · L4 1,20 · L5 1,38 · L6 1,55 · L7 1,70 · L8 1,85 · L9 2,00 ·… | C | 2, 3 |
| PAR-A-35 | `pull_up_1rm_total_bw_reference` | Männer 1,16 ± 0,15 · Frauen 0,73 ± 0,09 × KG | B | 2, 3 |
| PAR-A-36 | `og_level_mu_flag_pistol` | MU: exz. 3 · Kipping 4 · MU 5 · ohne FG 6 · strikt Stange 7 · L-Sit-M… | C | 2, 3 |
| PAR-A-37 | `og_level_push_dip_row` | Liegestütz: Standard 1 · Diamond 2 · Ringe 4 · RTO 5 · RTO-PPPU 40° 7… | C | 2, 3 |
| PAR-A-38 | `maltese_og_level` | 17 (jenseits des Charts) OG-Level | C | 2, 3 |
| PAR-A-43 | `strength_explains_skill_r2` | Schwalbe 0,76–0,85 · Stützwaage 0,42–0,59 · Kreuz-HS 0,38–0,48 (2021,… | B | 3 |
| PAR-A-45 | `est_weeks_per_og_level_step` | Ziel-Level ≤ 4: 2–8 · 5–8: 4–13 · 9–12: 8–26 · ≥ 13: 13–52 Wochen pro… | Heuristik | 3, A |
| PAR-A-47 | `coach_time_planche_cumulative_months` | Lean 0–2 · Tuck 2–6 · Adv Tuck 6–12 · Straddle 12–24 · Full 24–36 Mon… | D | 3, 12, A |
| PAR-A-51 | `goal_realism_min_weeks` | Σ Untergrenzen aus PAR-A-45 vom aktuellen zum Ziel-Level; z. B. Anfän… | Heuristik | 3, A |
| PAR-A-52 | `prereq_front_lever_entry` | 10 strikte Klimmzüge · 30 s Dead Hang · 60 s Hollow · 15 gestreckte B… | D | 2, 3, A |
| PAR-A-53 | `prereq_planche_entry` | 60 s Hollow · 3 × 20 Liegestütze · 3 min Handgelenk-Routine · 30 s Wa… | D/C | 2, 3, A |
| PAR-A-54 | `prereq_back_lever_pull_ups` | 8–10 Wdh. | D | 2 |
| PAR-A-55 | `handstand_free_hold_targets_s` | 10 (für viele Ziele genug) · ~60 (vor One-Arm-HS) s | C | 6 |
| PAR-A-60 | `release_elements_auto_unlock` | false (nur Selbstbestätigung) | Heuristik | 3 |
| PAR-A-62 | `edge_weight_defaults` | prerequisite 1,0 · recommended 0,3–0,7 · alternative 0,5 · antagonist… | Heuristik | 3, 5, A |
| PAR-A-63 | `hold_time_to_relative_intensity_shoulder` | max. Haltezeit 5 s ≈ 99 % · 10 s ≈ 85 % · 20 s ≈ 71 % · 30 s ≈ 63 % ·… | A (Übertragung: Heuristi… | 5 |
| PAR-A-64 | `iso_set_hold_fraction_of_max` | 0,60–0,70 laut Text; die Tabelle der 2. Aufl. ergibt ≈ 0,65–0,75 bei… | C | 2, 5, A |
| PAR-A-65 | `iso_max_hold_upper_bound_per_stage_s` | 30 s | C/D | 6, A |
| PAR-A-66 | `eccentric_to_first_rep_signal` | 3 Sätze × 3 Cluster-Wdh. à 7–10 s Exzentrik; OAC: 3–4 Sätze à 10 s | C | 6, A |
| PAR-A-67 | `weighted_interim_min_load_kg_ref75` | Klimmzug 12,5 · 25 · 37,5 · 57,5 · 75; Dip 15 · 27,5 · 40 · 65 · 85;… | Heuristik | 2, 3 |
| PAR-A-69 | `muscle_up_entry_strength` | 5 zügige Klimmzüge + 5 Dips, voller Weg Wdh. | C | 2, 3, A |
| PAR-A-78 | `straight_arm_bend_form_cap` | Armbeugung > 15° → `form_quality` ≤ 3 ° | Heuristik | 3, 6, A |
| PAR-B-01 | `periodization_model_novice` | linear (Doppelprogression, keine Wellen) | A | 4, 5, 6, A |
| PAR-B-02 | `periodization_model_trained` | weekly_undulating (schwer/mittel/leicht auf die Einheiten derselben Ü… | A | 4, 5, 6, A |
| PAR-B-03 | `mesocycle_length_weeks` | 4–8; Standard 6 (5 Aufbau + 1 Deload) Wochen | B | 5, 6, A |
| PAR-B-05 | `static_max_stage_window_max_hold_s` | 3–20 (frische Max.-Haltezeit der gewählten Stufe) s | Heuristik | 5, 6, A |
| PAR-B-06 | `static_hold_fraction_of_max` | 0.6–0.7 Anteil der Max.-Haltezeit | C | A |
| PAR-B-07 | `target_sir_fraction` | 0.3–0.4 der Max.-Haltezeit, mindestens 2 s Anteil / s | Heuristik | 5, A |
| PAR-B-08 | `static_max_sets` | 2–5; Standard 3 Sätze | C | 5, 7, A |
| PAR-B-09 | `static_total_hold_per_exercise_s` | 30–90; Standard 40–60 s | B | 2, 5, A |
| PAR-B-10 | `static_volume_hold_per_set_s` | 5–20 s | B | 4, 5, A |
| PAR-B-11 | `static_volume_sets` | 3–5 Sätze | C | 5, A |
| PAR-B-12 | `static_volume_total_hold_s` | 40–90 je Übung; bei Hypertrophieziel Summe aller Halteübungen einer E… | B | 5, A |
| PAR-B-14 | `hold_endurance_model` | Schulter ET = 14.86·f^−1.83; Ellbogen ET = 17.98·f^−2.21; allgemein E… | A | 5 |
| PAR-B-16 | `eccentric_rep_duration_s` | 3–5 zu Beginn, Ziel 7–10 s | C | 5, 6, A |
| PAR-B-17 | `dyn_strength_reps` | 1–6; Standard 3–6 Wdh. | A | 5, A |
| PAR-B-18 | `dyn_strength_sets_per_exercise` | 2–5; Standard 3 Sätze | B | 5, A |
| PAR-B-19 | `dyn_hypertrophy_reps` | 6–15 Wdh. | A | 5, A |
| PAR-B-21 | `weekly_sets_per_muscle_hypertrophy` | 10–20; Start 10 Sätze/Woche | A | 5, A |
| PAR-B-23 | `rep_to_load_threshold_reps` | 12–15 saubere Wdh. in allen Arbeitssätzen Wdh. | Heuristik | 5, 6, A |
| PAR-B-24 | `target_rir_strength` | 1–3; Standard 2 RIR | A | 5, A |
| PAR-B-27 | `failure_allowed_static_and_skill` | false bool | Heuristik | 5 |
| PAR-B-28 | `rir_estimate_uncertainty_reps` | ±1 (systematisch ~1 Wdh. zu niedrig geschätzt) Wdh. | A | 6, A |
| PAR-B-29 | `rir_trust_after_weeks` | 4 (Anfänger protokollieren RIR vorher nur); danach alle 4–6 Wochen Ka… | Heuristik | 5, 6, A |
| PAR-B-30 | `progression_trigger_sessions` | 2 aufeinanderfolgende Einheiten mit allen Arbeitssätzen an der Obergr… | B | 6, A |
| PAR-B-31 | `autoreg_adjust_threshold` | Abweichung ≥ 2 RIR bzw. SIR vom Ziel in 2 aufeinanderfolgenden Einhei… | Heuristik | 6, A |
| PAR-B-32 | `load_increment_pct` | 2–10; Standard 2.5 % der bewegten Masse (Körpergewicht + Zusatzlast),… | B | 1, 2, 5, 6, A |
| PAR-B-33 | `hold_progression_step_s` | +1–2 s je Satz und Woche innerhalb der Stufe s | Heuristik | 5, 6, A |
| PAR-B-34 | `sessions_per_skill_per_week` | 2–3 (Standard 2 Anfänger, 3 Trainierte); min. 1 (Erhalt); max. 4 Einh… | A | 5, 7, A |
| PAR-B-35 | `balance_skill_sessions_per_week` | 3–7, je 5–10 min, ermüdungsarm Einheiten/Woche | Heuristik | 5, A |
| PAR-B-36 | `sessions_per_week_by_level` | Anfänger 2–3; Fortgeschrittene 3–4; Erfahrene 4–5 Einheiten/Woche | B | 4, A |
| PAR-B-37 | `split_threshold_sessions` | Ganzkörper bei ≤ 3; Split (Push/Pull, Ober/Unter oder Straight/Bent)… | Heuristik | 5, A |
| PAR-B-38 | `min_hours_between_hard_same_pattern` | 48 nach harter Einheit (RIR ≤ 1 oder Maximalversuche an der Zielstufe… | Heuristik | 7, A |
| PAR-B-42 | `rest_dyn_strength_s` | Trainierte 180–300; Anfänger 120–180 s | A | 5, A |
| PAR-B-43 | `rest_hypertrophy_s` | 90–180; Standard 120 s | A | 5, A |
| PAR-B-44 | `rest_accessory_s` | 60–120 s | B | 5, A |
| PAR-B-45 | `rest_paired_sets_s` | 120 (Spanne 90–180) zwischen abwechselnden Antagonisten-Sätzen s | B | 5, 7, A |
| PAR-B-46 | `exercise_order` | Balance-/Technikskill → Maximalreiz Zielskill → Volumen → Zubringer/P… | A | 5, 7, A |
| PAR-B-47 | `straight_arm_hard_sets_per_session_max` | 8–12 (Anfänger, Intermediate); bis 18 (Fortgeschrittene) Sätze | Heuristik | 7, A |
| PAR-B-48 | `same_direction_straight_arm_skills_per_session_max` | 2 mit gemeinsamem Satzbudget Skills | Heuristik | 2, 5, 7, A |
| PAR-B-49 | `deload_triggers` | (a) Hauptübung stagniert/fällt in ≥ 2 aufeinanderfolgenden Einheiten;… | Heuristik | 6, A |
| PAR-B-50 | `deload_frequency_weeks` | 4–8; Standard: 6. Woche nach 5 Aufbauwochen Wochen | B | 5, 6, A |
| PAR-B-51 | `deload_duration_days` | 5–7 Tage | B | 5, A |
| PAR-B-52 | `deload_volume_factor` | 0.6 (Spanne 0.5–0.7) Anteil der Sätze | B | 5, A |
| PAR-B-53 | `deload_rir_increase` | +2 (Spanne +1 bis +3; bei Halten SIR +2 s) RIR | B | 5, A |
| PAR-B-54 | `deload_intensity_rule` | Stufe halten oder 1 Stufe leichter; keine Komplettpause als Standard | A | 5, A |
| PAR-B-55 | `weekly_volume_increase_max` | +10–20 % Sätze pro Woche bzw. max. +1 Satz je Übung %/Woche | Heuristik | 7, A |
| PAR-B-57 | `intensity_step_per_week_max` | 1 Stufenwechsel (Stufe, Band oder Last) je Übung und Woche Stufen/Woc… | Heuristik | 6, A |
| PAR-B-58 | `acwr_enabled` | false bool | B | 7, A |
| PAR-B-59 | `retrain_break_le_2wk` | 90–100 % Volumen, gleiche Stufe, kein Retest | Heuristik | 4, 6, A |
| PAR-B-60 | `retrain_break_1mo` | 3–6 Wochen Pause: Retest; Woche 1 70–80 % Volumen, Stufe nach Test; +… | Heuristik | 6, A |
| PAR-B-61 | `retrain_break_3mo` | 7–16 Wochen: Retest; Woche 1 ~60 % Volumen, 1 Stufe leichter oder nac… | Heuristik | 6, A |
| PAR-B-62 | `retrain_break_6mo` | ≥ 17 Wochen: Volltest; Woche 1 ~50 % Volumen, 1–2 Stufen leichter ode… | Heuristik | 6, A |
| PAR-B-63 | `maintenance_min_dose` | 1 Einheit/Woche, 1–2 Sätze je Übung, gleiche Stufe/Intensität (Ältere… | B | 5, A |
| PAR-B-64 | `session_template_30min` | Aufwärmen 5; Primärblock 2–3 Sätze; 1–2 Antagonisten-Paare × 2–3 Rund… | Heuristik | 2, 5, A |
| PAR-B-65 | `session_template_45min` | Aufwärmen 7; Skill 5; Primär 3–4; 2 Paare; optional 1 Ergänzung; 12–1… | Heuristik | 2, 5, A |
| PAR-B-66 | `session_template_60min` | Aufwärmen 10; Skill 8–10; Primär 3–5; Sekundär 3–4; 1 Paar; 1–2 Ergän… | Heuristik | 2, 5, A |
| PAR-B-67 | `session_template_90min` | 1 Maximalübung (2–3 Sätze) + 2 Volumenübungen (je 5) + 1–2 Zubringer… | C | 2, 5, A |
| PAR-B-68 | `session_cut_priority` | Ergänzung streichen → Volumensätze auf 2–3 → Antagonisten-Paare → Pau… | Heuristik | 2, 5, A |
| PAR-B-69 | `warmup_duration_min` | 5 (30 min), 7 (45 min), 10 (60 min), 12–15 (90 min) min | Heuristik | 2, 5, A |
| PAR-B-70 | `warmup_ramp_sets` | 2–3 Rampensätze mit leichteren Stufen/Assistenz vor dem Primärblock,… | B | 5, A |
| PAR-B-71 | `warmup_static_stretch_max_s` | < 60 je Muskelgruppe s | A | 5, A |
| PAR-B-72 | `srpe_formula` | session_load = perceived_fatigue (1–10) × Dauer (min); Abfrage ~30 mi… | B | 7, A |
| PAR-B-73 | `load_monitoring_window_weeks` | 6 (gleitend); Monotonie = Mittel/SD der Tageslast; Strain = Wochenlas… | B | 4, 7, 11, A |
| PAR-B-74 | `rest_skill_practice_s` | 30–90 nach Bedarf s | Heuristik | 5, A |
| PAR-B-75 | `max_sets_per_muscle_per_session` | Hypertrophie: ~11 «fraktionale» Sätze je Muskel und Einheit (indirekt… | B | 5, A |
| PAR-B-76 | `static_conditioning_dose` | Haltezeit 10–30 s je Satz, 2–3 Sätze, 30–90 s gesamt je Übung s / Sät… | C | 5, A |
| PAR-B-77 | `eccentric_dose` | 2–3 Sätze à 2–3 Cluster-Wdh. (3–10 s je Wdh., PAR-B-16); Pause 180 s… | C | 5, 6, A |
| PAR-B-78 | `dyn_endurance_accessory_dose` | ≥ 12 Wdh. (12–20), 2–3 Sätze, RIR 1–3; Pause PAR-B-44 Wdh. / Sätze | B | 5, A |
| PAR-B-79 | `assisted_set_counting` | Hypertrophie-Wochensätze: 1.0 bei RIR/SIR ≤ 3, sonst 0.5; Maximalkraf… | Heuristik | 4, 5, 7, A |
| PAR-B-80 | `gtg_protocol` | optional; nur Grundübungen mit gebeugtem Arm und Balance-Skills; nie… | Heuristik | 2, 5, A |
| PAR-B-81 | `skill_pairing_rule` | Straight-Arm-Skills gegensätzlicher Zugrichtung abwechselnd als Paar… | Heuristik | 2, 5, 7, A |
| PAR-C-03 | `stage_torque_ratio_tuck` | 0,60 (0,56–0,69); Frau 0,64 Anteil Full | B (Modell) | 2, 3, 4, A |
| PAR-C-04 | `stage_torque_ratio_advanced_tuck` | 0,75 (0,71–0,81); Frau 0,78 Anteil Full | B (Modell) | 2, 3, 4, A |
| PAR-C-05 | `stage_torque_ratio_one_leg` | 0,88 (0,855–0,906); Frau 0,89 Anteil Full | B (Modell) | 2, 3, 4, A |
| PAR-C-06 | `stage_torque_ratio_half_lay` | 0,93 (0,905–0,954) Anteil Full | B (Modell) | 2, 3, 4, A |
| PAR-C-07 | `stage_torque_ratio_straddle` | 0,85 (120°) / 0,91 (90°) / 0,96 (60°) Anteil Full | B (Modell) | 2, 3, 4, A |
| PAR-C-08 | `stage_torque_ratio_full` | 1,00 Anteil Full | Definition | 2, 3, 4 |
| PAR-C-09 | `hand_offset_behind_shoulder` | Tuck 0,156 H / Adv 0,195 H / Straddle 90° 0,237 H / Full 0,259 H m pr… | B (Modell) | 2, 4 |
| PAR-C-10 | `lean_torque_ratio_per_arm_degree` | ≈ 0,02 (5°: 0,08; 15°: 0,26; 25°: 0,47; 31°: 0,59; 36°: 0,72) Anteil… | B (Modell) | 2, 4 |
| PAR-C-13 | `band_torque_relief` | (F_band / KG) × (x_a / H) / 0,246 Anteil Full-Moment | B (Modell) | 2, 5, A |
| PAR-C-16 | `shoulder_flexor_extensor_capacity_ratio` | 0,72 Verhältnis | B | 3 |
| PAR-C-17 | `coaching_level_lever_stages` | FL: Tuck 4, Adv 5, Straddle 6, Half-Lay/One-Leg 7, Full 8; Planche: 5… | C | 2 |
| PAR-C-18 | `bw_fraction_push_up_standard` | 0,64 (Spitze) / 0,664 (Anfang) / 0,69 oben–0,75 unten (statisch) Ante… | B | 2, 3, 4 |
| PAR-C-19 | `bw_fraction_push_up_knee` | 0,49–0,54 oben / 0,62 unten Anteil KG | B | 2, 3, 4 |
| PAR-C-20 | `bw_fraction_push_up_hands_elevated` | 30,5 cm: 0,55; 61 cm: 0,41 Anteil KG | B | 2, 3, 4 |
| PAR-C-21 | `bw_fraction_push_up_feet_elevated` | 30,5 cm: 0,70; 61 cm: 0,74 Anteil KG | B | 2, 3, 4 |
| PAR-C-22 | `bw_fraction_no_foot_contact` | 1,00 Anteil KG | Mechanik (zwingend) | 2, 3, 4 |
| PAR-C-27 | `neutral_grip_wrist_rating_offset` | −1 Stufe (0–3) | Heuristik | 2, 4, 5, 8, A |
| PAR-C-32 | `lever_relative_demand_height_exponent` | 1 Exponent (H) | B (Modell) | 3, A |
| PAR-C-38 | `bent_arm_gate_for_straight_arm` | Bankdrücken notwendig, nicht hinreichend (r = 0,41) Regel | B | 3 |
| PAR-C-40 | `pull_up_grip_overhead_modifier` | +1 auf shoulder_overhead für weiten Griff und Untergriff (max. 3) Stu… | Heuristik | 2, 4, A |
| PAR-C-44 | `load_profile_stage_scaling` | Belastung = Profilwert × stage_torque_ratio (Statik) bzw. × bw_fracti… | Heuristik | 2, 4, A |
| PAR-C-45 | `joint_load_rating_scale` | 0 keine, 1 gering, 2 moderat/endgradig, 3 hoch/limitierend ordinal | Heuristik | 2, 4, 7, A |
| PAR-C-46 | `muscle_role_weights` | primär 1,0; sekundär 0,5; Stabilisator 0,25 Gewicht | Heuristik | 5, A |
| PAR-C-47 | `ring_stabilizer_rating_offset` | +1 auf biceps_distal und biceps_long_head/anterior_shoulder bei Stütz… | Heuristik | 2, 4, A |
| PAR-C-49 | `straight_arm_category_rule` | Ellbogen gestreckt UND Last über den Arm UND (Stufe ≥ Lean mit ≥ 15°… | Heuristik | 2 |
| PAR-C-54 | `female_lever_relative_demand_factor` | Full 1,24; Advanced Tuck 1,27; Tuck 1,32 (Spanne 1,19–1,36) Faktor ge… | Heuristik (Modell) | 3, A |
| PAR-C-61 | `band_nominal_force_uncertainty` | ± 0,20 Anteil | Heuristik | 2, 5, A |
| PAR-C-62 | `band_assist_rom_rule` | Assistenz maximal bei maximaler Dehnung (Klimmzug/Dip mit Band am Fus… | B / Heuristik | 2 |
| PAR-D-01 | `injury_region_priority_order` | shoulder > wrist > elbow = lumbar Rangfolge | B | 2, 5, A |
| PAR-D-02 | `prior_injury_increase_cap_factor` | 0.5 Faktor auf die Steigerungsdeckel der Region | Heuristik | 2, 4, 5, 7, 8, A |
| PAR-D-03 | `max_sessions_per_day_same_structure` | 1 Einheiten/Tag | Heuristik | 7, A |
| PAR-D-04 | `elevated_risk_training_age_months` | 6–48 Monate Calisthenics-Erfahrung | B | 4, 7, 14, A |
| PAR-D-05 | `warmup_allows_max_skill_attempts` | false bool | Heuristik | 5, A |
| PAR-D-06 | `tendon_adaptation_horizon_weeks` | 12 (Spanne 8–14) Wochen | A/B (Zeitverlauf) + Heur… | 4, 7, A |
| PAR-D-08 | `min_rest_hours_high_tendon_load_same_structure` | 48 Stunden | B (Zeitverlauf) / Wert H… | 7, A |
| PAR-D-09 | `max_weekly_load_increase_straight_arm_pct` | 10 % gegenüber Mittel der letzten 3 Wochen | Heuristik | 2, 4, 7, A |
| PAR-D-10 | `max_weekly_load_increase_bent_arm_pct` | 20 % gegenüber Mittel der letzten 3 Wochen | Heuristik (Richtung B) | 2, 4, 7, A |
| PAR-D-11 | `max_weekly_load_increase_wrist_extension_pct` | 10 % gegenüber Mittel der letzten 3 Wochen | Heuristik | 2, 4, 7, A |
| PAR-D-12 | `new_exercise_family_start_fraction` | 0.5 Anteil des Zielvolumens in Woche 1 | Heuristik | 4, 5, 6, 7, A |
| PAR-D-13 | `pain_rating_points` | during, immediately_after, next_morning, weekly_trend Zeitpunkte, NRS… | A (Reha-Kontext) | 4, 8, A |
| PAR-D-14 | `pain_green_max_nrs` | 2 NRS 0–10 | B (Analogie) + Heuristik | 4, 8, A |
| PAR-D-15 | `pain_accept_max_nrs` | 5 NRS 0–10 | A (Reha-Kontext) | 4, 5, 8, A |
| PAR-D-16 | `pain_next_morning_rule` | Schmerz am nächsten Morgen nicht höher als vor der Einheit Regel | A | 4, 8, A |
| PAR-D-17 | `pain_weekly_trend_rule` | Schmerz und Steifigkeit nehmen von Woche zu Woche nicht zu Regel | A | 4, 8, A |
| PAR-D-18 | `pain_breach_action` | nächste Einheit: Struktur 1 Stufe Regression, −30 % Volumen, eine Woc… | Heuristik | 4, 6, 8, A |
| PAR-D-19 | `referral_persistent_symptom_days` | 28 Tage ohne Besserung trotz Lastanpassung | A/B + Heuristik | 4, 8, A |
| PAR-D-20 | `referral_pain_breach_count` | 3 in 14 Tagen Anzahl | Heuristik | 4, 8, A |
| PAR-D-21 | `red_flag_action` | Struktur sperren, Abklärung mit Dringlichkeit N/D/A empfehlen, Freiga… | B + Heuristik | 4, 5, 8, A |
| PAR-D-22 | `minor_wrist_pain_referral_days` | 7 Tage | Heuristik | 8, A |
| PAR-D-23 | `minor_age_threshold_years` | 18 Jahre | Heuristik | 2, 4, 5, 7, 8, A |
| PAR-D-24 | `rtt_start_volume_fraction` | 0.5 Anteil des letzten beschwerdefreien Volumens | Heuristik | 4, 8, A |
| PAR-D-25 | `rtt_volume_steps` | 0.25 → 0.5 → 0.75 → 1.0 (Start je nach PAR-D-24/33) Anteil | Heuristik | 4, 8, A |
| PAR-D-26 | `rtt_step_advance_criteria` | keine Soreness während/nach/am Folgetag in ≥ 2 Einheiten der Stufe; S… | B + Heuristik | 8, A |
| PAR-D-27 | `rtt_progression_order` | Hang/Zug → Stütz ohne Impact → Stütz mit Impact/Maximalversuch; inner… | B + Heuristik | 8, A |
| PAR-D-28 | `rtt_regress_on_breach` | Schmerz > 1 h danach oder am Folgetag: 1 Tag Pause, Stufe wiederholen… | B (Expertenprotokoll) | 8, 12, A |
| PAR-D-29 | `layoff_restart_threshold_weeks` | 4 Wochen Pause der Struktur | B (Richtung) + Heuristik… | 4, 6, 8, A |
| PAR-D-30 | `acwr_hard_rule_enabled` | false bool | A/B | 7, A |
| PAR-D-31 | `max_session_spike_vs_30d_max_pct` | 10 % über der grössten Einheit derselben Struktur in 30 Tagen | B (Analogie) | 4, 6, 7, A |
| PAR-D-32 | `referral_soft_prompt_days` | 14 Tage ohne Besserung trotz Lastanpassung | Heuristik | 4, 8, A |
| PAR-D-33 | `rtt_start_volume_fraction_after_referral` | 0.25 Anteil des früheren Volumens | B (Expertenregel) + Heur… | 4, 6, 8, A |
| PAR-D-34 | `min_rest_hours_high_tendon_load_rtt` | 72 Stunden | B (Analogie) | 7, 8, A |
| PAR-D-37 | `prehab_sessions_per_week` | 3 (Spanne 2–4) Einheiten/Woche | A (Analogie) | 2, 5, A |
| PAR-D-38 | `prehab_evidence_label` | shoulder: `rct_other_sport`; elbow: `rct_other_sport`; wrist, finger:… | A/B | 2, 5, 9, A |
| PAR-D-39 | `pain_relief_claims_allowed` | false bool | A | 8, 9, A |
| PAR-D-40 | `complete_rest_default_on_pain` | false bool | A/B | 8, A |
| PAR-D-41 | `supinated_straight_arm_high_risk` | true bool (Metadatum je Übung) | B | 2, 5, 8, A |
| PAR-D-42 | `minor_back_pain_referral_days` | 7 Tage | Heuristik | 8, A |
| PAR-E-01 | `session_block_order` | `warmup` → `skill_max` → `skill_volume` → `strength_accessory` → `pre… | A/B/C | 5, A |
| PAR-E-02 | `skill_priority_first` | true: Der Skill mit der höchsten User-Priorität erhält den ersten Arb… | A/B | 5, A |
| PAR-E-03 | `equal_priority_order_rotation` | Bei gleicher Priorität wechselt die Reihenfolge je Einheit Regel | Heuristik | 5, A |
| PAR-E-04 | `max_effort_min_rest_s` | Default 300, Untergrenze 180 s | A/B/C | 5, 7, A |
| PAR-E-05 | `max_effort_min_rest_heavy_s` | 420 (Spanne 300–420) s | C | 5, A |
| PAR-E-06 | `skill_volume_rest_s` | 180–300 (Default 240) s | A/B/C | 5, A |
| PAR-E-07 | `accessory_rest_s` | 120–180 s | A/C | 5, A |
| PAR-E-08 | `max_attempts_per_skill_per_session` | Default 3, Spanne 2–5 Sätze/Versuche | C; A/B nur als Analogie | 5, A |
| PAR-E-09 | `working_sets_per_skill_session` | 12–18 Sätze | C | 7 |
| PAR-E-11 | `strength_skill_sessions_per_week` | Default 3, Spanne 2–4 Einheiten/Woche | A/B (Analogie) + Heurist… | 5, A |
| PAR-E-12 | `balance_skill_sessions_per_week` | Default 4, Spanne 3–6 Einheiten/Woche | A (Analogie) + Heuristik | 5, A |
| PAR-E-13 | `same_skill_min_spacing_h_max_effort` | 48 h | B + Heuristik | 7, A |
| PAR-E-14 | `same_skill_min_spacing_h_submax` | 24 h | B/A (Analogie) + Heurist… | 5, 7, A |
| PAR-E-15 | `quality_stop_form_drop` | Stopp, wenn `form_quality` ≤ (erster Arbeitssatz − 1) oder < 3 Formwe… | Heuristik | 5, 6, A |
| PAR-E-16 | `quality_stop_consecutive_failures` | 2 Fehlversuche in Folge | Heuristik | 5, 6, A |
| PAR-E-17 | `quality_stop_performance_drop_pct` | 20 % unter dem besten Satz der Einheit | Heuristik (Analogie zu A) | 5, A |
| PAR-E-18 | `skill_sets_to_failure` | false (`target_rir` ≥ 1) Regel | A/B/C | 5, A |
| PAR-E-19 | `fatigue_gate_replace_max_attempts` | Bei hohem Ermüdungssignal Maximalversuche durch submaximale Technik e… | Heuristik | 6, A |
| PAR-E-20 | `early_neural_phase_weeks` | 3–5 Wochen | B | 6, A |
| PAR-E-24 | `target_pattern_exposure_per_session` | ≥ 1 Block mit der Zielbewegung oder ihrer nächsten Regression je Skil… | A/B | 5, A |
| PAR-E-25 | `isometric_intent_cue` | «Spannung so schnell wie möglich maximal aufbauen, dann halten» für H… | A + Ableitung | 5 |
| PAR-E-26 | `pre_max_ramp_sets` | 2–3 (`kind = warmup`, submaximal, steigend; z. B. ~50 % und ~70 % der… | A/B | 5, A |
| PAR-E-27 | `gtg_allowed` | nur Grundübungen mit gebeugtem Arm (Klimmzug, Liegestütz, Dip); nie b… | C/D + B | 2, 5, A |
| PAR-E-28 | `skill_limiting_factor` | `strength` (Planche, Front Lever, Maltese, One-Arm Pull-up) · `balanc… | B/C + Heuristik | 2 |
| PAR-E-29 | `rationale_excludes_cns_fatigue` | true Regel | B | 9, A |
| PAR-E-31 | `progress_eval_on_first_fresh_set` | true: Fortschritt einer Stufe wird am ersten Arbeitssatz der Folgeein… | B + Heuristik (Anwendung) | 4, A |
| PAR-E-32 | `max_attempt_schedule` | `blocked` (alle Maximalversuche eines Skills hintereinander) Regel | A (Begründung) + Heurist… | 5, 7, A |
| PAR-E-33 | `interleave_submax_technique_allowed` | true (optional; Pause je Skill nach PAR-E-04/06 bleibt) Regel | A/B + Heuristik | A |
| PAR-E-34 | `variation_between_sessions` | Varianten wechseln zwischen Einheiten; Zielstufe bleibt in jeder Skil… | A/B/C | 5, 6, A |
| PAR-E-35 | `balance_block_minutes` | 11–15 min | A (Analogie) | 5, A |
| PAR-E-36 | `balance_set_duration_s` | 21–40 s pro Satz (inkl. Versuche) | A (Analogie) + Heuristik | 5, A |
| PAR-E-37 | `balance_min_sessions_before_review` | 16 Einheiten | A (Analogie) + Heuristik | 6, A |
| PAR-E-38 | `technique_rest_to_work_ratio_min` | 1.0 (Pause ≥ Versuchsdauer) Verhältnis | Heuristik | 5, A |
| PAR-E-41 | `sleep_loss_threshold_h` | ≤ 6 h in 24 h | A (Definition) | 6, A |
| PAR-E-42 | `sleep_loss_effect_reference_pct` | Skill −20.9; Kraft −2.85; gesamt −7.56 % | A | 6, A |
| PAR-E-43 | `sleep_loss_action` | Skill-Maximalversuche → submaximale Technik; Oberkörperkraft unveränd… | A + Heuristik | 6, A |
| PAR-E-44 | `rationale_excludes_sleep_enhancement` | true Regel | A/B | 9, A |
| PAR-E-45 | `static_stretch_max_s_before_max` | < 60 pro Muskelgruppe, danach dynamische Aktivität s | A | 5, A |
| PAR-E-46 | `pap_conditioning_for_holds` | false Regel | A/B | 5, A |
| PAR-E-47 | `general_warmup_min` | 5–10 min | Heuristik | 5, A |
| PAR-E-48 | `gtg_reps_pct_of_max` | ≤ 50 (Spanne 40–50) % der Maximalwiederholungen | D | 5, A |
| PAR-E-49 | `gtg_min_rir` | 2 Wdh. Reserve | C | 5, A |
| PAR-F-01 | `rep_test_sem_reps` | 2.0 (Spanne 1.1–2.0) Wdh. | B | 4, 6, A |
| PAR-F-02 | `rep_test_sem_reps_low_rep` | 1.0 (gemessen 0.7) Wdh. | B | 4, A |
| PAR-F-08 | `push_up_bw_fraction` | 0.64–0.75 Anteil Körpergewicht | B | 2 |
| PAR-F-09 | `knee_push_up_bw_fraction` | 0.49–0.62 Anteil KG | B | 2 |
| PAR-F-10 | `incline_push_up_bw_fraction` | 0.55 (Hände 30.5 cm); 0.41 (61 cm) Anteil KG | B | 2 |
| PAR-F-11 | `decline_push_up_bw_fraction` | 0.70 (Füsse 30.5 cm); 0.74 (61 cm) Anteil KG | B | 2 |
| PAR-F-16 | `hold_test_sd_frac` | Ausdauerhalte 0.15; Gleichgewichtshalte 0.25; Rumpfbeuger-Halte (Holl… | B | 4, A |
| PAR-F-20 | `self_report_prior_sd` | max(2 Wdh., 0.30 × μ) Wdh. | Heuristik | 4, A |
| PAR-F-21 | `self_report_recent_count_sd_factor` | 0.75 Faktor | Heuristik | 4, A |
| PAR-F-22 | `test_obs_sd_reps` | = PAR-F-01 (2.0) Wdh. | B (Wert) / Heuristik (Se… | 4, A |
| PAR-F-23 | `log_rir_obs_sd_reps` | 2.0 Wdh. | Heuristik | 4, A |
| PAR-F-24 | `log_rir_max_usable` | 3 RIR | Heuristik (gestützt auf… | 4, A |
| PAR-F-25 | `log_rir_bias_correction_reps` | 0 Wdh. | Heuristik | 4, 6, A |
| PAR-F-26 | `converted_estimate_sd_frac` | ≥ 0.35 Anteil von μ | Heuristik | 4, A |
| PAR-F-28 | `capacity_process_sd_per_week` | 0.5 Wdh./Woche | Heuristik | 4, A |
| PAR-F-30 | `confidence_class_cv_thresholds` | hoch < 0.15; mittel 0.15–0.30; niedrig ≥ 0.30 σ/μ | Heuristik | 4 |
| PAR-F-31 | `dose_from_estimate_offset_sd` | hoch 0; mittel 0.5; niedrig 1.0 σ | Heuristik | 4 |
| PAR-F-32 | `contradiction_threshold_sd` | 2 kombinierte σ | Heuristik | 4, A |
| PAR-F-33 | `contradiction_resolution` | `take_lower_with_higher_sd` enum | Heuristik | 4, A |
| PAR-F-41 | `capacity_evidence_min_form_quality` | 3 Skala 1–5 | Heuristik | 4, A |
| PAR-F-42 | `readiness_gate_mode` | `soft` enum | Heuristik | 3, 5, A |
| PAR-F-43 | `placement_scope` | `per_skill` enum | C | 4 |
| PAR-F-45 | `screening_refer_on` | Symptome oder bekannte Herz-Kreislauf-, Stoffwechsel- oder Nierenerkr… | B | 5, A |
| PAR-F-47 | `injury_history_window_months` | 12 Monate | Heuristik | 4 |
| PAR-F-48 | `training_status_scale` | Stufen 0–5 (sesshaft … Weltklasse) Skala | B | 4, A |
| PAR-F-53 | `mobility_used_for_placement` | false bool | Heuristik | 5, A |
| PAR-F-55 | `self_report_recall_bias_frac` | 0.05 (Klassenmitte × 0.95) Anteil | B | 4, A |
| PAR-F-56 | `self_report_filmed_sd` | = PAR-F-22 (wie Test) Wdh. | Heuristik | 4 |
| PAR-F-57 | `suspension_row_bw_fraction` | 0.69–0.76 (Körper laut Autoren waagrecht); ≈ 0.58 (Gurt 238 cm, Körpe… | B | 2 |
| PAR-F-58 | `suspension_push_up_bw_fraction` | 0.50 (gestreckt) – 0.75 (gebeugt) bei senkrechten Gurten [F-65]; 0.70… | B | 2 |
| PAR-F-62 | `readiness_hint_muscle_up` | «bereit zum Üben»: ≥ 8 Klimmzüge und ≥ 8 Dips (+ 20 s False Grip); «e… | C/D | 2, 3, A |
| PAR-F-63 | `readiness_hint_front_lever_start` | ≥ 10 Klimmzüge, ≥ 30 s Totehang, ≥ 60 s Hollow Hold Wdh. / s | D | 2, 3, A |
| PAR-F-64 | `readiness_hint_planche_start` | 3 × 20 Liegestütze, ≥ 60 s Hollow Hold, schmerzfreie Handgelenkvorber… | D | 2, 3, A |
| PAR-F-68 | `hold_capacity_error_model` | relativ (Anteil von μ, entspricht Log-Skala) Modell | Heuristik | 4, A |
| PAR-F-70 | `training_age_role` | verbreitert Startwerte, bestimmt nicht die Stufe Regel | Heuristik | 4 |
