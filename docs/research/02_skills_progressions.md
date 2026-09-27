# 02 — Skills & Progressionen

> Stream A der Phase-2-Recherche. Liefert die Progressionsleitern aller Skills,
> die der Planer als Knoten des Skill-Graphen braucht: Einstiegswurzeln unter
> dem PDF-«Beginner», Statics, Dynamics und Kraftskills. Für jede Stufe:
> Slug-Vorschlag, Formkriterien, ein Unlock-Kriterium in der bestehenden DSL
> und die typische Dauer bis zur nächsten Stufe. Dazu eine gemeinsame
> Schwierigkeitsskala, eine Kanten-Tabelle (Voraussetzung / Empfehlung /
> Alternative / Antagonist) und die Klärung der PDF-Kürzel.
>
> **Prüftiefe.** Studien wurden über Europe PMC/Crossref geprüft, wo Open Access
> im Volltext, sonst am Abstract (vermerkt in der Spalte «Titel» der
> Quellentabelle als «[VT]» bzw. «[Abs.]»). Der FIG Code of Points, die
> WSWCF-Regeln, die Overcoming-Gravity-Charts, die OG-Leseprobe und die
> Coaching-Seiten wurden direkt im Original gelesen. Nicht erreichbar waren
> Reddit (die r/bodyweightfitness-Routine wurde über einen GitHub-Spiegel
> gelesen, die Übungsseiten nicht) und die Kanäle von Daï-Long Huynh. Werte, die
> nur aus einem Such-Auszug stammen, tragen *(S)*.

**Legende**

| Kürzel | Bedeutung |
|---|---|
| `[A-xx]` | Quelle aus Stream A (Tabelle «Quellen»); `[A-xx S. n]` = Seite |
| `[P-0x S. n]` | PDF-Quelle (Daï-Long Huynh), siehe `01_pdf_extract.md` |
| *(S)* | Wert nur aus einem Such-Auszug; Original nicht eingesehen |
| **(H)** | **Praxisheuristik** — keine Quelle; Begründung im Code dahinter |
| (H-DUR) | Dauer-Heuristik: keine Quelle für genau diesen Schritt; Spanne nach PAR-A-45 aus dem OG-Level-Abstand, geeicht an den Coaching-Angaben in §3.6; mit App-Logs zu kalibrieren |
| (H-FORM) | Formkriterium = Definition der Position selbst; Winkeltoleranz nach §2.4 |
| (H-UNL) | Unlock-Schwelle nach der Vorlage in §2.4, ohne stufenspezifische Quelle |
| (H-PRE) | Kante aus Bewegungsverwandtschaft abgeleitet; keine Studie |
| (H-EQ) | Equipment-Effekt mechanisch begründet; nicht gemessen |
| (H-FAULT) | Häufiger Fehler = Umkehrung des Formkriteriums; keine Quelle |

DSL-Kurzschreibweise in den Stufentabellen:
`<exercise> <measure> ≥ <value> · <assistance> · form≥<n> · occ <n> · <within_days> d`.
Beispiel: `planche-tuck hold ≥ 10 s · none · form≥4 · occ 2 · 28 d` entspricht

```yaml
unlock_criteria:
  all:
    - { exercise: planche-tuck, measure: hold_seconds, op: ">=", value: 10,
        assistance: none, min_form_quality: 4, occurrences: 2, within_days: 28 }
```

## Kurzfassung

- **Leitern über schwerere Varianten sind als Überlastung belegt (Evidenz A):**
  4 Wochen progressives Liegestütz-Training (3×/Woche) steigerten das
  Bankdrück-1RM signifikant [A-01]; Band-Liegestütz und Bankdrücken bei gleicher
  6RM-Intensität brachten in 5 Wochen gleiche Kraftzuwächse [A-02].
- **Eine durchgehende Schwierigkeitsskala existiert:** Overcoming Gravity (OG)
  ordnet alle Leitern in 16 Level, abgeleitet aus dem FIG Code of Points;
  Beginner 1–5, Intermediate 6–9, Advanced 10–13, Elite 14–16 [A-30 S. 22,
  A-31, A-47]. Beispiele: Front Lever Tuck 4 → Full 8; Back Lever Tuck 3 → Full 7;
  Planche Tuck 5 → Straddle 8 → Full 11 (Boden/Barren) bzw. Full 14 an Ringen
  [A-31]. Die Hefesto-Ordinalskala übernimmt diese Level (§3).
- **FIG-Werte bestätigen die Ordnung:** an Ringen Front/Back Lever A, Straddle-
  Stützwaage A, Stützwaage (Planche) C, Kreuz C, Schwalbe (Maltese) D, Inverted
  Swallow E; am Boden Handstand A, Straddle Planche A, Planche und Schwalbe je C,
  Manna C; Mindesthaltedauer 2 s [A-29 S. 20, 28–29, 67].
- **Formqualität lässt sich an Wettkampfregeln koppeln:** FIG wertet
  Halteabweichungen > 5–20° als kleinen, > 20–45° als mittleren Fehler und
  erkennt > 45° nicht an [A-29 S. 19–20]; WSWCF akzeptiert höchstens 15° (Planche
  ±7,5°) und verlangt ≥ 3 s [A-33 S. 5]. Vorschlag: `form_quality` 4 = ≤ 15°;
  Unlock von Statics verlangt ≥ 4.
- **Wechselregeln (Evidenz C/D):** dynamisch 3×5–8, bei 3×8 sauber nächste
  Variante [A-44]; Grund-Isometrie Wechsel bei 3×30 s [A-44]; Planche-Stufen
  5×20 s (GMB) [A-35] bzw. 10 s in 3 Sätzen (TMA) [A-40].
- **Unlock-Vorlage (Vorschlag):** Zwischenstufen der Hebel-Statics ≥ 10 s,
  Endstufen ≥ 3 s (strengster Wettkampfstandard [A-33]), jeweils Form ≥ 4 und
  ≥ 2 Vorkommen; dynamisch ≈ 3×8 (§2.4).
- **Kraft ist notwendig, aber nicht hinreichend (Evidenz B):** Konditionierungs-
  kraft erklärt 76–85 % der Schwalbe-Leistung, 42–59 % der Stützwaage und
  38–48 % des Kreuz-Handstands [A-21]; Bankdrücken–Kreuz r = 0,41 mit
  Schwellenmuster («notwendig, nicht hinreichend») [A-23]; Stützwaage und
  Schwalbe aktivieren Muskeln anders als ihre Vorübungen [A-24, A-25].
  Kraftbaselines werden daher **empfohlene** Kanten, keine harten Voraussetzungen.
- **Kraft-Benchmarks existieren für Ringelemente und Gewichtsstufen:** Schwalbe
  braucht 63 % KG konzentrisch / 94 % KG exzentrisch in der
  Konditionierungsmessung, Stützwaage 60 % / 87 % [A-21]; 1RM «Schwalbe in
  Rückenlage» 73,4 % bzw. 67,4 % KG [A-20]. OG gibt gewichtete Klimmzüge und Dips
  als Vielfache des KG pro Level an (z. B. Klimmzug-Gesamtlast 1,18× KG =
  Level 4) [A-31]; Studierende schaffen im 1RM-Klimmzug 1,16× (Männer) bzw.
  0,73× KG (Frauen) [A-11].
- **Ringe machen Planche-Stufen schwerer:** OG setzt jede Planche-Stufe an Ringen
  1–3 Level höher als auf Boden/Barren [A-31]; FIG bewertet die Planche an beiden
  Geräten gleich (C) [A-29] (→ Widersprüche). Im Handstand an Ringen/Barren sinkt
  die Aktivität der Handgelenksbeuger (61 % → 44–46 %), die übrige Muskulatur
  arbeitet an Ringen mehr [A-16].
