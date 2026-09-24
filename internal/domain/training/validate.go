package training

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// FieldErrors maps a JSON pointer into the request body to what is wrong
// there. An empty map means valid.
type FieldErrors map[string]string

// Err returns the errors as an error, or nil when there are none.
func (f FieldErrors) Err() error {
	if len(f) == 0 {
		return nil
	}
	return &ValidationError{Fields: f}
}

// ValidationError is returned for input that is well-formed but breaks a
// rule of the domain.
type ValidationError struct {
	Fields FieldErrors
}

func (e *ValidationError) Error() string {
	keys := make([]string, 0, len(e.Fields))
	for k := range e.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + ": " + e.Fields[k]
	}
	return "invalid: " + strings.Join(parts, "; ")
}

var tempoRE = regexp.MustCompile(`^[0-9X]{4}$`)

// ValidateSet checks a set entry and its elements. It also fills in what the
// server derives: element order from array position and each element's
// assistance class. Paths are relative to the set's JSON body.
//
// This is the one validation path for every set. A plain set is not a
// special case; it is a set with one element.
func ValidateSet(s *SetEntry) error {
	errs := FieldErrors{}

	switch n := len(s.Elements); {
	case n == 0:
		errs["/elements"] = "a set has at least one element"
	case n > MaxElementsPerSet:
		errs["/elements"] = fmt.Sprintf("a set has at most %d elements", MaxElementsPerSet)
	}
	if s.IsPlanned && s.CompletedAt != nil {
		errs["/completed_at"] = "a planned set has not been performed yet"
	}
	if s.RPE != nil && math.Mod(*s.RPE*2, 1) != 0 {
		errs["/rpe"] = "RPE is given in half steps, e.g. 7.5"
	}

	seen := map[uuid.UUID]bool{}
	for i := range s.Elements {
		e := &s.Elements[i]
		p := fmt.Sprintf("/elements/%d", i)
		e.OrderIndex = i
		if seen[e.ID] {
			errs[p+"/id"] = "element ids must be unique within a set"
		}
		seen[e.ID] = true
		validateElement(errs, p, e)
		if s.CompletedAt != nil && !e.HasValue() {
			errs[p+"/"+string(e.Measure)] = "a completed set records a value for every element"
		}
		e.AssistanceClass = ClassOf(*e)
	}
	return errs.Err()
}

func validateElement(errs FieldErrors, p string, e *Element) {
	values := map[Measure]bool{
		MeasureReps:        e.Reps != nil,
		MeasureHoldSeconds: e.HoldSeconds != nil,
		MeasureDistanceM:   e.DistanceM != nil,
	}
	for m, set := range values {
		if set && m != e.Measure {
			errs[p+"/"+string(m)] = fmt.Sprintf("only allowed when measure is %s", m)
		}
	}
	if e.Tempo != nil && !tempoRE.MatchString(*e.Tempo) {
		errs[p+"/tempo"] = `four characters, digits or X, e.g. "30X1"`
	}
	if e.LoadKg < 0 {
		errs[p+"/load_kg"] = "added load is never negative; record assistance instead"
	}
	if e.ROMNote != nil && !e.IsPartialROM {
		errs[p+"/rom_note"] = "only allowed with is_partial_rom"
	}
	if e.FormQuality != nil && (*e.FormQuality < 1 || *e.FormQuality > 5) {
		errs[p+"/form_quality"] = "between 1 and 5"
	}
	if a := e.Assistance; a != nil {
		ap := p + "/assistance"
		if a.Type == AssistanceTypeBand {
			if a.BandID == nil {
				errs[ap+"/band_id"] = "required for band assistance"
			}
			if a.BandCount < 1 {
				errs[ap+"/band_count"] = "at least 1 for band assistance"
			}
		} else {
			if a.BandID != nil {
				errs[ap+"/band_id"] = "only allowed for band assistance"
			}
			if a.BandCount != 0 {
				errs[ap+"/band_count"] = "only allowed for band assistance"
			}
		}
	}
}

// ValidateBlock checks a block's own fields.
func ValidateBlock(b Block) error {
	errs := FieldErrors{}
	if b.IntervalS != nil && b.Kind != BlockKindEMOM && b.Kind != BlockKindAMRAP {
		errs["/interval_s"] = "only for emom and amrap blocks"
	}
	return errs.Err()
}

// ValidateSession checks a session's own fields after an update has been
// applied to it.
func ValidateSession(s Session) error {
	errs := FieldErrors{}
	if s.EndedAt != nil && s.EndedAt.Before(s.StartedAt) {
		errs["/ended_at"] = "a session cannot end before it starts"
	}
	if s.PerceivedFatigue != nil && (*s.PerceivedFatigue < 1 || *s.PerceivedFatigue > 10) {
		errs["/perceived_fatigue"] = "between 1 and 10"
	}
	if s.BodyweightKg != nil && *s.BodyweightKg <= 0 {
		errs["/bodyweight_kg"] = "must be positive"
	}
	return errs.Err()
}

// CanTransition reports whether a client may move a session between two
// statuses by editing it. Completion has its own operation (it runs the
// unlock engine), and a completed session is not reopened by an edit.
func CanTransition(from, to string) bool {
	if from == to {
		return true
	}
	switch from {
	case StatusDraft:
		return to == StatusAbandoned
	case StatusAbandoned:
		return to == StatusDraft
	default:
		return false
	}
}

// ValidateOrder checks that ids is exactly a permutation of current: every
// live sibling named once, nothing else.
func ValidateOrder(path string, ids, current []uuid.UUID) error {
	errs := FieldErrors{}
	want := make(map[uuid.UUID]bool, len(current))
	for _, id := range current {
		want[id] = true
	}
	seen := make(map[uuid.UUID]bool, len(ids))
	for i, id := range ids {
		switch {
		case seen[id]:
			errs[fmt.Sprintf("%s/%d", path, i)] = "listed twice"
		case !want[id]:
			errs[fmt.Sprintf("%s/%d", path, i)] = "not a live child here"
		}
		seen[id] = true
	}
	if len(errs) == 0 && len(seen) != len(want) {
		errs[path] = fmt.Sprintf("must list all %d live children, lists %d", len(want), len(seen))
	}
	return errs.Err()
}
