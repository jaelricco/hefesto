# Onboarding-Design

> Phase 3. Legt fest, welche Daten das Onboarding erhebt, warum, in welcher
> Form, wie sie validiert werden und wie sie in den Planer einfliessen.
> Grundlage sind die Recherchedateien `docs/research/02`–`07` und die
> Synthese `08_synthesis.md`. Quellen-IDs (`A-…` bis `F-…`, `P-…`) und
> Parameter-IDs (`PAR-…`) verweisen auf diese Dateien; das Verzeichnis steht in
> `docs/research/00_sources.md`. Was nicht belegt ist, heisst hier
> **Heuristik** und trägt eine Begründung.
>
> Das Onboarding stellt keine Diagnosen. Es erfasst, was der User angibt, passt
> damit das Training an und verweist bei Warnzeichen an Fachpersonen
> (ADR 0003, `CONTENT_AUTHORING.md`).

## 1. Zweck und Grundsätze

Das Onboarding liefert dem Planer einen **Start-Zustand**, mit dem er eine
sichere erste Trainingswoche erzeugen kann. Alles Weitere lernt der Planer aus
den Logs. Daraus folgen acht Grundsätze:

| Nr. | Grundsatz | Begründung |
|---|---|---|
| O-1 | **Kurzer Pflichtteil (Richtwert < 5 min), der Rest wird aus den Logs gelernt.** Jede Pflichtfrage muss eine Entscheidung der ersten Woche verändern; alles andere wird später im Kontext gefragt («Profil verfeinern», §3.8). | Auftrag; die Kapazität ändert sich ohnehin, und mehrere Log-Beobachtungen tragen mehr Information als ein einzelner Messwert (eigene Rechnung, `07_assessment.md` §8.5); aus zwei Messzeitpunkten sind Veränderungen einer Person kaum erkennbar [F-18]. |
| O-2 | **Einstufung pro Skill und Bewegungsmuster, nicht global.** | Kraft erklärt Skill-Leistung nur teilweise (R² 0.38–0.85 je Element, Elite-Turner an Ringen; Kreuz-Handstand nicht signifikant) [A-21 = F-40, PAR-A-43]; «Beginner» in den PDFs heisst bereits solide Tuck-Planche (`01_pdf_extract.md` §4.7, H-8); OG stuft nach Können, nicht nach Trainingsjahren ein [A-30 S. 23, F-69]. |
| O-3 | **Jeder Wert trägt Herkunft und Unsicherheit.** Selbstauskunft ist ein breiter Startwert, kein Messwert. | Selbstauskünfte ordnen Personen, messen aber nicht: von US-Soldaten erinnerte Liegestützzahlen im Mittel +4–7 % zu hoch, Einzelabweichung SD ≈ 13–18 % [F-51, F-52]; Heimtests bei norwegischen Stellungspflichtigen κ ≤ 0.34 [F-53]; Selbstauskunft vs. Messung körperlicher Aktivität r −0.71 bis 0.96 [F-13]. |
| O-4 | **Sicherheit vor Plan.** Belastungssymptome, Beschwerden je Region und Red Flags kommen vor der Plan-Erzeugung. | Überlastung dominiert Calisthenics-Verletzungen (62 % in 12 Monaten verletzt, Tendinopathie häufigste Diagnose) [D-02]; Screening nach Symptomen und gewünschter Intensität [F-41]. |
| O-5 | **Bei Unsicherheit konservativ dosieren und submaximal kalibrieren.** Niedrige Konfidenz → Dosierung aus dem unteren Ende der Schätzung; Kalibrierung über einen submaximalen Satz mit RIR 2–3, nicht über einen Maximaltest. | `07` §8.6; ein Log-Satz mit RIR ≤ 3 und ≤ 12 Wdh. trägt mit r = 2 Wdh. dieselbe Messgenauigkeit wie ein Maximaltest (`07` §8.2, **Heuristik**); ADR 0003. |
| O-6 | **Keine Maximalversuche im Pflichtteil; Tests sind freiwillig und an Sicherheitsbedingungen gebunden** (§4.3). Tests bringen kein XP. | ADR 0003 §5; Testende bei der ersten ungültigen Wiederholung (G-2, PAR-F-18, **Heuristik**). |
| O-7 | **Widersprüche werden nachgefragt, nie still aufgelöst.** Im Zweifel gilt der niedrigere Wert mit grösserer Unsicherheit. | Plausibilitätsregeln R-1 bis R-8 (`07_assessment.md` §8.7); Persona 6. |
| O-8 | **Kein Ziel wird abgelehnt oder bewertet.** Unrealistische Termine werden mit einer Zeitspanne und einem Zwischenziel beantwortet. | ADR 0003 §3 (keine Aussage, dass jemand zurückliegt); Persona 5. |

## 2. Ablauf und Zeitbudget

```
Start
 └─ A Rahmen (Geburtsjahr, Einwilligung, Hinweis)                ~15 s
 └─ B Ziele (bis 3 Skills, Reihenfolge, optional Datum)          ~30 s
 └─ C Verfügbarkeit (Einheiten, Minuten)                         ~20 s
 └─ D Equipment                                                  ~20 s
 └─ E Körpergewicht & Trainingshintergrund                       ~30 s
 └─ F Leistungsstand (Grundübungen, Stufe je Ziel-Skill)         ~95 s
      └─ Zusatzfragen bei 0 Klimmzügen/Dips, Hollow, Mobilität   +10–40 s
 └─ G Gesundheit (Belastungssymptome, Vorabfragen, Körperkarte)  ~75 s
      └─ je Region mit Beschwerde: Details + Red Flags          +60–90 s je Region
 └─ Plausibilitätsprüfung (höchstens 2 Rückfragen)               0–30 s
 └─ Ergebnis: Start-Zustand, erster Plan, Angebot Testtag
```

| Block | Pflicht | Zeit (s) | Grundlage der Zeitschätzung (`07_assessment.md` §7.3; alle Werte **Praxisheuristik**) |
|---|---|---|---|
| A Rahmen | ja | 15 | zwei Bestätigungen, eine Zahl (eigene Schätzung) |
| B Ziele | ja | 30 | Q1 = 30 s |
| C Verfügbarkeit | ja | 20 | Q3 = 20 s |
| D Equipment | ja | 20 | Q2 = 20 s |
| E Körpergewicht & Hintergrund | ja | 30 | Q4 = 20 s (Niveau, Trainingsalter, Pause) + Körpergewicht 10 s (Q8 ohne Grösse) |
| F Leistungsstand | ja | 95 | Q9–Q11 = 30 s, Q12 = 10 s, Q13 = 45 s (nur Ziel-Skills), Q14 = 10 s |
| G Gesundheit | ja | 75 | Q5 = 45 s; Q6 + Q7 auf **einer** Körperkarte (aktuell / letzte 12 Monate) ≈ 30 s statt 40 s |
| **Basispfad** | | **≈ 285 s ≈ 4.75 min** | eigene Rechnung |
| Verzweigungen ohne Beschwerde | bedingt | + 10–40 | Zusatzfragen §3.6 (je ≈ 10 s) |
| Plausibilitäts-Rückfragen | bedingt | + 0–30 | höchstens zwei (§5.5) |
| je Region mit Beschwerde | bedingt | + 60–90 | Sicherheit geht vor dem Richtwert (O-4) |

**Ehrliche Einordnung:** Der Basispfad liegt knapp unter 5 Minuten. Mit den
Verzweigungen, die viele User durchlaufen (0 Klimmzüge, Mobilitätsfrage für ein
Handstand-Ziel, eine Rückfrage), liegt ein typischer Durchlauf bei 5–5.5
Minuten. Um den Richtwert zu halten, sind Grösse, Geschlecht, bevorzugte Tage,
Bänder, Zusatzlast und Nebenziel aus dem Pflichtteil in «Profil verfeinern»
verschoben (§3.8). Die Zeiten sind Schätzungen für das Antippen von Auswahlen
und müssen im Usability-Test gemessen werden (OE-9).

## 3. Pflichtteil: Felder

Jede Tabelle nennt pro Feld den Schlüssel (englisch, für API und Datenmodell),
die Frage, Feldtyp und Werte, die Validierung, die Begründung mit Quelle und die
Verwendung im Algorithmus. «Weiss nicht» ist überall erlaubt, wo es sinnvoll
ist; der Planer behandelt es als Selbstauskunft mit maximaler Unsicherheit
(§5.2).

### 3.1 Block A — Rahmen

