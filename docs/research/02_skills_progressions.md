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

## 5. Statics

### 5.1 Handstand und Press (`handstand`, `press-handstand`, Familie `handstand`)

OG: Wand-HS Level 1–3, freistehend 4–5, One-Arm-HS 10; Ring-HS 7 [A-31]. FIG:
Handstand (2 s) = A am Boden [A-29 S. 28]. Biomechanik (**Evidenz A/B**): Balance
überwiegend über eine Handgelenksstrategie (> 75 % der Zeit, gemischte
Strategien ~2 %) [A-15]; bessere Balancen mit Handgelenk- und Schultermomenten,
schwächere mit Hüftmomenten [A-14]; Kopfposition, Sicht und Nackenreflexe
beeinflussen die Balance [A-03]. Ermüdung erhöht die Schwankung [A-18].
Standard-Schultertests sagen die Handstandqualität bei Novizen nicht voraus
[A-19].

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `handstand/wall` (existiert) · `wall-handstand-hold` | Handstand an der Wand | Arme gestreckt, Kopf neutral bis leicht im Nacken [A-03] | `wall-handstand-hold hold ≥ 30 s · occ 3 · 14 d` | 2–8 Wo. (H-DUR) | OG 1–3 [A-31]; 30 s Wand-HS als Planche-Voraussetzung [A-40] |
| 2 | `handstand/chest-to-wall` · `handstand-chest-to-wall` | Bauch zur Wand | Schultern voll geöffnet (Arme an den Ohren), Becken aufgerichtet, Beine gestreckt, Fussspitzen gestreckt [A-03] | `handstand-chest-to-wall hold ≥ 60 s · form≥4 · occ 2 · 14 d` | 2–8 Wo. (H-DUR) | «1-minute stamina hold» an der Wand [A-37] |
| 3 | `handstand/free-10s` · `handstand-freestanding` | freistehend 10 s | wie 2; Korrektur über Finger/Handgelenk, nicht über die Hüfte [A-14, A-15] | `handstand-freestanding hold ≥ 10 s · form≥4 · occ 3 · 14 d` | 4–26 Wo. (H-DUR; GMB: «viele Monate» [A-37]) | OG 4–5 [A-31]; 10 s genügt für viele Ziele [A-37] |
| 4 | `handstand/free-30s` · `handstand-freestanding` | freistehend 30 s | wie 3 | `handstand-freestanding hold ≥ 30 s · form≥4 · occ 3 · 14 d` (H-UNL) | 4–13 Wo. (H-DUR) | (H) |
| 5 | `handstand/free-60s` · `handstand-freestanding` | freistehend 60 s | wie 3 | `handstand-freestanding hold ≥ 60 s · form≥4 · occ 2 · 28 d` | — | ~60 s bequem vor One-Arm-HS-Training [A-37] |
| P1 | `press-handstand/straddle-stand` · `press-handstand-straddle` | Press aus dem Grätschstand, gestreckte Arme | kein Sprung; Arme gestreckt (FIG-Abzug ab 0–15° Beugung [A-29 S. 19]) | `press-handstand-straddle reps ≥ 1 · form≥4 · occ 2 · 28 d` | 4–13 Wo. (H-DUR) | OG 7 [A-31] |
| P2 | `press-handstand/l-sit-straddle` · `press-handstand-from-l-sit` | aus L-Sit/Straddle-L in den Handstand | wie P1 | `press-handstand-from-l-sit reps ≥ 1 · form≥4 · occ 2 · 28 d` | 4–13 Wo. (H-DUR) | OG 8 [A-31] |
| P3 | `press-handstand/l-sit-pike` · `press-handstand-pike-from-l-sit` | aus L-Sit mit geschlossenen Beinen | wie P1 | `… reps ≥ 1 · form≥4 · occ 2 · 28 d` | — | OG 9 [A-31] |
| (Übungen) | `press-handstand-wall-eccentric`, `press-handstand-elevated` | exzentrisch an der Wand, erhöhter Stand | — | Rolle `progression` | — | OG 5–6 [A-31] |

- **Voraussetzungen:** `wrist-conditioning/prep` (`recommended`); Press:
  `handstand/free-10s` (`prerequisite`, H-PRE: der Press endet im freien
  Handstand) und `l-sit/straddle` (`recommended`; Kompression und
  Hüftbeuge-Momente unterscheiden gute von schwachen Pressern [A-17]).
- **Häufige Fehler:** Hohlkreuz, gebeugte Arme, Kopf eingerollt oder überstreckt,
  Korrektur über die Hüfte statt über die Handgelenke [A-14]; zu viel
  Schulterbeuge-Moment statt Hechten im Press [A-17].
- **Carryover:** HSPU, Planche-Press; Handstand-Arbeit überträgt auf Liegestütz
  und Dips [A-30 S. 21]; der Handstand ist Grundlage für Elemente an anderen
  Geräten [A-03].
- **Equipment:** Boden → Barren/Ringe senkt die Aktivität der Handgelenksbeuger
  (61 % → 44–46 % NRMS), an Ringen steigt die übrige Muskelaktivität [A-16];
  Ringe-HS liegt 2–3 OG-Level über dem freistehenden HS am Boden [A-31].
- **Session-Hinweis:** Handstand-Balance früh in der Einheit üben (Ermüdung
  erhöht die Schwankung [A-18]; Detail Stream E); GMB: 2–4×/Woche, 15–20 min
  realistisch [A-37].

### 5.2 L-Sit → V-Sit → Manna (`l-sit`, `v-sit`, `manna`, Familie `core`)

OG: Tuck L-Sit 1 · einbeinig gebeugt 2 · L-Sit 3 · Straddle-L 4 · RTO L-Sit 5 ·
V-Sit 45° 6 · 75° 7 · 100° 8 · 120° 9 · 140° 10 · 155° 11 · 170° 12 · Manna 13
[A-31]. FIG: L-Sit A, V-Sit B (Ringe; V-Halte mit senkrechten Beinen), Manna C
(Boden) [A-29 S. 28, 61, 67]. Im Street-Workout-Wettkampf zählt ein gehockter
V-Sit nicht als Halt [A-34]. OG hält die L/V/Manna-Leiter (hintere Schulter) für
nötig, um die Planche-Stufen zu erreichen [A-30 S. 22].

| Skill / Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| L 1 | `l-sit/tuck` · `l-sit-tuck` | Stütz, Knie angezogen | Arme gestreckt, Schultern tief, Gesäss frei (H-FORM) | `l-sit-tuck hold ≥ 30 s · occ 3 · 7 d` | 2–8 Wo. (H-DUR) | OG 1 [A-31]; 30 s [A-44] |
| L 2 | `l-sit/one-leg` · `l-sit-one-leg` | ein Bein gestreckt | gestrecktes Bein waagrecht (H-FORM) | `l-sit-one-leg hold ≥ 5 s · occ 2 · 14 d` (je Seite) | 2–8 Wo. (H-DUR) | 5 s je Seite vor dem vollen L-Sit [A-36]; OG 2 [A-31] |
| L 3 | `l-sit/full` · `l-sit` | L-Sit | Beine gestreckt und waagrecht (H-FORM) | `l-sit hold ≥ 10 s · form≥4 · occ 2 · 28 d` | 2–8 Wo. (H-DUR) | OG 3 [A-31]; 10 s (H-UNL) |
| L 4 | `l-sit/straddle` · `straddle-l-sit` | gegrätschter L | Beine gestreckt, gegrätscht, waagrecht oder höher (H-FORM) | `straddle-l-sit hold ≥ 10 s · form≥4 · occ 2 · 28 d` | 2–8 Wo. (H-DUR) | OG 4 [A-31] |
| L 5 | `l-sit/rings` · `l-sit-rings-rto` | L-Sit an Ringen, RTO | wie L 3, Ringe nach aussen gedreht | `l-sit-rings-rto hold ≥ 10 s · form≥4 · occ 2 · 28 d` | 4–13 Wo. (H-DUR) | OG 5 [A-31]; FIG A [A-29] |
| V 1–4 | `v-sit/45` · `/75` · `/100` · `/120` · Übungen `v-sit-45` … `v-sit-120` | V-Sit, Beinwinkel über der Waagrechten | Winkel erreicht, Arme gestreckt, Beine gestreckt (H-FORM) | `v-sit-<winkel> hold ≥ 3 s · form≥4 · occ 2 · 28 d` | je 4–13 Wo. (45°–100°), 8–26 Wo. (120°) (H-DUR) | OG 6–9 [A-31]; FIG V-Sit B, 2 s [A-29] |
| V 5–7 | `v-sit/140` · `/155` · `/170` | steile V-Sits | wie oben | wie oben | je 8–26 Wo. (H-DUR) | OG 10–12 [A-31] |
| M 1 | `manna/full` · `manna` | Beine waagrecht, Rumpf hinter den Händen | Beine waagrecht, Arme gestreckt | `manna hold ≥ 3 s · form≥4 · occ 2 · 28 d` | — | OG 13 [A-31]; FIG C, 2 s [A-29 S. 28]; 3 s [A-33] |

- **Voraussetzungen:** `l-sit/tuck` ← `support-hold/parallel-bars`
  (`prerequisite`, H-PRE: L-Sit ist ein Stütz mit angehobenen Beinen);
  `v-sit/45` ← `l-sit/straddle`; `manna/full` ← `v-sit/170` (Reihenfolge
  [A-31]).
- **Häufige Fehler:** Schultern hochgezogen, Knie gebeugt, fehlende Kompression
  (Hüftbeuger krampfen) [A-36].
- **Carryover:** Press-Handstand [A-17]; Planche (hintere Schulter, laut OG
  nötig [A-30 S. 22]); «L-Sit to Tuck Planche» [P-01 S. 1–3].
- **Equipment:** Boden (wenig Freiraum) schwerer als Parallettes/Barren (H-EQ);
  Ringe RTO = OG 5 statt L-Sit OG 3 [A-31]. GMB: Halte 5–30 s, 3–5 Sätze,
  2–4×/Woche [A-36].

### 5.3 Front Lever (`front-lever`, Familie `pull`; bestehender Skill)

