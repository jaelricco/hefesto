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
| O-1 | **Kurzer Pflichtteil (< 5 min), der Rest wird aus den Logs gelernt.** Jede Pflichtfrage muss eine Entscheidung der ersten Woche verändern. | Auftrag; die Kapazität ändert sich ohnehin, und mehrere Log-Beobachtungen tragen mehr Information als ein einzelner Messwert [F-18]; Stream F schätzt den Pflichtteil auf ≈ 4.8 min (`07_assessment.md` §7.3). |
| O-2 | **Einstufung pro Skill und Bewegungsmuster, nicht global.** | Kraft erklärt Skill-Leistung nur teilweise (R² 0.42–0.85 je Element) [A-21, F-40]; «Beginner» in den PDFs heisst bereits solide Tuck-Planche (`01_pdf_extract.md` §4.7, H-8); OG stuft nach Können, nicht nach Trainingsjahren ein [A-30 S. 23, F-69]. |
| O-3 | **Jeder Wert trägt Herkunft und Unsicherheit.** Selbstauskunft ist ein breiter Startwert, kein Messwert. | Selbstauskünfte ordnen Personen, messen aber nicht: von US-Soldaten erinnerte Liegestützzahlen im Mittel +4–7 % zu hoch, Einzelabweichung SD ≈ 13–18 % [F-51, F-52]; Heimtests bei norwegischen Stellungspflichtigen κ ≤ 0.34 [F-53]; Selbstauskunft vs. Messung körperlicher Aktivität r −0.71 bis 0.96 [F-13]. |
| O-4 | **Sicherheit vor Plan.** Gesundheits-Vorabfragen, Beschwerden je Region und Red Flags kommen vor der Plan-Erzeugung. | Überlastung dominiert Calisthenics-Verletzungen (62 % in 12 Monaten verletzt, Tendinopathie häufigste Diagnose) [D-02]; Screening nach Symptomen und gewünschter Intensität [F-41]. |
| O-5 | **Bei Unsicherheit konservativ dosieren.** Niedrige Konfidenz → Dosierung aus dem unteren Ende der Schätzung und ein Testsatz. | `07_assessment.md` §8.6; ADR 0003 (keine Maximalversuche als Belohnung). |
| O-6 | **Keine Maximalversuche im Pflichtteil.** Tests sind freiwillig, einzeln begrenzt und bringen kein XP. | ADR 0003 §5; Testende bei der ersten ungültigen Wiederholung (G-2 in `07_assessment.md` §4.1). |
| O-7 | **Widersprüche werden nachgefragt, nie still aufgelöst.** Im Zweifel gilt der niedrigere Wert mit grösserer Unsicherheit. | Plausibilitätsregeln R-1 bis R-8 (`07_assessment.md` §8.7); Persona 6. |
| O-8 | **Kein Ziel wird abgelehnt oder bewertet.** Unrealistische Termine werden mit einer Zeitspanne und einem Zwischenziel beantwortet. | ADR 0003 §3 (keine Aussage, dass jemand zurückliegt); Persona 5. |

## 2. Ablauf und Zeitbudget

```
Start
 └─ A Rahmen (Alter, Einwilligung, Hinweis)                ~15 s
 └─ B Ziele (bis 3 Skills, Reihenfolge, optional Datum)    ~35 s
 └─ C Verfügbarkeit (Tage, Minuten)                        ~15 s
 └─ D Equipment                                            ~20 s
 └─ E Körper & Trainingshintergrund                        ~40 s
 └─ F Leistungsstand (Grundübungen, Stufe je Ziel-Skill)   ~85 s
 └─ G Gesundheit (Vorabfragen, Beschwerden, Verletzungen)  ~80 s
      └─ je Region mit Beschwerde: Details + Red Flags    +60–90 s je Region
 └─ Plausibilitätsprüfung (höchstens 2 Rückfragen)         0–30 s
 └─ Ergebnis: Start-Zustand, erster Plan, Angebot Testtag
```

| Block | Pflicht | Zeit (s) | Grundlage der Zeitschätzung |
|---|---|---|---|
| A Rahmen | ja | 15 | Praxisheuristik (zwei Bestätigungen, eine Zahl) |
| B Ziele | ja | 35 | Stream F: 30 s (`07_assessment.md` §7.3, Q1) |
| C Verfügbarkeit | ja | 15 | Stream F: 20 s (Q3) |
| D Equipment | ja | 20 | Stream F: 20 s (Q2) |
| E Körper & Hintergrund | ja | 40 | Stream F: Q4 20 s + Q8 15 s; plus Pausenfrage |
| F Leistungsstand | ja | 85 | Stream F: Q9–Q14 = 95 s; Hollow Hold verzweigt |
| G Gesundheit | ja | 80 | Stream F: Q5–Q7 = 85 s |
| **Summe ohne Beschwerden** | | **≈ 290 s ≈ 4.8 min** | eigene Rechnung; Zeitangaben je Frage sind Praxisheuristik |
| je Region mit Beschwerde | verzweigt | + 60–90 | Heuristik; Sicherheit geht hier bewusst vor dem 5-Minuten-Ziel (O-4) |

Die Zeitangaben sind Schätzungen für das Antippen von Auswahlen und müssen im
Usability-Test gemessen werden (Offene Entscheidung OE-9).

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
| `health_data_consent` | Einwilligung zur Verarbeitung von Beschwerde- und Gesundheitsangaben, mit Erklärung wofür | Bool, Pflichtentscheidung | muss beantwortet werden | Beschwerden und Vorabfragen sind Gesundheitsdaten; ohne Einwilligung darf Block G nicht gespeichert werden (Rechtsfrage, nicht Recherche; OE-2) | Ohne Einwilligung: Block G entfällt, der Planer nimmt «unbekannt» an und plant konservativ (keine Tests, keine Maximalversuche, Steigerungsdeckel × PAR-D-02). Der Hinweis dazu wird angezeigt. |
| `disclaimer_ack` | Kurzer Hinweis: «Hefesto plant Training. Es ersetzt keine ärztliche oder physiotherapeutische Abklärung.» | Bool | muss bestätigt werden | ADR 0003; Brief §11 (keine medizinischen Aussagen) | Voraussetzung für die Plan-Erzeugung. Der Disclaimer liegt auch im API-Payload jeder Verletzungsinformation. |

### 3.2 Block B — Ziele

