# 07 — Leistungsdiagnostik

> Stream F der Phase-2-Recherche. Thema: Wie der Planer die Ausgangsleistung
> eines Users erfasst (Selbstauskunft, optionaler Testtag) und aus den Logs
> nachkalibriert. Enthält Messtheorie, Kandidatentests mit Reliabilitätszahlen,
> Protokolle, Umrechnungen zwischen Übungsvarianten, Prädiktoren für
> Skill-Bereitschaft, eine 2-Stufen-Testbatterie und eine Skizze des
> Konfidenzmodells.
>
> **Status: Teilrecherche.** Das Websuche-Budget der Session (200 Aufrufe,
> geteilt mit den parallel laufenden Streams) war nach 17 Suchen dieses Streams
> erschöpft; Direktzugriffe auf PubMed, PMC, Verlags- und DOI-Seiten sind durch
> die Egress-Policy blockiert. Alle 16 zitierten Quellen wurden über
> Suchmaschinen-Auszüge ihrer Abstract- bzw. Landingpages geprüft, **kein
> Volltext**. Themen ohne geprüfte Quelle sind als **Lücke** markiert, Zahlen
> ohne Quelle als **Praxisheuristik** bzw. **Heuristik**. Die Lücken sind in
> «Offene Fragen» als Auftrag für eine Nachrecherche gelistet.

## Kurzfassung

- **Evidenzbasis:** 16 Quellen (6 × Evidenz A, 10 × Evidenz B), dazu die
  Praxisquellen P-01 bis P-04. Gut belegt sind Kraft- und Rumpftests sowie die
  Last von Liegestütz-Varianten; Mobilitätstests, Genauigkeit der
  Selbstauskunft, Screening-Fragebögen und 1RM-Formeln sind **Lücken**
  [F-01, F-05, F-06].
- **Wiederholungstests sind reproduzierbar, aber absolut unscharf:** Der
  Standardmessfehler (SEM) maximaler Wiederholungen liegt bei 0.7–1.1
  Wiederholungen; daraus folgt eine kleinste sicher erkennbare Veränderung
  (MDC95) von rund 2–3 Wiederholungen (eigene Rechnung) [F-09] (Evidenz B).
- **Maximalkrafttests (1RM) sind sehr reliabel:** median ICC 0.97, median
  CV 4.2 % über alle Übungen, unabhängig von Trainingserfahrung und Zahl der
  Gewöhnungseinheiten [F-05] (Evidenz A). Gewichteter Klimmzug und Dip:
  ICC 0.96–0.99, kleinste relevante Veränderung 3 % bzw. 4 % [F-08].
- **Liegestütz-Test:** Test-Retest-Reliabilität .90–.95; die Übereinstimmung
  zwischen Bewertern lag in einer Teilstudie bei nur .75–.88, in einer
  weiteren bei .95–.99 [F-07]. Die Bewertung ist also eine eigene
  Fehlerquelle; Selbsttests brauchen scharfe Formkriterien (Folgerung,
  Praxisheuristik).
- **Körpergewichtstests messen relative Kraft:** Feldtests (Klimmzug,
  Hängen mit gebeugten Armen, Liegestütz) korrelierten nur mit Kraft pro
  Körpergewicht, nicht mit absoluter Kraft [F-12]. Testwerte gehören immer
  zusammen mit dem Körpergewicht gespeichert.
- **Haltetests:** Unterarmstütz (Plank) und Seitstütz haben nur *moderate*
  Reliabilität (starke Evidenz) [F-01]. Der Handstand-Halt war bei jungen
  Turnerinnen hoch reliabel (ICC bis 1) und korrelierte mit dem
  Leistungsniveau (r² 0.65) [F-15, F-16]; ob das für Anfänger gilt, ist offen.
- **Mobilität:** Sit-and-Reach und Toe-Touch sind hoch reliabel (starke
  Evidenz) [F-01]; für Schulterflexion, Handgelenk, Sprunggelenk und
  Brustwirbelsäule fehlen in diesem Durchgang geprüfte Zahlen (Lücke).
- **Umrechnung Liegestütz:** Standard 64–75 % des Körpergewichts, Knie
  49–62 %, Hände 30.5/61 cm erhöht 55/41 %, Füsse 30.5/61 cm erhöht 70/74 %
  [F-13, F-14].
- **Umrechnung Zug/Druck:** Klimmzug-1RM (Körper + Zusatzlast) ist rund
  1.25–1.33 × Latzug-1RM [F-11]; Dip-1RM rund 1.1 × Klimmzug-1RM [F-08]
  (eigene Rechnung aus den Mittelwerten).
- **1RM aus Wiederholungen** stimmt im Gruppenmittel (Abweichung ≤ 1.3 kg),
  streut aber individuell so stark, dass die Methoden nicht austauschbar sind
  [F-10]. Umrechnungen taugen nur als Startwert mit grosser Unsicherheit.
- **Skill-Bereitschaft:** Sportartspezifische Turntests korrelieren mit dem
  Wettkampfniveau (r² 0.52–0.86) [F-15, F-16]. Belastbare Schwellen für Muscle-up,
  Front Lever oder Planche wurden nicht gefunden (Lücke); die Planche-Niveaus
  der Praxisquellen liefern Einstufungsanker auf Evidenz C [P-01 S. 1–3,
  P-02 S. 1].
- **Konfidenzmodell:** Ein eindimensionaler Kalman-Filter pro Übung und
  Messgrösse verbindet Selbstauskunft, Test und Logs. Der Messfehler von Tests
  ist belegt [F-09]; der Fehler von Selbstauskunft und RIR-basierter Schätzung
  ist hier **Heuristik**.
- **Stabilisierung:** Bei einem SEM von 1.1 Wiederholungen braucht es mindestens
  3 unabhängige Beobachtungen für ein 95-%-Intervall von ±1.5 Wiederholungen
  und 5 für ±1 Wiederholung (eigene Rechnung aus [F-09]).

## 1. Recherchestand und Grenzen

| Thema aus dem Auftrag | Stand | Quellen |
|---|---|---|
| Reliabilität Kraft-Feldtests (Liegestütz, Klimmzug, Dip, 1RM) | teilweise belegt | F-05, F-07, F-08, F-09, F-10 |
| Validität von Feldtests | nur Übersichtsebene | F-02, F-04, F-12 |
| Isometrische Haltetests (Plank, Seitstütz, Handstand) | teilweise belegt | F-01, F-15, F-16 |
| Hollow Hold, L-Sit, Stütz, Pistol Squat | **Lücke** | – |
| Mobilität (Schulter, Handgelenk, Sprunggelenk, BWS, Smartphone) | **Lücke**, ausser Sit-and-Reach/Toe-Touch | F-01 |
| Prädiktoren für Skills | Turnen belegt, Calisthenics **Lücke** | F-15, F-16, P-01–P-03 |
| Genauigkeit der Selbstauskunft, RIR-Genauigkeit | **Lücke** (RIR: Stream B) | – |
| Last in % Körpergewicht, Umrechnungen | Liegestütz gut, Zug/Dip teilweise | F-06, F-08, F-11, F-13, F-14 |
| Wiederholungen → 1RM | Validität der Methode belegt, Formeln **Lücke** | F-10 |
| Screening-Fragebogen (PAR-Q+), Trainingsalter | **Lücke** | – |
| Statistik der Nachkalibrierung | Messfehler belegt, Verfahren **Heuristik** | F-05, F-09 |

Verweise auf andere Streams (dort recherchiert, hier nicht zitiert):
RPE/RIR-Genauigkeit und Detraining → Stream B (`03_training_methods.md`);
frühere Verletzung als Risikofaktor und Red Flags → Stream D
(`05_injuries_prehab.md`); Voraussetzungen zwischen Skills → Stream A
(`02_skills_progressions.md`); Reihenfolge und Ermüdung in der Einheit →
Stream E (`06_cns_motor_learning.md`).

## 2. Messtheorie: wie viel ein Testwert aussagt

### 2.1 Begriffe

| Grösse | Bedeutung | Rechenregel | Beleg |
|---|---|---|---|
| ICC | relative Reliabilität: Anteil der Streuung zwischen Personen an der Gesamtstreuung | – | F-05, F-09 |
| SEM | absoluter Messfehler in der Einheit des Tests (Wdh., s, kg) | SEM = SD × √(1 − ICC) | Standarddefinition der Messtheorie; Quellenbeleg in Nachrecherche (**Heuristik**-Status) |
| MDC95 | kleinste Veränderung zwischen zwei Messungen, die mit 95 % kein Messfehler ist | MDC95 = 1.96 × √2 × SEM | wie oben |
| SWC | kleinste praktisch relevante Veränderung | studienspezifisch | F-08 |
| Objektivität | Übereinstimmung zwischen Bewertern | ICC/Korrelation zwischen Bewertern | F-07 |

Ein hoher ICC bedeutet nicht, dass der Einzelwert genau ist. Bei
Wiederholungstests war die relative Reliabilität bei 70 % 1RM besser als bei
90 % 1RM (ICC 0.86 vs. 0.65), der absolute Fehler aber bei 90 % kleiner
(SEM 0.7 vs. 1.1 Wiederholungen) [F-09]. Für den Planer zählt der absolute
Fehler, weil er einzelne User einstuft, nicht Gruppen vergleicht.

