# ADR 0021 — Der Trainingsplan in der iOS-App: der Unterbau

- Status: proposed (zur Prüfung mit dem Unterbau, 28.09.2026)
- Date: 2026-09-28
- Deciders: Jaelricco

Dieses ADR ist auf Deutsch, wie alle Dokumente des Planer-Tracks.

## Kontext

Der Planer hat seine API: Plan, Start mit Check-in, Abgleich und Abschluss
(ADR 0014, 0016 bis 0019). Die iOS-App kannte bisher keinen Plan. `spec.md`
§10.5 verlangt, dass der Client den Plan der laufenden Woche lokal hält, wie
die Skill-Karte (ADR 0011).

Zur selben Zeit gestaltet ein anderer Agent jeden Screen der App neu (eigener
Branch, dort ADR 0020 zum visuellen Design). Neue Screens im alten Stil gäben
Konflikte in denselben Dateien und doppelte Arbeit. Das Paket `HefestoKit`
fasst er nicht an. Die Screens des Plans warten deshalb auf das Design; dieses
ADR entscheidet den Unterbau im Paket.

Offen war:
- Wie hält die App den Plan offline?
- Was geschieht beim Start, auch bei einem wiederholten?
- Wie wird ein geplanter Satz zu einem ausgeführten?

## Entscheidungen

### 1. Der Plan wird als Dokument gehalten

Die Tabelle `cachedPlan` hält je Woche den Plan, wie der Server ihn sendet:
Wochenbeginn, `ETag`, JSON und Abrufzeit. Gelesen wird er mit dem generierten
Typ `TrainingPlan`, also ohne handgeschriebene DTOs (CLAUDE.md).

Anders als die Skill-Karte wird der Plan nicht auf Tabellen verteilt:
- Er ist das Lesemodell des Servers, wird immer ganz ersetzt und lokal nie
  geändert. Die Skill-Karte hat mit `seenAt` einen eigenen lokalen Zustand;
  der Plan hat keinen.
- Die App fragt in ihm nichts ab, was eine Tabelle bräuchte. Sie zeigt eine
  Woche oder eine Einheit.
- So bleibt jedes Feld des Plans erhalten, auch eines, das die App noch nicht
  zeigt: Gründe, Stopp-Regeln, Kalibrierung, Schmerzbeobachtung.

Ein Tag findet den Plan seiner Woche: den jüngsten Wochenbeginn höchstens
sechs Tage davor. Pläne, die mehr als vier Wochen älter sind als der neueste,
werden verworfen.

### 2. Abruf mit `ETag`

`SyncEngine.refreshPlan(on:)` fragt den Plan der Woche eines Tages ab, ohne
Angabe den von heute (UTC, wie die API). Das `ETag` des gehaltenen Plans geht
als `If-None-Match` mit; ein unveränderter Plan kostet ein 304.
- Ein Plan ändert sich mit jedem Ereignis: Abschluss, Schmerzbericht, Start.
  Die App soll ihn nach jedem Sync und nach jedem Start abrufen, wie die
  Skill-Karte. `sync()` selbst ruft ihn nicht ab: Ohne Onboarding scheiterte
  sonst jeder Sync. Das Verdrahten kommt mit den Screens.
- Ein gestoppter Plan (`stopped`) wird gehalten wie jeder andere. Seine Gründe
  erklären den Stopp und was ihn aufhebt.
- `onboarding-required` löscht alle gehaltenen Pläne: Der Server hat für
  diesen User keinen Plan.
- Die Probleme des Planers kommen als `PlanError`: Onboarding fehlt, Training
  gestoppt, Einheit nicht mehr im Plan, Planer nicht verfügbar.
- Der generierte Client kodiert Header-Parameter wie Teile einer URI
  (RFC 6570). Die Anführungszeichen eines `ETag` gingen deshalb als `%22`
  hinaus, und kein `If-None-Match` traf je, auch beim Übungskatalog nicht.
  Die Tests prüften das nicht. Jeder Client der App führt jetzt die
  `EntityTagMiddleware` aus, die den Header sendet, wie HTTP ihn definiert.

