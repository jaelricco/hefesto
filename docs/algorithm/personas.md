# Personas: erzeugte Pläne und Plausibilität

Stand: Phase 5, Regelwerk `0.1.0` (`content/training/`), 27.09.2026.

Dieses Dokument zeigt, was der Planer für die sechs Personas aus
`spec.md` §12.4 erzeugt, und prüft die Pläne gegen die Erwartungen der
Spezifikation und die Recherche. Die Pläne sind keine Handarbeit: Sie
stammen aus den Golden Files, die die Tests bei jeder Änderung neu vergleichen.

| Datei | Inhalt |
|---|---|
| `internal/domain/planning/testdata/<persona>.txt` | Onboarding-Ergebnis und Plan der ersten Woche mit allen Reasons |
| `internal/domain/planning/testdata/<persona>.weeks.txt` | Verlauf über 12 simulierte Wochen |
| `internal/domain/planning/personas_test.go` | Eingaben der Personas, Erwartungen aus §12.4 |
| `internal/domain/planning/scenarios_test.go` | Szenarien aus §12.5 |

Neu erzeugen: `go test ./internal/domain/planning -update`; jede Änderung an
einem Golden File ist eine Review-Entscheidung (§12.3).

## Methode

**Woche 1.** `Start` wandelt die Onboarding-Antworten in den Startzustand um,
`Generate` erzeugt die Woche ab Montag, 28.09.2026. Persona 6 beantwortet ihre
Rückfragen mit «nein».

**Zwölf Wochen.** Ein simulierter Athlet führt jede geplante Einheit aus und
loggt sie; danach folgen `Adapt` für die Einheit, das Wochenereignis mit dem
Spielraum des Plans und der nächste Plan. Der Athlet ist ein Testmodell,
keine Physiologie:

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
sind konstant. Die Simulation zeigt deshalb, wie die Regeln greifen, nicht wie
schnell ein echter Mensch Fortschritte macht.

## Persona 1: Anfänger, Outdoor-Park, 2× pro Woche, Ziel Muscle-up

**Eingaben:**
- Park-Ausrüstung (Klimmzugstange, niedrige Stange, Barren), 2 × 45 min.
- 1–3 Klimmzüge, 0 Dips, 10–30 s Stütz, 8–12 Liegestütze.
- Unter 6 Monate Training, trainiert aktuell.

**Woche 1** (Mo, Do; je ≈ 14 min Training plus Aufwärmen):
- Negativer Dip und negativer Klimmzug, je 1 × 3 Wdh. à 3 s.
- Dead Hang 1 × 10 s, Stütz am Barren 1 × 4 s, fast aufrechtes Rudern 1 × 5 (Kalibrierung).
- Handgelenk-Prehab im Aufwärmen und am Ende.

Kein Muscle-up-Block. Der Hinweis GOAL-04 nennt die Faustregel «5 + 5» als
Hinweis, nicht als Sperre. Ring-Übungen sind mit Begründung SEL-02
ausgeschlossen.

**Verlauf:**
- Das Rudern steigt über die Doppelprogression zur schrägen und waagrechten
  Variante.
- Negative bleiben, weil der Athlet mit 2 Klimmzügen startet. Ab Woche 4
  erscheinen konzentrische Angebote an Klimmzug und Barren-Dip.
- Deloads laufen in Woche 6 und 12 (Mesozyklus 6 Wochen).
- Szenario §12.5: Mit 6 Klimmzügen und 7 Dips wird die Muscle-up-Leiter
  innerhalb von 16 Wochen aktiv, nicht in Woche 1 (`TestScenarioMuscleUpActivates`).

**Plausibilität: plausibel.**
- Die Zubringer entsprechen der Recherche: Hang, Scapula, Negative, Rudern
  (`02` §4.2, §4.7, §6.1).
- Die sehr kleine erste Woche folgt aus LOAD-04b (50 %). Sie ist gewollt
  vorsichtig, nutzt die 45 min aber kaum (§15.4).
- Abweichung von §12.4: Die Liegestütz-Leiter erscheint nicht. Das Ziel
  Muscle-up zieht Zug und Dip nach, beide Richtungen sind vertreten, also
  greift GOAL-06 nicht.
- Anpassung von §12.5: Die Leiter wird bei 5 Klimmzügen und 8 Dips aktiv (die
  Unlock-Schwelle von `dip/parallel-bars`), nicht bei 5 + 5.

## Persona 2: Fortgeschritten, Gym, 4× pro Woche, Planche und Front Lever

**Eingaben:**
- Gym mit Ringen, Band und Parallettes, 4 × 60 min.
- Tuck Planche 10–19 s, Advanced Tuck Front Lever 4–9 s.
- 12–15 Klimmzüge, 1–4 Jahre Training, trainiert aktuell.

**Woche 1** (Mo, Di, Do, Sa):
- Mo, Do, Sa: Maximalblock mit Planche Lean 2 × 4 s und Tuck Front Lever mit
  Band 1 × 5 s, beide als Kalibrierungssätze mit 300 s Pause; dazu Klimmzüge,
  Plank und Hollow Hold.
