# ADR 0013 — Persistenz des Trainingsplaners

- Status: proposed (zur Prüfung mit dem Postgres-Adapter, 28.09.2026)
- Date: 2026-09-28
- Deciders: Jaelricco

Dieses ADR ist auf Deutsch, wie alle Dokumente des Planer-Tracks.

## Kontext

Der Kern des Planers (`internal/domain/planning`, ADR 0012) arbeitet auf
einem Snapshot je User: Profil, Ziele, Kapazitäten, Leitern, Regionen,
Screening, Auflagen, Trainingsphase, Pause, die geloggten Einheiten und die
Schmerzberichte. `Adapt` gibt nach jedem Ereignis einen ganzen neuen
Snapshot zurück (U-21). Der Anwendungsdienst `internal/planning` speichert
ihn über Ports; bisher gab es nur einen Speicher im Arbeitsspeicher.

`spec.md` §4.9 skizziert die Tabellen. Beim Umsetzen zeigten sich vier
Fragen, die die Skizze offen liess:

- Die Übungen und Leitern der Wissensbasis (`content/training/`) sind noch
  keine Zeilen der Content-Tabellen (U-1, ENT-10). Fremdschlüssel auf
  `exercises` oder `skills` sind deshalb nicht möglich.
- Der Kern liest die geloggten Einheiten mit den Slugs der Wissensbasis.
  Die Log-Tabellen verweisen auf Content-Übungen, die Übungen des Planers
  können dort also noch gar nicht stehen.
- Ein Snapshot muss unverändert zurückkommen. Sonst ändert sich der
  `input_hash`, und derselbe Plan wäre nicht mehr reproduzierbar (spec §1.3).
- Zwei Ereignisse desselben Users können gleichzeitig eintreffen, etwa ein
  wiederholter Abschluss oder der Sync zweier Geräte. Heute prüft der Dienst
  «schon gesehen» und schreibt danach, ohne Transaktion.

## Entscheidungen

### 1. Tabellen nach Zuständigkeit, ein Snapshot aus vielen Zeilen

Migration `00009_planning.sql` legt die Tabellen der Skizze an, zugeschnitten
auf den Snapshot:

- `user_training_profiles`, `user_goals`
- Gesundheitsdaten (mit der Einwilligung zu löschen): `user_screening`,
  `user_region_status`, `user_pain_reports`
- `planning_constraints`: Auflagen ohne Antworten, sie überleben einen
  Widerruf der Einwilligung (ENT-S-7)
- `user_capacity_estimates`, `user_ladder_states`
- `user_planner_states` (Mesozyklus, Deloads, Spielraum, Einstiegs-Konten,
  erreichte Stufen), `user_training_breaks`
- `planner_sessions` (Verlauf, §3)
- `training_plans`, `planned_sessions`
- `plan_decisions`

Der Adapter `internal/store/planning.go` setzt den Snapshot daraus zusammen
und zerlegt ihn wieder. Er merkt sich die in der Transaktion gelesenen
Zeilen und schreibt beim Speichern nur geänderte. Die Gesundheitsdaten
lassen sich so getrennt löschen, und Endpunkte wie `GET /v1/me/regions`
lesen nur ihre Tabelle.

Abweichungen von der Skizze:
- Das Equipment ist eine Spalte des Profils (`text[]`), keine eigene Tabelle.
  Der Kern kennt es nur als Liste, und beim späteren Sync wird das Profil
  ganz geschrieben.
- `planned_sessions` hat keinen eigenen Payload. Der Inhalt steht einmal im
  Plan (`training_plans.payload`); die Zeile ist der adressierbare Griff mit
  Status und Verweis auf die Log-Session (§10.2).
- `plan_decisions` hat eine Zeile je Ereignis mit allen Änderungen, nicht je
  Regel. `(trigger, source_id)` ist eindeutig und macht Ereignisse
  idempotent. Wie `skill_unlock_events` ist die Tabelle nur anhängbar.
- `user_bands`, `user_red_flag_answers`, `planner_knowledge` und die neuen
  Spalten an `workout_sessions` und `set_entries` fehlen noch. Der Snapshot
  braucht sie nicht; sie kommen mit den Endpunkten und dem Start einer
  geplanten Einheit (§10.2).

### 2. Wissensbasis-Slugs statt Fremdschlüssel

Skills, Stufen, Übungen, Regionen und Lastkonten sind Text mit einem
benannten Format-`CHECK`. Die Wissensbasis wird beim Start geprüft (KB-01 bis
KB-13), und jeder Plan trägt ihre `ruleset_version`. Ein Snapshot mit einem
Slug, den die Wissensbasis nicht kennt, ist ein Eingabefehler des Kerns,
kein Datenbankfehler. Sobald die Übungen des Planers Content-Zeilen sind,
kann eine additive Migration Fremdschlüssel ergänzen (Expand, Deploy,
Contract).