| Feld | Frage / UI | Typ und Werte | Validierung | Begründung (Quelle) | Verwendung im Algorithmus |
|---|---|---|---|---|---|
| `goals[]` | «Welche Skills willst du lernen?» Auswahl aus der Skill-Karte, mit Bild | Liste von Skill-Slugs, 1–3 | Slug existiert in der Wissensbasis; 1 ≤ Anzahl ≤ 3; keine Duplikate | Mehr als drei Prioritäten lassen sich bei 2–3 Einheiten pro Skill und Woche [PAR-B-34] und höchstens zwei Straight-Arm-Skills gleicher Richtung pro Einheit [PAR-B-48] nicht sinnvoll dosieren (**Heuristik**, abgeleitet) | Bestimmt die Zielknoten im Skill-Graph; der Planer ergänzt die Vorstufen und empfohlenen Kanten (`02_skills_progressions.md` §8). |
| `goals[].priority` | Reihenfolge per Ziehen | Ganzzahl 1–3, eindeutig | lückenlos 1..n | Die zuerst trainierte Übung gewinnt am meisten Kraft [B-50] (Evidenz A); Priorität bestimmt Position 1 der Einheit [PAR-B-46] | Reihenfolge in der Einheit, Zuteilung knapper Einheiten und des Straight-Arm-Satzbudgets [PAR-B-47, PAR-B-81]. |
| `goals[].target_level` | «Bis zu welcher Stufe?» Voreinstellung: die Meilenstein-Stufe des Skills | Level-Slug des Skills | gehört zum Skill; liegt über der aktuellen Stufe | Stufen sind Leistungsnachweise (`02` §2.1) | Endknoten des Pfads; die Unlock-Kriterien der Stufe gelten unverändert (ADR 0008). |
| `goals[].target_date` | optional: «Bis wann möchtest du das schaffen?» | Datum | in der Zukunft | Persona 5 | Nur für den Realismus-Check (§6). Das Datum verändert keine Dosierung. |
| `secondary_focus` | optional: «Nebenbei auch Muskelaufbau?» | `none` · `hypertrophy` | – | Hypertrophie braucht ≥ 10 Sätze pro Muskel und Woche [PAR-B-21] (Evidenz A), Kraft deutlich weniger [B-20, B-23] | Mit `hypertrophy`: zusätzliche Volumensätze für Grundübungen bis zu PAR-B-21, sofern Zeit bleibt; sonst reiner Skill-/Kraftfokus. |

### 3.3 Block C — Verfügbarkeit

| Feld | Frage / UI | Typ und Werte | Validierung | Begründung (Quelle) | Verwendung im Algorithmus |
|---|---|---|---|---|---|
| `sessions_per_week` | «Wie oft pro Woche kannst du trainieren?» | Ganzzahl 1–7 | 1–7 | Kraft profitiert von ≥ 2 Einheiten pro Woche [B-31]; eine Einheit genügt zum Erhalt [B-87, B-98]; empfohlene Einheiten nach Niveau: Anfänger 2–3, Fortgeschrittene 3–4, Erfahrene 4–5 [PAR-B-36] | Anzahl geplanter Einheiten. Übersteigt die Angabe die empfohlene Zahl harter Einheiten, plant der Planer die übrigen Tage als kurze, ermüdungsarme Balance- oder Mobilitätseinheiten [PAR-B-35] oder als geplante Ruhe, die im Streak zählt (ADR 0003). Ab 4 Einheiten: Split statt Ganzkörper [PAR-B-37]. |
| `session_minutes` | «Wie viel Zeit hast du pro Einheit?» | Auswahl 20 · 30 · 45 · 60 · 75 · 90+ | eine Auswahl | Die PDF-Einheiten dauern 60–90 min (`01` §4.9); Kürzungsregeln und Vorlagen für 30/45/60/90 min [PAR-B-64 bis PAR-B-68]; ein Satz 1–3×/Woche steigert die Kraft Trainierter [B-96] | Wahl der Einheitsvorlage und Kürzungsreihenfolge; Aufwärmen und Primärblock werden nie gestrichen [PAR-B-68]. 20 min: Minimaldosis (Primärblock + 1 Paar). |
| `preferred_days` | optional: Wochentage | Menge von ISO-Wochentagen | Anzahl ≥ `sessions_per_week`, sonst Hinweis | Hohe Reize derselben Struktur ≥ 48 h auseinander [PAR-D-08, PAR-E-13] | Platzierung der Einheiten mit Abstandsregeln; ohne Angabe verteilt der Planer gleichmässig. |

### 3.4 Block D — Equipment

| Feld | Frage / UI | Typ und Werte | Validierung | Begründung (Quelle) | Verwendung im Algorithmus |
|---|---|---|---|---|---|
| `equipment[]` | «Was hast du zur Verfügung?» Kacheln mit Bild | Mehrfachauswahl aus geschlossenem Vokabular: `floor`, `wall`, `pull_up_bar`, `low_bar`, `parallel_bars`, `parallettes`, `rings`, `resistance_bands`, `weight_vest`, `dumbbells_or_plates`, `dip_belt`, `box_or_bench`, `gym` (Latzug, Kabelzug, Dip-Station), `outdoor_park` | mindestens eine Angabe; `floor` und `wall` sind immer gesetzt; `outdoor_park` setzt `pull_up_bar`, `low_bar`, `parallel_bars`; `gym` setzt `pull_up_bar`, `parallel_bars`, `dumbbells_or_plates`, `box_or_bench` (**Heuristik**, typische Ausstattung, beim ersten Plan bestätigen lassen) | Die PDFs ersetzen Übungen nach Equipment (`Normal`/`Elastic`/`Home`, `01` §4.8, H-5); Ringe machen Stufen 1–3 OG-Level schwerer [PAR-A-27]; Parallettes senken die Handgelenkbeuger-Aktivität im Handstand von 61 % auf 44 % [PAR-A-57, PAR-C-28] | Filtert die Übungsauswahl (`exercises.equipment`); steuert Ersatzübungen und Assistenz (Band) sowie Zusatzlast als Progressionsweg (`01` §4.5). Das Vokabular ersetzt die heutigen Freitext-Tags (`codebase_notes.md` §5). |
| `bands[]` | optional, nur bei `resistance_bands`: Farbe/Stärke laut Hersteller | Liste {Bezeichnung, Herstellerangabe kg} | kg ≥ 0 | Herstellerangaben liegen 13–44 % zu hoch, gleiche Farben streuen 8–19 % [C-79, C-80]; Assistenz ist am stärksten, wo das Band am meisten gedehnt ist [PAR-C-62] | `estimated_assist_kg` mit Unsicherheit ± 20 % [PAR-C-61]; Entlastung am Hebel über den Angriffspunkt [PAR-C-13]. Assistierte Sätze zählen voll für Belastung, nie für Unlocks [PAR-B-79, PAR-A-21]. |
| `max_added_load_kg` | optional, bei Weste/Scheiben | Zahl | 0–100 | Zusatzlast überlädt beherrschte Stufen (`01` §4.5); Laststeigerung 2–10 % [PAR-B-32] | Obergrenze für Zusatzlast-Progressionen. |