OG: Tuck 4 · Adv Tuck 5 · Straddle 6 · Half-Lay/One-Leg 7 · Full 8 [A-31];
FIG A [A-29 S. 67]; im Street-Workout-Wettkampf zählen Tuck, Adv Tuck und
45°-Straddle nicht als Halt [A-34]; die RR nutzt den Tuck FL als Isometrie in der
Row-Leiter [A-44]. TMA nennt als Einstiegsbedingungen 10 strikte Klimmzüge, 30 s
Dead Hang, 60 s Hollow Hold, 15 gestreckte Beinheben und als Wechselpunkte
«5 s Tuck» → Adv Tuck, «10 s Adv Tuck» → Straddle, «5 s Straddle» → Full [A-41].

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `front-lever/tuck` (existiert) · `front-lever-tuck` | Knie zur Brust, Rumpf waagrecht | Arme gestreckt, Schultern depressiert, Rumpf waagrecht ≤ 15° (§2.4) (H-FORM) | `front-lever-tuck hold ≥ 10 s · none · form≥4 · occ 2 · 28 d` | ab 10 Klimmzügen 2–3 Mo. bis Adv Tuck [A-41] | OG 4 [A-31]; Platzhalter nennt 15 s |
| 2 | `front-lever/advanced-tuck` (existiert) · `front-lever-advanced-tuck` | Rücken flach, Hüfte offen, Knie gebeugt | flacher Rücken (H-FORM) | `front-lever-advanced-tuck hold ≥ 10 s · none · form≥4 · occ 2 · 28 d` | 2–3 Mo. [A-41] | OG 5 [A-31]; 10 s Adv Tuck vor Straddle [A-41] |
| 3 | `front-lever/straddle` · `front-lever-straddle` | Beine gestreckt, gegrätscht | Körper waagrecht ±7,5–15°, Beine gestreckt | `front-lever-straddle hold ≥ 3 s · none · form≥4 · occ 2 · 28 d` | 3–4 Mo. bis Full [A-41] | OG 6 [A-31]; 3 s [A-33] |
| 4 | `front-lever/one-leg` · `front-lever-one-leg` (Alt. `front-lever-half-lay`) | ein Bein gestreckt / Knie halb | Hüfte gestreckt, waagrecht (H-FORM) | `front-lever-one-leg hold ≥ 5 s · none · form≥4 · occ 2 · 28 d` (H-UNL) | 4–13 Wo. (H-DUR) | OG 7 [A-31] |
| 5 | `front-lever/full` · `front-lever` | Full Front Lever | gestreckt, waagrecht, Arme gestreckt | `front-lever hold ≥ 3 s · none · form≥4 · occ 2 · 28 d` | — | OG 8 [A-31]; FIG A [A-29]; 3 s [A-33] |
| (Übungen) | `front-lever-row-tuck` (OG 5), `band-assisted-tuck-fl` (existiert), `front-lever-negative` | Rows, Band, Negative | — | Rolle `progression` | — | [A-31] |

Gesamtdauer laut TMA: 18–24 Monate bei < 5 Klimmzügen, 12–18 Monate ab 10
Klimmzügen, 6–12 Monate ab gehaltenem Tuck FL [A-41] (D).

- **Voraussetzungen:** `pull-up/strict-5` (`prerequisite`, wie im Platzhalter);
  `pull-up/strict-10`, `hollow-body/full`, `hang-foundation/dead-hang`
  (`recommended`; TMA-Voraussetzungen [A-41]); `row/horizontal` (`recommended`,
  RR-Kontext [A-44]).
- **Häufige Fehler:** Hüfte unter Schulterlinie, gebeugte Ellbogen, Schultern
  nicht depressiert, runder Rücken im Adv Tuck (H-FAULT); zu viele Maximal-
  versuche — TMA rät zu 1–2 Wdh. Reserve und Maximaltests alle 2–3 Wochen [A-41].
- **Carryover:** Back Lever, Klimmzug, Muscle-up; Planche/Maltese als
  **Antagonist** (Druck vs. Zug); OG hält FL/BL-Kraft für die Planche nötig
  [A-30 S. 22].
- **Equipment:** Stange vs. Ringe (gleiche OG-Stufen; FIG an Ringen A) [A-31,
  A-29]; Beinlänge verschiebt die Schwierigkeit (Stream C).

### 5.4 Back Lever inkl. German Hang und Skin the Cat (`back-lever`, Familie `pull`)

OG: German Hang 1 · Skin the Cat 2 · Tuck 3 · Adv Tuck 4 · Straddle 5 ·
Half-Lay/One-Leg 6 · Full 7 · BL Pullout 8 [A-31]; FIG A [A-29 S. 67]; ein
einbeiniger Back Lever zählt im Street-Workout-Wettkampf nicht als Halt [A-34].
TMA: Vorbereitung (Skin the Cat) 4–8 Wo., Tuck/Pike-Stufen 8–12 Wo., Full
12–24 Wo., gesamt 6–12 Monate, jede Stufe 4–8 Wochen; 8–10 Klimmzüge als
Kraftbasis; höchstens 3–4×/Woche [A-42].

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `back-lever/german-hang` · `german-hang` | Hang in maximaler Schulterstreckung | kontrolliertes Ein- und Aussteigen, schmerzfrei (H-FORM) | `german-hang hold ≥ 15 s · occ 3 · 14 d` (H-UNL: Gewebetoleranz statt Kraft) | 4–8 Wo. (bis Skin the Cat sicher) [A-42] | OG 1 [A-31] |
| 2 | `back-lever/skin-the-cat` · `skin-the-cat` | Durchdrehen in den German Hang und zurück | Arme gestreckt, langsam (H-FORM) | `skin-the-cat reps ≥ 3 · occ 2 · 14 d` (H-UNL) | 4–8 Wo. [A-42] | OG 2 [A-31] |
| 3 | `back-lever/tuck` · `back-lever-tuck` | Tuck, Bauch nach unten | Arme gestreckt, Rumpf waagrecht (H-FORM) | `back-lever-tuck hold ≥ 10 s · form≥4 · occ 2 · 28 d` | 4–8 Wo. [A-42] | OG 3 [A-31] |
| 4 | `back-lever/advanced-tuck` · `back-lever-advanced-tuck` | Hüfte gestreckt, Knie gebeugt | flacher Rücken (H-FORM) | `back-lever-advanced-tuck hold ≥ 10 s · form≥4 · occ 2 · 28 d` | 4–8 Wo. [A-42] | OG 4 [A-31] |
| 5 | `back-lever/straddle` · `back-lever-straddle` | gegrätscht | waagrecht ≤ 15° | `back-lever-straddle hold ≥ 3 s · form≥4 · occ 2 · 28 d` | 4–8 Wo. [A-42] | OG 5 [A-31]; 3 s [A-33] |
| 6 | `back-lever/one-leg` · `back-lever-one-leg` (Alt. Half-Lay) | ein Bein / Knie halb | wie 5 | `back-lever-one-leg hold ≥ 5 s · form≥4 · occ 2 · 28 d` (H-UNL) | 4–8 Wo. [A-42] | OG 6 [A-31] |
| 7 | `back-lever/full` · `back-lever` | Full Back Lever | gestreckte Linie, waagrecht | `back-lever hold ≥ 3 s · form≥4 · occ 2 · 28 d` | — | OG 7 [A-31]; FIG A [A-29] |

- **Voraussetzungen:** `hang-foundation/dead-hang` (`prerequisite`, H-PRE);
  `pull-up/strict-10` (`recommended`; 8–10 Klimmzüge [A-42]); `arch-body/hold`
  (`recommended`, H-PRE).
- **Häufige Fehler:** Hüfte knickt, Arme beugen, zu schnelles Einrollen in den
  German Hang; Stufen zu schnell wechseln (TMA: jede 4–8 Wochen) [A-42].
  Straight-Arm-Belastung von Bizepssehne und vorderer Schulter ist Thema von
  Stream D (ADR 0003 nennt diese Strukturen).
- **Carryover:** Front Lever, Iron Cross (OG führt «Cross to Back Lever» in der
  Kreuz-Spalte [A-31]), Muscle-up-Schwungteile (H-PRE).
- **Equipment:** Ringe (freie Handrotation) vs. Stange (Griff fix) (H-EQ).

### 5.5 Planche (`planche`, `planche-press`, `planche-push-up`, Familie `push`)

OG (Boden/Barren): Frog Stand 3 · Frog Stand mit gestreckten Armen 4 · Tuck 5 ·
Adv Tuck 6 · Straddle 8 · Half-Lay/One-Leg 9 · Full 11; an Ringen jeweils 1–3
Level höher (Tuck 6, Adv Tuck 8, Straddle 10, Half-Lay 12, Full 14) [A-31]. FIG:
Straddle Planche A, Planche C am Boden und an Ringen [A-29 S. 29, 67]; an Ringen
wird die Stützwaage nur anerkannt, wenn die Schultern ganz über den Ringen sind;
«leicht weite oder ausgedrehte Hände» ändern den Wert nicht [A-29 S. 61–62].
In einer EMG-Studie galt als gültige Stützwaage: gestreckte Arme, Schultern
gebeugt bis die Hände in der Senkrechten der Hüfte stehen, Scapula abduziert
[A-24]. PDF-Leiter: Lean → Tuck (5–10 s) → Adv Tuck (über «L-Sit to Adv Tuck»)
→ (Wide) Straddle 2–4 s → Wide Planche 3–6 s → Full (Press, Kicks)
(`01_pdf_extract.md` §4.6). Die Körperproportionen (Rumpf-, Arm-, Beinlängen)
beeinflussen die Planche-Eignung deutlich [A-28].

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 0 | `planche/lean` · `planche-lean` | Liegestützposition, Schultern vor den Händen | Arme gestreckt, Protraktion, Körper gerade (TMA-Fehler: Hüfte hängt/knickt [A-40]) | `planche-lean hold ≥ 30 s · form≥4 · occ 2 · 28 d` | ≈ 2–4 Mo. bis Tuck (aus TMA-Kumulativwerten abgeleitet [A-40]) | TMA-Ziel 3 × 30–60 s [A-40]; PDF 8–20 s [P-01 S. 1, 3] |
| 1 | `planche/frog-stand` · `frog-stand` | Knie auf den Ellbogen | Füsse frei, kontrolliert (H-FORM) | `frog-stand hold ≥ 30 s · occ 3 · 7 d` | 2–8 Wo. (H-DUR) | OG 3 [A-31] |
| 2 | `planche/tuck` · `planche-tuck` | Tuck, Arme gestreckt | Arme gestreckt; Hüfte auf Schulterhöhe ≤ 15°; Hände etwa senkrecht unter der Hüfte; Scapula abduziert [A-24] | `planche-tuck hold ≥ 10 s · none · form≥4 · occ 2 · 28 d` | ≈ 4–6 Mo. bis Adv Tuck (abgeleitet [A-40]) | OG 5 [A-31]; 10 s [A-40]; PDF-Ziel 5–10 s [P-01 S. 1] |
| 3 | `planche/advanced-tuck` · `planche-advanced-tuck` | Rücken waagrecht, Knie vom Körper weg | Rücken parallel zum Boden [A-40] | `planche-advanced-tuck hold ≥ 10 s · none · form≥4 · occ 2 · 28 d` | ≈ 6–12 Mo. bis Straddle (abgeleitet [A-40]) | OG 6 [A-31]; «10 s Adv Tuck» vor Straddle [A-40] |
| 4 | `planche/straddle` · `planche-straddle` | Beine gestreckt, gegrätscht | waagrecht ±7,5–15°, Arme gestreckt | `planche-straddle hold ≥ 3 s · none · form≥4 · occ 2 · 28 d` | ≈ 12 Mo. bis Full (abgeleitet [A-40]) | OG 8 [A-31]; FIG A [A-29]; 3 s [A-33]; PDF 2–4 s [P-01 S. 2] |
| 5 | `planche/half-lay` · `planche-half-lay` (Alt. `planche-one-leg`) | Knie halb gebeugt / ein Bein | wie 4 | `planche-half-lay hold ≥ 3 s · none · form≥4 · occ 2 · 28 d` | 16–52 Wo. (H-DUR) | OG 9 [A-31] |
| 6 | `planche/full` · `planche` | Full Planche | gestreckt, waagrecht, Arme gestreckt | `planche hold ≥ 3 s · none · form≥4 · occ 2 · 28 d` | — | OG 11 [A-31]; FIG C [A-29]; TMA-Ziel ≥ 5 s [A-40] |
| R | `planche/rings` · `planche-rings` | Full Planche an Ringen | Schultern ganz über den Ringen [A-29 S. 61] | `planche-rings hold ≥ 3 s · none · form≥4 · occ 2 · 28 d` | — | OG 14 [A-31]; FIG C [A-29] |
| Pr 1 | `planche-press/straddle` · `planche-straddle-press` | Straddle Planche → Handstand | Arme gestreckt, kein Schwung | `planche-straddle-press reps ≥ 1 · none · form≥4 · occ 2 · 28 d` | — | FIG B [A-29 S. 29]; PDF Beginner 1–3 Wdh. [P-01 S. 2–3] |
| Pr 2 | `planche-press/full` · `planche-full-press` | Full Planche → Handstand | wie Pr 1 | `planche-full-press reps ≥ 1 · none · form≥4 · occ 2 · 28 d` | — | FIG D [A-29 S. 29]; PDF Intermediate 1–3 Wdh. [P-02 S. 2] |
| Pr 3 | `planche-press/full-5` · `planche-full-press` | 5 Wdh. | wie Pr 1 | `planche-full-press reps ≥ 5 · none · form≥4 · occ 2 · 28 d` | — | PDF Advanced 3–8 Wdh. [P-03 S. 2] |
| PU 1–3 | `planche-push-up/tuck` · `/straddle` · `/full` | Planche-Liegestütz je Stufe | Hüfte bleibt auf Schulterhöhe, voller Weg (H-FORM) | `planche-push-up-<stufe> reps ≥ 5 · none · form≥4 · occ 2 · 28 d` | — | OG 6/10/14 [A-31]; PDF 5–10 Wdh. [P-02 S. 3; P-03 S. 3] |
| (Übungen) | Kicks, «L-Sit to Tuck Planche», Negative, Band (`elastic`, `neck band`), `badform`-Halte, Vorübungen (Seilzug, Stützwaage mit Fussauflage, Kurzhanteln) | Volumen- und Technikübungen | — | Rolle `progression`/`accessory`; kein Level | — | [P-01 bis P-03]; Vorübungen [A-24] |

