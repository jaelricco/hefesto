// Package progress evaluates unlock criteria, XP and streaks.
//
// The evaluator is a pure function of (criteria, history, now), which is what
// makes the golden-file suite in testdata/criteria possible. It must stay that
// way: no clock reads, no database, no randomness.
package progress