- **Handstand-Balance ist eine Handgelenksstrategie (Evidenz A/B):** in > 75 % der
  Zeit über die Handgelenke reguliert [A-15]; gute Balancen nutzen Handgelenk
  und Schulter, schwache die Hüfte [A-14]; Kopfhaltung und Sicht beeinflussen die
  Balance [A-03]. Handgelenk-Konditionierung ist deshalb eine Wurzel.
- **Kipping und strikt sind verschiedene Bewegungen:** Kipping vergrössert den
  maximalen Hüftwinkel im Klimmzug um 48,8° [A-10]; Ring-Muscle-ups aktivieren
  Trapez, Bizeps und Unterarm stärker als Stangen-Muscle-ups, die Autoren raten
  zur Stange als Einstieg [A-13]. OG setzt Kipping-MU auf Level 4, den strikten
  Stangen-MU auf Level 7 [A-31].
- **Dauer bis zur nächsten Stufe ist nur auf Coaching-Niveau belegt (C/D):**
  Front Lever je Stufe 2–4 Monate, gesamt 12–18 Monate ab 10 Klimmzügen [A-41];
  Planche kumuliert Tuck nach 2–6, Straddle nach 12–24, Full nach 24–36 Monaten
  [A-40]; Back Lever 6–12 Monate [A-42]. Elite-Turner steigern spezifische
  Ringkraft in 3–4 Wochen um 3,6–8,7 % [A-26, A-27]. Studien zu Lernzeiten
  fehlen; Hefesto sollte Dauern aus den eigenen Logs lernen.
- **DSL-Lücken:** `occurrences` zählt Sätze, nicht Tage; `min_load_kg` ist absolut
  (keine %-KG-Standards); exzentrische und assistierte Elemente zählen nie →
  Negativ- und Band-Stufen können keine automatisch freischaltbaren Levels sein
  (§2.5).
- **PDF-Kürzel:** `supi` ist mit hoher Wahrscheinlichkeit die supinierte
  (aussenrotierte) Handstellung — sie ist die Handstellung der Schwalbe an Ringen
  [A-21], und FIG nennt «hands turned out» und «slightly wide» als wertneutrale
  Varianten der Stützwaage [A-29 S. 62]. `Elevator` existiert als Elementname in
  OG (Level 17) ohne Definition [A-31]. `fake supi`, `neck band`, `Zanetti` und
  `Dead Planche` bleiben **unklar** (§9).

## 1. Methodik und Grenzen

| Punkt | Umsetzung |
|---|---|
| Suche | Europe PMC (Titel- und Volltextsuche) für Studien zu Ringelementen, Handstand, Liegestütz, Klimmzug, Rudern, Muscle-up; Direktabruf der Regelwerke (FIG MAG CoP 2025–2028, WSWCF, Calisthenics Cup), der OG-Charts (Google Sheet des Autors), der OG-Leseprobe (Kap. 1–3) und von Coaching-Seiten (GMB, The Movement Athlete, Spiegel der r/bodyweightfitness-Routine). Das Websuch-Kontingent war nach ~45 Suchen erschöpft; danach nur Europe PMC und Direktabrufe bekannter URLs. |
| Prüftiefe | Studienzahlen wurden, wo Open Access, im Volltext geprüft (z. B. Tabelle 3 in [A-21], Push-up-Tabelle in der Proceedings-Fassung von [A-05]); sonst am Abstract. Die Prüftiefe steht je Quelle in der Quellentabelle. |
| Evidenzstufen | A = SR/MA/RCT; B = Einzelstudien, FIG-Regelwerk als Expertenkonsens; C = Coaching-Bücher/-Seiten, Street-Workout-Regelwerk; D = Community-Wikis, App-Marketingseiten, Einzel-Event-Regeln. |
| r/bodyweightfitness | **D** (Community-Wiki), aber seit Jahren kuratiert und breit genutzt; nur für Reihenfolgen und Wechselregeln der Grundübungen verwendet. |
| Overcoming Gravity | **C**. Charts laut Autor aus dem FIG Code of Points konstruiert [A-47]; Level-Definition in Kap. 3 [A-30 S. 21–25]. Das Chart nennt **keine Haltezeiten oder Wiederholungen** pro Level. |
| The Movement Athlete (TMA) | **D**: kommerzielle App-Seiten mit Marketing-Aussagen (z. B. «research shows 10 seconds …» ohne Beleg [A-40]). Einzige gefundene Quelle mit stufenweisen Zeitangaben; nur als Praxisindiz verwendet. |
| Nicht verwendet | Sommer, *Building the Gymnastic Body* (2008): Existenz über den Goodreads-Eintrag geprüft (195 S.), Inhalt nicht zugänglich — nicht als Beleg verwendet. Frühere Such-Auszüge zu OG-Levels und FIG-Werten wurden durch die Originale ersetzt. |

## 2. Übergreifende Regeln für Stufen, Unlocks und Wechsel

### 2.1 Was eine Stufe ist

Eine Stufe (Level) ist ein **Leistungsnachweis**; eine Übung (Exercise) ist, was
geloggt wird (`CONTENT_AUTHORING.md`).

| Regel | Begründung |
|---|---|
| Stufen = Positionen/Varianten, die unassistiert, mit vollem Weg und nicht nur exzentrisch gezeigt werden. | Die Unlock-Engine zählt fehlgeschlagene, partielle und rein exzentrische Elemente nie; `assistance: none` ist Default (`internal/domain/progress/evaluate.go`). |
| Negative, Band-assistierte Varianten, Kicks und Übergänge sind **Übungen mit Rolle `progression`**, keine Stufen. | DSL (s. o.); in den PDFs reguläres Trainingsvolumen auf allen Niveaus (`01_pdf_extract.md` §4.4); Stützgeräte (Herdos, Gurte) sind auch im Turnen Werkzeug für Halteelemente [A-04]. |
| Kombinationen («L-Sit to Tuck Planche», «Hold to Press») sind ein `set_entry` mit mehreren Elementen und zählen als **ein** Vorkommen. | CLAUDE.md («one code path for sets»); `evaluate.go` zählt Set-Entries. |
| Bestehende Level-Slugs bleiben: `pull-up/strict-5`, `handstand/wall`, `front-lever/tuck`, `front-lever/advanced-tuck`. Neue Einstiegsstufen darunter werden als **eigene Wurzel-Skills** angelegt. | Level-Slugs sind permanent (`CONTENT_AUTHORING.md`); ob die `order` bestehender Levels verschoben werden darf, ist offen. |
| Die OG-Level bestimmen die **Ordinalskala**, nicht die Level-Liste eines Hefesto-Skills: Ein Hefesto-Level ist ein sinnvoller Meilenstein, auch wenn OG dazwischen weitere Übungen führt. | OG versteht die Charts als «approximate knowledge» der Lage auf dem Kontinuum [A-30 S. 22]. |

### 2.2 Wechselregeln aus der Literatur

| Regel | Wert | Quelle | Evidenz |
|---|---|---|---|
| Dynamische Grundübungen: Arbeitsbereich | 3 × 5–8 Wdh. der schwersten sauberen Variante; jede Einheit die Vorwerte schlagen | [A-44] | D |
| Wechsel | bei 3 × 8 mit guter Form; neue Variante mit 3 × 5 beginnen | [A-44] | D |
| Tempo | «10X0»: 1 s ab, ohne Pause, explosiv auf | [A-44] | D |
| Grund-Isometrie (Support, Tuck FL in der Row-Leiter) | Sätze à 10–30 s; Wechsel, wenn alle 3 Sätze 30 s erreichen | [A-44] | D |
| Planche-Stufen (GMB) | auf einer Stufe bis 5 × 20 s (2–3 min Pause), dann nächste Stufe | [A-35] | C |
| Planche-Stufen (TMA) | 10 s in 3 Sätzen mit gleichbleibender Form, ohne Gelenkschmerz | [A-40] | D |
| L-Sit (GMB) | einbeinig mind. 5 s pro Seite, dann voller L-Sit; Halte 5–30 s, 3–5 Sätze | [A-36] | C |
| Tuck → Straddle Planche (GMB) | zwei Übungen ≥ 3 Wochen, dann neu bewerten; die nächste Stufe laufend testen | [A-35] | C |
| Untrainierte Anfänger (OG) | zuerst höhere Wiederholungszahlen (Gewöhnung des Bindegewebes); trainierte Anfänger 5–15 Wdh. | [A-30 S. 24] | C |

