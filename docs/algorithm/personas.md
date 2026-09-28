# Personas: erzeugte Pläne und Plausibilität

Stand: Phase 5 nach dem unabhängigen Review, Regelwerk `0.1.0`
(`content/training/`), 28.09.2026.

Dieses Dokument zeigt, was der Planer für die sechs Personas aus `spec.md`
§12.4 erzeugt, und prüft die Pläne gegen die Erwartungen der Spezifikation und
die Recherche. Die Pläne stammen aus den Golden Files, die die Tests bei jeder
Änderung vergleichen. Die frühere Fassung dieses Dokuments war zu positiv; der
Review hat das zu Recht bemängelt (`review.md`, C und M-4).

| Datei | Inhalt |
|---|---|
| `internal/domain/planning/testdata/<persona>.txt` | Onboarding-Ergebnis mit Hinweisen und Plan der ersten Woche mit allen Reasons |
| `internal/domain/planning/testdata/<persona>.weeks.txt` | Verlauf über 12 simulierte Wochen mit Sprossen, Änderungen und Zustand der Regionen |
| `internal/domain/planning/personas_test.go` | Eingaben der Personas, Erwartungen aus §12.4 |
| `internal/domain/planning/scenarios_test.go` | Szenarien aus §12.5 und Tests zu Pausen |

Neu erzeugen: `go test ./internal/domain/planning -update`; jede Änderung an
einem Golden File ist eine Review-Entscheidung (§12.3).

## Methode

**Woche 1.** `Start` wandelt die Onboarding-Antworten in den Startzustand um,
`Generate` erzeugt die Woche ab Montag, 28.09.2026. Persona 6 beantwortet ihre
Rückfragen mit «nein».

**Zwölf Wochen.** Ein simulierter Athlet führt jede geplante Einheit aus und
loggt sie; danach folgen `Adapt` für die Einheit, die Schmerzberichte, das
Wochenereignis mit dem Spielraum des Plans und der nächste Plan. Der Athlet ist
ein Testmodell, keine Physiologie:

- Er hat je Übung eine «wahre» Maximalleistung (Wdh. oder Sekunden). Leichtere
  Sprossen ohne Angabe schafft er mit dem 1.5-Fachen plus 4, schwerere mit
  dem Umgekehrten. Mit Band hält er das 1.5-Fache plus 3.
- Er führt jeden Satz wie geplant aus, höchstens bis zu seinem Maximum, und
  meldet die Reserve ehrlich. Kalibrierungssätze beendet er 2 vor dem
  Maximum. Angebote nimmt er an.
- Trainierte Übungen werden je Woche 3 % besser.
- Regionen mit Beschwerde meldet er nach jeder Einheit mit 1/10 während der
  Einheit und am nächsten Morgen.

Jeder Plan jeder simulierten Woche muss die Eigenschaften I-1 bis I-11
erfüllen (§12.1). Sonst scheitert der Test.

**Grenzen der Simulation.** Die Leistung wächst in Wochenschritten statt je
Einheit. Das Modell kennt weder Tagesform noch Ermüdung, und die Schmerzwerte
sind konstant. Die Simulation zeigt, wie die Regeln greifen, nicht wie schnell
ein echter Mensch Fortschritte macht. «Sätze» in den Wochenzeilen zählen
Aufwärm- und Prehab-Sätze mit.

## Persona 1: Anfänger, Outdoor-Park, 2× pro Woche, Ziel Muscle-up

**Eingaben:** Park (Klimmzugstange, niedrige Stange, Barren), 2 × 45 min;
1–3 Klimmzüge, 0 Dips, 10–30 s Stütz, 8–12 Liegestütze; unter 6 Monaten
Training, trainiert aktuell.

**Woche 1** (Mo, Do; je ≈ 14 min einschliesslich 7 min Aufwärmen):
- Negativer Dip und negativer Klimmzug, je 1 × 3 Wdh. à 3 s.
- Dead Hang 1 × 10 s, Stütz am Barren 1 × 4 s, fast aufrechtes Rudern 1 × 5,
  alle als Kalibrierung.
- Handgelenk-Prehab im Aufwärmen und am Ende.
- Kein Muscle-up-Block; der Hinweis GOAL-04 nennt «5 + 5» als Faustregel, nicht
  als Sperre. Ring-Übungen sind ausgeschlossen (SEL-02).