Kumuliert laut TMA: Tuck nach 2–6, Adv Tuck nach 6–12, Straddle nach 12–24,
Full nach 24–36 Monaten (D) [A-40]; OG: Full Planche = Advanced (Level 11), für
Turner dagegen ein «Intermediate»-Element [A-30 S. 23].

- **Voraussetzungen:** `planche/lean` ← `push-up/full` (`prerequisite`) und
  `wrist-conditioning/plank` (`recommended`; 30-s-Plank als Startbedingung
  [A-35]); `planche/tuck` ← `hollow-body/full` und `handstand/wall`
  (`recommended`; TMA: 60 s Hollow, 3 × 20 Liegestütze, 30 s Wand-HS [A-40]);
  `planche-press/straddle` ← `handstand/free-10s` (`prerequisite`, H-PRE) und
  `planche/straddle` (`prerequisite`).
- **Häufige Fehler:** gebeugte Arme, Hüfte zu hoch/zu tief, fehlende Protraktion,
  zu wenig Vorlage (H-FAULT); Stufen überspringen und zu viele Einheiten (GMB:
  höchstens 3 Tage/Woche, nach 2 Monaten ein Tag mehr [A-35]). Die PDFs üben
  bewusst `badform`-Halte [P-01 S. 2] — mit niedriger `form_quality` loggen,
  zählen nicht für den Unlock.
- **Carryover:** Pseudo-Planche-Liegestütz, HSPU (P-04 kombiniert Weighted Lean
  Planche mit Weighted HSPU [P-04 S. 1]), Maltese (Wide Planche als Vorstufe
  [P-02 S. 4]); OG: für die Planche braucht es auch L/V/Manna, Front und Back
  Lever [A-30 S. 22]. Vorübungen korrelieren (r = 0,69 mit «Schwalbe in
  Rückenlage») [A-20], erklären aber nur 42–59 % [A-21] und aktivieren anders
  [A-24].
- **Equipment:** Boden, Parallettes (`pbar`, in den PDFs Standard), Ringe (+1 bis
  +3 OG-Level [A-31]). Parallettes entlasten die Handgelenk-Extension (TMA [A-40])
  und erlauben tiefere Liegestütze («DEEP» [P-02 S. 3]). Weite und ausgedrehte
  Hände sind bei FIG wertneutral [A-29 S. 62]; ihre Wirkung auf die Schwierigkeit
  ist nicht gemessen (H-EQ).

### 5.6 Maltese (`maltese`, Familie `push`) — PDF-Leiter validiert und erweitert

PDF-Leiter (`01_pdf_extract.md` §4.6): Lean Maltese mit Band (Beginner) → Lean
Maltese frei 5–15 s → Lean Maltese Elevator 2–5 Wdh. → Wide Planche Hold/Press →
Maltese Hold mit Band 3–8 s → Maltese Press 1–3 Wdh. mit Band → ohne Band → am
Boden [P-01 S. 2; P-02 S. 1, 4; P-03 S. 1, 4; P-04 S. 1].

**Validierung gegen externe Quellen:**

| Befund | Quelle | Folge für die Leiter |
|---|---|---|
| FIG: Schwalbe/Maltese an Ringen D, am Boden C (gleich wie Planche) | [A-29 S. 29, 67] | Ringe-Maltese als eigene, höhere Stufe |
| FIG-Definition an Ringen: waagrecht, Körper gestreckt, Schultermitte auf Höhe der Ringunterkante, Arme weit, ohne Kontakt zum Oberkörper | [A-29 S. 61] | Formkriterien |
| OG: Maltese = Level 17, jenseits des 16-Level-Charts; Full Planche 11 | [A-31] | Maltese klar oberhalb der Full Planche |
| Konditionierungskraft für die Schwalbe: 63,05 % KG konzentrisch, 94,10 % KG exzentrisch; höchste Werte aller Ringelemente; nur 4 von 19 Elite-Turnern hielten die Schwalbe mit ≥ Körpergewicht | [A-21] | Realismus-Check bei Zielen |
| 1RM «Schwalbe in Rückenlage» 73,4 % KG nötig; Korrelation mit Schwalbe r = 0,71, mit Bankdrücken r = 0,71 | [A-20] | Kraftbasis als `recommended` |
| Die Schwalbe an Ringen wird mit supinierten (90° aussenrotierten) Händen ausgeführt, die Stützwaage meist mit normaler Handstellung | [A-21] | stützt die Deutung `supi` = supiniert (§9) |
| Vorübungen (Gegengewicht, Kurzhantel, Langhantel) koordinieren die Schultermuskeln anders als die Schwalbe | [A-25] | Vorübungen ersetzen die spezifische Übung nicht |
| Stützgeräte (Herdos, Gurte) senken die Last bei Halteelementen | [A-04] | bestätigt Band-Assistenz als Werkzeug wie in den PDFs |
| Der Maltese-**Halt** erscheint in den PDFs nur mit Band | `01_pdf_extract.md` §4.4 | unassistierter Halt liegt jenseits der Programme → eigene Endstufe |

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `maltese/lean` · `maltese-lean` | Lean mit weit seitlich gestellten Händen | Arme gestreckt, Körper gerade (H-FORM) | `maltese-lean hold ≥ 10 s · none · form≥4 · occ 2 · 28 d` | 13–52 Wo. (H-DUR) | PDF frei 5–15 s [P-02 S. 4; P-03 S. 4] |
| 2 | `maltese/lean-elevator` · `maltese-lean-elevator` | dynamische Lean-Variante mit Wdh. (Deutung §9) | wie 1 | `maltese-lean-elevator reps ≥ 3 · none · form≥4 · occ 2 · 28 d` | 13–52 Wo. (H-DUR) | PDF 2–5 Wdh. [P-02 S. 4; P-03 S. 4] |
| 3 | `maltese/wide-planche` · `planche-wide` | Planche mit weiter Handstellung | waagrecht ≤ 15° | `planche-wide hold ≥ 3 s · none · form≥4 · occ 2 · 28 d` | 13–52 Wo. (H-DUR) | PDF 3–6 s [P-02 S. 1]; FIG: weite Hände wertneutral [A-29 S. 62] |
| 4 | `maltese/press` · `maltese-press` | Press aus der Maltese in den Handstand, unassistiert | Arme gestreckt | `maltese-press reps ≥ 1 · none · form≥4 · occ 2 · 28 d` | — | PDF Advanced 1–3 Wdh. ohne Band [P-03 S. 4] |
| 5 | `maltese/hold` · `maltese` | Maltese-Halt, unassistiert (Boden/Parallettes) | Körper waagrecht auf Handhöhe, Arme weit und gestreckt [A-29 S. 61] | `maltese hold ≥ 3 s · none · form≥4 · occ 2 · 28 d` | — | FIG C (Boden) [A-29]; 3 s [A-33]; **Erweiterung** |
| 6 | `maltese/rings` · `maltese-rings` | Schwalbe an Ringen | Schultermitte auf Höhe der Ringunterkante [A-29 S. 61] | `maltese-rings hold ≥ 3 s · none · form≥4 · occ 2 · 28 d` | — | FIG D [A-29]; **Erweiterung** |
| (Übungen) | `maltese-lean-band`, `maltese-hold-band`, `maltese-press-band`, `maltese-elevator`, `zanetti-vertical-band`, `swallow-supine-dumbbell` | assistierte Volumen-, Technik- und Kraftübungen | — | Rolle `progression`/`accessory` | — | [P-01 bis P-04]; Vorübung [A-20] |

Die Reihenfolge Press (4) vor Halt (5) folgt den PDFs (Press ohne Band ab
Advanced, Halt nie ohne Band); ob der unassistierte Halt schwerer ist als der
Press, bleibt offen.

