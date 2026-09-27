# 08 — Synthese: was der Planer umsetzen muss

> Phase 3. Führt die PDF-Auswertung (`01`) und die sechs Recherche-Streams
> (`02`–`07`) zusammen: gerankte Kernprinzipien, der Abgleich der PDF-Muster mit
> der Literatur, zwischen den Streams abgestimmte Parameter, Widersprüche und
> offene Fragen. Quellen-IDs und Parameter-IDs verweisen auf die Stream-Dateien;
> das Gesamtverzeichnis steht in `00_sources.md`. Das Onboarding-Design steht
> in `docs/algorithm/onboarding.md`.

## 1. Evidenzlage in einem Satz je Stream

| Stream | Datei | Tragfähig belegt | Dünn oder nur Praxis |
|---|---|---|---|
| A Skills & Progressionen | `02` | Überlastung durch schwerere Varianten (RCTs), Wettkampfstandards für «gehalten» (FIG, WSWCF), Schwierigkeitsordnung (OG, FIG), Kraft als notwendige, nicht hinreichende Bedingung | Unlock-Schwellen je Stufe, Dauer bis zur nächsten Stufe (nur Coaching C/D), Muscle-up, Flagge, Pistol |
| B Trainingsmethodik | `03` | Volumen-Dosis-Wirkung, Versagen unnötig, Reihenfolge, Frequenz, RIR-Genauigkeit, Detraining, Aufwärmen | Dosierung von Straight-Arm-Statics, Band-Assistenz, Deload-Tiefe, GtG, Interferenz zweier Skills |
| C Anatomie & Biomechanik | `04` | Lastanteile im Liegestütz (Kraftmessplatte), EMG für Klimmzug, Dip, Muscle-up, Ringelemente, Hebelmodell aus anthropometrischen Daten | EMG/Kinetik für Front/Back Lever, Flagge, Manna, HSPU; Handgelenkwinkel in statischen Halten |
| D Verletzungen & Prehab | `05` | Epidemiologie Calisthenics/Street Workout, Sehnen-Zeitverläufe (untere Extremität), Kritik an ACWR und 10-%-Regel, Schmerz-Monitoring (Reha-RCTs), Red Flags aus Leitlinien | Obere-Extremität-Sehnen, Präventionsstudien Handgelenk, Übungsausschlüsse bei Beschwerden |
| E CNS & motorisches Lernen | `06` | Spezifität, Reihenfolgeeffekt, Ermüdung schadet dem Lernen, CI klein in der Praxis, Schlafmangel, Erholung zentraler Ermüdung | Handstand-spezifische Frequenz, Skill-Maximalversuche pro Einheit, Erholung nach Straight-Arm-Arbeit |
| F Leistungsdiagnostik | `07` | Reliabilität von Klimmzug, Liegestütz, 1RM, Mobilitätstests; Genauigkeit von RIR und erinnerten Testwerten | Haltetests (L-Sit, Hollow, Handstand), Selbstmessung der Schulter, validierte Skill-Schwellen |

**Gesamtbild:** Die Literatur trägt die **Struktur** des Planers gut
(Reihenfolge, Frequenz, Volumenlogik, Autoregulation, Belastungssteuerung,
Unsicherheit der Selbstauskunft). Die **Zahlen für Straight-Arm-Skills**
(Haltezeiten, Stufenschwellen, Dauer, Sehnenbelastbarkeit von Ellbogen und
Schulter) sind fast nur Praxis oder Übertragung. Der Planer braucht deshalb
belegte Regeln mit bewusst konservativen, als Heuristik markierten Werten, die
später aus den eigenen Logs nachgeschärft werden.

## 2. Kernprinzipien, gerankt nach Evidenz und Wirkung

### 2.1 Bewertung

- **Evidenz** der tragenden Belege: A = 3, B = 2, C = 1, D oder Heuristik = 0.5.
  Übertragungen aus anderen Populationen senken um eine Stufe.
- **Wirkung** auf Planentscheidungen: hoch = 3 (verändert jeden Plan oder die
  Sicherheit), mittel = 2 (verändert viele Pläne), niedrig = 1 (Einzelfälle,
  Texte).
- **Score** = Evidenz × Wirkung. Die drei Sicherheitsprinzipien S-1 bis S-3
  stehen unabhängig vom Score vorne: Sie sind Voraussetzung jeder
  Planerzeugung (ADR 0003).

### 2.2 Sicherheitsprinzipien (vor jedem Ranking)

