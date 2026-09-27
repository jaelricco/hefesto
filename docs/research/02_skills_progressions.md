# 02 — Skills & Progressionen

> Stream A der Phase-2-Recherche. Deckt die Progressionsleitern aller Skills ab,
> die der Planer als Knoten des Skill-Graphen braucht: Einstiegswurzeln unter
> dem PDF-«Beginner», Statics, Dynamics, Kraftskills. Für jede Stufe ein
> Slug-Vorschlag, Formkriterien, ein Unlock-Kriterium in der bestehenden DSL und
> eine Dauer bis zur nächsten Stufe. Dazu eine gemeinsame Schwierigkeitsskala,
> eine Kanten-Tabelle (Voraussetzung / Empfehlung / Alternative / Antagonist)
> und die Klärung der PDF-Kürzel.
>
> **Wichtige Einschränkung dieser Fassung (bitte vor der Nutzung lesen):** In
> dieser Sitzung war der Abruf von Webseiten (WebFetch) für alle
> Fachverlags-, Datenbank- und Coaching-Hosts gesperrt (u. a. PubMed, PMC,
> MDPI, Springer, Wiley, ResearchGate, Reddit, Wikipedia, stevenlow.org,
> gymnastics.sport), und das Websuch-Kontingent der Sitzung war nach rund
> 45 Suchen dieses Streams erschöpft. Alle Quellen unten wurden deshalb
> **über die Landing-Page-Auszüge der Suchmaschine** (Titel, Autoren, Jahr,
> Abstract-Auszüge) bzw. über einen GitHub-Spiegel (r/bodyweightfitness)
> geprüft, **nicht im Volltext**. Zahlen, die nur aus einem solchen Auszug
> stammen und im Original noch zu prüfen sind, tragen den Zusatz *(S)*.
> Viele Stufen-Schwellen und fast alle Dauern sind deshalb
> **Praxisheuristiken** und als solche markiert. Ein Verifikationsdurchgang
> mit Volltextzugriff ist nötig, bevor Werte als `status: active` in
> `content/` landen (siehe «Offene Fragen»).

**Legende für diese Datei**

| Kürzel | Bedeutung |
|---|---|
| `[A-xx]` | Quelle aus Stream A (Tabelle «Quellen») |
| `[P-0x S. n]` | PDF-Quelle (Daï-Long Huynh), siehe `01_pdf_extract.md` |
| *(S)* | Wert nur aus einem Such-Auszug gelesen; Volltext/Original nicht eingesehen |
| **(H)** | **Praxisheuristik** — keine Quelle gefunden; Begründung im Code dahinter |
| (H-DUR) | Dauer-Heuristik: keine Daten gefunden; Spanne nach OG-Level-Abstand (PAR-A-40) und RR-Mindestdauer (PAR-A-39); muss aus App-Logs kalibriert werden |
| (H-FORM) | Formkriterium = Definition der Position selbst (was die Stufe zur Stufe macht); Toleranz an FIG-Winkelabzügen angelehnt [A-04] *(S)* |
| (H-UNL) | Unlock-Schwelle nach der generischen Vorlage in §2.3 (10 s Zwischenstufe [A-12], 3 s Wettkampfhalt [A-09], 3×8 dynamisch [A-03], ≥ 2 Vorkommen nach ADR 0003); stufenspezifische Anpassung ohne eigene Quelle |
| (H-PRE) | Voraussetzung/Kante aus Bewegungsverwandtschaft abgeleitet; keine Studie, Coaching-Quelle in dieser Sitzung nicht einsehbar |
| (H-EQ) | Equipment-Effekt mechanisch begründet (Hebel, Griff, Instabilität); nicht gemessen |
| (H-FAULT) | Häufiger Fehler = Umkehrung des Formkriteriums; keine Quelle |

DSL-Kurzschreibweise in den Stufentabellen:
`<exercise> <measure> ≥ <value> · <assistance> · form≥<n> · occ <n> · <within_days> d`.
Beispiel `planche-tuck hold ≥ 10 s · none · form≥4 · occ 2 · 28 d` entspricht

```yaml
unlock_criteria:
  all:
    - { exercise: planche-tuck, measure: hold_seconds, op: ">=", value: 10,
        assistance: none, min_form_quality: 4, occurrences: 2, within_days: 28 }
```

## Kurzfassung

- **Progression über schwerere Varianten ist als Überlastungsmethode belegt
  (Evidenz A):** 4 Wochen progressives Liegestütz-Training (3×/Woche) brachten
  gleiche Zuwächse im Bankdrück-1RM und in der Muskeldicke wie Bankdrücken
  [A-14]. Die Leiter-Logik des Skill-Graphen hat damit eine belastbare Basis.
- **Die einzige quantitativ belegte Grundleiter ist der Liegestütz:** Wand-/Schräg-
  (41–55 % KG), Knie- (49 %), Standard- (64 %) und erhöhte Füsse (70–74 %)
  [A-15]; positionsabhängig 69–75 % bzw. 54–62 % [A-16]. Für Dip, Row, Pull-up
  gibt es in dieser Fassung nur die Reihenfolge aus der Community-Routine [A-03].
- **Generische Wechselregeln (Evidenz C/D):** dynamisch 3×5–8, +1 Wdh. pro Satz,
  bei 3×8 sauber die nächste Variante, dort wieder mit 3×5 beginnen [A-03];
  Grund-Isometrie 10–30 s, Wechsel bei 3×30 s [A-03]; Hebel-Statics: nächste
  Stufe erst nach 10 s sauberem Halt [A-12].
- **Wettkampfstandards liefern die Messlatte für «erreicht»:** FIG verlangt 2 s
  Halt [A-04] *(S)*, WSWCF mindestens 3 s [A-09]; Tuck/Advanced Tuck gelten im
  Street-Workout-Wettkampf nicht als Halt [A-10] *(S)*. Vorschlag: Endstufen
  schalten bei ≥ 3 s (2 Vorkommen, Form ≥ 4) frei, Zwischenstufen bei ≥ 10 s.
- **Formqualität lässt sich an FIG-Winkelabzüge koppeln:** ≤ 15° Abweichung
  kleiner Abzug, 15–30° mittel, > 30° gross, > 45° nicht anerkannt [A-04]
  *(S)* → Vorschlag `form_quality` 4 = ≤ 15°, Unlock verlangt ≥ 4.
- **Eine gemeinsame Ordinalskala ist über Overcoming Gravity (OG) möglich:**
  OG-Level 1–16, aus dem FIG-Code abgeleitet [A-01, A-02]. Front Lever Tuck→Full
  = Level 4→8, Back Lever 3→7, Planche Tuck 5, Straddle 8, Full ≥ 10 [A-02]
  *(S)*. FIG-Wert A entspricht dabei OG 7–8 (Full BL, Full FL, Straddle
  Planche), C entspricht ≥ 10 (Full Planche) [A-02, A-05, A-06].
- **Kraft sagt Statics nur teilweise voraus (Evidenz B):** Vorbereitende
  Kraftübungen korrelieren mit Ringhalteelementen (r 0,65–0,92 für
  Schwalbe/Stützwaage) [A-23], erklären beim Kreuz-Handstand aber nur bis 48 %
  [A-24]; Bankdrücken–Kreuz r = 0,41 mit Schwellencharakter [A-26]; die
  Muskelaktivität der Stützwaage (Planche) weicht deutlich von ihren
  Vorübungen ab [A-25]. Folge: Kraftbaselines sind **empfohlene**, keine harten
  Voraussetzungen; die Zielfigur muss spezifisch geübt werden.
- **Maltese liegt klar über der Planche:** FIG bewertet die Maltese am Boden C,
  an den Ringen D [A-07, A-05]; Kraft-Benchmarks für Ringhalteelemente reichen
  bis 94,1 % Körpergewicht (Schwalbe, exzentrisch) [A-23]. Die PDF-Leiter
  (Lean Maltese → Elevator → Wide Planche → Maltese mit Band → Press) ist mit
  diesen Quellen verträglich [P-02 S. 4; P-03 S. 4].
- **Handstand-Balance ist primär eine Handgelenksstrategie (Evidenz A):**
  Korrekturen laufen zuerst über die Handgelenke; Augen zu und Nackenflexion
  verschlechtern die Balance [A-20]. Handgelenk-Konditionierung gehört als
  Wurzel vor den Handstand.
- **Kipping ist eine andere Bewegung:** Kipping-Klimmzüge verändern Kinematik
  und Muskelaktivität deutlich [A-19] → Kipping- und strikte Varianten brauchen
  getrennte Exercise-Slugs; nur strikte zählen für strikte Stufen.
- **Typische Dauer bis zur nächsten Stufe: keine belastbaren Daten gefunden.**
  Alle `est_weeks_from_prev`-Werte in dieser Datei sind Heuristiken (H-DUR). Die
  einzige ableitbare Untergrenze: Die RR-Regel braucht ≥ 4 Einheiten pro
  dynamischer Variante [A-03]. Hefesto sollte Dauern aus eigenen Logs lernen.
- **DSL-Lücken gefunden:** `occurrences` zählt Sätze, nicht Tage (eine Einheit
  mit 2 Sätzen erfüllt «occ 2»); `min_load_kg` ist absolut, gewichtete Standards
  in % Körpergewicht sind nicht ausdrückbar; rein exzentrische und assistierte
  Elemente zählen nie — Negativ- und Band-Stufen können daher keine
  automatisch freischaltbaren Levels sein (§2.4).
- **PDF-Kürzel:** Mangels Zugriff auf Daï-Long Huynhs eigene Kanäle bleiben
  `supi`, `fake supi`, `neck band`, `Zanetti`, `Dead Planche`, `Elevator`
  und `wide` **unklar bzw. unsicher**; §9 gibt kontextbasierte Hypothesen.

