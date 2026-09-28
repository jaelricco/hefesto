package planning

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Reason explains one decision with the rule that made it, the parameters
// it used and their sources (spec §9.1).
type Reason struct {
	RuleID   string            `json:"rule_id"`
	Params   []string          `json:"params"`
	Sources  []string          `json:"sources"`
	Evidence string            `json:"evidence"`
	Text     string            `json:"text"`
	Args     map[string]string `json:"args,omitempty"`
	// Region is set when the reason concerns a region with a complaint or a
	// red flag; clients then show only the evidence level, never source
	// titles (EXPL-07).
	Region string `json:"region,omitempty"`
}

// Rule IDs the planner emits. Every one must exist in the catalogue (KB-03).
const (
	RuleOnboardingRequired = "SAFE-01"
	RuleStopped            = "SAFE-02"
	RuleScreening          = "SAFE-03"
	RuleNoConsent          = "SAFE-04"
	RuleRegionLocked       = "SAFE-05"
	RuleRegionRTT0         = "SAFE-06"
	RuleMinor              = "SAFE-07"

	RuleGoalOrder   = "GOAL-01"
	RuleGoalPath    = "GOAL-02"
	RuleMaintenance = "GOAL-03"
	RuleReadiness   = "GOAL-04"
	RuleRealism     = "GOAL-05"
	RuleAntagonist  = "GOAL-06"

	RuleDays        = "WEEK-01"
	RuleSessionKind = "WEEK-02"
	RuleSplit       = "WEEK-03"
	RuleFrequency   = "WEEK-04"
	RuleAllocation  = "WEEK-05"
	RuleTooFewDays  = "WEEK-06"
	RuleDeloadWeek  = "WEEK-07"

	RuleTemplate = "SESS-01"
	RuleOrder    = "SESS-02"
	RuleWarmup   = "SESS-03"
	RuleBalance  = "SESS-04"
	RuleMaxBlock = "SESS-05"
	RuleVolume   = "SESS-06"
	RuleStrength = "SESS-07"
	RuleEndBlock = "SESS-08"
	RuleLightDay = "SESS-09"

	RuleEquipment  = "SEL-02"
	RuleRegionCell = "SEL-03"
	RuleSupinated  = "SEL-04"
	RuleMobility   = "SEL-05"
	RuleHoldRung   = "SEL-07"
	RuleEntryRung  = "SEL-08"
	RuleRepRung    = "SEL-09"
	RuleSubstitute = "SEL-10"
	RulePlausible  = "SEL-12"

	RuleDoseMaxHold = "DOSE-01"
	RuleDoseMaxReps = "DOSE-02"
	RuleDoseVolume  = "DOSE-03"
	RuleDoseCond    = "DOSE-04"
	RuleDoseNovice  = "DOSE-05"
	RuleDoseTrained = "DOSE-06"
	RuleDoseEcc     = "DOSE-07"
	RuleDoseBalance = "DOSE-08"
	RuleDoseTech    = "DOSE-09"
	RuleDosePrehab  = "DOSE-10"
	RuleDoseLoad    = "DOSE-11"
	RuleDoseAccess  = "DOSE-12"
	RuleStopForm    = "DOSE-20"
	RuleStopFails   = "DOSE-21"
	RuleStopDrop    = "DOSE-22"
	RuleStopPain    = "DOSE-23"

	RuleLoadUnit   = "LOAD-01"
	RuleWeekCap    = "LOAD-02"
	RuleSessionCap = "LOAD-03"
	RuleNewLoad    = "LOAD-04"
	RuleEntryRamp  = "LOAD-04b"
	RuleSpacing    = "LOAD-05"
	RuleBudget     = "LOAD-06"
	RuleSameDir    = "LOAD-07"
	RulePairing    = "LOAD-08"
	RuleNewRung    = "LOAD-09"
	RuleTrim       = "LOAD-10"

	RuleCapacity   = "ADAPT-01"
	RuleContradict = "ADAPT-03"
	RuleHoldGrowth = "ADAPT-04"
	RuleProbe      = "ADAPT-05"
	RuleRungUp     = "ADAPT-06"
	RuleProbeGate  = "ADAPT-06a"
	RuleDoubleProg = "ADAPT-07"
	RuleUndulating = "ADAPT-08"
	RuleLoadProg   = "ADAPT-09"
	RuleEccToConc  = "ADAPT-10"
	RuleAutoreg    = "ADAPT-11"
	RuleRungDown   = "ADAPT-12"
	RulePlateau    = "ADAPT-13"
	RuleDeload     = "ADAPT-14"
	RuleMissed     = "ADAPT-15"
	RuleBreak      = "ADAPT-16"
	RuleCheckin    = "ADAPT-17"

	RuleRedFlagAsk  = "INJ-01"
	RuleRedFlagAct  = "INJ-02"
	RuleRegionState = "INJ-03"
	RuleMatrix      = "INJ-04"
	RuleModify      = "INJ-05"
	RuleRamp        = "INJ-06"
	RulePain        = "INJ-07"
	RuleReferral    = "INJ-08"
	RuleMinorRules  = "INJ-09"
	RuleRestriction = "INJ-10"
)