| Nr. | Prinzip | Regel im Planer | Evidenz | Belege | Parameter |
|---|---|---|---|---|---|
| S-1 | **Warnzeichen screenen und verweisen, nie diagnostizieren.** | Red-Flag-Fragen bei jeder Beschwerdemeldung; bei «Ja» Region sperren (bzw. Training stoppen), Abklärung mit Dringlichkeit empfehlen, Freigabe durch den User nötig. Belastungssymptome (Brustschmerz, Atemnot, Schwindel) stoppen jede Planung. | B (Leitlinien); Einzel-Red-Flags wenig trennscharf, daher bewusst breit | D-70, D-73, D-86, D-88, D-89, D-92; F-41 | PAR-D-21, RF-01–RF-13 |
| S-2 | **Die Sehne passt sich langsamer an als die Kraft.** Straight-Arm-, Handgelenk- und supinierte Last steigt langsamer als die Leistung. | Wochendeckel je Struktur (Straight-Arm und Handgelenk +10 %, Bent-Arm +20 % gegenüber dem 3-Wochen-Mittel); Einheitsdeckel 110 % des 30-Tage-Maximums; neue Stufen gelten erst nach ≈ 12 Wochen als voll belastbar; ≥ 48 h zwischen hohen Reizen derselben Struktur | A/B für den Zeitverlauf (untere Extremität), Übertragung und Deckelwerte Heuristik; Einheitsdeckel B (Analogie Laufen) | D-17, D-18, D-19, D-20, D-23, D-36; B-12, B-79 | PAR-D-06, 08, 09, 10, 11, 31; PAR-B-56 |
| S-3 | **Bei Schmerz reduzieren statt pausieren, nach klaren Grenzen.** | Schmerz ≤ 5/10 während und direkt nach, am nächsten Morgen zurück auf Ausgangsniveau, kein Anstieg über Wochen; sonst Regression und −30 % Volumen für eine Woche (als Deload erfasst); wiederholt oder > 28 Tage ohne Besserung → Fachperson | A im Reha-Kontext (Achilles, Patella, Schulter); Übertragung auf Trainierende Heuristik | D-40, D-42, D-43, D-45, D-47, D-88 | PAR-D-13–20, 32, 40 |

### 2.3 Gerankte Kernprinzipien