### 2.2 Belegte Fehlergrössen

| Test | Relative Reliabilität | Absoluter Fehler | Stichprobe | Quelle | Evidenz |
|---|---|---|---|---|---|
| 1RM, alle Übungen | ICC 0.64–0.99, median 0.97; 92 % der ICC ≥ 0.90 | CV 0.5–12.1 %, median 4.2 % | Systematic Review | F-05 | A |
| 1RM, Oberkörper | median ICC 0.98 | CV 1.0–7.9 %, median 4.1 % | Systematic Review | F-05 | A |
| 1RM gewichteter Klimmzug / Dip | ICC 0.96–0.99 | SWC 3 % (Klimmzug), 4 % (Dip), relativ zum Körpergewicht | 15 Männer, Freizeit bis international, 7 Tage Abstand | F-08 | B |
| Maximale Wdh. bei 70 % 1RM | ICC 0.86 [0.71; 0.93] | SEM 1.1 Wdh. [0.8; 1.4] | Test-Retest, Bayes-Analyse | F-09 | B |
| Maximale Wdh. bei 90 % 1RM | ICC 0.65 [0.39; 0.83] | SEM 0.7 Wdh. [0.5; 0.9] | wie oben | F-09 | B |
| 1RM-Vorhersage (Wdh. bis Versagen, Last-Geschwindigkeit, isometrische Maximalkraft) | ICC «praktisch perfekt» | CV meist 2.3–4.4 %, SEM ~1–2 kg; isometrisch Bank 8.3 % / 4.2 kg | 28 Freizeitsportler, Smith-Maschine | F-10 | B |
| Liegestütz (revidiertes Protokoll) | Stabilität .90/.93 (Frauen), .95/.95 (Männer) | – | 152 Studierende, 2 Tage | F-07 | B |
| Liegestütz, Bewerter-Übereinstimmung | .75 (Frauen), .88 (Männer) in Teilstudie 2; .95–.99 in Teilstudie 3 (Grund des Unterschieds im Abstract nicht genannt) | – | 80 bzw. 152 Studierende | F-07 | B |
| Unterarmstütz, Seitstütz | moderat (starke Evidenz) | – | Systematic Review, 19–64 J. | F-01 | A |
| Sit-up, Partial Curl-up | moderat bis hoch (moderate Evidenz) | – | wie oben | F-01 | A |
| Handkraft, Rückenkraft, Sit-and-Reach, Toe-Touch | hoch (starke Evidenz) | – | wie oben | F-01 | A |
| GFMT (Turntest-Batterie, 10 Tests) | Gesamt ICC 0.97 (1 Woche); Einzeltests 0.75–0.97; Handstand 1; Spagat 0.998 | – | junge Turnerinnen | F-15, F-16 | B |

### 2.3 Abgeleitete Zahlen (eigene Rechnung)

| Ableitung | Rechnung | Ergebnis | Grundlage |
|---|---|---|---|
| MDC95 bei ~70 % 1RM (Sätze um 10 Wdh.) | 1.96 × √2 × 1.1 | **3.0 Wdh.** | F-09 |
| MDC95 bei ~90 % 1RM (Sätze um 3 Wdh.) | 1.96 × √2 × 0.7 | **1.9 Wdh.** | F-09 |
| MDC95 eines 1RM-Tests, falls CV ≈ typischer Fehler | 1.96 × √2 × 4.2 % | **≈ 11.6 %** | F-05 (Näherung) |
| SWC gewichteter Klimmzug, absolut | 3 % × 105.9 kg Systemmasse | ≈ 3.2 kg | F-08 |
| Beobachtungen für 95-%-Intervall ±1 Wdh. | n ≥ (1.96 × 1.1 / 1)² = 4.6 | **5** | F-09 |
| Beobachtungen für ±1.5 Wdh. | n ≥ (1.96 × 1.1 / 1.5)² = 2.1 | **3** | F-09 |

**Folgerungen für den Planer:**

1. Ein Unterschied von 1–2 Wiederholungen zwischen zwei Tests bei Sätzen um
   10 Wiederholungen ist nicht von Messfehler zu unterscheiden [F-09]. Der
   Planer darf daraus weder Fortschritt noch Rückschritt ableiten.
2. Die Zahlen aus F-09 stammen aus Krafttraining mit externer Last, nicht aus
   Körpergewichtsübungen. Ihre Übertragung auf Klimmzüge oder Liegestütze ist
   **Heuristik**: Es gibt in diesem Durchgang keine geprüfte SEM-Angabe für
   Wiederholungstests mit Körpergewicht bei Erwachsenen.
3. Gewöhnungseinheiten veränderten die Reliabilität von 1RM-Tests nicht
   [F-05]. Ein Pflicht-Probetest vor dem eigentlichen Test lässt sich damit
   nicht begründen.

## 3. Kandidatentests (Deliverable 1)

Spalten «Protokoll», «Zeit» und «Equipment» sind **Praxisheuristik**
(Standardbeschreibung, keine geprüfte Quelle), ausser wo eine Quelle steht.
«Selbsttest» meint: ohne zweite Person und ohne Spezialgerät durchführbar.

### 3.1 Kraft-, Halte- und Sprungtests

| Test (Slug) | Protokoll & Formkriterien (Kurzform) | Reliabilität / Validität | Zeit | Equipment | Selbsttest |
|---|---|---|---|---|---|
| `push_up_max` | Hände schulterbreit, Körper gerade; gültig: Ellbogen ≥ 90° gebeugt und oben voll gestreckt; Ende bei erster ungültiger Wdh. | Stabilität .90–.95; Bewerter-Übereinstimmung .75–.99 je nach Teilstudie [F-07]; Last 64–75 % KG [F-13, F-14] | 3 min | keins | ja, Video empfohlen |
| `knee_push_up_max`, `incline_push_up_max` | wie oben, Knie bzw. Hände erhöht | Reliabilität: **Lücke**; Last Knie 49–62 %, Hände erhöht 41–55 % KG [F-13, F-14] | 3 min | Box/Bank | ja |
| `pull_up_max` | Obergriff, Start im Hang mit gestreckten Armen, Kinn über Stange, kein Schwung; Ende bei erster ungültiger Wdh. | Wdh.-Test Erwachsene: **Lücke**; SEM-Analogie ~1 Wdh. (übertragen, Heuristik) [F-09]; misst relative Kraft [F-12] | 3 min | Stange | ja |
| `chin_up_max` | wie oben, Untergriff | **Lücke** | 3 min | Stange | ja |
| `dip_max` | Barren, Tiefe: Oberarm mindestens parallel zum Boden, oben gestreckt | Wdh.-Test: **Lücke** | 3 min | Barren | ja |
| `weighted_pull_up_1rm`, `weighted_dip_1rm` | steigende Zusatzlast bis zur schwersten sauberen Wdh. | ICC 0.96–0.99; SWC 3 % / 4 % [F-08] (B); 1RM allgemein median ICC 0.97 [F-05] (A) | 15–20 min | Gürtel + Gewichte | bedingt, nur Fortgeschrittene |
| `flexed_arm_hang`, `dead_hang` | Halt mit Kinn über Stange bzw. mit gestreckten Armen; Ende bei Absinken bzw. Loslassen | Feldtests inkl. Hängen korrelierten bei Kindern nur mit relativer Kraft [F-12]; Reliabilität bei Erwachsenen: **Lücke** | 2 min | Stange | ja |
| `plank_hold` | Unterarmstütz, Körperlinie; Ende bei Verlassen der Linie | moderat reliabel (starke Evidenz) [F-01] | 2–4 min | keins | ja |
| `side_plank_hold` | Seitstütz je Seite | moderat reliabel (starke Evidenz) [F-01] | 3–5 min | keins | ja |
| `hollow_body_hold` | Rückenlage, LWS am Boden, Arme/Beine gestreckt; Ende bei Abheben der LWS | **Lücke** | 2 min | keins | ja, schwer selbst zu bewerten |
| `handstand_hold_wall`, `handstand_hold_free` | Wand (Bauch zur Wand) bzw. frei; Ende bei Abstieg/Fussabsetzen | GFMT-Handstand ICC 1, r² 0.65 mit Niveau (junge Turnerinnen) [F-15, F-16] | 2–3 min | Wand | ja |
| `l_sit_hold` | Stütz auf Boden/Parallettes, Beine gestreckt waagrecht; Ende bei Beinsenken/Bodenkontakt | **Lücke** (vermutlich keine Daten; nicht recherchiert) | 2 min | Parallettes optional | ja |
| `support_hold` | Stütz am Barren/Ringen mit gestreckten Armen | **Lücke** | 2 min | Barren/Ringe | ja |
| `hanging_pike` / Beinheben im Hang | Beine gestreckt zur Stange bzw. über Hüfthöhe | GFMT: r² 0.86 mit Wettkampfniveau (Turnerinnen) [F-15, F-16] | 2 min | Stange | ja |
| Skill-Halte (`tuck_planche_hold`, `tuck_front_lever_hold` …) | beste saubere Haltezeit der aktuellen Stufe | Reliabilität: **Lücke**; Praxisquellen stufen Niveaus über solche Halte ein [P-01 S. 1–3, P-02 S. 1] (C) | je 3 min | je nach Skill | ja |
| `pistol_squat_max` | einbeinig, volle Tiefe, Ferse am Boden | **Lücke** | 3 min | keins | ja |
| `standing_broad_jump`, `vertical_jump` | Weitsprung aus dem Stand bzw. Strecksprung | Standweitsprung valide für Muskelfitness (Jugend) [F-04]; Strecksprung Teil der GFMT [F-15, F-16] | 3 min | Massband / App | ja |
| `handgrip` | Handdynamometer | hoch reliabel (starke Evidenz) [F-01]; valide (Jugend) [F-04] | 2 min | Dynamometer | nein |

