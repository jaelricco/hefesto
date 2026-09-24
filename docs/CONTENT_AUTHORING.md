# Content authoring

Everything a user sees about skills, exercises, bands and injuries comes from
YAML files in `content/`. This page describes every field, so content can be
written without reading Go. The JSON Schemas in `content/schema/` are the
machine-readable version of the same rules; where this page and a schema
disagree, the schema wins and this page has a bug.

```
content/
├── families.yaml          the family vocabulary
├── exercises/<slug>.yaml  one file per exercise
├── skills/<slug>.yaml     one file per skill, with its levels and injury notes
├── bands/<brand>.yaml     global band catalogue, one file per brand
└── schema/*.schema.json   validation rules
```

## Workflow

```sh
make content-validate   # schema + cross-file checks; run it after every edit
make seed               # import into your local database (make up does this too)
```

`make content-validate` must be clean before a merge; CI runs it with
`-strict`, so warnings fail there too. Production imports the content on every
deploy.

## Ground rules

- **Do not invent data.** Numbers in unlock criteria, band resistances and
  injury notes must come from research. Until then, mark the file
  `status: draft_placeholder` and say so in its summary.
- **Injury content is educational, not medical advice.** Every injury entry
  carries `disclaimer: educational_only`; the API sends it with the content.
  Do not write diagnoses, treatment plans or return-to-training advice.
- **Slugs are permanent.** Users' logs and progress point at them. Use
  lowercase letters, digits and single hyphens: `front-lever-tuck`.
- **A file is named after its slug**: `exercises/front-lever-tuck.yaml` holds
  `slug: front-lever-tuck`.

## Families — `families.yaml`

```yaml
families:
  - { slug: push, name: Push }
```

| Field | Required | Meaning |
|---|---|---|
| `slug` | yes | Referenced by skills and exercises. |
| `name` | yes | Display name. |

File order is display order.

## Exercises — `exercises/<slug>.yaml`

An exercise is **something an athlete performs and logs**: `front-lever-tuck`,
`weighted-pull-up`, `band-assisted-muscle-up`. It is not a skill. A skill
level points at exercises; the logger records exercises.

```yaml
slug: front-lever-tuck
status: active
name: Tuck Front Lever
aka: [Tuck FL]
family: pull
default_measure: hold_seconds
load_semantics: added
is_bodyweight: true
unilateral: false
tempo_applicable: true
equipment: [bar, rings]
summary: One or two sentences on what the exercise is.
cues:
  - Short coaching cue
common_faults:
  - Short description of a common fault
```

| Field | Required | Default | Meaning |
|---|---|---|---|
| `slug` | yes | | Permanent identifier. |
| `status` | no | `active` | `active`, `draft_placeholder`, or `retired` (hidden from search, still valid in old logs). |
| `name` | yes | | Display name. |
| `aka` | no | `[]` | Other names; searched. |
| `family` | yes | | A slug from `families.yaml`. |
| `default_measure` | yes | | What the logger asks for: `reps`, `hold_seconds`, `distance_m`, or `none`. |
| `load_semantics` | no | `added` | How a logged load reads: `added` (extra weight on a bodyweight movement), `external` (the implement is the load, e.g. a dumbbell curl), `none` (load is meaningless). |
| `is_bodyweight` | no | `true` | Whether bodyweight is part of the resistance. |
| `unilateral` | no | `false` | Performed one side at a time. |
| `tempo_applicable` | no | `true` | Whether the logger offers a tempo field. |
| `equipment` | no | `[]` | Free-text tags: `bar`, `rings`, `parallettes`, `floor`, `wall`, `band`, … |
| `summary` | no | `""` | Short description. |
| `cues` | no | `[]` | Coaching cues, one line each. |
| `common_faults` | no | `[]` | Faults to watch for, one line each. |

An exercise that nothing references (no level, criterion or injury note) is
reported as a warning.

## Skills — `skills/<slug>.yaml`

A skill is **a node on the map**: `front-lever`. It has ordered levels, each
tested and trained by exercises.

```yaml
slug: front-lever
status: active
name: Front Lever
family: pull
difficulty_tier: 7
is_milestone: true
aka: [FL]
summary: >
  What the skill is.
primary_muscles: [latissimus_dorsi, lower_trapezius]
common_faults:
  - Hips sagging below shoulder line
map: { constellation: pull_north, x: 340, y: 120 }
levels:
  - slug: tuck
    name: Tuck Front Lever
    order: 1
    description: What this level looks like.
    est_weeks_from_prev: 6
    exercises:
      - { slug: front-lever-tuck, role: primary_test }
      - { slug: scapular-pull-up, role: progression }
    unlock_criteria:
      all:
        - { exercise: front-lever-tuck, measure: hold_seconds, op: ">=", value: 15, occurrences: 2, within_days: 30 }
    prerequisites:
      - { skill: pull-up, level: strict-5 }
    links:
      - { skill: planche, level: tuck, relation: antagonist }
injuries:
  - region: elbow
    name: Medial epicondylalgia
    description: Educational description.
    risk_factors: [Rapid volume jumps]
    early_signs: [Pain on the inner elbow after pulling]
    prehab_exercises: [ { slug: pronated-curl-eccentric } ]
    disclaimer: educational_only
```

### Skill fields