### 3. Der Start braucht das Netz

`SyncEngine.startPlannedSession` sendet den Start mit einer ID, die die App
erzeugt, der Zeitzone und optional dem Check-in (ADR 0019).
- Die Antwort ist der Draft mit seinen geplanten Sätzen. Die App hält ihn wie
  jede Session des Servers, ohne Einträge in der Outbox.
- Ein wiederholter Start antwortet mit der ersten Session. Hält die App diese
  Session schon, auch gelöscht, bleibt ihre Kopie unberührt. Sie kann
  Änderungen tragen, die noch nicht gesendet sind; den Rest bringt der Sync.
- Die Antworten des Check-ins gehen nur in die Anfrage. Die App speichert sie
  nicht, wie der Server (ADR 0019).
- Offline startet keine Einheit. Der Offline-Start über den Sync (§10.5) ist
  weiter nicht gebaut.

### 4. Die Verweise auf den Plan werden gespiegelt, nicht gesendet

`Session.plannedSessionId` und `SetEntry.plannedItemId` kommen aus dem
Sync-Feed und aus dem Start. Die App sendet sie nie zurück; der Server behält
sie bei jedem Schreiben (ADR 0016).
- Über `plannedItemId` findet die App zu einem geplanten Satz sein Plan-Item:
  Ziel, Reserve, Kalibrierung, Stopp-Regeln, Schmerzbeobachtung.
- Sessions, die vor diesem Update gezogen wurden, haben keine Verweise. Das
  betrifft keine geplante Einheit, denn die App konnte bisher keine starten.

### 5. Ein geplanter Satz wird an seinem Platz ausgeführt

`LoggerModel.perform(plannedSet:elements:rir:sirS:)` macht aus dem geplanten
Satz den ausgeführten, mit derselben ID. So bleibt der Verweis des Servers auf
das Plan-Item erhalten.
- `isPlanned` wird `false`, `completedAt` die Zeit der Ausführung.
- Die Istwerte ersetzen die Ziele. Die Reserve ist, was der Athlet angibt,
  nicht die Ziel-Reserve.
- Die geplante Pause des Satzes startet.
- Nicht ausgeführte geplante Sätze bleiben offen. Sie zählen für den Planer
  nicht (ADR 0018).

Ein eigener Satz neben dem Plan bleibt `logSet` oder `logCombo`, wie bisher.
Es gibt einen Pfad für Sätze (CLAUDE.md): Beide schreiben einen Satz mit
seinen Elementen.

## Folgen

- Die App kann den Plan offline zeigen, eine geplante Einheit mit Check-in
  starten und ihre Sätze ausführen. Es fehlen die Screens.
- **Nicht gebaut:**
  - Die Screens: Woche und Heute aus dem Plan, Check-in und Start, der Logger
    mit Zielen, Angebote auf aktive Wahl, die Nachfrage nach der Reserve beim
    Kalibrierungssatz und nach Schmerz bei beobachteten Items (`monitor`).
    Sie folgen, sobald das Design des anderen Branches in `main` ist.
  - Das Onboarding in der App (`POST /v1/me/onboarding`, Katalog des Planers).
    Ohne Onboarding hat ein User keinen Plan.
  - Schmerzberichte und Symptome aus der App.
  - Die Wahl des Bands bei einem geplanten Band-Satz. Sie kommt mit dem
    Screen der Bänder.
  - Der Offline-Start.
- **Befund:** Der Befund aus ADR 0017 wird mit der App konkret. Führt der
  Athlet offline einen geplanten Satz aus, den der Server inzwischen ersetzt
  hat, lehnt der Sync die Ausführung ab (ADR 0009). Die App sollte vor einem
  Ereignis synchronisieren.