## 1. Methodik und Grenzen

| Punkt | Umsetzung |
|---|---|
| Suchstrategie | Englische Suchen nach Primärstudien (Ringe/Turnen, Push-up-Kinetik, Pull-up-EMG/Kinematik, Handstand-Biomechanik), Regelwerken (FIG, WSWCF), Coaching-Literatur (Overcoming Gravity, Sommer, GMB) und Community-Routinen (r/bodyweightfitness). |
| Prüfung | Jede Quelle wurde mindestens auf Titel, Autor(en), Jahr und Abstract/Landing-Page-Auszug geprüft. Volltexte waren nicht erreichbar (siehe Einschränkung oben). Die r/bodyweightfitness-Routine wurde im Wortlaut über den GitHub-Spiegel `redditbwf/redditbwf.github.io` gelesen [A-03]. |
| Evidenzstufen | A = SR/MA/RCT; B = Einzelstudien, Regelwerk der FIG als Expertenkonsens; C = Coaching-Bücher/-Seiten, Street-Workout-Regelwerk; D = Wikis, Foren, Blogs. |
| r/bodyweightfitness | Stufe **D** (Community-Wiki), aber seit Jahren kuratiert und breit genutzt; hier nur für Reihenfolgen und Wechselregeln der Grundübungen verwendet, nicht für Schwellen fortgeschrittener Skills. |
| Overcoming Gravity | Stufe **C**. Die Charts wurden laut Autor aus dem FIG Code of Points abgeleitet [A-01, A-02]. Die Level-Zahlen stammen aus Such-Auszügen von Chart-Kopien *(S)* und sind im Original zu verifizieren. |
| Nicht erreichbar | Muscle-up-, Human-Flag-, Pistol-, One-Arm-Pull-up- und Weighted-Standards-Quellen, Daï-Long Huynhs Kanäle, r/bodyweightfitness-Übungsseiten, der FIG-Volltext. Diese Lücken sind in «Offene Fragen» gelistet. |

## 2. Übergreifende Regeln für Stufen, Unlocks und Wechsel

### 2.1 Was eine Stufe ist

Eine Stufe (Level) ist ein **Leistungsnachweis**, eine Übung (Exercise) ist, was
geloggt wird (`CONTENT_AUTHORING.md`). Daraus folgt für den Graphen:

| Regel | Begründung |
|---|---|
| Stufen = Positionen/Varianten, die ein Athlet **unassistiert, voll, nicht nur exzentrisch** zeigen kann. | Die Unlock-Engine zählt fehlgeschlagene, partielle und rein exzentrische Elemente nie; `assistance: none` ist Default (`internal/domain/progress/evaluate.go`). |
| Negative, Band-assistierte, Kicks und Übergänge sind **Übungen mit Rolle `progression`**, keine Stufen. | Sie können mit der DSL nicht automatisch freischalten (s. o.) und sind in den PDFs reguläres Trainingsvolumen auf allen Niveaus [P-01 bis P-03; `01_pdf_extract.md` §4.4]. |
| Kombinationen (z. B. «L-Sit to Tuck Planche», «Hold to Press») sind ein `set_entry` mit mehreren Elementen und zählen als **ein** Vorkommen. | CLAUDE.md («one code path for sets»); `evaluate.go` zählt Set-Entries. |
| Bestehende Level-Slugs bleiben: `pull-up/strict-5`, `handstand/wall`, `front-lever/tuck`, `front-lever/advanced-tuck`. Neue Einstiegsstufen darunter werden als **eigene Wurzel-Skills** angelegt statt die Reihenfolge bestehender Skills umzubauen. | Level-Slugs sind permanent (`CONTENT_AUTHORING.md`); ob `order` bestehender Levels verschoben werden darf, ist offen (Offene Fragen). |

### 2.2 Wechselregeln aus der Literatur

| Regel | Wert | Quelle | Evidenz |
|---|---|---|---|
| Dynamische Grundübungen: Arbeitsbereich | 3 Sätze × 5–8 Wdh. der schwersten machbaren Variante | [A-03] | D |
| Steigerung innerhalb der Variante | jede Einheit versuchen, die Vorwerte zu schlagen (+1 Wdh. pro Satz) | [A-03] | D |
| Wechsel zur nächsten Variante | bei 3×8 mit guter Form; dort mit 3×5 neu beginnen | [A-03] | D |
| Grund-Isometrie (Support-Halt, Tuck Front Lever in der Row-Leiter) | Sätze à 10–30 s; Wechsel, wenn alle 3 Sätze 30 s erreichen | [A-03] | D |
| Hebel-Statics (Planche-Leiter) | nächste Stufe erst, wenn die aktuelle 10 s sauber gehalten wird | [A-12] | C |
| Tuck → Straddle | Übergang «fliessender» als die Tuck-Stufen; Straddle-Varianten parallel üben | [A-11] *(S)* | C |
| Wettkampfhalt FIG | 2 s | [A-04] *(S)* | B |
| Wettkampfhalt Street Workout (WSWCF) | ≥ 3 s | [A-09] | C |
| Nicht als Halt gewertet (Street-Workout-Wettkampf) | Tuck/Adv-Tuck-Planche; Tuck/Adv-Tuck/45°-Straddle-Front-Lever | [A-10] *(S)* | D |
| Winkelabzüge bei Halteelementen (FIG) | bis 15° → 0,1; 15–30° → 0,3; > 30° → 0,5; > 45° ggf. nicht anerkannt | [A-04] *(S)* | B |

*Ableitung (RR-Mindestdauer):* Von 3×5 bis 3×8 mit +1 Wdh. pro Satz und
Einheit braucht es mindestens 4 Einheiten (5→6→7→8) [A-03]; bei 3 Einheiten pro
Woche ≈ 1–1,5 Wochen, bei 2 Einheiten ≈ 1,5–2 Wochen (eigene Rechnung). Das ist
eine **Untergrenze**, keine typische Dauer.

### 2.3 Generische Unlock-Vorlage (Vorschlag)

Die Quellen geben selten stufenspezifische Schwellen. Die Vorlage übersetzt die
Regeln aus §2.2 in die DSL und unterscheidet **Unlock** (Karte, «erreicht»,
nie zurückgenommen) von **Trainingswechsel** (Planer stellt das Haupttraining
auf die nächste Stufe um).

| Stufentyp | Unlock (DSL) | Trainingswechsel zur nächsten Stufe | Quellen |
|---|---|---|---|
| Dynamische Grundübung | `reps ≥ 8 · none · occ 3 · 7 d` (≈ 3×8) | 3×8 sauber | [A-03]; occ/within (H-UNL) |
| Grund-Isometrie (Support, Hollow, L-Sit-Vorstufen, Tuck FL in Row-Leiter) | `hold ≥ 30 s · none · occ 3 · 7 d` | 3×30 s | [A-03]; occ/within (H-UNL) |
| Zwischenstufe Hebel-Static (Tuck, Adv Tuck, One-Leg) | `hold ≥ 10 s · none · form≥4 · occ 2 · 28 d` | 10 s sauber (Vorschlag: in 2 Sätzen) | [A-12]; Wettkampf wertet diese Stufen nicht [A-10] *(S)* |
| Wettkampffähige Stufe (Straddle, Half-Lay, Full) | `hold ≥ 3 s · none · form≥4 · occ 2 · 28 d` | ≥ 10 s sauber, bevor die nächste Stufe Haupttraining wird | Halt ≥ 3 s [A-09]; 10 s [A-12] |
| Press/Kraftskill mit Wdh. (HSPU, OAP, Press) | `reps ≥ 1 · none · form≥4 · occ 2 · 28 d` für die Erststufe, danach Wdh.-Stufen (3, 5) | 3×3–5 sauber | (H-UNL) — PDF dosiert Maximalpressen mit 1–3 Wdh. [P-01 S. 2–3; P-02 S. 2] |
| Gewichtete Stufe | `reps ≥ 5 · min_load_kg X · occ 2 · 28 d` | — | (H-UNL); % KG nicht ausdrückbar, siehe §2.4 |

`form_quality`-Zuordnung (Vorschlag, an FIG-Abzüge angelehnt [A-04] *(S)*):
5 = keine sichtbare Abweichung; 4 = ≤ 15°; 3 = 15–30°; 2 = 30–45°; 1 = > 45°
(im Wettkampf nicht anerkannt). Unlock für Statics verlangt ≥ 4. Für dynamische
Grundübungen wird `min_form_quality` weggelassen (H-UNL: Nutzer bewerten Form
optional; ein gesetztes `min_form_quality` schliesst unbewertete Elemente aus,
`evaluate.go`).

### 2.4 Gefundene Grenzen der DSL

| Grenze | Folge | Vorschlag (für die Spezifikation, Phase 4) |
|---|---|---|
| `occurrences` zählt verschiedene Set-Entries, nicht Tage. | «Wiederholung über Tage» ist nicht erzwingbar; zwei Sätze in einer Einheit genügen für `occ 2`. | Feld `min_distinct_days` ergänzen; bis dahin `occ` ≥ 2 mit `within_days` als Näherung. |
| `min_load_kg` ist absolut. | Standards in % Körpergewicht (z. B. +25 % KG) sind nicht global formulierbar. | Feld `min_load_pct_bw` (bezogen auf `workout_sessions.bodyweight_kg`). |
| `min_form_quality` schliesst Elemente ohne Formbewertung aus. | Wer Form nie bewertet, schaltet Statics nie frei. | UI fordert Formbewertung bei Unlock-relevanten Sätzen an. |
| Exzentrische und assistierte Elemente zählen nie (bei `assistance: none`). | Negativ-/Band-Stufen können nicht automatisch freischalten. | Als `progression`-Übungen führen, nicht als Levels (§2.1). |
| Kein Kriterium für «Ausführung aus totem Stand vs. mit Schwung». | Kipping und strikt nur über getrennte Slugs trennbar. | Getrennte Exercise-Slugs (`muscle-up-bar-kipping` vs. `muscle-up-bar-strict`) [A-19]. |