*Ableitung (RR-Mindestdauer):* Von 3×5 bis 3×8 mit +1 Wdh. pro Satz und
Einheit braucht es mindestens 4 Einheiten (5→6→7→8) [A-44]; bei 3 Einheiten pro
Woche ≈ 1–1,5 Wochen, bei 2 Einheiten ≈ 1,5–2 Wochen (eigene Rechnung). Das ist
eine **Untergrenze**, keine typische Dauer.

### 2.3 Wettkampfstandards als Messlatte für «erreicht»

| Standard | FIG MAG 2025–2028 | WSWCF Freestyle | Calisthenics Cup |
|---|---|---|---|
| Mindesthaltedauer | 2 s ab vollständigem Stillstand [A-29 S. 20] | 3 s; länger bringt keine Punkte [A-33 S. 5] | 2 s ab Stillstand [A-34] |
| Winkeltoleranz Halteposition | > 5–20° klein (−0,1), > 20–45° mittel (−0,3), > 45° gross (−0,5) und nicht anerkannt [A-29 S. 19–22] | max. 15° im Hauptgelenk, Planche ±7,5° zur Horizontalen; sonst Punktabzug [A-33 S. 5] | bessere Form = höhere Bewertung; keine Gradangabe [A-34] |
| Armbeugung bei Halten/Pressen | 0–15° klein, > 15–30° mittel, > 30–45° gross, > 45° nicht anerkannt [A-29 S. 19] | gebeugte Ellbogen = Abzug [A-33 S. 5] | gebeugte Arme = schlechtere Bewertung [A-34] |
| Nicht als Halt gewertet | Positionen > 45° neben der Sollposition, z. B. Stützwaage mit > 45° Hüftbeugung [A-29 S. 61] | — (ein Tuck-Planche-Halt von 3 s erscheint als Kombinationsbeispiel) [A-33 S. 8] | Tuck/Adv-Tuck-Planche; Tuck/Adv-Tuck/45°-Straddle-Front-Lever; gehockter V-Sit; einbeiniger Back Lever [A-34] |
| Gerät | Werte teils geräteabhängig (§3.3) | — | p-Bars, Stange und Boden werden unterschiedlich bewertet [A-34] |

### 2.4 Generische Unlock-Vorlage (Vorschlag)

Unterschieden werden **Unlock** (Karte, «erreicht», nie zurückgenommen) und
**Trainingswechsel** (der Planer macht die nächste Stufe zum Haupttraining).

| Stufentyp | Unlock (DSL) | Trainingswechsel | Quellen |
|---|---|---|---|
| Dynamische Grundübung | `reps ≥ 8 · none · occ 3 · 7 d` (≈ 3×8) | 3×8 sauber | [A-44]; occ/within (H-UNL) |
| Grund-Isometrie (Support, Hollow, Plank, L-Sit-Vorstufen) | `hold ≥ 30 s · none · occ 3 · 7 d` | 3×30 s | [A-44]; occ/within (H-UNL) |
| Zwischenstufe Hebel-Static (Tuck, Adv Tuck, One-Leg) | `hold ≥ 10 s · none · form≥4 · occ 2 · 28 d` | 10 s in 3 Sätzen [A-40] bzw. 5×20 s [A-35] | 10 s [A-40]; Zwischenstufen zählen im Wettkampf teils nicht [A-34] |
| Endstufe (Straddle, Half-Lay, Full) | `hold ≥ 3 s · none · form≥4 · occ 2 · 28 d` | ≥ 10 s sauber, bevor die nächste Stufe Haupttraining wird | 3 s = strengster Wettkampfstandard [A-33]; 10 s [A-40] |
| Press/Kraftskill mit Wdh. | `reps ≥ 1 · none · form≥4 · occ 2 · 28 d`, danach Wdh.-Stufen (3, 5) | 3×3–5 sauber | (H-UNL); die PDFs dosieren Maximalpressen mit 1–3 Wdh. [P-01 S. 2–3; P-02 S. 2] |
| Gewichtete Stufe | `reps ≥ 1 · min_load_kg X · occ 2 · 28 d` | — | OG in Vielfachen des KG [A-31]; % KG in der DSL nicht ausdrückbar (§2.5) |

**`form_quality`-Zuordnung (Vorschlag):** 5 = ≤ 5° Abweichung (bei FIG
abzugsfrei [A-29 S. 20]); 4 = > 5–15° (innerhalb der WSWCF-Toleranz
[A-33 S. 5]); 3 = > 15–30°; 2 = > 30–45°; 1 = > 45° (bei FIG nicht anerkannt
[A-29 S. 19–20]). Die Grenzen 15/30/45° folgen der FIG-Skala für Armbeugung
[A-29 S. 19]. Statics verlangen für den Unlock ≥ 4. Für dynamische
Grundübungen wird `min_form_quality` weggelassen (H-UNL: ein gesetztes
`min_form_quality` schliesst unbewertete Elemente aus, `evaluate.go`).

### 2.5 Gefundene Grenzen der DSL

| Grenze | Folge | Vorschlag (Spezifikation, Phase 4) |
|---|---|---|
| `occurrences` zählt verschiedene Set-Entries, nicht Tage. | Zwei Sätze in einer Einheit erfüllen `occ 2`; «Wiederholung über Tage» ist nicht erzwingbar. | Feld `min_distinct_days`; bis dahin `occ` ≥ 2 mit `within_days`. |
| `min_load_kg` ist absolut. | OG-Standards in Vielfachen des KG [A-31] sind nicht global formulierbar. | Feld `min_load_pct_bw` (bezogen auf `workout_sessions.bodyweight_kg`). |
| `min_form_quality` schliesst Elemente ohne Formbewertung aus. | Wer Form nie bewertet, schaltet Statics nie frei. | UI fordert Formbewertung bei Unlock-relevanten Sätzen an. |
| Exzentrische und (bei `none`) assistierte Elemente zählen nie. | Negativ-/Band-Stufen können nicht automatisch freischalten. | Als `progression`-Übungen führen (§2.1). |
| Kein Merkmal «mit/ohne Schwung». | Kipping und strikt nur über getrennte Slugs trennbar. | Getrennte Exercise-Slugs [A-10, A-13]. |

## 3. Relative Schwierigkeit: eine gemeinsame Ordinalskala

### 3.1 Das OG-Levelsystem

OG ordnet Kraft- und Skillprogressionen in **16 Level**; ein Level soll über alle
Spalten hinweg ähnliche Leistungsfähigkeit zeigen. Die Level sind in vier Viertel
gruppiert (Basic, A, B, C — angelehnt an die FIG-Werte) und in Athletenklassen:
**Beginner 1–5, Intermediate 6–9, Advanced 10–13, Elite 14–16** [A-30 S. 22;
A-31]. Die Charts sind laut Autor aus dem FIG Code of Points konstruiert [A-47]
und geben eine «ungefähre» Einordnung [A-30 S. 22]. Der Körper macht auf
niedrigen Leveln schneller Fortschritte als auf hohen [A-30 S. 23]. Für die
Einordnung zählt das Können, nicht die Trainingsdauer [A-30 S. 23]. Das Buch
enthält keine Schwung-, Salto- oder Riesenfelgen-Elemente [A-47].