- **Voraussetzungen:** `maltese/lean` ← `planche/advanced-tuck` (`prerequisite`;
  die PDFs beginnen Lean Maltese auf der Stufe, die an der Straddle Planche
  arbeitet [P-01 S. 2]; H-PRE). `maltese/wide-planche` ← `planche/straddle`
  (`prerequisite`). `maltese/press` ← `planche-press/full` (`prerequisite`; der
  Maltese-Press mit Band erscheint erst mit dem Full-Planche-Press [P-02 S. 2, 4];
  H-PRE). Kraftbasis «Schwalbe in Rückenlage ≥ 73,4 % KG (1RM)» und Bankdrücken
  als `recommended` [A-20].
- **Häufige Fehler:** Hände wandern nach innen (wird zur Planche), Arme berühren
  den Oberkörper (FIG-Abzug [A-29 S. 61]), Hüfte knickt, Arme beugen (H-FAULT).
- **Carryover:** Planche (beidseitig), Iron Cross (weite Armhaltung), Victorian
  (H-PRE).
- **Equipment:** Boden, `pbar`, `supi bar`, Ringe [P-02 bis P-04]. FIG wertet
  Ringe höher als Boden (D vs. C) [A-29]; beim Autor steht der Press **am Boden**
  an der Spitze [P-04 S. 1] (→ Widersprüche).

### 5.7 Human Flag (`human-flag`, Familie `core`)

OG: Tuck 5 · Adv Tuck 6 · Straddle 7 · Full 8 [A-31]; WSWCF führt die Flagge als
isometrisches Element der Frontalebene [A-33 S. 4]. Weitere Quellen wurden nicht
gefunden; Schwellen und Dauern sind Heuristik.

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 0 | `human-flag/vertical` · `human-flag-vertical` | Körper senkrecht seitlich an der Stange | untere Hand drückt, obere zieht, Arme gestreckt (H-FORM) | `human-flag-vertical hold ≥ 10 s · form≥4 · occ 2 · 28 d` (H-UNL) | 4–13 Wo. (H-DUR) | Stufe (H) |
| 1 | `human-flag/tuck` · `human-flag-tuck` | Tuck, Rumpf waagrecht | Arme gestreckt | `human-flag-tuck hold ≥ 10 s · form≥4 · occ 2 · 28 d` | 4–13 Wo. (H-DUR) | OG 5 [A-31] |
| 2 | `human-flag/advanced-tuck` · `human-flag-advanced-tuck` | Hüfte gestreckt, Knie gebeugt | wie 1 | `human-flag-advanced-tuck hold ≥ 10 s · form≥4 · occ 2 · 28 d` | 4–13 Wo. (H-DUR) | OG 6 [A-31] |
| 3 | `human-flag/straddle` · `human-flag-straddle` | gegrätscht, waagrecht | waagrecht ≤ 15° | `human-flag-straddle hold ≥ 3 s · form≥4 · occ 2 · 28 d` | 4–13 Wo. (H-DUR) | OG 7 [A-31]; 3 s [A-33] |
| 4 | `human-flag/full` · `human-flag` | Beine geschlossen, waagrecht | waagrecht ≤ 15° | `human-flag hold ≥ 3 s · form≥4 · occ 2 · 28 d` | — | OG 8 [A-31] |

- **Voraussetzungen:** `pull-up/strict-10` und `support-hold/parallel-bars`
  (`recommended`, H-PRE).
- **Häufige Fehler:** untere Hand gebeugt, Rotation nach vorn, Hüfte hängt
  (H-FAULT).
- **Equipment:** senkrechte Stange vs. Sprossenwand (H-EQ). Beide Seiten getrennt
  loggen (`unilateral: true`).

## 6. Dynamics

### 6.1 Muscle-up — Stange und Ringe (`muscle-up`, Familie `dynamic`)

OG: MU exzentrisch 3 · Kipping-MU 4 · Muscle-up 5 · breit/ohne False Grip 6 ·
strikter Stangen-MU 7 · L-Sit-MU 8 [A-31]; in der Nachbarspalte Kipping-Klimmzug
mit Klatschen 4, Klatschen ohne Kipping 5 [A-31]. EMG (Evidenz B): Beim
Ring-MU sind oberer Trapez, Bizeps und Unterarmbeuger in der Zugphase und
Trizeps/Bizeps in der Stützphase stärker aktiv als beim Stangen-MU; die Autoren
empfehlen den Stangen-MU als ersten Lernschritt [A-13] (gemessen mit erlaubtem
Schwung an Probanden, die je 5 MUs an Stange und Ringen schafften [A-13]). Der
Ring-MU entspricht einem Stemme-vorwärts-Muster, der Stangen-MU einer Kippe
[A-13]. Kipping verändert Kinematik und Muskelaktivierung deutlich (maximaler
Hüftwinkel +48,8°) [A-10]. False Grip = im Handgelenk abgeknickter Griff; an den
Ringen ist er für Kraft-Halteelemente im Turnen nicht erlaubt [A-29 S. 62], im
MU erleichtert er den Übergang (OG: MU ohne False Grip ein Level schwerer
[A-31]).

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| (Übungen) | `muscle-up-negative`, `muscle-up-band`, `pull-up-explosive`, `pull-up-chest-to-bar`, `straight-bar-dip` | Zubringer | — | Rolle `progression`/`accessory` | — | OG 3–5 [A-31] |
| 1 | `muscle-up/bar-kipping` · `muscle-up-bar-kipping` | Stangen-MU mit Schwung | beide Arme gleichzeitig über die Stange (kein «Chicken Wing»), Endposition gestreckter Stütz (H-FORM) | `muscle-up-bar-kipping reps ≥ 1 · none · form≥4 · occ 2 · 28 d` | 2–8 Wo. (H-DUR) | OG 4 [A-31]; Stange zuerst [A-13] |
| 2 | `muscle-up/rings` · `muscle-up-rings` | Ring-MU (False Grip erlaubt, ohne Kipping) | Ringe nah am Körper, Übergang ohne Pause im tiefen Dip, Endposition Stütz (H-FORM) | `muscle-up-rings reps ≥ 1 · none · form≥4 · occ 2 · 28 d` | 4–13 Wo. (H-DUR) | OG 5 [A-31]; [A-13] |
| 3 | `muscle-up/bar-strict` · `muscle-up-bar-strict` | strikter Stangen-MU | ohne Beinschwung und ohne Kippe (H-FORM; Kipping-Kriterien [A-10]) | `muscle-up-bar-strict reps ≥ 1 · none · form≥4 · occ 2 · 28 d` | 8–26 Wo. (H-DUR) | OG 7 [A-31] |
| 4 | `muscle-up/l-sit` · `muscle-up-l-sit` | MU im L-Sit | Beine waagrecht während der ganzen Bewegung | `muscle-up-l-sit reps ≥ 1 · none · form≥4 · occ 2 · 28 d` | — | OG 8 [A-31] |
| (Alt.) | `muscle-up-rings-no-false-grip` | Ring-MU ohne False Grip | — | Alternative Übung zu Stufe 2/3 | — | OG 6 [A-31] |

- **Voraussetzungen:** `muscle-up/bar-kipping` ← `pull-up/strict-10`
  (`recommended`) und `dip/parallel-bars` (`prerequisite`) (H-PRE: der MU ist
  «Klimmzug + Dip» [A-13]); `muscle-up/rings` ← `support-hold/rings` und
  `dip/rings` (`prerequisite`, H-PRE); `muscle-up/l-sit` ← `l-sit/full`
  (`prerequisite`).
- **Häufige Fehler:** ein Arm nach dem anderen über die Stange, Hängenbleiben im
  tiefen Dip, Ringe weit vom Körper, übermässiger Beinschwung bei «strikten»
  Wiederholungen (H-FAULT).
- **Carryover:** Klimmzug (explosiv), Dips, Stangen-Skills (Kippe, Pullover)
  (H-PRE).
- **Equipment:** Stange zuerst, Ringe verlangen mehr Arm-Rekrutierung [A-13];
  Kipping und strikt als getrennte Übungen führen [A-10].
- **Unlock-Hinweis:** Kipping-Wiederholungen zählen nie für strikte Stufen
  (getrennte Slugs, §2.5).

### 6.2 Schwung- und Release-Elemente: Umfang (`bar-swing`, Familie `dynamic`)

**Entscheidung zum Umfang (Vorschlag):** Aufgenommen werden Elemente an der
Stange, die (a) in der Street-Workout-Skillkategorie vorkommen [A-33 S. 4],
(b) sich stufenweise an einer Stange lernen lassen und (c) ohne Salto und ohne
grosse Flugphase auskommen. **Nicht** aufgenommen (v1): Riesenfelgen,
Salto-/Flip-Elemente (Frontflip Regrab, Shrimp Flip, Gainer), 540er und Abgänge.
Begründung: Selbst im Spitzenturnen gelten an Reck und Ringen Landematten
(20 + 10 cm) und ein Helfer als Recht des Athleten [A-29 S. 6]; die App kann
solche Bedingungen nicht prüfen. OG behandelt Schwung-, Riesenfelgen- und
Salto-Elemente bewusst nicht [A-47, A-30 S. 21]. Flugelemente müssen im Turnen
eine deutliche Steigphase zeigen [A-29 S. 115] — ein Merkmal, das die App nicht
messen kann (H).

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `bar-swing/tap-swing` · `tap-swing` | Hohlkreuz-/Hollow-Schwung im Hang | aktive Schultern, gestreckte Arme, kontrollierte Umkehr (H-FORM) | `tap-swing reps ≥ 10 · occ 2 · 14 d` (H-UNL) | 2–6 Wo. (H-DUR) | (H) |
| 2 | `bar-swing/kip` · `kip-to-support` | Kippe in den Stütz | aus Schwung ohne Zug-Schwung-Mischung; Endposition Stütz | `kip-to-support reps ≥ 3 · occ 2 · 28 d` (H-UNL) | 4–13 Wo. (H-DUR) | WSWCF-Skillbeispiel [A-33 S. 4]; Ringe-Kippe OG 6 [A-31] |
| 3 | `bar-swing/pullover` · `pullover` | Umschwung aus dem Hang in den Stütz | Arme gebeugt erlaubt, ohne Beinstoss (H-FORM) | `pullover reps ≥ 3 · occ 2 · 28 d` (H-UNL) | 4–13 Wo. (H-DUR) | OG 5 [A-31] |
| 4 | `bar-swing/hip-circle` · `hip-circle` | Hüftumschwung/Felgumschwung im Stütz | ohne Bodenkontakt, Endposition Stütz | `hip-circle reps ≥ 3 · occ 2 · 28 d` (H-UNL) | 4–13 Wo. (H-DUR) | WSWCF-Skillbeispiel [A-33 S. 4] |
| 5 | `bar-swing/release-180` · `swing-180-regrip` | Schwung mit halber Drehung und Umgreifen | beidhändiges Wiederfassen, kein Bodenkontakt [A-33 S. 6] | nur Selbstbestätigung (H: Sicherheitsvorbehalt) | — | Validitätsregel [A-33 S. 6] |
| 6 | `bar-swing/release-360` · `swing-360` | Release mit ganzer Drehung (Swing 360) | wie 5 | nur Selbstbestätigung | — | WSWCF-Beispiel [A-33 S. 6] |

