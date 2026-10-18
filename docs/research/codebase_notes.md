# Codebase-Notizen für den Trainingsplan-Algorithmus

> Phase 1. Kurze Bestandsaufnahme dessen, woran der Planer andocken muss.
> Stand: Branch `claude/busy-babbage-fqio1j`, basierend auf `main` nach Phase 6
> (Merge #9).

## 0. Vorab: Abweichung zwischen Auftrag und Repository

Der Auftrag nennt **rung.fit** mit **Svelte-Frontend**. Das Repository heisst
**Hefesto** (`hefesto.fit`, ADR 0002). Sein einziger Client ist eine **iOS-App
(SwiftUI)**; Svelte kommt im Repository nicht vor. Das Backend ist wie
beschrieben Go.

Für diese Arbeit hat das wenig Folgen: Die API ist OpenAPI-first (ADR 0007),
jeder Client, auch ein Svelte-Client, lässt sich daraus generieren. Die
Spezifikation (Phase 4) beschreibt die Schnittstellen deshalb client-neutral.
**Offene Frage für den Checkpoint:** Ist rung.fit ein neuer Name oder ein
zweiter Client für dieselbe API?

## 1. Stack und Regeln, die für den Planer gelten

| Thema | Stand | Folge für den Planer |
|---|---|---|
| Sprache/Frameworks | Go 1.24, `chi`, `pgx/v5`, `sqlc`, `goose`, Postgres 16 | Keine neuen Frameworks nötig. |
| Domänenschicht | `internal/domain/*` ist rein; `arch_test.go` verbietet `os`, `database/sql`, `net/http`, `internal/store`, `internal/http` … | Der Planer-Kern wird ein neues Paket unter `internal/domain/` (z. B. `planning`), ohne I/O. Die Wissensbasis wird ausserhalb geladen und als Wert hineingereicht. |
| Content | YAML in `content/`, JSON-Schema-validiert (`internal/content`, `cmd/contentlint`), idempotenter Seed per Slug, `content_versions` mit Checksumme | Die Wissensbasis des Planers passt in dieses Muster: YAML + Schema + Lint. |
| API | `api/openapi.yaml` ist Quelle der Wahrheit; Request-Bodies werden zur Laufzeit dagegen validiert, Contract-Tests prüfen Routen und Antworten (ADR 0007) | Neue Endpunkte zuerst im Spec. |
| Fehler | `%w`, `application/problem+json`, `slog` mit Request-ID; kein `panic` ausser in `main` | — |
| Tests | Table-driven; Golden Files unter `testdata/`; Integration gegen echtes Postgres (testcontainers) | Persona-Szenarien passen als Golden Files. |
| Migrationen | Nur Struktur; nie editiert; expand → deploy → contract | Neue Tabellen (Profil, Plan) sind additive Migrationen. |
| Schema-Konventionen | UUIDv7 in Go, `text` + benannter `CHECK` statt ENUM, `user_id` auf jeder Zeile, zusammengesetzte FKs `(id, user_id)`, Sync-Spalten | Gilt auch für neue Tabellen. |

## 2. Das Log-Datenmodell (was der Planer lesen kann)

```
workout_sessions ─┬─ session_blocks ─── set_entries ─── set_elements ─── set_element_assistance
                  │   (straight|superset|circuit|emom|amrap)
                  └─ user_training_days (Rollup pro Tag)
```

| Tabelle | Felder, die für Adaption relevant sind |
|---|---|
| `workout_sessions` | `local_date`, `status` (nur `completed` zählt), `perceived_fatigue` 1–10, `bodyweight_kg`, `is_rest_day`, `template_id`, `started_at`/`ended_at` (Dauer) |
| `set_entries` | `kind` (`working`, `warmup`, `backoff`, `drop`, `cluster`, `test`), `is_planned`, `rest_after_planned_s`, `rest_after_actual_s`, `rpe` 1–10, `rir` 0–10, `completed_at` |
| `set_elements` | `exercise_id`, `measure` (`reps`/`hold_seconds`/`distance_m`/`none`), Wert, `tempo` (`30X1`), `load_kg` (nur Zusatzlast, ≥ 0), `is_eccentric_only`, `is_partial_rom`, `form_quality` 1–5, `failed`, `assistance_class` (`unassisted`/`assisted`/`loaded`) |
| `set_element_assistance` | `type` (band, partner, machine, incline, counterweight, foot_support), `band_id`, `band_count`, `estimated_assist_kg` |
| `user_bodyweight_log` | Körpergewicht an trainingsfreien Tagen |
| `user_training_days` | `had_session`, `planned_rest`, `deload`, `freeze_used` (Streak nach ADR 0003) |

Beobachtungen:

- **Eine Kombination ist ein `set_entry` mit N `set_elements`.** Der Planer muss
  Kombinationen (z. B. «Hold to Press», P-01 S. 2) genauso als einen Satz mit
  zwei Elementen erzeugen und auswerten. Kein zweiter Codepfad.
- RPE/RIR hängen am **Satz**, Formqualität am **Element**. Beides ist optional.
  Der Planer muss mit fehlenden Werten umgehen.
- `is_planned` erlaubt geplante, noch nicht ausgeführte Sätze in einer Session.
  Das ist ein natürlicher Träger für den generierten Plan.
- `user_training_days.deload` existiert, **wird aber von keinem Code
  geschrieben**. Ein Planer mit Deload-Wochen kann die Spalte endlich befüllen.
- `user_exercise_bests` existiert als Cache-Tabelle, **wird aber noch nicht
  befüllt** (ADR 0008: «nicht nötig, solange die Historie klein ist»).

## 3. Templates

`workout_templates → template_blocks → template_set_entries →
template_set_elements` spiegeln die Log-Struktur mit Zielwerten (`target_reps`,
`target_hold_seconds`, `target_load_kg`, `tempo`, `target_rpe`,
`rest_after_planned_s`). Sie sind **in der Datenbank, aber nicht in der API**:
kein `/v1/templates`, und der Sync kennt nur `session`, `block`, `set`,
`bodyweight`. Ein generierter Plan könnte als Template-Folge oder als Draft-
Sessions mit geplanten Sätzen ausgeliefert werden. Die Wahl gehört in die
Spezifikation (Phase 4).

## 4. Content: Skills, Levels, Übungen

| Objekt | Felder | Stand |
|---|---|---|
| `families` | push, pull, core, legs, handstand, dynamic, mobility | fertig |
| `exercises` | `default_measure`, `load_semantics`, `is_bodyweight`, `unilateral`, `tempo_applicable`, `equipment[]` (Freitext), `cues[]`, `common_faults[]` | **7 Platzhalter** (`status: draft_placeholder`) |
| `skills` | `difficulty_tier` 1–10, `is_milestone`, `primary_muscles[]`, `common_faults[]`, Map-Position | **3 Platzhalter**: front-lever, handstand, pull-up |
| `skill_levels` | `order`, `unlock_criteria` (DSL), `est_weeks_from_prev`, Übungen mit Rolle `primary_test`/`progression`/`accessory`/`prehab` | 4 Levels |
| `skill_edges` | `prerequisite` (DAG, zyklenfrei geprüft), `recommended`, `alternative`, `antagonist` mit `weight` | 1 Kante |
| `skill_injury_risks` | `region`, `name`, `risk_factors[]`, `early_signs[]`, Prehab-Übungen, `disclaimer: educational_only` (DB-`CHECK`) | 1 Eintrag |

**Unlock-DSL** (`internal/domain/progress/criteria.go`): `all`/`any` aus
Bedingungen `{exercise, measure, op (>=|>), value, assistance (none|any),
min_load_kg, max_load_kg, min_form_quality, occurrences, within_days}`. Ein Level
ohne Kriterien ist nur selbst bestätigbar.

**Unlock-Engine** (ADR 0008): `progress.Evaluate` ist rein; `progress.Advance`
läuft in topologischer Reihenfolge; ein Level bleibt `locked`, bis alle
Voraussetzungen `unlocked` sind; Unlocks werden nie zurückgenommen (DB-Trigger);
fehlgeschlagene, partielle und rein exzentrische Elemente zählen nie.

## 5. Was für den Planer fehlt

Der Planer braucht Daten, die es heute nicht gibt:

1. **Nutzerprofil für das Training:** Ziele und Prioritäten, Equipment,
   Verfügbarkeit (Tage, Minuten), Trainingsalter, Verletzungen und
   Einschränkungen, Testergebnisse mit Konfidenz. Heute kennt `users` nur
   Locale, Einheiten, Zeitzone.
2. **Übungs-Metadaten für die Belastungssteuerung:** Bewegungsmuster,
   Straight-Arm vs. Bent-Arm, belastete Gelenke und Strukturen mit Gewichtung,
   Muskeln, Schwierigkeit relativ zu anderen Übungen, Regressionen und
   Progressionen, Kontraindikationen, Dosierungsbereiche. `equipment` ist
   heute Freitext ohne Vokabular.
3. **Pläne:** keine Tabellen für Plan, Mesozyklus, Woche, geplante Einheit.
4. **Parameter der Methodik:** Satz-/Wiederholungs-/Pausenbereiche,
   Steigerungsraten, Volumengrenzen. Nichts davon existiert.
5. **Echte Inhalte:** Alle Skills und Übungen sind Platzhalter. CLAUDE.md und
   `CONTENT_AUTHORING.md` verbieten erfundene Inhalte; die Recherche dieser
   Arbeit ist die erste belegte Quelle dafür.

## 6. Leitplanken aus bestehenden ADRs, die der Planer erben muss

- **ADR 0003:** Keine Mechanik, die Pausen bestraft. Deloads und geplante Ruhe
  zählen als eingehalten. Keine Maximalversuche als Belohnungsziel. Für den
  Planer heisst das: Ruhetage und Deloads sind Teil des Plans und werden im
  Streak als geplante Ruhe bzw. Deload erfasst. Kein «du bist im Rückstand».
- **ADR 0003 / Brief §11:** Keine medizinischen Aussagen. Verletzungsinhalte
  tragen `disclaimer: educational_only` im Payload.
- **`CONTENT_AUTHORING.md`:** Verletzungsinhalte enthalten «keine Diagnosen,
  Behandlungspläne oder Return-to-Training-Ratschläge». Der Auftrag verlangt
  aber einen schrittweisen Wiedereinstieg nach Beschwerden. **Spannung, die im
  Checkpoint geklärt werden muss.** Vorschlag: Der Planer passt nur die
  Trainingslast an (weniger, leichter, schmerzfrei) und formuliert das als
  Trainingssteuerung, nicht als Rehabilitation. Bei Red Flags oder
  anhaltenden Beschwerden verweist er an Fachpersonen und plant die betroffene
  Struktur nicht mehr, bis der User eine Freigabe bestätigt.
- **ADR 0008:** Unlocks sind Erfolgsprotokoll, nicht aktuelle Form. Für den
  Planer zählt die aktuelle Form (`best_value`, jüngste Logs), nicht der
  Unlock-Status allein. Nach einer Pause kann ein Level entsperrt sein, aber
  nicht mehr trainierbar auf diesem Niveau.

## 7. Wo der Planer ins Repository passt (vorläufig)

| Teil | Ort |
|---|---|
| Reiner Kern | `internal/domain/planning` (neu), ruft `internal/domain/progress` für Kriterien auf |
| Wissensbasis | `content/` (neue Dateien für Methodik-Parameter, Übungs-Metadaten, Kontraindikationen), Schema in `content/schema/`, Laden und Validieren in `internal/content` |
| Datenzugriff | `internal/store` über Interfaces, die der Kern definiert |
| API | neue Pfade in `api/openapi.yaml`, Handler in `internal/http` |
| Tests | table-driven im Kern; Persona-Szenarien als Golden Files in `testdata/planning/` |

Die endgültige Aufteilung legt die Spezifikation in Phase 4 fest.