### 3.5 Block E — Körper und Trainingshintergrund

| Feld | Frage / UI | Typ und Werte | Validierung | Begründung (Quelle) | Verwendung im Algorithmus |
|---|---|---|---|---|---|
| `bodyweight_kg` | «Wie schwer bist du?» | Dezimalzahl | 30–250 | Körpergewichtstests messen relative Kraft (Kinder [F-24], Liegestütz bei College-Männern [F-56], Beugehang bei Studentinnen [F-57]) [PAR-F-14]; Umrechnungen hängen am Körpergewicht [F-25, F-26]; +10 % Masse → −53 % Klimmzug-Wdh. [PAR-C-31] | Lastanteile je Übung [PAR-C-18 bis PAR-C-22], Hebelmomente [PAR-C-01], Zusatzlast-Stufen [PAR-A-67]. Gespeichert in `user_bodyweight_log`. |
| `height_cm` | «Wie gross bist du?» | Ganzzahl | 120–230 | Das Schultermoment im vollen Hebel ist ≈ 0.246 × Grösse × Körpergewicht [PAR-C-01]; relative Hebel-Anforderung steigt proportional zur Grösse [PAR-C-32] | Nur als Faktor für Zeitschätzungen und für die Planche-Lean-Tiefe [PAR-C-09, PAR-C-10]; nie als Sperre. Ohne Angabe: Referenz 1.75 m. |
| `sex` | optional: «Geschlecht (für Zeitschätzungen)» | `female` · `male` · `diverse` · `no_answer` | – | Relative Schulterkraft von Frauen ≈ 0.80 der Männer (isometrisch, Handdynamometer, aktive Studierende) [PAR-C-53]; daraus modellierte relative Hebel-Anforderung 1.24 für die Full-Stufe, 1.27 Adv Tuck, 1.32 Tuck (Modellableitung, Heuristik) [PAR-C-54]; Klimmzug-1RM 0.73 vs. 1.16 × KG [A-11] | Nur als Prior für die Dauer bis zur nächsten Stufe (§6), aus den Logs nachkalibriert; **nie** als Sperre oder Dosierungsgrenze [PAR-C-54]. Ohne Angabe: kein Faktor, breiteres Zeitband. |
| `training_level` | «Wie würdest du dein Training beschreiben?» | 4 Optionen, abgeleitet aus dem Participant Classification Framework [F-45]: `sedentary` (0) · `recreational` (1) · `trained` (2) · `highly_trained` (≥ 3) | eine Auswahl | Sechsstufige Skala [F-45]; Plausibilitätsregel R-3 (`07` §8.7) | Plausibilitätsprüfung; Breite der Startwerte; Gate für den Maximalkraft-Testblock (Niveau ≥ 2, `07` §7.4). |
| `calisthenics_training_age` | «Wie lange trainierst du regelmässig mit dem Körpergewicht?» | `lt_3_months` · `3_to_12_months` · `1_to_3_years` · `gt_3_years` | eine Auswahl | Kategorien aus Stream F (**Heuristik**, `07` §7.2); Verletzungsrisiko erhöht bei 6–48 Monaten Erfahrung [PAR-D-04]; UP-Periodisierung hilft erst ab ≈ 6 Monaten [PAR-B-01, PAR-B-02] | Periodisierungsmodell (linear < 6 Monate, wellenförmig ab 6 Monaten); konservativere Steigerung im Risikofenster PAR-D-04; verbreitert die Startwerte, stuft aber nicht ein (O-2). |
| `recent_frequency` | «Wie oft hast du in den letzten 3 Monaten pro Woche trainiert?» | 0 · 1 · 2 · 3 · 4 · 5+ | eine Auswahl | Neue Belastungsarten starten mit 50 % des Zielvolumens [PAR-D-12]; Einheitsdeckel gegen das 30-Tage-Maximum [PAR-D-31] brauchen eine Ausgangslast | Startwert der chronischen Last je Struktur, bis 3–4 Wochen Logs vorliegen. |
| `last_regular_training` | «Wann hast du zuletzt regelmässig (mindestens 1× pro Woche) trainiert?» | `now` · `2_4_weeks_ago` · `1_3_months_ago` · `3_6_months_ago` · `gt_6_months_ago` · `never` | eine Auswahl | Kraft bleibt bis ≈ 3 Wochen stabil [B-84, B-85]; Sehnensteifigkeit ist nach 1–2 Monaten Pause zurück auf dem Ausgangswert, Kraft nicht [D-18, D-19]; Rückkehr zur früheren 1RM nach 12 Wochen Pause in < 8 Wochen [B-90] | Pausenklasse → Wiedereinstiegsrampe [PAR-B-59 bis PAR-B-62]; ab 4 Wochen Pause starten Straight-Arm-Strukturen in RTT-Stufe 1 [PAR-D-29, PAR-D-33]. Persona 4. |
| `pre_break_level` | nur bei Pause ≥ 1 Monat: «Was konntest du vor der Pause?» (gleiche Auswahl wie Block F) | wie Block F | – | Muskelgedächtnis: Wiederaufbau schneller als Erstaufbau [B-90, B-135] | Obergrenze und Tempo der Rampe; **nicht** Startwert der Dosierung (die Sehne hat sich schneller zurückgebildet als die Kraft [D-18, D-19]). |

### 3.6 Block F — Leistungsstand

Alle Leistungsfragen zählen **saubere** Wiederholungen bzw. Sekunden: voller
Bewegungsweg, ohne Schwung, ohne Band. Eine kurze Bildanleitung zeigt, was
sauber heisst (Formkriterien aus `02` und `07` §4.2).