### 3.2 Mobilitätstests

| Test (Slug) | Protokoll (Kurzform) | Reliabilität / Validität | Zeit | Equipment | Selbsttest |
|---|---|---|---|---|---|
| `sit_and_reach` / `toe_touch` (Pike-Proxy) | Langsitz bzw. Stand, Beine gestreckt, Reichweite in cm | hoch reliabel (starke Evidenz) [F-01]; Validität für Hamstring/LWS: **Lücke** | 2 min | Box/Lineal | ja |
| `split_front` / `split_side` | Abstand Hüfte–Boden | GFMT-Spagat ICC 0.998, r² 0.52 mit Niveau (Turnerinnen) [F-15, F-16] | 3 min | Massband | ja |
| `shoulder_flexion` (Wandtest, Goniometer, Smartphone) | Rücken an der Wand, gestreckte Arme über Kopf zur Wand | Schulterbeweglichkeit ist GFMT-Bestandteil [F-15, F-16]; Zahlen und Smartphone-Reliabilität: **Lücke** | 2 min | Wand / Smartphone | ja |
| `wrist_extension` | Handfläche flach, Unterarm nach vorn neigen | **Lücke** | 1 min | keins | ja |
| `ankle_dorsiflexion_wblt` | Ausfallschritt, Knie zur Wand, max. Fuss-Wand-Abstand | **Lücke** | 2 min | Wand, Massband | ja |
| `thoracic_extension` | – | **Lücke** | – | – | offen |

**Einordnung:** Für die Onboarding-Pflichtstufe taugen nur Tests, die ohne
Gerät, ohne zweite Person und in wenigen Minuten gehen. Bei allen Selbsttests
bewertet der User seine eigene Form. Wie stark das die Objektivität senkt, ist
nicht untersucht; die Liegestütz-Daten zeigen aber, dass die Übereinstimmung
zwischen zwei Fremdbewertern in einer Teilstudie nur .75–.88 erreichte
[F-07].

## 4. Standardisierte Protokolle für den Testtag

### 4.1 Allgemeine Regeln

| Nr. | Regel | Begründung | Beleg |
|---|---|---|---|
| G-1 | Jeder Test hat eine schriftliche Anleitung mit Start-, Gültigkeits- und Endkriterium und ein Beispielvideo. | Die Bewerter-Übereinstimmung beim Liegestütz schwankte zwischen Teilstudien von .75–.88 bis .95–.99 [F-07]; die Bewertung ist eine eigene Fehlerquelle. Dass Anleitung und Video sie senken, ist Praxisheuristik. | F-07 (B) / **Praxisheuristik** |
| G-2 | Test endet bei der **ersten ungültigen Wiederholung** bzw. beim ersten Verlassen der Halteposition (technisches Versagen), nicht beim Muskelversagen. | Die App belohnt keine Maximalversuche an unvorbereiteten Gelenken (ADR 0003); ein klares Endkriterium verringert Bewertungsspielraum. | **Praxisheuristik** |
| G-3 | Ein gewerteter Versuch pro Test und Tag. | Mehrere Versuche vermischen Leistung und Ermüdung; Mittelung über Tage ist statistisch sauberer (Abschnitt 8). | **Praxisheuristik** |
| G-4 | Vor jedem maximalen Kraft- oder Skilltest mindestens 5 min Pause, zwischen Tests für andere Muskelgruppen mindestens 3 min. | In allen Programm-Workouts der Praxisquellen haben Maximalversuche an der Zielstufe mindestens 5 min Pause [P-01 S. 1–3, P-02 S. 1–4, P-03 S. 1–4] (C). Die 3 min sind Heuristik. | P-01–P-03 (C) / **Heuristik** |
| G-5 | Reihenfolge: Mobilität → Skill-Halte (Handstand, Planche-/Lever-Stufe) → Maximalkraft mit wenigen Wdh. → Wiederholungstests → Rumpfhalte. | Die Praxisquellen stellen die intensivste, spezifischste Übung an Position 1 [P-01 S. 1–3, P-02 S. 1–4, P-03 S. 1–4] (C). Ob das für Tests gilt, prüft Stream E. | P-01–P-03 (C) / **Heuristik** |
| G-6 | Körpergewicht am Testtag erfassen und mit jedem Testwert speichern. | Körpergewichtstests messen relative Kraft [F-12]; Umrechnungen hängen am Körpergewicht [F-13, F-14]. | F-12, F-13, F-14 (B) |
| G-7 | Test-Retest-Abstand in der Validierung: 7 Tage; Verlaufstests frühestens nach 4 Wochen. | Die Reliabilitätsstudien testeten im Abstand von 7 Tagen bzw. einer Woche [F-08, F-15, F-16]. Die 4 Wochen sind Heuristik: Eine Veränderung muss die MDC (≈ 3 Wdh.) übersteigen, sonst ist der Retest wertlos (Abschnitt 2.3). | F-08, F-15, F-16 (B) / **Heuristik** |
| G-8 | Schmerz während eines Tests beendet diesen Test; der Planer markiert die Region und verweist gemäss Stream D. Keine Diagnose. | Grundregel des Projekts (keine medizinischen Aussagen). | **Praxisheuristik** |
| G-9 | Standardisiertes Aufwärmen vor dem Testtag. | Inhalt und Dauer: Stream B. | Verweis Stream B |
| G-10 | Testsätze werden als `set_entry` mit `kind = test` geloggt, mit `form_quality`, `failed` und Körpergewicht der Session. | Das Log kennt `kind = test` bereits (`codebase_notes.md` §2); kein zweiter Codepfad. | Projektvorgabe |

### 4.2 Protokolle je Test

Alle Formkriterien in dieser Tabelle sind **Praxisheuristik** (übliche
Testbeschreibungen, in diesem Durchgang nicht gegen eine Quelle geprüft).
Gemeinsam ist allen: Ende nach G-2, ein Versuch nach G-3.

| Test | Start | Gültige Wiederholung / gültiger Halt | Ende | Erfasster Wert |
|---|---|---|---|---|
| `push_up_max` | Hochstütz, Hände schulterbreit, Körper gerade von Kopf bis Ferse | Ellbogen mindestens 90° gebeugt, oben volle Streckung, Hüfte in Linie | erste Wdh. ohne Tiefe oder Streckung, Hüfte sackt, Pause > 2 s | Wdh. (`reps`) |
| `pull_up_max` | Hang mit gestreckten Armen, Obergriff schulterbreit | Kinn über Stange, unten volle Streckung, kein Schwung, kein Beinschlag | erste Wdh. ohne Kinn über Stange oder ohne Streckung | Wdh. |
| `dip_max` | Stütz mit gestreckten Armen | Oberarm mindestens parallel zum Boden, oben volle Streckung | erste Wdh. ohne Tiefe oder Streckung | Wdh. |
| `handstand_hold_wall` | Bauch zur Wand, Hände 10–20 cm von der Wand | Arme gestreckt, Körper gerade | Abstieg oder Beugen der Arme | Sekunden (`hold_seconds`) |
| `handstand_hold_free` | Aufschwingen frei | Arme gestreckt, keine Schritte mit den Händen | Fuss am Boden oder Handschritt | Sekunden |
| `l_sit_hold` | Stütz auf Boden oder Parallettes | Beine gestreckt, mindestens waagrecht, Arme gestreckt | Ferse oder Gesäss am Boden, Beine unter Waagrechte | Sekunden |
| `hollow_body_hold` | Rückenlage | LWS am Boden, Schulterblätter angehoben, Arme und Beine gestreckt | LWS hebt ab | Sekunden |
| `plank_hold` | Unterarmstütz | Körperlinie Kopf–Ferse | Hüfte sackt oder hebt deutlich | Sekunden |
| Skill-Halt (z. B. `tuck_planche_hold`) | Position der aktuellen Stufe | Stufenspezifische Formkriterien aus Stream A | Verlassen der Position | Sekunden |
| `sit_and_reach` / `toe_touch` | Langsitz bzw. Stand, Knie gestreckt | langsames Vorbeugen, 2 s halten | – | cm (Messgrösse im Log: offen, siehe Offene Fragen) |
| `shoulder_flexion` (Wandtest) | Rücken, Gesäss und LWS an der Wand | gestreckte Arme über Kopf Richtung Wand | – | Kategorie: Daumen erreichen Wand ja/nein |
| `wrist_extension` | Vierfüsslerstand, Handflächen flach | Unterarm nach vorn neigen, Ferse der Hand bleibt am Boden | – | Kategorie oder Winkel (Smartphone) |
| `ankle_dorsiflexion_wblt` | Ausfallschritt zur Wand | Knie berührt Wand, Ferse bleibt am Boden | – | cm Fuss–Wand |

