package planning_test

import (
	"fmt"
	"testing"

	"github.com/jaelricco/hefesto/internal/domain/planning"
)

// counter gives IDs that show their order.
func counter() func() string {
	n := 0
	return func() string {
		n++
		return fmt.Sprintf("id-%03d", n)
	}
}

// A planned session becomes one set entry per planned set, each with one
// element carrying the targets (spec §10.2).
func TestMaterialize(t *testing.T) {
	k := kb(t)
	checked := map[string]bool{}
	for _, pa := range personas {
		s, _ := start(t, k, pa.answers())
		p, err := planning.Generate(k, s, now, monday)
		if err != nil {
			t.Fatal(err)
		}
		for si, ps := range p.Sessions {
			for bi := range ps.Blocks {
				for ii := range ps.Blocks[bi].Items {
					ps.Blocks[bi].Items[ii].ID = fmt.Sprintf("item-%d-%d-%d", si, bi, ii)
				}
			}
			d := planning.Materialize(k, ps, counter())
			seen := map[string]bool{}
			if len(d.Blocks) != len(ps.Blocks) || d.Title == "" {
				t.Fatalf("%s: %d blocks for %d planned", pa.name, len(d.Blocks), len(ps.Blocks))
			}
			for bi, b := range ps.Blocks {
				db := d.Blocks[bi]
				want, items := 0, map[string]planning.Item{}
				for _, it := range b.Items {
					if !it.Offer {
						want += it.Sets
						items[it.ID] = it
					}
				}
				if len(db.Sets) != want {
					t.Errorf("%s: block %s has %d sets, want %d", pa.name, b.Role, len(db.Sets), want)
				}
				if b.Paired && len(items) > 1 {
					checked["superset"] = true
					if db.Kind != planning.DraftSuperset || db.Sets[0].Round == nil || *db.Sets[0].Round != 0 ||
						db.Sets[0].ItemID == db.Sets[1].ItemID {
						t.Errorf("%s: a paired block does not alternate by round: %+v", pa.name, db.Sets[:2])
					}
				}
				for _, ds := range db.Sets {
					it, ok := items[ds.ItemID]
					if !ok {
						t.Fatalf("%s: set of unknown or offered item %q", pa.name, ds.ItemID)
					}
					for _, id := range []string{ds.ID, ds.ElementID} {
						if seen[id] {
							t.Fatalf("%s: ID %s given twice", pa.name, id)
						}
						seen[id] = true
					}
					switch {
					case it.HoldS > 0:
						if ds.Measure != planning.MeasureHold || ds.HoldS == nil || *ds.HoldS != it.HoldS || ds.RIR != nil {
							t.Errorf("%s: hold %+v from %+v", pa.name, ds, it)
						}
						checked["hold"] = true
					case it.Reps > 0:
						if ds.Measure != planning.MeasureReps || ds.Reps == nil || *ds.Reps != it.Reps || ds.RIR == nil || *ds.RIR != it.Reserve {
							t.Errorf("%s: reps %+v from %+v", pa.name, ds, it)
						}
						checked["reps"] = true
					}
					if (it.Assist == planning.AssistBand) != (ds.AssistID != "") {
						t.Errorf("%s: band assistance lost on %s", pa.name, it.Exercise)
					}
					if it.Assist == planning.AssistBand {
						checked["band"] = true
					}
					if ds.Kind != it.Kind || ds.RestS != it.RestS || ds.LoadKg != it.LoadKg || ds.Exercise != it.Exercise {
						t.Errorf("%s: set %+v from %+v", pa.name, ds, it)
					}
				}
			}
		}
	}
	for _, c := range []string{"hold", "reps"} {
		if !checked[c] {
			t.Errorf("no persona plan exercised the %s case", c)
		}
	}
}

// An antagonist pair alternates by round; an offer is left out; a band set
// keeps its assistance.
func TestMaterializePairsAndOffers(t *testing.T) {
	k := kb(t)
	ps := planning.PlannedSession{ID: "ps", Blocks: []planning.Block{
		{Role: planning.BlockMax, Paired: true, Items: []planning.Item{
			{ID: "a", Exercise: "planche-tuck", Kind: planning.KindWorking, Sets: 3, HoldS: 6, Reserve: 2, RestS: 120},
			{ID: "b", Exercise: "front-lever-tuck", Kind: planning.KindWorking, Sets: 2, HoldS: 5, Reserve: 2, RestS: 120},
			{ID: "probe", Exercise: "planche-advanced-tuck", Kind: planning.KindWorking, Sets: 1, HoldS: 3, Offer: true},
		}},
		{Role: planning.BlockStrength, Items: []planning.Item{
			{ID: "c", Exercise: "pull-up", Kind: planning.KindWorking, Sets: 2, Reps: 4, Reserve: 3, Assist: planning.AssistBand,
				Stimulus: planning.StimEccentric},
		}},
	}}
	d := planning.Materialize(k, ps, counter())
	pair := d.Blocks[0]
	if pair.Kind != planning.DraftSuperset {
		t.Fatalf("pair block is %s", pair.Kind)
	}
	var order []string
	for _, ds := range pair.Sets {
		order = append(order, fmt.Sprintf("%s%d", ds.ItemID, *ds.Round))
	}
	if got := fmt.Sprint(order); got != "[a0 b0 a1 b1 a2]" {
		t.Errorf("pair order %s", got)
	}
	band := d.Blocks[1]
	if band.Kind != planning.DraftStraight || len(band.Sets) != 2 || band.Sets[0].AssistID == "" || !band.Sets[0].Eccentric ||
		band.Sets[0].Round != nil || *band.Sets[0].RIR != 3 {
		t.Errorf("band block %+v", band)
	}
}

func TestStopRule(t *testing.T) {
	k := kb(t)
	s, _ := start(t, k, persona2())
	if r := k.StopRule(s, now); r != "" {
		t.Errorf("an adult without stops: %s", r)
	}
	s.Constraints = append(s.Constraints, planning.Constraint{Kind: planning.ConstraintStopped, Created: monday})
	if r := k.StopRule(s, now); r != "SAFE-02" {
		t.Errorf("after a stop: %q", r)
	}
	s.Profile.BirthYear = now.Year() - 15
	if r := k.StopRule(s, now); r != "SAFE-07" {
		t.Errorf("a minor: %q", r)
	}
}
