# Unlock-criteria golden suite

Each `*.json` file is one case for `progress.Evaluate`, run by
`internal/domain/progress/golden_test.go`. Expectations are written by hand —
never generated from the code under test.

```json
{
  "description": "what this case pins down",
  "now": "2026-09-30T12:00:00Z",
  "criteria": { "all": [ ... ], "any": [ ... ] },
  "history": [
    { "set": "s1", "exercise": "front-lever-tuck", "measure": "hold_seconds",
      "value": 15, "load_kg": 0, "assistance": "unassisted",
      "form_quality": 4, "failed": false, "partial_rom": false,
      "eccentric_only": false, "at": "2026-09-20T18:00:00Z" }
  ],
  "expect": {
    "met": true, "self_attest_only": false, "started": true,
    "evidence": "s1",
    "all": [ { "met": true, "count": 1, "best": 15 } ],
    "any": []
  }
}
```

`set` is a name; observations with the same name belong to one set entry
(a combo). Omitted observation fields take their zero value, and
`assistance` defaults to `unassisted`. A case that must be rejected uses
`"expect_error": "<substring>"` instead of `expect`.