| Rang | Prinzip | Regel im Planer | Evidenz | Wirkung | Score | Belege | Parameter |
|---|---|---|---|---|---|---|---|
| 1 | **Fortschritt über schwerere Varianten und Stufen**, nicht über mehr Sätze. | Leitern je Skill; Wechsel, wenn die Obergrenze des Arbeitsbereichs in 2 Einheiten erreicht ist («2-for-2»); Volumen je Einheit bleibt über die Niveaus fast konstant. | A | hoch | 9 | A-01, A-02, A-56; B-108–110; B-132; B-20, B-23; `01` §4.1 | PAR-B-30, PAR-A-03 |
| 2 | **Das Wichtigste zuerst, frisch.** | Reihenfolge: Aufwärmen → Maximalblock des Skills mit höchster Priorität → Volumen → Kraft/Zubringer → Prehab; bei gleicher Priorität rotieren. | A (Reihenfolge), B (Lernen) | hoch | 9 | B-50 = E-26, E-27, E-60; E-28; P-01–P-03 | PAR-B-46, PAR-E-01–03 |
| 3 | **Spezifität.** | Jede Skill-Einheit enthält die Zielstufe oder ihre nächste Regression; Zubringer ersetzen das nicht; Kraftgewinne sind modus- und winkelbezogen, der Übertrag auf benachbarte Winkel ist vorhanden, seine Breite aber umstritten (Nachbarstufen zählen nur teilweise, Faktor Heuristik). | A/B | hoch | 9 | E-21, E-22, E-24, E-25, E-45; B-13, B-14 (`03` W-23, `06` W-24) | PAR-E-23, PAR-E-24, PAR-B-15 |
| 4 | **Frequenz vor Einzelvolumen, mit Abstand.** | 2–3 Einheiten je Skill und Woche (Anfänger 2, Trainierte 3, max. 4); Wochenvolumen verteilen; ≥ 48 h zwischen harten Reizen derselben Struktur, 24 h nach submaximaler Technik. | A (Muskelgruppen-Frequenz; Übertragung auf Skills) / B | hoch | 9 | B-23, B-31, B-120–122; E-53, E-54, E-56; E-82; D-20, D-23 | PAR-B-34, PAR-B-38, PAR-E-11, 13, 14, PAR-D-08 |
| 5 | **Nicht bis zum Versagen; Qualität stoppt den Block.** | Ziel-RIR 1–3 bzw. Sekunden in Reserve; Skill- und Straight-Arm-Sätze nie bis zum Versagen (Tests ausgenommen); Block endet bei Formabfall ≥ 1 Punkt oder < 3, nach 2 Fehlversuchen oder bei > 20 % Leistungsabfall. | A (Versagen); Erholung nach Versagen langsamer, Unterschiede bis 24–48 h [E-83]; Stoppschwellen Heuristik | hoch | 9 | B-25, B-26, B-31, B-58; E-28, E-83, E-89, E-90 | PAR-B-24, 27, PAR-E-15–18 |
| 6 | **Selbstauskunft ist ein breiter Startwert.** | Kapazität als Schätzung mit Unsicherheit; bei niedriger Konfidenz vom unteren Ende dosieren und einen Testsatz einplanen; Kalibrierung über die ersten Logs; Widersprüche nie still zum höheren Wert auflösen. | A/B (Messfehler belegt, Filterverfahren Heuristik) | hoch | 9 | F-13, F-14, F-18, F-20, F-28, F-51–F-55 | PAR-F-20, 21, 55; `07` §8 |
| 7 | **Maximale Halte kurz, Pausen lang.** | Arbeitsfenster der Stufe: frische Maximalhaltezeit 3–20 s; Satzhaltezeit 60–70 % davon; 2–5 Sätze, 30–90 s Gesamtzeit je Übung; 180–300 s Pause, schwerste Elemente bis 420 s. | B/C (Isometrie-Review B, Praxis C) | hoch | 6 | B-09, B-113; A-63; E-81, E-87; P-01–P-04 | PAR-B-04–09, 39, PAR-E-04, 05, PAR-A-64 |
| 8 | **Wiedereinstieg nach Pausen mit Rampe.** | Kraft bleibt ≈ 3 Wochen; danach Rampe nach Pausendauer (2 Wochen / 1 / 3 / 6 Monate); ab 4 Wochen Pause starten Straight-Arm-Strukturen mit 25 % des früheren Volumens, weil die Sehnensteifigkeit schneller verloren geht als die Kraft. | A/B (Detraining), Rampenwerte Heuristik | hoch | 6 | B-83–B-90, B-135; D-18, D-19; D-53 | PAR-B-59–62, PAR-D-29, 33 |
| 9 | **Autoregulation schlägt starre Vorgaben nicht, ist aber mindestens gleich gut.** | RIR/SIR steuern Dosis und Wechsel; RIR wird im Mittel um ≈ 1 Wdh. unterschätzt, ist nahe am Versagen genauer und wird nicht durch Übung besser → Kalibrierungs-Testsatz alle 4–6 Wochen. | A | mittel | 6 | B-37, B-40, B-41, B-130; F-08, F-38 | PAR-B-28, 29, 31 |
| 10 | **Krafttraining und kurzes Prehab schützen.** | Prehab 2–4×/Woche im Aufwärmen, nach Regionen-Rangfolge (Schulter > Handgelenk > Ellbogen = Rücken); kein Schutzversprechen ohne Evidenz. | A (Sport allgemein), Übertragung | mittel | 6 | D-38, D-39, D-75, D-11, D-80; D-01, D-02 | PAR-D-01, 37, 38 |
| 11 | **Kraft ist notwendig, aber nicht hinreichend.** | Kraft-Baselines (Klimmzüge, Dips, Hollow) sind empfohlene Kanten, nie harte Voraussetzungen; Stufen werden nicht übersprungen, auch bei hoher Grundkraft. | B | mittel | 4 | A-21, A-23; F-40; C-12 | PAR-A-43, 62, PAR-C-38 |
| 12 | **Hebelphysik ordnet Stufen, Bänder und Zusatzlast.** | Relative Last je Stufe (Tuck ≈ 0.60, Adv Tuck ≈ 0.75, Straddle 0.85–0.96, Full 1.0 des Schultermoments); Band an der Hüfte entlastet, am Hals kaum; Planche ≈ 1.4× Front Lever bei gleicher Stufe. Steuert Ersatzübungen und Belastungsgewichte. | B (Modell) | mittel | 4 | C-01, C-02, C-15; C-26–C-29 | PAR-C-01–16 |
| 13 | **Deload heisst weniger Volumen, nicht Pause.** | Alle 4–8 Wochen (Standard: Woche 6) oder ausgelöst (Stagnation in 2 Einheiten, hohe Ermüdung, Beschwerden): Sätze × 0.6, +2 RIR, Frequenz und Übungen gleich. Zählt im Streak als eingehalten. | B (Praxisbefragungen, Konsens), ein RCT gegen Komplettpause | mittel | 4 | B-62–B-65 | PAR-B-49–54 |
| 14 | **Unlocks an Wettkampfstandards koppeln.** | Zwischenstufe ≥ 10 s, Endstufe ≥ 3 s, Form ≥ 4 (≤ 15° Abweichung), ≥ 2 Vorkommen in 28 Tagen; assistierte, partielle und exzentrische Sätze zählen nie. | B (FIG) / C (WSWCF, Coaching) | mittel | 4 | A-29, A-33, A-40; ADR 0003/0008 | PAR-A-10–21 |
| 15 | **Aufwärmen allgemein plus spezifisch, ohne Vorbelastung.** | 5–15 min je nach Einheitslänge; 2–3 Rampensätze (~50 % → ~70 % der Arbeitsschwierigkeit); statisches Dehnen < 60 s je Muskel; keine maximalen Isometrien als Potenzierung; keine Maximalversuche als Aufwärmen. | A/B | niedrig | 3 | B-67–B-71; E-92, E-96, E-97; D-01 | PAR-B-69–71, PAR-E-26, 45, 46, PAR-D-05 |
| 16 | **Einfache Periodisierung genügt.** | Anfänger (< 6 Monate): lineare Doppelprogression; Trainierte: Schwer/Mittel/Leicht über die Woche; Mesozyklus 4–8 Wochen. | A (moderater Effekt, methodisch schwach) | niedrig | 3 | B-01, B-02, B-05, B-06, B-31 | PAR-B-01–03 |
| 17 | **Maximalversuche geblockt, Variation zwischen Einheiten.** | Alle Maximalversuche eines Skills hintereinander; Varianten wechseln zwischen Einheiten, die Zielstufe bleibt. | A für die Begründung (CI-Effekt in der Praxis klein); dass geblocktes Üben besser ist, zeigt keine Quelle → Heuristik | niedrig | 3 | E-36–E-39, E-44, E-45 | PAR-E-32–34 |
| 18 | **Keine Begründung mit «CNS-Fatigue» oder «Schlaf verstärkt Lernen».** | Erklärtexte nennen belegte Gründe (periphere Ermüdung, Sehnenerholung, Ermüdung schadet dem Lernen); Schlafmangel (≤ 6 h) nur, falls ein Check-in existiert: Maximalversuche → Technik. | A/B | niedrig | 3 | E-81, E-82, E-85; E-62, E-64, E-65, E-70 | PAR-E-29, 41–44 |
| 19 | **Assistierte Sätze sind Volumen, nie Nachweis.** | Band-Sätze zählen für Straight-Arm-Budget und Deckel voll, für Hypertrophie 1.0 bei RIR/SIR ≤ 3 (sonst 0.5), für Progression der Zielstufe 0 und nie für Unlocks. | C (Praxismuster) / Heuristik | mittel | 2 | `01` §4.4; B-29, B-134; ADR 0008 | PAR-B-79, PAR-A-21 |
| 20 | **Zeitangaben sind Erfahrungswerte, keine Prognosen.** | Realismus-Check mit der oberen Hälfte der Coaching-Spannen; Ziel nie ablehnen; Zwischenziel anbieten; Dauern später aus eigenen Logs lernen. | C/D | mittel | 1–2 | A-40, A-41, A-42, A-67 | PAR-A-45, 51 |
| 21 | **Greasing the Groove nur für Bent-Arm-Grundübungen.** | Optional; ≤ 50 % der Max.-Wdh., ≥ 2 RIR; nie Straight-Arm, nie Versagen; zählt nicht als Hypertrophie-Satz. | C/D (Konzept), B (Laboranalogie) | niedrig | 1 | E-57, E-58, E-100–E-103; B-120–122 | PAR-B-80, PAR-E-27, 48, 49 |