| Feld | Frage / UI | Typ und Werte | Validierung | Begründung (Quelle) | Verwendung im Algorithmus |
|---|---|---|---|---|---|
| `birth_year` | «In welchem Jahr bist du geboren?» | Ganzzahl | 1920 ≤ Jahr ≤ aktuelles Jahr − Mindestalter; Mindestalter ist offen (OE-1) | Unter 18 gelten eigene Risiken: Wachstumsfuge am Handgelenk (Risikoalter 10–14 bzw. 10–16) [D-06, D-53], Rückenschmerz bei Extension als Warnzeichen [D-82] | Setzt `is_minor` (PAR-D-23 = 18). Minderjährige: Steigerungsdeckel für Handgelenk und Straight-Arm × PAR-D-02 (0.5); Red Flags RF-12 und RF-13 aktiv. |
| `health_data_consent` | Einwilligung zur Speicherung von Beschwerde- und Gesundheitsangaben, mit Erklärung wofür | Bool, Pflichtentscheidung | muss beantwortet werden | Beschwerden und Vorabfragen sind Gesundheitsdaten (Rechtsfrage, nicht Recherche; OE-2) | **Mit** Einwilligung: Block G vollständig, Angaben werden gespeichert. **Ohne** Einwilligung: Die Sicherheitsfragen in Block G (Belastungssymptome, Red Flags bei gemeldeter Beschwerde) werden trotzdem gestellt, aber nur für die Entscheidung «Plan ja/nein, Region planen ja/nein» verwendet und nicht gespeichert (Vorschlag, rechtlich zu prüfen, OE-2). Die übrigen Angaben entfallen; der Planer plant konservativ (keine Tests, keine Maximalversuche, Steigerungsdeckel × PAR-D-02) und sagt, dass er Beschwerden ohne Einwilligung nicht berücksichtigen kann. |
| `disclaimer_ack` | Kurzer Hinweis: «Hefesto plant Training. Es ersetzt keine ärztliche oder physiotherapeutische Abklärung.» | Bool | muss bestätigt werden | ADR 0003; Brief §11 (keine medizinischen Aussagen) | Voraussetzung für die Plan-Erzeugung. Der Disclaimer liegt auch im API-Payload jeder Verletzungsinformation. |

### 3.2 Block B — Ziele

| Feld | Frage / UI | Typ und Werte | Validierung | Begründung (Quelle) | Verwendung im Algorithmus |
|---|---|---|---|---|---|
| `goals[]` | «Welche Skills willst du lernen?» Auswahl aus der Skill-Karte, mit Bild | Liste von Skill-Slugs, 1–3 | Slug existiert in der Wissensbasis; 1 ≤ Anzahl ≤ 3; keine Duplikate | Mehr als drei Prioritäten lassen sich bei 2–3 Einheiten pro Skill und Woche [PAR-B-34] und höchstens zwei Straight-Arm-Skills gleicher Richtung pro Einheit [PAR-B-48] nicht sinnvoll dosieren (**Heuristik**, abgeleitet) | Bestimmt die Zielknoten im Skill-Graph; der Planer ergänzt die Vorstufen und empfohlenen Kanten (`02_skills_progressions.md` §8). |
| `goals[].priority` | Reihenfolge per Ziehen | Ganzzahl 1–3, eindeutig | lückenlos 1..n | Die zuerst trainierte Übung gewinnt am meisten Kraft [B-50] (Evidenz A); Priorität bestimmt die erste Position des Maximalblocks [PAR-B-46, PAR-E-02] | Reihenfolge in der Einheit, Zuteilung knapper Einheiten und des Straight-Arm-Satzbudgets [PAR-B-47, PAR-B-81]. |
| `goals[].target_level` | «Bis zu welcher Stufe?» Voreinstellung: die Meilenstein-Stufe des Skills | Level-Slug des Skills | gehört zum Skill; liegt über der aktuellen Stufe | Stufen sind Leistungsnachweise (`02` §2.1) | Endknoten des Pfads; die Unlock-Kriterien der Stufe gelten unverändert (ADR 0008). |
| `goals[].target_date` | optional: «Bis wann möchtest du das schaffen?» | Datum | in der Zukunft | Persona 5 | Nur für den Realismus-Check (§6). Das Datum verändert keine Dosierung. |

### 3.3 Block C — Verfügbarkeit

| Feld | Frage / UI | Typ und Werte | Validierung | Begründung (Quelle) | Verwendung im Algorithmus |
|---|---|---|---|---|---|
| `sessions_per_week` | «Wie oft pro Woche kannst du trainieren?» | Ganzzahl 1–7 | 1–7 | Kraft profitiert von ≥ 2 Einheiten pro Woche [B-31]; eine Einheit genügt zum Erhalt [B-87, B-98]; empfohlene Einheiten nach Niveau: Anfänger 2–3, Fortgeschrittene 3–4, Erfahrene 4–5 [PAR-B-36] | Anzahl geplanter Einheiten. Übersteigt die Angabe die empfohlene Zahl harter Einheiten, plant der Planer die übrigen Tage als kurze, ermüdungsarme Balance- oder Mobilitätseinheiten [PAR-B-35, PAR-E-12] oder als geplante Ruhe, die im Streak zählt (ADR 0003). Ab 4 Einheiten: Split statt Ganzkörper [PAR-B-37]. |
| `session_minutes` | «Wie viel Zeit hast du pro Einheit?» | Auswahl 20 · 30 · 45 · 60 · 75 · 90+ | eine Auswahl | Die PDF-Einheiten dauern 60–90 min (`01` §4.9); Kürzungsregeln und Vorlagen für 30/45/60/90 min [PAR-B-64 bis PAR-B-68]; ein Satz 1–3×/Woche steigert die Kraft Trainierter [B-96] | Wahl der Einheitsvorlage und Kürzungsreihenfolge; Aufwärmen und Primärblock werden nie gestrichen [PAR-B-68]. 20 min: Minimaldosis (Primärblock + 1 Paar; **Heuristik**, kürzer als die kürzeste belegte Vorlage PAR-B-64). |

### 3.4 Block D — Equipment

| Feld | Frage / UI | Typ und Werte | Validierung | Begründung (Quelle) | Verwendung im Algorithmus |
|---|---|---|---|---|---|
| `equipment[]` | «Was hast du zur Verfügung?» Kacheln mit Bild | Mehrfachauswahl aus geschlossenem Vokabular: `floor`, `wall`, `pull_up_bar`, `low_bar`, `parallel_bars`, `parallettes`, `rings`, `resistance_bands`, `weight_vest`, `dumbbells_or_plates`, `dip_belt`, `box_or_bench`, `gym` (Latzug, Kabelzug, Dip-Station), `outdoor_park` | mindestens eine Angabe; `floor` und `wall` sind immer gesetzt; `outdoor_park` setzt `pull_up_bar`, `low_bar`, `parallel_bars`; `gym` setzt `pull_up_bar`, `parallel_bars`, `dumbbells_or_plates`, `box_or_bench` (**Heuristik**, typische Ausstattung, beim ersten Plan bestätigen lassen) | Die PDFs ersetzen Übungen nach Equipment (`Normal`/`Elastic`/`Home`, `01` §4.8, H-5); Ringe machen Stufen 1–3 OG-Level schwerer [PAR-A-27]; am Barren sinkt die Handgelenkbeuger-Aktivität im Handstand von 61 % auf 44 % NRMS (Turner) [PAR-A-57, PAR-C-28]; für Parallettes gilt das nur per Analogie (Neutralgriff, **Heuristik** [PAR-C-27]) | Filtert die Übungsauswahl (`exercises.equipment`); steuert Ersatzübungen und Assistenz (Band) sowie Zusatzlast als Progressionsweg (`01` §4.5). Das Vokabular ersetzt die heutigen Freitext-Tags (`codebase_notes.md` §5). Ohne `resistance_bands` plant der Planer statt Band-Assistenz Regressionen. |

### 3.5 Block E — Körpergewicht und Trainingshintergrund