### 3.2 OG-Level der für Hefesto relevanten Spalten (Original-Chart [A-31])

| Spalte | Level → Übung |
|---|---|
| Planche (Barren/Boden) | 3 Frog Stand · 4 Frog Stand mit gestreckten Armen · 5 Tuck · 6 Adv Tuck · 8 Straddle · 9 Half-Lay/One-Leg · 11 Full · 12 Straddle Planche → Handstand (gestreckte Arme) |
| Planche (Ringe) | 4 Frog Stand · 5 Frog Stand mit gestreckten Armen · 6 Tuck · 8 Adv Tuck · 10 Straddle · 12 Half-Lay/One-Leg · 14 Full |
| Planche-Liegestütz (Barren/Boden) | 6 Tuck · 8 Adv Tuck · 10 Straddle · 12 Half-Lay/One-Leg · 14 Full |
| Front Lever | 4 Tuck · 5 Adv Tuck · 6 Straddle · 7 Half-Lay/One-Leg · 8 Full · 9 FL to Inverted |
| Front-Lever-Rows | 5 Tuck · 6 Adv Tuck · 8 Straddle · 10 Full |
| Back Lever | 1 German Hang · 2 Skin the Cat · 3 Tuck · 4 Adv Tuck · 5 Straddle · 6 Half-Lay/One-Leg · 7 Full · 8 BL Pullout |
| Handstand | 1–3 Wand-HS · 4–5 freistehend · 6 Vorstufen One-Arm-HS · 10 One-Arm-HS; Ringe: 5 Schulterstand · 6 HS mit Seilkontakt · 7 Ring-HS |
| Handstand-Liegestütz | 1 Pike Headstand-PU · 2 Box Headstand-PU · 3 Wand-Headstand-PU exzentrisch · 4 Wand-Headstand-PU · 5 Wand-HSPU · 6 freier Headstand-PU · 7 freier HSPU; Ringe: 7 breit · 8 mit Seilkontakt · 9 frei |
| Press in den Handstand (gestreckte Arme) | 5 Wand-Straddle-Press exzentrisch · 6 erhöhter Straddle-Stand-Press · 7 Straddle/Pike-Stand-Press · 8 L-Sit/Straddle-L → Straddle-Press · 9 L-Sit/Straddle-L → Pike-Press |
| L / Straddle-L / V / Manna | 1 Tuck L-Sit · 2 einbeinig gebeugt · 3 L-Sit · 4 Straddle-L · 5 L-Sit an Ringen (RTO) · 6 V-Sit 45° · 7 75° · 8 100° · 9 120° · 10 140° · 11 155° · 12 170° · 13 Manna |
| Klimmzüge | 1 Sprung-Klimmzüge · 2 exzentrisch · 3 Klimmzug · 4 L-Klimmzug · 5 Pullover |
| Ringe-Klimmzüge + One-Arm-Chin (OAC) | 4 L · 5 breit · 6 breit L · 7 Archer · 8 OAC exzentrisch · 9 OAC · 10 OAC + 15 lb (6,8 kg) · 11 OAC + 25 lb (11,3 kg) |
| Gewichtete Klimmzüge (Gesamtlast × KG) | 2 assistiert · 3 1,00 · 4 1,18 · 5 1,35 · 6 1,50 · 7 1,65 · 8 1,78 · 9 1,90 · 10 2,00 · 11 2,10 |
| Explosive Klimmzüge | 2 Kipping-Klimmzüge · 3 Klimmzug · 4 Kipping mit Klatschen · 5 Klatschen ohne Kipping |
| Rudern | 1 exzentrisch · 2 Ringrudern · 3 breit · 4 Archer · 5 Archer-in · 6 einarmig gegrätscht · 7 einarmig |
| Liegestütz | 1 Standard · 2 Diamond · 3 Ringe breit · 4 Ringe · 5 Ringe RTO · 6 RTO Archer · 7 RTO 40°-Pseudo-Planche · 8 60° · 9 RTO Maltese-Liegestütz · 10 Wand-Pseudo-Planche |
| Dips (Barren) | 1 Sprung-Dips · 2 exzentrisch · 3 Dips · 4 L-Dips · 5 45°-Dips |
| Ring-Dips | 1 Stützhalt · 2 Stütz RTO · 3 exzentrisch · 4 Ring-Dips · 5 L-Dips · 6 breit · 7 RTO 45° |
| Gewichtete Dips (Gesamtlast × KG) | 2 assistiert · 3 1,00 · 4 1,20 · 5 1,38 · 6 1,55 · 7 1,70 · 8 1,85 · 9 2,00 · 10 2,13 · 11 2,25 · danach Vermerk «Maltese (L17)» |
| Muscle-ups | 3 MU exzentrisch · 4 Kipping-MU · 5 Muscle-up · 6 breit/ohne False Grip · 7 strikter Stangen-MU · 8 L-Sit-MU · 9 einarmig · … · Vermerk «(L17) Elevator» |
| Flagge | 5 Tuck · 6 Adv Tuck · 7 Straddle · 8 Full |
| Kniebeuge / Pistol | 1 parallel · 2 tief · 3 seitlich verlagert · 4 Pistol · 5 Pistol mit 1,2× KG · 6 1,35× · 7 1,5× · 8 1,65× · 9 1,8× · 10 1,9× · 11 2,0× |
| Ab Wheel / Plank | 2 Plank 25 s · 3 Plank 60 s · 4 Plank 1 Arm/1 Bein · 5 Ab Wheel kniend · 8 Ab Wheel voll |
| Ringe «Full Statics» | 5 RTO L-Sit · 6 RTO Straddle-L · 7 Back Lever · 8 Front Lever · 9 V-Sit 90° · 10 Iron Cross / Straddle Planche · 14 Full Planche · 16 Inverted Cross |

Die Zusatzspalten rechts im Chart (u. a. Victorian-Varianten, Dragon Flag) sind
laut Autor **nicht im Buch** und werden hier nicht verwendet [A-31]. Das Chart
nennt **keine Haltezeiten oder Wiederholungen** pro Level.

### 3.3 FIG-Elementwerte (Original [A-29])

In den Wertetabellen entspricht die Position in der Zeile dem Wert
(Nr. 1/7/13/19/25 = A, 2/8/14/20/26 = B, 3/9/15/21/27 = C, 4/10/16/22 = D,
5/11/17/23 = E); die Werte unten sind so aus den Tabellen gelesen.

| Element | Gerät | Wert | Fundstelle |
|---|---|---|---|
| L-Sit oder Straddle-L-Sit (2 s) | Ringe | A | [A-29 S. 67, Nr. 1] |
| V-Sit (2 s) | Ringe | B | [A-29 S. 67, Nr. 2] |
| Hangwaage rücklings = Back Lever (2 s) | Ringe | A | [A-29 S. 67, Nr. 7] |
| Hangwaage vorlings = Front Lever (2 s) | Ringe | A | [A-29 S. 67, Nr. 13] |
| Stützwaage gegrätscht = Straddle Planche (2 s) | Ringe | A | [A-29 S. 67, Nr. 19] |
| Stützwaage = Planche (2 s) | Ringe | C | [A-29 S. 67, Nr. 9] |
| Kreuz / V-Kreuz (2 s) | Ringe | C | [A-29 S. 67, Nr. 15; S. 61] |
| Kreuz-Handstand (Inverted Cross) (2 s) | Ringe | D | [A-29 S. 67, Nr. 4] |
| Schwalbe = Maltese auf Ringhöhe (2 s) | Ringe | D | [A-29 S. 67, Nr. 10] |
| Inverted Swallow (Victorian) (2 s) | Ringe | E | [A-29 S. 67, Nr. 11] |
| Handstand (2 s) | Boden | A | [A-29 S. 28, Nr. 19] |
| V-Sit (2 s) | Boden | B | [A-29 S. 28, Nr. 2] |
| Manna (V-Sit mit waagrechten Beinen) (2 s) | Boden | C | [A-29 S. 28, Nr. 3] |
| Planche gegrätscht (2 s) | Boden | A | [A-29 S. 29, Nr. 25] |
| Planche (2 s) / Schwalbe (2 s) | Boden | C | [A-29 S. 29, Nr. 27] |
| Aus gegrätschter Planche Press in den Handstand | Boden | B | [A-29 S. 29, Nr. 32] |
| Aus Planche Press in den Handstand | Boden | D | [A-29 S. 29, Nr. 34] |
| Manna (2 s) und Press in den Handstand | Boden | D | [A-29 S. 28, Nr. 10] |

