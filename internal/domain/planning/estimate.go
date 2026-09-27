package planning

import (
	"math"
	"time"
)

// Confidence classes (spec §4.8).
const (
	ConfHigh = "high"
	ConfMid  = "medium"
	ConfLow  = "low"
)

// Confidence returns the class of an estimate: [0, 0.15) high, [0.15, 0.30)
// medium, otherwise (or μ ≤ 0) low (PAR-F-30).
func (k *Knowledge) Confidence(e Estimate) string {
	if e.Mu <= 0 {
		return ConfLow
	}
	cv := e.Sigma / e.Mu
	switch {
	case cv < k.T.ConfHigh:
		return ConfHigh
	case cv < k.T.ConfLow:
		return ConfMid
	default:
		return ConfLow
	}
}

// Dose returns the value dosing uses: μ, μ − 0.5σ or μ − σ by confidence,
// never below 0 (PAR-F-31).
func (k *Knowledge) Dose(e Estimate) float64 {
	var d float64
	switch k.Confidence(e) {
	case ConfHigh:
		d = e.Mu
	case ConfMid:
		d = e.Mu - float64(k.T.DoseOffsetMid*e.Sigma)
	default:
		d = e.Mu - float64(k.T.DoseOffsetLow*e.Sigma)
	}
	return math.Max(0, d)
}

// obsFrac is the relative observation error of a hold (PAR-F-16, PAR-S-03).
func (k *Knowledge) obsFrac(e *Exercise) float64 {
	switch e.HoldClass {
	case HoldBalance:
		return k.T.ObsBalance
	case HoldCore:
		return k.T.ObsCore
	case HoldEndurance:
		return k.T.ObsEndurance
	default:
		return k.T.ObsSkillStatic
	}
}

// minSD is the floor of σ for an exercise (PAR-S-38).
func (k *Knowledge) minSD(e *Exercise) float64 {
	if e.Measure == MeasureHold {
		return k.T.MinHoldSD
	}
	return k.T.MinRepSD
}

// predict widens an estimate for the time since its last observation
// (spec §4.3 step 1): σ² += q² · Δweeks.
func (k *Knowledge) predict(e Estimate, ex *Exercise, now time.Time) Estimate {
	if e.At.IsZero() || !now.After(e.At) {
		return e
	}
	weeks := now.Sub(e.At).Hours() / (24 * 7)
	q := k.T.ProcessReps
	if ex.Measure == MeasureHold {
		q = float64(k.T.ProcessHoldFrac * e.Mu)
	}
	e.Sigma = math.Sqrt(float64(e.Sigma*e.Sigma) + float64(q*q)*weeks)
	return e
}

// Observation is one classified logged set (spec §4.3 step 2).
type Observation struct {
	X          float64
	R          float64
	LowerBound bool
	Test       bool
}

// classify turns a logged set into an observation for the unassisted or
// band capacity of its exercise, or reports false when the set is not
// evidence. The rows apply in order; the first that matches wins.
func (k *Knowledge) classify(set LoggedSet, ex *Exercise, first bool, mu float64) (Observation, bool) {
	if set.Kind == KindWarmup || set.Partial || set.Eccentric || ex.Eccentric {
		return Observation{}, false
	}
	if set.Form != nil && *set.Form < k.T.MinFormEvidence {
		return Observation{}, false
	}
	holdR := func(x float64) float64 {
		return math.Max(k.T.MinObsSD, float64(k.obsFrac(ex)*math.Max(x, mu)))
	}
	if !first {
		return Observation{X: set.Value, LowerBound: true}, true
	}
	if set.Kind == KindTest || set.Failed {
		o := Observation{X: set.Value, Test: set.Kind == KindTest}
		if ex.Measure == MeasureHold {
			o.R = holdR(set.Value)
		} else {
			o.R = k.T.RepObsSD
			if set.Value <= k.T.LowRepMax {
				o.R = k.T.LowRepObsSD
			}
		}
		return o, true
	}
	if set.Reserve == nil {
		return Observation{X: set.Value, LowerBound: true}, true
	}
	res := *set.Reserve
	if ex.Measure == MeasureReps {
		if res <= k.T.RIRMaxUsable {
			x := set.Value + res
			if set.Value <= k.T.MaxRepsFull {
				return Observation{X: x, R: k.T.RepObsSD}, true
			}
			return Observation{X: x, LowerBound: true}, true
		}
		return Observation{X: set.Value, LowerBound: true}, true
	}
	if res <= math.Max(k.T.SIRUsableAbs, float64(k.T.SIRUsableFrac*set.Value)) {
		x := set.Value + res
		return Observation{X: x, R: holdR(x)}, true
	}
	return Observation{X: set.Value, LowerBound: true}, true
}