## 3. Die PDF-Muster im Licht der Recherche

| Hypothese (`01` §6) | Urteil | Begründung |
|---|---|---|
| F-1 Einheit = 1 Maximalübung (2–3 Sätze, lange Pause) → 2 Volumenübungen (je 5) → 1–2 Zubringer | **Reihenfolge gestützt, Satzschema Praxis** | Reihenfolge belegt [B-50, E-26, E-28] (A/B); das 2-5-5-3-Schema ist nicht untersucht und liegt mit 12–18 Arbeitssätzen im Rahmen von ≈ 2–3 direkten Sätzen je Übung für Kraft [B-24, B-117] und ≤ 10–12 Sätzen je Muskel und Einheit [PAR-B-75]. |
| F-2 Straight-Arm-Isometrie kurz (2–20 s), Pausen ≥ 3 min, Maximalversuche ≥ 5 min | **Pausen gestützt, Intensitätsdeutung plausibel** | 3–5 min bei schweren Sätzen [E-60, E-87, E-88], zentrale Erholung ≈ 2 min, periphere 3–5 min [E-81]; kurze Halte bei 80–100 % MVC sind Maximalkrafttraining [B-09]. Die Umrechnung Haltezeit → % MVC ist eine Extrapolation aus Einzelgelenkmodellen und gilt nur, wenn ein Satz nahe an der frischen Maximalhaltezeit endet (Audit B, `03` §17). |
| F-3 Assistierte Sätze zählen voll in der Belastungssteuerung | **Praxis, konsistent mit der Sehnenlogik** | Keine Trainingsstudie zu Band-Assistenz [B §3.5]; Sehnen reagieren auf die Lasthöhe [B-12, D-17]; Planer-Regel PAR-B-79. |
| F-4 Zusatzlast ist eigener Progressionsweg für beherrschte Stufen | **Gestützt** | Laststeigerung 2–10 % [B-32]; Überlastung durch schwerere Varianten [A-01, A-02]; Hebelwirkung von Knöchel- und Hüftgewichten berechenbar [PAR-C-14]. |
| F-5 Jede Übung braucht Ersatz nach Equipment und Schwierigkeit | **Gestützt (Mechanik), Auswahl Praxis** | Ringe 1–3 OG-Level schwerer [PAR-A-27]; Parallettes senken die Handgelenkbeuger-Last [PAR-A-57, PAR-C-28]; Band- und Lastwirkung am Hebel [PAR-C-13, 14]. |
| F-6 Volumen je Einheit bleibt konstant, Progression über Übungswahl | **Gestützt** | Kraft sättigt mit Volumen früh [B-20, B-23]; Progression über Varianten ist wirksam [A-01, A-02, A-56]. |
| F-7 60–90 min Einheiten brauchen eine Kürzungsregel | **Gestützt** | Ein Satz 1–3×/Woche steigert Kraft [B-96]; Minimum ≥ 4 Sätze/Muskel/Woche [B-95]; Supersätze ohne Nachteil [B-101]; Vorlagen PAR-B-64–68 (Heuristik). |
| F-8 «Beginner» der PDFs ≠ Calisthenics-Anfänger | **Bestätigt** | Tuck Planche liegt auf OG-Level 5 von 16 [PAR-A-25]; Coaching: erste Tuck Planche nach 3–6 Monaten [A-40]. Die Wurzeln unter dem PDF-Niveau kommen aus `02` §4. |