| Feld | Frage / UI | Typ und Werte | Validierung | Begründung (Quelle) | Verwendung im Algorithmus |
|---|---|---|---|---|---|
| `push_up_class` | «Wie viele saubere Liegestütze schaffst du am Stück?» | Klassen 0 · 1–3 · 4–7 · 8–12 · 13–20 · 21–30 · > 30 · weiss nicht | eine Auswahl | Klassen nicht schmaler als die MDC95 von 2–5 Wdh. [F-17, F-20, F-28]; erinnerte Werte +4–7 % zu hoch [F-51, F-52] | Startschätzung μ = Klassenmitte × 0.95 [PAR-F-55], σ nach §5.2; Stufe in der Liegestütz-Leiter (`02` §4.5). |
| `pull_up_class` | «… saubere Klimmzüge (Kinn über die Stange, unten gestreckt)?» | 0 · 1–3 · 4–7 · 8–12 · 13–20 · > 20 · weiss nicht | eine Auswahl | wie oben; Klimmzug ist die genaueste Selbstauskunft unter Heimtests [F-53] | wie oben; bei 0 folgen `dead_hang_class` und `row_class`; Muscle-up-Einstieg [PAR-A-69], Front-Lever-Einstieg [PAR-A-52] als empfohlene Kanten. |
| `dead_hang_class` | nur bei 0 Klimmzügen: «Wie lange kannst du frei hängen?» | < 10 s · 10–30 s · 30–60 s · > 60 s | eine Auswahl | Hang-Grundlagen sind Wurzel der Zug-Leiter (`02` §4.2); Beugehang mit 90° ist reliabel (ICC 0.98) und Alternative zum Klimmzug-Test [F-57] | Einstieg in `hang-foundation`; Testvorschlag Beugehang statt Klimmzug (`07` §7.3). |
| `row_class` | nur bei 0 Klimmzügen: «Wie viele Rudern am niedrigen Holm (Körper schräg)?» | 0 · 1–5 · 6–10 · 11–15 · > 15 · weiss nicht | eine Auswahl | Rudern ist Vorstufe des Klimmzugs (`02` §4.7) | Stufe in der Ruder-Leiter. |
| `dip_class` | «… saubere Dips (Oberarm mindestens parallel)?» | 0 · 1–3 · 4–7 · 8–12 · 13–20 · > 20 · weiss nicht | eine Auswahl | wie oben; Muscle-up-Einstieg 5 Dips [PAR-A-69] | Stufe in der Dip-Leiter; bei 0 folgt `support_hold_class`. |
| `support_hold_class` | nur bei 0 Dips: «Wie lange hältst du den Stütz mit gestreckten Armen?» | < 10 s · 10–30 s · > 30 s | eine Auswahl | Stütz ist Wurzel der Druck-Leiter (`02` §4.4); Ring-Dip-Einstieg ab 30 s Ring-Stütz [PAR-A-49] | Stufe `support-hold`. |
| `handstand_class` | «Handstand?» | keiner · Wand < 30 s · Wand ≥ 30 s · frei < 10 s · frei ≥ 10 s · weiss nicht | eine Auswahl | Stream F Q12 (**Heuristik**); Novizen balancieren frei im Mittel nur 0.4–1.1 s [PAR-A-70] | Stufe im Handstand-Skill; Handgelenk-Belastbarkeit (Wurzel `wrist-conditioning`). |
| `hollow_hold_class` | nur wenn ein Ziel Planche, Front Lever, Back Lever oder L-Sit enthält: «Wie lange hältst du den Hollow Body?» | < 15 s · 15–30 s · 30–60 s · > 60 s · weiss nicht | eine Auswahl | Einstieg Front Lever und Planche laut Coaching 60 s Hollow [PAR-A-52, PAR-A-53] (Evidenz D, nur empfohlen) | Empfohlene Kante; Rumpf-Vorbereitung im Plan. |
| `skill_stages[]` | für jeden Ziel-Skill und seine direkten Vorstufen-Skills: «Welche Stufe kannst du sauber halten?» Bildauswahl der Stufen | je Skill: Level-Slug oder `none` · `unknown` | Level gehört zum Skill; Plausibilität R-2 | Einstufung pro Skill (O-2); Stufenbilder aus den PDFs und `02` §5–7 [P-01 S. 1–3] | Aktuelle Arbeitsstufe je Skill (siehe `skill_stage_hold_class`). |
| `skill_stage_hold_class` | zur gewählten Stufe: «Wie lange?» (Halte) bzw. «Wie viele?» (Wdh.) | Halte: < 3 s · 3–9 s · 10–19 s · ≥ 20 s; Wdh.: 1 · 2–3 · 4–5 · > 5 | eine Auswahl | Unlock-Vorlage: Zwischenstufe ≥ 10 s, Endstufe ≥ 3 s [PAR-A-10, PAR-A-11]; Arbeitsfenster der Maximal-Halte 3–20 s [PAR-B-05] | Liegt die Angabe unter 3 s, trainiert der Planer eine Stufe tiefer bzw. mit Band; ab 20 s bietet er die nächste Stufe an [PAR-B-05, PAR-A-65]. Die Angabe ist Selbstauskunft und wird in der ersten Einheit geprüft (§5.4). |
| `data_confidence` | «Wie sicher sind diese Zahlen?» | `estimated` · `counted_last_4_weeks` · `filmed` | eine Auswahl | Selbst gezählte Heim-Liegestütze lagen 17.3 % über der Videozählung desselben Tests (Paraplegie, n = 33, nur Abstract) [F-55]; Faktoren **Heuristik** (`07` §8.2, PAR-F-20, 21, 56) | Faktor auf σ aller Selbstauskünfte (§5.2). |

**Selbstbestätigung auf der Karte.** Das Onboarding schaltet keine Level frei.
Es bietet nach dem Plan an, die angegebenen Stufen auf der Skill-Karte als
«selbst bestätigt» zu markieren. Das bringt kein XP (ADR 0008) und berührt die
Dosierung nicht, weil der Planer aus der Kapazitätsschätzung dosiert, nicht aus
dem Unlock-Status (`codebase_notes.md` §6). Offene Entscheidung OE-5.

### 3.7 Block G — Gesundheit, Beschwerden, Verletzungen

Nur mit `health_data_consent = true`. Die App nennt keine Verdachtsdiagnose;
Fragen und Texte beschreiben Symptome und Orte (`05_injuries_prehab.md` §8,
§9).

