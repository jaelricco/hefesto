# ADR 0018 — Der Abschluss einer Einheit erreicht den Planer

- Status: proposed (zur Prüfung mit dem Abschluss, 28.09.2026)
- Date: 2026-09-28
- Deciders: Jaelricco

Dieses ADR ist auf Deutsch, wie alle Dokumente des Planer-Tracks.

## Kontext

`spec.md` §10.2 und §6.1 sehen vor: Nach dem Abschluss einer Einheit
(`POST /v1/sessions/{id}/complete` oder Sync) ruft der Dienst `Adapt`. Die
Antwort bekommt das optionale Feld `plan_changes[]`. Scheitert die Adaption,
bleibt der Abschluss gültig, und der nächste Planabruf holt sie nach. Bisher
erreichte keine geloggte Einheit den Planer. Kapazitäten, Leitern, Plateaus
und Rampen lernten also nichts aus dem Log.

Offen war:
- Welche Einheiten zählen?
- Wie wird aus Elementen des Logs ein Satz im Verlauf des Planers?
- Was geschieht mit Unterstützung, die der Planer nicht kennt, und mit
  fehlenden Reserven?
- Wie wird eine verpasste Adaption nachgeholt?

## Entscheidungen

### 1. Jede abgeschlossene Einheit zählt, nicht nur geplante

Jede abgeschlossene, nicht gelöschte Einheit seit dem Tag des Onboardings
erreicht den Planer, ausser einem geloggten Ruhetag. Das gilt auch für
Einheiten, die der User selbst zusammengestellt hat. Sie belasten dieselben
Strukturen, und die Deckel (§7) sollen sie sehen. Übungen, die der Planer
nicht kennt, stehen im Verlauf, der Kern übergeht sie.

### 2. Aus dem Log wird der Verlauf des Planers

Es zählen nur ausgeführte Sätze (`NOT is_planned`), in der Reihenfolge der
Ausführung. Aus jedem Element wird ein Satz:
- Wert: Wiederholungen oder Sekunden. Distanzen hat der Planer nicht und
  lässt sie weg.
- Unterstützung: keine oder Band. Andere Unterstützung (Partner, Maschine,
  Schräge, Gegengewicht, Fussstütze) misst weder die unassistierte noch die
  Band-Kapazität (PAR-A-21). Solche Sätze werden weggelassen; die Tabelle
  der Kapazitäten kennt auch nur `none` und `band`.
- Reserve: das geloggte RIR eines Wiederholungssatzes. RPE wird nicht
  umgerechnet, die Wissensbasis hat dafür keine Regel. Für Halte hat das Log
  keine Spalte (`set_entries.sir_s` fehlt); ein Halt ohne Reserve zählt im
  Kern als Untergrenze (§4.3).
- Dazu kommen: Zusatzlast, Art des Satzes, Form (1–5), `failed`,
  Teilbewegung und reine Exzentrik.
- Die Einheit trägt den Kalendertag, die empfundene Ermüdung und `deload`,
  wenn sie aus einer Deload-Einheit des Plans gestartet wurde.

### 3. Nach dem Commit, einmal je Einheit

Der Abschluss im Log läuft in seiner eigenen Transaktion. Danach wendet der
Dienst die Einheit in einer Transaktion des Planers an:
- Er markiert die geplante Einheit als `completed`, in jedem Plan, der sie
  trägt.
- Wurde die Einheit aus einer Deload-Einheit gestartet, markiert er ihren Tag
  als Deload-Tag (`user_training_days.deload`, spec §10.2). Für den Streak
  zählt der Tag ohnehin (ADR 0003).
- Er wendet die Einheit als Ereignis an (Auslöser `session_completed`, Quelle
  die Session). Das Änderungsprotokoll macht das idempotent: Ein
  wiederholter Abschluss antwortet mit denselben `plan_changes`.
- Der neue Plan gleicht Entwürfe ab (ADR 0017). Die abgeschlossene Einheit
  ist kein Entwurf mehr und bleibt, wie sie ist.

### 4. Scheitert der Planer, holt der nächste Planabruf nach

Scheitert der Planer, ist er nicht verfügbar, oder fehlt das Onboarding, fehlt
`plan_changes` in der Antwort. Der Abschluss bleibt gültig.
`GET /v1/me/plan` sucht vorher abgeschlossene Einheiten ohne Eintrag im
Änderungsprotokoll: höchstens 20 je Aufruf, die ältesten zuerst, jede in
eigener Transaktion. Ein Fehler wird protokolliert und blockiert den Plan
nicht. Gibt es nichts nachzuholen, kostet das eine Abfrage.

### 5. Additive API

`CompletionResult.plan_changes` ist optional. Es steht in der REST-Antwort
und im Ergebnis der Sync-Operation `session.complete`. oasdiff meldet nichts
Brechendes.

## Folgen

- Kapazitäten, Leitern, Plateaus, Deloads und Rampen lernen aus dem Log.
  Eine geplante Einheit durchläuft jetzt: Start, Abgleich, Ausführung,
  Abschluss.
- **Befunde für den Review:**
  - Wer vor allem mit Partner oder Maschine trainiert, hinterlässt dem
    Planer wenig. Diese Sätze fehlen auch in der Belastungshistorie, und die
    Deckel sehen weniger Belastung, als stattfand.
  - Einheiten am Tag des Onboardings zählen, auch wenn sie vor dem
    Onboarding abgeschlossen wurden. `onboarded_at` ist ein Kalendertag.
  - Eine abgeschlossene Einheit, die der User später löscht, bleibt im
    Verlauf des Planers (wie bisher, ADR 0013).
- **Nicht gebaut:**
  - die Reserve eines Halts (`sir_s`);
  - der Check-in beim Start (ADAPT-17);
  - der Offline-Start über den Sync.