**Eine Spannung, die der Planer auflösen muss:** Die PDFs setzen
supinierte/aussenrotierte Handstellungen (`supi`) ab Intermediate regelmässig
ein [P-02, P-03]. Stream D zeigt, dass der supinierte, fast gestreckte Arm unter
isometrischer Last die typische Rissposition der distalen Bizepssehne ist
[D-68, D-69]. Der Planer führt supinierte Straight-Arm-Varianten deshalb als
eigene Risikokategorie (PAR-D-41): nur ab Intermediate, mit Straight-Arm-Deckel,
Einstieg mit halbem Volumen (PAR-D-12) und ausgeschlossen bei Beschwerden in der
Ellenbeuge (`05` §8).

## 4. Zwischen den Streams abgestimmte Parameter

Wo Streams unterschiedliche Werte nennen, legt diese Tabelle den Wert für die
Spezifikation fest. Regel: Bei Sicherheitsgrössen gilt der strengere Wert, bei
Leistungsgrössen der besser belegte.

| Grösse | Werte in den Streams | Festlegung | Begründung |
|---|---|---|---|
| Pause vor Maximalversuch | B: 180–300, Standard 300, Maltese 420 (PAR-B-39); E: 300, Untergrenze 180, schwerste 420 (PAR-E-04, 05) | **300 s** (Untergrenze 180, schwerste Elemente 420) | übereinstimmend; Literatur 3–5 min, Praxis ≥ 5 min |
| Pause Volumensätze Statics | B: 120–240, Standard 180 (PAR-B-40); E: 180–300, Standard 240 (PAR-E-06) | **180 s** (Spanne 120–300) | PDF-Volumenpositionen 3–5 min; B-Standard am unteren Ende, E am oberen; Mittelwert der Praxis |
| Pause Leans/Zubringer | B: 90–120 (Heuristik, PAR-B-41); E: 120–180 (PAR-E-07); PDF 2–3 min | **120 s** (Spanne 90–180) | PDF-Werte liegen höher als B; 120 s deckt beide ab |
| Einheiten je Skill und Woche | B: 2–3, max 4 (PAR-B-34); E: Kraft-Skills 3 (2–4) (PAR-E-11); A: Coaching 2–4 (PAR-A-59) | **Anfänger 2, Trainierte 3, Maximum 4** | übereinstimmend; Maximum aus Sehnenlogik (S-2) |
| Balance-Skills (Handstand) | B: 3–7 × 5–10 min, Heuristik (PAR-B-35); E: 4 (3–6) × 11–15 min, A-Analogie (PAR-E-12, 35) | **4× pro Woche, 10–15 min**; bei Zeitmangel 5–10 min | E ist besser belegt (Meta-Analyse zu Gleichgewichtstraining, Analogie); B-Untergrenze als Kurzform; Handgelenk-Wochendeckel gilt zusätzlich |
| Abstand harter Reize derselben Struktur | B: 48 h hart / 24 h submaximal (PAR-B-38); E: 48 / 24 (PAR-E-13, 14); D: 48, in der Rampe 72 (PAR-D-08, 34) | **48 h, in der Rampe 72 h; 24 h nach submaximaler Technik** | übereinstimmend |
| Wöchentliche Steigerung | B: +10–20 % Sätze (PAR-B-55); D: Straight-Arm +10 %, Handgelenk +10 %, Bent-Arm +20 % (PAR-D-09–11) | **D-Werte je Struktur**; Vorverletzung × 0.5 (PAR-D-02) | Sicherheitsgrösse → strengerer, strukturgenauer Wert |
| Einheitsspitze | B: ≤ 110 % (Toleranz 120) des 28-Tage-Maximums (PAR-B-56); D: ≤ +10 % über 30-Tage-Maximum (PAR-D-31) | **≤ 110 % des 30-Tage-Maximums je Struktur** | nach Audit übereinstimmend; Analogie Laufen [D-36 = B-79] |
| Straight-Arm-Arbeitssätze je Einheit | B: 8–12 (bis Intermediate), bis 18 (Fortgeschrittene) (PAR-B-47); E: 12–18 je Skill-Einheit (PAR-E-09) | **8–12 bis Intermediate, bis 18 für Fortgeschrittene** über alle Straight-Arm-Skills der Einheit | E-Wert ist die PDF-Spanne für Fortgeschrittene; B differenziert nach Niveau |
| Maximalversuche je Skill und Einheit | B: 2–5, Standard 3 (PAR-B-08); E: 3 (2–5) (PAR-E-08) | **3 (2–5)**, solange keine Stoppregel greift | übereinstimmend |
| Satzhaltezeit | B: 2–10 s an der Zielstufe, 0.6–0.7 × Maximalhaltezeit (PAR-B-04, 06); A: 0.60–0.70 (PAR-A-64) | **0.6–0.7 × frische Maximalhaltezeit**, mindestens 2 s | übereinstimmend (OG-Tabelle, Isometrie-Review) |
| Stufenfenster | B: frische Maximalhaltezeit 3–20 s (PAR-B-05); A: ab ≈ 30 s nächste Stufe (PAR-A-65) | **Arbeiten bei 3–20 s; ab 20 s nächste Stufe anbieten, ab 30 s wechseln** | B für Maximalkraft, A als Obergrenze; beide Heuristik |
| Haltezeit → Intensität | A: Exponentialmodell (10 s ≈ 85 %, 20 s ≈ 71 %, 30 s ≈ 63 %) (PAR-A-63); B: Potenzmodell (20 s ≈ 85 %, 30 s ≈ 68 %) (PAR-B-14) | **Nicht für Entscheidungen verwenden**, nur für Erklärtexte | zwei Modellformen derselben Meta-Analyse [A-49 = B-10] widersprechen sich im Kurzzeitbereich; der Planer steuert über Haltezeit-Fenster, nicht über % MVC |
| Deload | B: Woche 6 (4–8), Sätze × 0.6, +2 RIR, Frequenz gleich (PAR-B-49–54); D: bei Schmerzverletzung −30 % Volumen, 1 Stufe Regression, 1 Woche (PAR-D-18) | **Geplant: B-Werte; schmerzausgelöst: D-Werte**, beide als Deload erfasst | unterschiedliche Auslöser, keine Konkurrenz |
| Wiedereinstieg | B: Rampen nach Pausendauer (PAR-B-59–62); D: ab 4 Wochen Pause Straight-Arm in RTT-Stufe 1 mit 25 % (PAR-D-29, 33) | **Allgemein B, für Straight-Arm-/Handgelenk-Strukturen D** (strenger gewinnt) | Sehne verliert schneller als Kraft [D-18, D-19] |
| Neue Belastungsart | D: Woche 1 mit 50 % des Zielvolumens (PAR-D-12) | **50 %** | ohne Gegenwert |
| Aufwärmen | B: 5/7/10/12–15 min je Einheitslänge, 2–3 Rampensätze (PAR-B-69, 70); E: 5–10 min + 2–3 Rampensätze (PAR-E-26, 47) | **B-Tabelle nach Einheitslänge, 2–3 Rampensätze** | übereinstimmend |
| Unlock-Schwellen | A: Zwischenstufe 10 s, Endstufe 3 s, Form ≥ 4, occ 2, 28 d (PAR-A-10, 11, 17–19) | **A-Werte**; DSL-Erweiterungen `min_distinct_days` und `min_load_pct_bw` prüfen | einzige Quelle; FIG/WSWCF als Anker |
| Konfidenzklassen | F: σ/μ < 0.15 / 0.15–0.30 / ≥ 0.30 → Dosis aus μ / μ − 0.5σ / μ − σ + Testsatz (`07` §8.6) | **F-Werte** | einzige Quelle |