- Dienstag hat keine Straight-Arm-Arbeit, nur Rumpf und Handgelenk
  (§12.4: «vierter Tag ohne Straight-Arm-Sprossen»).
- Prehab: Band-Aussenrotation und Handgelenk-Vorbereitung.

**Verlauf:**
- Woche 2: Das Band-Angebot führt zur Tuck ohne Band. Die Planche wechselt
  nach zwei Kalibrierungseinheiten von der Lean zur Tuck (SEL-08).
- Wochen 2–3: Angebote an der Advanced Tuck. Der Athlet hält sie nur ≈ 7 s,
  deshalb kein Wechsel.
- 18–22 Straight-Arm-Sätze je Woche, geplanter Deload in Woche 6,
  Stagnations-Deload in Woche 10.

**Plausibilität: plausibel mit einer Auffälligkeit.**
- 2 × 4 s Lean für jemanden, der die Tuck Planche 10–19 s hält, ist zu leicht.
  Das folgt aus SEL-08 (erste zwei Einheiten eine Stufe unter der Angabe) und
  `PAR-S-39` (die leichtere Sprosse erbt nur die Untergrenze). Die erste
  Einheit korrigiert es, weil beide Sätze Kalibrierungen sind (§15.4).
- Der Band-Einstieg am Front Lever ist vorsichtig (Angabe 4–9 s an der
  Advanced Tuck, SEL-07) und nach einer Woche aufgelöst.
- Pausen, Budget (≤ 12 Sätze je Einheit) und 48 h Abstand hält jeder Plan ein
  (I-3, I-4).

## Persona 3: Wie 2, mediale Ellbogenbeschwerden, Ziel Planche

**Eingaben:** wie Persona 2, Ziel nur Planche; aktuell Schmerz innen am
Ellbogen (Alltag 1.5/10, Training 3.5/10, schleichend, seit 6 Wochen), keine
Red Flag.

**Woche 1:**
- Region `elbow_inner` in Rampenstufe 1 (Start 50 %, INJ-03).
- Planche Lean 2 × 4 s nur Mo und Do: 72 h Abstand in der Rampe (PAR-D-34),
  Frequenz begründet mit WEEK-04. Die Planche-Familie ist `M` (INJ-05, eine
  Sprosse tiefer).
- Klimmzug `M`, Plank `S`, beide mit Schmerz-Monitoring (`monitor`).
- Die Red-Flag-Fragen für die Region liegen im Plan (RF-01 bis RF-07, RF-10).
- Keine Angebote (ADAPT-06a).

**Verlauf mit 1/10:**
- Die Rampe steigt über `rtt_2` bis `rtt_5` und ist nach 2 Wochen ohne
  Verletzung wieder `normal` (`PAR-S-32`).
- Ab Woche 6 arbeitet die Planche an der Tuck.

**Szenario 6/10 nach Einheit 2** (§12.5, `TestScenarioElbowPain`):
- Red-Flag-Fragen und ein Schmerz-Deload (PAR-D-18).
- Die dritte Verletzung in 14 Tagen erzeugt den Verweis-Hinweis (INJ-08).
- Alle Reasons zur Region tragen die Region, damit Clients keine Quellentitel
  zeigen (EXPL-07).

**Szenario RF-10:** Das Training stoppt; die Freigabe der Region hebt den Stopp
auf (`TestScenarioRedFlagStops`).

**Plausibilität: plausibel.**
- Wie in `05` §6.2 vorgesehen wird die Last reduziert, nicht vollständig
  pausiert (PAR-D-40).
- Klimmzug `M` bleibt dieselbe Übung: Die Wissensbasis hat noch keinen
  Neutralgriff (SEL-10, §15.3).
- Für den Ellbogen gibt es kein Prehab, weil die Recherche keine übertragbare
  Übung nennt.

## Persona 4: Wiedereinsteiger nach 6 Monaten Pause

**Eingaben:**
- Pause 17–26 Wochen, vorher Advanced Tuck Planche.
- 8–11 Klimmzüge, Gym mit Ringen, 3 × 60 min, 1–4 Jahre Training.

**Woche 1:**
- Pause aus dem Onboarding: 119 Tage (`PAR-S-41`).
- Planche Lean 1 × 5 s: zwei Sprossen unter dem Stand vor der Pause, nur in
  Woche 1 (§6.11).
- Straight-Arm- und Handgelenk-Konten stehen bei 25 % des Zielvolumens
  (ADAPT-16).
- Klimmzüge sind Unterstützung (GOAL-06). Alle Schätzungen sind um 1.25
  verbreitert (`PAR-S-44`).

**Verlauf** (§12.5, `TestScenarioReturnerRamp`):
- Die Rampe steigt je Woche: 25 % → 50 % → 75 % → 100 % in Woche 1–4
  (1 → 6 → 9 Straight-Arm-Sätze).
- Nach 9 Wochen endet die Pause. Ab Woche 10 kommen Angebote; ab Woche 11
  arbeitet die Planche an der Tuck.

