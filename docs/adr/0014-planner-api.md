# ADR 0014 — HTTP-Schnittstelle des Trainingsplaners

- Status: proposed (zur Prüfung mit den Endpunkten, 28.09.2026)
- Date: 2026-09-28
- Deciders: Jaelricco

Dieses ADR ist auf Deutsch, wie alle Dokumente des Planer-Tracks.

## Kontext

Der Planer rechnet im reinen Kern (ADR 0012) und speichert über Postgres
(ADR 0013). `spec.md` §10.3 skizziert seine Endpunkte, §10.4 die Fehler.
ADR 0007 legt fest, dass `api/openapi.yaml` zuerst geändert wird: Die API
prüft Anfragen zur Laufzeit gegen das Schema, Contract-Tests halten Routen
und Antworten deckungsgleich, und der Swift-Client wird daraus erzeugt.

Beim Umsetzen blieben Fragen offen, die die Skizze nicht beantwortet:

- Was geschieht bei einem zweiten Onboarding? Ein neuer Start würde Auflagen
  (Sperren, Stopps) verwerfen, die ein früheres Ereignis gesetzt hat.
- Wie antwortet ein wiederholtes Ereignis? Der Dienst überspringt es, und die
  zweite Antwort war leer. Ging die erste verloren, erfuhr der Client nie,
  dass er die Red-Flag-Fragen einer Region stellen soll.
- Welche Werte prüft das Schema, welche der Kern? Skills, Stufen,
  Antwortklassen, Regionen und Red Flags stehen in der Wissensbasis und
  können sich mit ihr ändern.
- Woher kennt ein Client die Skills, Übungen und Regionen des Planers? Sie
  sind noch keine Content-Zeilen (U-1) und fehlen in `/v1/skills`.
- Wie hält die API EXPL-07 ein, wenn ein Client Quellentitel neben einer
  Beschwerde zeigen könnte?

## Entscheidungen

### 1. Endpunkte

Unter dem Tag `planning` entstehen:

| Methode und Pfad | Zweck |
|---|---|
| `POST /v1/me/onboarding` | alle Antworten auf einmal, Ergebnis mit erstem Plan |
| `GET`, `PUT /v1/me/training-profile` | Profil lesen, änderbaren Teil ersetzen |
| `GET`, `PUT /v1/me/goals` | Ziele mit Realismus-Check |
| `GET /v1/me/plan` | Plan der Woche (`?week=` für frühere Wochen) |
| `POST /v1/me/plan/regenerate` | Plan der laufenden Woche neu erzeugen |
| `GET /v1/me/plan/sessions/{plannedSessionId}` | eine geplante Einheit |
| `GET /v1/me/plan/decisions` | Änderungsprotokoll, neueste zuerst |
| `POST`, `GET /v1/me/pain-reports` | Schmerzbericht melden, Berichte lesen |
| `POST /v1/me/symptoms` | Belastungssymptome (RF-10) |
| `GET /v1/me/regions` | Regionen mit Zustand, Auflagen und Red-Flag-Fragen |
| `POST /v1/me/regions/{region}/red-flags` | Antworten auf die Red-Flag-Fragen |
| `POST /v1/me/regions/{region}/clearance` | Freigabe einer Region bestätigen |
| `POST /v1/me/screening/clearance` | Freigabe für das Training insgesamt |
| `GET /v1/me/capacity` | Kapazitäten mit Konfidenz |
| `GET /v1/planner/rules`, `/sources`, `/parameters`, `/catalogue` | öffentlicher Katalog |

Noch nicht gebaut, jeweils mit Grund:
- `POST /v1/me/plan/sessions/{id}/start` und `plan_changes[]` am Abschluss
  einer Einheit: Die Log-Tabellen verweisen auf Content-Übungen, die Übungen
  des Planers sind keine (U-1, ADR 0013 §2). Der Start ist nachgezogen in
  ADR 0016, `plan_changes[]` nicht.
- Sync der Schmerzberichte: Die Tabellen bekommen ihre Sync-Spalten erst mit
  diesem Schritt.
