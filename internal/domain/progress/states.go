package progress

import (
	"fmt"
	"sort"

	"github.com/google/uuid"
)

// State is a user's standing on one skill level.
type State string

// States, from nothing to achieved. Unlocked is terminal.
const (
	Locked     State = "locked"
	Available  State = "available"
	InProgress State = "in_progress"
	Unlocked   State = "unlocked"
)

// LevelNode is a skill level as the state machine sees it: its prerequisite
// levels (including the implicit previous level of its own skill).
type LevelNode struct {
	ID            uuid.UUID
	Prerequisites []uuid.UUID
}

// Advance computes every level's state.
//
//   - An unlocked level stays unlocked, whatever else holds (ADR 0003 §6).
//   - A level whose prerequisites are not all unlocked is locked, even if its
//     criteria are met: the map is a path, and the self-attest fallback is
//     how an athlete fills in levels they skipped logging.
//   - A level with its prerequisites unlocked unlocks when its criteria are
//     met, is in progress when the athlete has started on it, and is
//     available otherwise.
//
// Levels are visited in prerequisite order, so evidence for a whole chain
// unlocks the whole chain in one pass. results may omit levels (treated as
// not met, not started); unlocked lists the levels that became unlocked, in
// that order.
func Advance(levels []LevelNode, current map[uuid.UUID]State, results map[uuid.UUID]Result) (next map[uuid.UUID]State, unlocked []uuid.UUID, err error) {
	order, err := topoSort(levels)
	if err != nil {
		return nil, nil, err
	}
	next = make(map[uuid.UUID]State, len(levels))
	for _, n := range order {
		if current[n.ID] == Unlocked {
			next[n.ID] = Unlocked
			continue
		}
		ready := true
		for _, p := range n.Prerequisites {
			if next[p] != Unlocked {
				ready = false
				break
			}
		}
		r := results[n.ID]
		switch {
		case !ready:
			next[n.ID] = Locked
		case r.Met:
			next[n.ID] = Unlocked
			unlocked = append(unlocked, n.ID)
		case r.Started:
			next[n.ID] = InProgress
		default:
			next[n.ID] = Available
		}
	}
	return next, unlocked, nil
}

// topoSort orders levels so that every prerequisite comes first. Ties are
// broken by id for determinism. Prerequisites outside the set are ignored
// for ordering (and, in Advance, are never unlocked, so they keep the level
// locked).
func topoSort(levels []LevelNode) ([]LevelNode, error) {
	byID := make(map[uuid.UUID]LevelNode, len(levels))
	indeg := make(map[uuid.UUID]int, len(levels))
	out := map[uuid.UUID][]uuid.UUID{}
	for _, n := range levels {
		byID[n.ID] = n
		indeg[n.ID] += 0
	}
	for _, n := range levels {
		for _, p := range n.Prerequisites {
			if _, ok := byID[p]; ok {
				out[p] = append(out[p], n.ID)
				indeg[n.ID]++
			}
		}
	}
	var ready []uuid.UUID
	for id, d := range indeg {
		if d == 0 {
			ready = append(ready, id)
		}
	}
	var order []LevelNode
	for len(ready) > 0 {
		sort.Slice(ready, func(i, j int) bool { return ready[i].String() < ready[j].String() })
		id := ready[0]
		ready = ready[1:]
		order = append(order, byID[id])
		for _, m := range out[id] {
			indeg[m]--
			if indeg[m] == 0 {
				ready = append(ready, m)
			}
		}
	}
	if len(order) != len(levels) {
		return nil, fmt.Errorf("%w: prerequisite cycle among %d levels", ErrInvalidCriteria, len(levels)-len(order))
	}
	return order, nil
}
