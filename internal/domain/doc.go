// Package domain holds pure types and business rules.
//
// Nothing here performs I/O. No package under internal/domain may import
// internal/store, internal/http, internal/auth, internal/media or
// internal/sync; arch_test.go enforces that.
//
// Subpackages:
//
//	training — sessions, blocks, set entries, set elements, assistance
//	skills   — the skill graph, levels and states
//	progress — the unlock rules engine, XP and streaks
package domain