## 5. Widersprüche

Die Stream-Dateien dokumentieren ihre internen Widersprüche einzeln (`02` W-1
bis W-22, `03` W-1 bis W-24, `04` 1–17, `05` W-1 bis W-16, `06` W-1 bis W-22,
`07` 1–18). Hier stehen die, die den Planer direkt betreffen, und die, die
zwischen Streams auftreten.

| Thema | Position 1 | Position 2 | Auflösung im Planer |
|---|---|---|---|
| Pausenlänge für Kraft | > 2 min bei Trainierten nötig [B-55]; 3–5 min klassisch [B-32, E-87] | ACSM 2026: kein Einfluss kurzer vs. langer Pausen auf Kraft [B-31] | Lange Pausen für Maximalversuche (Qualität, Lernen), kürzere für Zubringer (§4) |
| Nutzen der Periodisierung | ES 0.43 für 1RM [B-01] | Grundannahmen methodisch kaum geprüft; ACSM 2026 stuft sie herab [B-05, B-06, B-31] | Einfache Modelle (Rang 16), keine komplexen Blöcke |
| 10-%-Regel und ACWR | Praxisregel; ACWR «Sweet Spot» [D-27] | RCT ohne Effekt [D-33]; ACWR mit Artefakten, bei Läufern umgekehrt [D-29, D-30, D-36] | Keine ACWR-Sperre; Deckel je Struktur und Einheitsspitze (S-2), als Heuristik markiert |
| Haltezeit ↔ Intensität | Exponentialmodell [A-49] | Potenzmodell derselben Meta-Analyse [B-10] | Nur Texte, keine Steuerung (§4) |
| Contextual interference | Laborvorteil SMD 0.92 [E-37] | angewandt n. s., nur 20 % der Ergebnisse passen [E-36, E-39] | Maximalversuche geblockt, Variation zwischen Einheiten (Rang 17) |
| Schlaf und Konsolidierung | +20 % Tempo nach einer Nacht [E-61] | nach Kontrolle kein Effekt [E-62, E-65] | Keine Regel auf «Schlaf verstärkt Lernen» (Rang 18) |
| Balance-Frequenz | 3–7 × 5–10 min (B, Heuristik) | 3–6, am besten 3 oder 6 × 11–15 min (E, Meta-Analyse, Analogie) | E-Wert (§4) |
| Dauer bis zur nächsten Stufe | Heuristik PAR-A-45: Straddle Planche in 24–84 Wochen | Coaching: 12–24 Monate [A-40], 2–4 Jahre bis A-Elemente [A-67] | Obere Hälfte der Bänder für den Realismus-Check (Rang 20) |
| Planche an Ringen vs. Boden | OG: Ringe 1–3 Level höher [A-31] | FIG: gleicher Wert an beiden Geräten [A-29] | OG-Offsets für Ersatzlogik; FIG nur für «gehalten» |
| Winkeltoleranz | FIG: > 5–20° kleiner Fehler [A-29] | WSWCF: ±7.5° [A-33] | Form 4 = ≤ 15° für Unlocks (strenger als FIG, lockerer als WSWCF) |
| Was «Calisthenics» in Studien ist | Street Workout / Bodyweight [D-01, D-02] | australischer Performance-Sport aus Turnen und Tanz [D-03] | D-03 nicht für Street-Workout-Aussagen verwenden |
| Vorverletzung als stärkster Risikofaktor | häufig so zitiert | im Street Workout hatte hohe körperliche Aktivität eine höhere Odds Ratio (4.37–5.63) als Vorverletzung (4.08) [D-02] | Beide als Deckel-Modifikator; Vorverletzung × 0.5 (PAR-D-02, Heuristik) |
| Supinierte Straight-Arm-Varianten | PDF-Standard ab Intermediate [P-02, P-03] | typische Rissposition der distalen Bizepssehne [D-68, D-69] | Eigene Risikokategorie (§3) |
| Selbsteinschätzung: Richtung | erinnerte Testwerte zu hoch [F-51, F-52] | Haltezeiten im Voraus eher unterschätzt [F-54] | Korrektur −5 % nur für erinnerte Wdh.; Halte ohne Korrektur, breites σ |
| RIR-Genauigkeit und Übung | Erfahrung verbessert tendenziell [F-37] | kein Einfluss in Meta-Analyse und Studie [F-08, B-130] | Kalibrierungs-Testsatz statt Vertrauen auf Übung (Rang 9) |
| Frauen: relative Kraft | absolut ≈ 52 % der Männer [C-84] | relativ zum Körpergewicht ≈ 80 % [C-77] | Nur relativer Wert als Prior für Zeitschätzungen, nie als Sperre (PAR-C-54) |
| Muskelmasse vs. Sehne nach Pause | Kraft nach 1–2 Monaten erhalten [D-18, D-19; B-84] | Sehnensteifigkeit zurück auf Ausgangswert [D-18, D-19] | Strengere Rampe für Straight-Arm (§4) |