**Plausibilität: plausibel, bewusst vorsichtig.**
- Ohne Prüfversuche während der Rampe bleibt die Planche 9 Wochen an der Lean,
  obwohl der Athlet die Tuck hielte. So verlangt es §6.11 («keine
  Straight-Arm-Tests vor dem Ende der Rampe»).
- Die Rampe ersetzt den Wochendeckel (§15.2 U-13, zur Bestätigung). Mit der
  Regel vorher blieb der Wiedereinsteiger monatelang bei einem Straight-Arm-Satz
  je Woche.

## Persona 5: Anfänger, Full Planche in 8 Wochen

**Eingaben:**
- Ziel Full Planche mit Datum in 8 Wochen, keine Planche-Stufe.
- 4–7 Liegestütze, Hollow unter 15 s.
- Boden und Wand, 3 × 45 min, noch nie regelmässig trainiert.

**Woche 1:**
- Liegestütze mit erhöhten Händen 2 × 5 und Plank 1 × 10 s, beide als
  Kalibrierung. Die Lean setzt die Stufe `push-up/full` voraus, deshalb gibt
  es keinen Planche-Block.
- Handgelenk-Prehab.
- Klimmzug-Übungen sind mangels Stange ausgeschlossen (SEL-02).

**Realismus-Check (GOAL-05):**
- Untergrenze 48 Wochen, gezeigte Spanne 105–162 Wochen, das Datum liegt in
  7 Wochen: «unrealistisch».
- Etappenziel ist die Tuck Planche, 28.5–45 Wochen.
- Der Text nennt die Zahlen «Erfahrungswerte aus dem Coaching, keine
  Prognose»; der Plan folgt der Arbeitsstufe, nicht dem Datum.

**Verlauf:**
- Liegestütze mit erhöhten Händen, dazu wöchentlich ein Kalibrierungssatz an
  normalen Liegestützen (ADAPT-07).
- In 12 Wochen wird die Stufe `push-up/full` (8 Wdh.) nicht erreicht, weil
  der Athlet mit 6 startet und 3 % je Woche zulegt.

**Plausibilität: plausibel.**
- Die Spanne entspricht §12.4.
- Abweichung: Das Etappenziel ist die Tuck statt der Lean, weil die Lean kein
  OG-Level hat (§15.2 U-19).
- Stütz fehlt, weil kein Barren angegeben ist. Hollow fehlt, weil die
  Empfehlung erst an der Tuck-Stufe hängt (`02` §8).

## Persona 6: Widersprüchliche Angaben

**Eingaben:**
- Inaktiv (`sedentary`), 0 Liegestütze, 0 Klimmzüge.
- Angaben: Straddle Planche und Full Front Lever, je 4–9 s.
- Park, 3 × 60 min.

**Onboarding:**
- Zwei Rückfragen (R-3): «Hältst du die Stufe «Straddle Planche» ohne Band
  und mit gestreckten Armen?» und dieselbe für den Front Lever.
- Mit «nein» löst der Planer konservativ auf: R-3 eine Stufe tiefer, danach
  R-2 bis zur höchsten Stufe mit plausiblen Vorstufen. Weil Liegestütze und
  Klimmzüge fehlen, bleibt keine Planche- und keine Front-Lever-Stufe; beide
  Angaben werden verworfen, σ × 1.25 (`PAR-S-44`, `PAR-S-46`).

**Woche 1:** nur Grundlagen:
- Liegestütze mit erhöhten Händen 2 × 5, Plank.
- Negativer Klimmzug, Dead Hang, gehockter Hollow Hold, fast aufrechtes Rudern.
- Kein Straight-Arm-Satz.
- Der Front-Lever-Hinweis nennt die Einstiegswerte aus dem Coaching.

**Plausibilität: plausibel.**
- Höchstens zwei Rückfragen, keine stille Übernahme der hohen Angaben, alle
  Startwerte mit Kalibrierung (§12.4).
- Ohne Antwort hätte der Planer genauso konservativ geplant (§15.2 U-20).

## Zusammenfassung

| Persona | Erwartung §12.4 erfüllt | Abweichung oder Befund |
|---|---|---|
| 1 | ja | keine Liegestütz-Leiter; Aktivierung bei 5 Klimmzügen und 8 Dips |
| 2 | ja | Lean 2 × 4 s zum Einstieg zu leicht (Kalibrierung korrigiert) |
| 3 | ja | kein Neutralgriff-Ersatz; kein Ellbogen-Prehab |
| 4 | ja, mit U-13 | Rampe ersetzt den Deckel (Review); Lean bis zum Rampenende |
| 5 | ja | Etappe Tuck statt Lean |
| 6 | ja | – |

Übergreifende Befunde (Spezifikation §15.4):
- Die erste Woche nutzt das Zeitbudget kaum.
- Abgeleitete Startwerte sind reine Untergrenzen.
- Die Plateau-Definition löst in der Simulation häufig Stagnations-Deloads aus.
- Kleine Straight-Arm-Volumina wachsen unter LOAD-02 langsam.
