# ADR 0016 — Start einer geplanten Einheit im Trainings-Log

- Status: proposed (zur Prüfung mit dem Endpunkt, 28.09.2026)
- Date: 2026-09-28
- Deciders: Jaelricco

Dieses ADR ist auf Deutsch, wie alle Dokumente des Planer-Tracks.

## Kontext

`spec.md` §10.2 beschreibt den Start einer geplanten Einheit so: Der Dienst
erzeugt aus dem Plan eine `draft`-Session in den bestehenden Log-Tabellen,
mit einem geplanten Satz-Eintrag je Satz und den Zielwerten in den Elementen.
Ausgeführt wird danach über die Log-API oder den Sync, wie bisher. ADR 0014
hat den Endpunkt zurückgestellt, weil ein Element auf eine Übung des
Content-Katalogs verweist und die Übungen des Planers keine Content-Zeilen
waren (U-1, ADR 0013 §2).

Beim Umsetzen waren diese Fragen offen:
- Wie kommen die Übungen des Planers in den Katalog, ohne Inhalt zu erfinden?
- Wie wird aus einem geplanten Item ein Satz im Log, bei Paaren, Angeboten,
  Haltezeiten und Band-Unterstützung?
- Was geschieht bei einem zweiten Start und bei einem gelöschten Draft?
- Was geschieht mit einem Start, wenn der Plan der Woche neu erzeugt wird?
  Das passiert bei jedem Ereignis.
- Wo steht, ob eine geplante Einheit gestartet ist?

## Entscheidungen

### 1. Die Übungen des Planers stehen im Katalog (U-50)

49 der 54 Übungen des Planers fehlten in `content/exercises/`. Sie stehen dort
jetzt als `draft_placeholder`, abgeleitet aus `content/training/exercises.yaml`:
- Name, Messgrösse und Equipment kommen aus der Wissensbasis.
- Die Familie folgt aus dem Bewegungsmuster; Prehab wird nach seiner Richtung
  eingeordnet.

Es gibt keine Beschreibungen und keine Hinweise zur Ausführung. Die Einträge
fallen unter die inhaltliche Prüfung (ENT-10), wie der übrige Katalog.

Die Validierung scheitert, wenn eine Übung des Planers im Katalog fehlt oder
anders gemessen wird. Sie zählt die Übung als benutzt, damit
`contentlint -strict` sie nicht als Waise meldet.

Die Wissensbasis bleibt die Quelle für alles, was der Planer rechnet (U-1).

### 2. `Materialize` im reinen Kern

`Materialize` übersetzt eine geplante Einheit in den Entwurf, den das Log
speichert:
- **Ein Satz-Eintrag je geplantem Satz**, mit einem Element: Übung,
  Messgrösse, Ziel in `reps` oder `hold_seconds`, Zusatzlast, `kind`,
  `rest_after_planned_s` und `is_eccentric_only` für Exzentrik-Reize. Es
  gibt keinen Sonderweg für einfache Sätze (CLAUDE.md).
- **Antagonisten-Paare** werden ein Block `superset`. Die Sätze wechseln sich
  je Runde ab (`round_index`).
- **Angebote** (Prüfversuch, erster Versuch) werden nicht geschrieben. Sie
  gelten nur, wenn der User sie aktiv wählt (ADAPT-05, U-10). Die App fügt
  sie dann als Satz hinzu.
- **Blöcke ohne geplanten Satz** werden nicht geschrieben, etwa das
  allgemeine Aufwärmen oder ein Block nur mit Angeboten. Der Plan zeigt sie
  weiter.
- **`rir`** trägt die Reserve eines Wiederholungssatzes. Die Reserve eines
  Halts (Sekunden) hat im Log keine Spalte; sie bleibt im Plan-Item.
- **Band-Unterstützung** bleibt im Plan-Item (`assist: band`). Das Log
  verlangt für eine Band-Unterstützung das Band selbst (`band_id`,
  `set_element_assistance_band_ck`), und das kennt der Planer nicht. Die App
  fragt beim Loggen nach dem Band. Ein geplanter Satz ist kein Beleg für die
  Skill-Karte (`NOT is_planned`, ADR 0008). Ein fehlendes Band im Draft kann
  also nichts freischalten; der ausgeführte Satz trägt die Unterstützung, die
  die App schreibt.
- **Titel:** Die Session hat keinen Titel. Ein Text im Code umginge die
  Texte der Wissensbasis und ihre Sprachen (KB-11). Die App benennt die
  Einheit aus dem Plan.

### 3. Der Start ist eine Transaktion des Planers

`POST /v1/me/plan/sessions/{plannedSessionId}/start` mit `{id, timezone,
started_at?}` läuft im Planer-Aufruf, also unter der Sperre je User
(ADR 0013). Die Transaktion:
1. sperrt die geplante Einheit (nur in aktiven Plänen);
2. legt die Session mit der Client-ID an, wie `POST /v1/sessions`, mit
   `planned_session_id`;
