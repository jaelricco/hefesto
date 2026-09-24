package progress

import (
	"testing"

	"github.com/google/uuid"
)

func id(n byte) uuid.UUID { return uuid.UUID{15: n} }

// A ladder: 1 -> 2 -> 3, and 4 depends on 2 as well (a cross-skill edge).
var ladder = []LevelNode{
	{ID: id(3), Prerequisites: []uuid.UUID{id(2)}},
	{ID: id(1)},
	{ID: id(4), Prerequisites: []uuid.UUID{id(2)}},
	{ID: id(2), Prerequisites: []uuid.UUID{id(1)}},
}

func TestAdvance(t *testing.T) {
	met := Result{Met: true}
	started := Result{Started: true}

	cases := []struct {
		name     string
		current  map[uuid.UUID]State
		results  map[uuid.UUID]Result
		want     map[uuid.UUID]State
		unlocked []uuid.UUID
	}{
		{
			name: "fresh athlete: only the root is available",
			want: map[uuid.UUID]State{id(1): Available, id(2): Locked, id(3): Locked, id(4): Locked},
		},
		{
			name:    "started on the root",
			results: map[uuid.UUID]Result{id(1): started},
			want:    map[uuid.UUID]State{id(1): InProgress, id(2): Locked, id(3): Locked, id(4): Locked},
		},
		{
			name:     "a whole chain proved at once unlocks in one pass",
			results:  map[uuid.UUID]Result{id(1): met, id(2): met, id(3): met},
			want:     map[uuid.UUID]State{id(1): Unlocked, id(2): Unlocked, id(3): Unlocked, id(4): Available},
			unlocked: []uuid.UUID{id(1), id(2), id(3)},
		},
		{
			name:    "criteria met with a prerequisite missing stays locked",
			results: map[uuid.UUID]Result{id(3): met},
			want:    map[uuid.UUID]State{id(1): Available, id(2): Locked, id(3): Locked, id(4): Locked},
		},
		{
			name:     "unlocks are never revoked",
			current:  map[uuid.UUID]State{id(1): Unlocked, id(2): Unlocked},
			results:  map[uuid.UUID]Result{id(4): started},
			want:     map[uuid.UUID]State{id(1): Unlocked, id(2): Unlocked, id(3): Available, id(4): InProgress},
			unlocked: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, unlocked, err := Advance(ladder, tc.current, tc.results)
			if err != nil {
				t.Fatal(err)
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("level %d: %s, want %s", k[15], got[k], v)
				}
			}
			if len(unlocked) != len(tc.unlocked) {
				t.Fatalf("unlocked %v, want %v", unlocked, tc.unlocked)
			}
			for i := range unlocked {
				if unlocked[i] != tc.unlocked[i] {
					t.Fatalf("unlocked %v, want %v (prerequisite order)", unlocked, tc.unlocked)
				}
			}
		})
	}
}

func TestAdvanceEdgeCases(t *testing.T) {
	// A prerequisite outside the graph keeps the level locked.
	got, _, err := Advance([]LevelNode{{ID: id(1), Prerequisites: []uuid.UUID{id(9)}}}, nil, map[uuid.UUID]Result{id(1): {Met: true}})
	if err != nil || got[id(1)] != Locked {
		t.Fatalf("got %v %v", got, err)
	}
	// A cycle is an error, never a silent partial answer.
	cyclic := []LevelNode{{ID: id(1), Prerequisites: []uuid.UUID{id(2)}}, {ID: id(2), Prerequisites: []uuid.UUID{id(1)}}}
	if _, _, err := Advance(cyclic, nil, nil); err == nil {
		t.Fatal("cycle accepted")
	}
}
