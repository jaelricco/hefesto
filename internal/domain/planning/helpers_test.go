package planning_test

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jaelricco/hefesto/internal/content"
	"github.com/jaelricco/hefesto/internal/domain/planning"
)

var (
	kbOnce sync.Once
	kbVal  *planning.Knowledge
	kbErr  error
)

// kb loads the knowledge base from the repository once per test binary.
func kb(t testing.TB) *planning.Knowledge {
	t.Helper()
	kbOnce.Do(func() {
		k, issues, err := content.LoadTraining("../../../content")
		if err != nil {
			kbErr = err
			return
		}
		if content.HasErrors(issues, true) {
			kbErr = fmt.Errorf("knowledge base has issues: %v", issues)
			return
		}
		kbVal = k
	})
	if kbErr != nil {
		t.Fatal(kbErr)
	}
	return kbVal
}

// monday is the first plan week of every persona; now is Monday morning.
var (
	monday = time.Date(2026, time.September, 28, 0, 0, 0, 0, time.UTC)
	now    = monday.Add(7 * time.Hour)
)

// render prints a plan as compact text: one line per item, sessions and
// blocks as headers, then exclusions and hints. It is what the golden files
// and docs/algorithm/personas.md show.
func render(k *planning.Knowledge, p planning.Plan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "week %s  ruleset %s  stopped=%v", p.WeekStart.Format("2006-01-02"), p.RulesetVersion, p.Stopped)
	if p.Deload != "" {
		fmt.Fprintf(&b, "  deload=%s", p.Deload)
	}
	b.WriteString("\n")
	for _, r := range p.Reasons {
		fmt.Fprintf(&b, "  plan: [%s] %s\n", r.RuleID, r.Text)
	}
	for _, s := range p.Sessions {
		fmt.Fprintf(&b, "session %d %s %s ~%.0f min\n", s.Index+1, s.Date.Format("Mon 02.01."), s.Kind, s.EstMinutes)
		for _, bl := range s.Blocks {
			fmt.Fprintf(&b, "  %s", bl.Role)
			if bl.Minutes > 0 {
				fmt.Fprintf(&b, " (%g min)", bl.Minutes)
			}
			if bl.Paired {
				b.WriteString(" paired")
			}
			b.WriteString("\n")
			for _, it := range bl.Items {
				b.WriteString("    " + item(k, it) + "\n")
			}
		}
	}
	if len(p.RestDays) > 0 {
		var days []string
		for _, d := range p.RestDays {
			days = append(days, d.Format("Mon"))
		}
		fmt.Fprintf(&b, "rest: %s\n", strings.Join(days, " "))
	}
	for _, e := range p.Exclusions {
		fmt.Fprintf(&b, "excluded %s: [%s] %s\n", e.Exercise, e.Reason.RuleID, e.Reason.Text)
	}
	for _, h := range p.Hints {
		fmt.Fprintf(&b, "hint: [%s] %s\n", h.RuleID, h.Text)
	}
	for _, r := range p.Realism {
		fmt.Fprintf(&b, "realism %s/%s: lower %g, shown %g–%g weeks, date in %g weeks, unrealistic=%v",
			r.Skill, r.TargetLevel, r.LowerWeeks, r.ShownFrom, r.ShownTo, r.WeeksToDate, r.Unrealistic)
		if r.Milestone != "" {
			fmt.Fprintf(&b, ", milestone %s %g–%g weeks", r.Milestone, r.MilestoneFrom, r.MilestoneTo)
		}
		b.WriteString("\n")
	}
	for _, m := range p.Monitor {
		var ids []string
		for _, q := range m.Questions {
			ids = append(ids, q.ID)
		}
		fmt.Fprintf(&b, "monitor %s: %s\n", m.Region, strings.Join(ids, " "))
	}
	if len(p.Headroom) > 0 {
		keys := make([]string, 0, len(p.Headroom))
		for k := range p.Headroom {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var parts []string
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("%s=%.2f", k, p.Headroom[k]))
		}
		fmt.Fprintf(&b, "headroom: %s\n", strings.Join(parts, " "))
	}
	return b.String()
}

func item(k *planning.Knowledge, it planning.Item) string {
	name := it.Exercise
	if ex, ok := k.Exercise(it.Exercise); ok {
		name = ex.Name
	}
	var dose string
	switch {
	case it.HoldS > 0 && it.Reps > 0:
		dose = fmt.Sprintf("%d × %d à %d s", it.Sets, it.Reps, it.HoldS)
	case it.HoldS > 0:
		dose = fmt.Sprintf("%d × %d s", it.Sets, it.HoldS)
	case it.Reps > 0:
		dose = fmt.Sprintf("%d × %d", it.Sets, it.Reps)
	default:
		dose = fmt.Sprintf("%d sets", it.Sets)
	}
	var flags []string
	if it.Kind == planning.KindWarmup {
		flags = append(flags, "warm-up")
	}
	if it.Assist != "" {
		flags = append(flags, "assist="+it.Assist)
	}
	if it.LoadKg > 0 {
		flags = append(flags, fmt.Sprintf("+%g kg", it.LoadKg))
	}
	if it.Calibration {
		flags = append(flags, "calibration")
	}
	if it.Offer {
		flags = append(flags, "offer")
	}
	if it.Monitor {
		flags = append(flags, "monitor")
	}
	var rules []string
	for _, r := range it.Reasons {
		rules = append(rules, r.RuleID)
	}
	s := fmt.Sprintf("%-38s %-12s res %d  rest %3ds  %s p%d %s", name, dose, it.Reserve, it.RestS, it.Stimulus, it.Priority, it.Role)
	if len(flags) > 0 {
		s += "  [" + strings.Join(flags, ", ") + "]"
	}
	return s + "  {" + strings.Join(rules, " ") + "}"
}