## 3. Relative Schwierigkeit: eine gemeinsame Ordinalskala

### 3.1 Quellen für Schwierigkeit

| Skala | Was sie liefert | Quelle | Evidenz |
|---|---|---|---|
| FIG Code of Points MAG 2025–2028 | Elementwerte A (leicht) … J; Halteelemente 2 s | [A-04]; Werte über [A-05, A-06, A-07, A-08] *(S)* | B (Werte via C/D) |
| Overcoming Gravity 2. Aufl., Progression Charts | Level 1–16 pro Übungsspalte, aus dem FIG-Code abgeleitet | [A-01, A-02] *(S)* | C |
| Push-up-Kinetik | Anteil Körpergewicht je Variante | [A-15, A-16] | B |
| Ring-Kraft-Benchmarks | nötige Konditionierungskraft in % KG | [A-23] | B |

### 3.2 FIG-Elementwerte (sofern gefunden)

| Element | Gerät | Wert | Quelle | Anmerkung |
|---|---|---|---|---|
| Straddle Planche (2 s) | Boden | A | [A-06] | URL/Titel der Seite nennt «A – Straddle planche» (Boden Männer) |
| Planche (2 s) | Boden | C | [A-04] *(S)* | Wert aus Such-Auszug |
| Straddle Planche | Ringe | A | [A-05] | |
| Planche (Stützwaage) | Ringe | C | [A-05] | |
| Front Lever | Ringe | A | [A-05] | |
| Back Lever | Ringe | A | [A-05] | |
| Maltese (Schwalbe) | Boden | C | [A-07] | |
| Maltese (Schwalbe) | Ringe | D | [A-05, A-07] | |
| Victorian / Inverted Swallow | Ringe | E | [A-05] | |
| Kreuz (Iron, L, V) | Ringe | C (bis 2024: B) | [A-08] | Aufwertung im Zyklus 2025–2028 |
| Handstand (2 s), Planche → Press zum Handstand, Manna (2 s) → Handstand | Boden | als Elemente der Gruppe I gelistet | [A-04] *(S)* | Werte nicht verifiziert; Manna-Wert «C» nur aus unklarer Such-Zusammenfassung → nicht verwendet |

### 3.3 Overcoming-Gravity-Level (Such-Auszüge aus Chart-Kopien *(S)*)

| Spalte | Level → Stufe | Quelle |
|---|---|---|
| Planche | 3 Frog Stand · 5 Tuck · 6 Adv Tuck · 8 Straddle · 9 Half-Lay / One-Leg · 10–16 Full und Maltese-Varianten | [A-02] *(S)* |
| Front Lever | 4 Tuck · 5 Adv Tuck · 6 Straddle · 7 Half-Lay / One-Leg · 8 Full | [A-02] *(S)* |
| Front-Lever-Rows | Tuck (Zeile 4) · Adv Tuck (5–7) · Straddle (8) · Full (10) | [A-02] *(S)* |
| Back Lever | 1 German Hang · 2 Skin the Cat · 3 Tuck · 4 Adv Tuck · 5 Straddle · 6 Half-Lay / One-Leg · 7 Full | [A-02] *(S)* |
| Handstand | 1 Wall HS · 4–5 Freestanding HS · höher: Varianten bis One-Arm | [A-02] *(S)* |
| Handstand Push-up | 1 Pike Headstand-PU · 2 Box Headstand-PU · 3 Wall Headstand-PU exzentrisch · 4 Wall Headstand-PU · höher: freistehend, volle ROM | [A-02] *(S)* |
| Pull-ups / One-Arm | 7 Archer Pull-up · 8 One-Arm-Chin exzentrisch · 9 One-Arm-Chin · 10 One-Arm-Chin + 15 lb (≈ 6,8 kg) | [A-02] *(S)* |
| Core (Sitzpositionen) | L-Sit · Straddle-L · V-Sit 45°/75°/100°/120° · Manna (Level-Zahlen nicht gelesen) | [A-02] *(S)* |

**Konsistenzprüfung (eigene Ableitung):** FIG-A-Elemente liegen in OG auf
Level 7–8 (Full Back Lever 7, Full Front Lever 8, Straddle Planche 8), das
FIG-C-Element Full Planche auf ≥ 10 [A-02, A-05, A-06]. Die beiden Skalen sind
also für diese Anker monoton verträglich. Für B, D, E fehlen OG-Anker; die
Zuordnung unten ist dort Heuristik.

### 3.4 Hefesto-Ordinalskala (Vorschlag)

Ordinal = OG-Level, wo vorhanden; sonst Heuristik über FIG-Anker
(A ≈ 7–8, B ≈ 9, C ≈ 10–11, D ≈ 12–13, E ≈ 14–15). `difficulty_tier`
(1–10, Schema-Feld der Skills) = ⌈Ordinal × 10 / 16⌉ (PAR-A-37). Einstiegs-
wurzeln unter OG-Level 1 erhalten Ordinal 0 und Tier 1.

| Stufe | Ordinal | Tier | Status |
|---|---|---|---|
| Wand-Liegestütz, Schräg-Liegestütz, Support-Halt, Dead Hang, Scapula-Pulls, Wrist Prep | 0 | 1 | (H) — unter OG-Level 1 |
| Wall Handstand; German Hang; Pike Headstand-PU | 1 | 1 | OG [A-02] *(S)* |
| Skin the Cat; Box Headstand-PU | 2 | 2 | OG [A-02] *(S)* |
| Frog Stand; Tuck Back Lever; Wall Headstand-PU exzentrisch | 3 | 2 | OG [A-02] *(S)* |
| Tuck Front Lever; Adv Tuck Back Lever; Wall Headstand-PU | 4 | 3 | OG [A-02] *(S)* |
| Freestanding Handstand | 4–5 | 3–4 | OG [A-02] *(S)* |
| Tuck Planche; Adv Tuck Front Lever; Straddle Back Lever | 5 | 4 | OG [A-02] *(S)* |
| Adv Tuck Planche; Straddle Front Lever; Half-Lay/One-Leg Back Lever | 6 | 4 | OG [A-02] *(S)* |
| Half-Lay/One-Leg Front Lever; Full Back Lever (FIG A); Archer Pull-up | 7 | 5 | OG [A-02] *(S)*; FIG [A-05] |
| Straddle Planche (FIG A); Full Front Lever (FIG A); One-Arm-Chin exzentrisch | 8 | 5 | OG [A-02] *(S)*; FIG [A-05, A-06] |
| Half-Lay/One-Leg Planche; One-Arm-Chin | 9 | 6 | OG [A-02] *(S)* |
| Full Planche (FIG C); One-Arm-Chin + 6,8 kg | 10 | 7 | OG [A-02] *(S)*; FIG [A-05] |
| Iron Cross (FIG C seit 2025) | 10–11 | 7 | (H) über FIG-Anker [A-08] |
| Maltese Boden (FIG C) | 12 | 8 | (H): FIG-Wert wie Full Planche [A-07], aber in der PDF-Leiter klar nach der Full Planche [P-03; P-04] |
| Maltese Ringe (FIG D) | 13 | 9 | (H) über FIG-Anker [A-05, A-07] |
| Victorian (FIG E) | 14–15 | 9–10 | (H) über FIG-Anker [A-05] |
| Muscle-up, Human Flag, Pistol, L-/V-Sit, Manna | — | — | keine Level-Zahl gefunden; Einstufung in den Skill-Abschnitten als (H) |

### 3.5 Relative Last innerhalb der Liegestütz-Leiter (belegt)

| Variante | Anteil Körpergewicht | relativ zum Standard (= 1,00) | Quelle |
|---|---|---|---|
| Hände 61 cm erhöht | 41 % | 0,64 | [A-15] |
| Knie-Liegestütz | 49 % (Spitzenkraft); 53,6 % oben / 61,8 % unten | 0,77 | [A-15]; [A-16] |
| Hände 30,5 cm erhöht | 55 % | 0,86 | [A-15] |
| Standard-Liegestütz | 64 % (Spitzenkraft); 69,2 % oben / 75,0 % unten | 1,00 | [A-15]; [A-16] |
| Füsse 30,5 cm erhöht | 70 % | 1,09 | [A-15] |
| Füsse 61 cm erhöht | 74 % | 1,16 | [A-15] |

Relativwerte: eigene Rechnung aus [A-15]. Die Differenz zwischen [A-15] und
[A-16] ist in «Widersprüche» dokumentiert.

## 4. Wurzeln: Grundlagen unter dem PDF-«Beginner»

Die PDFs setzen eine solide Tuck-Planche voraus (`01_pdf_extract.md` §4.7). Für
echte Anfänger braucht der Graph Wurzeln. Grundgerüst ist die
r/bodyweightfitness-Routine (RR) mit Push-up-, Dip-, Row- und Pull-up-Leiter und
dem Aufwärmblock [A-03]; die Wirksamkeit progressiver Varianten ist für den
Liegestütz durch ein RCT gestützt [A-14]. Alles darüber hinaus ist markiert.

### 4.1 Handgelenk-Konditionierung (`wrist-conditioning`, Familie `mobility`)