Offene Punkte: Kadenz (metronomgeführt oder frei) ist für keinen Test geprüft;
die Messgrössen `cm` und «Kategorie» fehlen im heutigen `measure`-Vokabular
(`reps`/`hold_seconds`/`distance_m`/`none`, `codebase_notes.md` §2).

## 5. Kapazität schätzen und zwischen Varianten umrechnen

### 5.1 Last in Prozent des Körpergewichts (Liegestütz)

| Variante | Anteil Körpergewicht | Messart | Quelle |
|---|---|---|---|
| Standard-Liegestütz, oben / unten | 69.16 % / 75.04 % | statische Positionen, 28 krafttrainierte Männer | F-13 |
| Knie-Liegestütz, oben / unten | 53.56 % / 61.80 % | wie oben | F-13 |
| Standard-Liegestütz | 64 % | Spitzen-Bodenreaktionskraft, 23 Freizeitsportler | F-14 |
| Knie-Liegestütz | 49 % | wie oben | F-14 |
| Hände 30.5 cm erhöht | 55 % | wie oben | F-14 |
| Hände 61 cm erhöht | 41 % | wie oben | F-14 |
| Füsse 30.5 cm erhöht | 70 % | wie oben | F-14 |
| Füsse 61 cm erhöht | 74 % | wie oben | F-14 |

Verhältnis zur Standardvariante (eigene Rechnung aus [F-13, F-14]): Knie
0.77–0.82, Hände 30.5 cm erhöht 0.86, Hände 61 cm erhöht 0.64, Füsse
30.5 cm erhöht 1.09, Füsse 61 cm erhöht 1.16. Eine Übersicht mit kinetischen
Daten zu 46 Liegestütz-Varianten aus 26 Studien liegt vor [F-06] (Evidenz A);
ihre Einzelwerte konnten hier nicht geprüft werden.

**Nutzung:** Die Anteile ordnen Varianten nach Last und begründen
Plausibilitätsregeln (Abschnitt 8.7). Sie erlauben **keine** direkte
Umrechnung von Wiederholungszahlen zwischen Varianten, weil die
Wiederholungszahl bei gleicher relativer Last individuell stark streut [F-10].

### 5.2 Zug und Dip

| Grösse | Männer | Frauen | Quelle |
|---|---|---|---|
| Klimmzug-1RM (Körper + Zusatzlast) / Körpergewicht | 1.16 ± 0.15 | 0.73 ± 0.09 | F-11 (College-Studierende, n = 35 / 23) |
| Latzug-1RM / Körpergewicht | 0.93 ± 0.17 | 0.55 ± 0.11 | F-11 |
| Verhältnis Klimmzug-1RM / Latzug-1RM (eigene Rechnung) | ≈ 1.25 | ≈ 1.33 | F-11 |
| Klimmzug-1RM / Körpergewicht | 1.43 ± 0.15 | – | F-08 (15 Männer, Freizeit bis international) |
| Dip-1RM / Körpergewicht | 1.59 ± 0.23 | – | F-08 |
| Verhältnis Dip-1RM / Klimmzug-1RM (eigene Rechnung) | ≈ 1.11 | – | F-08 |

Maximalkraft liess sich aus Wiederholungen der analogen Übung bei Männern
besser vorhersagen als bei Frauen [F-11]. Der Mittelwert der Frauen unter
1.0 × Körpergewicht (0.73 ± 0.09) heisst, dass praktisch keine Teilnehmerin
einen unassistierten Klimmzug schaffte und der Klimmzug-1RM offenbar mit
Assistenz bestimmt wurde [F-11]; das Messverfahren selbst wurde nicht geprüft
(Ableitung aus den Mittelwerten). Das Verhältnis 1.33 für Frauen gilt deshalb
für assistierte Klimmzüge. Normwerte je Geschlecht und
Niveau fehlen in diesem Durchgang.

### 5.3 Wiederholungen → Maximalkraft

- Drei Verfahren zur 1RM-Schätzung (Wiederholungen bis Versagen,
  Last-Geschwindigkeit, isometrische Maximalkraft) lagen im Gruppenmittel
  höchstens 1.3 kg neben dem echten 1RM; nur die Last-Geschwindigkeits-Methode
  unterschätzte das Bankdrücken um 5 kg [F-10] (Evidenz B).
- Die individuellen Abweichungen waren so gross, dass die Verfahren **nicht
  austauschbar** sind [F-10].
- Die Wiederholungszahl bei fester relativer Last ist über kurze Zeit gut
  reproduzierbar (SEM 0.7–1.1 Wdh.) [F-09].
- Konkrete Formeln (z. B. Epley, Brzycki) und ihre Gültigkeit bei hohen
  Wiederholungszahlen: **Lücke**. Stream B enthält voraussichtlich die
  Wiederholungen-pro-%1RM-Beziehung.

**Folgerung (Heuristik):** Der Planer rechnet Wiederholungen nicht über
Übungsgrenzen um, um Dosierungen festzulegen. Er dosiert jede Übung aus ihren
eigenen Logs. Umrechnungen dienen nur als Startwert (Prior) mit grosser
Unsicherheit, wenn eine Übung noch nie geloggt wurde.

## 6. Prädiktoren für Skill-Bereitschaft

| Befund | Wert | Population | Quelle | Evidenz |
|---|---|---|---|---|
| Gesamtscore sportartspezifischer Turntests vs. Wettkampfniveau | r² 0.62 | junge Turnerinnen | F-15, F-16 | B |
| Einzeltests vs. Niveau: Beinheben zur Stange, Handstand, Spagat | r² 0.86 / 0.65 / 0.52 | wie oben | F-15, F-16 | B |
| Feldtests für den Oberkörper spiegeln relative, nicht absolute Kraft | signifikant nur relativ zum Körpergewicht (p < .01) | 9–10-Jährige | F-12 | B |
| Valide Muskelfitness-Tests laut Übersicht | Handkraft, Standweitsprung | Kinder und Jugendliche | F-04 | A |
| Planche-Niveau «Zero to Beginner» | Tuck Planche 5–10 s (auch mit Band) | Programmteilnehmer eines Elite-Athleten | P-01 S. 1 | C |
| Planche-Niveau «Beginner» | (Wide) Straddle Planche Hold 2–4 s (auch «badform»), Straddle Planche Press 1–3 Wdh. | wie oben | P-01 S. 2–3 | C |
| Planche-Niveau «Intermediate» | Wide Planche Hold 3–6 s, Full Planche Press 1–3 Wdh. | wie oben | P-02 S. 1–2 | C |
| Planche-Niveau «Advanced» | Full Planche Press 3–8 Wdh., Full Planche Push Ups 5–10 Wdh. | wie oben | P-03 S. 1–3 | C |

Die Planche-Zeilen beschreiben die Zielübungen der Programme je Niveau
(Ableitung in `01_pdf_extract.md` §4.7), keine geprüften Einstiegskriterien.

**Nicht gefunden (Lücke):** Studien oder geprüfte Coaching-Quellen, die
Klimmzug- oder Dipzahlen mit dem Muscle-up, relative Zugkraft mit dem Front
Lever oder Liegestütz-/Dip-Kapazität mit Planche-Fortschritt verknüpfen. In
der Coaching-Literatur existieren solche Faustregeln vermutlich (Evidenz C/D);
sie wurden hier nicht geprüft. Stream A liefert die Voraussetzungen zwischen
Skills.

**Folgerungen:**

1. Einstufung **pro Skill**, nicht global: Die Praxisquellen definieren Niveaus
   über Leistungen in den Zielübungen, und «Beginner» meint dort bereits eine
   solide Tuck Planche [P-01 S. 1–3] (C).
2. Sportartspezifische Tests sagen das Niveau besser voraus als allgemeine
   Tests [F-15, F-16] (B). Für die Einstufung zählt deshalb zuerst die beste
   saubere Leistung in der Skill-Stufe selbst, erst danach allgemeine
   Kraftwerte (**Heuristik**, abgeleitet).
3. Ohne belegte Schwellen setzt der Planer allgemeine Kraftwerte nur als
   **weiche** Bereitschaftsregel ein: Er empfiehlt eine Stufe, sperrt sie aber
   nicht. Harte Voraussetzungen bleiben den `prerequisite`-Kanten des
   Skill-Graphen vorbehalten (**Heuristik**).

## 7. Selbsteinschätzung, Screening und 2-Stufen-Batterie (Deliverable 2)

### 7.1 Was über Selbsteinschätzung belegt ist

| Frage | Stand | Folge |
|---|---|---|
| Wie genau schätzen User ihre maximalen Wiederholungen, Haltezeiten oder ihr Niveau? | **Lücke**: keine geprüfte Studie in diesem Durchgang | Fehlergrösse der Selbstauskunft ist Heuristik (PAR-F-20) |
| Wie valide sind Selbstauskünfte zu körperlicher Aktivität und Fitness allgemein? | **Lücke** | – |
| Wie genau ist die RIR-Schätzung, und hängt sie von der Erfahrung ab? | in diesem Stream **Lücke**; Stream B recherchiert RIR-Genauigkeit | Fehlergrösse RIR-basierter Log-Schätzungen ist Heuristik (PAR-F-24) |
| Wie stark hängt ein Testwert davon ab, wer bewertet? | belegt: Übereinstimmung zwischen Fremdbewertern .75–.88 in einer, .95–.99 in einer anderen Teilstudie [F-07]; Selbstbewertung nicht untersucht | Selbsttests brauchen scharfe Formkriterien; selbst bewertete Tests erhalten einen Fehlerzuschlag (PAR-F-22) |
| Messen Körpergewichtstests absolute oder relative Kraft? | relative [F-12] | Körpergewicht wird mit jedem Wert gespeichert (PAR-F-14) |