**Konsistenz OG ↔ FIG (eigene Ableitung):** Die OG-Viertel (Basic 1–4, A 5–8,
B 9–12, C 13–16) [A-30 S. 22; A-31] treffen die FIG-Werte für Front Lever (A,
OG 8), Back Lever (A, OG 7), Straddle Planche (A, OG 8), Planche an Ringen (C,
OG 14), Manna (C, OG 13) und das Kreuz nach dem alten Code (B, OG 10; seit 2025
C [A-29]). Abweichend: Die Planche am Boden ist bei FIG C, bei OG Level 11
(B-Viertel), und die Maltese am Boden ist bei FIG gleichwertig mit der Planche
(C), bei OG erst Level 17 (→ «Widersprüche»).

### 3.4 Hefesto-Ordinalskala (Vorschlag)

Ordinal = OG-Level [A-31]; Elemente ohne OG-Level über FIG-Wert und OG-Viertel
(H). `difficulty_tier` (1–10, Schema-Feld der Skills) = ⌈Ordinal × 10 / 16⌉
(PAR-A-22, H: lineare Abbildung der OG-Skala auf das bestehende Schemafeld).
Einstiegswurzeln unter OG-Level 1 erhalten Ordinal 0 und Tier 1.

| Ordinal | Tier | Stufen (Beispiele) | Status |
|---|---|---|---|
| 0 | 1 | Wand-/Schräg-Liegestütz, Dead Hang, Scapula-Pull, Handgelenk-Vorbereitung | (H) unter OG 1 |
| 1 | 1 | Wand-HS; German Hang; Pike Headstand-PU; Tuck L-Sit; Standard-Liegestütz; Sprung-Klimmzug; Stützhalt | [A-31] |
| 2 | 2 | Skin the Cat; Box Headstand-PU; Diamond-Liegestütz; exzentrischer Klimmzug/Dip; Plank 25 s | [A-31] |
| 3 | 2 | Frog Stand; Tuck BL; L-Sit; Klimmzug; Barren-Dip; Plank 60 s; MU exzentrisch | [A-31] |
| 4 | 3 | Tuck FL; Adv Tuck BL; Wand-Headstand-PU; Straddle-L; Pistol; Kipping-MU; Ring-Dip; freistehender HS | [A-31] |
| 5 | 4 | Tuck Planche; Adv Tuck FL; Straddle BL; Wand-HSPU; RTO L-Sit; Muscle-up; Tuck-Flagge; freistehender HS | [A-31] |
| 6 | 4 | Adv Tuck Planche; Straddle FL; Half-Lay BL; freier Headstand-PU; V-Sit 45°; Adv-Tuck-Flagge | [A-31] |
| 7 | 5 | Half-Lay/One-Leg FL; Full BL (FIG A); Archer-Klimmzug (Ringe); freier HSPU; strikter Stangen-MU; Straddle-Flagge; Ring-HS | [A-31]; FIG [A-29] |
| 8 | 5 | Straddle Planche (FIG A); Full FL (FIG A); OAC exzentrisch; L-Sit-MU; Full Flagge | [A-31]; FIG [A-29] |
| 9 | 6 | Half-Lay/One-Leg Planche; One-Arm-Chin; V-Sit 120° | [A-31] |
| 10 | 7 | Iron Cross / Straddle Planche an Ringen; OAC + 6,8 kg | [A-31] |
| 11 | 7 | Full Planche (Boden/Barren) | [A-31] |
| 12 | 8 | Half-Lay Planche an Ringen; Straddle Planche → Handstand mit gestreckten Armen | [A-31] |
| 13 | 9 | Manna (FIG C) | [A-31]; FIG [A-29] |
| 14 | 9 | Full Planche an Ringen (FIG C); Full-Planche-Liegestütz | [A-31]; FIG [A-29] |
| 16 | 10 | Inverted Cross an Ringen | [A-31] |
| 17 | 10 | Maltese (OG-Vermerk «L17»); an Ringen FIG D | [A-31]; FIG [A-29] |
| > 17 | 10 | Victorian (FIG E) | (H) über FIG-Wert [A-29] |

### 3.5 Relative Last innerhalb der Liegestütz-Leiter (Evidenz B)

| Variante | Spitzen-GRF in % KG [A-05] | relativ zum Standard | statisch (oben / unten) |
|---|---|---|---|
| Hände 60,96 cm erhöht | 41 % (0,41 ± 0,06) | 0,64 | — |
| Knie-Liegestütz | 49 % (0,49 ± 0,05) | 0,77 | 53,6 % / 61,8 % [A-06] *(S)* |
| Hände 30,48 cm erhöht | 55 % (0,55 ± 0,05) | 0,86 | — |
| Standard | 64 % (0,64 ± 0,04) | 1,00 | 69,2 % / 75,0 % [A-06] *(S)* |
| Füsse 30,48 cm erhöht | 70 % (0,70 ± 0,02) | 1,09 | — |
| Füsse 60,96 cm erhöht | 74 % (0,74 ± 0,02) | 1,16 | — |

Werte aus Tabelle 1 der Proceedings-Volltextfassung von [A-05]; Relativwerte
eigene Rechnung. In der statischen Messung tragen die Arme unten mehr Last als
oben, beim Knie-Liegestütz ist die Änderung grösser [A-06]. Liegestütz-
Progressionen mit erhöhten Füssen steigern die Aktivität von Serratus anterior
und oberem Trapez [A-07].

### 3.6 Typische Dauer bis zur nächsten Stufe — was belegt ist

**Es gibt keine Studie zu Lernzeiten calisthenischer Skills.** Belegt sind nur
(a) Coaching-Angaben, (b) Trainingsstudien an Elite-Turnern und (c) die
qualitative OG-Aussage, dass niedrige Level schneller fallen.

| Skill / Schritt | Angabe | Quelle | Evidenz |
|---|---|---|---|
| Planche kumuliert ab Start | Lean 0–2 Mo · Tuck 2–6 Mo · Adv Tuck 6–12 Mo · Straddle 12–24 Mo · Full 24–36 Mo; «erste Tuck Planche meist nach 3–6 Monaten, Full nach 1–3 Jahren» | [A-40] | D |
| Front Lever je Stufe | Fundament → Tuck 3–4 Mo · Tuck → Adv Tuck 2–3 Mo · Adv Tuck → Straddle 2–3 Mo · Straddle → Full 3–4 Mo · Full 3–6 Mo | [A-41] | D |
| Front Lever gesamt | 18–24 Mo (< 5 Klimmzüge), 12–18 Mo (≥ 10 Klimmzüge), 6–12 Mo (ab Tuck FL) | [A-41] | D |
| Back Lever | Vorbereitung 4–8 Wo · Pike/Straddle 8–12 Wo · Full 12–24 Wo; gesamt 6–12 Mo; jede Stufe 4–8 Wo | [A-42] | D |
| Erster Ring-Dip | 4–12 Wo (Ø 6–8) ab 30 s Ring-Stütz und 15 Barren-Dips | [A-43] | D |
| Erster Klimmzug (Dead Hang → Scapula → Negative → assistiert → voll) | 2–6 Monate | [A-46] | D |
| Allgemein pro Variante | «manche Varianten 2 Wochen, manche 2 Monate» | [A-46] | D |
| Klimmzug-Aufbauprogramm | 4 Wochen à 3 Tage können zu kurz oder zu lang sein | [A-38] | C |
| Freistehender Handstand | «viele Monate, manchmal Jahre» | [A-37] | C |
| Elite-Turner, spezifische Ringkraft | +4,1 % (Schwalbe) / +3,6 % (Stützwaage) nach 4 Wochen [A-26]; +8,7 % / +8,3 % nach 3 Wochen [A-27] | [A-26, A-27] | B |
| Entwicklung allgemein | niedrige Level schneller als hohe; die stärksten Turner trainieren 10–15 Jahre und mehr | [A-30 S. 23] | C |

