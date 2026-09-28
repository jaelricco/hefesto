package planning

import "time"

// StopRule returns the rule that stops all training, or "" when training
// may go on: SAFE-07 for a minor, SAFE-02 after a stop or unclear exertion
// symptoms. The plan and the start of a planned session share it.
func (k *Knowledge) StopRule(s Snapshot, now time.Time) string {
	switch {
	case isMinor(k, s.Profile.BirthYear, now):
		return RuleMinor
	case hasConstraint(s, ConstraintStopped) || s.Screening.ExertionSymptoms && !s.Screening.Cleared:
		return RuleStopped
	}
	return ""
}

// SessionDraft is a planned session as the training log records it (spec
// §10.2): its blocks with sets in order, one set entry per planned set, each
// with one element that carries the targets. A block without planned sets
// (a general warm-up, a block of offers) is not written; the plan shows it.
// The log session has no title: the app names it from the plan, in the
// athlete's language.
type SessionDraft struct {
	Blocks []DraftBlock
}

// DraftBlock is one block of the draft.
type DraftBlock struct {
	ID   string
	Role string
	// Kind is superset for an antagonist pair, whose sets alternate by
	// round, and straight otherwise.
	Kind string
	Sets []DraftSet
}

// DraftSet is one planned set: the set entry and its one element.
type DraftSet struct {
	ID     string
	ItemID string // the plan item the set comes from
	Round  *int   // the round in a superset
	Kind   string // working or warmup
	RestS  int
	RIR    *int // the target reserve of a rep set; a hold's reserve stays in the plan

	ElementID string
	Exercise  string
	Measure   string // reps or hold_seconds
	Reps      *int
	HoldS     *int
	LoadKg    float64
	Eccentric bool
	// A band target stays in the plan item: the log records assistance
	// with the band used, which the athlete names when logging the set.
}

// Block kinds of the log.
const (
	DraftStraight = "straight"
	DraftSuperset = "superset"
)

// maxRIR is the largest reserve the log records.
const maxRIR = 10

// Materialize turns a planned session into the draft the log stores (spec
// §10.1, §10.2). Offers (probes, first attempts) are left out: they are done
// only on the athlete's active choice, which the client asks for. newID
// gives every block, set and element its ID.
func Materialize(k *Knowledge, p PlannedSession, newID func() string) SessionDraft {
	var d SessionDraft
	for _, b := range p.Blocks {
		if db := draftBlock(k, b, newID); len(db.Sets) > 0 {
			d.Blocks = append(d.Blocks, db)
		}
	}
	return d
}

// materialized reports whether an item becomes sets in the log.
func materialized(it Item) bool { return !it.Offer && it.Sets > 0 }

func draftBlock(k *Knowledge, b Block, newID func() string) DraftBlock {
	db := DraftBlock{ID: newID(), Role: b.Role, Kind: DraftStraight}
	var items []Item
	for _, it := range b.Items {
		if materialized(it) {
			items = append(items, it)
		}
	}
	if b.Paired && len(items) > 1 {
		db.Kind = DraftSuperset
		rounds := 0
		for _, it := range items {
			rounds = max(rounds, it.Sets)
		}
		for r := 0; r < rounds; r++ {
			for _, it := range items {
				if r < it.Sets {
					round := r
					ds := draftSet(k, it, newID)
					ds.Round = &round
					db.Sets = append(db.Sets, ds)
				}
			}
		}
		return db
	}
	for _, it := range items {
		for range it.Sets {
			db.Sets = append(db.Sets, draftSet(k, it, newID))
		}
	}
	return db
}

func draftSet(k *Knowledge, it Item, newID func() string) DraftSet {
	ds := DraftSet{ID: newID(), ItemID: it.ID, Kind: it.Kind, RestS: it.RestS, ElementID: newID(), Exercise: it.Exercise,
		LoadKg: it.LoadKg, Eccentric: it.Stimulus == StimEccentric}
	switch {
	case it.HoldS > 0:
		hold := it.HoldS
		ds.Measure, ds.HoldS = MeasureHold, &hold
	case it.Reps > 0:
		reps := it.Reps
		ds.Measure, ds.Reps = MeasureReps, &reps
	default:
		ds.Measure = MeasureReps
		if ex, ok := k.exercises[it.Exercise]; ok {
			ds.Measure = ex.Measure
		}
	}
	if ds.Measure == MeasureReps {
		rir := min(max(it.Reserve, 0), maxRIR)
		ds.RIR = &rir
	}
	return ds
}