Warum als Wurzel: Im Handstand wird die Balance zuerst über die Handgelenke
reguliert («wrist strategy»), was den Druck auf die Handgelenke erhöht [A-20].
Die RR stellt eine Handgelenk-Vorbereitung (GMB Wrist Prep, 10+ Wdh. je Übung)
an den Anfang jeder Einheit [A-03].

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `wrist-conditioning/prep` · `wrist-prep-routine` | Handgelenk-Vorbereitung im Vierfüsslerstand (Kreisen, Handflächen-/Handrücken-Varianten) | schmerzfrei, kontrolliert, voller aktiver Bewegungsumfang (H-FORM) | Selbstbestätigung oder `wrist-prep-routine reps ≥ 10 · occ 3 · 14 d` (H-UNL) | 1–2 Wo. (H-DUR) | [A-03] (Übung, 10+ Wdh.) |
| 2 | `wrist-conditioning/loaded-extension` · `quadruped-wrist-rock` | belastete Extension: Schultern im Vierfüsslerstand über/vor die Hände schieben | Finger gespreizt, Handballen am Boden, keine Schmerzen (H-FORM) | `quadruped-wrist-rock reps ≥ 15 · occ 3 · 14 d` (H-UNL) | 1–3 Wo. (H-DUR) | (H) |
| 3 | `wrist-conditioning/plank-lean` · `plank-lean-hold` | Liegestützposition, Schultern leicht vor den Händen | Arme gestreckt, Körper gerade, Schulterblätter protrahiert (H-FORM) | `plank-lean-hold hold ≥ 30 s · occ 3 · 7 d` (Isometrie-Regel [A-03]) | geht in `planche/lean` über (§5.5) | [A-03] (Regel); Übung (H) |

- **Voraussetzungen:** keine (Wurzel).
- **Häufige Fehler:** Schmerz «durchtrainieren»; Finger nicht gespreizt; Last
  zu früh auf volle Extension (H-FAULT).
- **Carryover:** Handstand, Planche-Lean, Liegestütz-Varianten mit Fingern nach
  hinten (Pseudo-Planche) (H-PRE).
- **Equipment:** Parallettes/Griffe reduzieren die Handgelenk-Extension
  gegenüber dem Boden (H-EQ). Die PDFs üben Leans bewusst am Boden mit
  `supi`-Handstellung [P-01 S. 1, 3] (Deutung von `supi` unsicher, §9).
- **Empfehlung für den Graphen:** Stufe 1 als Voraussetzung (`recommended`,
  nicht `prerequisite`) für `handstand/wall` und `planche/lean`, damit
  Anfänger nicht blockiert werden (H-PRE).

### 4.2 Hang- und Scapula-Grundlagen (`hang-foundation`, Familie `pull`)

Die RR beginnt die Pull-up-Leiter mit Scapula-Pulls und Arch Hangs, gefolgt von
negativen Klimmzügen [A-03]. Weil `pull-up` bereits mit `strict-5` beginnt,
werden diese Stufen als eigener Wurzel-Skill vorgeschlagen (§2.1).

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `hang-foundation/dead-hang` · `dead-hang` | passiver Hang an der Stange | Arme gestreckt, ohne Bodenkontakt (H-FORM) | `dead-hang hold ≥ 30 s · occ 3 · 7 d` | 1–2 Wo. (H-DUR) | Isometrie-Regel 30 s [A-03]; Stufe (H) |
| 2 | `hang-foundation/scapular-pull` · `scapular-pull-up` (existiert) | aus dem Hang Schulterblätter nach unten/hinten ziehen, Arme bleiben gestreckt | Ellbogen gestreckt, Bewegung nur aus den Schulterblättern (H-FORM) | `scapular-pull-up reps ≥ 8 · occ 3 · 7 d` | 1–2 Wo. (H-DUR) | Reihenfolge und 3×8-Regel [A-03] |
| 3 | `hang-foundation/arch-hang` · `arch-hang` | Scapula-Zug plus Brustkorb nach oben/vorn, leichter Bogen | Arme gestreckt, Brust Richtung Stange (H-FORM) | `arch-hang reps ≥ 8 · occ 3 · 7 d` | 1–3 Wo. (H-DUR) | Reihenfolge [A-03]; Aufwärm-Dosis 10 Wdh. [A-03] |
| (Übung) | `pull-up-negative` (Rolle `progression` bei `pull-up/strict-5`) | langsames Absenken aus der oberen Position | kontrolliert, voller Weg (H-FORM) | kein Level (exzentrisch zählt nicht, §2.4) | — | Reihenfolge [A-03] |

- **Voraussetzungen:** keine.
- **Häufige Fehler:** Scapula-Pull mit gebeugten Ellbogen (wird zum
  Teil-Klimmzug); Schultern hochgezogen im Hang (H-FAULT).
- **Carryover:** Klimmzug, Front Lever, Muscle-up; die RR ergänzt Arch Hangs
  im Aufwärmen, sobald negative Klimmzüge erreicht sind [A-03].
- **Equipment:** Ringe statt Stange erlauben freie Rotation der Hände (H-EQ).

### 4.3 Körperspannung: Hollow Body und Arch (`hollow-body`, `arch-body`, Familie `core`)

Grundpositionen (fundamental static positions) sind Teil des Turn-
Konditionstrainings nach Sommer [A-13]. Die RR enthält im Aufwärmen Deadbugs
30 s [A-03]. Stufenschwellen sind nicht belegt.

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `hollow-body/tuck` · `hollow-hold-tuck` | Rückenlage, Knie angezogen, Schultern vom Boden | Lendenwirbelsäule am Boden (H-FORM) | `hollow-hold-tuck hold ≥ 30 s · occ 3 · 7 d` | 1–2 Wo. (H-DUR) | Regel [A-03]; Stufe (H) |
| 2 | `hollow-body/full` · `hollow-hold` | Arme über Kopf, Beine gestreckt, knapp über dem Boden | Lendenwirbelsäule bleibt am Boden, Beine gestreckt (H-FORM) | `hollow-hold hold ≥ 30 s · form≥4 · occ 3 · 7 d` | 2–4 Wo. (H-DUR) | Regel [A-03]; Stufe (H) |
| 3 | `hollow-body/rocks` · `hollow-rock` | Schaukeln in der Hollow-Position | Form bleibt über die ganze Wiederholung (H-FORM) | `hollow-rock reps ≥ 15 · occ 3 · 7 d` (H-UNL) | — | (H) |
| 1 | `arch-body/hold` · `arch-hold` | Bauchlage, Arme/Beine gestreckt angehoben | Gesäss angespannt, Nacken neutral (H-FORM) | `arch-hold hold ≥ 30 s · occ 3 · 7 d` | 1–2 Wo. (H-DUR) | Regel [A-03]; Stufe (H) |

- **Voraussetzungen:** keine.
- **Häufige Fehler:** Hohlkreuz im Hollow; Nacken überstreckt im Arch (H-FAULT).
- **Carryover:** Hollow → Front Lever, Handstand-Linie, Planche (Becken-
  position); Arch → Back Lever, Tap Swing (H-PRE).
- **Equipment:** keines.

### 4.4 Stützhalte (`support-hold`, Familie `push`)

Die RR-Dip-Leiter beginnt mit dem Parallelbarren-Stütz; ab den negativen Dips
kommt ein 30-s-Stützhalt ins Aufwärmen [A-03].

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `support-hold/parallel-bars` · `support-hold-pb` | Stütz auf Barren/Dip-Griffen | Ellbogen gestreckt, Schultern tief (nicht hochgezogen) (H-FORM) | `support-hold-pb hold ≥ 30 s · occ 3 · 7 d` | 1–2 Wo. (H-DUR) | [A-03] |
| 2 | `support-hold/rings` · `support-hold-rings` | Stütz an Ringen | wie 1, Ringe nah am Körper (H-FORM) | `support-hold-rings hold ≥ 30 s · occ 3 · 7 d` | 2–4 Wo. (H-DUR) | Regel [A-03]; Stufe (H) |
| 3 | `support-hold/rings-turned-out` · `support-hold-rings-rto` | Ringe nach aussen gedreht (RTO) | Ellbogenbeuge zeigt nach vorn (H-FORM) | `support-hold-rings-rto hold ≥ 30 s · form≥4 · occ 3 · 7 d` | — | (H) |

- **Voraussetzungen:** keine.
- **Häufige Fehler:** Schultern hochgezogen, Ellbogen gebeugt, Ringe driften
  vom Körper weg (H-FAULT).
- **Carryover:** Dips, Muscle-up-Endposition, L-Sit, Planche an Ringen (H-PRE).
- **Equipment:** Barren (stabil) < Ringe (instabil) < Ringe RTO (H-EQ).

### 4.5 Liegestütz (`push-up`, Familie `push`)

Reihenfolge nach RR: Wand → Schräg → Standard → Diamond → Pseudo-Planche [A-03].
Lastanteile siehe §3.5 [A-15, A-16]. Progressives Liegestütz-Training erzielt
nach 4 Wochen (3×/Woche) vergleichbare Kraft- und Dickenzuwächse wie
Bankdrücken (**Evidenz A**) [A-14].

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `push-up/wall` · `push-up-wall` | Hände an der Wand | Körper gerade, voller Weg (H-FORM) | `push-up-wall reps ≥ 8 · occ 3 · 7 d` | ≥ 4 Einheiten; 1–2 Wo. (H-DUR) | Stufe, Regel [A-03] |
| 2 | `push-up/incline` · `push-up-incline` | Hände erhöht (Bank, Stange) | wie 1; Last 41–55 % KG je nach Höhe [A-15] | `push-up-incline reps ≥ 8 · occ 3 · 7 d` | 1–3 Wo. (H-DUR) | [A-03, A-15] |
| 2a | Alternative: `push-up-knee` (Übung, Rolle `progression`) | Knie am Boden | 49 % KG [A-15]; 54–62 % [A-16] | kein eigenes Level (Alternative zu 2) | — | [A-15, A-16] |
| 3 | `push-up/full` · `push-up` | Standard-Liegestütz | Körperlinie ohne Durchhängen/Abknicken; Brust nahe Boden; Ellbogen oben gestreckt (H-FORM) | `push-up reps ≥ 8 · occ 3 · 7 d` | 1–4 Wo. (H-DUR) | [A-03]; 64 % KG [A-15] |
| 4 | `push-up/diamond` · `push-up-diamond` | Hände eng unter der Brust | wie 3 (H-FORM) | `push-up-diamond reps ≥ 8 · occ 3 · 7 d` | 2–4 Wo. (H-DUR) | [A-03] |
| 5 | `push-up/pseudo-planche` · `pseudo-planche-push-up` | Hände auf Hüfthöhe, Schultern vor den Händen | Protraktion, Arme oben gestreckt, Schultern bleiben vor den Händen (H-FORM) | `pseudo-planche-push-up reps ≥ 8 · form≥4 · occ 3 · 7 d` | — | [A-03]; in den PDFs Zubringer 5–15 Wdh. [P-01 S. 1; P-03 S. 3] |
| (Alt.) | `push-up-decline` (Übung) | Füsse erhöht | 70–74 % KG [A-15] | als Überlastungsvariante zu Stufe 3/4 | — | [A-15] |

