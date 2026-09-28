package planning_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/jaelricco/hefesto/internal/domain/planning"
)

// session is a planned session of two blocks: tuck planche holds and
// pull-ups, with the item IDs given.
func session(tuckID string, tuckSets, tuckHold int, pullID string, pullSets int) planning.PlannedSession {
	return planning.PlannedSession{ID: "ps", Blocks: []planning.Block{
		{Role: planning.BlockMax, Items: []planning.Item{
			{ID: tuckID, Exercise: "planche-tuck", Kind: planning.KindWorking, Sets: tuckSets, HoldS: tuckHold, Reserve: 2, RestS: 120},
			{ID: "offer-" + tuckID, Exercise: "planche-advanced-tuck", Kind: planning.KindWorking, Sets: 1, HoldS: 3, Offer: true},
		}},
		{Role: planning.BlockStrength, Items: []planning.Item{
			{ID: pullID, Exercise: "pull-up", Kind: planning.KindWorking, Sets: pullSets, Reps: 5, Reserve: 2, RestS: 150},
		}},
	}}
}

// logged is the draft of a session as the log holds it: every set open.
func logged(ps planning.PlannedSession) []planning.DraftState {
	var out []planning.DraftState
	d := planning.Materialize(nil, ps, counter())
	for _, b := range d.Blocks {
		for _, s := range b.Sets {
			out = append(out, planning.DraftState{ID: s.ID, BlockID: b.ID, ItemID: s.ItemID, Open: true})
		}
	}
	return out
}

func ids(sets []planning.DraftState, pick ...int) []string {
	var out []string
	for _, i := range pick {
		out = append(out, sets[i].ID)
	}
	return out
}