- Einwilligung ändern oder widerrufen: Ein Widerruf muss Gesundheitsdaten
  löschen und Auflagen behalten (ENT-S-7); das braucht einen eigenen
  Endpunkt. Nachgezogen in ADR 0015.
- `?explain=trace`: Der Kern zeichnet noch keine Spur auf.

### 2. Das Onboarding läuft einmal

Ein zweites Onboarding antwortet `409 already-onboarded`. Änderungen danach
gehen über Profil und Ziele. Sonst könnte ein neuer Start mit anderen
Antworten eine Sperre oder einen Stopp aufheben, ohne dass eine Freigabe
bestätigt wurde. Solange Rückfragen offen sind (`needs_answers`), speichert
der Dienst nichts; der Client schickt die Antworten mit `clarifications`
erneut. Ein optionaler `Idempotency-Key` macht die Wiederholung sicher: Die
erste Antwort wird gespeichert und wiedergegeben, mit demselben Code wie der
Sync-Push (ADR 0009).

`PUT /v1/me/training-profile` ersetzt nur den änderbaren Teil:
Verfügbarkeit, Equipment, Körpergewicht, bevorzugte Tage, Mobilitäts-Checks
und die Zusatzlast-Grenzen. Einwilligung, Geburtsjahr und
Trainingshintergrund bleiben, wie sie beantwortet wurden. Profil- und
Zieländerungen erzeugen den Plan neu (WEEK-08).

### 3. Ereignisse sind idempotent und antworten gleich

Schmerzberichte, Symptome, Red-Flag-Antworten und Freigaben tragen eine
Client-ID (UUIDv7). Sie ist die Quelle im Änderungsprotokoll. Ein
wiederholtes Ereignis wird nicht noch einmal angewandt, und die Antwort
enthält die damals aufgezeichneten Änderungen mit `replayed: true`. So
erreicht auch ein wiederholter Aufruf die Aufforderung `ask_red_flags`.
Dieselbe ID mit anderem Inhalt gilt als dasselbe Ereignis.

### 4. Schema und Kern teilen sich die Prüfung

Das Schema prüft die Struktur und die Vokabulare, die der Code festlegt
(Equipment, Trainingsniveau, Pausenklasse, Zeitpunkte des Schmerzberichts,
Mobilitäts-Checks). Der Kern prüft alles, was die Wissensbasis bestimmt:
Skills und Stufen der Ziele, Antwortklassen, Stufen je Skill, Regionen und
Red Flags. Beide melden `422 validation` mit Feldfehlern, deren Schlüssel
JSON-Pointer in den Anfragekörper sind.

Jede Red-Flag-Frage, die für eine Region gestellt wird, muss beantwortet
sein, eine Folgefrage genau nach einem Ja (`onboarding.md` §3.7: «alle
gestellten beantwortet»). Das gilt im Onboarding und am Red-Flag-Endpunkt.
Eine leere Antwort bedeutet also nicht mehr «alles verneint». Zwei Tests
halten die Vokabulare gleich: die des Kerns mit den Enums der API und das
Equipment mit dem Schema der Wissensbasis.

### 5. Pläne: erzeugen bei Bedarf, ETag ist die Plan-ID

`GET /v1/me/plan` erzeugt den Plan der laufenden Woche beim ersten Abruf
(§10.3). Frühere Wochen liefert `?week=` so, wie sie endeten; Pläne späterer
Wochen gibt es noch nicht (404).

Der Dienst vergibt die IDs von Plänen, geplanten Einheiten und Protokoll-
Einträgen selbst (`IDSource`), der Speicher übernimmt sie. Damit liefern der
Speicher im Arbeitsspeicher und Postgres dieselben Antworten bis auf die ID.
Ein gespeicherter Plan ändert sich nie; jede Neuerzeugung ist ein neuer Plan
mit neuer ID. Deshalb ist die Plan-ID das `ETag`, nicht der `input_hash`:
Nach einer Neuerzeugung mit gleichen Eingaben hat der Plan denselben Hash,
aber neue Einheiten-IDs. Seit ADR 0016 nimmt das `ETag` auch die Starts der
Einheiten auf. Geplante Einheiten findet die API nur in aktiven
Plänen.