## 6. Offene Fragen

### 6.1 Forschungslücken (nicht mit Literatur lösbar, später aus Logs)

1. Dosis-Wirkung und Erholung für Straight-Arm-Statics (Planche, Lever,
   Maltese); keine Interventionsstudie existiert [`03` Kurzfassung].
2. Sehnen-Zeitverläufe der oberen Extremität; alle Daten stammen von Achilles-
   und Patellarsehne [D-17–D-19].
3. Reliabilität von Halte-Tests (L-Sit, Hollow, Stütz, Handstand am Boden) und
   Pistol-Max-Wdh. [`07` Offene Fragen].
4. Validierte Bereitschafts-Schwellen (Muscle-up, Front Lever, Planche); nur
   Coaching-Angaben mit Widersprüchen [F-70–F-76].
5. Lernzeiten bis zur nächsten Stufe; nur Coaching-Angaben [`02` §3.6].
6. Band-Assistenz: Zählweise als Trainingsreiz [`03` §3.5].
7. Interferenz zwischen zwei Straight-Arm-Skills [`03` §7].

Für alle sieben gilt: Der Planer startet mit markierten Heuristiken und
protokolliert, was er für eine spätere Auswertung braucht (Stufe, Haltezeit,
Form, Assistenz, Schmerz, Pausen). Eine Kalibrierung aus aggregierten,
einwilligten Logs ist ein eigener späterer Schritt.