*Ableitung für den Planer:* Coaching-Angaben für einen Stufenschritt im mittleren
Bereich (OG 4–8) liegen bei **2–4 Monaten** [A-41, A-42], für die Planche-
Endstufen bei **12 Monaten und mehr** [A-40]. Die Elite-Daten zeigen, dass
spezifische Kraft selbst unter optimalen Bedingungen nur um einige Prozent pro
Monat wächst [A-26, A-27]. Daraus die Heuristik PAR-A-45 (Wochen pro OG-Level-
Schritt nach Band); sie erzeugt alle mit (H-DUR) markierten Dauern.

## 4. Wurzeln: Grundlagen unter dem PDF-«Beginner»

Die PDFs setzen eine solide Tuck-Planche voraus (`01_pdf_extract.md` §4.7). Für
echte Anfänger braucht der Graph Wurzeln. Grundgerüst: die RR mit Pull-up-,
Dip-, Row- und Push-up-Leiter und ihrem Aufwärmblock [A-44] sowie die unteren
OG-Level [A-31]. OG empfiehlt untrainierten Anfängern zuerst höhere
Wiederholungszahlen und eine ausgewogene Ganzkörperroutine [A-30 S. 24]; die
meisten Anfängerprogramme sind Ganzkörper 3×/Woche [A-30 S. 24], die RR ebenso
[A-44].

### 4.1 Handgelenk-Konditionierung (`wrist-conditioning`, Familie `mobility`)

Warum Wurzel: Im Handstand wird die Balance überwiegend über die Handgelenke
reguliert [A-15, A-03]; am Boden sind die Handgelenksbeuger dabei stärker aktiv
als an Barren oder Ringen [A-16]. Die RR beginnt jede Einheit mit einer
Handgelenk-Vorbereitung (GMB Wrist Prep, 10+ Wdh.) [A-44]; GMB empfiehlt die
Routine 2–3×/Woche vor Oberkörper-Einheiten [A-39]. TMA nennt «3 Minuten
Handgelenk-Vorbereitung ohne Schmerz» als Planche-Voraussetzung [A-40].

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `wrist-conditioning/prep` · `wrist-prep-routine` | Handgelenk-Routine im Vierfüsslerstand (Schütteln, Kreisen, Handflächen-/Handrücken-Varianten, 5–10 Wdh., Halte 10–20 s) | schmerzfrei, kontrolliert (H-FORM) | Selbstbestätigung oder `wrist-prep-routine reps ≥ 10 · occ 3 · 14 d` (H-UNL) | 1–2 Wo. (H-DUR) | Übungen und Dosis [A-39]; 10+ Wdh. [A-44] |
| 2 | `wrist-conditioning/loaded-extension` · `quadruped-wrist-rock` | belastete Extension: Schultern im Vierfüsslerstand über/vor die Hände | Ellbogenbeugen nach vorn, Finger gespreizt (H-FORM) | `quadruped-wrist-rock reps ≥ 15 · occ 3 · 14 d` (H-UNL) | 1–3 Wo. (H-DUR) | Pulsieren mit Ellbogenbeugen nach vorn [A-39] |
| 3 | `wrist-conditioning/plank` · `plank-hold` | Liegestützposition, Arme gestreckt | Körperlinie gerade, Schultern nicht eingesunken (H-FORM) | `plank-hold hold ≥ 30 s · occ 3 · 7 d` | → `planche/lean` | 30 s Plank = Startbedingung für Planche-Leans [A-35]; OG: Plank 25 s = Level 2, 60 s = Level 3 [A-31] |

- **Voraussetzungen:** keine (Wurzel).
- **Häufige Fehler:** Schmerz «durchtrainieren»; Last zu früh auf volle
  Extension (GMB rät zu kurzen Sätzen, geringem Druck, Pausen [A-39]).
- **Carryover:** Handstand, Planche-Lean, Pseudo-Planche-Liegestütz (H-PRE).
- **Equipment:** Parallettes/Griffe reduzieren den Handgelenkwinkel (TMA [A-40];
  EMG-Hinweis [A-16]). Die PDFs üben Leans bewusst am Boden in `supi`-Stellung
  [P-01 S. 1, 3].
- **Kante:** Stufe 1 als `recommended` (nicht `prerequisite`) für
  `handstand/wall` und `planche/lean` (H-PRE: Anfänger nicht blockieren).

### 4.2 Hang- und Scapula-Grundlagen (`hang-foundation`, Familie `pull`)

Die RR-Pull-up-Leiter beginnt mit Scapula-Pulls; Arch Hangs kommen ins Aufwärmen,
sobald negative Klimmzüge erreicht sind [A-44]. GMB baut den ersten Klimmzug über
«Pulling Prep» (3–5 × 5–10, gestreckte Ellbogen, Bewegung nur im Schultergürtel),
Rudern und Negative (8 × 1–3) auf [A-38]. OG: Sprung-Klimmzug Level 1,
exzentrisch Level 2, Klimmzug Level 3 [A-31]. Weil `pull-up` bereits mit
`strict-5` beginnt, werden diese Stufen als eigener Wurzel-Skill vorgeschlagen.

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `hang-foundation/dead-hang` · `dead-hang` | passiver Hang | Arme gestreckt, ohne Bodenkontakt (H-FORM) | `dead-hang hold ≥ 30 s · occ 3 · 7 d` | 1–2 Wo. (H-DUR) | 30 s Isometrie-Regel [A-44]; 30 s Dead Hang als FL-Voraussetzung [A-41] |
| 2 | `hang-foundation/scapular-pull` · `scapular-pull-up` (existiert) | Schulterblätter aus dem Hang nach unten ziehen | Ellbogen gestreckt, Bewegung nur im Schultergürtel [A-38] | `scapular-pull-up reps ≥ 8 · occ 3 · 7 d` | 1–2 Wo. (H-DUR) | [A-44, A-38] |
| 3 | `hang-foundation/arch-hang` · `arch-hang` | Scapula-Zug plus Brust Richtung Stange | Arme gestreckt (H-FORM) | `arch-hang reps ≥ 8 · occ 3 · 7 d` | 2–8 Wo. bis `pull-up/strict-5` (H-DUR) | RR-Aufwärmen 10 Wdh. [A-44] |
| (Übungen) | `pull-up-jump`, `pull-up-negative`, `row-ring` (Rolle `progression` bei `pull-up/strict-5`) | Sprung-, Negativ-Klimmzug, Rudern | kontrolliert | kein Level (§2.5) | — | OG 1–2 [A-31]; GMB [A-38] |

- **Voraussetzungen:** keine.
- **Häufige Fehler:** Scapula-Pull mit gebeugten Ellbogen; Schultern hochgezogen
  im Hang (H-FAULT).
- **Carryover:** Klimmzug, Front Lever, Muscle-up (H-PRE).
- **Equipment:** Ringe (freie Rotation) oder Stange (H-EQ).

### 4.3 Körperspannung: Hollow Body, Arch, Plank (`hollow-body`, `arch-body`, Familie `core`)