- **Voraussetzungen:** `hang-foundation/arch-hang` und `hollow-body/full`
  (`prerequisite` für den Tap Swing, H-PRE); `muscle-up/bar-kipping`
  (`recommended` für Kippe, H-PRE); Releases ← `bar-swing/hip-circle`
  (`prerequisite`, H-PRE).
- **Häufige Fehler:** Schwung aus gebeugten Armen, Loslassen ohne Steigphase,
  einhändiges Wiederfassen (WSWCF-Abzug [A-33 S. 6]).
- **Sicherheit:** Releases nur mit Matten/Helfer empfehlen (H, analog FIG
  [A-29 S. 6]); keine automatische Freischaltung, kein Planer-Autoprogramm ohne
  Bestätigung der Rahmenbedingungen (H).

## 7. Kraftskills

### 7.1 One-Arm Pull-up / One-Arm Chin-up (`one-arm-pull-up`, Familie `pull`)

OG (Ringe-Spalte): Archer 7 · One-Arm-Chin exzentrisch 8 · One-Arm-Chin (OAC) 9 ·
OAC + 15 lb (6,8 kg) 10 · OAC + 25 lb (11,3 kg) 11 [A-31]. Auf gleicher Höhe
(Level 9) steht der gewichtete Klimmzug mit 1,90× KG Gesamtlast [A-31] — nach OG
zeigen Übungen desselben Levels ähnliche Leistungsfähigkeit [A-30 S. 22]. Im
Advanced-Bereich nennt OG den Bizeps als typisches Schwachglied beim einarmigen
Klimmzug [A-30 S. 25]. WSWCF führt One-Arm-Pull-ups als dynamisches Kraftelement
[A-33 S. 4].

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `one-arm-pull-up/archer` · `pull-up-archer` | Archer-Klimmzug (Ringe oder Stange) | Hilfsarm gestreckt, Kinn über Griff (H-FORM) | `pull-up-archer reps ≥ 3 · none · form≥4 · occ 2 · 28 d` (je Seite) | 8–26 Wo. (H-DUR) | OG 7 [A-31] |
| 2 | `one-arm-pull-up/typewriter` · `pull-up-typewriter` | seitliches Verschieben oben | Kinn bleibt über der Stange (H-FORM) | `pull-up-typewriter reps ≥ 3 · none · occ 2 · 28 d` (H-UNL) | 8–26 Wo. (H-DUR) | Stufe (H) |
| 3 | `one-arm-pull-up/chin` · `one-arm-chin-up` | einarmiger Klimmzug im Untergriff | voller Weg aus gestrecktem Hang, freie Hand ohne Kontakt (H-FORM) | `one-arm-chin-up reps ≥ 1 · none · form≥4 · occ 2 · 28 d` (je Seite) | 8–26 Wo. (H-DUR) | OG 9 [A-31] |
| 4 | `one-arm-pull-up/weighted-7kg` · `one-arm-chin-up` | OAC mit 6,8 kg | wie 3 | `one-arm-chin-up reps ≥ 1 · min_load_kg 6.8 · occ 2 · 28 d` | — | OG 10 [A-31] |
| (Übungen) | `one-arm-chin-eccentric` (OG 8), `one-arm-pull-up-assisted` (Hand am Handgelenk/Band), `bicep-curl` | Zubringer | — | Rolle `progression`/`accessory` | — | [A-31]; Bizeps-Schwachglied [A-30 S. 25] |

- **Voraussetzungen:** `pull-up/archer` bzw. Stufe 1 ← `pull-up/strict-10`
  (`prerequisite`, H-PRE); `weighted-pull-up/bw-150` (`recommended`; nach OG
  gleichwertige Kraft auf niedrigerem Level [A-31]).
- **Häufige Fehler:** Rumpfrotation, Schwung, halber Weg (H-FAULT).
- **Equipment:** OG misst den OAC an Ringen/Stange im Untergriff; der Obergriff
  (One-Arm Pull-up) gilt in der Praxis als schwerer (H-EQ).

### 7.2 Handstand-Liegestütz (`hspu`, Familie `handstand`)

OG: Pike Headstand-PU 1 · Box Headstand-PU 2 · Wand-Headstand-PU exzentrisch 3 ·
Wand-Headstand-PU 4 · Wand-HSPU (volle ROM) 5 · freier Headstand-PU 6 · freier HSPU
7; Ringe: breit 7, mit Seilkontakt 8, frei 9 [A-31]. OG unterscheidet
Headstand-PU (Kopf bis Boden) und HSPU (volle ROM, z. B. auf Parallettes) [A-31].
WSWCF zählt HSPU als dynamisches Kraftelement [A-33 S. 4]; der PDF-Autor nutzt
gewichtete HSPU [P-04 S. 1].

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `hspu/pike` · `pike-push-up` | Pike-Liegestütz, Kopf zum Boden | Hüfte hoch, Ellbogen ≤ 45° vom Körper (H-FORM) | `pike-push-up reps ≥ 8 · occ 3 · 7 d` | 2–8 Wo. (H-DUR) | OG 1 [A-31]; 3×8 [A-44] |
| 2 | `hspu/pike-elevated` · `pike-push-up-elevated` | Füsse auf Box | wie 1 | `pike-push-up-elevated reps ≥ 8 · occ 3 · 7 d` | 2–8 Wo. (H-DUR) | OG 2 [A-31] |
| 3 | `hspu/wall-headstand` · `headstand-push-up-wall` | Wand, Kopf bis Boden | Kopf und Hände bilden Dreieck, voller Weg bis Kopf (H-FORM) | `headstand-push-up-wall reps ≥ 5 · occ 2 · 28 d` (H-UNL) | 2–8 Wo. (H-DUR) | OG 4 [A-31] |
| 4 | `hspu/wall-full` · `handstand-push-up-wall-deficit` | Wand, Hände auf Parallettes, volle ROM | Schultern bis Hände, oben gestreckt (H-FORM) | `handstand-push-up-wall-deficit reps ≥ 5 · occ 2 · 28 d` (H-UNL) | 4–13 Wo. (H-DUR) | OG 5 [A-31] |
| 5 | `hspu/free-headstand` · `headstand-push-up-free` | freistehend, Kopf bis Boden | ohne Wandkontakt | `headstand-push-up-free reps ≥ 1 · form≥4 · occ 2 · 28 d` | 4–13 Wo. (H-DUR) | OG 6 [A-31] |
| 6 | `hspu/free-full` · `handstand-push-up-free` | freistehend, volle ROM | ohne Wandkontakt, Balance über Handgelenke [A-15] | `handstand-push-up-free reps ≥ 1 · form≥4 · occ 2 · 28 d` | — | OG 7 [A-31] |
| (Übungen) | `headstand-push-up-wall-eccentric` (OG 3), `handstand-push-up-weighted` | Zubringer / Überlastung | — | Rolle `progression` | — | [A-31]; [P-04 S. 1] |

- **Voraussetzungen:** `hspu/wall-headstand` ← `handstand/wall` (`prerequisite`,
  H-PRE); `hspu/free-*` ← `handstand/free-30s` (`prerequisite`, H-PRE).
- **Häufige Fehler:** Ellbogen weit ausgestellt, Hohlkreuz, halber Weg (H-FAULT).
- **Carryover:** Liegestütz/Dips (OG [A-30 S. 21]), Planche-Press (H-PRE).
- **Equipment:** Wand (Balance entlastet) < frei; Parallettes = volle ROM
  (Deficit); Ringe +2 Level [A-31].

### 7.3 Pistol Squat (`pistol-squat`, Familie `legs`)

OG: parallele Kniebeuge 1 · tiefe 2 · seitlich verlagerte 3 · Pistol 4 · Pistol mit
1,2× KG Gesamtlast 5 · 1,35× 6 · 1,5× 7 · 1,65× 8 · 1,8× 9 · 1,9× 10 · 2,0× 11
[A-31]. Die RR beginnt die Beinleiter mit der assistierten Kniebeuge [A-45]. Keine
Studie zu Pistol-Progressionen gefunden.

| Stufe | Slug-Vorschlag | Beschreibung | Formkriterien | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|---|
| 1 | `pistol-squat/deep-squat` · `squat-bodyweight` | tiefe Kniebeuge | Fersen am Boden, Oberschenkel unter der Waagrechten (H-FORM) | `squat-bodyweight reps ≥ 8 · occ 3 · 7 d` | 1–4 Wo. (H-DUR) | OG 2 [A-31] |
| 2 | `pistol-squat/side-to-side` · `squat-side-to-side` | seitlich verlagerte Kniebeuge (Cossack) | Belastung auf einem Bein (H-FORM) | `squat-side-to-side reps ≥ 8 · occ 3 · 7 d` (je Seite) | 2–8 Wo. (H-DUR) | OG 3 [A-31] |
| 3 | `pistol-squat/box` · `pistol-squat-box` | einbeinig auf eine Box | kontrolliert ab, ohne Abprallen (H-FORM) | `pistol-squat-box reps ≥ 8 · occ 3 · 7 d` (je Seite) | 2–8 Wo. (H-DUR) | Stufe (H) |
| 4 | `pistol-squat/full` · `pistol-squat` | voller Pistol | tiefste Position ohne Abstützen, Knie in Fussrichtung, Ferse am Boden (H-FORM) | `pistol-squat reps ≥ 5 · form≥4 · occ 2 · 28 d` (je Seite) | — | OG 4 [A-31] |
| 5 | `pistol-squat/weighted-20` · `pistol-squat` | Pistol mit +20 % KG | wie 4 | `pistol-squat reps ≥ 1 · min_load_kg <0,2×KG> · occ 2 · 28 d` (%-KG nicht ausdrückbar, §2.5) | — | OG 5 [A-31] |
| (Übungen) | `pistol-squat-assisted` (Halten an Ring/Pfosten) | assistiert | — | Rolle `progression` | — | Stufe (H) |

- **Voraussetzungen:** keine ausser Stufe 1 (Wurzel für die Beine).
- **Häufige Fehler:** Knie fällt nach innen, Ferse hebt ab, Rundrücken ganz unten
  (H-FAULT).

### 7.4 Gewichtete Grundübungen (`weighted-pull-up`, `weighted-dip`, Familie `pull`/`push`)

OG definiert Level über die **Gesamtlast als Vielfaches des Körpergewichts**
[A-31] (Wiederholungszahl im Chart nicht angegeben):