| Feld | Frage / UI | Typ und Werte | Validierung | Begründung (Quelle) | Verwendung im Algorithmus |
|---|---|---|---|---|---|
| `screening[]` | 7 Ja/Nein-Fragen zur Trainingsbereitschaft: bekannte Herz-, Kreislauf-, Stoffwechsel- oder Nierenerkrankung; Brustschmerz, Atemnot, Schwindel oder Ohnmacht bei Belastung; ärztliche Einschränkung für Training; Medikamente, die die Belastbarkeit betreffen; Schwangerschaft; andere Erkrankung mit Einfluss aufs Training; Knochen-/Gelenkproblem, das sich durch Training verschlechtern könnte | 7 × Bool | alle beantwortet | Screening nach Aktivität, Symptomen/Erkrankung und gewünschter Intensität [F-41]; PAR-Q+-Konzept mit 7 Fragen und Folgefragen [F-42, F-43]. **Eigene Formulierung**, weil PAR-Q+ urheberrechtlich geschützt ist [F-43] (OE-3) | Belastungssymptome (Brustschmerz, Atemnot, Schwindel, Ohnmacht) → wie RF-10: kein Plan, ärztliche Abklärung vor dem Training empfehlen. Jedes andere «Ja» → Abklärung empfehlen; der Planer plant bis zur bestätigten Freigabe ohne Tests und Maximalversuche (**Heuristik**, abgeleitet aus [F-41]). |
| `complaints[]` | «Hast du aktuell Beschwerden, die dein Training beeinflussen?» Körperkarte mit Orten | Mehrfachauswahl aus Ort-Schlüsseln: `shoulder_front`, `shoulder_top_side`, `elbow_inner`, `elbow_outer`, `elbow_crease`, `wrist_back_extension`, `wrist_pinky_side`, `fingers_forearm_inner`, `lower_back`, `knee`, `other` | Schlüssel aus geschlossenem Vokabular | Beschwerden beeinflussen Training weit öfter als Ausfälle: wöchentlich 39 % [F-44]; Orte statt Diagnosen, wie in der Matrix (`05` §8) | Je Ort: Matrix «Beschwerde × Übungsfamilie» (X/M/S), Red-Flag-Fragen, Rampe (`05` §6.2). |
| `complaints[].pain_daily` | «Wie stark im Alltag?» | NRS 0–10 | 0–10 | Grün für höhere Last: Alltagsschmerz 1–2/10 [PAR-D-14] | ≤ 2: Rampe ab Stufe 1; > 2: Stufe 0 (Region nicht planen), übrige Regionen normal. |
| `complaints[].pain_training` | «Wie stark beim oder nach dem Training?» | NRS 0–10 | 0–10 | Schmerzgrenze ≤ 5/10 während und direkt nach Belastung [PAR-D-15] (Reha-Kontext, Übertragung Heuristik) | > 5: betroffene Übungsfamilien der Region auf M/X nach Matrix, Volumen nach PAR-D-24. Baseline für das Schmerz-Monitoring [PAR-D-13]. |
| `complaints[].onset` | «Wie hat es begonnen?» | `sudden` · `gradual` | eine Auswahl | Plötzlicher Beginn mit Knall/Kraftverlust ist Red Flag RF-01 [D-70, D-73] | `sudden` stellt RF-01, RF-03, RF-05, RF-06 zuerst. |
| `complaints[].duration` | «Seit wann?» | < 2 Wochen · 2–4 Wochen · 4–12 Wochen · > 12 Wochen | eine Auswahl | Verweis bei fehlender Besserung nach 4 Wochen [PAR-D-19] (Leitlinien LWS/Schulter; Übertragung Heuristik); früherer Hinweis nach 14 Tagen [PAR-D-32] | > 4 Wochen: freundlicher Hinweis auf eine Fachperson schon im Onboarding (A-Dringlichkeit). |
| `complaints[].professional_assessment` | «Hat eine Fachperson das angeschaut?» | `no` · `yes_overuse_or_tendon` · `yes_tear_or_suspected_tear` · `yes_other` · `in_progress` | eine Auswahl | Angaben des Users, keine Diagnose der App | `yes_tear_or_suspected_tear` → wie Red Flag: Region gesperrt, Abklärung empfehlen, Freigabe nötig [PAR-D-21]. `yes_overuse_or_tendon` → Rampe mit PAR-D-33 (Start 25 %) statt PAR-D-24. |
| `complaints[].restrictions` | optional: «Hat dir eine Fachperson Einschränkungen gegeben?» | Mehrfachauswahl Bewegungskategorien (Stütz gestreckt, Hängen/Ziehen, Überkopf, Handgelenk gestreckt belastet, Supination unter Last, Wirbelsäulen-Extension) + Freitext | – | Fachliche Vorgaben gehen jeder App-Regel vor (Heuristik, Sicherheit) | Harte Ausschlüsse im Planer, bis der User sie aufhebt. |
| `red_flags[]` | bei jeder gemeldeten Beschwerde: RF-01 bis RF-10 als Ja/Nein; bei `is_minor` zusätzlich RF-12, RF-13 | Bool je Frage | alle beantwortet | Liste aus Leitlinien und Reviews, bewusst breit, weil einzelne Red Flags wenig trennscharf sind [D-85, D-86, D-87] (`05` §9) | Jedes «Ja» → Region gesperrt (bzw. Training gestoppt bei N-Dringlichkeit), Abklärung mit Dringlichkeit N/D/A empfehlen, Freigabe durch den User nötig [PAR-D-21]. Übrige Regionen planbar, ausser bei RF-07, RF-08, RF-10 (Training insgesamt stoppen). |
| `injuries_12_months[]` | «Hattest du in den letzten 12 Monaten eine Verletzung, die Training verhindert hat?» je Ort | Mehrfachauswahl Ort-Schlüssel | – | Vorverletzung erhöht das Risiko (OR 4.08) [D-02], auch in Klettern und CrossFit [D-09, D-11, D-12]; Zeitraum 12 Monate **Heuristik** (`07` §7.2) | Steigerungsdeckel der Region × PAR-D-02 (0.5); Prehab dieser Region im Aufwärmen [PAR-D-37]. |

**Wie der Planer eine Beschwerde ohne Red Flag umsetzt:** Die Region bekommt
eine Rampenstufe (`05` §6.2): Stufe 0, wenn Alltagsschmerz > PAR-D-14; sonst
Stufe 1 mit Startvolumen PAR-D-24 (bzw. PAR-D-33 nach Verweis oder
Sehnenbefund). Die Matrix (`05` §8) bestimmt, welche Übungsfamilien der Region
ausgeschlossen (X), modifiziert (M) oder unter Schmerzregel erlaubt (S) sind.
Jede Lastreduktion wird als Deload erfasst und zählt im Streak als eingehalten
(ADR 0003, `05` §5.4).

## 4. Optionaler standardisierter Testtag

Der Testtag ist freiwillig und wird nach dem ersten Plan angeboten. Er ersetzt
Selbstauskünfte durch Messungen und verengt σ sofort (Rechenbeispiel: ein
sauberer Test ersetzt eine Selbstauskunft zu rund zwei Dritteln, `07` §8.4).
Protokolle und Regeln stammen aus `07_assessment.md` §4 und §7.4.

### 4.1 Allgemeine Regeln