| Feld | Frage / UI | Typ und Werte | Validierung | Begründung (Quelle) | Verwendung im Algorithmus |
|---|---|---|---|---|---|
| `bodyweight_kg` | «Wie schwer bist du?» | Dezimalzahl | 30–250 | Körpergewichtstests messen relative Kraft (Kinder [F-24], Liegestütz bei College-Männern [F-56], Beugehang bei Studentinnen [F-57]) [PAR-F-14]; Umrechnungen hängen am Körpergewicht [F-25, F-26]; +10 % zusätzliche (nicht-kontraktile) Masse → −53 % Klimmzug-Wdh. bei fitten jungen Männern [PAR-C-31] | Lastanteile je Übung [PAR-C-18 bis PAR-C-22], Hebelmomente [PAR-C-01], Zusatzlast-Stufen [PAR-A-67]. Gespeichert in `user_bodyweight_log`. |
| `training_level` | «Wie würdest du dein Training beschreiben?» | 4 Optionen, abgeleitet aus dem Participant Classification Framework [F-45]: `sedentary` (0) · `recreational` (1) · `trained` (2) · `highly_trained` (≥ 3) | eine Auswahl | Sechsstufige Skala [F-45]; Plausibilitätsregel R-3 (`07` §8.7) | Plausibilitätsprüfung; Breite der Startwerte; Gate für den Maximalkraft-Testblock (Niveau ≥ 2, `07` §7.4). |
| `calisthenics_training_age` | «Wie lange trainierst du regelmässig mit dem Körpergewicht?» | `lt_6_months` · `6_to_12_months` · `1_to_4_years` · `gt_4_years` | eine Auswahl | Klassengrenzen so gewählt, dass sie die Schwellen des Planers treffen: Periodisierungswechsel bei ≈ 6 Monaten [PAR-B-01, PAR-B-02]; erhöhtes Verletzungsrisiko bei 6–48 Monaten Erfahrung [PAR-D-04] (Klassen selbst **Heuristik**) | `lt_6_months` → lineare Doppelprogression; sonst wochenweise wellenförmig. `6_to_12_months` und `1_to_4_years` → Risikofenster PAR-D-04 (konservativere Steigerung). Verbreitert die Startwerte, stuft aber nicht ein (O-2). |
| `last_regular_training` | «Wann hast du zuletzt regelmässig (mindestens 1× pro Woche) trainiert?» | `current_or_lt_3_weeks` · `3_to_6_weeks` · `7_to_16_weeks` · `17_to_26_weeks` · `gt_26_weeks` · `never` | eine Auswahl | Klassen entsprechen den Rampen-Bändern ≤ 2–3 Wochen / 3–6 / 7–16 / ≥ 17 Wochen [PAR-B-59 bis PAR-B-62]; Kraft bleibt ≈ 3 Wochen stabil [B-84, B-85]; Sehnensteifigkeit nach 1–2 Monaten Pause zurück auf dem Ausgangswert, Kraft nicht (junge Männer, Beintraining, n = 8–9) [D-18, D-19]; Rückkehr zur früheren 1RM nach 12 Wochen Pause in < 8 Wochen (ältere Männer) [B-90] | Pausenklasse → Wiedereinstiegsrampe [PAR-B-59 bis PAR-B-62]. Ab `3_to_6_weeks` gilt für Straight-Arm- und Handgelenk-Strukturen PAR-D-29 (≥ 4 Wochen → RTT-Stufe 1 mit PAR-D-33); die Klasse beginnt bei 3 Wochen, der Planer wendet die strengere Regel an (**Heuristik**, Sicherheitsgrösse). Persona 4. `never` → alle Strukturen als neue Belastungsart (PAR-D-12). |
| `pre_break_level` | nur bei `last_regular_training` ≥ `7_to_16_weeks`: «Was konntest du vor der Pause?» (gleiche Auswahl wie Block F) | wie Block F | – | Kraft kommt schneller zurück, als sie aufgebaut wurde [B-88, B-90]; ob Muskelgedächtnis das Retraining beschleunigt, ist offen [B-135] | Obergrenze und Tempo der Rampe; **nicht** Startwert der Dosierung (die Sehne hat sich schneller zurückgebildet als die Kraft [D-18, D-19]). |

**Ausgangslast ohne Logs:** Eine eigene Frage nach der Trainingshäufigkeit der
letzten Monate entfällt. Solange weniger als 3 Wochen Logs vorliegen, startet
jede Belastungsart mit PAR-D-12 (Woche 1: 50 % des Zielvolumens); danach
gelten die Deckel gegen das 3-Wochen-Mittel und das 30-Tage-Maximum
[PAR-D-09 bis PAR-D-11, PAR-D-31]. Das ist für Trainierte konservativ, aber
einfach und erklärbar (**Heuristik**).

### 3.6 Block F — Leistungsstand

Alle Leistungsfragen zählen **saubere** Wiederholungen bzw. Sekunden: voller
Bewegungsweg, ohne Schwung, ohne Band. Eine kurze Bildanleitung zeigt, was
sauber heisst (Formkriterien aus `02` und `07` §4.2). Die Klassengrenzen der
Zusatzfragen (Hang, Rudern, Stütz, Hollow Body, Mobilität) sind **Heuristik**;
für sie gibt es keine Reliabilitäts- oder MDC-Werte. Für offene Randklassen
(«> 20», «> 30») setzt der Planer μ auf die untere Grenze (konservativ,
**Heuristik**).

| Feld | Frage / UI | Typ und Werte | Validierung | Begründung (Quelle) | Verwendung im Algorithmus |
|---|---|---|---|---|---|
| `push_up_class` | «Wie viele saubere Liegestütze schaffst du am Stück?» | Klassen 0 · 1–3 · 4–7 · 8–12 · 13–20 · 21–30 · > 30 · weiss nicht | eine Auswahl | Klassen nicht schmaler als die MDC95 von 2–5 Wdh. [F-17, F-20, F-28]; erinnerte Werte +4–7 % zu hoch [F-51, F-52] | Startschätzung μ = Klassenmitte × 0.95 [PAR-F-55], σ nach §5.2; Stufe in der Liegestütz-Leiter (`02` §4.5). |
| `pull_up_class` | «… saubere Klimmzüge (Kinn über die Stange, unten gestreckt)?» | 0 · 1–3 · 4–7 · 8–11 · 12–15 · 16–20 · > 20 · weiss nicht | eine Auswahl | wie oben; Klimmzug ist die genaueste Selbstauskunft unter Heimtests [F-53]; Grenze 12 trifft das Gate des Maximalkraft-Testblocks (`07` §7.4) | wie oben; bei 0 folgen `dead_hang_class` und `row_class`; Muscle-up-Einstieg [PAR-A-69], Front-Lever-Einstieg [PAR-A-52] als empfohlene Kanten. |
| `dead_hang_class` | nur bei 0 Klimmzügen: «Wie lange kannst du frei hängen?» | < 10 s · 10–30 s · 30–60 s · > 60 s | eine Auswahl | Hang-Grundlagen sind Wurzel der Zug-Leiter (`02` §4.2); Beugehang mit 90° ist reliabel (ICC 0.98) und Alternative zum Klimmzug-Test [F-57] | Einstieg in `hang-foundation`; Testvorschlag Beugehang statt Klimmzug (`07` §7.3). |
| `row_class` | nur bei 0 Klimmzügen und `low_bar` oder `rings`: «Wie viele Rudern am niedrigen Holm (Körper schräg)?» | 0 · 1–5 · 6–10 · 11–15 · > 15 · weiss nicht | eine Auswahl | Rudern ist Vorstufe des Klimmzugs (`02` §4.7) | Stufe in der Ruder-Leiter. |
| `dip_class` | «… saubere Dips (Oberarm mindestens parallel)?» | 0 · 1–3 · 4–7 · 8–12 · 13–20 · > 20 · weiss nicht | eine Auswahl | wie oben; Muscle-up-Einstieg 5 Dips [PAR-A-69] | Stufe in der Dip-Leiter; bei 0 folgt `support_hold_class`. |
| `support_hold_class` | nur bei 0 Dips: «Wie lange hältst du den Stütz mit gestreckten Armen?» | < 10 s · 10–30 s · > 30 s | eine Auswahl | Stütz ist Wurzel der Druck-Leiter (`02` §4.4); Ring-Dip-Einstieg ab 30 s Ring-Stütz [PAR-A-49] | Stufe `support-hold`. |
| `handstand_class` | «Handstand?» | keiner · Wand < 30 s · Wand ≥ 30 s · frei < 10 s · frei ≥ 10 s · weiss nicht | eine Auswahl | Stream F Q12 (**Heuristik**); Novizen balancieren frei im Mittel nur 0.4–1.1 s [PAR-A-70] | Stufe im Handstand-Skill; Handgelenk-Belastbarkeit (Wurzel `wrist-conditioning`). |
| `hollow_hold_class` | nur wenn ein Ziel Planche, Front Lever, Back Lever oder L-Sit enthält: «Wie lange hältst du den Hollow Body?» | < 15 s · 15–30 s · 30–60 s · > 60 s · weiss nicht | eine Auswahl | Einstieg Front Lever und Planche laut Coaching 60 s Hollow [PAR-A-52, PAR-A-53] (Evidenz D, nur empfohlen) | Empfohlene Kante; Rumpf-Vorbereitung im Plan. |
| `skill_stages[]` | für jeden **Ziel-Skill**: «Welche Stufe kannst du sauber halten?» Bildauswahl der Stufen | je Skill: Level-Slug oder `none` · `unknown` | Level gehört zum Skill; Plausibilität R-2 | Einstufung pro Skill (O-2); Stufenbilder aus den PDFs und `02` §5–7 [P-01 S. 1–3] | Aktuelle Arbeitsstufe je Ziel-Skill. Vorstufen-Skills werden nicht einzeln abgefragt; der Planer leitet sie aus den Grundübungen ab und fragt nur bei Widerspruch nach (R-2). |
| `skill_stage_hold_class` | zur gewählten Stufe: «Wie lange?» (Halte) bzw. «Wie viele?» (Wdh.) | Halte: < 4 s · 4–9 s · 10–19 s · ≥ 20 s; Wdh.: 1 · 2–3 · 4–5 · > 5 | eine Auswahl | Unlock-Vorlage: Zwischenstufe ≥ 10 s, Endstufe ≥ 3 s [PAR-A-10, PAR-A-11]; Arbeitsfenster 3–20 s [PAR-B-05]; ein Satz braucht ≥ 2 s Halt und ≥ 2 s Reserve [PAR-B-07], also eine Maximalhaltezeit ≥ 4 s (08 §4) | < 4 s: Planer trainiert eine Stufe tiefer bzw. mit Band; ab 20 s bietet er die nächste Stufe an [PAR-B-05, PAR-A-65]. Die Angabe ist Selbstauskunft (§5.4). |
| `mobility_checks[]` | nur je Ziel, für das Mobilität die Übungswahl steuert, höchstens zwei: **Handgelenk** (Handstand, Planche, HSPU): «Kannst du die Handflächen flach aufsetzen und mit gestreckten Armen schmerzfrei Gewicht nach vorn verlagern?»; **Schulter** (Handstand, HSPU, Press): «Kommen deine gestreckten Arme über Kopf neben die Ohren, ohne Hohlkreuz?»; **Sprunggelenk** (Pistol): «Berührt dein Knie im Ausfallschritt die Wand, wenn die Zehen eine Handbreite entfernt sind, ohne dass die Ferse abhebt?»; **Kompression** (L-Sit, V-Sit, Manna): «Erreichst du im Langsitz mit gestreckten Beinen deine Zehen?» | je Check `yes` · `partly` · `no` | eine Auswahl | Mobilität sagt das Niveau nicht voraus (Spagat, Schulterflexion vs. Turnniveau r² ≤ 0.01) [F-28], steuert aber die Übungswahl [PAR-F-53]; Handstand verlangt ≈ 180° Schulterflexion [PAR-C-51] (Evidenz C); Handgelenk trägt im Handstand die Balance [PAR-A-58]; Ausfallschritt-Test ist reliabel [F-10]; Fragen als Selbstcheck **Heuristik** | Nur Übungswahl, nie Einstufung: `no`/`partly` Handgelenk → Parallettes/Fäuste statt Boden, Handgelenk-Vorbereitung im Aufwärmen [PAR-C-27]; Schulter → Brust-zur-Wand-Varianten und Mobilitätsblock; Sprunggelenk → erhöhte Ferse bei Pistol-Regressionen; Kompression → Kompressionsleiter (`02` §4.9). |
| `data_confidence` | «Wie sicher sind diese Zahlen?» | `estimated` · `counted_last_4_weeks` · `filmed` | eine Auswahl | Selbst gezählte Heim-Liegestütze lagen 17.3 % über der Videozählung desselben Tests (Paraplegie, n = 33, nur Abstract) [F-55]; Faktoren **Heuristik** (`07` §8.2, PAR-F-20, 21, 56) | Faktor auf σ aller Selbstauskünfte (§5.2). |