- **Voraussetzungen:** keine; `wrist-conditioning/prep` empfohlen (H-PRE).
- **Häufige Fehler:** Hüfte hängt durch oder knickt ein, halber Weg,
  Schulterblätter kollabieren oben (H-FAULT).
- **Carryover:** Dip, Planche-Lean und Pseudo-Planche-Liegestütz → Planche
  [P-01 bis P-03]; Bankdrück-Kraft [A-14].
- **Equipment:** Boden, Parallettes (tiefere ROM möglich, neutraler Griff),
  Ringe (instabil) — alle (H-EQ); Erhöhung der Hände senkt, Erhöhung der Füsse
  erhöht die Last [A-15].

### 4.6 Dip (`dip`, Familie `push`)

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| (Vorst.) | `support-hold/parallel-bars` (§4.4) | Stütz | — | siehe §4.4 | — | [A-03] |
| (Übung) | `dip-negative` (Rolle `progression`) | langsames Absenken | kontrolliert (H-FORM) | kein Level (§2.4) | — | [A-03] |
| 1 | `dip/parallel-bars` · `dip-pb` | Barren-Dip | oben gestreckt, unten Oberarm mindestens parallel zum Boden, Schultern nicht hochgezogen (H-FORM) | `dip-pb reps ≥ 8 · occ 3 · 7 d` | 2–6 Wo. (H-DUR) | [A-03] |
| 2 | `dip/rings` · `dip-rings` | Ring-Dip | wie 1, Ringe nah am Körper, oben stabiler Stütz (H-FORM) | `dip-rings reps ≥ 8 · occ 3 · 7 d` | 4–8 Wo. (H-DUR) | [A-03] |
| 3 | `dip/weighted` · `dip-weighted` | Barren-Dip mit Zusatzlast | wie 1 | siehe §7.4 | — | (H) |

- **Voraussetzungen:** `support-hold/parallel-bars` (`prerequisite`) [A-03];
  für Ring-Dips `support-hold/rings` (`prerequisite`) (H-PRE).
- **Häufige Fehler:** zu flach, Schultern rollen nach vorn/oben, Schwung aus
  den Beinen (H-FAULT).
- **Carryover:** Muscle-up (Stützphase), HSPU, Planche-Liegestütz (H-PRE).
- **Equipment:** Barren < gerade Stange (Straight-Bar-Dip) < Ringe (H-EQ; die RR
  stellt Ring-Dips hinter Barren-Dips [A-03]).

### 4.7 Australian Row / Rudern (`row`, Familie `pull`)

Reihenfolge nach RR: vertikal → schräg → horizontal → breit → Archer [A-03]. Die
RR führt den Tuck Front Lever als isometrische Stufe in der Row-Progression
[A-03] (siehe §5.3).

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `row/vertical` · `row-vertical` | fast aufrecht an Ringen/Tisch | Körper gerade, Brust zu den Griffen, Schulterblätter zurück (H-FORM) | `row-vertical reps ≥ 8 · occ 3 · 7 d` | 1–2 Wo. (H-DUR) | [A-03] |
| 2 | `row/incline` · `row-incline` | schräg | wie 1 | `row-incline reps ≥ 8 · occ 3 · 7 d` | 1–3 Wo. (H-DUR) | [A-03] |
| 3 | `row/horizontal` · `row-horizontal` | Körper waagrecht (Australian Row) | wie 1; Arme unten gestreckt (H-FORM) | `row-horizontal reps ≥ 8 · occ 3 · 7 d` | 2–4 Wo. (H-DUR) | [A-03] |
| 4 | `row/wide` · `row-wide` | breiter Griff | wie 3 | `row-wide reps ≥ 8 · occ 3 · 7 d` | 2–4 Wo. (H-DUR) | [A-03] |
| 5 | `row/archer` · `row-archer` | ein Arm zieht, der andere bleibt gestreckt | Rumpf rotiert nicht (H-FORM) | `row-archer reps ≥ 8 · occ 3 · 7 d` (je Seite) | — | [A-03] |

- **Voraussetzungen:** keine (Wurzel).
- **Häufige Fehler:** Hüfte hängt, Kinn zieht statt Brust, halber Weg (H-FAULT).
- **Carryover:** Klimmzug, Front Lever (Rows im Tuck-FL sind eine OG-Spalte
  [A-02] *(S)*).
- **Equipment:** Die Last steigt mit flacherem Körperwinkel (H-EQ; Messwerte
  in dieser Sitzung nicht verifiziert); Ringe erlauben Rotation, Tisch/Stange
  fixieren den Griff.

### 4.8 Klimmzug / Chin-up (`pull-up`, Familie `pull`; bestehender Skill)

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| (Wurzel) | `hang-foundation/arch-hang` (§4.2) | — | — | — | — | [A-03] |
| 1 | `pull-up/strict-5` (existiert) · `pull-up` | 5 strikte Klimmzüge | aus gestrecktem Hang, Kinn über Stange, kein Kipping (H-FORM; Kipping ändert die Kinematik [A-19]) | `pull-up reps ≥ 5 · none · max_load_kg 0 · occ 2 · 28 d` (H-UNL) | 4–12 Wo. ab Arch Hang (H-DUR) | [A-03] (Reihenfolge) |
| 2 | `pull-up/strict-8` · `pull-up` | 3×8 strikt | wie 1 | `pull-up reps ≥ 8 · none · max_load_kg 0 · occ 3 · 7 d` | 2–8 Wo. (H-DUR) | Regel 3×8 [A-03] |
| 3 | `pull-up/chest-to-bar` · `pull-up-chest-to-bar` | Brust berührt Stange | wie 1, Brust an der Stange (H-FORM) | `pull-up-chest-to-bar reps ≥ 5 · occ 2 · 28 d` (H-UNL) | 4–8 Wo. (H-DUR) | (H) — Muscle-up-Zubringer |
| 4 | `pull-up/archer` · `pull-up-archer` | Archer-Klimmzug | Hilfsarm gestreckt (H-FORM) | `pull-up-archer reps ≥ 3 · occ 2 · 28 d` (je Seite) (H-UNL) | siehe §7.1 | OG-Level 7 [A-02] *(S)* |
| (Alt.) | `chin-up` (Übung) | Untergriff | wie 1 | als Alternative zu Stufe 1–2 | — | EMG-Vergleich Pull-up/Chin-up [A-17] |
| (weiter) | Weighted Pull-up §7.4; One-Arm §7.1 | | | | | [A-03] (Weighted als letzte RR-Stufe) |

- **Voraussetzungen:** `hang-foundation/arch-hang` (`prerequisite`) [A-03].
- **Häufige Fehler:** halber Weg unten, Kipping/Beinschwung, Kinn reckt statt
  Zug bis Kinn über Stange (H-FAULT).
- **Carryover:** Rows, Front Lever, Muscle-up, One-Arm-Chin (H-PRE); die
  Griff-/Technikvariante verändert die Kräfte in der Rotatorenmanschette
  [A-18] — für die Belastungssteuerung relevant (Stream C/D).
- **Equipment:** Stange (fixer Griff) vs. Ringe (freie Rotation); ein
  rotierender Griff brachte keine höhere Muskelaktivierung als Pull-up oder
  Chin-up [A-17].

### 4.9 Kompression / Pike-Grundlagen (Teil von `l-sit`, Familie `core`)

Keine Quelle für eigene Stufen gefunden. Vorschlag: Kompression wird nicht als
eigener Skill geführt, sondern als **Übungen mit Rolle `accessory`** an den
ersten L-Sit-Stufen (§5.2): `seated-pike-leg-lift` (sitzend, Hände neben den
Knien, gestreckte Beine anheben) und `pike-compression-hold`. Unlock-relevant
ist erst der Tuck-L-Sit (H: Kompression lässt sich im Log schlecht objektiv
messen; der L-Sit ist der messbare Nachweis).

## 5. Statics

### 5.1 Handstand (`handstand`, Familie `handstand`; bestehender Skill)

