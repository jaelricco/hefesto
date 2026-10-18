package planning

import (
	"cmp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Catalogue views of the knowledge base for the explanations and the
// clients (spec §10.3). They return copies; the knowledge base is shared.

// Rules returns the rule catalogue in ID order.
func (k *Knowledge) Rules() []Rule { return inIDOrder(k.rules) }

// Sources returns the sources in ID order (A-9 before A-10).
func (k *Knowledge) Sources() []Source { return inIDOrder(k.sources) }

// Params returns the parameter catalogue in ID order.
func (k *Knowledge) Params() []Param { return inIDOrder(k.params) }

// Regions returns the body-map regions in the order of the knowledge base.
func (k *Knowledge) Regions() []Region {
	out := make([]Region, 0, len(k.regionIDs))
	for _, id := range k.regionIDs {
		out = append(out, k.regions[id])
	}
	return out
}

// Region returns one body-map region.
func (k *Knowledge) Region(id string) (Region, bool) {
	r, ok := k.regions[id]
	return r, ok
}

// Skills returns the skills in slug order.
func (k *Knowledge) Skills() []Skill {
	out := make([]Skill, 0, len(k.skillIDs))
	for _, id := range k.skillIDs {
		out = append(out, *k.skills[id])
	}
	return out
}

// Exercises returns the exercises in slug order.
func (k *Knowledge) Exercises() []Exercise {
	out := make([]Exercise, 0, len(k.exercises))
	for _, slug := range sortedKeys(k.exercises) {
		out = append(out, *k.exercises[slug])
	}
	return out
}

// Onboarding returns the answer classes of the onboarding questions.
func (k *Knowledge) Onboarding() OnboardingConfig { return k.onboarding }

// IsMinor reports whether a user born in birthYear counts as a minor at now
// (PAR-D-23).
func (k *Knowledge) IsMinor(birthYear int, now time.Time) bool { return isMinor(k, birthYear, now) }

// inIDOrder returns the values ordered by their key, comparing the digits at
// the end of a key as a number.
func inIDOrder[V any](m map[string]V) []V {
	keys := sortedKeys(m)
	slices.SortFunc(keys, compareIDs)
	out := make([]V, len(keys))
	for i, key := range keys {
		out[i] = m[key]
	}
	return out
}

func compareIDs(a, b string) int {
	pa, na := splitID(a)
	pb, nb := splitID(b)
	return cmp.Or(strings.Compare(pa, pb), cmp.Compare(na, nb), strings.Compare(a, b))
}

// splitID splits "A-100" into "A-" and 100; an ID without trailing digits
// has the number -1.
func splitID(id string) (string, int) {
	i := len(id)
	for i > 0 && id[i-1] >= '0' && id[i-1] <= '9' {
		i--
	}
	n, err := strconv.Atoi(id[i:])
	if err != nil {
		return id, -1
	}
	return id[:i], n
}

// RegionStatus is one region as the user sees it (spec §8, GET
// /v1/me/regions): its tolerance state, the constraints on it and the
// red-flag questions asked for it. Without consent to health data no state
// is kept (Tracked false); the constraints still hold (ENT-S-7).
type RegionStatus struct {
	Region      Region
	Tracked     bool
	State       RegionState
	Constraints []string // kinds of the constraints naming the region
	RedFlags    []RedFlag
}

// RegionStatuses lists the regions with a state or a constraint in the
// order of the knowledge base.
func (k *Knowledge) RegionStatuses(s Snapshot, now time.Time) []RegionStatus {
	minor := isMinor(k, s.Profile.BirthYear, now)
	var out []RegionStatus
	for _, id := range k.regionIDs {
		rs, tracked := s.Regions[id]
		var kinds []string
		for _, c := range s.Constraints {
			if c.Region == id && !slices.Contains(kinds, c.Kind) {
				kinds = append(kinds, c.Kind)
			}
		}
		if !tracked && len(kinds) == 0 {
			continue
		}
		slices.Sort(kinds)
		out = append(out, RegionStatus{Region: k.regions[id], Tracked: tracked, State: rs, Constraints: kinds,
			RedFlags: k.RedFlags(id, minor)})
	}
	return out
}