**Keine Freischaltungen aus dem Onboarding.** Das Onboarding bietet keine
Sammel-Selbstbestätigung an. Wer eine Stufe schon vor Hefesto erreicht hat,
nutzt den bestehenden Weg in der Skill-Detailansicht («Ich kann das schon»,
`verification: self_attested`, ohne XP, ADR 0008). Begründung: Eine
Sammelbestätigung würde unbestätigte Selbstauskunft in dauerhafte
Freischaltungen verwandeln und die an Wettkampfstandards gekoppelten
Unlock-Kriterien umgehen (08 §2, Rang 16). Die Dosierung hängt ohnehin an der
Kapazitätsschätzung, nicht am Unlock-Status (`codebase_notes.md` §6).

### 3.7 Block G — Gesundheit, Beschwerden, Verletzungen

Die App nennt keine Verdachtsdiagnose; Fragen und Texte beschreiben Symptome
und Orte (`05_injuries_prehab.md` §8, §9). Die mit «immer» markierten Felder
werden auch ohne `health_data_consent` gestellt, aber dann nicht gespeichert
(§3.1).

| Feld | Frage / UI | Typ und Werte | Validierung | Begründung (Quelle) | Verwendung im Algorithmus |
|---|---|---|---|---|---|
| `exertion_symptoms` (immer) | «Hattest du bei körperlicher Belastung schon Brustschmerz, ungewohnte Atemnot, Schwindel oder eine Ohnmacht?» | Bool | beantwortet | Symptome sind das Kernkriterium des Screenings [F-41, PAR-F-45]; Warnsymptome gehen Herz-Kreislauf-Ereignissen häufig voraus [D-91] (RF-10) | «Ja» → kein Plan; ärztliche Abklärung vor dem Training empfehlen (Dringlichkeit N/D wie RF-10). Der Plan wird erzeugt, sobald der User eine Freigabe bestätigt. |
| `screening[]` | 6 weitere Ja/Nein-Fragen: bekannte Herz-, Kreislauf-, Stoffwechsel- oder Nierenerkrankung; ärztliche Einschränkung für Training; Medikamente, die die Belastbarkeit betreffen; Schwangerschaft; andere Erkrankung mit Einfluss aufs Training; Knochen-/Gelenkproblem, das sich durch Training verschlechtern könnte | 6 × Bool | alle beantwortet | Belegt ist die Frage nach bekannter Herz-, Stoffwechsel- oder Nierenerkrankung [F-41, PAR-F-45]; die übrigen Fragen sind **Heuristik** in Anlehnung an das PAR-Q+-Konzept [F-42, F-43]. Die Nähe zu PAR-Q+ muss rechtlich geprüft werden (OE-3) | Jedes «Ja» → Abklärung empfehlen; der Planer plant bis zur bestätigten Freigabe ohne Tests und Maximalversuche (**Heuristik**, abgeleitet aus [F-41]). |
| `body_map` | «Hast du Beschwerden, die dein Training beeinflussen? Und hattest du in den letzten 12 Monaten eine Verletzung, die Training verhindert hat?» Eine Körperkarte, je Ort zwei Schalter (aktuell / letzte 12 Monate) | je Ort-Schlüssel: `current` · `past_12_months` · beides · nichts; Orte: `shoulder_front`, `shoulder_top_side`, `elbow_inner`, `elbow_outer`, `elbow_crease`, `wrist_back_extension`, `wrist_pinky_side`, `fingers_forearm_inner`, `chest`, `lower_back`, `knee`, `other` (`chest` nach ENT-S-9) | Schlüssel aus geschlossenem Vokabular | Beschwerden beeinflussen Training weit öfter als Ausfälle: wöchentlich 39 % [F-44]; Orte statt Diagnosen wie in der Matrix (`05` §8); Vorverletzung erhöht das Risiko (OR 4.08) [D-02], auch in Klettern und CrossFit [D-09, D-11, D-12]; Zeitraum 12 Monate **Heuristik** (`07` §7.2) | `current` → Detailfragen und Red Flags für diesen Ort. `past_12_months` → Steigerungsdeckel der Region × PAR-D-02 (0.5), Prehab dieser Region im Aufwärmen [PAR-D-37]. |
| `complaints[].pain_daily` | «Wie stark im Alltag?» | NRS 0–10 | 0–10 | Grün für höhere Last: Alltagsschmerz 1–2/10 [PAR-D-14] | ≤ 2: Rampe ab Stufe 1; > 2: Stufe 0 (Region nicht planen), übrige Regionen normal. |
| `complaints[].pain_training` | «Wie stark beim oder nach dem Training?» | NRS 0–10 | 0–10 | Schmerzgrenze ≤ 5/10 während und direkt nach Belastung [PAR-D-15] (Reha-Kontext, Übertragung Heuristik) | > 5: betroffene Übungsfamilien der Region auf M/X nach Matrix, Volumen nach PAR-D-24. Baseline für das Schmerz-Monitoring [PAR-D-13]. |
| `complaints[].onset` | «Wie hat es begonnen?» | `sudden` · `gradual` | eine Auswahl | Plötzlicher Beginn mit Knall/Kraftverlust ist Red Flag RF-01 [D-70, D-73] | `sudden` stellt RF-01, RF-03, RF-05, RF-06 zuerst. |
| `complaints[].duration` | «Seit wann?» | < 2 Wochen · 2–4 Wochen · 4–12 Wochen · > 12 Wochen | eine Auswahl | Verweis bei fehlender Besserung nach 4 Wochen [PAR-D-19] (Leitlinien LWS/Schulter; Übertragung Heuristik); früherer Hinweis nach 14 Tagen [PAR-D-32] | > 4 Wochen: freundlicher Hinweis auf eine Fachperson schon im Onboarding (A-Dringlichkeit). |
| `complaints[].suspected_serious` | «Vermutest du selbst einen Riss oder eine ernsthafte Verletzung?» | `yes` · `no` · `unsure` | eine Auswahl | Der Auftrag nennt «Verdacht auf Sehnenriss»; Rissmechanismen und frühe Versorgung [D-68, D-70]; die App diagnostiziert nicht, nimmt die Einschätzung des Users aber ernst (**Heuristik**, Sicherheit) | `yes` → wie Red Flag: Region gesperrt, zeitnahe Abklärung empfehlen (D), Freigabe nötig [PAR-D-21]. `unsure` → Region in Stufe 0 und freundlicher Hinweis auf Abklärung (A). |
| `complaints[].professional_assessment` | «Hat eine Fachperson das angeschaut?» | `no` · `yes_overuse_or_tendon` · `yes_tear_or_suspected_tear` · `yes_other` · `in_progress` | eine Auswahl | Angaben des Users, keine Diagnose der App | `yes_tear_or_suspected_tear` → Region gesperrt, Freigabe nötig [PAR-D-21]. `yes_overuse_or_tendon` und `yes_other` → Rampe mit PAR-D-33 (Start 25 %) statt PAR-D-24. `in_progress` → Region in Stufe 0 bis zur Rückmeldung des Users (**Heuristik**). `no` → Regeln der Tabelle. |
| `complaints[].restrictions` | optional: «Hat dir eine Fachperson Einschränkungen gegeben?» | Mehrfachauswahl Bewegungskategorien (Stütz gestreckt, Hängen/Ziehen, Überkopf, Handgelenk gestreckt belastet, Supination unter Last, Wirbelsäulen-Extension) + Freitext | – | Fachliche Vorgaben gehen jeder App-Regel vor (Heuristik, Sicherheit) | Harte Ausschlüsse im Planer, bis der User sie aufhebt. |
| `red_flags[]` (immer, bei `current`) | ortsabhängig: RF-01 bis RF-07 und RF-10 bei jedem Ort; RF-08, RF-09 nur bei `lower_back`; RF-12 nur bei `wrist_*` und `is_minor`; RF-13 nur bei `lower_back` und `is_minor` | Bool je Frage | alle gestellten beantwortet | Liste aus Leitlinien und Reviews, bewusst breit, weil einzelne Red Flags wenig trennscharf sind [D-85, D-86, D-87] (`05` §9); RF-08/RF-09 sind in `05` als Rückenfragen formuliert | Jedes «Ja» → Region gesperrt, Abklärung mit Dringlichkeit N/D/A empfehlen, Freigabe durch den User nötig [PAR-D-21]. Training insgesamt stoppen bei N-Dringlichkeit: RF-05 (Gelenk sichtbar verschoben), RF-07, RF-08, RF-10. |