| Regel | Inhalt | Quelle |
|---|---|---|
| G-1 | Jeder Test hat eine schriftliche Anleitung mit Start-, Gültigkeits- und Endkriterium und ein Beispielbild oder -video. | [F-16, F-28] |
| G-2 | Wiederholungstests enden bei der **ersten ungültigen Wiederholung**; Haltetests beim ersten Verlassen der Position. | [F-28]; strenger als die Studie wegen ADR 0003 (**Heuristik**) |
| G-3 | Ein Versuch; Gleichgewichtshalte (Handstand) bester von 2 Versuchen. | [F-28, F-31] |
| G-4 | Vor jedem maximalen Kraft- oder Skilltest ≥ 5 min Pause, zwischen anderen Muskelgruppen ≥ 3 min. | [F-28]; [P-01 S. 1–3, P-02, P-03]; 3 min **Heuristik** |
| G-5 | Reihenfolge: Mobilität → Skill-Halte → Maximalkraft → Wiederholungstests → Rumpfhalte. | [F-28, B-50, E-26] |
| G-6 | Körpergewicht am Testtag speichern. | [F-24, F-25] |
| G-7 | Verlaufstests frühestens nach 4 Wochen. | [F-18]; 4 Wochen **Heuristik** |
| G-8 | Schmerz beendet den Test; die Region läuft in die Schmerz- und Red-Flag-Logik. | Projektvorgabe |
| G-9 | Standardisiertes Aufwärmen (5–10 min, Rampensätze bis ~70 % der Testschwierigkeit). | [PAR-B-69, PAR-B-70] |
| G-10 | Tests werden als `set_entry` mit `kind = test` geloggt, mit `form_quality`, `failed` und Körpergewicht. Kein XP für Testversuche. | `codebase_notes.md` §2; ADR 0003 §5 |

### 4.2 Testbatterie

| Block | Tests (Schlüssel aus `07` §4.2) | Gültigkeits- und Formkriterien (Kurzform) | Dauer | Bedingung |
|---|---|---|---|---|
| 0 Aufwärmen | – | Stream B | 10 min | immer |
| 1 Mobilität | `wrist_extension` (Smartphone-Winkel), `shoulder_flexion_prone_lift` oder `shoulder_flexion_angle`, `ankle_dorsiflexion_wblt` (nur bei Pistol-Ziel), `toe_touch` (nur bei Kompressions-/Manna-Zielen) | Handgelenk: App am Handrücken, Unterarm aufgelegt [F-33, F-34]; Schulter: Stab maximal anheben, Ellbogen gestreckt [F-28]; Ausfallschritt: Ferse am Boden, Knie an der Wand [F-10] | 8 min | immer |
| 2 Skill-Halte | `handstand_hold_free` (2 Versuche) oder `handstand_hold_wall`; aktuelle Stufe von bis zu 2 Ziel-Skills | Zeit ab vollständigem Stillstand [A-29 S. 20]; Ende bei Verlassen der Position; Form nach `form_quality`-Winkelskala [PAR-A-16] | 15 min | nur wenn Block F eine Stufe angibt |
| 3 Wiederholungen | `push_up_max`, `pull_up_max` (bei 0: `bent_arm_hang` 90°), `dip_max`; optional `handstand_push_up_max` | Liegestütz: Ellbogen ≥ 90°, oben gestreckt, Hüfte in Linie, keine Hand-Release-Variante [F-56]; Klimmzug: Kinn über Stange, unten volle Streckung, kein Schwung [F-18, F-28]; Dip: Oberarm ≥ parallel, oben gestreckt | 12 min | nach Equipment |
| 4 Rumpf | `hanging_pike_max` oder `hollow_body_hold`, `plank_hold` | Körperlinie; Ende bei Positionsverlust [F-30] | 5 min | immer |
| 5 Maximalkraft (eigener Tag) | `weighted_pull_up_1rm`, `weighted_dip_1rm` | 1RM mit Zusatzlast; ICC 0.96–0.99 [F-19] | 30–40 min | nur `training_level` ≥ `trained` und ≥ 12 Klimmzüge (**Heuristik**, `07` §7.4) |

**Alternative ohne Testtag (Standard):** Die Tests aus Block 2 und 3 laufen als
erster Satz (`kind = test`) in den ersten beiden regulären Einheiten des
jeweiligen Musters. Das kostet keinen Extratag und liefert zwei unabhängige
Beobachtungen (`07` §7.4, **Heuristik**). Beobachtungen aus zwei Einheiten
tragen mehr Information als ein einzelner Testtag [F-18].

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
| Klasse, `estimated` | Klassenmitte × 0.95 | max(2 Wdh., 0.30 × μ) | Verzerrung +4–7 % [F-51, F-52] → PAR-F-55; Betrag 0.30 **Heuristik** für Werte ohne Test |
| Klasse, `counted_last_4_weeks` | Klassenmitte × 0.95 | 0.75 × obiger Wert | Faktor **Heuristik**, Richtung [F-51–F-55] |
| Klasse, `filmed` | Klassenmitte | wie Test (2.0 Wdh.) | Video senkt die Überschätzung des Selbstzählens [F-55] (Population nur bedingt übertragbar); Gleichsetzung mit Test **Heuristik** [PAR-F-56] |
| `weiss nicht` | Populations-Prior je Trainingsniveau (z. B. freier Handstand Novizen 0.4–1.1 s [PAR-A-70]) | ≥ 0.5 × μ, mindestens 2 Wdh. bzw. 3 s | **Heuristik** |
| Skill-Stufe mit Halteklasse | untere Klassengrenze der Haltezeit | Klassenbreite | konservativ (O-5), **Heuristik** |
| Umrechnung aus anderer Übung | nach `07` §5 (z. B. Dip-1RM ≈ 1.11 × Klimmzug-1RM [F-19]) | ≥ 0.35 × μ | Streuung zwischen Personen [F-07, F-21] |

Zusätzlich verbreitern zwei Angaben σ um den Faktor 1.25 (**Heuristik**):
`training_level = sedentary` bei gleichzeitig fortgeschrittener Skill-Angabe
(R-3) und eine Pause ≥ 3 Monate (die frühere Leistung sagt die aktuelle
schlechter voraus [B-83, B-88]).

### 5.3 Wie Konfidenz die Dosierung steuert

| Klasse | σ / μ | Dosierung | Quelle |
|---|---|---|---|
| hoch | < 0.15 | aus μ | Anker: SEM eines Klimmzugtests ≈ 17 % des Mittels [F-28] |
| mittel | 0.15–0.30 | aus μ − 0.5 σ | **Heuristik** |
| niedrig | ≥ 0.30 | aus μ − σ, dazu ein Testsatz in der nächsten Einheit | **Heuristik**; Selbstauskunft allein liegt hier [PAR-F-20] |