3. schreibt Blöcke, Satz-Einträge mit `planned_item_id` und Elemente;
4. markiert die geplante Einheit als `started`.

Regeln des Starts:
- **Einmal je geplanter Einheit.** Ein zweiter Start antwortet `200` mit der
  zuerst gestarteten Session, auch mit einer anderen `id`. So erzeugen zwei
  Geräte oder ein wiederholter Aufruf keinen zweiten Draft.
- **Neustart nach Löschen.** Ist die Session gelöscht, gilt die Einheit
  wieder als geplant und startet neu.
- **Belegte ID.** Gehört die `id` einer anderen Session, antwortet der Start
  mit `409 already-exists`.
- **Stopp zuerst.** Ist das Training gestoppt (SAFE-02, SAFE-07), antwortet
  der Start mit `409 training-stopped` und nennt die Regel. Das gilt auch für
  einen wiederholten Start: Nach einem Stopp soll der Client den Stopp
  erfahren, nicht den alten Draft. Grund und empfohlene Abklärung stehen in
  den Gründen des Plans (U-41).
- **Fehlende Übung.** Fehlt eine Übung im Katalog, antwortet der Start mit
  `503 planning-unavailable` und schreibt nichts.

### 4. Der Start steht in der Zeile, nicht im Plan-Payload

`planned_sessions.status` und `workout_session_id` halten den Start. Der
Payload bleibt, was der Kern geplant hat. Beim Lesen füllt der Speicher
`status` und `workout_session_id` jeder Einheit.

Das `ETag` des Plans ist die Plan-ID. Sobald eine Einheit gestartet ist,
kommt ein Hash der Starts dazu. Mit der Plan-ID allein bekäme ein Client
nach einem Start `304` und sähe den Start nicht (ADR 0014 §5 ergänzt).

### 5. Ein neuer Plan übernimmt den Start des Tages

Jedes Ereignis und jede Neuerzeugung legt einen neuen Plan an. Eine Einheit
des neuen Plans am Tag einer gestarteten Einheit übernimmt deren Status und
Session. Die Session behält ihren Verweis auf die geplante Einheit, mit der
sie gestartet wurde; die gehört dann zu einem ersetzten Plan.

### 6. Item-IDs

Der Dienst gibt jedem Item eines Plans eine ID. Ein Satz-Eintrag verweist in
`planned_item_id` auf sie, ohne Fremdschlüssel, denn die Items stehen im
Payload. So lassen sich Ist und Soll zuordnen (§10.2).

### 7. Additive API

Neu sind:
- in Plänen: `id` je Item, `status` und `workout_session_id` je Einheit;
- im Log und im Sync: `planned_session_id` an der Session, `planned_item_id`
  am Satz.

Neu ist auch der Problemtyp `training-stopped` (409). oasdiff meldet nichts
Brechendes. Die Log-Tabellen bekommen zwei nullbare Spalten, beide additiv
in `00009_planning.sql`.

## Folgen

- Ein Plan lässt sich im Log ausführen. Ausgeführt wird unverändert über die
  Log-API oder den Sync; ein ausgeführter Satz behält `planned_item_id`.
- **Nicht gebaut:**
  - `completed` und `plan_changes[]` beim Abschluss: `Adapt` nach dem
    Abschluss einer gestarteten Session. `planned_session_id` ist der
    Anknüpfungspunkt.
  - Der Check-in beim Start (ADAPT-17, optional), nachgezogen in ADR 0019.
  - Der Offline-Start aus §10.5: Die Sync-Operationen tragen weder
    `planned_session_id` noch `planned_item_id`. Offline angelegte Sessions
    bleiben ohne Verweis auf den Plan.
- **Befund für den Review (Sicherheit), behoben in ADR 0017:** Der Server
  ändert einen gestarteten Draft nicht. Das Ereignis bleibt dabei für den Plan wirksam;
  nur der Draft zeigt es nicht. Beispiel: Während einer Einheit meldet der
  User Schmerz oder Symptome, und der neue Plan schliesst eine Region aus
  oder stoppt das Training.
  - Die App muss dann die Antwort des Ereignisses zeigen und die offenen
    geplanten Sätze des Drafts mit der Einheit im neuen Plan abgleichen.
  - Vorschlag: Der Server ersetzt beim Speichern eines neuen Plans die
    offenen geplanten Sätze eines gestarteten Drafts (`is_planned`, ohne
    `completed_at`). Bei einem Stopp löscht er sie; ausgeführte Sätze
    bleiben.
- **Befund:** WEEK-08 verlangt eine Neuerzeugung «ab der nächsten nicht
  begonnenen Einheit». Der Kern erzeugt die ganze Woche neu.
  - Eine gestartete Einheit behält ihre Session. Die neue Einheit desselben
    Tages kann aber anders aussehen.
  - Hat der neue Plan an diesem Tag keine Einheit, zeigt keine geplante
    Einheit die Session. Die Woche kann dann eine Einheit mehr enthalten, als
    der Plan zählt.
  - Das braucht einen Kern, der gestartete Einheiten beim Erzeugen festhält.