### 7.2 Screening und Vorgeschichte

- **Gesundheits-Vorabfragen:** Der Auftrag nennt PAR-Q+ als Kandidaten.
  Inhalt, Validierung und Lizenz wurden in diesem Durchgang **nicht geprüft
  (Lücke)**. Bis dahin sieht die Batterie einen Platzhalter-Block vor, dessen
  Fragen aus einer geprüften Quelle übernommen werden müssen, nicht selbst
  formuliert.
- **Beschwerden und Verletzungen:** Frühere Verletzungen als Risikofaktor und
  die Red-Flag-Liste recherchiert Stream D. Diese Batterie fragt nur ab, *wo*
  aktuell Beschwerden bestehen und *wo* in den letzten 12 Monaten eine
  Verletzung war; die Folgen (Kontraindikationen, Verweis an Fachpersonen)
  bestimmt Stream D. Der 12-Monats-Zeitraum ist **Heuristik**.
- **Trainingsalter:** Eine geprüfte Definition von Trainingsstatus fehlt
  (**Lücke**). Vorschlag (**Heuristik**, übliche Einteilung ohne geprüfte
  Quelle): `none` (< 3 Monate strukturiertes Krafttraining), `novice`
  (3–12 Monate), `intermediate` (1–3 Jahre), `advanced` (> 3 Jahre); getrennt
  gefragt für Krafttraining allgemein und für Calisthenics-Skills, plus
  «Trainingspause > 3 Monate im letzten Jahr» für den Wiedereinstieg
  (Persona 4, Detraining: Stream B).

### 7.3 Stufe 1: Pflicht-Selbstauskunft (< 5 min)

Zeitangaben je Frage sind **Praxisheuristik** (Schätzung für Antippen einer
Auswahl). Summe: 285 s ≈ 4.8 min, davon 215 s für die Stream-F-Fragen Q4–Q14
(eigene Rechnung).

| Nr. | Frage | Antwortformat | Speist | Zeit (s) | Beleg |
|---|---|---|---|---|---|
| Q1 | Ziel-Skills (bis 3, priorisiert) | Mehrfachauswahl | Planer | 30 | – |
| Q2 | Verfügbares Equipment | Mehrfachauswahl | Übungsauswahl | 20 | – |
| Q3 | Tage pro Woche, Minuten pro Einheit | Auswahl | Wochenstruktur | 20 | – |
| Q4 | Trainingsalter Kraft / Calisthenics; Pause > 3 Monate im letzten Jahr | Kategorien (7.2) | Startniveau, σ der Prior | 20 | **Heuristik** |
| Q5 | Gesundheits-Vorabfragen | ja/nein, Fragen aus geprüfter Quelle | Freigabe / Verweis | 45 | **Lücke** (7.2) |
| Q6 | Aktuelle Beschwerden: Schulter, Ellbogen, Handgelenk, Rücken, Knie, andere | ja/nein je Region | Stream-D-Regeln | 20 | Verweis Stream D |
| Q7 | Verletzung in den letzten 12 Monaten je Region | ja/nein je Region | Stream-D-Regeln | 20 | Verweis Stream D |
| Q8 | Körpergewicht, Grösse (optional) | Zahl | Normierung | 15 | F-12 |
| Q9 | Saubere Liegestütze am Stück | 0 · 1–3 · 4–7 · 8–12 · 13–20 · 21–30 · > 30 | Prior `push_up` | 10 | Klassen s. u. |
| Q10 | Saubere Klimmzüge am Stück | 0 · 1–3 · 4–7 · 8–12 · 13–20 · > 20 | Prior `pull_up` | 10 | Klassen s. u. |
| Q11 | Saubere Dips am Stück | 0 · 1–3 · 4–7 · 8–12 · 13–20 · > 20 | Prior `dip` | 10 | Klassen s. u. |
| Q12 | Handstand | keiner · Wand < 30 s · Wand ≥ 30 s · frei < 10 s · frei ≥ 10 s | Prior Handstand | 10 | **Heuristik** |
| Q13 | Aktuelle Stufe je Ziel-Skill (Bildauswahl) und Haltezeit-Klasse | Auswahl | Einstufung pro Skill | 45 | P-01 S. 1–3 (C) |
| Q14 | Wie sicher sind diese Zahlen? | geschätzt · in den letzten 4 Wochen gezählt · gefilmt | σ-Faktor | 10 | **Heuristik** |

**Klassenbreite (Q9–Q11):** Bei Sätzen um 10 Wiederholungen liegt die MDC95
eines echten Tests bei rund 3 Wiederholungen [F-09] (eigene Rechnung).
Antwortklassen, die schmaler als dieser Messfehler sind, täuschen Genauigkeit
vor; die Klassen oben sind deshalb im Bereich ab 4 Wiederholungen mindestens
4 Wiederholungen breit (**Heuristik**, abgeleitet).

### 7.4 Stufe 2: optionaler standardisierter Testtag

Reihenfolge nach G-5, Pausen nach G-4, Endkriterium nach G-2. Dauern sind
**Praxisheuristik**; Summe ohne 1RM-Block ≈ 48 min (eigene Rechnung).

| Block | Tests | Dauer (min) | Pause | Bedingung |
|---|---|---|---|---|
| 0 Aufwärmen | Inhalt aus Stream B | 10 | – | immer |
| 1 Mobilität | `sit_and_reach` oder `toe_touch`, `shoulder_flexion`, `wrist_extension`, `ankle_dorsiflexion_wblt` | 6 | keine | immer; nur Sit-and-Reach/Toe-Touch mit belegter Reliabilität [F-01] |
| 2 Skill-Halte | `handstand_hold_wall` oder `_free`; aktuelle Stufe von bis zu 2 Ziel-Skills | 15 | ≥ 5 min | nur wenn Q12/Q13 eine Stufe angeben |
| 3 Wiederholungen | `push_up_max` (oder Knie-/erhöhte Variante), `pull_up_max`, `dip_max` | 12 | ≥ 3 min | je nach Equipment (Q2) |
| 4 Rumpf | `hollow_body_hold` oder `l_sit_hold`, `plank_hold` | 5 | ≥ 2 min | immer |
| 5 Maximalkraft (eigener Tag) | `weighted_pull_up_1rm`, `weighted_dip_1rm` | 30–40 | ≥ 5 min | nur `advanced` und ≥ 12 Klimmzüge (**Heuristik**) |

**Alternative ohne Testtag (Heuristik):** Die Tests aus Block 2–3 laufen als
erster Satz (`kind = test`) in den ersten beiden regulären Einheiten. Das
kostet keinen Extratag und liefert nach Abschnitt 8.5 bereits zwei
unabhängige Beobachtungen.

## 8. Konfidenzmodell und Nachkalibrierung (Deliverable 3)

### 8.1 Zustand

Pro User, Übung, Messgrösse (Wiederholungen, Haltezeit oder 1RM-Last) und
Assistenzklasse führt der Planer eine Schätzung der aktuellen Kapazität
(maximale saubere Wiederholungen, Haltezeit oder 1RM) als Normalverteilung
mit Mittelwert μ, Standardabweichung σ und Zeitpunkt der letzten Evidenz. Das
Modell ist ein eindimensionaler Kalman-Filter (**Heuristik**: Standardverfahren
der Zustandsschätzung; eine Validierung für Trainingsdaten wurde nicht
geprüft).

### 8.2 Evidenzarten und ihre Fehler

| Evidenz | Beobachtung x | Fehler σ | Beleg |
|---|---|---|---|
| Selbstauskunft (Klasse) | Klassenmitte | max(2 Wdh., 0.30 × μ); × 0.75, wenn «in den letzten 4 Wochen gezählt» | **Heuristik**: kein geprüfter Beleg zur Genauigkeit der Selbstauskunft |
| Selbstauskunft «gefilmt» | angegebener Wert | wie selbst bewerteter Test | **Heuristik** |
| Test, selbst bewertet, Wdh. um 10 | Testwert | 1.1 × 1.25 ≈ 1.4 Wdh. | SEM [F-09] (B); Zuschlag 1.25 **Heuristik**, Richtung aus [F-07] |
| Test, selbst bewertet, ≤ 5 Wdh. | Testwert | 0.7 × 1.25 ≈ 0.9 Wdh. | wie oben |
| Test, Haltezeit | Testwert | 0.15 × μ | **Heuristik**; Rumpfhalte nur moderat reliabel [F-01] |
| Test, 1RM mit Zusatzlast | Testwert | 4.2 % × 1.25 der Systemmasse | CV [F-05] (A); Zuschlag **Heuristik** |
| Log-Satz mit RIR | reps + RIR | 1.5 Wdh. | **Heuristik**; RIR-Fehler aus Stream B übernehmen, sobald belegt |
| Log-Satz bis Versagen (`failed` oder RIR 0) | reps | wie Test | **Heuristik** |
| Log-Satz ohne RIR | untere Schranke: Kapazität ≥ reps | Update nur, wenn reps > μ, dann wie Test | **Heuristik** |
| Umrechnung aus anderer Variante | nach Abschnitt 5 | mindestens 0.35 × μ | Richtung [F-10] (individuelle Streuung), Betrag **Heuristik** |
| Assistiert, partiell, nur exzentrisch | – | zählt nicht für die unassistierte Kapazität | analog zur Unlock-Engine (`codebase_notes.md` §4); **Heuristik** |

