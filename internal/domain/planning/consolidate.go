package planning

import (
	"math"
	"slices"
	"time"
)

// consolidate turns a session with fewer than PAR-S-50 working sets into a
// planned rest day and moves its sets into sessions of the week that already
// hold the same exercise, so the spacing does not change (WEEK-09,
// ENT-R-5). Prehab moves to a session without it, or is dropped while it
// stays in PAR-D-37 (lo) sessions. A session is kept when a ramp counts it
// (PAR-D-26), when a set has nowhere to go, or when a move would break a
// session cap, the straight-arm budget or the time.
func (g *gen) consolidate() {
	k := g.k
	for si := range g.plan.Sessions {
		n := workingSets(g.plan.Sessions[si])
		if n == 0 || n >= int(k.T.MinSessionSets) || g.countsForRamp(si) {
			continue
		}
		saved := cloneSessions(g.plan.Sessions)
		if !g.moveOut(si) {
			g.plan.Sessions = saved
			continue
		}
		g.plan.Reasons = append(g.plan.Reasons, k.reason(RuleConsolidate,
			"date", g.plan.Sessions[si].Date.Format(time.DateOnly), "sets", n))
	}
}

// workingSets counts the sets of a session outside the warm-up and prehab.
func workingSets(ps PlannedSession) int {
	n := 0
	for _, b := range ps.Blocks {
		if b.Role == BlockWarmup {
			continue
		}
		for _, it := range b.Items {
			if it.Sets > 0 && it.Stimulus != StimPrehab {
				n += it.Sets
			}
		}
	}
	return n
}

// countsForRamp reports whether a ramp counts session si as one of its
// sessions (PAR-D-26): a break ramp counts straight-arm and wrist load, a
// ramp after a complaint every session that loads the region at 2 or more.
func (g *gen) countsForRamp(si int) bool {
	k := g.k
	for _, b := range g.plan.Sessions[si].Blocks {
		for _, it := range b.Items {
			ex := k.exercises[it.Exercise]
			if it.Sets <= 0 || it.Kind == KindWarmup || ex == nil {
				continue
			}
			if g.breakRamp() {
				for a := range g.itemLoad(it) {
					if isStraightAccount(a) {
						return true
					}
				}
			}
			for id, rs := range g.s.Regions {
				if st := rttStage(rs.State); st >= 1 && st <= 4 && k.maxRating(id, ex) >= 2 {
					return true
				}
			}
		}
	}
	return false
}

// moveOut empties session si into the other sessions of the week.
func (g *gen) moveOut(si int) bool {
	moved := map[int]map[string]float64{} // session → moved load per structure
	for bi := range g.plan.Sessions[si].Blocks {
		role := g.plan.Sessions[si].Blocks[bi].Role
		for ii := range g.plan.Sessions[si].Blocks[bi].Items {
			it := g.plan.Sessions[si].Blocks[bi].Items[ii]
			if it.Sets <= 0 || role == BlockWarmup && it.Stimulus != StimPrehab {
				continue // ramp sets of the warm-up go with their work
			}
			var ok bool
			if it.Stimulus == StimPrehab {
				ok = g.movePrehab(si, role, it, moved)
			} else {
				ok = g.moveWork(si, role, it, moved)
			}
			if !ok {
				return false
			}
			g.plan.Sessions[si].Blocks[bi].Items[ii].Sets = 0
		}
	}
	src := &g.plan.Sessions[si]
	for i := range src.Blocks {
		src.Blocks[i].Items = slices.DeleteFunc(src.Blocks[i].Items, func(it Item) bool { return it.Sets <= 0 })
	}
	src.Blocks = slices.DeleteFunc(src.Blocks, func(b Block) bool { return len(b.Items) == 0 && b.Role != BlockWarmup })
	return true
}

// moveWork places the sets of it in another session with training: on the
// same exercise at the same intensity class, or, for a light item (no
// spacing, LOAD-05), also in a session without it. The session with the
// fewest sets of the exercise goes first, then the one with the fewest
// working sets. Offers stay where they are; a calibration only joins a
// calibration.
func (g *gen) moveWork(si int, role string, it Item, moved map[int]map[string]float64) bool {
	if it.Offer {
		return false
	}
	type target struct{ s, b, i, sets, work int }
	var cands []target
	for sj := range g.plan.Sessions {
		n := workingSets(g.plan.Sessions[sj])
		if sj == si || n == 0 {
			continue
		}
		found := false
		for b, blk := range g.plan.Sessions[sj].Blocks {
			for i, o := range blk.Items {
				if o.Exercise != it.Exercise || o.Sets <= 0 {
					continue
				}
				found = true
				if o.Kind == it.Kind && o.Stimulus == it.Stimulus && o.Class == it.Class && !o.Offer && o.Calibration == it.Calibration {
					cands = append(cands, target{sj, b, i, o.Sets, n})
				}
			}
		}
		if !found && it.Class == classLight && !g.resting(g.k.exercises[it.Exercise], g.plan.Sessions[sj].Date) {
			cands = append(cands, target{sj, -1, -1, 0, n})
		}
	}
	slices.SortStableFunc(cands, func(a, b target) int {
		if a.sets != b.sets {
			return a.sets - b.sets
		}
		return a.work - b.work
	})
	date := g.plan.Sessions[si].Date.Format(time.DateOnly)
	for _, t := range cands {
		saved := cloneSessions(g.plan.Sessions)
		var dst *Item
		if t.b >= 0 {
			dst = &g.plan.Sessions[t.s].Blocks[t.b].Items[t.i]
			dst.Sets += it.Sets
		} else {
			dst = insertItem(&g.plan.Sessions[t.s], role, it, g.k.reason(ruleOfBlock(role)))
		}
		g.note(dst, g.k.reason(RuleConsolidate, "date", date, "sets", it.Sets), "")
		if g.filled(t.s, it, moved) {
			return true
		}
		g.plan.Sessions = saved
	}
	return false
}