Beispiel (eigene Rechnung nach `07` §8.4): Selbstauskunft 10 Klimmzüge
(`estimated`) → μ 9.5, σ 3 → Klasse niedrig → Arbeitssätze aus 6.5 Wdh.; nach
einem Testsatz mit 8 Wdh. → μ 8.6, σ 1.7 → Klasse mittel.

### 5.4 Nachkalibrierung über die ersten Logs

| Zeitraum | Was passiert | Quelle |
|---|---|---|
| Einheit 1–2 je Muster | erster Satz der Hauptübung als Testsatz (`kind = test`), wenn Konfidenz niedrig oder kein Testtag; Zielstufe eines Skills: ein kurzer Prüfversuch der angegebenen Stufe, Arbeitssätze eine Stufe tiefer oder mit Band | `07` §7.4; O-5; ADR 0003 (Testende bei erster ungültiger Wdh.) |
| Woche 1–2 | Volumen neuer Belastungsarten 50 % des Ziels [PAR-D-12]; RIR wird protokolliert, aber noch nicht für Dosierung vertraut | [PAR-D-12, PAR-B-29] |
| laufend | Kalman-Update je Beobachtung: Log-Sätze mit RIR ≤ 3 und ≤ 12 Wdh. zählen mit r = 2 Wdh.; Sätze bis Versagen wie Test; Sätze mit RIR > 3 nur als Untergrenze; assistierte, partielle und exzentrische Sätze zählen nicht für die unassistierte Kapazität | `07` §8.2–8.3; [F-08, F-38] |
| Stufenkorrektur | Prüfversuch < 3 s bzw. Form < 3 → Arbeitsstufe eine tiefer; Arbeitsstufe ≥ 20 s mit Form ≥ 4 in zwei Einheiten → nächste Stufe anbieten | [PAR-B-05, PAR-A-65, PAR-A-78] |
| Widerspruch | neue Beobachtung weicht > 2 × √(σ² + r²) ab → kein stilles Überschreiben; niedrigerer Wert mit grösserem σ gilt, Testsatz vorschlagen | `07` §8.3 Schritt 4 |
| ab Woche 4 | RIR-Angaben fliessen in die Autoregulation ein; alle 4–6 Wochen ein Kalibrierungs-Testsatz | [PAR-B-29] |
| stabil | nach ≈ 4 Beobachtungen ±2 Wdh., nach ≈ 7 ±1.5 Wdh. (95 %) | eigene Rechnung, `07` §8.5 |

Die Nachkalibrierung ändert nie den Unlock-Status (ADR 0008). Sie ändert nur,
was der Planer dosiert.

### 5.5 Plausibilitätsregeln (Persona 6)

Das Onboarding prüft die Angaben gegen die Regeln R-1 bis R-8 aus `07` §8.7 und
zwei eigene Regeln. Es stellt höchstens zwei Rückfragen; danach gilt die
konservative Auflösung.

| Regel | Beispiel | Rückfrage | Konservative Auflösung |
|---|---|---|---|
| R-2 Stufenfolge | Straddle Planche angegeben, aber keine Tuck Planche | «Kannst du auch die Tuck Planche 10 s halten?» | Stufe = höchste Stufe, deren Vorstufen plausibel sind; σ × 1.25 |
| R-3 Niveau vs. Skill | `sedentary`, aber Full Front Lever | «Hältst du den Front Lever ohne Band und mit gestreckten Armen?» | Testsatz in Einheit 1; bis dahin eine Stufe tiefer |
| R-5 Kraft vs. Skill | 20 Klimmzüge, aber keine Front-Lever-Stufe | keine | Stufe nicht überspringen [F-40]; Tuck als Einstieg |
| R-7 Skill vs. gewichtete Kraft | angegebene Stufe > 2 OG-Stufen über der Zugkraft [F-68] | «Wie lange hältst du sie, sauber?» | Testsatz, σ × 1.25 |
| R-8 Muscle-up ohne Zugbasis | Muscle-up, aber ≤ 3 Klimmzüge | «Mit Schwung (Kipping) oder mit Band?» | als Kipping-Variante werten; strikter MU nicht angenommen |
| R-9 Zeit vs. Ziele (neu) | 3 Straight-Arm-Ziele bei 2 × 30 min | keine | Ziele nach Priorität; Ziel 3 wird mit Erhaltungsdosis geplant und der User erfährt warum [PAR-B-34, PAR-B-47] (**Heuristik**) |
| R-10 Beschwerde vs. Ziel (neu) | Ellenbeuge-Beschwerde, Ziel Planche | keine | Ziel bleibt; supinierte Straight-Arm-Varianten X, Planche-Stufen M nach Matrix [D-68, PAR-D-41]; Rampe |

## 6. Realismus-Check für Ziele (Persona 5)

| Schritt | Regel | Quelle |
|---|---|---|
| 1 | Aktuelle und Ziel-Stufe auf die OG-Ordinalskala abbilden | [PAR-A-22 bis PAR-A-29] |
| 2 | Mindestdauer = Summe der Wochen je OG-Schritt aus PAR-A-45, **obere Hälfte** jedes Bands, bis eigene Logs vorliegen | PAR-A-45 ist am optimistischen Ende der Coaching-Angaben (`02` §3.6); Coaching-Werte [A-40, A-41, A-42, A-67] (Evidenz D) |
| 3 | Optional mit Prior-Faktoren skalieren: Hebel-Skills × relative Anforderung nach Grösse [PAR-C-32] und, falls angegeben, Geschlecht [PAR-C-54] | Modellwerte (Grösse B-Modell, Geschlecht Heuristik), nur Prior, nie Sperre |
| 4 | Liegt `target_date` vor der Untergrenze: Spanne anzeigen und ein erreichbares Zwischenziel vorschlagen | Beispiel: Anfänger → Full Planche ≥ 48 Wochen Untergrenze [PAR-A-51]; Coaching: Full nach 24–36 Monaten [A-40] |
| 5 | Formulierung ohne Wertung, ohne Druck: «Laut Coaching-Erfahrung dauert der Weg zur Full Planche meist 2–3 Jahre. Bis zu deinem Datum ist die Tuck Planche ein realistisches Etappenziel. Willst du das als Zwischenziel setzen?» | ADR 0003 §3; O-8 |

Das Ziel wird nie abgelehnt; der Plan richtet sich nach der aktuellen Stufe,
nicht nach dem Datum. Zeitangaben sind als Coaching-Erfahrungswerte
gekennzeichnet, nicht als Prognose.