| Field | Required | Default | Meaning |
|---|---|---|---|
| `slug` | yes | | Permanent identifier. |
| `status` | no | `active` | As for exercises. Deleting a skill file retires the skill. |
| `name` | yes | | Display name. |
| `family` | yes | | A slug from `families.yaml`. |
| `difficulty_tier` | yes | | 1 (easiest) to 10. |
| `is_milestone` | no | `false` | Drawn large on the map. A milestone must have `map`. |
| `aka` | no | `[]` | Other names; searched. |
| `summary` | no | `""` | What the skill is. |
| `primary_muscles` | no | `[]` | snake_case muscle names. |
| `common_faults` | no | `[]` | One line each. |
| `map` | milestones | | Position on the constellation map: `constellation` (a group name, lowercase with `_` or `-`), `x`, `y`. Positions are designed, not computed; two skills cannot share a spot in one constellation. |
| `levels` | yes | | At least one; see below. |
| `injuries` | no | `[]` | See below. |

### Level fields

| Field | Required | Default | Meaning |
|---|---|---|---|
| `slug` | yes | | Unique within the skill. **Levels cannot be removed or renamed** once seeded; user progress points at them. |
| `name` | yes | | Display name. |
| `order` | yes | | 1, 2, 3, … without gaps. |
| `description` | no | `""` | What achieving this level looks like. |
| `est_weeks_from_prev` | no | | Typical weeks from the previous level. A hint for display only. |
| `exercises` | yes | | `{ slug, role }` pairs. `role` is `primary_test` (exactly one per level — the exercise that proves the level), `progression`, `accessory` or `prehab`. |
| `unlock_criteria` | no | `{}` | When the level unlocks automatically; see below. Empty means self-attest only. |
| `prerequisites` | no | `[]` | Levels of **other** skills that must be unlocked first: `{ skill, level, weight? }`. |
| `links` | no | `[]` | Other relationships: `{ skill, level, relation, weight? }` with `relation` one of `recommended`, `alternative`, `antagonist`. |

**Each level implicitly requires the previous level of the same skill.** Do
not write that as a prerequisite. A level with no prerequisite of any kind —
in practice, level 1 of a skill with no cross-skill prerequisites — is a root,
available from account creation.

Prerequisites must not form a cycle; the linter prints the levels involved.

### Unlock criteria

A level unlocks when **every** condition in `all` holds and, if `any` is
present, **at least one** condition in `any` holds. The condition is checked
against the athlete's logged set elements when a session is completed.

| Field | Required | Default | Meaning |
|---|---|---|---|
| `exercise` | yes | | Exercise slug. Usually the level's `primary_test`; the linter warns otherwise. |
| `measure` | yes | | `reps`, `hold_seconds` or `distance_m`. The linter warns if it differs from the exercise's `default_measure`. |
| `op` | yes | | `>=` or `>`. |
| `value` | yes | | The threshold. |
| `assistance` | no | `none` | `none`: only unassisted elements count. `any`: assisted elements count too. |
| `min_load_kg` | no | | Minimum added load, for weighted standards. |
| `max_load_kg` | no | | Maximum added load; `0` means strictly unweighted. |
| `min_form_quality` | no | | 1–5; elements logged with lower form quality do not count. |
| `occurrences` | no | `1` | How many separate qualifying elements are needed. Prefer `2` or more, so one grinding attempt does not unlock a level (ADR 0003 §5). |
| `within_days` | no | | Qualifying elements must fall inside this many days. |

### Injury fields

| Field | Required | Meaning |
|---|---|---|
| `region` | yes | Body region: `elbow`, `shoulder`, `wrist`, `lumbar`, … |
| `name` | yes | Name of the condition. `(region, name)` is unique within a skill. |
| `description` | no | Educational description. |
| `risk_factors` | no | Training patterns associated with it, one line each. |
| `early_signs` | no | Signs worth noticing, one line each. |
| `prehab_exercises` | no | `{ slug }` list of exercises. |
| `disclaimer` | yes | Always `educational_only`. |

## Bands — `bands/<brand>.yaml`

The global catalogue. Users add their own bands through the app.

```yaml
bands:
  - brand: Example Brand
    colour_label: Red
    resistance_min_kg: 5
    resistance_max_kg: 15
    length_cm: 208
    thickness_mm: 4.5
```

| Field | Required | Meaning |
|---|---|---|
| `brand` | yes | Manufacturer. |
| `colour_label` | yes | The colour or name the manufacturer uses. `(brand, colour_label)` is unique. |
| `resistance_min_kg`, `resistance_max_kg` | yes | The manufacturer's stated range; min ≤ max. |
| `length_cm`, `thickness_mm` | no | Physical size. |

Removing a band from the catalogue hides it from new logs; old logs keep it.

## What the linter checks

1. Every file validates against its schema.
2. Slugs are unique; each file is named after its slug.
3. Every prerequisite and link names a level that exists.
4. Every referenced exercise and family exists.
5. The prerequisite graph, including each skill's level ladder, has no cycle.
6. No orphan exercises (warning).
7. Level orders run 1..n, and each level has exactly one `primary_test`.
8. Milestones are on the map, and no two skills share a spot.
9. Injury entries carry `disclaimer: educational_only`.

Plus: criteria testing an exercise not attached to the level, or a measure
different from the exercise's default (warnings), and `min_load_kg` above
`max_load_kg` (error).