OG führt Plank 25 s (Level 2) und 60 s (Level 3) als Einstieg der Ab-Wheel-Leiter
[A-31]. TMA nennt einen 60-s-Hollow-Hold als Voraussetzung für Front Lever und
Planche [A-41, A-40]; die RR nutzt Deadbugs (30 s) im Aufwärmen [A-44].

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `hollow-body/tuck` · `hollow-hold-tuck` | Rückenlage, Knie angezogen, Schultern vom Boden | Lendenwirbelsäule am Boden (H-FORM) | `hollow-hold-tuck hold ≥ 30 s · occ 3 · 7 d` | 1–2 Wo. (H-DUR) | Regel [A-44]; Stufe (H) |
| 2 | `hollow-body/full` · `hollow-hold` | Arme über Kopf, Beine gestreckt knapp über dem Boden | Lendenwirbelsäule am Boden [A-41] | `hollow-hold hold ≥ 60 s · form≥4 · occ 2 · 14 d` | 2–6 Wo. (H-DUR) | 60 s [A-41, A-40] |
| 1 | `arch-body/hold` · `arch-hold` | Bauchlage, Arme/Beine angehoben | Gesäss angespannt, Nacken neutral (H-FORM) | `arch-hold hold ≥ 30 s · occ 3 · 7 d` | 1–2 Wo. (H-DUR) | Regel [A-44]; Stufe (H) |
| 1 | `plank/hold-60` · `plank-hold` | Unterarm- oder Liegestütz-Plank | Körperlinie gerade (H-FORM) | `plank-hold hold ≥ 60 s · occ 2 · 14 d` | — | OG Level 3 [A-31] |

- **Voraussetzungen:** keine.
- **Häufige Fehler:** Hohlkreuz im Hollow; Nacken überstreckt im Arch (H-FAULT).
- **Carryover:** Hollow → Front Lever, Planche, Handstand-Linie (TMA nennt Hollow
  als Voraussetzung für FL und Planche [A-41, A-40]); Arch → Back Lever (H-PRE).

### 4.4 Stützhalte (`support-hold`, Familie `push`)

OG: Stützhalt = Ring-Dip-Leiter Level 1, Stütz RTO = Level 2 [A-31]. Die RR
beginnt die Dip-Leiter mit dem Barren-Stütz und nimmt 30 s Stützhalt ins
Aufwärmen, sobald negative Dips erreicht sind [A-44]. TMA verlangt 30 s
Ring-Stütz vor dem ersten Ring-Dip [A-43].

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `support-hold/parallel-bars` · `support-hold-pb` | Stütz auf Barren/Dip-Griffen | Ellbogen gestreckt, Schultern tief (H-FORM) | `support-hold-pb hold ≥ 30 s · occ 3 · 7 d` | 1–2 Wo. (H-DUR) | [A-44] |
| 2 | `support-hold/rings` · `support-hold-rings` | Stütz an Ringen | wie 1, Ringe nah am Körper (H-FORM) | `support-hold-rings hold ≥ 30 s · occ 3 · 7 d` | 2–4 Wo. (H-DUR) | 30 s [A-43]; OG 1 [A-31] |
| 3 | `support-hold/rings-turned-out` · `support-hold-rings-rto` | Ringe nach aussen gedreht (RTO) | Ellbogenbeuge zeigt nach vorn (H-FORM) | `support-hold-rings-rto hold ≥ 30 s · form≥4 · occ 3 · 7 d` | — | OG 2 [A-31] |

- **Häufige Fehler:** Schultern hochgezogen, Ellbogen gebeugt, Ringe driften weg
  (H-FAULT).
- **Carryover:** Dips, Muscle-up-Endposition, L-Sit, Planche an Ringen (H-PRE).
- **Equipment:** Barren (stabil) < Ringe < Ringe RTO (OG-Level 1 → 2 [A-31]).

### 4.5 Liegestütz (`push-up`, Familie `push`)

Reihenfolge RR: Wand (vertikal) → Schräg → Standard → Diamond → Pseudo-Planche
[A-44; Leiter bestätigt in A-45, A-46]; OG: Standard 1, Diamond 2, Ringe breit 3,
Ringe 4, RTO 5, RTO Archer 6, RTO-Pseudo-Planche 40°/60° 7/8 [A-31].
Lastanteile §3.5 [A-05, A-06]. Progressive Varianten steigern die Kraft
(**Evidenz A**) [A-01, A-02].

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `push-up/wall` · `push-up-wall` | Hände an der Wand | Körper gerade, voller Weg (H-FORM) | `push-up-wall reps ≥ 8 · occ 3 · 7 d` | ≥ 4 Einheiten; 1–2 Wo. (H-DUR) | [A-44, A-45] |
| 2 | `push-up/incline` · `push-up-incline` | Hände erhöht | wie 1; Last 41–55 % KG je nach Höhe [A-05] | `push-up-incline reps ≥ 8 · occ 3 · 7 d` | 1–3 Wo. (H-DUR) | [A-44, A-05] |
| 2a | Alternative `push-up-knee` (Übung) | Knie am Boden | 49 % KG [A-05] | kein Level | — | [A-05, A-06] |
| 3 | `push-up/full` · `push-up` | Standard-Liegestütz | Körperlinie ohne Durchhängen; Brust nahe Boden; oben gestreckt (H-FORM) | `push-up reps ≥ 8 · occ 3 · 7 d` | 1–4 Wo. (H-DUR) | [A-44]; OG 1 [A-31] |
| 4 | `push-up/diamond` · `push-up-diamond` | Hände eng | wie 3 (H-FORM) | `push-up-diamond reps ≥ 8 · occ 3 · 7 d` | 2–8 Wo. (H-DUR) | [A-44]; OG 2 [A-31] |
| 5 | `push-up/rings` · `push-up-rings` | Ring-Liegestütz | Ringe nah, oben gestreckt (H-FORM) | `push-up-rings reps ≥ 8 · occ 3 · 7 d` | 4–16 Wo. (H-DUR) | OG 4 [A-31] |
| 6 | `push-up/pseudo-planche` · `pseudo-planche-push-up` | Hände auf Hüfthöhe, Schultern vor den Händen | Protraktion, Arme oben gestreckt, Schultern bleiben vor den Händen (H-FORM) | `pseudo-planche-push-up reps ≥ 8 · form≥4 · occ 3 · 7 d` | — | [A-44]; OG 7–8 (Ringe RTO) [A-31]; PDF-Zubringer 5–15 Wdh. [P-01 S. 1; P-03 S. 3] |
| (Alt.) | `push-up-decline` (Übung) | Füsse erhöht | 70–74 % KG [A-05] | Überlastung zu 3/4 | — | [A-05] |

- **Voraussetzungen:** keine; `wrist-conditioning/prep` empfohlen (H-PRE). TMA
  nennt 3 × 20 saubere Liegestütze als Planche-Voraussetzung [A-40].
- **Häufige Fehler:** Hüfte hängt oder knickt, halber Weg, Schulterblätter
  kollabieren oben (H-FAULT).
- **Carryover:** Dip, Planche-Lean, Pseudo-Planche → Planche [P-01 bis P-03];
  Bankdrück-Kraft [A-01, A-02]; Handstand/HSPU-Training überträgt auf
  Liegestütz und Dips [A-30 S. 21].
- **Equipment:** Erhöhung der Hände senkt, der Füsse erhöht die Last [A-05];
  Ringe +3 OG-Level gegenüber Boden (Standard 1 vs. Ringe 4) [A-31].

### 4.6 Dip (`dip`, Familie `push`)

