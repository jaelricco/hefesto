# ADR 0015 — Einwilligung zu Gesundheitsdaten: Widerruf und nachträgliche Erteilung

- Status: proposed (zur Prüfung mit dem Endpunkt, 28.09.2026)
- Date: 2026-09-28
- Deciders: Jaelricco

Dieses ADR ist auf Deutsch, wie alle Dokumente des Planer-Tracks.

## Kontext

Die Einwilligung zu Gesundheitsdaten wird im Onboarding gegeben oder nicht
(`onboarding.md` §3.1). Ohne sie stellt die App die Sicherheitsfragen
trotzdem, wertet sie aber nur für «Plan ja/nein, Region planen ja/nein» aus.
Der Planer plant dann vorsichtiger (SAFE-04: keine Tests, Deckel × 0.5), und
eine aktuelle Beschwerde schliesst die Region wie Stufe 0 aus (Auflage
`region_excluded`). `spec.md` §4.9 und §13.4 verlangen, dass
Gesundheitsangaben beim Widerruf gelöscht werden. ENT-S-7 (a) hält fest, dass
Auflagen ohne Antworten und Werte bleiben. ADR 0014 hat den Endpunkt dafür
zurückgestellt.

Offen war:
- Was löscht ein Widerruf, und was bleibt als Schutz?
- Was geschieht bei einer späteren Einwilligung mit Angaben, die nie
  gespeichert wurden?
- Wie hebt man einen Ausschluss auf? Bisher gab es dafür keinen Weg.

Beim Umsetzen zeigten sich zwei Lücken ohne Einwilligung:
- Eine Freigabe hob eine Sperre auf, zu der der Planer keinen Zustand
  hält. Die Region war danach sofort frei, ohne Rampe.
- Eine Red Flag mit der Aktion «Stufe 0» (Dringlichkeit A, etwa Schmerz in
  Ruhe oder nachts) hinterliess ohne Einwilligung nichts. Die Region wurde
  weiter normal geplant.

## Entscheidungen

### 1. Ein Ereignis wie die anderen

`POST /v1/me/health-consent` mit `{id, granted}` erteilt oder widerruft die
Einwilligung. Es läuft wie jedes Ereignis durch das Änderungsprotokoll
(Auslöser `consent`): idempotent über die Client-ID, eine Wiederholung
antwortet mit den aufgezeichneten Änderungen. Wer erteilt, was schon erteilt
ist, oder widerruft, was schon widerrufen ist, ändert nichts.

### 2. Der Widerruf löscht Gesundheitsangaben und behält den Schutz

Gelöscht werden die Tabellen, die §4.9 als Gesundheitsangaben führt:
- Zustände der Regionen (`user_region_status`),
- Schmerzberichte (`user_pain_reports`),
- Screening (`user_screening`).

Red-Flag-Antworten werden ohnehin nicht gespeichert. Als Auflage ohne
Antworten und Werte bleibt:
- Eine gesperrte Region bleibt gesperrt (`region_locked`).
- Eine Region mit Beschwerde oder in der Rampe wird ausgeschlossen wie
  Stufe 0 (`region_excluded`, SAFE-04).
- Ein Trainingsstopp bleibt (`plan_stopped`).

Der Plan wird danach unter SAFE-04 neu erzeugt. Verlauf, Kapazitäten,
Leitern und Pause sind Trainingsdaten und bleiben.

### 3. Die nachträgliche Einwilligung fragt neu und verfolgt die Ausschlüsse

Was ohne Einwilligung nicht gespeichert wurde, fragt die Erteilung noch
einmal:
- die sechs Screening-Fragen (Pflicht),
- die Regionen mit einer Verletzung in den letzten 12 Monaten (optional).

Ohne diese Antworten würde ein früheres «Ja» im Screening stillschweigend
Tests und Prüfversuche erlauben.

Jeder Ausschluss wird eine verfolgte Region in Stufe 0. Sie schliesst
dieselben Übungen aus (SAFE-06 wie SAFE-04), hat aber einen Ausgang: grüner
Alltagsschmerz und verneinte Red Flags führen in die Rampe (§8.3). Eine
Sperre ohne Zustand wird eine verfolgte Sperre, damit ihre Freigabe in die
Rampe führt.

### 4. Ohne Einwilligung bleibt eine Region draussen, bis sie verfolgt wird

Ohne Einwilligung kann der Planer keine Rampe führen. Deshalb gilt:
- Die Freigabe einer Sperre ohne Zustand lässt einen Ausschluss zurück.
- Eine Red Flag mit der Aktion «Stufe 0» schliesst die Region aus.

Beides schliesst die zwei Lücken. Mit Einwilligung führt eine solche
Freigabe in die Rampe (Start 0.25, PAR-D-33).

### 5. Der Text von SAFE-04 stimmt

SAFE-04 sagte, der Planer speichere «keine Einschränkungen dauerhaft». Das
widersprach ENT-S-7. Neu: «Er speichert keine Beschwerden und keine
Schmerzwerte, nur Sperren und Ausschlüsse aus den Sicherheitsfragen.»

## Folgen

- Ein User kann die Einwilligung jederzeit widerrufen und erteilen. Die App
  sollte nach der Erteilung zum Alltagsschmerz der Regionen in Stufe 0
  fragen, weil erst er den Ausgang öffnet.
- Ohne Einwilligung bleibt eine Region nach einer Beschwerde ausgeschlossen,
  auch nach einer Freigabe. Das ist strenger als vorher und der Preis dafür,
  dass der Planer ohne Einwilligung nichts über die Region weiss.
- **Offen für die rechtliche Prüfung (ENT-4):** Abgeleitete Angaben bleiben
  auch nach einem Widerruf erhalten:
  - `plan_decisions`: Änderungen mit Region und Zustand, Auslöser wie
    `pain_report` oder `red_flags`;
  - `training_plans`: Begründungen mit Region, Fragen für die Überwachung.

  Das Protokoll ist nur anhängbar. Die Idempotenz hängt an ihm: Eine
  wiederholte Freigabe darf nicht später noch einmal wirken. Pläne sind die
  Erklärung vergangener Wochen. Alternativen sind, die Einträge der
  Gesundheits-Auslöser zu schwärzen und nur Auslöser und Quelle zu behalten,
  und Pläne vergangener Wochen zu löschen.
- Einen Nachweis der Einwilligung (Zeitpunkt, Version des Einwilligungstexts)
  gibt es nicht. Das Protokoll hält fest, wann sie sich geändert hat, aber
  nicht, welchem Text zugestimmt wurde. Ob es das braucht, gehört zur
  rechtlichen Prüfung.