// movePrehab moves a prehab item to a session with training that has no
// such item, into the block of the same role. Without one it is dropped
// while the exercise stays in PAR-D-37 (lo) sessions of the week.
func (g *gen) movePrehab(si int, role string, it Item, moved map[int]map[string]float64) bool {
	have := 0
	for sj := range g.plan.Sessions {
		if sj == si || workingSets(g.plan.Sessions[sj]) == 0 {
			continue
		}
		if g.hasExercise(sj, it.Exercise) {
			have++
			continue
		}
		saved := cloneSessions(g.plan.Sessions)
		insertItem(&g.plan.Sessions[sj], role, it, g.k.reason(ruleOfBlock(role)))
		if g.filled(sj, it, moved) {
			return true
		}
		g.plan.Sessions = saved
	}
	return float64(have) >= g.k.T.PrehabSessionsLo
}

// filled checks session sj after it received it: the time, the
// straight-arm budget and the session caps, which it may exceed by the
// moved sets as with the minimum step, because the week load does not
// change.
func (g *gen) filled(sj int, it Item, moved map[int]map[string]float64) bool {
	add := map[string]float64{}
	for st, u := range moved[sj] {
		add[st] = u
	}
	for a, u := range g.itemLoad(it) {
		add[accountStructure(a)] += u
	}
	ps := &g.plan.Sessions[sj]
	if g.minutes(*ps) > float64(g.s.Profile.SessionMinutes)+eps {
		return false
	}
	for st, v := range g.sessionLoads(sj) {
		if v > math.Max(g.sessionCap(sj, st, g.targetSess[sj][st]), g.hist.sessionMax[st]+add[st])+eps {
			return false
		}
	}
	if b, n := g.budget(sj); float64(n) > b {
		return false
	}
	moved[sj] = add
	ps.EstMinutes = math.Round(g.minutes(*ps))
	return true
}

// blockOrder is the order of the blocks in a session (SESS-02).
var blockOrder = []string{BlockWarmup, BlockBalance, BlockMax, BlockVolume, BlockStrength, BlockEnd}

// insertItem appends it to the block of role in ps, adding the block in
// session order when ps has none, and returns the stored item.
func insertItem(ps *PlannedSession, role string, it Item, r Reason) *Item {
	for bi := range ps.Blocks {
		if ps.Blocks[bi].Role == role {
			ps.Blocks[bi].Items = append(ps.Blocks[bi].Items, it)
			return &ps.Blocks[bi].Items[len(ps.Blocks[bi].Items)-1]
		}
	}
	at := len(ps.Blocks)
	for bi, b := range ps.Blocks {
		if slices.Index(blockOrder, b.Role) > slices.Index(blockOrder, role) {
			at = bi
			break
		}
	}
	ps.Blocks = slices.Insert(ps.Blocks, at, Block{Role: role, Items: []Item{it}, Reasons: []Reason{r}})
	return &ps.Blocks[at].Items[0]
}

// ruleOfBlock is the rule that explains a block.
func ruleOfBlock(role string) string {
	switch role {
	case BlockBalance:
		return RuleBalance
	case BlockMax:
		return RuleMaxBlock
	case BlockVolume:
		return RuleVolume
	case BlockEnd:
		return RuleEndBlock
	case BlockWarmup:
		return RuleWarmup
	}
	return RuleStrength
}

func (g *gen) hasExercise(si int, slug string) bool {
	for _, b := range g.plan.Sessions[si].Blocks {
		for _, it := range b.Items {
			if it.Exercise == slug && it.Sets > 0 {
				return true
			}
		}
	}
	return false
}

func cloneSessions(in []PlannedSession) []PlannedSession {
	out := slices.Clone(in)
	for i := range out {
		out[i].Blocks = slices.Clone(out[i].Blocks)
		for j := range out[i].Blocks {
			out[i].Blocks[j].Items = slices.Clone(out[i].Blocks[j].Items)
			for n := range out[i].Blocks[j].Items {
				out[i].Blocks[j].Items[n].Reasons = slices.Clone(out[i].Blocks[j].Items[n].Reasons)
			}
		}
	}
	return out
}