RR: Barren-Stütz → negative Dips → Barren-Dips → Ring-Dips [A-44; A-45 *(S)*];
OG: Barren-Dips 3, L-Dips 4, 45°-Dips 5; Ring-Dips 4, Ring-L-Dips 5, breite
Ring-Dips 6, RTO 45° 7 [A-31].

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| (Vorst.) | `support-hold/parallel-bars` (§4.4) | Stütz | — | §4.4 | — | [A-44] |
| (Übung) | `dip-negative` (Rolle `progression`) | Absenken | kontrolliert | kein Level | — | OG 2 [A-31] |
| 1 | `dip/parallel-bars` · `dip-pb` | Barren-Dip | oben gestreckt, unten Oberarm mind. parallel, Schultern nicht hochgezogen (H-FORM) | `dip-pb reps ≥ 8 · occ 3 · 7 d` | 4–12 Wo. bis Ring-Dip [A-43] | OG 3 [A-31] |
| 2 | `dip/rings` · `dip-rings` | Ring-Dip | wie 1, Ringe nah, oben stabiler Stütz (H-FORM) | `dip-rings reps ≥ 8 · occ 3 · 7 d` | 4–13 Wo. je Folgelevel (H-DUR) | OG 4 [A-31]; TMA-Standards: 1–5 Wdh. Anfänger, 6–15 Fortgeschritten [A-43] |
| 3 | `dip/rings-l-sit` · `dip-rings-l` | Ring-Dip im L-Sit | Beine waagrecht | `dip-rings-l reps ≥ 5 · occ 2 · 28 d` (H-UNL) | — | OG 5 [A-31] |
| (weiter) | Weighted Dips §7.4 | | | | | [A-31] |

- **Voraussetzungen:** `dip/parallel-bars` ← `support-hold/parallel-bars`
  (`prerequisite`) [A-44]; `dip/rings` ← `support-hold/rings` (`prerequisite`)
  und `dip-pb` 15 Wdh. (`recommended`) [A-43].
- **Häufige Fehler:** zu flach, Schultern rollen nach vorn/oben, Schwung aus den
  Beinen (H-FAULT).
- **Carryover:** Muscle-up-Stützphase, HSPU, Planche-Liegestütz (H-PRE).
- **Equipment:** Barren (Level 3) < Ringe (Level 4) [A-31].

### 4.7 Australian Row / Rudern (`row`, Familie `pull`)

RR: vertikal → schräg → horizontal → breit → Archer [A-44; A-45 *(S)*]; die RR
führt den Tuck Front Lever als Isometrie-Stufe der Row-Leiter [A-44]. OG:
exzentrisch 1, Ringrudern 2, breit 3, Archer 4, Archer-in 5, einarmig 7 [A-31].
Beim Inverted Row aktivieren Latissimus, Bizeps, unterer Trapez und hinterer
Deltamuskel über 61 % MVIC, unabhängig von Ober- oder Untergriff und ein- oder
beidbeinigem Stand [A-12].

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `row/vertical` · `row-vertical` | fast aufrecht | Körper gerade, Brust zu den Griffen (H-FORM) | `row-vertical reps ≥ 8 · occ 3 · 7 d` | 1–2 Wo. (H-DUR) | [A-44, A-45] |
| 2 | `row/incline` · `row-incline` | schräg | wie 1 | `row-incline reps ≥ 8 · occ 3 · 7 d` | 1–3 Wo. (H-DUR) | [A-44] |
| 3 | `row/horizontal` · `row-horizontal` | waagrecht (Australian Row) | Arme unten gestreckt, Brust an Griff/Stange (H-FORM) | `row-horizontal reps ≥ 8 · occ 3 · 7 d` | 2–8 Wo. (H-DUR) | [A-44]; Ringrudern OG 2 [A-31] |
| 4 | `row/wide` · `row-wide` | breit | wie 3 | `row-wide reps ≥ 8 · occ 3 · 7 d` | 2–8 Wo. (H-DUR) | [A-44]; OG 3 [A-31] |
| 5 | `row/archer` · `row-archer` | ein Arm zieht, einer bleibt gestreckt | Rumpf rotiert nicht (H-FORM) | `row-archer reps ≥ 8 · occ 3 · 7 d` (je Seite) | — | [A-44]; OG 4 [A-31] |

- **Voraussetzungen:** keine (Wurzel).
- **Häufige Fehler:** Hüfte hängt, Kinn statt Brust zur Stange, halber Weg
  (H-FAULT).
- **Carryover:** Klimmzug, Front Lever (FL-Rows sind eine eigene OG-Spalte ab
  Level 5 [A-31]).
- **Equipment:** Die Last steigt mit flacherem Körperwinkel (H-EQ; keine
  verifizierte Messung); Stand ein- oder beidbeinig änderte die Aktivierung nicht
  [A-12].

### 4.8 Klimmzug / Chin-up (`pull-up`, Familie `pull`; bestehender Skill)

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| (Wurzel) | `hang-foundation/arch-hang` (§4.2) | — | — | — | 2–6 Mo. von Dead Hang bis zum ersten Klimmzug [A-46] | [A-44] |
| 1 | `pull-up/strict-5` (existiert) · `pull-up` | 5 strikte Klimmzüge | aus gestrecktem Hang, Kinn über Stange, ohne Kipping (Kipping ändert die Kinematik [A-10]) | `pull-up reps ≥ 5 · none · max_load_kg 0 · occ 2 · 28 d` (H-UNL) | 2–8 Wo. (H-DUR) | OG 3 [A-31] |
| 2 | `pull-up/strict-10` · `pull-up` | 10 strikte Klimmzüge | wie 1 | `pull-up reps ≥ 10 · none · max_load_kg 0 · occ 2 · 28 d` | 4–13 Wo. (H-DUR) | 10 Klimmzüge = FL-Voraussetzung [A-41]; 8–10 = BL-Voraussetzung [A-42] |
| 3 | `pull-up/l-sit` · `pull-up-l-sit` | L-Klimmzug | Beine waagrecht (H-FORM) | `pull-up-l-sit reps ≥ 5 · occ 2 · 28 d` (H-UNL) | 4–13 Wo. (H-DUR) | OG 4 [A-31] |
| 4 | `pull-up/archer` · `pull-up-archer` | Archer an Ringen | Hilfsarm gestreckt (H-FORM) | `pull-up-archer reps ≥ 3 · occ 2 · 28 d` (je Seite, H-UNL) | siehe §7.1 | OG 7 [A-31] |
| (Alt.) | `chin-up` (Übung) | Untergriff | wie 1 | Alternative zu 1–2 | — | [A-08, A-09] |
| (weiter) | Weighted §7.4; One-Arm §7.1 | | | | | [A-31] |

- **Voraussetzungen:** `hang-foundation/arch-hang` (`prerequisite`) [A-44].
- **Häufige Fehler:** halber Weg unten, Kipping, Kinn reckt statt Zug (H-FAULT).
- **Carryover:** Rudern, Front Lever, Muscle-up, One-Arm-Chin (H-PRE). Weiter
  Griff betont den Latissimus, Frontgriff Bizeps/Brachialis, Untergriff anteilig
  die Rotatorenmanschette; die Autoren empfehlen alle drei Varianten [A-09].
- **Equipment:** Stange vs. Ringe (OG führt Ringe-Klimmzüge als eigene Spalte ab
  Level 4 [A-31]); rotierende Griffe brachten keine höhere Aktivierung als
  Pull-up oder Chin-up [A-08].

### 4.9 Kompression / Pike (Teil von `l-sit`, Familie `core`)

Kompression wird nicht als eigener Skill geführt, sondern als Übung mit Rolle
`accessory` an den ersten L-Sit-Stufen (§5.2): GMB empfiehlt sitzendes
einbeiniges Beinheben mit gestrecktem Bein, 30–45 s je Seite, 3–5 Sätze, weil
L-Sits meist an der Hüftbeuger-Kompression scheitern [A-36]. Für den Press in den
Handstand ist Kompression entscheidend: gute Turner zeigen eine engere
Hechtposition und Hüftbeuge-Momente in der späten Phase [A-17].