**Orte ohne Matrixzeile.** `knee` betrifft nur die Bein-Familie (Pistol und
Regressionen): Tiefe begrenzen, Pistol-Stufen M, übrige Familien nicht
betroffen [PAR-C-42] (Aktion **Heuristik**). `other`: Freitext; Red Flags
werden gestellt, eine automatische Übungsregel gibt es nicht, der User kann
einzelne Übungen ausschliessen (**Heuristik**).

**Wie der Planer eine Beschwerde ohne Red Flag umsetzt:** Die Region bekommt
eine Rampenstufe (`05` §6.2): Stufe 0, wenn Alltagsschmerz > PAR-D-14; sonst
Stufe 1 mit Startvolumen PAR-D-24 (bzw. PAR-D-33 nach Verweis oder
Sehnenbefund). Die Matrix (`05` §8) bestimmt, welche Übungsfamilien der Region
ausgeschlossen (X), modifiziert (M) oder unter Schmerzregel erlaubt (S) sind.
Jede Lastreduktion wird als Deload erfasst und zählt im Streak als eingehalten
(ADR 0003, `05` §5.4).

### 3.8 «Profil verfeinern»: später und im Kontext

Diese Felder verbessern Schätzungen, ändern aber die erste Woche kaum. Der
Planer fragt sie, wenn sie gebraucht werden (erste Zusatzlast-Übung, erster
Band-Satz, Realismus-Check) oder der User sie im Profil setzt.

| Feld | Wann gefragt | Typ | Begründung (Quelle) | Verwendung |
|---|---|---|---|---|
| `height_cm` | beim Realismus-Check oder beim ersten Planche-Lean | Ganzzahl 120–230 | Schultermoment im vollen Hebel ≈ 0.246 × Grösse × Körpergewicht [PAR-C-01]; relative Hebel-Anforderung proportional zur Grösse [PAR-C-32] | Prior für Zeitschätzungen, Lean-Tiefe [PAR-C-09, PAR-C-10]; nie Sperre. Ohne Angabe: Referenz 1.75 m. |
| `sex` | beim Realismus-Check, optional | `female` · `male` · `diverse` · `no_answer` | Relative Schulterkraft von Frauen ≈ 0.80 der Männer (isometrisch, Handdynamometer, aktive Studierende) [PAR-C-53]; modellierte relative Hebel-Anforderung 1.24 (Full), 1.27 (Adv Tuck), 1.32 (Tuck) (Modellableitung, **Heuristik**) [PAR-C-54]; Klimmzug-1RM 0.73 vs. 1.16 × KG [A-11] | Nur Prior für Zeitschätzungen (§6), aus Logs nachkalibriert; nie Sperre oder Dosierungsgrenze. |
| `preferred_days` | beim ersten Wochenplan, optional | Menge von ISO-Wochentagen | Hohe Reize derselben Struktur ≥ 48 h auseinander [PAR-D-08, PAR-E-13] | Platzierung mit Abstandsregeln; ohne Angabe gleichmässig. |
| `bands[]` | beim ersten Band-Satz | Liste {Bezeichnung, Herstellerangabe kg} | Herstellerangaben 13–44 % zu hoch, gleiche Farben streuen 8–19 % [C-79, C-80] | `estimated_assist_kg` ± 20 % [PAR-C-61]; Entlastung am Hebel [PAR-C-13]; assistierte Sätze zählen voll für Belastung, nie für Unlocks [PAR-B-79, PAR-A-21]. |
| `max_added_load_kg` | bei der ersten Zusatzlast-Progression | Zahl 0–100 | Laststeigerung 2–10 % [PAR-B-32] | Obergrenze für Zusatzlast-Stufen. |
| `secondary_focus` | im Profil, optional | `none` · `hypertrophy` | Hypertrophie-Zielvolumen 10–20 Sätze/Muskel/Woche, Start 10 [PAR-B-21] (A); Minimum 4 [PAR-B-22]; Kraft braucht deutlich weniger [B-20, B-23] | Zusätzliche Volumensätze für Grundübungen, sofern Zeit bleibt. |

## 4. Optionaler Testtag und Kalibrierung

### 4.1 Grundsatz

Standard ist die **submaximale Kalibrierung in den ersten Einheiten**: Der
erste Arbeitssatz jeder Hauptübung endet bei RIR 2–3 (bzw. 2–3 s Reserve bei
Halten) und wird mit RIR/SIR geloggt. Ein solcher Satz geht mit demselben
Fehler (r = 2 Wdh.) in die Schätzung ein wie ein Maximaltest (`07` §8.2), ohne
dass jemand an seine Grenze gehen muss (O-5, O-6). Zwei solche Sätze aus zwei
Einheiten tragen mehr Information als ein einzelner Testtag (eigene Rechnung,
`07` §8.5).

Der **Testtag** ist ein freiwilliges Angebot für User, die ihre Werte genau
wissen wollen. Er verengt σ sofort (ein sauberer Test ersetzt eine
Selbstauskunft zu rund zwei Dritteln, `07` §8.4). Protokolle und Regeln stammen
aus `07_assessment.md` §4 und §7.4; die Blockdauern sind dort
**Praxisheuristik** (Summe ohne Maximalkraft-Block ≈ 50 min, PAR-F-38).