**Verlauf:**
- Das Rudern wechselt in Woche 2 zur schrägen Variante; in Woche 4 und 11
  kommt je ein Kalibrierungssatz an der waagrechten Variante (ADAPT-07).
- Die Negativen verlängern sich von 3 auf 7 s. Ab Woche 5 erscheinen
  Barren-Dips.
- Der Klimmzug bleibt zwölf Wochen negativ: Der Athlet schafft 2 Klimmzüge,
  mit 3 % je Woche knapp 3; eine konzentrische Wiederholung mit Reserve 2 ist
  nicht möglich. Angekündigte Angebote scheitern ab Woche 3 an den knappen
  Deckeln (§15.3).
- Volumen 14 → 16 Sätze; Deloads in Woche 6 und 12 mit 14 Sätzen.
- Szenario §12.5: Mit 6 Klimmzügen und 7 Dips wird die Muscle-up-Leiter
  innerhalb von 16 Wochen aktiv (`TestScenarioMuscleUpActivates`).

**Plausibilität: sicher, langsam.**
- Die Zubringer entsprechen der Recherche: Hang, Stütz, Negative, Rudern
  (`02` §4.2, §4.7, §6.1).
- Die Einheiten nutzen die 45 min kaum (§15.4).
- Abweichung von §12.4: Die Liegestütz-Leiter erscheint nicht; Zug und Dip
  sind vertreten, also greift GOAL-06 nicht. Die Leiter wird bei 5 Klimmzügen
  und 8 Dips aktiv, den Schwellen der Stufen.

## Persona 2: Fortgeschritten, Gym, 4× pro Woche, Planche und Front Lever

**Eingaben:** Gym mit Ringen, Band und Parallettes, 4 × 60 min; Tuck Planche
10–19 s, Advanced Tuck Front Lever 4–9 s; 12–15 Klimmzüge; 1–4 Jahre Training,
trainiert aktuell.

**Woche 1** (Mo, Di, Do, Sa; 13–21 min):
- Mo, Do, Sa: Planche Lean 1–2 × 4 s als Konditionshalt (DOSE-04) und Tuck
  Front Lever mit Band 1–2 × 2 s (Technik-Dosierung aus der Untergrenze, U-31),
  beide als Kalibrierung; Klimmzüge 1 × 6.
- Dienstag hat keine Straight-Arm-Arbeit, nur Plank und Hollow Hold (§12.4:
  «vierter Tag ohne Straight-Arm-Sprossen»).
- Prehab: Band-Aussenrotation und Handgelenk-Vorbereitung.

**Verlauf:**
- Planche: Woche 2 an der Tuck mit einem Angebot an der Advanced Tuck, ab
  Woche 3 an der Advanced Tuck.
- Front Lever: Woche 3 an der Tuck ohne Band, ab Woche 4 an der Advanced Tuck
  (der anteilige σ-Boden macht die kurzen Halte erreichbar, U-28).
- 7–11 Straight-Arm-Sätze je Woche; Deload in Woche 6 (16 statt 20 Sätze, 6
  statt 9 Straight-Arm-Sätze) und ein Stagnations-Deload in Woche 10.
- Klimmzüge bleiben im Plan, nach erreichter Empfehlung mit Erhaltungsdosis
  (1 Einheit, ≤ 2 Sätze, U-29); der Hollow Hold entfällt nach der Tuck-Stufe,
  weil die Empfehlung an ihr hängt (`02` §8).

**Plausibilität: Progression plausibel, Volumen niedrig.**
- Die Sprossen folgen der Leistung des Athleten; Angebote, Kalibrierung und
  Pausen (300 s am Front Lever) passen zu §12.4.
- Der Einstieg ist doppelt vorsichtig: Lean 4 s für eine Tuck von 10–19 s,
  Tuck Front Lever mit Band für eine Advanced Tuck (I-1, teilweise). Die erste
  Woche korrigiert beides.
- 9 Straight-Arm-Sätze je Woche ab Woche 7 sind für einen Fortgeschrittenen
  wenig. Ursache ist LOAD-02 auf dem kleinen Einstiegsvolumen (ENT-R-2).

## Persona 3: Wie 2, mediale Ellbogenbeschwerden, Ziel Planche

**Eingaben:** wie Persona 2, Ziel nur Planche; aktuell Schmerz innen am
Ellbogen (Alltag 1.5/10, Training 3.5/10, schleichend, seit 6 Wochen), keine
Red Flag.