Ist das Training gestoppt (SAFE-02, SAFE-07), antwortet der Plan mit
`200` und `stopped: true`, ohne Einheiten und mit den Gründen und dem
Disclaimer. Der Fehler `training-stopped` aus §10.4 bleibt für den Start
einer Einheit: Der Client soll den Grund lesen und offline zwischenspeichern
können.

### 6. EXPL-07 setzt der Server durch

Eine Begründung, die an eine Region mit Beschwerde oder Red Flag gebunden ist,
kommt ohne Quellen-IDs. Red-Flag-Fragen tragen keine Quellen. So kann ein
Client keinen Studientitel neben einer Beschwerde zeigen, auch nicht aus
Versehen. Die vollständigen IDs bleiben in `plan_decisions` (§9.1).

### 7. Gesundheitsdaten nur mit Einwilligung

Schmerzberichte brauchen die Einwilligung und antworten ohne sie
`409 consent-required`. Ohne Einwilligung zeigt `GET /v1/me/regions` keine
Zustände (`tracked: false`), aber die Auflagen, die trotzdem gelten.
Red-Flag-Antworten wertet der Kern auch ohne Einwilligung aus und speichert
nur die Auflage (`onboarding.md` §3.1).

### 8. Ein öffentlicher Katalog

`/v1/planner/rules`, `/sources` und `/parameters` erklären die Gründe.
`/v1/planner/catalogue` liefert, was ein Client zum Fragen und Anzeigen
braucht: die Skills des Planers mit Stufen und Stufen-Klassen, die Übungen
mit Namen, die Regionen der Körperkarte mit ihren Red-Flag-Fragen und die
Antwortklassen. Alle vier sind ohne Anmeldung lesbar und tragen die
`ruleset_version` als `ETag`. Pläne nennen zusätzlich den Namen jeder Übung.

### 9. Rücksicht auf den Swift-Generator

`swift-openapi-generator` verwirft `oneOf [ref, null]` (ADR 0010). Der Plan
im Onboarding-Ergebnis ist deshalb optional statt nullbar, die Folgefrage
einer Red Flag ist als Schema selbst nullbar. Das Schema nutzt weder
`propertyNames` noch Integer-Enums; `session_minutes` prüft der Kern.
Vokabulare, die mit der Wissensbasis wachsen (Änderungsarten, Auslöser,
Reiztypen, Regionszustände), sind Zeichenketten mit Beschreibung, damit ein
älterer Client einen neuen Wert nicht als Fehler dekodiert.

### 10. Fehler und Verfügbarkeit

Neue Problemtypen: `onboarding-required` (409), `already-onboarded` (409),
`consent-required` (409) und `planning-unavailable` (503). Die Routen sind
immer registriert. Ohne Planer oder mit ungültiger Wissensbasis antworten
nur sie mit 503; Log, Sync und Skill-Karte laufen weiter (§2.6).
`cmd/api` baut den Dienst mit dem Postgres-Adapter und der Wissensbasis aus
`HEFESTO_CONTENT_DIR`.

## Folgen

- Der Swift-Client lässt sich aus dem Schema erzeugen. Die iOS-CI läuft bei
  jeder Änderung von `api/openapi.yaml` mit; die Änderung ist additiv
  (oasdiff: nichts Brechendes).
- In Produktion antwortet der Planer mit 503, bis die Übungen der
  Wissensbasis geprüft sind (ENT-10). Die Wissensbasis lehnt dort Entwürfe
  ab (U-3).
- Jede Neuerzeugung und jedes Ereignis legt einen neuen Plan an. Ersetzte
  Pläne bleiben; wann sie gelöscht werden, ist weiter offen (ADR 0013).
- `GET /v1/me/pain-reports` blättert im Snapshot, der ohnehin ganz gelesen
  wird. Mit dem Fenster für lange Verläufe (ADR 0013) braucht es eine eigene
  Abfrage.
- Befund: Keine Übung der Wissensbasis trägt `restriction_tags`. Die
  Einschränkungen einer Fachperson aus dem Onboarding werden gespeichert,
  wirken aber noch nicht auf die Auswahl (SEL-03). Die Zuordnung ist Inhalt
  und braucht eine fachliche Prüfung (§15.4).