### 6.2 Entscheidungen für den Checkpoint

| Nr. | Frage | Vorschlag |
|---|---|---|
| D-1 | Name und Client: Der Auftrag nennt rung.fit mit Svelte; das Repository ist Hefesto mit iOS-Client. | API bleibt client-neutral (OpenAPI); ein Svelte-Client kann generiert werden. Klären, ob rung.fit ein neuer Name ist. |
| D-2 | Wiedereinstieg vs. `CONTENT_AUTHORING.md` («keine Return-to-Training-Ratschläge») | Rampe als Trainingslaststeuerung formulieren, nicht als Rehabilitation; Red Flags → Verweis; Content-Regel um diese Unterscheidung ergänzen (ADR in Phase 4). |
| D-3 | Minderjährige zulassen? | Rechtlich klären; wenn ja, PAR-D-23 und RF-12/13. |
| D-4 | Gesundheitsdaten (Beschwerden, Schmerzwerte, Screening) | Eigene Einwilligung, getrennte Tabellen, Löschung mit Konto. |
| D-5 | DSL-Erweiterungen `min_distinct_days`, `min_load_pct_bw` | Additiv in Phase 4 spezifizieren (Golden Files zuerst, ADR 0008). |
| D-6 | Implizite Vorstufe je Skill (`states.go`) | Beibehalten; Geräte- und Varianten-Stufen als eigene Skills mit `recommended`-Kanten (wie in `02` umgesetzt). |
| D-7 | Neue Daten im Log: Schmerzwerte, optional Check-in (Schlaf, Tagesform) | Schmerz ja (`user_pain_reports`); Check-in optional und ohne Streak-Folgen (ADR 0003). |
| D-8 | PAR-Q+ ist urheberrechtlich geschützt | Eigene Formulierung nach ACSM-Logik. |
| D-9 | Gewichtete Stufen bis `min_load_pct_bw` existiert | Übergangsweise absolute Last mit 75-kg-Referenz (PAR-A-67) oder nur Selbstbestätigung. |
| D-10 | Umfang der Wissensbasis für v1 | Alle Skills aus `02`, aber nur mit belegten oder als Heuristik markierten Zahlen; `status: draft_placeholder` bleibt, bis ein Mensch die Einträge abgenommen hat. |

## 7. Was die Spezifikation (Phase 4) daraus übernehmen muss

- **Übungs-Metadaten** in der Wissensbasis: Bewegungsmuster, Straight-Arm-
  Kategorie (PAR-C-49), Gelenk-/Struktur-Lastprofil 0–3 (`04` Tabelle 2,
  PAR-C-44/45), Supinations-Risiko (PAR-D-41), limitierender Faktor
  Kraft/Balance (PAR-E-28), Stufen-Momentverhältnis (PAR-C-03–08),
  Regressions- und Progressionsnachbarn, Equipment aus geschlossenem Vokabular.
- **Nutzerzustand**: Kapazitätsschätzungen mit σ, Strukturstatus je Region
  (normal, Rampenstufe, gesperrt), Belastungshistorie je Struktur (3-Wochen-
  Mittel, 30-Tage-Maximum), Schmerzberichte, Ziele, Verfügbarkeit, Equipment.
- **Regeln** aus §2 mit den Werten aus §4, jede mit Regel-ID und Quellen-ID für
  die Erklärbarkeit.
- **Ausgabe** als geplante Sätze im bestehenden Log-Modell (`is_planned`,
  `kind`, `rest_after_planned_s`) oder als Templates; eine Kombination bleibt ein
  `set_entry` mit mehreren Elementen.
- **Guardrails**: keine XP für Maximalversuche und Tests, Deloads und
  Lastreduktionen zählen im Streak, keine Texte über Rückstand (ADR 0003).