Belegt: Wall-HS = OG-Level 1, freistehend = Level 4–5 [A-02] *(S)*; FIG führt
den Handstand (2 s) als Bodenelement [A-04] *(S)*. Biomechanik (**Evidenz A**):
Balance primär über die Handgelenke; gelingt das nicht, gemischte Strategie mit
Handgelenk, Schulter, Hüfte, Ellbogen; Augen schliessen senkt die Stabilität,
Nackenflexion verschlechtert die Leistung [A-20].

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `handstand/wall` (existiert) · `wall-handstand-hold` | Handstand mit Rücken zur Wand | Arme gestreckt, Kopf neutral [A-20] | `wall-handstand-hold hold ≥ 30 s · occ 3 · 14 d` (Isometrie-Regel [A-03], H-UNL) | 2–4 Wo. (H-DUR) | OG 1 [A-02] *(S)* |
| 2 | `handstand/chest-to-wall` · `handstand-chest-to-wall` | Bauch zur Wand, Hände nahe der Wand | Hand–Schulter–Hüfte–Fuss gestapelt, Rippen drin, Schultern voll geöffnet (H-FORM) | `handstand-chest-to-wall hold ≥ 60 s · form≥4 · occ 2 · 14 d` (H-UNL: Linienausdauer als Basis fürs Balancieren) | 2–6 Wo. (H-DUR) | (H) |
| 3 | `handstand/free-10s` · `handstand-freestanding` | freistehend | wie 2; Korrektur über Finger/Handgelenk statt Ellbogen/Hüfte [A-20] | `handstand-freestanding hold ≥ 10 s · form≥4 · occ 3 · 14 d` | 4–16 Wo. (H-DUR) | OG 4–5 [A-02] *(S)*; 10-s-Regel [A-12] übertragen (H-UNL) |
| 4 | `handstand/free-30s` · `handstand-freestanding` | freistehend 30 s | wie 3 | `handstand-freestanding hold ≥ 30 s · form≥4 · occ 3 · 14 d` | 4–16 Wo. (H-DUR) | (H-UNL) |
| 5 | `handstand/free-60s` · `handstand-freestanding` | freistehend 60 s | wie 3 | `handstand-freestanding hold ≥ 60 s · form≥4 · occ 2 · 28 d` | — | (H-UNL) |
| P1 | `press-handstand/straddle` · `press-handstand-straddle` | Press aus dem Grätschstand in den Handstand | Arme gestreckt, kein Sprung (H-FORM) | `press-handstand-straddle reps ≥ 1 · form≥4 · occ 2 · 28 d` | 8–26 Wo. (H-DUR) | FIG listet Press-Elemente zum Handstand [A-04] *(S)*; Sommer behandelt Press-Handstände [A-13] |
| P2 | `press-handstand/pike` · `press-handstand-pike` | Press mit geschlossenen Beinen | wie P1 | `press-handstand-pike reps ≥ 1 · form≥4 · occ 2 · 28 d` | — | (H) |

- **Voraussetzungen:** `wrist-conditioning/prep` (`recommended`);
  `push-up/full` (`recommended`) (H-PRE). Press-Handstand: `handstand/free-10s`
  (`prerequisite`) und `l-sit/full` (`recommended`) (H-PRE).
- **Häufige Fehler:** Hohlkreuz («Banane»), gebeugte Arme, Kopf zu weit
  eingerollt oder überstreckt [A-20: Nackenflexion verschlechtert die Balance],
  Korrektur über Hüfte/Ellbogen statt Handgelenk [A-20].
- **Carryover:** HSPU (§7.2), Planche-Press (§5.5), Press-Handstand (H-PRE).
- **Equipment:** Boden (Handgelenk in voller Extension), Parallettes (neutraler
  Griff, andere Korrekturstrategie über den Griff) (H-EQ); Ringe haben in OG
  eine eigene, deutlich schwerere Spalte [A-02] *(S)*.

### 5.2 L-Sit → V-Sit → Manna (`l-sit`, `v-sit`, `manna`, Familie `core`)

Belegt ist nur die Reihenfolge: L-Sit → Straddle-L → V-Sit 45°/75°/100°/120° →
Manna [A-02] *(S)*; Sommer behandelt L-Sit und Manna [A-13]; die Manna (2 s) ist
ein FIG-Bodenelement, auch mit anschliessendem Press zum Handstand [A-04] *(S)*.

| Skill / Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| L 1 | `l-sit/tuck` · `l-sit-tuck` | Stütz, Knie angezogen | Arme gestreckt, Schultern tief, Gesäss frei (H-FORM) | `l-sit-tuck hold ≥ 30 s · occ 3 · 7 d` | 1–4 Wo. (H-DUR) | Isometrie-Regel [A-03]; Stufe (H) |
| L 2 | `l-sit/one-leg` · `l-sit-one-leg` | ein Bein gestreckt | wie 1, gestrecktes Bein waagrecht (H-FORM) | `l-sit-one-leg hold ≥ 20 s · occ 3 · 14 d` (je Seite, H-UNL) | 2–4 Wo. (H-DUR) | (H) |
| L 3 | `l-sit/full` · `l-sit` | L-Sit | Beine gestreckt und waagrecht, Knie gestreckt (H-FORM) | `l-sit hold ≥ 10 s · form≥4 · occ 2 · 28 d` | 2–8 Wo. (H-DUR) | Reihenfolge [A-02] *(S)*; 10 s [A-12] übertragen |
| L 4 | `l-sit/full-30s` · `l-sit` | L-Sit 30 s | wie 3 | `l-sit hold ≥ 30 s · form≥4 · occ 2 · 28 d` | 4–12 Wo. (H-DUR) | 30 s [A-03] übertragen |
| L 5 | `l-sit/straddle` · `straddle-l-sit` | gegrätschter L | Beine gestreckt, weit gegrätscht, waagrecht oder höher (H-FORM) | `straddle-l-sit hold ≥ 10 s · form≥4 · occ 2 · 28 d` | 4–12 Wo. (H-DUR) | [A-02] *(S)* |
| V 1–4 | `v-sit/45` · `/75` · `/100` · `/120` · `v-sit` | V-Sit, Beinwinkel über der Waagrechten | Winkel erreicht, Arme gestreckt (H-FORM) | `v-sit hold ≥ 5 s · form≥4 · occ 2 · 28 d` je Winkelstufe (eigene Exercise-Slugs je Winkel oder Winkel als Formkriterium) (H-UNL) | je 4–16 Wo. (H-DUR) | Winkelstufen [A-02] *(S)* |
| M 1 | `manna/full` · `manna` | Beine waagrecht, Rumpf hinter den Händen | Beine waagrecht, Arme gestreckt (H-FORM) | `manna hold ≥ 3 s · form≥4 · occ 2 · 28 d` | — | FIG 2 s [A-04] *(S)*; 3 s [A-09] |

- **Voraussetzungen:** `support-hold/parallel-bars` für `l-sit/tuck`
  (`prerequisite`) (H-PRE); `v-sit/45` ← `l-sit/straddle`; `manna/full` ←
  `v-sit/120` (`prerequisite`, Reihenfolge [A-02] *(S)*).
- **Häufige Fehler:** Schultern hochgezogen, Knie gebeugt, Rundrücken statt
  aktiver Kompression (H-FAULT).
- **Carryover:** Press-Handstand (Kompression), Planche-Übergänge «L-Sit to
  Tuck Planche» [P-01 S. 1–3], Muscle-up-Varianten (H-PRE).
- **Equipment:** Boden (schwerste Variante: wenig Freiraum) > Parallettes/
  Barren (mehr Freiraum) ; Ringe (instabil) (H-EQ).

### 5.3 Front Lever (`front-lever`, Familie `pull`; bestehender Skill)

Belegt: OG-Level Tuck 4, Adv Tuck 5, Straddle 6, Half-Lay/One-Leg 7, Full 8
[A-02] *(S)*; FIG-Wert A (Ringe) [A-05]; im Street-Workout-Wettkampf zählen Tuck,
Adv Tuck und 45°-Straddle nicht als Halt [A-10] *(S)*; die RR nutzt den Tuck FL
als Isometrie in der Row-Leiter [A-03].

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `front-lever/tuck` (existiert) · `front-lever-tuck` | Knie zur Brust, Rumpf waagrecht | Arme gestreckt, Schultern depressiert, Rumpf waagrecht ±15° (H-FORM; [A-04] *(S)*) | `front-lever-tuck hold ≥ 10 s · none · form≥4 · occ 2 · 28 d` | 4–12 Wo. ab `pull-up/strict-5` (H-DUR) | OG 4 [A-02] *(S)*; 10 s [A-12]; Platzhalter nennt 15 s |
| 2 | `front-lever/advanced-tuck` (existiert) · `front-lever-advanced-tuck` | Rücken flach, Hüfte offen, Knie gebeugt | flacher Rücken, Oberschenkel etwa senkrecht zum Rumpf (H-FORM) | `front-lever-advanced-tuck hold ≥ 10 s · none · form≥4 · occ 2 · 28 d` | 4–12 Wo. (H-DUR) | OG 5 [A-02] *(S)*; [A-12] |
| 3 | `front-lever/straddle` · `front-lever-straddle` | Beine gestreckt, gegrätscht | Körper waagrecht ±15°, Beine gestreckt (H-FORM) | `front-lever-straddle hold ≥ 3 s · none · form≥4 · occ 2 · 28 d` | 8–26 Wo. (H-DUR) | OG 6 [A-02] *(S)*; 3 s [A-09] |
| 4 | `front-lever/one-leg` · `front-lever-one-leg` (Alt.: `front-lever-half-lay`) | ein Bein gestreckt / Knie halb gebeugt | Hüfte gestreckt, Körperlinie waagrecht (H-FORM) | `front-lever-one-leg hold ≥ 10 s · none · form≥4 · occ 2 · 28 d` | 4–16 Wo. (H-DUR) | OG 7 [A-02] *(S)* |
| 5 | `front-lever/full` · `front-lever` | Full Front Lever | Körper gestreckt, waagrecht ±15°, Arme gestreckt | `front-lever hold ≥ 3 s · none · form≥4 · occ 2 · 28 d` | — | OG 8 [A-02] *(S)*; FIG A [A-05]; 3 s [A-09] |
| (Übungen) | `front-lever-row-tuck`, `band-assisted-tuck-fl` (existiert), `front-lever-negative` | Rows in Tuck-Position, Band-Assistenz, Negative | — | Rolle `progression`; kein Level | — | FL-Rows als OG-Spalte [A-02] *(S)* |