func TestReconcile(t *testing.T) {
	k := kb(t)
	prev := session("t1", 3, 6, "p1", 2)
	base := logged(prev) // tuck: sets 0-2 in block A; pull-up: sets 3-4 in block B

	t.Run("a stop removes every open set, the athlete's too", func(t *testing.T) {
		sets := slices.Clone(base)
		sets[0].Open = false // performed
		sets = append(sets, planning.DraftState{ID: "own", BlockID: sets[4].BlockID, Open: true})
		a := planning.Reconcile(k, prev, nil, sets, counter())
		if want := []string{sets[1].ID, sets[2].ID, sets[3].ID, sets[4].ID, "own"}; !slices.Equal(a.Remove, want) {
			t.Errorf("removed %v, want %v", a.Remove, want)
		}
		if len(a.Replace)+len(a.Append)+len(a.Blocks)+len(a.Relink) != 0 {
			t.Errorf("more than removals: %+v", a)
		}
	})

	t.Run("an unchanged session only follows the new item IDs", func(t *testing.T) {
		sets := append(slices.Clone(base), planning.DraftState{ID: "own", BlockID: base[4].BlockID, Open: true})
		a := planning.Reconcile(k, prev, ptr(session("t2", 3, 6, "p2", 2)), sets, counter())
		if a.Visible() || len(a.Relink) != 5 || a.Relink[0] != (planning.Relink{SetID: base[0].ID, ItemID: "t2"}) ||
			a.Relink[4] != (planning.Relink{SetID: base[4].ID, ItemID: "p2"}) {
			t.Errorf("adjustment %+v", a)
		}
		if a := planning.Reconcile(k, prev, &prev, base, counter()); !a.Empty() {
			t.Errorf("the same plan changed something: %+v", a)
		}
	})

	t.Run("a lower target replaces the open sets, done ones count", func(t *testing.T) {
		sets := slices.Clone(base)
		sets[0].Open = false
		a := planning.Reconcile(k, prev, ptr(session("t2", 3, 4, "p2", 2)), sets, counter())
		if len(a.Replace) != 2 || a.Replace[0].SetID != sets[1].ID || a.Replace[1].SetID != sets[2].ID {
			t.Fatalf("replaced %+v", a.Replace)
		}
		if r := a.Replace[0].Set; *r.HoldS != 4 || r.ItemID != "t2" || r.Exercise != "planche-tuck" {
			t.Errorf("replacement %+v", r)
		}
		if len(a.Remove)+len(a.Append)+len(a.Blocks) != 0 || len(a.Relink) != 2 {
			t.Errorf("adjustment %+v", a)
		}
	})

	t.Run("fewer sets remove from the end, more are appended", func(t *testing.T) {
		sets := slices.Clone(base)
		sets[0].Open = false // one tuck set done; the new plan wants two in all
		a := planning.Reconcile(k, prev, ptr(session("t2", 2, 6, "p2", 4)), sets, counter())
		if want := ids(sets, 2); !slices.Equal(a.Remove, want) {
			t.Errorf("removed %v, want %v", a.Remove, want)
		}
		if len(a.Append) != 2 || a.Append[0].BlockID != sets[4].BlockID || a.Append[0].Set.ItemID != "p2" {
			t.Errorf("appended %+v", a.Append)
		}
		if len(a.Relink) != 3 || len(a.Replace)+len(a.Blocks) != 0 {
			t.Errorf("adjustment %+v", a)
		}
	})

	t.Run("a dropped item goes, a new one comes in a new block", func(t *testing.T) {
		next := session("t2", 3, 6, "p2", 2)
		next.Blocks[1].Items[0].Exercise = "chin-up"
		a := planning.Reconcile(k, prev, &next, base, counter())
		if want := ids(base, 3, 4); !slices.Equal(a.Remove, want) {
			t.Errorf("removed %v, want %v", a.Remove, want)
		}
		if len(a.Blocks) != 1 || a.Blocks[0].Role != planning.BlockStrength || len(a.Blocks[0].Sets) != 2 ||
			a.Blocks[0].Sets[0].Exercise != "chin-up" {
			t.Errorf("new blocks %+v", a.Blocks)
		}
	})

	t.Run("more sets of an item done in full come in a new block", func(t *testing.T) {
		sets := slices.Clone(base)
		sets[3].Open, sets[4].Open = false, false
		a := planning.Reconcile(k, prev, ptr(session("t2", 3, 6, "p2", 3)), sets, counter())
		if len(a.Blocks) != 1 || len(a.Blocks[0].Sets) != 1 || a.Blocks[0].Sets[0].ItemID != "p2" || len(a.Append) != 0 {
			t.Errorf("adjustment %+v", a)
		}
	})

	t.Run("a pair changes its target together", func(t *testing.T) {
		pair := func(a, b string, hold int) planning.PlannedSession {
			return planning.PlannedSession{Blocks: []planning.Block{{Role: planning.BlockMax, Paired: true, Items: []planning.Item{
				{ID: a, Exercise: "planche-tuck", Kind: planning.KindWorking, Sets: 2, HoldS: hold, Reserve: 2, RestS: 120},
				{ID: b, Exercise: "front-lever-tuck", Kind: planning.KindWorking, Sets: 2, HoldS: 5, Reserve: 2, RestS: 120},
			}}}}
		}
		old := pair("a1", "b1", 6)
		a := planning.Reconcile(k, old, ptr(pair("a2", "b2", 4)), logged(old), counter())
		var got []string
		for _, r := range a.Replace {
			got = append(got, fmt.Sprintf("%s:%d", r.Set.ItemID, *r.Set.HoldS))
		}
		if fmt.Sprint(got) != "[a2:4 a2:4]" || len(a.Relink) != 2 {
			t.Errorf("replaced %v, relinked %+v", got, a.Relink)
		}
	})
}

func ptr[T any](v T) *T { return &v }

func TestSessionAdjustedChange(t *testing.T) {
	c := kb(t).SessionAdjusted("s-1")
	if c.Kind != planning.ChangeAdjusted || c.Session != "s-1" || len(c.Reasons) != 1 ||
		c.Reasons[0].RuleID != planning.RuleAdjusted || c.Reasons[0].Text == "" {
		t.Errorf("change %+v", c)
	}
}
