// Package planning is the training-plan generator: a deterministic,
// rule-based core that turns a knowledge base and a user-state snapshot into
// a weekly plan, and adapts that state from logged sessions and pain reports.
//
// The specification is docs/algorithm/spec.md (German); rule IDs such as
// DOSE-01 and parameter IDs such as PAR-D-09 in comments refer to it and to
// the research files under docs/research/. ADR 0012 records the design.
//
// Nothing here performs I/O and nothing reads a clock: every function takes
// the knowledge base, a snapshot and the instant it should consider "now".
// Identical inputs give byte-identical plans. The planner never writes
// unlocks, XP, streaks or training days; those stay with package progress.
package planning
