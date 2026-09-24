package progress

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

// ErrInvalidCriteria is returned for criteria the evaluator cannot interpret.
// contentlint and the JSON Schema keep these out of the database; the check
// here is the last line.
var ErrInvalidCriteria = errors.New("invalid unlock criteria")

// Assistance classes of an observed element (mirrors training.AssistanceClass).
const (
	ClassUnassisted = "unassisted"
	ClassAssisted   = "assisted"
	ClassLoaded     = "loaded"
)

// Observation is one logged set element, reduced to what criteria can ask
// about. The store builds these from set_elements joined to their set entry
// and session.
type Observation struct {
	SetEntryID    uuid.UUID
	SessionID     uuid.UUID
	Exercise      string // exercise slug, as criteria name it
	Measure       string
	Value         *float64 // nil when the element recorded no value
	LoadKg        float64
	Assistance    string // unassisted | assisted | loaded
	FormQuality   *int
	Failed        bool
	PartialROM    bool
	EccentricOnly bool
	PerformedAt   time.Time
}

// SetHistory is an athlete's observations, in any order.
type SetHistory []Observation

// ConditionResult explains one condition.
type ConditionResult struct {
	Condition Condition
	Met       bool
	// Count is the number of distinct set entries that satisfy the condition.
	Count int
	// Best is the best value among strict candidates: the right exercise and
	// measure, the required assistance and load, full range, not failed.
	// It ignores the threshold, form and window (but not the future), so it
	// shows progress towards a condition not yet met.
	Best *float64
	// Evidence is the most recent qualifying set entry, when met.
	Evidence *uuid.UUID
	// Attempts counts distinct set entries that were strict candidates.
	Attempts int
	// LastAttempt is when the most recent candidate was performed.
	LastAttempt *time.Time
}

// Result is the outcome of evaluating criteria.
type Result struct {
	// Met: every All condition holds and, if Any is non-empty, one Any does.
	Met bool
	// SelfAttestOnly: empty criteria never unlock automatically.
	SelfAttestOnly bool
	All            []ConditionResult
	Any            []ConditionResult
	// Started: the athlete has logged at least one of the exercises named by
	// the criteria, in the named measure, in any form.
	Started bool
	// Evidence is the most recent evidence among the conditions that made
	// the criteria hold.
	Evidence *uuid.UUID
}

// Evaluate checks criteria against an athlete's history as of now. It is a
// pure function: no clock, no I/O.
//
// What counts as an occurrence of a condition:
//
//   - an element of the named exercise, recorded in the named measure, whose
//     value passes op/value;
//   - performed, not failed, with full range of motion, not eccentric-only;
//   - assistance "none" (the default) excludes assisted elements; "any"
//     admits them; added load is governed by min/max_load_kg alone;
//   - with min_form_quality, only elements rated at least that well;
//   - with within_days, only elements performed in the last within_days×24h;
//   - occurrences counts distinct set entries: two qualifying elements in one
//     combo are one occurrence, because they are one attempt.
func Evaluate(c Criteria, h SetHistory, now time.Time) (Result, error) {
	c = c.WithDefaults()
	if err := check(c); err != nil {
		return Result{}, err
	}
	res := Result{SelfAttestOnly: c.IsEmpty()}
	if res.SelfAttestOnly {
		return res, nil
	}

	for _, cond := range c.All {
		res.All = append(res.All, evaluate(cond, h, now))
	}
	for _, cond := range c.Any {
		res.Any = append(res.Any, evaluate(cond, h, now))
	}

	allMet := true
	for _, r := range res.All {
		allMet = allMet && r.Met
		res.Evidence = latest(res.Evidence, r.Evidence, h)
	}
	anyMet := len(res.Any) == 0
	for _, r := range res.Any {
		if r.Met {
			anyMet = true
			res.Evidence = latest(res.Evidence, r.Evidence, h)
		}
	}
	res.Met = allMet && anyMet
	if !res.Met {
		res.Evidence = nil
	}

	for _, cond := range c.Conditions() {
		for _, o := range h {
			if o.Exercise == cond.Exercise && o.Measure == cond.Measure {
				res.Started = true
			}
		}
	}
	return res, nil
}

