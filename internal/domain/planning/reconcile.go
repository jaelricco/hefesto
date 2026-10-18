package planning

// DraftState is a set of a started log session as the reconciliation reads
// it, in log order: blocks in order, sets in order within their block.
type DraftState struct {
	ID      string
	BlockID string
	// ItemID is the plan item the set comes from; empty for a set the
	// athlete added.
	ItemID string
	// Open is a planned set not yet performed (is_planned, no
	// completed_at). Only open sets change; a performed set is history.
	Open bool
}

// Adjustment is what a new plan changes in a started draft (ADR 0017).
type Adjustment struct {
	// Relink points open sets whose item did not change at the item of the
	// new plan. The log shows no difference.
	Relink []Relink
	// Remove lists open sets the new plan no longer asks for.
	Remove []string
	// Replace swaps an open set whose target changed for a new set in its
	// place. The log never changes an element's values (ADR 0009).
	Replace []Replacement
	// Append adds sets of an item after the item's open sets, in their block.
	Append []Appended
	// Blocks holds new items, in new blocks after the session's blocks.
	Blocks []DraftBlock
}

// Relink is an open set and its item in the new plan.
type Relink struct{ SetID, ItemID string }

// Replacement is an open set and the set that takes its place.
type Replacement struct {
	SetID string
	Set   DraftSet
}

// Appended is a new set in an existing block of the log.
type Appended struct {
	BlockID string
	Set     DraftSet
}

// Visible reports whether the athlete sees the adjustment: anything but a
// relink.
func (a Adjustment) Visible() bool {
	return len(a.Remove)+len(a.Replace)+len(a.Append)+len(a.Blocks) > 0
}

// Empty reports whether there is nothing to write.
func (a Adjustment) Empty() bool { return !a.Visible() && len(a.Relink) == 0 }

// Reconcile compares the planned session a draft was started from, or last
// adjusted to, with its successor in a new plan, and says how the draft's
// open sets follow (ADR 0017). next is nil when the new plan has no session
// on the day, for instance because training stopped: then every open set
// goes. Otherwise items are matched by block role, exercise and kind, in
// order:
//   - an item whose target did not change keeps its open sets, which the
//     athlete may have edited;
//   - an item whose target changed has its open sets replaced;
//   - an item with fewer sets loses open sets from the end, one with more
//     gets sets appended; sets no longer open (performed or deleted by the
//     athlete) count as done;
//   - an item the new plan dropped loses its open sets, a new item comes in
//     a new block.
//
// Performed sets never change. Open sets that belong to no item of prev,
// such as planned sets the athlete added, stay unless next is nil.
func Reconcile(k *Knowledge, prev PlannedSession, next *PlannedSession, sets []DraftState, newID func() string) Adjustment {
	var a Adjustment
	open := map[string][]DraftState{}
	for _, s := range sets {
		if !s.Open {
			continue
		}
		if next == nil {
			a.Remove = append(a.Remove, s.ID)
			continue
		}
		open[s.ItemID] = append(open[s.ItemID], s)
	}
	if next == nil {
		return a
	}

	// Old items waiting for a successor, by key, in session order.
	type key struct{ role, exercise, kind string }
	waiting := map[key][]Item{}
	var order []key
	for _, b := range prev.Blocks {
		for _, it := range b.Items {
			if materialized(it) {
				k := key{b.Role, it.Exercise, it.Kind}
				if len(waiting[k]) == 0 {
					order = append(order, k)
				}
				waiting[k] = append(waiting[k], it)
			}
		}
	}
	for _, b := range next.Blocks {
		fresh := Block{Role: b.Role, Paired: b.Paired}
		for _, n := range b.Items {
			if !materialized(n) {
				continue
			}
			kk := key{b.Role, n.Exercise, n.Kind}
			if len(waiting[kk]) == 0 {
				fresh.Items = append(fresh.Items, n)
				continue
			}
			o := waiting[kk][0]
			waiting[kk] = waiting[kk][1:]
			if more := a.follow(k, o, n, linked(open, o.ID), newID); more > 0 {
				n.Sets = more
				fresh.Items = append(fresh.Items, n)
			}
		}
		if db := draftBlock(k, fresh, newID); len(db.Sets) > 0 {
			a.Blocks = append(a.Blocks, db)
		}
	}
	for _, kk := range order {
		for _, o := range waiting[kk] {
			for _, s := range linked(open, o.ID) {
				a.Remove = append(a.Remove, s.ID)
			}
		}
	}
	return a
}

// linked returns the open sets of an item. A plan from before items had
// IDs links none; the athlete's own sets have no item.
func linked(open map[string][]DraftState, itemID string) []DraftState {
	if itemID == "" {
		return nil
	}
	return open[itemID]
}

// follow moves the open sets of an old item to its successor. It returns
// how many sets the successor still needs in a new block, when the old item
// has no open set to place them after.
func (a *Adjustment) follow(k *Knowledge, o, n Item, open []DraftState, newID func() string) int {
	done := max(0, o.Sets-len(open))
	want := max(0, n.Sets-done)
	same := sameTarget(k, o, n)
	for i, s := range open {
		switch {
		case i >= want:
			a.Remove = append(a.Remove, s.ID)
		case same && o.ID != n.ID:
			a.Relink = append(a.Relink, Relink{SetID: s.ID, ItemID: n.ID})
		case !same:
			a.Replace = append(a.Replace, Replacement{SetID: s.ID, Set: draftSet(k, n, newID)})
		}
	}
	if want <= len(open) {
		return 0
	}
	if len(open) == 0 {
		return want
	}
	block := open[len(open)-1].BlockID
	for range want - len(open) {
		a.Append = append(a.Append, Appended{BlockID: block, Set: draftSet(k, n, newID)})
	}
	return 0
}

// sameTarget reports whether two items ask the log for the same sets.
func sameTarget(k *Knowledge, a, b Item) bool {
	none := func() string { return "" }
	x, y := draftSet(k, a, none), draftSet(k, b, none)
	x.ItemID, y.ItemID = "", ""
	return equalDraftSet(x, y)
}

func equalDraftSet(x, y DraftSet) bool {
	eq := func(p, q *int) bool { return (p == nil) == (q == nil) && (p == nil || *p == *q) }
	return x.Kind == y.Kind && x.RestS == y.RestS && eq(x.RIR, y.RIR) && eq(x.SIR, y.SIR) && x.Exercise == y.Exercise &&
		x.Measure == y.Measure && eq(x.Reps, y.Reps) && eq(x.HoldS, y.HoldS) && x.LoadKg == y.LoadKg &&
		x.Eccentric == y.Eccentric
}

// SessionAdjusted is the change that tells the athlete a started session
// followed a new plan (ADAPT-19).
func (k *Knowledge) SessionAdjusted(sessionID string) Change {
	return Change{Kind: ChangeAdjusted, Session: sessionID, Reasons: []Reason{k.reason(RuleAdjusted)}}
}