// update applies one observation (spec §4.3 steps 3–4). It returns the new
// estimate and whether the observation contradicted the old one.
func (k *Knowledge) update(e Estimate, ex *Exercise, o Observation, at time.Time) (Estimate, bool) {
	if o.LowerBound {
		if o.X <= e.Mu {
			return e, false
		}
		o.R = k.T.RepObsSD
		if ex.Measure == MeasureHold {
			o.R = math.Max(k.T.MinObsSD, float64(k.obsFrac(ex)*o.X))
		}
	}
	o.R = math.Max(o.R, k.T.MinObsSD)
	floor := k.minSD(ex)
	gap := math.Sqrt(float64(e.Sigma*e.Sigma) + float64(o.R*o.R))
	if e.N > 0 && math.Abs(o.X-e.Mu) > float64(k.T.ContradictionSD*gap) {
		// ADAPT-03: never overwrite silently. A second deviation in the same
		// direction confirms it (PAR-S-21).
		if e.Pending != nil && math.Signbit(*e.Pending-e.Mu) == math.Signbit(o.X-e.Mu) {
			e.Mu = (*e.Pending + o.X) / 2
			e.Sigma = math.Max(o.R, floor)
			e.Pending = nil
			e.N++
			e.At, e.Origin = at, OriginLog
			return e, true
		}
		x := o.X
		e.Pending = &x
		e.Mu = math.Min(e.Mu, o.X)
		e.Sigma = math.Max(math.Max(e.Sigma, o.R), floor)
		e.At = at
		return e, true
	}
	v := float64(e.Sigma * e.Sigma)
	kGain := v / (v + float64(o.R*o.R))
	e.Mu += float64(kGain * (o.X - e.Mu))
	e.Sigma = math.Max(math.Sqrt(float64((1-kGain)*v)), floor)
	e.Pending = nil
	e.N++
	e.At = at
	if o.Test {
		e.Origin = OriginTest
	} else {
		e.Origin = OriginLog
	}
	return e, false
}

// selfReport returns the prior for a class answer (onboarding §5.2).
// confidence is estimated, counted_last_4_weeks or filmed.
func (k *Knowledge) selfReport(ex *Exercise, c AnswerClass, confidence string, at time.Time) Estimate {
	var mu float64
	switch {
	case c.Open:
		mu = c.Lo
	case c.Lo == 0 && c.Hi > 0 && ex.Measure == MeasureHold:
		mu = c.Hi / 2 // open-lower hold class (PAR-S-38)
	case ex.Measure == MeasureHold:
		mu = c.Lo
	default:
		mu = (c.Lo + c.Hi) / 2
	}
	if ex.Measure == MeasureReps && confidence != "filmed" && !c.Open {
		mu = float64(mu * (1 - k.T.RecallBias))
	}
	var sd float64
	switch {
	case confidence == "filmed":
		sd = k.T.RepObsSD
		if ex.Measure == MeasureHold {
			sd = float64(k.obsFrac(ex) * mu)
		}
	case ex.Measure == MeasureHold:
		sd = float64(k.T.SelfReportFrac * mu)
	default:
		sd = math.Max(k.T.SelfReportMin, float64(k.T.SelfReportFrac*mu))
		if confidence == "counted_last_4_weeks" {
			sd = float64(sd * k.T.CountedFactor)
		}
	}
	sd = math.Max(sd, k.minSD(ex))
	return Estimate{Mu: mu, Sigma: sd, Origin: OriginSelf, At: at}
}

// derivedPrior returns the prior for an easier rung or a band variant from a
// harder or unassisted one: the harder value is a lower bound (PAR-S-39).
func (k *Knowledge) derivedPrior(from Estimate, ex *Exercise, at time.Time) Estimate {
	mu := from.Mu
	sd := math.Max(k.minSD(ex), float64(k.T.DerivedFrac*mu))
	return Estimate{Mu: mu, Sigma: sd, Origin: OriginDerived, At: at}
}