### 4.2 Allgemeine Regeln

| Regel | Inhalt | Quelle |
|---|---|---|
| G-1 | Jeder Test hat eine schriftliche Anleitung mit Start-, Gültigkeits- und Endkriterium und ein Beispielbild oder -video. | [F-16, F-28] |
| G-2 | Wiederholungstests enden bei der **ersten ungültigen Wiederholung**; Haltetests beim ersten Verlassen der Position. | [F-28]; strenger als die Studie wegen ADR 0003 (**Heuristik**) |
| G-3 | Ein Versuch; Gleichgewichtshalte (Handstand) bester von 2 Versuchen. | [F-28, F-31] |
| G-4 | Vor jedem maximalen Kraft- oder Skilltest ≥ 5 min Pause, zwischen anderen Muskelgruppen ≥ 3 min. | [F-28]; [P-01 S. 1–3, P-02, P-03]; 3 min **Heuristik** |
| G-5 | Reihenfolge am Testtag: Mobilität → Skill-Halte → Wiederholungstests → Rumpfhalte. Maximalkraft (1RM) an einem eigenen Tag. | [F-28, B-50, E-26]; konkrete Reihenfolge **Heuristik** (`07` §4.1) |
| G-6 | Körpergewicht am Testtag speichern. | [PAR-F-14] |
| G-7 | Verlaufstests frühestens nach 4 Wochen. | [F-18]; 4 Wochen **Heuristik** |
| G-8 | Schmerz beendet den Test; die Region läuft in die Schmerz- und Red-Flag-Logik. | Projektvorgabe |
| G-9 | Standardisiertes Aufwärmen (5–10 min, Rampensätze bis ~70 % der Testschwierigkeit). | Dauer [PAR-E-47] (**Heuristik**); Rampensätze [PAR-B-70] |
| G-10 | Tests werden als `set_entry` mit `kind = test` geloggt, mit `form_quality`, `failed` und Körpergewicht. Kein XP für Testversuche. | `codebase_notes.md` §2; ADR 0003 §5 |

### 4.3 Wann Tests und Stufen-Prüfversuche erlaubt sind

Ein Maximaltest (Testtag oder `kind = test` in einer Einheit) und ein
Prüfversuch der angegebenen Skill-Stufe werden nur angeboten, wenn **alle**
Bedingungen gelten; sonst bleibt es bei der submaximalen Kalibrierung (§4.1):

| Bedingung | Begründung |
|---|---|
| Der User wählt den Test aktiv (Angebot, keine Voreinstellung). | O-6; ADR 0003 |
| `health_data_consent = true`, `exertion_symptoms = false`, Screening ohne «Ja» oder Freigabe bestätigt. | S-1 in `08` §2.2; [F-41] |
| Die getestete Region ist `normal` (keine RTT-Stufe 0–4, keine Sperre). | Maximalversuche erst in RTT-Stufe 5 (`05` §6.2) |
| Für Straight-Arm- und Handgelenk-Tests: keine Pause ≥ 4 Wochen in den letzten 12 Wochen ohne abgeschlossene Rampe. | PAR-D-29; die Sehnensteifigkeit ist nach der Pause zurück, die Kraft nicht [D-18, D-19]. Die «Volltest»-Empfehlung in PAR-B-62 gilt deshalb nur für Bent-Arm- und Beinübungen. |
| Nicht in Woche 1 einer neuen Belastungsart. | PAR-D-12 |

Ein Stufen-Prüfversuch ist ein einzelner Halt der angegebenen Stufe bis zum
ersten Formverlust, höchstens bis zur angegebenen Klassengrenze; er ersetzt die
Selbstauskunft wie ein Test (`07` §8.2).

### 4.4 Testbatterie

| Block | Tests (Schlüssel aus `07` §4.2) | Gültigkeits- und Formkriterien (Kurzform) | Dauer | Bedingung |
|---|---|---|---|---|
| 0 Aufwärmen | – | Stream B | 10 min | immer |
| 1 Mobilität | `wrist_extension` (Smartphone-Winkel), `shoulder_flexion_prone_lift` oder `shoulder_flexion_angle`, `ankle_dorsiflexion_wblt` (nur bei Pistol-Ziel), `toe_touch` (nur bei Kompressions-/Manna-Zielen) | Handgelenk: App am Handrücken, Unterarm aufgelegt [F-33, F-34]; Schulter: Stab maximal anheben, Ellbogen gestreckt [F-28]; Ausfallschritt: Ferse am Boden, Knie an der Wand [F-10] | 8 min | immer |
| 2 Skill-Halte | `handstand_hold_free` (2 Versuche) oder `handstand_hold_wall`; aktuelle Stufe von bis zu 2 Ziel-Skills | Freier Handstand: Zeit läuft, sobald die Füsse über den Händen sind; Ende bei Bodenkontakt oder Handwechsel [F-28]. Wand-Handstand (Bauch zur Wand, **Praxisheuristik**): Zeit ab Stillstand, Ende bei Verlassen der Linie. Skill-Halte: Zeit ab vollständigem Stillstand [A-29 S. 20], Ende bei Verlassen der Position, Form nach `form_quality`-Winkelskala [PAR-A-16] | 15 min | nur wenn Block F eine Stufe angibt und §4.3 erfüllt ist |
| 3 Wiederholungen | `push_up_max`, `pull_up_max` (bei 0: `bent_arm_hang` 90°), `dip_max`; optional `handstand_push_up_max` | Liegestütz: Ellbogen ≥ 90°, oben gestreckt, Hüfte in Linie (Kriterien **Praxisheuristik**, PAR-F-69), keine Hand-Release-Variante [F-56]; Klimmzug: Kinn über Stange, unten volle Streckung, kein Schwung [F-18, F-28]; Dip: Oberarm ≥ parallel, oben gestreckt (**Praxisheuristik**) | 12 min | nach Equipment und §4.3 |
| 4 Rumpf | `hanging_pike_max` oder `hollow_body_hold`, `plank_hold` | Pike: Füsse berühren die Stange, kurz im toten Hang zwischen den Wdh. [F-28]; Hollow Body: Lendenwirbelsäule am Boden, Schulterblätter und Beine angehoben, Ende beim Abheben des unteren Rückens (**Praxisheuristik**); Plank: Körperlinie, Ende bei Positionsverlust [F-30] | 5 min | immer |
| 5 Maximalkraft (eigener Tag) | `weighted_pull_up_1rm`, `weighted_dip_1rm` | 1RM mit Zusatzlast; ICC 0.96–0.99 [F-19] | 30–40 min | nur `training_level` ≥ `trained`, ≥ 12 Klimmzüge und §4.3 (**Heuristik**, `07` §7.4) |

## 5. Ungenaue Selbsteinschätzung: Konfidenz und Nachkalibrierung

### 5.1 Zustand je Kapazität

Pro User, Übung, Messgrösse (Wdh., Haltezeit, 1RM-Last) und Assistenzklasse
führt der Planer eine Schätzung als Normalverteilung: Mittelwert μ,
Standardabweichung σ, Herkunft (`self_report`, `test`, `log`, `derived`) und
Zeitpunkt [F-46, F-47, F-49]. Die Fortschreibung als eindimensionaler
Kalman-Filter ist **Heuristik**; ein für Trainingsdaten validiertes Verfahren
gibt es nicht (`07` §8.1).

### 5.2 Startwerte aus dem Onboarding

| Angabe | μ | σ (Fehler r) | Quelle |
|---|---|---|---|
| Wdh.-Klasse, `estimated` | Klassenmitte × 0.95 (offene Randklasse: untere Grenze) | max(2 Wdh., 0.30 × μ) | Verzerrung +4–7 % [F-51, F-52] → PAR-F-55; Betrag 0.30 **Heuristik** für Werte ohne Test |
| Wdh.-Klasse, `counted_last_4_weeks` | Klassenmitte × 0.95 | 0.75 × obiger Wert | Faktor **Heuristik**, Richtung [F-51–F-55] |
| Halteklasse (Hang, Stütz, Hollow, Handstand, Skill-Stufe) | untere Klassengrenze, **ohne** −5 % | 0.30 × μ (relativer Teil von PAR-F-20; relatives Fehlermodell PAR-F-68; **Heuristik**) | Haltezeiten wurden im Voraus eher unterschätzt [F-54]; PAR-F-55 gilt dort nicht; untere Grenze konservativ (O-5) |
| Klasse, `filmed` | Klassenmitte | wie Test (2.0 Wdh.) | Video senkt die Überschätzung des Selbstzählens [F-55] (Population nur bedingt übertragbar); Gleichsetzung mit Test **Heuristik** [PAR-F-56] |
| `weiss nicht` | Populations-Prior je Trainingsniveau (z. B. freier Handstand Novizen 0.4–1.1 s [PAR-A-70]) | ≥ 0.5 × μ, mindestens 2 Wdh. bzw. 3 s | **Heuristik** |
| Umrechnung aus anderer Übung | nach `07` §5 (z. B. Dip-1RM ≈ 1.11 × Klimmzug-1RM [F-19]) | ≥ 0.35 × μ | Streuung zwischen Personen [F-07, F-21] |