### 8.3 Aktualisierung

1. **Vorhersage** vor jeder neuen Evidenz: σ² ← σ² + q² × Δt (Δt in Wochen),
   q = 0.5 Wdh. pro Woche (**Heuristik**; Fortschritts- und Detrainingsraten
   aus Stream B können q später je Trainingsalter setzen).
2. **Gewicht** der neuen Beobachtung: K = σ² / (σ² + r²), mit r = Fehler der
   Evidenzart aus 8.2.
3. **Update:** μ ← μ + K × (x − μ); σ² ← (1 − K) × σ².
4. **Widerspruch** (|x − μ| > 2 × √(σ² + r²)): kein stilles Überschreiben; der
   Planer nimmt bis zur Klärung den niedrigeren Wert mit dem grösseren σ und
   schlägt einen Testsatz vor (**Heuristik**, konservativ im Sinne von
   ADR 0003).

### 8.4 Rechenbeispiele (eigene Rechnung)

| Fall | Eingang | Ergebnis |
|---|---|---|
| Selbstauskunft 10 Klimmzüge (σ = 0.30 × 10 = 3), dann selbst bewerteter Test 8 (σ 1.4) | K = 9 / (9 + 1.89) = 0.83 | μ = 8.3, σ = 1.25 |
| wie oben, Test fremd bewertet (σ 1.1) | K = 9 / (9 + 1.21) = 0.88 | μ = 8.2, σ = 1.0 |
| Gleiche Selbstauskunft, dann Log-Satz 6 Wdh. mit RIR 2 (x = 8, σ 1.5) | K = 0.80 | μ = 8.4, σ = 1.3 |
| Laufender Betrieb, q = 0.5/Woche, eine Log-Evidenz pro Woche (r = 1.5) | stationäres K ≈ 0.28 | σ ≈ 0.8 Wdh. |
| wie oben, ein selbst bewerteter Testsatz pro Woche (r = 1.4) | stationäres K ≈ 0.30 | σ ≈ 0.76 Wdh. |

Die Beispiele zeigen: Ein einziger sauberer Test überschreibt eine
Selbstauskunft grösstenteils (K 0.83–0.88); im laufenden Betrieb verschiebt
eine einzelne Einheit die Schätzung nur um rund 28–30 % der Abweichung. Das
ist die gewünschte Trägheit gegen Tagesform.

### 8.5 Wie viele Beobachtungen eine Schätzung braucht

Mit SEM = 1.1 Wiederholungen [F-09] und unabhängigen Beobachtungen gleicher
Güte gilt für das 95-%-Intervall des Mittelwerts ±1.96 × 1.1 / √n (eigene
Rechnung):

| n | Halbbreite 95 % |
|---|---|
| 1 | ±2.2 Wdh. |
| 2 | ±1.5 Wdh. |
| 3 | ±1.2 Wdh. |
| 5 | ±1.0 Wdh. |

Unter der Annahme stabiler Kapazität reichen also 3 Beobachtungen für eine
Einstufung auf ±1.5 Wiederholungen und 5 für ±1 Wiederholung. Weil sich die
Kapazität zwischen Einheiten verändert, ist das eine Untergrenze.

### 8.6 Konfidenzklassen für Erklärungen

| Klasse | σ / μ | Verhalten des Planers | Beleg |
|---|---|---|---|
| hoch | < 0.10 | dosiert direkt aus μ | Anker: SEM eines Tests ≈ 10 % bei 10 Wdh. [F-09] |
| mittel | 0.10–0.25 | dosiert aus μ − 0.5 σ | **Heuristik** |
| niedrig | > 0.25 | dosiert aus μ − σ und schlägt Testsatz vor | **Heuristik** |

### 8.7 Plausibilitätsregeln (Persona 6)

| Regel | Prüfung | Beleg |
|---|---|---|
| R-1 Laststufen | Kapazität der leichteren Variante ≥ der schwereren: Knie ≥ Standard; Hände erhöht ≥ Standard; Standard ≥ Füsse erhöht. Die Reihenfolge Knie vs. Hände 30.5 cm erhöht ist zwischen den Studien nicht stabil (siehe Widersprüche) und wird nicht geprüft. | Lastreihenfolge [F-13, F-14] (B) |
| R-2 Stufenfolge | angegebene Skill-Stufe setzt die Vorstufen voraus; sonst Vorstufe erfragen | `prerequisite`-Kanten (Stream A); **Heuristik** |
| R-3 Trainingsalter | `none` und fortgeschrittene Skill-Stufe → Testsatz vorschlagen, σ erhöhen | **Heuristik** |
| R-4 Zug vs. Druck | Dip-1RM weit unter Klimmzug-1RM ist möglich, aber auffällig (Mittel 1.11 ×) | [F-08] (B), nur Hinweis, keine Sperre |
| R-5 Widerspruch | nie still den höheren Wert übernehmen (8.3, Schritt 4) | **Heuristik** |

### 8.8 Anbindung an das Datenmodell

- Evidenz aus Logs: nur Sessions mit `status = completed`; nur `set_entries`
  mit `kind` in (`working`, `test`); Elemente mit `assistance_class =
  unassisted`, nicht `is_partial_rom`, nicht `is_eccentric_only`
  (`codebase_notes.md` §2, §4). `form_quality` unter 3 zählt nicht
  (**Heuristik**).
- `rir` hängt am Satz, nicht am Element; bei Kombinationen (N Elemente) gilt
  das RIR nur für das letzte Element (**Heuristik**).
- Die Schätzung (μ, σ, Quelle, Zeitpunkt) braucht eine neue Tabelle im
  Nutzerprofil; `user_exercise_bests` ist ein Bestwert-Cache und kein Ersatz
  (`codebase_notes.md` §2). Festlegung in Phase 4.

## Parameter für den Algorithmus