## 7. Ergebnis des Onboardings: der Start-Zustand

Das Onboarding übergibt dem Planer (Spezifikation Phase 4):

| Teil | Inhalt | Aus |
|---|---|---|
| Ziele | Skill, Ziel-Stufe, Priorität, optional Datum und Zwischenziel | Block B, §6 |
| Verfügbarkeit | Einheiten/Woche, Minuten, bevorzugte Tage | Block C |
| Equipment | Menge aus geschlossenem Vokabular, Bänder mit Unsicherheit, max. Zusatzlast | Block D |
| Person | Körpergewicht, Grösse, optional Geschlecht, `is_minor` | Blöcke A, E |
| Trainingsstatus | Periodisierungsmodell (linear/wellenförmig), Risikofenster PAR-D-04, Pausenklasse und Rampe | Block E |
| Kapazitäten | je Übung/Messgrösse: μ, σ, Herkunft, Zeitpunkt | Block F, §5 |
| Skill-Stufen | Arbeitsstufe je Skill, Status `claimed` bis zur Prüfung | Block F, §5.4 |
| Strukturen | je Region: `normal` · RTT-Stufe 0–5 · `locked_pending_clearance`; Deckelfaktor bei Vorverletzung | Block G |
| Screening | Status `clear` · `clearance_recommended` · `stop` | Block G |
| Einwilligungen | Gesundheitsdaten, Hinweis bestätigt | Block A |

## 8. Datenmodell (Skizze für Phase 4)

Heute kennt `users` nur Locale, Einheiten und Zeitzone (`codebase_notes.md`
§5). Das Onboarding braucht neue, additive Tabellen; die genaue DDL gehört in
die Spezifikation:

- `user_training_profiles` (1 : 1): Verfügbarkeit, Trainingsniveau,
  Trainingsalter, Pausenklasse, Einwilligungen, Geburtsjahr, Grösse,
  optional Geschlecht.
- `user_goals`: Skill, Ziel-Level, Priorität, Datum.
- `user_equipment`: Vokabular-Schlüssel, Bänder.
- `user_region_status` und `user_pain_reports`: Ort, Rampenstufe, Sperre,
  Schmerzwerte (NRS) mit Zeitpunkt. Das Log hat heute keinen Schmerzwert
  (PAR-D-13).
- `user_capacity_estimates`: μ, σ, Herkunft, Zeitpunkt je Übung und
  Messgrösse. `user_exercise_bests` ist ein Bestwert-Cache und kein Ersatz
  (`07` §8.8).
- `user_screening`: Antworten und Zeitpunkt.

Konventionen: UUIDv7 in Go, `text` + benannter `CHECK`, `user_id` auf jeder
Zeile, Sync-Spalten für alles, was der Client offline ändert (CLAUDE.md).
Gesundheitsangaben brauchen eine eigene Aufbewahrungs- und Löschregel (OE-2).

## 9. Abdeckung der Personas

| Persona | Was das Onboarding erfasst | Folge im Start-Zustand |
|---|---|---|
| 1 Anfänger, Outdoor-Park, 2×/Woche, Muscle-up | `outdoor_park`, 2 × Minuten, 0–3 Klimmzüge, Hang/Rudern/Stütz | Wurzeln Hang, Rudern, Stütz, Liegestütz; MU-Kanten 5 Klimmzüge + 5 Dips [PAR-A-69]; Ganzkörper 2× [PAR-B-37] |
| 2 Fortgeschritten, Gym, 4×/Woche, Planche + Front Lever | beide Ziele mit Priorität, Stufen und Halteklassen, `gym` | zwei Straight-Arm-Skills gegensätzlicher Richtung als Paar [PAR-B-81]; Split ab 4 Einheiten [PAR-B-37]; gemeinsames Straight-Arm-Budget [PAR-B-47] |
| 3 Mediale Ellbogenbeschwerden, Ziel Planche | `elbow_inner`, NRS, Dauer, Red Flags | Matrix: Planche-Familie M, Ringe-Straight-Arm X [D-63, D-64]; Rampe; Schmerz-Monitoring |
| 4 Wiedereinsteiger nach 6 Monaten | `last_regular_training = gt_6_months_ago` / `3_6_months_ago`, `pre_break_level` | Rampe PAR-B-62; Straight-Arm in RTT-Stufe 1 mit 25 % [PAR-D-29, PAR-D-33]; σ × 1.25 |
| 5 Full Planche in 8 Wochen als Anfänger | Ziel + Datum, Stufe `none` | Realismus-Check (§6), Zwischenziel Tuck Planche; Plan ab Wurzeln |
| 6 Widersprüchliche Angaben | Plausibilitätsregeln | höchstens zwei Rückfragen, danach konservativ (§5.5) |

## 10. Offene Entscheidungen

| Nr. | Frage | Vorschlag |
|---|---|---|
| OE-1 | Mindestalter; sind Minderjährige zugelassen? | Mindestalter rechtlich klären (Einwilligungsalter); wenn zugelassen, gelten PAR-D-23 und RF-12/RF-13. |
| OE-2 | Gesundheitsdaten: Einwilligung, Speicherort, Aufbewahrung, Löschung | Eigene Einwilligung, getrennte Tabellen, Löschung mit dem Konto; ADR in Phase 4. |
| OE-3 | PAR-Q+ ist urheberrechtlich geschützt | Eigene Formulierung nach ACSM-Logik [F-41]; PAR-Q+ nur mit Lizenz. |
| OE-4 | Geschlecht erfragen? | Optional, nur als Prior für Zeitschätzungen, mit «keine Angabe». |
| OE-5 | Onboarding-Angaben als «selbst bestätigt» auf der Karte markieren? | Nur auf ausdrücklichen Wunsch, ohne XP (ADR 0008). |
| OE-6 | Testtag in v1 oder nur die Alternative in den ersten Einheiten? | v1: Alternative in den ersten Einheiten; Testtag als optionaler Ablauf später. |
| OE-7 | Messgrössen `cm`, Winkel, Verhältnis fehlen im Log (`measure`) | Mobilitätstests zunächst im Profil speichern, nicht im Log; Erweiterung in Phase 4 prüfen. |
| OE-8 | Schmerzwerte im Log | Neue Tabelle `user_pain_reports` statt Feld im Satz; Abfrage nach der Einheit und am nächsten Morgen (PAR-D-13). |
| OE-9 | Zeitbudget | Im Usability-Test messen; Ziel < 5 min ohne Beschwerde-Verzweigung. |