var requiredRules = []string{
	RuleOnboardingRequired, RuleStopped, RuleScreening, RuleNoConsent, RuleRegionLocked, RuleRegionRTT0, RuleMinor,
	RuleGoalOrder, RuleGoalPath, RuleMaintenance, RuleReadiness, RuleRealism, RuleAntagonist,
	RuleDays, RuleSessionKind, RuleSplit, RuleFrequency, RuleAllocation, RuleTooFewDays, RuleDeloadWeek,
	RuleTemplate, RuleOrder, RuleWarmup, RuleBalance, RuleMaxBlock, RuleVolume, RuleStrength, RuleEndBlock, RuleLightDay,
	RuleEquipment, RuleRegionCell, RuleSupinated, RuleMobility, RuleHoldRung, RuleEntryRung, RuleRepRung, RuleSubstitute, RulePlausible,
	RuleDoseMaxHold, RuleDoseMaxReps, RuleDoseVolume, RuleDoseCond, RuleDoseNovice, RuleDoseTrained, RuleDoseEcc,
	RuleDoseBalance, RuleDoseTech, RuleDosePrehab, RuleDoseLoad, RuleDoseAccess,
	RuleStopForm, RuleStopFails, RuleStopDrop, RuleStopPain,
	RuleLoadUnit, RuleWeekCap, RuleSessionCap, RuleNewLoad, RuleEntryRamp, RuleSpacing, RuleBudget, RuleSameDir,
	RulePairing, RuleNewRung, RuleTrim,
	RuleCapacity, RuleContradict, RuleHoldGrowth, RuleProbe, RuleRungUp, RuleProbeGate, RuleDoubleProg, RuleUndulating,
	RuleLoadProg, RuleEccToConc, RuleAutoreg, RuleRungDown, RulePlateau, RuleDeload, RuleMissed, RuleBreak, RuleCheckin,
	RuleRedFlagAsk, RuleRedFlagAct, RuleRegionState, RuleMatrix, RuleModify, RuleRamp, RulePain, RuleReferral,
	RuleMinorRules, RuleRestriction,
}

// reason builds a Reason for a rule. Arguments are key/value pairs; values
// are formatted deterministically. Placeholders {key} in the rule text are
// replaced. A missing rule is a validation error caught by Build, so here it
// only yields a reason with the bare ID.
func (k *Knowledge) reason(id string, kv ...any) Reason {
	r := Reason{RuleID: id}
	rule, ok := k.rules[id]
	if !ok {
		r.Text = id
		return r
	}
	r.Params = slices.Clone(rule.Params)
	r.Evidence = rule.Evidence
	src := map[string]bool{}
	for _, s := range rule.Sources {
		src[s] = true
	}
	for _, p := range rule.Params {
		for _, s := range k.params[p].Sources {
			src[s] = true
		}
	}
	r.Sources = sortedKeys(src)
	if len(kv) > 0 {
		r.Args = map[string]string{}
	}
	for i := 0; i+1 < len(kv); i += 2 {
		key, _ := kv[i].(string)
		r.Args[key] = formatArg(kv[i+1])
	}
	r.Text = render(rule.Text, r.Args)
	return r
}

// regional marks a reason as tied to a region with a complaint (EXPL-07).
func regional(r Reason, region string) Reason {
	r.Region = region
	return r
}

func formatArg(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case int:
		return strconv.Itoa(x)
	case float64:
		return formatNum(x)
	case bool:
		return strconv.FormatBool(x)
	default:
		return fmt.Sprint(x)
	}
}

// formatNum prints a number with at most one decimal, without a trailing
// ".0", so the same value always renders the same way.
func formatNum(x float64) string {
	s := strconv.FormatFloat(x, 'f', 1, 64)
	return strings.TrimSuffix(s, ".0")
}

func render(tpl string, args map[string]string) string {
	if len(args) == 0 {
		return tpl
	}
	keys := sortedKeys(args)
	out := tpl
	for _, k := range keys {
		out = strings.ReplaceAll(out, "{"+k+"}", args[k])
	}
	return out
}

// Forbidden reports the first forbidden phrase in a user-facing text
// (spec §9.3), or "".
func (k *Knowledge) Forbidden(text string) string { return forbiddenIn(k.forbidden, text) }