| Param-ID | Parameter (key, English snake_case) | Wert/Spanne | Einheit | Quelle(n) | Evidenz | Anmerkung |
|---|---|---|---|---|---|---|
| PAR-F-01 | `rep_test_sem_reps_moderate_load` | 1.1 (0.8–1.4) | Wdh. | F-09 | B | Sätze bei ~70 % 1RM (um 10 Wdh.); ICC 0.86. Übertragung auf Körpergewichtsübungen ist Heuristik. |
| PAR-F-02 | `rep_test_sem_reps_heavy_load` | 0.7 (0.5–0.9) | Wdh. | F-09 | B | Sätze bei ~90 % 1RM (≤ 5 Wdh.); ICC nur 0.65. |
| PAR-F-03 | `rep_test_mdc95_reps` | 3.0 (um 10 Wdh.); 1.9 (≤ 5 Wdh.) | Wdh. | F-09 | B | Eigene Rechnung 1.96 × √2 × SEM. Kleinere Unterschiede zwischen zwei Tests nicht als Veränderung werten. |
| PAR-F-04 | `one_rm_test_cv_pct` | 4.2 (0.5–12.1); Oberkörper 4.1 | % | F-05 | A | Median über Studien. |
| PAR-F-05 | `one_rm_test_mdc95_pct` | ≈ 11.6 | % | F-05 | A | Eigene Rechnung; Näherung CV ≈ typischer Fehler. |
| PAR-F-06 | `weighted_pull_up_swc_pct` | 3 | % der relativen Kraft | F-08 | B | n = 15 Männer; ICC 0.96–0.99. |
| PAR-F-07 | `weighted_dip_swc_pct` | 4 | % der relativen Kraft | F-08 | B | wie oben. |
| PAR-F-08 | `push_up_bw_fraction` | 0.64–0.75 | Anteil Körpergewicht | F-13, F-14 | B | 0.64 dynamische Spitzenkraft [F-14]; 0.69 oben / 0.75 unten statisch [F-13]. |
| PAR-F-09 | `knee_push_up_bw_fraction` | 0.49–0.62 | Anteil KG | F-13, F-14 | B | 0.49 [F-14]; 0.54 oben / 0.62 unten [F-13]. |
| PAR-F-10 | `incline_push_up_bw_fraction` | 0.55 (Hände 30.5 cm); 0.41 (61 cm) | Anteil KG | F-14 | B | Andere Höhen: nicht interpolieren ohne Quelle. |
| PAR-F-11 | `decline_push_up_bw_fraction` | 0.70 (Füsse 30.5 cm); 0.74 (61 cm) | Anteil KG | F-14 | B | – |
| PAR-F-12 | `pull_up_to_lat_pull_1rm_ratio` | 1.25 (Männer); 1.33 (Frauen) | Verhältnis | F-11 | B | Eigene Rechnung aus Mittelwerten (Klimmzug-1RM inkl. Körpergewicht); Frauenwert bezieht sich auf assistierte Klimmzüge. |
| PAR-F-13 | `dip_to_pull_up_1rm_ratio` | 1.11 | Verhältnis | F-08 | B | Eigene Rechnung; nur Männer, n = 15; nur Plausibilitätshinweis. |
| PAR-F-14 | `store_bodyweight_with_test` | true | bool | F-12 | B | Körpergewichtstests messen relative Kraft. |
| PAR-F-15 | `field_test_reliability_class` | high: `handgrip`, `sit_and_reach`, `toe_touch`; moderate: `plank_hold`, `side_plank_hold` | Klasse | F-01 | A | Starke Evidenz für beide Einstufungen. |
| PAR-F-16 | `handstand_hold_reliability_icc` | ≈ 1.0 | ICC | F-15, F-16 | B | Nur junge Turnerinnen; Übertragung auf Anfänger offen. |
| PAR-F-17 | `familiarization_test_required` | false | bool | F-05 | A | Zahl der Gewöhnungseinheiten änderte die 1RM-Reliabilität nicht. |
| PAR-F-18 | `test_termination_rule` | `first_invalid_rep` | enum | – | Heuristik | Klares Endkriterium; keine Maximal-Grinds (ADR 0003). |
| PAR-F-19 | `test_attempts_per_day` | 1 | Versuche | – | Heuristik | Mittelung über Tage statt über Versuche (Abschnitt 8.5). |
| PAR-F-20 | `self_report_prior_sd` | max(2 Wdh., 0.30 × μ) | Wdh. | – | Heuristik | Kein geprüfter Beleg zur Genauigkeit der Selbstauskunft; bewusst breit, damit ein Test dominiert. |
| PAR-F-21 | `self_report_recent_count_sd_factor` | 0.75 | Faktor | – | Heuristik | Für «in den letzten 4 Wochen gezählt». |
| PAR-F-22 | `self_scored_test_sd_factor` | 1.25 | Faktor auf SEM | F-07 | Heuristik | Bewertung ist belegte Fehlerquelle (Übereinstimmung teils nur .75–.88); Selbstbewertung nicht untersucht, Betrag geschätzt. |
| PAR-F-23 | `hold_test_sd_frac` | 0.15 | Anteil von μ | F-01 | Heuristik | Keine SEM für Halte gefunden; Rumpfhalte nur moderat reliabel. |
| PAR-F-24 | `log_rir_set_obs_sd_reps` | 1.5 | Wdh. | – | Heuristik | Grösser als Test-SEM wegen RIR-Schätzfehler; durch Stream-B-Zahlen ersetzen. |
| PAR-F-25 | `converted_estimate_sd_frac` | ≥ 0.35 | Anteil von μ | F-10 | Heuristik | Individuelle Streuung der Umrechnung belegt, Betrag geschätzt. |
| PAR-F-26 | `capacity_process_sd_per_week` | 0.5 | Wdh./Woche | – | Heuristik | Lässt alte Evidenz verblassen; Stream B (Fortschritt, Detraining) soll kalibrieren. |
| PAR-F-27 | `min_obs_stable_estimate` | 3 (±1.5 Wdh.); 5 (±1 Wdh.) | Beobachtungen | F-09 | B | Eigene Rechnung, Untergrenze bei stabiler Kapazität. |
| PAR-F-28 | `confidence_class_cv_thresholds` | hoch < 0.10; mittel 0.10–0.25; niedrig > 0.25 | σ/μ | F-09 | Heuristik | 0.10 ≈ Test-SEM bei 10 Wdh.; übrige Grenzen geschätzt. |
| PAR-F-29 | `dose_from_estimate_offset_sd` | hoch 0; mittel 0.5; niedrig 1.0 | σ | – | Heuristik | Konservative Dosierung bei unsicherer Schätzung. |
| PAR-F-30 | `contradiction_threshold_sd` | 2 | kombinierte σ | – | Heuristik | Ab dieser Abweichung gilt Evidenz als widersprüchlich (8.3). |
| PAR-F-31 | `contradiction_resolution` | `take_lower_with_higher_sd` | enum | – | Heuristik | Konservativ im Sinne von ADR 0003. |
| PAR-F-32 | `rep_bucket_min_width_reps` | ≥ 3, im Bereich ab 4 Wdh. gewählt ≥ 4 | Wdh. | F-09 | Heuristik | Klassen schmaler als die MDC95 täuschen Genauigkeit vor. |
| PAR-F-33 | `onboarding_self_report_max_s` | 300 (Entwurf: 285) | s | – | Heuristik | Projektvorgabe < 5 min; Zeiten je Frage geschätzt. |
| PAR-F-34 | `rest_before_max_test_min` | 5 | min | P-01, P-02, P-03 | C | Praxis bei Maximalversuchen an der Zielstufe (alle Programm-Workouts). |
| PAR-F-35 | `rest_between_tests_other_group_min` | 3 | min | – | Heuristik | Tests anderer Muskelgruppen. |
| PAR-F-36 | `test_day_duration_min` | ≈ 48 (ohne 1RM-Block) | min | – | Heuristik | Summe der Blockschätzungen (7.4). |
| PAR-F-37 | `retest_min_interval_weeks` | 4 | Wochen | – | Heuristik | Erwartete Veränderung muss MDC übersteigen; Validierungsstudien nutzen 7 Tage [F-08, F-15, F-16]. |
| PAR-F-38 | `one_rm_test_eligibility` | `advanced` und ≥ 12 Klimmzüge | Bedingung | – | Heuristik | 1RM mit Zusatzlast nur für Geübte. |
| PAR-F-39 | `capacity_evidence_min_form_quality` | 3 | Skala 1–5 | – | Heuristik | Schlechte Form zählt nicht als Kapazitätsnachweis. |
| PAR-F-40 | `readiness_gate_mode` | `soft` | enum | F-15, F-16 | Heuristik | Keine belegten Schwellen für Calisthenics-Skills; sportartspezifische Leistung zählt vor allgemeinen Kraftwerten. |
| PAR-F-41 | `placement_scope` | `per_skill` | enum | P-01 | C | Niveaus der Praxisquellen sind skillspezifisch [P-01 S. 1–3]. |

## Widersprüche

1. **Last beim Liegestütz: 64 % oder 69–75 % des Körpergewichts?** Ebben et al.
   messen dynamisch die Spitzenkraft bei 23 Freizeitsportlern und finden 64 %
   [F-14]; Suprak et al. messen statisch in zwei Positionen bei 28
   kraftrainierten Männern und finden 69 % oben und 75 % unten [F-13]. Auch
   beim Knie-Liegestütz liegen die Werte auseinander (49 % vs. 54–62 %).
   Folge: Die Knie-Variante kann nach Suprak schwerer sein als die Variante
   mit 30.5 cm erhöhten Händen nach Ebben (62 % vs. 55 %); die Reihenfolge
   dieser beiden ist nicht stabil. PAR-F-08/09 führen deshalb Spannen.
2. **Relative vs. absolute Reliabilität bei Wiederholungstests:** Bei 70 % 1RM
   ist der ICC besser (0.86 vs. 0.65), bei 90 % 1RM der absolute Fehler
   kleiner (SEM 0.7 vs. 1.1 Wiederholungen) [F-09]. Wer nur ICC-Werte liest,
   zieht den falschen Schluss für Einzelpersonen.
3. **Gruppenmittel vs. Einzelperson bei der 1RM-Schätzung:** Die mittlere
   Abweichung ist trivial (≤ 1.3 kg), die individuellen Abweichungen sind so
   gross, dass die Methoden nicht austauschbar sind [F-10].
4. **Klimmzug-1RM relativ zum Körpergewicht:** 1.16 bei College-Männern
   [F-11] vs. 1.43 bei Männern von Freizeit- bis internationalem Niveau
   [F-08]. Eine einzige Norm gibt es nicht; die Werte hängen stark von der
   Population ab.
5. **GFMT-Reliabilität:** Für die Einzeltests wird eine Spanne von
   ICC 0.75–0.97 berichtet, zugleich ICC 1 für den Handstand und 0.998 für den
   Spagat [F-15, F-16]. Die Angaben stammen vermutlich aus verschiedenen
   Berichten (Kongress-Abstract vs. Artikel); ohne Volltext ist nicht zu
   klären, welche gilt.
6. **Validität von Oberkörper-Feldtests:** Die Jugend-Übersicht hebt nur
   Handkraft und Standweitsprung als valide hervor [F-04]; für den
   überarbeiteten Liegestütz-Test wird dagegen ausreichende Validitätsevidenz
   berichtet (logischer, Gruppenunterschieds- und Kriteriumsansatz) [F-07].
   Die Aussagen beruhen auf unterschiedlichen Kriterien und Populationen.
7. **Haltetests:** Rumpfhalte (Plank, Seitstütz) sind in der
   Erwachsenen-Übersicht nur moderat reliabel [F-01], der Handstand-Halt bei
   Turnerinnen dagegen nahezu perfekt [F-15, F-16]. Unterschiedliche
   Populationen (Allgemeinbevölkerung vs. trainierte Turnerinnen) und
   unterschiedliche Endkriterien (Ermüdung vs. Gleichgewichtsverlust) sind
   plausible Gründe; geprüft ist das nicht.