**Verbreiterung um den Faktor 1.25** (**Heuristik**) bei: `training_level`
Stufe 0–1 mit fortgeschrittener Skill-Angabe (R-3, `07` §8.7); Pause
≥ `7_to_16_weeks` (die frühere Leistung sagt die aktuelle schlechter voraus,
weil der Verlust individuell und dosisabhängig ist [B-83, B-84]); jede
Auflösung nach R-2 und R-7 (§5.5). Mehrere Anlässe multiplizieren sich nicht,
der Faktor gilt einmal.

### 5.3 Wie Konfidenz die Dosierung steuert

| Klasse | σ / μ | Dosierung | Quelle |
|---|---|---|---|
| hoch | < 0.15 | aus μ | Anker: SEM eines Klimmzugtests ≈ 17 % des Mittels [F-28] |
| mittel | 0.15–0.30 | aus μ − 0.5 σ | **Heuristik** |
| niedrig | ≥ 0.30 | aus μ − σ; erster Satz als submaximaler Kalibrierungssatz (§4.1) | **Heuristik**; Selbstauskunft allein liegt hier [PAR-F-20] |

Beispiel (eigene Rechnung nach `07` §8.3): Selbstauskunft 8–11 Klimmzüge
(`estimated`) → μ = 9.5 × 0.95 ≈ 9.0, σ = max(2, 0.30 × 9.0) = 2.7 →
σ/μ = 0.30 → Klasse niedrig → Arbeitssätze aus ≈ 6.3 Wdh.; nach einem
Kalibrierungssatz mit 6 Wdh. bei RIR 2 (x = reps + RIR = 8, r = 2.0, K ≈ 0.65) → μ ≈ 8.4,
σ ≈ 1.6 → σ/μ ≈ 0.19 → Klasse mittel.

### 5.4 Nachkalibrierung über die ersten Logs

| Zeitraum | Was passiert | Quelle |
|---|---|---|
| Einheit 1–2 je Muster | erster Arbeitssatz der Hauptübung als submaximaler Kalibrierungssatz (RIR/SIR 2–3); Ziel-Skill: Arbeitssätze eine Stufe unter der angegebenen oder mit Band; Prüfversuch der angegebenen Stufe nur nach §4.3 | O-5, O-6; `07` §8.2; §4.3 |
| Woche 1 | Volumen neuer Belastungsarten 50 % des Ziels | [PAR-D-12] |
| laufend (ab Einheit 1) | Kapazitäts-Update je Beobachtung: Log-Sätze mit RIR ≤ 3 und ≤ 12 Wdh. zählen mit r = 2 Wdh.; Sätze bis Versagen wie Test; Sätze mit RIR > 3 nur als Untergrenze; assistierte, partielle und exzentrische Sätze zählen nicht für die unassistierte Kapazität | `07` §8.2–8.3; [F-08, F-38] |
| Woche 1–3 | RIR-Angaben gehen mit ihrem Fehler in die Kapazitätsschätzung ein, steuern aber noch nicht die Autoregulation der Zielwerte (Ziel-RIR, Stufenwechsel) | [PAR-B-29] |
| Stufenkorrektur | Prüfversuch < 4 s bzw. Form < 3 → Arbeitsstufe eine tiefer; Arbeitsstufe ≥ 20 s mit Form ≥ 4 in zwei Einheiten → nächste Stufe anbieten | [PAR-B-05, PAR-B-07, PAR-A-65, PAR-A-78] |
| Widerspruch | neue Beobachtung weicht > 2 × √(σ² + r²) ab → kein stilles Überschreiben; niedrigerer Wert mit grösserem σ gilt, Kalibrierungssatz vorschlagen | `07` §8.3 Schritt 4 |
| ab Woche 4 | RIR-Angaben steuern die Autoregulation; alle 4–6 Wochen ein Kalibrierungssatz | [PAR-B-29] |
| stabil | nach ≈ 4 Beobachtungen ±2 Wdh., nach ≈ 7 ±1.5 Wdh. (95 %) | eigene Rechnung, `07` §8.5 |

Die Nachkalibrierung ändert nie den Unlock-Status (ADR 0008). Sie ändert nur,
was der Planer dosiert.

### 5.5 Plausibilitätsregeln (Persona 6)

Das Onboarding prüft die Angaben gegen die Regeln aus `07` §8.7, soweit die
nötigen Daten vorliegen, und gegen zwei eigene Regeln. R-1 (Variantenfolge),
R-4 (Zug vs. Druck) und R-7 (Skill vs. gewichtete Kraft) brauchen Daten, die
erst Tests oder Logs liefern; sie laufen ab der ersten Einheit. Das Onboarding
stellt höchstens zwei Rückfragen (**Heuristik**, Zeitbudget); danach gilt die
konservative Auflösung.

| Regel | Beispiel | Rückfrage | Konservative Auflösung |
|---|---|---|---|
| R-2 Stufenfolge | Straddle Planche angegeben, aber 0 Liegestütze | «Kannst du die Tuck Planche 10 s halten?» | Stufe = höchste Stufe, deren Vorstufen plausibel sind; σ × 1.25 |
| R-3 Niveau vs. Skill | `sedentary`, aber Full Front Lever | «Hältst du den Front Lever ohne Band und mit gestreckten Armen?» | eine Stufe tiefer bis zum ersten Kalibrierungs- oder Prüfsatz |
| R-5 Kraft vs. Skill | 20 Klimmzüge, aber keine Front-Lever-Stufe | keine | Stufe nicht überspringen [F-40]; Tuck als Einstieg |
| R-8 Muscle-up ohne Zugbasis | Muscle-up, aber ≤ 3 Klimmzüge | «Mit Schwung (Kipping) oder mit Band?» | als Kipping-Variante werten; strikter MU nicht angenommen |
| R-9 Zeit vs. Ziele (neu) | 3 Straight-Arm-Ziele bei 2 × 30 min | keine | Ziele nach Priorität; Ziel 3 wird mit Erhaltungsdosis geplant und der User erfährt warum [PAR-B-34, PAR-B-47] (**Heuristik**) |
| R-10 Beschwerde vs. Ziel (neu) | Ellenbeuge-Beschwerde, Ziel Planche | keine | Ziel bleibt; supinierte Straight-Arm-Varianten X, Planche-Stufen M nach Matrix [D-68, PAR-D-41]; Rampe |

## 6. Realismus-Check für Ziele (Persona 5)

| Schritt | Regel | Quelle |
|---|---|---|
| 1 | Aktuelle und Ziel-Stufe auf die OG-Ordinalskala abbilden | [PAR-A-22 bis PAR-A-38] |
| 2 | Untergrenze = Summe der Untergrenzen je OG-Schritt aus PAR-A-45 [PAR-A-51]; angezeigte Erwartung = Summe aus der **oberen Hälfte** jedes Bands, bis eigene Logs vorliegen | PAR-A-45 ist am optimistischen Ende der Coaching-Angaben (`02` §3.6, PAR-A-51); Coaching-Werte [A-40, A-41, A-42, A-67] (Evidenz D) |
| 3 | Optional mit Prior-Faktoren skalieren: Hebel-Skills × relative Anforderung nach Grösse [PAR-C-32] und, falls angegeben, Geschlecht [PAR-C-54] | Modellwerte (Grösse B-Modell, Geschlecht Heuristik), nur Prior, nie Sperre |
| 4 | Liegt `target_date` vor der Untergrenze: Spanne anzeigen und ein erreichbares Zwischenziel vorschlagen | Beispiel: Anfänger → Full Planche ≥ 48 Wochen Untergrenze [PAR-A-51]; Coaching: Full nach 24–36 Monaten [A-40] |
| 5 | Formulierung ohne Wertung, ohne Druck: «Laut Coaching-Erfahrung dauert der Weg zur Full Planche meist 2–3 Jahre. Als nächstes Etappenziel schlagen wir die erste Planche-Stufe vor; die Tuck Planche folgt danach und braucht erfahrungsgemäss mehrere Monate. Willst du das als Zwischenziel setzen?» | ADR 0003 §3; O-8 |