### 3. Der Verlauf ist vorerst eine Kopie im Planer

`planner_sessions` hält die abgeschlossenen Einheiten so, wie der Kern sie
gelesen hat: Datum, Deload, Ermüdung und die Sätze als `jsonb` mit Slugs.
Die ID ist die der Log-Session. Der Fremdschlüssel `(id, user_id)` auf
`workout_sessions` hält den Verlauf an das Log gebunden und schliesst fremde
Sessions aus.

Das ist eine Kopie. Sobald die Übungen des Planers im Log stehen können,
lässt sich der Verlauf aus den Log-Tabellen ableiten, wie §4.1 es vorsieht,
und die Tabelle kann entfallen. Die Schmerzberichte sind dagegen schon die
endgültige Tabelle (`user_pain_reports`). Sie haben keinen Fremdschlüssel
auf die Session, weil ein offline geschriebener Bericht vor seiner Session
ankommen kann.

Dafür bekommt `PainReport` eine `ID`. Der Kern liest sie nicht; der Dienst
setzt sie aus dem Ereignis bzw. vergibt eine UUIDv7 für den Basiswert aus
dem Onboarding. Verlauf und Schmerzberichte wachsen nur und werden in der
Reihenfolge gelesen, in der der Kern sie hat (Datum und Eingang bzw.
Eingang).

### 4. Exakte Rundreise

Ein gespeicherter Snapshot liest sich byte-gleich zurück. Dafür gilt:
- Alle Zeitpunkte sind `timestamptz`, auch Kalendertage, die der Kern als
  Mitternacht UTC führt. Die Nullzeit («nie») ist `NULL`.
- Der Dienst rundet seine Uhr auf Mikrosekunden, die Auflösung von Postgres.
- Berechnete Zahlen sind `double precision`, damit ein `float64`
  unverändert zurückkommt.
- Maps, die der Kern leer oder `nil` führt, bleiben es (`jsonb` bzw.
  `NULL`).

Zwei Tests prüfen das gegen echtes Postgres:
- ein Snapshot, in dem jedes Feld gesetzt ist; ein Test schlägt fehl, wenn
  ein neues Feld in der Fixture fehlt;
- der Dienst mit drei Onboarding-Profilen über drei Wochen, einmal mit dem
  Speicher im Arbeitsspeicher, einmal mit Postgres. Snapshots, Pläne und
  Änderungen müssen gleich sein.

### 5. Eine Transaktion je Aufruf, eine Sperre je User

Der Dienst bekommt den Port `Transactor`. Jeder Aufruf läuft in einer
Transaktion: Laden, «schon gesehen», `Adapt`, Snapshot speichern, Änderungen
protokollieren und Plan erzeugen. Die Transaktion hält eine
Advisory-Sperre je User (`pg_advisory_xact_lock`). Damit laufen Ereignisse
eines Users nacheinander, und ein Ereignis wirkt ganz oder gar nicht. Andere
User warten nicht. Die Zeile in `users` wird bewusst nicht gesperrt, weil
`touch_sync` sie bei jedem Sync-Schreibzugriff aktualisiert. Der
Speicher im Arbeitsspeicher serialisiert gleich, rollt aber nicht zurück.

### 6. Pläne werden ersetzt, nicht überschrieben

Jede Neuerzeugung speichert einen neuen Plan und setzt den aktiven Plan
derselben Woche auf `superseded`. Ein eindeutiger Teilindex erlaubt
höchstens einen aktiven Plan je User und Woche. Ersetzte Pläne bleiben für
die Nachvollziehbarkeit. Wann sie gelöscht werden, ist offen.

## Folgen

- Der Planer kann auf Postgres laufen. Verdrahtet ist er noch nicht: Die
  Endpunkte folgen nach OpenAPI (ADR 0007).
- Nichts davon wird synchronisiert. Profil, Ziele und Schmerzberichte
  bekommen ihre Sync-Spalten mit den Endpunkten, additiv.
- Der Widerruf der Einwilligung ist noch nicht gebaut. Die Tabellen dafür
  sind getrennt, und Auflagen bleiben erhalten.
- Ein Speichern schreibt nur geänderte Zeilen. Beim ersten Speichern nach
  dem Onboarding sind es ein paar Dutzend Upserts in einer Transaktion.
- Verlauf und Schmerzberichte werden bei jedem Aufruf ganz gelesen. Bei
  Jahren an Logs braucht es ein Fenster. Wie tief der Kern mindestens lesen
  muss (Pausenerkennung, Referenzmittel, Plateaus), ist dann festzulegen.