- **Voraussetzungen:** `pull-up/strict-5` (`prerequisite`, wie im Platzhalter)
  (H-PRE); `hollow-body/full` (`recommended`) (H-PRE); `row/horizontal`
  (`recommended`, RR-Kontext [A-03]).
- **Häufige Fehler:** Hüfte unter Schulterlinie, gebeugte Ellbogen, Schultern
  nicht depressiert, runder Rücken im Adv Tuck (H-FAULT; die ersten zwei auch im
  bestehenden Platzhalter).
- **Carryover:** Back Lever (gemeinsame Straight-Arm-Zugmuster), Klimmzug,
  Muscle-up-Übergang, Maltese/Planche als **Antagonist** (Druck vs. Zug) (H-PRE).
- **Equipment:** Stange vs. Ringe (freie Rotation, gleiche Stufen) (H-EQ);
  Anthropometrie (Beinlänge) verschiebt die Schwierigkeit der Straddle/Half-Lay-
  Stufen → Stream C.

### 5.4 Back Lever inkl. German Hang und Skin the Cat (`back-lever`, Familie `pull`)

Belegt: OG-Level German Hang 1, Skin the Cat 2, Tuck 3, Adv Tuck 4, Straddle 5,
Half-Lay/One-Leg 6, Full 7 [A-02] *(S)*; FIG-Wert A [A-05].

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `back-lever/german-hang` · `german-hang` | Hang in maximaler Schulterextension | kontrolliertes Ein- und Aussteigen, schmerzfrei (H-FORM) | `german-hang hold ≥ 15 s · occ 3 · 14 d` (H-UNL: Gewebetoleranz statt Kraft) | 2–6 Wo. (H-DUR) | OG 1 [A-02] *(S)* |
| 2 | `back-lever/skin-the-cat` · `skin-the-cat` | Durchdrehen in den German Hang und zurück | Arme gestreckt, langsam (H-FORM) | `skin-the-cat reps ≥ 3 · occ 2 · 14 d` (H-UNL) | 2–6 Wo. (H-DUR) | OG 2 [A-02] *(S)* |
| 3 | `back-lever/tuck` · `back-lever-tuck` | Tuck, Rücken nach unten | Arme gestreckt, Rumpf waagrecht (H-FORM) | `back-lever-tuck hold ≥ 10 s · form≥4 · occ 2 · 28 d` | 2–8 Wo. (H-DUR) | OG 3 [A-02] *(S)*; [A-12] |
| 4 | `back-lever/advanced-tuck` · `back-lever-advanced-tuck` | Hüfte gestreckt, Knie gebeugt | flacher Rücken (H-FORM) | `back-lever-advanced-tuck hold ≥ 10 s · form≥4 · occ 2 · 28 d` | 4–12 Wo. (H-DUR) | OG 4 [A-02] *(S)* |
| 5 | `back-lever/straddle` · `back-lever-straddle` | gegrätscht | waagrecht ±15° (H-FORM) | `back-lever-straddle hold ≥ 3 s · form≥4 · occ 2 · 28 d` | 4–12 Wo. (H-DUR) | OG 5 [A-02] *(S)*; [A-09] |
| 6 | `back-lever/one-leg` · `back-lever-one-leg` | ein Bein / Half-Lay | wie 5 | `back-lever-one-leg hold ≥ 10 s · form≥4 · occ 2 · 28 d` | 4–12 Wo. (H-DUR) | OG 6 [A-02] *(S)* |
| 7 | `back-lever/full` · `back-lever` | Full Back Lever | gestreckte Linie, waagrecht ±15° | `back-lever hold ≥ 3 s · form≥4 · occ 2 · 28 d` | — | OG 7 [A-02] *(S)*; FIG A [A-05] |

- **Voraussetzungen:** `hang-foundation/dead-hang` (`prerequisite`) (H-PRE);
  `arch-body/hold` (`recommended`) (H-PRE).
- **Häufige Fehler:** Hüfte knickt ein, Arme beugen, zu schnelles Einrollen in
  den German Hang (H-FAULT). Straight-Arm-Belastung der Bizepssehne und der
  vorderen Schulter ist Thema von Stream D (ADR 0003 nennt diese Strukturen).
- **Carryover:** Front Lever, Maltese/Victorian-Familie, Muscle-up-Schwungteile
  (H-PRE).
- **Equipment:** Ringe (Hände rotieren frei) vs. Stange (Griff fix; Ober- oder
  Untergriff) (H-EQ).

### 5.5 Planche (`planche`, `planche-press`, `planche-push-up`, Familie `push`)

Belegt: OG-Level Frog Stand 3, Tuck 5, Adv Tuck 6, Straddle 8, Half-Lay/One-Leg
9, Full ≥ 10 [A-02] *(S)*; FIG Boden Straddle A / Full C [A-06; A-04 *(S)*],
Ringe Straddle A / Full C [A-05]. Die PDF-Leiter: Lean → Tuck (5–10 s) → Adv Tuck
(über «L-Sit to Adv Tuck») → (Wide) Straddle 2–4 s → Wide Planche 3–6 s → Full
(Press, Kicks) (`01_pdf_extract.md` §4.6; [P-01 S. 1–3; P-02 S. 1; P-03]).
Wechselregel 10 s sauber je Stufe [A-12]; Tuck → Straddle «fliessender» [A-11]
*(S)*. Die Stützwaage an den Ringen aktiviert Trapezius, Pectoralis und Bizeps
deutlich anders als ihre fünf gängigen Vorübungen [A-25]; ihre Leistung
korreliert mit der Vorübung «Schwalbe in Rückenlage» (r = 0,69) [A-22].

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 0 | `planche/lean` · `planche-lean` | Liegestützposition, Schultern weit vor den Händen | Arme gestreckt, Protraktion, Körper gerade (H-FORM) | `planche-lean hold ≥ 20 s · form≥4 · occ 2 · 28 d` (20 s = obere PDF-Dosis [P-01 S. 3]; H-UNL) | 2–6 Wo. (H-DUR) | PDF-Zubringer 8–20 s [P-01 S. 1, 3] |
| 1 | `planche/frog-stand` · `frog-stand` | Knie auf den Ellbogen, Balance | Füsse frei, kontrolliert (H-FORM) | `frog-stand hold ≥ 30 s · occ 3 · 7 d` ([A-03] Isometrie-Regel) | 1–4 Wo. (H-DUR) | OG 3 [A-02] *(S)* |
| 2 | `planche/tuck` · `planche-tuck` | Tuck, Arme gestreckt | Arme gestreckt, Hüfte auf Schulterhöhe ±15°, Protraktion (H-FORM; [A-04] *(S)*) | `planche-tuck hold ≥ 10 s · none · form≥4 · occ 2 · 28 d` | 4–16 Wo. (H-DUR) | OG 5 [A-02] *(S)*; [A-12]; PDF-Ziel 5–10 s [P-01 S. 1] |
| 3 | `planche/advanced-tuck` · `planche-advanced-tuck` | Rücken flach, Knie vom Körper weg | flacher Rücken, Hüfte auf Schulterhöhe (H-FORM; [A-12] *(S)*) | `planche-advanced-tuck hold ≥ 10 s · none · form≥4 · occ 2 · 28 d` | 8–26 Wo. (H-DUR) | OG 6 [A-02] *(S)*; [A-12] |
| 4 | `planche/straddle` · `planche-straddle` | Beine gestreckt, gegrätscht | Körper waagrecht ±15°, Arme gestreckt | `planche-straddle hold ≥ 3 s · none · form≥4 · occ 2 · 28 d` | 12–52 Wo. (H-DUR) | OG 8 [A-02] *(S)*; FIG A [A-05, A-06]; 3 s [A-09]; PDF 2–4 s [P-01 S. 2] |
| 5 | `planche/half-lay` · `planche-half-lay` (Alt.: `planche-one-leg`) | Knie halb gebeugt / ein Bein | wie 4 | `planche-half-lay hold ≥ 3 s · none · form≥4 · occ 2 · 28 d` | 8–26 Wo. (H-DUR) | OG 9 [A-02] *(S)* |
| 6 | `planche/full` · `planche` | Full Planche | gestreckte Linie, waagrecht ±15° | `planche hold ≥ 3 s · none · form≥4 · occ 2 · 28 d` | — | OG ≥ 10 [A-02] *(S)*; FIG C [A-05]; 3 s [A-09] |
| Pr 1 | `planche-press/straddle` · `planche-straddle-press` | Straddle Planche → Handstand | Arme gestreckt, kein Schwung (H-FORM) | `planche-straddle-press reps ≥ 1 · none · form≥4 · occ 2 · 28 d` | — | PDF Beginner 1–3 Wdh. [P-01 S. 2–3] |
| Pr 2 | `planche-press/full` · `planche-full-press` | Full Planche → Handstand | wie Pr 1 | `planche-full-press reps ≥ 1 · none · form≥4 · occ 2 · 28 d` | — | PDF Intermediate 1–3 Wdh. [P-02 S. 2] |
| Pr 3 | `planche-press/full-5` · `planche-full-press` | 5 Wdh. | wie Pr 1 | `planche-full-press reps ≥ 5 · none · form≥4 · occ 2 · 28 d` | — | PDF Advanced 3–8 Wdh. [P-03 S. 2] |
| PU 1–3 | `planche-push-up/tuck` · `/straddle` · `/full` | Planche-Liegestütze je Stufe | Hüfte bleibt auf Schulterhöhe, voller Weg (H-FORM) | `planche-push-up-<stufe> reps ≥ 5 · none · form≥4 · occ 2 · 28 d` | — | PDF 5–10 Wdh. [P-02 S. 3; P-03 S. 3] |
| (Übungen) | Kicks, «L-Sit to Tuck Planche», Negative, Band (`elastic`, `neck band`), `badform`-Halte | Volumen- und Technikübungen | — | Rolle `progression`, kein Level | — | [P-01 bis P-03] |