Das Ziel wird nie abgelehnt; der Plan richtet sich nach der aktuellen Stufe,
nicht nach dem Datum. Zeitangaben sind als Coaching-Erfahrungswerte
gekennzeichnet, nicht als Prognose.

## 7. Ergebnis des Onboardings: der Start-Zustand

Das Onboarding übergibt dem Planer (Spezifikation Phase 4):

| Teil | Inhalt | Aus |
|---|---|---|
| Ziele | Skill, Ziel-Stufe, Priorität, optional Datum und Zwischenziel | Block B, §6 |
| Verfügbarkeit | Einheiten/Woche, Minuten, später bevorzugte Tage | Block C, §3.8 |
| Equipment | Menge aus geschlossenem Vokabular, später Bänder mit Unsicherheit und max. Zusatzlast | Block D, §3.8 |
| Person | Körpergewicht, `is_minor`, später Grösse und optional Geschlecht | Blöcke A, E, §3.8 |
| Trainingsstatus | Periodisierungsmodell (linear/wellenförmig), Risikofenster PAR-D-04, Pausenklasse und Rampe | Block E |
| Kapazitäten | je Übung/Messgrösse: μ, σ, Herkunft, Zeitpunkt | Block F, §5 |
| Skill-Stufen | Arbeitsstufe je Ziel-Skill, Status `claimed` bis zur ersten Kalibrierung | Block F, §5.4 |
| Mobilität | je Check `yes`/`partly`/`no` als Übungswahl-Filter | Block F |
| Strukturen | je Region: `normal` · RTT-Stufe 0–5 · `locked_pending_clearance`; Deckelfaktor bei Vorverletzung | Block G |
| Screening | Status `clear` · `clearance_recommended` · `stop` | Block G |
| Einwilligungen | Gesundheitsdaten, Hinweis bestätigt | Block A |

## 8. Datenmodell (Skizze für Phase 4)

Heute kennt `users` nur Locale, Einheiten und Zeitzone (`codebase_notes.md`
§5). Das Onboarding braucht neue, additive Tabellen; die genaue DDL gehört in
die Spezifikation:

- `user_training_profiles` (1 : 1): Verfügbarkeit, Trainingsniveau,
  Trainingsalter, Pausenklasse, Einwilligungen, Geburtsjahr, später Grösse und
  optional Geschlecht, Mobilitäts-Checks.
- `user_goals`: Skill, Ziel-Level, Priorität, Datum.
- `user_equipment`: Vokabular-Schlüssel, Bänder.
- `user_region_status` und `user_pain_reports`: Ort, Rampenstufe, Sperre,
  Schmerzwerte (NRS) mit Zeitpunkt. Das Log hat heute keinen Schmerzwert
  (PAR-D-13).
- `user_capacity_estimates`: μ, σ, Herkunft, Zeitpunkt je Übung und
  Messgrösse. `user_exercise_bests` ist ein Bestwert-Cache und kein Ersatz
  (`07` §8.8).
- `user_screening`: Antworten und Zeitpunkt (nur mit Einwilligung).

Konventionen: UUIDv7 in Go, `text` + benannter `CHECK`, `user_id` auf jeder
Zeile, Sync-Spalten für alles, was der Client offline ändert (CLAUDE.md).
Gesundheitsangaben brauchen eine eigene Aufbewahrungs- und Löschregel (OE-2).

## 9. Abdeckung der Personas

| Persona | Was das Onboarding erfasst | Folge im Start-Zustand |
|---|---|---|
| 1 Anfänger, Outdoor-Park, 2×/Woche, Muscle-up | `outdoor_park`, 2 × Minuten, 0–3 Klimmzüge, Hang/Rudern/Stütz | Wurzeln Hang, Rudern, Stütz, Liegestütz; MU-Kanten 5 Klimmzüge + 5 Dips als weicher Hinweis [PAR-A-69] (andere Coaching-Quellen nennen 8–18 Klimmzüge, `08` §5); Ganzkörper 2× [PAR-B-37] |
| 2 Fortgeschritten, Gym, 4×/Woche, Planche + Front Lever | beide Ziele mit Priorität, Stufen und Halteklassen, `gym` | zwei Straight-Arm-Skills gegensätzlicher Richtung als Paar [PAR-B-81]; Split ab 4 Einheiten [PAR-B-37]; gemeinsames Straight-Arm-Budget [PAR-B-47] |
| 3 Mediale Ellbogenbeschwerden, Ziel Planche | `elbow_inner` (aktuell), NRS, Dauer, Verdacht, Red Flags | Matrix: Planche-Familie M, Ringe-Straight-Arm X [D-63, D-64]; Rampe; Schmerz-Monitoring; keine Tests an der Region (§4.3) |
| 4 Wiedereinsteiger nach 6 Monaten | `last_regular_training = 17_to_26_weeks`, `pre_break_level` | Rampe PAR-B-62; Straight-Arm in RTT-Stufe 1 mit 25 % [PAR-D-29, PAR-D-33]; keine Straight-Arm-Tests bis nach der Rampe (§4.3); σ × 1.25 |
| 5 Full Planche in 8 Wochen als Anfänger | Ziel + Datum, Stufe `none` | Realismus-Check (§6), Etappenziel erste Planche-Stufe (Lean), danach Tuck Planche mit eigener Spanne (`spec.md` §3.6); Plan ab Wurzeln |
| 6 Widersprüchliche Angaben | Plausibilitätsregeln | höchstens zwei Rückfragen, danach konservativ (§5.5) |

## 10. Entscheidungen

Alle Vorschläge wurden am Checkpoint nach Phase 3 (27.09.2026) angenommen. Die
Spalte «Umsetzung» nennt die Stelle in `spec.md`; rechtliche Prüfungen bleiben
offen und sind Voraussetzung für den öffentlichen Betrieb (`spec.md` §13.3).

| Nr. | Frage | Entscheidung (angenommen) | Umsetzung |
|---|---|---|---|
| OE-1 | Mindestalter; sind Minderjährige zugelassen? | Mindestalter rechtlich klären (Einwilligungsalter); wenn zugelassen, gelten PAR-D-23 und RF-12/RF-13. | `spec.md` §8.8; rechtlich offen |
| OE-2 | Gesundheitsdaten: Einwilligung, Speicherort, Aufbewahrung, Löschung; Sicherheitsfragen ohne Einwilligung | Eigene Einwilligung, getrennte Tabellen, Löschung mit dem Konto. Sicherheitsfragen (Belastungssymptome, Red Flags) auch ohne Einwilligung stellen, aber nur flüchtig auswerten und nicht speichern — rechtlich prüfen; Alternative: ohne Einwilligung kein Plan. | `spec.md` §4.9, §5.2 (SAFE-04), §13.4; rechtlich offen |
| OE-3 | PAR-Q+ ist urheberrechtlich geschützt; die Vorabfragen ähneln ihm | Eigene Formulierung nach ACSM-Logik [F-41] und rechtliche Prüfung, ob sie als Bearbeitung gilt; PAR-Q+ nur mit Lizenz. | §3.7; rechtlich offen |
| OE-4 | Geschlecht erfragen? | Optional, nur im Kontext des Realismus-Checks, nur als Prior, mit «keine Angabe». | `spec.md` §3.6: Prior in v1 abgeschaltet (`PAR-S-19`), Feld bleibt optional |
| OE-5 | Sammel-Selbstbestätigung nach dem Onboarding? | Nein (§3.6); der bestehende Weg in der Skill-Detailansicht bleibt. | `spec.md` §6.13 |
| OE-6 | Testtag in v1? | v1: submaximale Kalibrierung in den ersten Einheiten (§4.1); Testtag als optionaler Ablauf später. | `spec.md` §5.6 (SEL-08), §5.7 (Kalibrierungssatz) |
| OE-7 | Messgrössen `cm`, Winkel, Verhältnis fehlen im Log (`measure`) | Mobilitäts-Checks und Testtag-Mobilität zunächst im Profil speichern, nicht im Log; Erweiterung in Phase 4 prüfen. | `spec.md` §4.9 (`mobility` im Profil); keine Log-Erweiterung in v1 |
| OE-8 | Schmerzwerte im Log | Neue Tabelle `user_pain_reports` statt Feld im Satz; Abfrage nach der Einheit und am nächsten Morgen (PAR-D-13). | `spec.md` §4.9, §8.6 |
| OE-9 | Zeitbudget | Im Usability-Test messen; Richtwert < 5 min für den Basispfad (§2). | offen bis zum Usability-Test |