8. **Gewöhnung:** Die 1RM-Reliabilität war unabhängig von der Zahl der
   Gewöhnungseinheiten [F-05]. Das widerspricht der verbreiteten Praxis,
   einen ersten Test als «Probelauf» zu verwerfen; für diese Praxis wurde hier
   keine Quelle geprüft.

## Offene Fragen

**Nachrecherche (Suchbudget erschöpft; in dieser Reihenfolge):**

1. Reliabilität, SEM und MDC von maximalen Klimmzügen, Dips und Liegestützen
   mit Körpergewicht bei Erwachsenen; ersetzt die Übertragung aus [F-09] in
   PAR-F-01 bis PAR-F-03.
2. Mobilitätstests: Schulterflexion (Goniometer, Inklinometer, Smartphone,
   Wandtest), Handgelenkextension, Weight-Bearing Lunge Test (Systematic
   Review), Brustwirbelsäulen-Extension; Reliabilität selbst durchgeführter
   Smartphone-Messungen (Systematic Review).
3. Validität von Sit-and-Reach als Mass für Hamstring- und LWS-Beweglichkeit
   (Meta-Analyse), da Toe-Touch/Sit-and-Reach als Pike-Proxy dienen sollen.
4. Genauigkeit der Selbstauskunft: selbstberichtete vs. gemessene Fitness,
   Kraft und Wiederholungen; Übersichten zur Validität selbstberichteter
   körperlicher Aktivität; Unterschiede zwischen Anfängern und Erfahrenen.
   Ersetzt PAR-F-20/21.
5. Genauigkeit der RIR-Vorhersage (mit Stream B abgleichen); ersetzt
   PAR-F-24.
6. PAR-Q+: Inhalt, Validierung, Nutzungsbedingungen; Definitionen von
   Trainingsstatus/Trainingsalter; kurze Verletzungsanamnese (mit Stream D).
7. Wiederholungen → 1RM: Formeln und ihre Gültigkeit oberhalb von
   10 Wiederholungen; Wiederholungen je %1RM (mit Stream B).
8. Kraftmessplatten-Daten für Pike-Liegestütz, Dips, Klimmzug-Varianten und
   Band-Assistenz (mit Stream C).
9. Coaching-Schwellen (Evidenz C/D) für Muscle-up, Front Lever und Planche;
   Abgleich mit Stream A.
10. Reliabilität von Hollow Hold, L-Sit, Stütz, Pistol Squat und
    Skill-Halten; vermutlich keine Daten, das ist zu bestätigen.
11. Volltextprüfung der Parameterquellen F-05, F-08, F-09, F-13, F-14, F-15,
    F-16: vollständige Autorenliste der GFMT-Arbeiten, Jahrgang von F-08 und
    F-15, die
    Übung in F-09 und die Herkunft der GFMT-Einzelwerte (Widerspruch 5).
12. Als Titel gesichtet, Inhalt nicht geprüft: die Männer-Version der GFMT
    (IJSPT 2016), eine kanadische Validierung der GFMT (Physical Therapy in
    Sport), eine Übersicht zu Feldtests bei Turnern (Apunts 2022) und eine
    Übersicht zu Machbarkeit und Sicherheit von Feldtests (Sports Medicine –
    Open).

**Produkt- und Spezifikationsfragen (Phase 4):**

- Bleibt der Testtag vollständig optional, oder wird er für bestimmte Ziele
  empfohlen? Die Alternative «Testsatz in den ersten zwei Einheiten» (7.4)
  vermeidet einen Extratag.
- Sollen User Testvideos hochladen können, um die Bewertung zu verbessern
  (Objektivität, [F-07])? Das berührt Speicher und Datenschutz.
- Das `measure`-Vokabular kennt keine Zentimeter und keine Kategorien; für
  Mobilitätstests braucht es eine additive Erweiterung.
- Wo liegen μ, σ, Evidenzquelle und Zeitpunkt der Kapazitätsschätzungen?
  Vorschlag: neue Tabelle im Nutzerprofil, nicht `user_exercise_bests`.
- Wie wird ein Testtag nach ADR 0003 dargestellt, ohne Maximalversuche zu
  belohnen (keine XP für Testwerte, nur für das Absolvieren)?

## Quellen

| ID | Titel | Autor(en) | Jahr | URL/DOI | Typ | Evidenz |
|---|---|---|---|---|---|---|
| F-01 | Reliability of Field-Based Fitness Tests in Adults: A Systematic Review | Cuenca-Garcia M, Marin-Jimenez N, Perez-Bey A, et al. | 2022 | https://doi.org/10.1007/s40279-021-01635-2 | Systematic Review | A |
| F-02 | Criterion-Related Validity of Field-Based Fitness Tests in Adults: A Systematic Review | Castro-Piñero J, Marín-Jiménez N, Fernández-Santos JR, Martín-Acosta F, Segura-Jiménez V, Izquierdo-Gómez R, Ruiz JR, Cuenca-García M | 2021 | https://doi.org/10.3390/jcm10163743 | Systematic Review | A |
| F-03 | Reliability of field-based fitness tests in youth | Artero EG, España-Romero V, Castro-Piñero J, Ortega FB, Suni J, Castillo-Garzon MJ, et al. | 2011 | https://pubmed.ncbi.nlm.nih.gov/21165805/ | Systematic Review | A |
| F-04 | Criterion-related validity of field-based fitness tests in youth: a systematic review | Castro-Piñero J, Artero EG, España-Romero V, Ortega FB, Sjöström M, Suni J, Ruiz JR | 2010 | https://www.researchgate.net/publication/24276000_Criterion-related_validity_of_field-based_fitness_tests_in_youth_A_systematic_review | Systematic Review | A |
| F-05 | Test–Retest Reliability of the One-Repetition Maximum (1RM) Strength Assessment: a Systematic Review | Grgic J, Lazinica B, Schoenfeld BJ, Pedisic Z | 2020 | https://doi.org/10.1186/s40798-020-00260-z | Systematic Review | A |
| F-06 | Kinetic analysis of push-up exercises: a systematic review with practical recommendations | Dhahbi W, Chaabene H, Chaouachi A, Padulo J, Behm DG, Cochrane J, Burnett A, Chamari K | 2022 | https://doi.org/10.1080/14763141.2018.1512149 | Systematic Review | A |
| F-07 | Objectivity, Reliability, and Validity for a Revised Push-Up Test Protocol | Baumgartner TA, Oh S, Chung H, Hales D | 2002 | https://doi.org/10.1207/S15327841MPEE0604_2 | Querschnittstudie | B |
| F-08 | Reliability of pull up and dip maximal strength tests | Coyne JOC, Tran TT, et al. | 2015 | https://ro.ecu.edu.au/ecuworkspost2013/2422/ | Querschnittstudie | B |
| F-09 | Reproducibility of strength performance and strength-endurance profiles: A test-retest study | Mitter B, Csapo R, Bauer P, Tschan H, Gruet M | 2022 | https://doi.org/10.1371/journal.pone.0268074 | Querschnittstudie | B |
| F-10 | Validity and reliability of upper body push and pull tests to determine one-repetition maximum | Sigvaldsen E, Loturco I, Larsen F, Bruusgaard J, Kalhovde JM, Haugen T | 2023 | https://doi.org/10.1371/journal.pone.0288649 | Querschnittstudie | B |
| F-11 | Relationship of lat-pull repetitions and pull-ups to maximal lat-pull and pull-up strength in men and women | Johnson D, Lynch J, Nash K, Cygan J, Mayhew JL | 2009 | https://doi.org/10.1519/JSC.0b013e3181a2d7f5 | Querschnittstudie | B |
| F-12 | Validity of field tests of upper body muscular strength | Pate RR, et al. | 1993 | https://doi.org/10.1080/02701367.1993.10608774 | Querschnittstudie | B |
| F-13 | The effect of position on the percentage of body mass supported during traditional and modified push-up variants | Suprak DN, Dawes J, Stephenson MD | 2011 | https://doi.org/10.1519/JSC.0b013e3181bde2cf | Biomechanik-Studie | B |
| F-14 | Kinetic analysis of several variations of push-ups | Ebben WP, Wurm B, VanderZanden TL, Spadavecchia ML, Durocher JJ, Bickham CT, Petushek EJ | 2011 | https://doi.org/10.1519/JSC.0b013e31820c8587 | Biomechanik-Studie | B |
| F-15 | Measuring fitness in female gymnasts: the gymnastics functional measurement tool | Sleeper MD, et al. (Autorenliste nicht vollständig geprüft) | 2012 | https://www.researchgate.net/publication/224822330_Measuring_fitness_in_female_gymnasts_the_gymnastics_functional_measurement_tool | Querschnittstudie | B |
| F-16 | The Gymnastics Functional Measurement Tool: A Reliable and Valid means of Measuring Gymnastics Physical Abilities (Kongress-Abstract 1847) | Sleeper MD, et al. (Autorenliste nicht vollständig geprüft) | 2010 | https://journals.lww.com/acsm-msse/Fulltext/2010/05001/The_Gymnastics_Functional_Measurement_Tool__A.1255.aspx | Querschnittstudie | B |

Die Praxisquellen P-01 bis P-04 sind in `01_pdf_extract.md` beschrieben und
werden in `00_sources.md` geführt.