- **Voraussetzungen:** `planche/lean` ← `push-up/full` (`prerequisite`) und
  `wrist-conditioning/prep` (`recommended`) (H-PRE); `planche/tuck` ←
  `support-hold/parallel-bars` (`recommended`) (H-PRE); `planche-press/straddle`
  ← `handstand/free-10s` (`prerequisite`, der Press endet im Handstand; H-PRE)
  und `planche/straddle` (`prerequisite`).
- **Häufige Fehler:** gebeugte Arme, Hüfte zu hoch oder zu tief (Winkelabzug
  [A-04] *(S)*), fehlende Protraktion, zu wenig Vorlage (H-FAULT). Die PDFs üben
  bewusst `badform`-Halte [P-01 S. 2] — solche Sätze mit niedriger
  `form_quality` loggen; sie zählen dann nicht für den Unlock.
- **Carryover:** Pseudo-Planche-Liegestütz, HSPU (P-04 kombiniert Weighted Lean
  Planche mit Weighted HSPU [P-04 S. 1]), Maltese (Wide Planche als Vorstufe
  [P-02 S. 4]); Front Lever als Antagonist (H-PRE). Kraft-Vorübungen tragen nur
  teilweise: gute Korrelationen [A-22, A-23], aber abweichende
  Muskelaktivierung [A-25].
- **Equipment:** Boden, Parallettes (`pbar`, in den PDFs Standard für Halte und
  Presses), Ringe. FIG bewertet die Full Planche am Boden und an den Ringen
  gleich (C) [A-04 *(S)*, A-05]. Parallettes erlauben tiefere Liegestütze
  («DEEP» auf `pbar` [P-02 S. 3]) und entlasten die Handgelenk-Extension; Ringe
  fügen Instabilität hinzu (H-EQ). Handbreite und -rotation (`wide`, `supi`)
  variieren die Schwierigkeit; die Richtung ist ohne Quelle nicht belegbar
  (§9).

### 5.6 Maltese (`maltese`, Familie `push`) — PDF-Leiter validiert und erweitert

PDF-Leiter (`01_pdf_extract.md` §4.6): Lean Maltese mit Band (Beginner) → Lean
Maltese frei 5–15 s → Lean Maltese Elevator 2–5 Wdh. → Wide Planche Hold/Press →
Maltese Hold mit Band 3–8 s → Maltese Press 1–3 Wdh. mit Band → ohne Band →
am Boden [P-01 S. 2; P-02 S. 1, 4; P-03 S. 1, 4; P-04 S. 1].

**Validierung gegen externe Quellen:**

| Befund | Quelle | Folge für die Leiter |
|---|---|---|
| FIG: Maltese Boden C, Ringe D; Full Planche C | [A-07, A-05] | Maltese ≥ Full Planche; Ringe-Maltese als eigene, höhere Stufe |
| OG: Level 10–16 der Planche-Spalte enthalten Full Planche und Maltese-Varianten | [A-02] *(S)* | Maltese oberhalb der Full Planche einordnen |
| Kraft-Benchmarks: bis 94,1 % KG (Schwalbe, exzentrisch) in der Konditionierungsmessung; r 0,65–0,92 zwischen Konditionierungskraft und Elementleistung | [A-23] | extrem hohe Kraftanforderung; Realismus-Check bei Zielen |
| Schwalbe korreliert mit «Schwalbe in Rückenlage» (r = 0,71) und Bankdrücken (r = 0,71) | [A-22] | Bankdrücken/Schwalbe-Vorübung als `recommended`-Kraftbasis, nicht als harte Voraussetzung |
| Spezifische Vorübungen und Stützgeräte (Herdos, Gurte) reduzieren die Last bei Halteelementen | [A-21] | bestätigt Band-Assistenz als reguläres Werkzeug, wie in den PDFs |
| Der Maltese-**Halt** erscheint in den PDFs nur mit Band | `01_pdf_extract.md` §4.4 | ein unassistierter Maltese-Halt liegt jenseits der dokumentierten Programme → eigene Endstufe |

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `maltese/lean` · `maltese-lean` | Lean mit weit seitlich gestellten Händen | Arme gestreckt, Körper gerade, Schultern weit vor (H-FORM) | `maltese-lean hold ≥ 10 s · none · form≥4 · occ 2 · 28 d` | 8–26 Wo. (H-DUR) | PDF frei 5–15 s [P-02 S. 4; P-03 S. 4] |
| 2 | `maltese/lean-elevator` · `maltese-lean-elevator` | dynamische Lean-Variante mit Wdh. (Deutung §9) | wie 1 | `maltese-lean-elevator reps ≥ 3 · none · form≥4 · occ 2 · 28 d` | 8–26 Wo. (H-DUR) | PDF 2–5 Wdh. [P-02 S. 4; P-03 S. 4] |
| 3 | `maltese/wide-planche` · `planche-wide` | Planche mit weiter Handstellung (Deutung `wide` unsicher, §9) | waagrecht ±15° | `planche-wide hold ≥ 3 s · none · form≥4 · occ 2 · 28 d` | 12–52 Wo. (H-DUR) | PDF 3–6 s [P-02 S. 1] |
| 4 | `maltese/press` · `maltese-press` | Press aus der Maltese in den Handstand (unassistiert) | Arme gestreckt (H-FORM) | `maltese-press reps ≥ 1 · none · form≥4 · occ 2 · 28 d` | — | PDF Advanced 1–3 Wdh. ohne Band [P-03 S. 4] |
| 5 | `maltese/hold` · `maltese` | unassistierter Maltese-Halt (Boden/Parallettes) | Körper waagrecht auf Handhöhe, Arme gestreckt (H-FORM) | `maltese hold ≥ 3 s · none · form≥4 · occ 2 · 28 d` | — | FIG C Boden [A-07]; 3 s [A-09]; **Erweiterung** über PDF hinaus |
| 6 | `maltese/rings` · `maltese-rings` | Maltese an Ringen | wie 5 | `maltese-rings hold ≥ 3 s · none · form≥4 · occ 2 · 28 d` | — | FIG D [A-05, A-07]; **Erweiterung** |
| (Übungen) | `maltese-lean-band`, `maltese-hold-band`, `maltese-press-band`, `maltese-elevator`, `zanetti-vertical-band` | assistierte Volumen- und Technikübungen | — | Rolle `progression`; kein Level | — | [P-01 bis P-04] |

Die Reihenfolge von Stufe 4 (Press) vor 5 (Halt) folgt den PDFs (Press ohne
Band ab Advanced, Halt nie ohne Band); ob ein unassistierter Halt schwerer ist
als ein unassistierter Press, ist offen (Offene Fragen).

- **Voraussetzungen:** `maltese/lean` ← `planche/advanced-tuck`
  (`prerequisite`; die PDFs beginnen Lean Maltese auf der Stufe, die an der
  Straddle Planche arbeitet [P-01 S. 2]; H-PRE). `maltese/wide-planche` ←
  `planche/straddle` (`prerequisite`, H-PRE). `maltese/press` ←
  `planche-press/full` (`prerequisite`; in den PDFs erscheint der Maltese-Press
  mit Band erst mit dem Full-Planche-Press [P-02 S. 2, 4]; H-PRE).
- **Häufige Fehler:** Hände wandern nach innen (wird zur Planche), Hüfte
  knickt, Arme beugen (H-FAULT).
- **Carryover:** Planche (bidirektional), Iron Cross (weite Armhaltung, andere
  Richtung), Victorian (H-PRE).
- **Equipment:** Boden, `pbar`, `supi bar`, Ringe [P-02 bis P-04]. FIG wertet
  Ringe höher als Boden (D vs. C) [A-05, A-07]; beim Autor steht dagegen der
  Press **am Boden** an der Spitze [P-04 S. 1] → «Widersprüche».

### 5.7 Human Flag (`human-flag`, Familie `core`)

**Keine Quelle gefunden.** Die ganze Leiter ist Praxisheuristik und nur mit
Selbstbestätigung oder vorsichtigen Schwellen freizugeben.

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `human-flag/vertical` · `human-flag-vertical` | Körper senkrecht seitlich an der Stange, Arme gestreckt | untere Hand drückt, obere zieht; Körper gestreckt (H-FORM) | `human-flag-vertical hold ≥ 10 s · form≥4 · occ 2 · 28 d` (H-UNL) | 4–12 Wo. (H-DUR) | (H) |
| 2 | `human-flag/tuck` · `human-flag-tuck` | Tuck, Rumpf waagrecht | Arme gestreckt (H-FORM) | `human-flag-tuck hold ≥ 10 s · form≥4 · occ 2 · 28 d` | 4–12 Wo. (H-DUR) | (H) |
| 3 | `human-flag/straddle` · `human-flag-straddle` | gegrätscht, waagrecht | waagrecht ±15° (H-FORM) | `human-flag-straddle hold ≥ 3 s · form≥4 · occ 2 · 28 d` | 8–26 Wo. (H-DUR) | Halt 3 s [A-09] (WSWCF führt die Flag als Static-Element) |
| 4 | `human-flag/full` · `human-flag` | Beine geschlossen, waagrecht | waagrecht ±15° | `human-flag hold ≥ 3 s · form≥4 · occ 2 · 28 d` | — | [A-09] |

- **Voraussetzungen:** `pull-up/strict-8` und `support-hold/parallel-bars`
  (`recommended`) (H-PRE).
- **Häufige Fehler:** untere Hand nicht gestreckt, Rotation nach vorne, Hüfte
  hängt (H-FAULT).
- **Carryover:** gering zu anderen Statics; seitliche Rumpfkraft (H-PRE).
- **Equipment:** senkrechte Stange (Standard) vs. Sprossenwand/Gitter (Griffe
  frei wählbar) (H-EQ). Beide Seiten getrennt loggen (`unilateral: true`).