| OG-Level | Klimmzug (× KG) | Zusatzlast | Dip (× KG) | Zusatzlast |
|---|---|---|---|---|
| 3 | 1,00 | 0 % | 1,00 | 0 % |
| 4 | 1,18 | +18 % | 1,20 | +20 % |
| 5 | 1,35 | +35 % | 1,38 | +38 % |
| 6 | 1,50 | +50 % | 1,55 | +55 % |
| 7 | 1,65 | +65 % | 1,70 | +70 % |
| 8 | 1,78 | +78 % | 1,85 | +85 % |
| 9 | 1,90 | +90 % | 2,00 | +100 % |
| 10 | 2,00 | +100 % | 2,13 | +113 % |
| 11 | 2,10 | +110 % | 2,25 | +125 % |

Quelle: [A-31]; Zusatzlast = eigene Umrechnung (× KG − 1). Referenzwerte aus
einer Studie: Studierende erreichten im 1RM-Klimmzug 1,16 ± 0,15× (Männer) bzw.
0,73 ± 0,09× KG (Frauen) Gesamtlast [A-11] — Männer im Mittel also knapp OG-Level
4, Frauen unter Level 3. TMA nennt +40 % KG im gewichteten Klimmzug als Mittel
gegen ein Front-Lever-Plateau [A-41]. Streetlifting (Maximallast in gewichteten
Klimmzügen, Dips, Muscle-ups, Kniebeugen) wird vor allem vom Verhältnis Kraft zu
Körpergewicht bestimmt [A-48].

| Stufe | Slug-Vorschlag | Beschreibung | Unlock-Kriterium | Dauer bis nächste Stufe | Quellen |
|---|---|---|---|---|---|
| WP 1 | `weighted-pull-up/bw-118` · `pull-up-weighted` | Klimmzug +18 % KG | Ziel-DSL: `pull-up-weighted reps ≥ 1 · min_load_pct_bw 18 · occ 2 · 28 d`; bis zur DSL-Erweiterung Selbstbestätigung | 4–13 Wo. (H-DUR) | OG 4 [A-31] |
| WP 2 | `weighted-pull-up/bw-135` | +35 % KG | wie WP 1 mit 35 | 4–13 Wo. (H-DUR) | OG 5 [A-31] |
| WP 3 | `weighted-pull-up/bw-150` | +50 % KG | wie WP 1 mit 50 | 4–13 Wo. (H-DUR) | OG 6 [A-31] |
| WP 4 | `weighted-pull-up/bw-178` | +78 % KG | wie WP 1 mit 78 | 8–26 Wo. (H-DUR) | OG 8 [A-31] |
| WP 5 | `weighted-pull-up/bw-200` | +100 % KG | wie WP 1 mit 100 | — | OG 10 [A-31] |
| WD 1–5 | `weighted-dip/bw-120` · `/bw-138` · `/bw-155` · `/bw-185` · `/bw-213` | Dip +20/+38/+55/+85/+113 % KG | analog | wie WP | OG 4/5/6/8/10 [A-31] |

- **Voraussetzungen:** `weighted-pull-up/bw-118` ← `pull-up/strict-10`;
  `weighted-dip/bw-120` ← `dip/parallel-bars` (`prerequisite`, H-PRE).
- **Offen:** Wiederholungszahl je OG-Level (Chart schweigt; Buch Kap. 23 nicht
  eingesehen); geschlechtsspezifische Normen (Unterschied in [A-11]).
- **DSL:** `min_load_pct_bw` fehlt (§2.5). Übergangsweise: Selbstbestätigung
  oder absolute Stufen (z. B. +10/+20/+30 kg) — Letztere ohne Quelle (H).

## 8. Kanten zwischen Skills (Voraussetzung, Empfehlung, Alternative, Antagonist)

Gewichte (Vorschlag, H): `prerequisite` 1,0 (hart); `recommended` 0,3–0,7 je nach
Stärke des Belegs; `alternative` 0,5; `antagonist` 0,5. «Kraftbaselines» sind
wegen [A-21, A-23, A-24] nie `prerequisite`.

| Von (Skill/Level) | Nach (Skill/Level) | Relation | Gewicht | Begründung | Quelle / Evidenz |
|---|---|---|---|---|---|
| `hang-foundation/arch-hang` | `pull-up/strict-5` | prerequisite | 1,0 | RR-Leiter Scapula → Arch → Negative → Klimmzug | [A-44] D |
| `support-hold/parallel-bars` | `dip/parallel-bars` | prerequisite | 1,0 | RR-Dip-Leiter beginnt mit dem Stütz | [A-44] D |
| `support-hold/rings` | `dip/rings` | prerequisite | 1,0 | 30 s Ring-Stütz vor dem ersten Ring-Dip | [A-43] D |
| `dip/parallel-bars` | `dip/rings` | recommended | 0,7 | 15 Barren-Dips vor Ring-Dips; OG 3 → 4 | [A-43] D; [A-31] C |
| `support-hold/parallel-bars` | `l-sit/tuck` | prerequisite | 1,0 | L-Sit ist ein Stütz mit gehobenen Beinen | (H-PRE) |
| `push-up/full` | `planche/lean` | prerequisite | 1,0 | Lean = Liegestützposition mit Vorlage | (H-PRE); TMA: 15+ saubere Liegestütze [A-40] D |
| `wrist-conditioning/prep` | `handstand/wall`, `planche/lean` | recommended | 0,5 | Handgelenke tragen die Balance bzw. die Extension | [A-15, A-16] B; [A-39] C |
| `wrist-conditioning/plank` | `planche/lean` | recommended | 0,5 | 30 s Plank als Startbedingung für Leans | [A-35] C |
| `hollow-body/full` | `front-lever/tuck`, `planche/tuck` | recommended | 0,5 | 60 s Hollow als Voraussetzung genannt | [A-41, A-40] D |
| `handstand/wall` | `planche/tuck` | recommended | 0,4 | 30 s Wand-HS als Voraussetzung genannt | [A-40] D |
| `push-up/pseudo-planche` | `planche/tuck` | recommended | 0,5 | Zubringer in allen PDF-Niveaus | [P-01 bis P-03] C |
| `l-sit/straddle`, `back-lever/tuck`, `front-lever/tuck` | `planche/straddle` | recommended | 0,4 | OG: für die Planche braucht es auch hintere Schulter (L/V/Manna, FL, BL) | [A-30 S. 22] C |
| `pull-up/strict-5` | `front-lever/tuck` | prerequisite | 1,0 | bestehender Platzhalter | (H-PRE) |
| `pull-up/strict-10` | `front-lever/tuck` | recommended | 0,6 | 10 strikte Klimmzüge als Einstieg | [A-41] D |
| `hang-foundation/dead-hang` | `front-lever/tuck` | recommended | 0,3 | 30 s Dead Hang als Einstieg | [A-41] D |
| `pull-up/strict-10` | `back-lever/tuck` | recommended | 0,5 | 8–10 Klimmzüge als Kraftbasis | [A-42] D |
| `hang-foundation/dead-hang` | `back-lever/german-hang` | prerequisite | 1,0 | Hangfähigkeit vor Schulterstreckung im Hang | (H-PRE) |
| `weighted-pull-up/bw-135` | `front-lever/straddle` | recommended | 0,3 | +40 % KG als Mittel gegen FL-Plateau | [A-41] D |
| `handstand/free-10s` | `press-handstand/straddle-stand`, `planche-press/straddle` | prerequisite | 1,0 | Press endet im freien Handstand | (H-PRE) |
| `l-sit/straddle` | `press-handstand/straddle-stand` | recommended | 0,6 | Hechten/Hüftbeuge-Momente unterscheiden gute Presser | [A-17] B |
| `planche/straddle` | `planche-press/straddle` | prerequisite | 1,0 | Press beginnt in der Straddle Planche | [A-29 S. 29] B |
| `planche/advanced-tuck` | `maltese/lean` | prerequisite | 1,0 | PDF: Lean Maltese auf der Straddle-Planche-Stufe | [P-01 S. 2] C (H-PRE) |
| `planche/straddle` | `maltese/wide-planche` | prerequisite | 1,0 | Wide Planche ist eine Planche-Variante | (H-PRE); [P-02 S. 1] |
| `planche-press/full` | `maltese/press` | prerequisite | 1,0 | Maltese-Press mit Band erst ab Full-Planche-Press | [P-02 S. 2, 4] C (H-PRE) |
| `planche/full` | `maltese/hold` | recommended | 0,7 | OG: Planche 11 < Maltese 17; FIG: Ringe-Maltese D > Planche C | [A-31] C; [A-29] B |
| `dip/parallel-bars` | `muscle-up/bar-kipping` | prerequisite | 1,0 | MU = Klimmzug + Dip | [A-13] B (H-PRE) |
| `pull-up/strict-10` | `muscle-up/bar-kipping` | recommended | 0,6 | Zugkraft für den Übergang | (H-PRE) |
| `muscle-up/bar-kipping` | `muscle-up/rings` | recommended | 0,5 | Stange zuerst lernen | [A-13] B |
| `support-hold/rings`, `dip/rings` | `muscle-up/rings` | prerequisite | 1,0 | Endposition und Stützphase an Ringen | (H-PRE) |
| `l-sit/full` | `muscle-up/l-sit` | prerequisite | 1,0 | L-Position über die ganze Bewegung | (H-PRE) |
| `handstand/wall` | `hspu/wall-headstand` | prerequisite | 1,0 | HSPU an der Wand setzt Wand-HS voraus | (H-PRE) |
| `handstand/free-30s` | `hspu/free-headstand` | prerequisite | 1,0 | frei drücken setzt frei balancieren voraus | (H-PRE) |
| `hspu/wall-full` | `dip/rings`, `push-up/rings` | recommended | 0,3 | Handstand/HSPU überträgt auf Liegestütz und Dips | [A-30 S. 21] C |
| `pull-up/strict-10` | `one-arm-pull-up/archer`, `weighted-pull-up/bw-118` | prerequisite | 1,0 | Basiskraft | (H-PRE) |
| `weighted-pull-up/bw-150` | `one-arm-pull-up/chin` | recommended | 0,5 | gleiche Leistungshöhe liegt auf gleichem OG-Level (OAC 9 ≈ 1,9× KG) | [A-31, A-30 S. 22] C |
| `dip/parallel-bars` | `weighted-dip/bw-120` | prerequisite | 1,0 | Basiskraft | (H-PRE) |
| `l-sit/straddle` | `v-sit/45` | prerequisite | 1,0 | OG-Reihenfolge 4 → 6 | [A-31] C |
| `v-sit/170` | `manna/full` | prerequisite | 1,0 | OG-Reihenfolge 12 → 13 | [A-31] C |
| `hollow-body/full`, `hang-foundation/arch-hang` | `bar-swing/tap-swing` | prerequisite | 1,0 | Schwung = Wechsel Hollow/Arch im Hang | (H-PRE) |
| `bar-swing/hip-circle` | `bar-swing/release-180` | prerequisite | 1,0 | Stangenkontrolle vor Releases | (H-PRE) |
| `planche/*` | `front-lever/*` (gleiche Stufe) | antagonist | 0,5 | Druck- vs. Zugmuster; bestehender Platzhalter-Link | (H-PRE) |
| `handstand/*`, `hspu/*` | `pull-up/*` | antagonist | 0,3 | vertikales Drücken vs. Ziehen | (H-PRE) |
| `push-up-knee` | `push-up/incline` | alternative | 0,5 | ähnliche Last (49 % vs. 41–55 % KG) | [A-05] B |
| `chin-up` | `pull-up/strict-*` | alternative | 0,5 | gleiche Rekrutierung, andere Gewichtung | [A-08, A-09] B |
| `front-lever-half-lay` | `front-lever/one-leg` | alternative | 0,5 | gleiches OG-Level | [A-31] C |
| `planche-one-leg` | `planche/half-lay` | alternative | 0,5 | gleiches OG-Level | [A-31] C |
| `muscle-up-rings-no-false-grip` | `muscle-up/bar-strict` | alternative | 0,4 | OG 6 vs. 7 | [A-31] C |