**Onboarding:** Region `rtt_1` mit Start 0.5; Hinweis auf eine Fachperson,
weil die Beschwerde länger als 4 Wochen besteht (INJ-08). Der Alltagswert 1.5
ist der erste Basiswert des Monitorings (U-30).

**Woche 1:**
- Planche Lean 1 × 4 s nur Mo und Do: 72 h Abstand in der Rampe (PAR-D-34),
  Frequenz begründet mit WEEK-04; die Planche-Familie ist `M` (INJ-05).
- Klimmzug mit Band 1 × 6 (M bei Wiederholungen, U-24; vorher unverändert,
  Review I-7); Plank 1 × 10 s; alle mit Schmerz-Monitoring.
- Samstag enthält nur einen Plank-Satz (ENT-R-5).
- Die Red-Flag-Fragen für die Region liegen im Plan (RF-01 bis RF-07, RF-10);
  keine Angebote (ADAPT-06a).

**Verlauf mit 1/10:**
- Die Rampe steigt je Woche: `rtt_2` (Woche 2), `rtt_3`, `rtt_4`, `rtt_5`
  (Woche 5); nach zwei Wochen ohne Verletzung ist die Region wieder `normal`
  (`PAR-S-32`).
- In der Rampe wählt der Planer die Tuck; die M-Zelle senkt sie auf die Lean
  mit Technik-Halten bis 10 s (U-24). Ab Woche 7 arbeitet die Planche an der
  Tuck.
- 2–3 Straight-Arm-Sätze je Woche.

**Szenarien:**
- 6/10 nach Einheit 2 (`TestScenarioElbowPain`): Red-Flag-Fragen,
  Schmerz-Deload (PAR-D-18); die dritte Verletzung in 14 Tagen erzeugt den
  Verweis (INJ-08). Alle Reasons zur Region tragen die Region (EXPL-07).
- Soreness-Regeln (`TestRampPainRules`): Folgetag-Soreness wiederholt den
  Schritt mit einem Tag Pause, anhaltender Aufwärmschmerz geht einen Schritt
  zurück mit zwei Tagen Pause, 8/10 über 1 h löst beides aus.
- RF-10 stoppt das Training; die Freigabe der Region hebt den Stopp auf
  (`TestScenarioRedFlagStops`).

**Plausibilität: plausibel, vorsichtig.** Die Last wird reduziert, nicht
pausiert (PAR-D-40). Für den Ellbogen gibt es kein Prehab, weil die Recherche
keine übertragbare Übung nennt.

## Persona 4: Wiedereinsteiger nach 6 Monaten Pause

**Eingaben:** Pause 17–26 Wochen, vorher Advanced Tuck Planche; 8–11
Klimmzüge; Gym mit Ringen, 3 × 60 min, 1–4 Jahre Training.

**Woche 1:**
- Pause aus dem Onboarding: 119 Tage (`PAR-S-41`), keine geloggte Referenz.
- Planche Lean 1 × 10 s als Kalibrierung, zwei Sprossen unter dem Stand vor der
  Pause (§6.11).
- Straight-Arm- und Handgelenk-Konten: LOAD-04 mit der Rampe (25 %) als
  zusätzlicher Obergrenze. U-13 ist zurückgenommen (Review A-3).
- Klimmzüge als Gegenrichtung (GOAL-06); alle Schätzungen σ × 1.25.

**Verlauf:**
- Die Rampe steigt je Woche (Stufe 1 → 3 bis Woche 3) und endet nach Woche 7.
- Woche 2: Kalibrierungssatz an der Tuck; ab Woche 2 arbeitet die Planche an
  der Tuck, eine Sprosse unter dem Stand vor der Pause (U-25).
- Ein Straight-Arm-Satz je Woche über alle zwölf Wochen: LOAD-02 wächst vom
  Einstieg aus mit 7.5 % je Woche; ein zweiter Satz passt erst nach etwa 13
  Wochen.

**Plausibilität: sicher, aber zu wenig Straight-Arm-Volumen.** Das ist die
offene Entscheidung ENT-R-1: Die Rampe auf das Zielvolumen (U-13) wäre
schneller, lässt aber ohne Logs ein Niveau zu, das der User nie nachweislich
getragen hat.

## Persona 5: Anfänger, Full Planche in 8 Wochen