func check(c Criteria) error {
	for i, cond := range c.Conditions() {
		switch {
		case cond.Exercise == "":
			return fmt.Errorf("%w: condition %d names no exercise", ErrInvalidCriteria, i)
		case cond.Measure != "reps" && cond.Measure != "hold_seconds" && cond.Measure != "distance_m":
			return fmt.Errorf("%w: condition %d measure %q", ErrInvalidCriteria, i, cond.Measure)
		case cond.Op != ">=" && cond.Op != ">":
			return fmt.Errorf("%w: condition %d op %q", ErrInvalidCriteria, i, cond.Op)
		case cond.Assistance != AssistanceNone && cond.Assistance != AssistanceAny:
			return fmt.Errorf("%w: condition %d assistance %q", ErrInvalidCriteria, i, cond.Assistance)
		case cond.Occurrences < 1:
			return fmt.Errorf("%w: condition %d occurrences %d", ErrInvalidCriteria, i, cond.Occurrences)
		case cond.WithinDays != nil && *cond.WithinDays < 1:
			return fmt.Errorf("%w: condition %d within_days %d", ErrInvalidCriteria, i, *cond.WithinDays)
		case cond.MinFormQuality != nil && (*cond.MinFormQuality < 1 || *cond.MinFormQuality > 5):
			return fmt.Errorf("%w: condition %d min_form_quality %d", ErrInvalidCriteria, i, *cond.MinFormQuality)
		}
	}
	return nil
}

// candidate reports whether o is a strict attempt at cond, ignoring the
// threshold, form and time window.
func candidate(cond Condition, o Observation) bool {
	switch {
	case o.Exercise != cond.Exercise || o.Measure != cond.Measure || o.Value == nil:
		return false
	case o.Failed || o.PartialROM || o.EccentricOnly:
		return false
	case cond.Assistance == AssistanceNone && o.Assistance == ClassAssisted:
		return false
	case cond.MinLoadKg != nil && o.LoadKg < *cond.MinLoadKg:
		return false
	case cond.MaxLoadKg != nil && o.LoadKg > *cond.MaxLoadKg:
		return false
	}
	return true
}

func qualifies(cond Condition, o Observation, now time.Time) bool {
	if !candidate(cond, o) {
		return false
	}
	v := *o.Value
	if cond.Op == ">=" && v < cond.Value || cond.Op == ">" && v <= cond.Value {
		return false
	}
	if cond.MinFormQuality != nil && (o.FormQuality == nil || *o.FormQuality < *cond.MinFormQuality) {
		return false
	}
	if cond.WithinDays != nil {
		since := now.Add(-time.Duration(*cond.WithinDays) * 24 * time.Hour)
		if o.PerformedAt.Before(since) {
			return false
		}
	}
	return !o.PerformedAt.After(now)
}

func evaluate(cond Condition, h SetHistory, now time.Time) ConditionResult {
	r := ConditionResult{Condition: cond}
	type hit struct {
		id uuid.UUID
		at time.Time
	}
	hits := map[uuid.UUID]time.Time{}
	attempts := map[uuid.UUID]bool{}
	for _, o := range h {
		if candidate(cond, o) && !o.PerformedAt.After(now) {
			attempts[o.SetEntryID] = true
			if r.LastAttempt == nil || o.PerformedAt.After(*r.LastAttempt) {
				at := o.PerformedAt
				r.LastAttempt = &at
			}
			if r.Best == nil || *o.Value > *r.Best {
				v := *o.Value
				r.Best = &v
			}
		}
		if qualifies(cond, o, now) {
			if at, ok := hits[o.SetEntryID]; !ok || o.PerformedAt.After(at) {
				hits[o.SetEntryID] = o.PerformedAt
			}
		}
	}
	r.Attempts = len(attempts)
	r.Count = len(hits)
	r.Met = r.Count >= cond.Occurrences
	if r.Met {
		list := make([]hit, 0, len(hits))
		for id, at := range hits {
			list = append(list, hit{id, at})
		}
		// Most recent first; ties broken by id so the result is deterministic.
		sort.Slice(list, func(i, j int) bool {
			if !list[i].at.Equal(list[j].at) {
				return list[i].at.After(list[j].at)
			}
			return list[i].id.String() < list[j].id.String()
		})
		ev := list[0].id
		r.Evidence = &ev
	}
	return r
}

// latest returns whichever of two set entries was performed more recently.
func latest(a, b *uuid.UUID, h SetHistory) *uuid.UUID {
	switch {
	case b == nil:
		return a
	case a == nil:
		return b
	}
	at := func(id uuid.UUID) time.Time {
		var t time.Time
		for _, o := range h {
			if o.SetEntryID == id && o.PerformedAt.After(t) {
				t = o.PerformedAt
			}
		}
		return t
	}
	ta, tb := at(*a), at(*b)
	if tb.After(ta) || tb.Equal(ta) && b.String() < a.String() {
		return b
	}
	return a
}

// Primary is the condition that describes a level's headline number (its
// best value and staleness): the first `all` condition, else the first
// `any`. It is nil for self-attest-only criteria.
func (r Result) Primary() *ConditionResult {
	switch {
	case len(r.All) > 0:
		return &r.All[0]
	case len(r.Any) > 0:
		return &r.Any[0]
	}
	return nil
}