## 9. Klärung der PDF-Kürzel (aus `01_pdf_extract.md` §3)

Daï-Long Huynhs eigene Kanäle waren nicht erreichbar; es gibt deshalb **keine
D-Quelle** vom Autor selbst. Die Tabelle nutzt externe Hinweise und den
PDF-Kontext.

| Kürzel | Befund | Deutung | Sicherheit |
|---|---|---|---|
| `supi floor`, `supi bar` | Die Schwalbe an Ringen wird mit supinierten (90° aussenrotierten) Händen ausgeführt [A-21]; FIG nennt «ausgedrehte Hände» als wertneutrale Variante der Stützwaage [A-29 S. 62]; die PDFs nutzen `supi` bei Leans, Pseudo-Liegestützen, Wide-Planche- und Maltese-Übungen [P-01 bis P-04] | supinierte/aussenrotierte Handstellung (Finger nach aussen/hinten) am Boden bzw. an einer Stange; als Brücke zur Maltese plausibel | **mittel–hoch** (gestützt, im PDF nicht erklärt) |
| `fake supi bar` | kein externer Befund; einmalig bei Straddle-Planche-Liegestützen [P-03 S. 3] | eine nachgeahmte supinierte Stellung an einer Stange? | **unklar** |
| `neck band` | kein externer Befund; stets als Alternative zu `elastic` bei schweren Varianten [P-01 S. 2–3; P-02; P-03] | Band als Assistenz, anders befestigt (Nacken/oberer Rücken?) | **unklar** |
| `ZANETTI (vertical, elastic)` | nicht im Text des FIG CoP 2025–2028 gefunden (Elementnamen der Ringe-Seiten geprüft) [A-29]; im PDF 3–8 Wdh. mit Band in Maltese-Einheiten [P-03 S. 1, 4] | dynamisches Element der Maltese-Familie, wohl nach einer Person benannt (nicht verifiziert) | **unklar** |
| `DEAD PLANCHE` (Push-ups, Hold) | kein externer Befund; im PDF auf Position 1 (Maximalintensität) mit Band, 2–4 Wdh. bzw. 5–10 s [P-03 S. 3–4] | eine Variante, die schwerer ist als Full-Planche-Liegestütze | **unklar** |
| `ELEVATOR` (Lean Maltese / Maltese) | OG führt «(L17) Elevator» als Element jenseits von Level 16 in der Muscle-up-Spalte, ohne Definition [A-31]; im PDF in Wiederholungen dosiert (2–8) [P-02 S. 4; P-03 S. 4; P-04 S. 1] | der Begriff ist als Elementname etabliert; im PDF wohl eine dynamische Auf-ab-Bewegung in der Maltese-Position (H) | **unsicher** |
| `WIDE` (Straddle/Planche) | FIG: «leicht weite» Handstellung ist bei der Stützwaage wertneutral [A-29 S. 62]; im PDF v. a. in Maltese-Einheiten [P-02 S. 4; P-03 S. 4] | weite Handstellung als Maltese-Vorstufe; bei «Wide Straddle» könnte zusätzlich die Beinweite gemeint sein | **mittel** |
| `KICKS` | kein externer Befund | Einschwingen in die Position, Sekunden = Haltezeit nach dem Kick? | **unsicher** (wie in 01) |
| `BADFORM` | FIG definiert Formfehler über Winkelabweichungen [A-29 S. 19–20] | bewusst unsaubere Form (Hüfte/Arme); als niedrige `form_quality` loggen | **naheliegend** |

## Parameter für den Algorithmus