**Eingaben:** Ziel Full Planche mit Datum in 8 Wochen, keine Planche-Stufe;
4–7 Liegestütze, Hollow unter 15 s; Boden und Wand, 3 × 45 min, noch nie
regelmässig trainiert.

**Woche 1:**
- Liegestütze mit erhöhten Händen 2 × 5 (Mo, Fr) und Plank 1 × 10 s (Mo, Mi),
  alle als Kalibrierung; der Mittwoch enthält nur den Plank (ENT-R-5).
- Kein Planche-Block: Die Lean setzt die Stufe `push-up/full` voraus.
- Hinweise: Gegenrichtung fehlt, weil keine Stange angegeben ist (GOAL-06,
  SEL-02).

**Realismus-Check (GOAL-05):** Untergrenze 48 Wochen, angezeigte Spanne
105–162 Wochen, Datum in 8 Wochen → «unrealistisch». Etappenziel Tuck Planche
mit 28.5–45 Wochen aus dem Bandmodell; das Coaching nennt 2–6 Monate
(PAR-A-47). Der Plan folgt der Arbeitsstufe, nicht dem Datum.

**Verlauf:**
- Erhöhte Liegestütze wachsen von 2 × 5 auf 3 × 14 je Einheit (Woche 11);
  wöchentlich ein Kalibrierungssatz an normalen Liegestützen (ADAPT-07).
- Die Stufe `push-up/full` (8 Wdh.) wird in 12 Wochen nicht erreicht: Der
  Athlet startet mit 6 und schafft nach 11 Wochen gut 8; die vorsichtige Dosis
  (μ − σ) liegt darunter.
- Deloads in Woche 6 und 12 (10 statt 13–15 Sätze).

**Plausibilität: plausibel.** Der Realismus-Check entspricht §12.4; das
Volumen wächst seit PAR-S-48 (Review C-1).

## Persona 6: Widersprüchliche Angaben

**Eingaben:** inaktiv (`sedentary`), 0 Liegestütze, 0 Klimmzüge; angegeben
Straddle Planche und Full Front Lever, je 4–9 s; Park, 3 × 60 min.

**Onboarding:**
- Zwei Rückfragen (R-3): «Hältst du die Stufe «Straddle Planche» ohne Band und
  mit gestreckten Armen?» und dieselbe für den Front Lever.
- Mit «nein» löst der Planer konservativ auf (R-3, dann R-2); weil Liegestütze
  und Klimmzüge fehlen, bleibt keine Planche- und keine Front-Lever-Stufe.
  Der Text SEL-12 erklärt das je Skill (vorher irreführend SEL-08, Review M-1).

**Woche 1:** nur Grundlagen, 16–18 min:
- Liegestütze mit erhöhten Händen 2 × 5, Plank, gehockter Hollow Hold 1 × 10 s.
- Negativer Klimmzug 1 × 3 à 3 s, Dead Hang 1 × 4 s, fast aufrechtes Rudern
  1 × 5.
- Kein Straight-Arm-Satz; der Front-Lever-Hinweis nennt die Einstiegswerte aus
  dem Coaching.

**Verlauf:** Hollow Hold ab Woche 3 gestreckt, Rudern bis zur waagrechten
Variante, Liegestütze zwischen Wand und erhöhten Händen (der Athlet schafft
einen Liegestütz); Volumen 20 → 23 Sätze.

**Plausibilität: plausibel.** Höchstens zwei Rückfragen, keine stille
Übernahme der hohen Angaben, alle Startwerte mit Kalibrierung (§12.4).

## Zusammenfassung

| Persona | Erwartung §12.4 | Stand |
|---|---|---|
| 1 | erfüllt, ohne Liegestütz-Leiter | sicher; Klimmzug bleibt negativ; kurze Einheiten |
| 2 | erfüllt | Progression gut; Volumen niedrig (ENT-R-2); doppelt vorsichtiger Einstieg |
| 3 | erfüllt | Rampe, Monitoring, Soreness-Regeln, Band am Klimmzug |
| 4 | erfüllt ohne U-13 | ein Straight-Arm-Satz je Woche (ENT-R-1) |
| 5 | erfüllt | Volumen wächst; Mittwoch fast leer (ENT-R-5) |
| 6 | erfüllt | – |

Übergreifende Befunde: `spec.md` §15.4; offene Entscheidungen: §15.5; alle
Befunde des Reviews: `review.md`.