| Param-ID | Parameter (key, English snake_case) | Wert/Spanne | Einheit | Quelle(n) | Evidenz | Anmerkung |
|---|---|---|---|---|---|---|
| PAR-A-01 | `dynamic_work_sets` | 3 | Sätze | [A-44] | D | RR-Standard für Grundübungen |
| PAR-A-02 | `dynamic_work_rep_range` | 5–8 | Wdh. | [A-44] | D | schwerste saubere Variante |
| PAR-A-03 | `dynamic_advance_threshold_reps` | 8 (in allen 3 Sätzen, gute Form) | Wdh. | [A-44] | D | dann nächste Variante mit 3×5 beginnen |
| PAR-A-04 | `dynamic_restart_reps_after_advance` | 5 | Wdh. | [A-44] | D | |
| PAR-A-05 | `dynamic_rep_increment_per_session` | +1 | Wdh. pro Satz und Einheit | [A-44, A-46] | D | «Vorwerte schlagen»; ergibt ≥ 4 Einheiten pro Variante (PAR-A-39) |
| PAR-A-06 | `basic_iso_hold_range_s` | 10–30 | s | [A-44] | D | Support-Halt, Tuck FL in der Row-Leiter |
| PAR-A-07 | `basic_iso_advance_hold_s` | 30 (in allen 3 Sätzen) | s | [A-44] | D | Wechsel zur nächsten Stufe |
| PAR-A-08 | `lever_advance_hold_s` | 10 (3 Sätze, ohne Gelenkschmerz) | s | [A-40] | D | TMA-«10-Sekunden-Regel»; Studienbeleg dort behauptet, nicht angegeben |
| PAR-A-09 | `lever_advance_sets_x_hold_gmb` | 5 × 20 | Sätze × s | [A-35] | C | GMB-Alternative zu PAR-A-08 (strenger) |
| PAR-A-10 | `lever_intermediate_unlock_hold_s` | 10 | s | [A-40]; Heuristik | Heuristik | Unlock für Tuck/Adv Tuck/One-Leg; Wechselschwelle PAR-A-08 als Unlock übernommen |
| PAR-A-11 | `terminal_hold_unlock_s` | 3 | s | [A-33] | C | strengster Wettkampfstandard; FIG und Calisthenics Cup 2 s (PAR-A-12) |
| PAR-A-12 | `competition_min_hold_s` | FIG 2 · Calisthenics Cup 2 · WSWCF 3 | s | [A-29 S. 20; A-34; A-33 S. 5] | B | ab vollständigem Stillstand |
| PAR-A-13 | `fig_hold_deviation_bands_deg` | ≤ 5 abzugsfrei · > 5–20 klein · > 20–45 mittel · > 45 nicht anerkannt | ° | [A-29 S. 19–22] | B | Haltepositionen; Abzüge 0,1/0,3/0,5 |
| PAR-A-14 | `fig_arm_bend_bands_deg` | 0–15 klein · > 15–30 mittel · > 30–45 gross · > 45 nicht anerkannt | ° | [A-29 S. 19] | B | Halte und Pressen |
| PAR-A-15 | `wswcf_max_deviation_deg` | 15 (Planche ±7,5 zur Horizontalen) | ° | [A-33 S. 5] | C | strenger als FIG |
| PAR-A-16 | `form_quality_angle_map` | 5: ≤ 5° · 4: > 5–15° · 3: > 15–30° · 2: > 30–45° · 1: > 45° | ° | [A-29 S. 19–20; A-33 S. 5] | Heuristik | Zuordnung eigene; Grenzen aus FIG/WSWCF |
| PAR-A-17 | `unlock_min_form_quality_statics` | 4 | 1–5 | Heuristik | Heuristik | entspricht ≤ 15° (PAR-A-16) |
| PAR-A-18 | `unlock_occurrences_default` | 2 (dynamisch ≈ 3×8: 3) | Vorkommen | ADR 0003 §5; [A-44] | Heuristik | Projektvorgabe ADR 0003; zählt Sätze, nicht Tage (§2.5) |
| PAR-A-19 | `unlock_within_days_statics` | 28 | Tage | Heuristik | Heuristik | ≈ 4 Wochen: genug Einheiten für 2 Vorkommen bei 2–3 Einheiten/Woche |
| PAR-A-20 | `unlock_within_days_dynamic` | 7 | Tage | Heuristik | Heuristik | 3 Sätze ≥ 8 innerhalb einer Woche ≈ «3×8» |
| PAR-A-21 | `counts_for_unlock` | nur unassistiert, voller Weg, nicht exzentrisch, nicht gescheitert; Kipping nur für Kipping-Slugs | — | `evaluate.go`; [A-10, A-13] | Heuristik | Assistierte Sätze zählen für die Last (01 §4.4), nie für Unlocks |
| PAR-A-22 | `difficulty_tier_from_og_level` | ⌈OG-Level × 10 / 16⌉; Wurzeln = 1 | Tier 1–10 | [A-31]; Heuristik | Heuristik | lineare Abbildung auf das Schemafeld |
| PAR-A-23 | `og_level_bands` | Beginner 1–5 · Intermediate 6–9 · Advanced 10–13 · Elite 14–16 | OG-Level | [A-30 S. 22] | C | Einstufung nach Können, nicht Trainingsjahren [A-30 S. 23] |
| PAR-A-24 | `og_fig_quartiles` | Basic 1–4 · A 5–8 · B 9–12 · C 13–16 | OG-Level | [A-30 S. 22; A-31] | C | stimmt mit FIG für FL, BL, Straddle PL, Ringe-PL, Manna überein (§3.3) |
| PAR-A-25 | `og_level_planche_floor` | Frog 3 · SA Frog 4 · Tuck 5 · Adv Tuck 6 · Straddle 8 · Half-Lay/One-Leg 9 · Full 11 | OG-Level | [A-31] | C | Boden/Barren |
| PAR-A-26 | `og_level_planche_rings` | Frog 4 · SA Frog 5 · Tuck 6 · Adv Tuck 8 · Straddle 10 · Half-Lay 12 · Full 14 | OG-Level | [A-31] | C | |
| PAR-A-27 | `rings_level_offset` | Planche +1…+3; Liegestütz +3 (1 → 4); Dip +1 (3 → 4); L-Sit +2 (3 → 5); HS +2…+3 | OG-Level | [A-31] | C | Ersatzlogik Ringe ↔ Boden/Barren; FIG sieht für die Planche keinen Unterschied (Widersprüche) |
| PAR-A-28 | `og_level_front_lever` | Tuck 4 · Adv Tuck 5 · Straddle 6 · Half-Lay/One-Leg 7 · Full 8 | OG-Level | [A-31] | C | FIG A [A-29] |
| PAR-A-29 | `og_level_back_lever` | German Hang 1 · Skin the Cat 2 · Tuck 3 · Adv Tuck 4 · Straddle 5 · Half-Lay/One-Leg 6 · Full 7 | OG-Level | [A-31] | C | FIG A [A-29] |
| PAR-A-30 | `og_level_handstand_hspu` | Wand-HS 1–3 · frei 4–5 · Ring-HS 7 · One-Arm-HS 10; Pike-HeSPU 1 · Box 2 · Wand exz. 3 · Wand-HeSPU 4 · Wand-HSPU 5 · frei HeSPU 6 · frei HSPU 7 | OG-Level | [A-31] | C | |
| PAR-A-31 | `og_level_l_v_manna` | Tuck-L 1 · 1-Bein 2 · L 3 · Straddle-L 4 · RTO-L 5 · V45 6 · V75 7 · V100 8 · V120 9 · V140 10 · V155 11 · V170 12 · Manna 13 | OG-Level | [A-31] | C | FIG: L A, V B, Manna C [A-29] |
| PAR-A-32 | `og_level_pull_oac` | Sprung 1 · exz. 2 · Klimmzug 3 · L 4 · Pullover 5; Ringe: L 4 · breit 5 · breit-L 6 · Archer 7 · OAC exz. 8 · OAC 9 · OAC+6,8 kg 10 · OAC+11,3 kg 11 | OG-Level | [A-31] | C | 15/25 lb umgerechnet |
| PAR-A-33 | `og_weighted_pull_up_total_bw` | L3 1,00 · L4 1,18 · L5 1,35 · L6 1,50 · L7 1,65 · L8 1,78 · L9 1,90 · L10 2,00 · L11 2,10 | × KG | [A-31] | C | Wiederholungszahl nicht angegeben |
| PAR-A-34 | `og_weighted_dip_total_bw` | L3 1,00 · L4 1,20 · L5 1,38 · L6 1,55 · L7 1,70 · L8 1,85 · L9 2,00 · L10 2,13 · L11 2,25 | × KG | [A-31] | C | Wiederholungszahl nicht angegeben |
| PAR-A-35 | `pull_up_1rm_total_bw_reference` | Männer 1,16 ± 0,15 · Frauen 0,73 ± 0,09 | × KG | [A-11] | B | Studierende; 1RM mit Zusatz- oder Gegengewicht |
| PAR-A-36 | `og_level_mu_flag_pistol` | MU: exz. 3 · Kipping 4 · MU 5 · ohne FG 6 · strikt Stange 7 · L-Sit-MU 8; Flagge: Tuck 5 · Adv 6 · Straddle 7 · Full 8; Pistol 4 (mit 1,2× KG 5 … 2,0× KG 11) | OG-Level | [A-31] | C | |
| PAR-A-37 | `og_level_push_dip_row` | Liegestütz: Standard 1 · Diamond 2 · Ringe 4 · RTO 5 · RTO-PPPU 40° 7; Dips: Barren 3 · L 4 · 45° 5; Ring-Dips: Stütz 1 · RTO-Stütz 2 · exz. 3 · Dip 4; Rudern: exz. 1 · Ringe 2 · breit 3 · Archer 4 · einarmig 7 | OG-Level | [A-31] | C | |
| PAR-A-38 | `maltese_og_level` | 17 (jenseits des Charts) | OG-Level | [A-31] | C | FIG: Boden C = Planche, Ringe D [A-29] |
| PAR-A-39 | `min_sessions_per_dynamic_variant` | 4 | Einheiten | [A-44] (abgeleitet) | D | 5→6→7→8 Wdh.; Untergrenze |
| PAR-A-40 | `pushup_load_fraction_bw` | Hände 61 cm 0,41 · Knie 0,49 · Hände 30 cm 0,55 · Standard 0,64 · Füsse 30 cm 0,70 · Füsse 61 cm 0,74 | × KG | [A-05] | B | Spitzen-GRF; statisch 0,69–0,75 bzw. 0,54–0,62 [A-06] *(S)* |
| PAR-A-41 | `rings_conditioning_benchmark_pct_bw` | Schwalbe kon. 63,05 / exz. 94,10 · Stützwaage 60,37 / 86,79 · Kreuz-HS 56,66 / 70,86 | % KG | [A-21] | B | Konditionierungsmessung am Seilzuggerät; Elite-Turner |
| PAR-A-42 | `swallow_supine_1rm_benchmark_pct_bw` | Schwalbe 73,4 · Stützwaage 67,4 | % KG | [A-20] | B | 1RM «Schwalbe in Rückenlage»; n = 10 |
| PAR-A-43 | `strength_explains_skill_r2` | Schwalbe 0,76–0,85 · Stützwaage 0,42–0,59 · Kreuz-HS 0,38–0,48 (2021) bzw. 0,60 (2025) · Bankdrücken–Kreuz r 0,41 | R² / r | [A-21, A-22, A-23] | B | Kraftbaselines nur `recommended` |
| PAR-A-44 | `elite_specific_strength_gain` | +3,6–4,1 % in 4 Wo.; +8,3–8,7 % in 3 Wo. | % | [A-26, A-27] | B | Schwalbe/Stützwaage, exzentrisches Spezialtraining |
| PAR-A-45 | `est_weeks_per_og_level_step` | Ziel-Level ≤ 4: 2–8 · 5–8: 4–13 · 9–12: 8–26 · ≥ 13: 13–52 | Wochen pro Level | Heuristik, geeicht an [A-41, A-42, A-40] | Heuristik | FL Tuck→Full ergibt 16–52 Wo. (TMA 7–10+ Mo.); BL 16–55 Wo. (TMA 6–12 Mo.); Planche ab 0 ergibt 48–162 Wo. (TMA 24–36 Mo., also eher obere Hälfte) |
| PAR-A-46 | `coach_time_front_lever_months` | Fundament→Tuck 3–4 · Tuck→Adv 2–3 · Adv→Straddle 2–3 · Straddle→Full 3–4 · Full 3–6; gesamt 12–18 ab 10 Klimmzügen | Monate | [A-41] | D | Praxisindiz |
| PAR-A-47 | `coach_time_planche_cumulative_months` | Lean 0–2 · Tuck 2–6 · Adv Tuck 6–12 · Straddle 12–24 · Full 24–36 | Monate ab Start | [A-40] | D | Praxisindiz; «Full in 1–3 Jahren» |
| PAR-A-48 | `coach_time_back_lever_weeks` | Vorbereitung 4–8 · Tuck/Pike 8–12 · Full 12–24; je Stufe 4–8 | Wochen | [A-42] | D | gesamt 6–12 Monate |
| PAR-A-49 | `coach_time_first_ring_dip_weeks` | 4–12 (Ø 6–8) | Wochen | [A-43] | D | ab 30 s Ring-Stütz und 15 Barren-Dips |
| PAR-A-50 | `coach_time_first_pull_up_months` | 2–6 | Monate | [A-46] | D | ab Dead Hang |
| PAR-A-51 | `goal_realism_min_weeks` | Σ Untergrenzen aus PAR-A-45 vom aktuellen zum Ziel-Level; z. B. Anfänger → Full Planche ≥ 48 Wo. | Wochen | PAR-A-45; [A-40] | Heuristik | Persona 5 («Full Planche in 8 Wochen») klar unrealistisch; Hinweis ohne Wertung (ADR 0003) |
| PAR-A-52 | `prereq_front_lever_entry` | 10 strikte Klimmzüge · 30 s Dead Hang · 60 s Hollow · 15 gestreckte Beinheben | Wdh./s | [A-41] | D | als `recommended`-Kanten, nicht hart |
| PAR-A-53 | `prereq_planche_entry` | 60 s Hollow · 3 × 20 Liegestütze · 3 min Handgelenk-Routine · 30 s Wand-HS; Leans ab 30 s Plank | Wdh./s/min | [A-40, A-35] | D/C | als `recommended`-Kanten |
| PAR-A-54 | `prereq_back_lever_pull_ups` | 8–10 | Wdh. | [A-42] | D | `recommended` |
| PAR-A-55 | `handstand_free_hold_targets_s` | 10 (für viele Ziele genug) · ~60 (vor One-Arm-HS) | s | [A-37] | C | Stufen `free-10s`, `free-60s` |
| PAR-A-56 | `l_sit_one_leg_before_full_s` | 5 | s je Seite | [A-36] | C | Halte allgemein 5–30 s, 3–5 Sätze |
| PAR-A-57 | `handstand_wrist_flexor_nrms_by_apparatus` | Boden 61 % · Barren 44 % · Ringe 46 % | % NRMS | [A-16] | B | Ersatzlogik bei Handgelenkbeschwerden: Barren/Parallettes (Detail Stream C/D) |
| PAR-A-58 | `handstand_wrist_strategy_share` | > 75 % der Zeit; gemischt ~2 % | % | [A-15] | B | Formhinweis «über Finger/Handgelenk korrigieren» |
| PAR-A-59 | `skill_practice_frequency_coach` | HS 2–4/Wo · L-Sit 2–4/Wo · Planche ≤ 3/Wo (+1 nach 2 Mo.) · FL 2–3/Wo · BL ≤ 3–4/Wo · Handgelenk-Routine 2–3/Wo | Einheiten/Woche | [A-37, A-36, A-35, A-41, A-42, A-39] | C/D | gehört inhaltlich zu Stream B; hier als Coaching-Konsens |
| PAR-A-60 | `release_elements_auto_unlock` | false (nur Selbstbestätigung) | — | [A-29 S. 6; A-33 S. 6]; Heuristik | Heuristik | Matten/Helfer selbst im Spitzenturnen; App kann Sicherheit nicht prüfen |
| PAR-A-61 | `release_valid_catch` | beidhändig, ohne Bodenkontakt | — | [A-33 S. 6] | C | Formkriterium Releases |
| PAR-A-62 | `edge_weight_defaults` | prerequisite 1,0 · recommended 0,3–0,7 · alternative 0,5 · antagonist 0,5 | Gewicht | Heuristik | Heuristik | Kraftbaselines nie `prerequisite` (PAR-A-43) |
