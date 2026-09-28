package planning

import "fmt"

// Tuning holds every number the planner uses, resolved once from the
// parameter catalogue. Code reads these fields, never literals and never the
// catalogue map, so a parameter the code needs but the catalogue lacks fails
// validation (KB-03). The comment on each field names its parameter.
type Tuning struct {
	// Dosing of holds (spec §5.7, 08 §4).
	SetHoldFraction        float64 // PAR-S-01
	TargetTotalHold        float64 // PAR-S-02
	MinSetHold             float64 // PAR-B-07 min
	MinSets, MaxSets       float64 // PAR-B-08 range
	DefaultMaxSets         float64 // PAR-B-08 standard
	VolumeHoldMin          float64 // PAR-B-10
	VolumeHoldMax          float64 // PAR-B-10
	VolumeSetsMin          float64 // PAR-B-11
	VolumeSetsMax          float64 // PAR-B-11
	CondHoldMin            float64 // PAR-B-76
	CondHoldMax            float64 // PAR-B-76
	CondMaxTotal           float64 // PAR-B-76
	CondSetsLo, CondSetsHi float64 // PAR-B-76
	StageOffer             float64 // PAR-B-05 upper
	StageSwitch            float64 // PAR-A-65
	HoldStepPerWeek        float64 // PAR-B-33 upper

	// Rest (08 §4, PAR-E-04–07, PAR-B-42–45, PAR-B-74, PAR-B-77).
	RestMax             float64 // PAR-E-04 default
	RestMaxFloor        float64 // PAR-E-04 lower bound
	RestHeaviest        float64 // PAR-E-05
	HeaviestOG          float64 // PAR-S-05
	RestVolume          float64 // PAR-E-06 default
	RestVolumeFloor     float64 // PAR-E-06 lower bound
	RestAccessory       float64 // PAR-E-07 lower bound
	RestStrengthNovice  float64 // PAR-B-42 novice lower bound (PAR-S-29)
	RestStrengthTrained float64 // PAR-B-42 trained lower bound (PAR-S-29)
	RestLight           float64 // PAR-B-43 default
	RestPrehab          float64 // PAR-S-36 within PAR-B-44
	RestPaired          float64 // PAR-B-45
	RestSkillPractice   float64 // PAR-B-74 upper
	RestEccentric       float64 // PAR-B-77
	RestRound           float64 // PAR-S-18

	// Dynamic dosing.
	RIRStrength        float64 // PAR-B-24 standard
	NoviceRepsLo       float64 // PAR-A-02
	NoviceRepsHi       float64 // PAR-A-02 / PAR-A-03
	NoviceRestart      float64 // PAR-A-04
	NoviceSets         float64 // PAR-A-01
	HeavyLo, HeavyHi   float64 // PAR-B-17
	MediumLo, MediumHi float64 // PAR-S-13
	LightLo, LightHi   float64 // PAR-B-19 / PAR-S-13
	StrengthSets       float64 // PAR-B-18 standard
	RepToLoad          float64 // PAR-B-23 lower
	LoadIncrementPct   float64 // PAR-B-32 standard
	SmallestPlate      float64 // PAR-S-37
	NextRungMinDose    float64 // PAR-S-33
	EccentricStart     float64 // PAR-B-16 lower
	EccentricTarget    float64 // PAR-B-16 target lower
	EccentricStep      float64 // PAR-S-37
	EccentricReps      float64 // PAR-B-77 clusters
	EccentricSets      float64 // PAR-B-77 lower
	AccessorySets      float64 // PAR-S-36
	AccessoryReps      float64 // PAR-B-78 lower
	PrehabSets         float64 // PAR-S-36
	PrehabReps         float64 // PAR-S-36
	TechniqueFrac      float64 // PAR-S-26
	TechniqueMax       float64 // PAR-S-26
	TechniqueTries     float64 // PAR-S-36
	ProbeAttempts      float64 // PAR-S-24
	ProbeHold          float64 // PAR-S-24
	BalanceMin         float64 // PAR-E-35 point value (PAR-S-36)
	BalanceShortMin    float64 // 08 §4 short form point value (PAR-S-36)
	BalanceFloorMin    float64 // 08 §4 lower bound of the short form
	BalanceSetS        float64 // PAR-E-36 point value
	BalanceShortUnder  float64 // 08 §4: sessions under 45 min

	// Week structure (spec §5.4).
	FreqNovice       float64 // PAR-B-34
	FreqTrained      float64 // PAR-B-34
	FreqMax          float64 // PAR-B-34
	FreqBalance      float64 // PAR-E-12
	FullNovice       float64 // PAR-B-36
	FullIntermediate float64 // PAR-B-36
	FullAdvanced     float64 // PAR-B-36
	SplitAt          float64 // PAR-B-37
	MesoWeeks        float64 // PAR-B-03 standard
	MaxBuildWeeks    float64 // PAR-B-49 d
	DeloadSets       float64 // PAR-B-52
	DeloadReserve    float64 // PAR-B-53
	FatigueHigh      float64 // PAR-B-49 b
	FatigueCount     float64 // PAR-B-49 b
	FatigueWindow    float64 // PAR-B-49 b

	// Load management (spec §7).
	CapStraight                        float64 // PAR-D-09
	CapBent                            float64 // PAR-D-10
	CapWrist                           float64 // PAR-D-11
	NewTypeFraction                    float64 // PAR-D-12
	SpikeCap                           float64 // PAR-D-31
	PriorInjury                        float64 // PAR-D-02
	RiskWindow                         float64 // PAR-S-25
	RiskWindowLo                       float64 // PAR-D-04 lower (months)
	RiskWindowHi                       float64 // PAR-D-04 upper (months)
	MinorAge                           float64 // PAR-D-23
	SpacingHard                        float64 // PAR-D-08
	SpacingRamp                        float64 // PAR-D-34
	SpacingModerate                    float64 // PAR-B-38 submaximal
	SpacingRating                      float64 // PAR-S-07
	BudgetBeginner                     float64 // PAR-S-23
	BudgetIntermediate                 float64 // PAR-S-23
	BudgetAdvanced                     float64 // PAR-S-23
	IntermediateOG                     float64 // PAR-A-23 band lower bounds
	AdvancedOG                         float64 // PAR-A-23
	Priority1, Priority2, Priority3    float64 // PAR-S-09
	MinMaxBlockSets                    float64 // PAR-S-09
	EntryStep1, EntryStep2, EntryStep3 float64 // PAR-S-43
	EntrySpike                         float64 // PAR-S-43
	WindowWeeks                        float64 // PAR-B-73
	RefWeeks                           float64 // PAR-S-14
	SpikeDays                          float64 // PAR-D-31 window
	NewRungWeeks                       float64 // PAR-D-06
	SupinatedMinOG                     float64 // PAR-A-23 intermediate
	WarmupWeight                       float64 // PAR-S-08

	// Ramps and pain (spec §8).
	PainGreen                                  float64 // PAR-D-14
	PainAccept                                 float64 // PAR-D-15
	PainDeloadVol                              float64 // PAR-D-18 (1 − 0.30)
	PainDeloadDays                             float64 // PAR-D-18
	RestNextDay                                float64 // PAR-D-28
	RestWarmup                                 float64 // PAR-D-28
	RTTStart                                   float64 // PAR-D-24
	RTTStartReferral                           float64 // PAR-D-33
	RampStep1, RampStep2, RampStep3, RampStep4 float64 // PAR-D-25
	RampMinDays                                float64 // PAR-D-26
	RampSessions                               float64 // PAR-D-26
	LayoffDays                                 float64 // PAR-D-29 (weeks × 7)
	ReferralSoft                               float64 // PAR-D-32
	ReferralDays                               float64 // PAR-D-19
	BreachCount                                float64 // PAR-D-20
	BreachWindow                               float64 // PAR-D-20
	RTT5Weeks                                  float64 // PAR-S-32
	PainEntryDays                              float64 // PAR-S-42
	PrehabSessions                             float64 // PAR-D-37
	PlausibleFrac                              float64 // PAR-S-46
	PainTrendPoints                            float64 // PAR-S-42

	// Breaks (spec §6.11).
	BreakShort, BreakMid, BreakMonth, BreakQuarter, BreakHalf             float64 // day thresholds
	BreakFactorMid, BreakFactorMonth, BreakFactorQuarter, BreakFactorHalf float64 // PAR-B-59–62
	BreakGrowth                                                           float64 // PAR-B-60 lower

	// Estimates (spec §4.3, §4.8).
	ConfHigh         float64 // PAR-F-30
	ConfLow          float64 // PAR-F-30
	DoseOffsetMid    float64 // PAR-F-31
	DoseOffsetLow    float64 // PAR-F-31
	RepObsSD         float64 // PAR-F-22
	LowRepObsSD      float64 // PAR-F-02
	LowRepMax        float64 // PAR-F-02 (≤ 5 reps)
	RIRMaxUsable     float64 // PAR-F-24
	MaxRepsFull      float64 // PAR-F-23
	ContradictionSD  float64 // PAR-F-32
	ProcessReps      float64 // PAR-F-28
	ProcessHoldFrac  float64 // PAR-S-04
	ObsSkillStatic   float64 // PAR-S-03
	ObsBalance       float64 // PAR-F-16
	ObsCore          float64 // PAR-F-16
	ObsEndurance     float64 // PAR-F-16
	SelfReportFrac   float64 // PAR-F-20
	SelfReportMin    float64 // PAR-F-20
	RecallBias       float64 // PAR-F-55
	CountedFactor    float64 // PAR-F-21
	DerivedFrac      float64 // PAR-F-26
	Widening         float64 // PAR-S-44
	UnknownFrac      float64 // PAR-S-45
	MinHoldSD        float64 // PAR-S-38
	MinRepSD         float64 // PAR-S-38
	MinObsSD         float64 // PAR-S-38
	MinFormEvidence  float64 // PAR-F-41
	SIRUsableAbs     float64 // PAR-S-31
	SIRUsableFrac    float64 // PAR-S-31
	CalibrationWeeks float64 // PAR-B-29 upper interval
	RIRTrustWeeks    float64 // PAR-B-29

	// Adaptation (spec §6).
	PlateauMinWeeks  float64 // PAR-S-17
	PlateauDeloadGap float64 // PAR-S-17
	BalanceReview    float64 // PAR-E-37
	AutoregDelta     float64 // PAR-B-31
	SleepShort       float64 // PAR-E-41
	CheckinFatigue   float64 // PAR-S-28
	FormDrop         float64 // PAR-E-15
	FormMin          float64 // PAR-E-15
	FailStop         float64 // PAR-E-16
	PerfDrop         float64 // PAR-E-17
	RecommendedMin   float64 // PAR-S-37

	// Time model (spec §5.8).
	// Realism check (spec §3.6).
	RealLo4, RealHi4, RealLo8, RealHi8, RealLo12, RealHi12, RealLoTop, RealHiTop float64 // PAR-A-45

	SecondsPerRep      float64 // PAR-S-11
	TransitionS        float64 // PAR-S-11
	MinWarmup          float64 // PAR-B-68
	GeneralWarmupShort float64 // PAR-S-37
}

type tuningRef struct {
	dst *float64
	id  string
	key string // "" selects Value
}

// resolveTuning fills Tuning from the catalogue and reports every missing
// parameter (KB-03).
func resolveTuning(k *Knowledge, issues []Issue) (Tuning, []Issue) {
	var t Tuning
	refs := []tuningRef{
		{&t.SetHoldFraction, "PAR-S-01", ""},
		{&t.TargetTotalHold, "PAR-S-02", ""},
		{&t.MinSetHold, "PAR-B-07", "min_s"},
		{&t.MinSets, "PAR-B-08", "lo"},
		{&t.MaxSets, "PAR-B-08", "hi"},
		{&t.DefaultMaxSets, "PAR-B-08", "standard"},
		{&t.VolumeHoldMin, "PAR-B-10", "lo"},
		{&t.VolumeHoldMax, "PAR-B-10", "hi"},
		{&t.VolumeSetsMin, "PAR-B-11", "lo"},
		{&t.VolumeSetsMax, "PAR-B-11", "hi"},
		{&t.CondHoldMin, "PAR-B-76", "hold_lo"},
		{&t.CondHoldMax, "PAR-B-76", "hold_hi"},
		{&t.CondMaxTotal, "PAR-B-76", "total_hi"},
		{&t.CondSetsLo, "PAR-B-76", "sets_lo"},
		{&t.CondSetsHi, "PAR-B-76", "sets_hi"},
		{&t.StageOffer, "PAR-B-05", "hi"},
		{&t.StageSwitch, "PAR-A-65", ""},
		{&t.HoldStepPerWeek, "PAR-B-33", "hi"},

		{&t.RestMax, "PAR-E-04", "default"},
		{&t.RestMaxFloor, "PAR-E-04", "floor"},
		{&t.RestHeaviest, "PAR-E-05", ""},
		{&t.HeaviestOG, "PAR-S-05", ""},
		{&t.RestVolume, "PAR-E-06", "default"},
		{&t.RestVolumeFloor, "PAR-E-06", "lo"},
		{&t.RestAccessory, "PAR-E-07", "lo"},
		{&t.RestStrengthNovice, "PAR-B-42", "novice_lo"},
		{&t.RestStrengthTrained, "PAR-B-42", "trained_lo"},
		{&t.RestLight, "PAR-B-43", "standard"},
		{&t.RestPrehab, "PAR-S-36", "rest_prehab"},
		{&t.RestPaired, "PAR-B-45", ""},
		{&t.RestSkillPractice, "PAR-B-74", "hi"},
		{&t.RestEccentric, "PAR-B-77", "rest"},
		{&t.RestRound, "PAR-S-18", "rest_round"},

		{&t.RIRStrength, "PAR-B-24", "standard"},
		{&t.NoviceRepsLo, "PAR-A-02", "lo"},
		{&t.NoviceRepsHi, "PAR-A-02", "hi"},
		{&t.NoviceRestart, "PAR-A-04", ""},
		{&t.NoviceSets, "PAR-A-01", ""},
		{&t.HeavyLo, "PAR-B-17", "standard_lo"},
		{&t.HeavyHi, "PAR-B-17", "standard_hi"},
		{&t.MediumLo, "PAR-S-13", "medium_lo"},
		{&t.MediumHi, "PAR-S-13", "medium_hi"},
		{&t.LightLo, "PAR-S-13", "light_lo"},
		{&t.LightHi, "PAR-S-13", "light_hi"},
		{&t.StrengthSets, "PAR-B-18", "standard"},
		{&t.RepToLoad, "PAR-B-23", "lo"},
		{&t.LoadIncrementPct, "PAR-B-32", "standard"},
		{&t.SmallestPlate, "PAR-S-37", "smallest_plate_kg"},
		{&t.NextRungMinDose, "PAR-S-33", ""},
		{&t.EccentricStart, "PAR-B-16", "start_lo"},
		{&t.EccentricTarget, "PAR-B-16", "target_lo"},
		{&t.EccentricStep, "PAR-S-37", "eccentric_step_s"},
		{&t.EccentricReps, "PAR-B-77", "clusters_hi"},
		{&t.EccentricSets, "PAR-B-77", "sets_lo"},
		{&t.AccessorySets, "PAR-S-36", "accessory_sets"},
		{&t.AccessoryReps, "PAR-B-78", "reps_lo"},
		{&t.PrehabSets, "PAR-S-36", "prehab_sets"},
		{&t.PrehabReps, "PAR-S-36", "prehab_reps"},
		{&t.TechniqueFrac, "PAR-S-26", "fraction"},
		{&t.TechniqueMax, "PAR-S-26", "max_s"},
		{&t.TechniqueTries, "PAR-S-36", "technique_tries"},
		{&t.ProbeAttempts, "PAR-S-24", "attempts"},
		{&t.ProbeHold, "PAR-S-24", "hold_s"},
		{&t.BalanceMin, "PAR-S-36", "balance_min"},
		{&t.BalanceShortMin, "PAR-S-36", "balance_short_min"},
		{&t.BalanceFloorMin, "PAR-S-36", "balance_floor_min"},
		{&t.BalanceSetS, "PAR-S-36", "balance_set_s"},
		{&t.BalanceShortUnder, "PAR-S-36", "balance_short_under_min"},

		{&t.FreqNovice, "PAR-B-34", "novice"},
		{&t.FreqTrained, "PAR-B-34", "trained"},
		{&t.FreqMax, "PAR-B-34", "max"},
		{&t.FreqBalance, "PAR-E-12", "default"},
		{&t.FullNovice, "PAR-B-36", "novice_hi"},
		{&t.FullIntermediate, "PAR-B-36", "intermediate_hi"},
		{&t.FullAdvanced, "PAR-B-36", "advanced_hi"},
		{&t.SplitAt, "PAR-B-37", ""},
		{&t.MesoWeeks, "PAR-B-03", "standard"},
		{&t.MaxBuildWeeks, "PAR-B-49", "max_build_weeks"},
		{&t.DeloadSets, "PAR-B-52", ""},
		{&t.DeloadReserve, "PAR-B-53", ""},
		{&t.FatigueHigh, "PAR-B-49", "fatigue"},
		{&t.FatigueCount, "PAR-B-49", "fatigue_count"},
		{&t.FatigueWindow, "PAR-B-49", "fatigue_window"},

		{&t.CapStraight, "PAR-D-09", ""},
		{&t.CapBent, "PAR-D-10", ""},
		{&t.CapWrist, "PAR-D-11", ""},
		{&t.NewTypeFraction, "PAR-D-12", ""},
		{&t.SpikeCap, "PAR-D-31", ""},
		{&t.PriorInjury, "PAR-D-02", ""},
		{&t.RiskWindow, "PAR-S-25", ""},
		{&t.RiskWindowLo, "PAR-D-04", "lo"},
		{&t.RiskWindowHi, "PAR-D-04", "hi"},
		{&t.MinorAge, "PAR-D-23", ""},
		{&t.SpacingHard, "PAR-D-08", ""},
		{&t.SpacingRamp, "PAR-D-34", ""},
		{&t.SpacingModerate, "PAR-B-38", "submax"},
		{&t.SpacingRating, "PAR-S-07", ""},
		{&t.BudgetBeginner, "PAR-S-23", "beginner"},
		{&t.BudgetIntermediate, "PAR-S-23", "intermediate"},
		{&t.BudgetAdvanced, "PAR-S-23", "advanced"},
		{&t.IntermediateOG, "PAR-A-23", "intermediate_lo"},
		{&t.AdvancedOG, "PAR-A-23", "advanced_lo"},
		{&t.Priority1, "PAR-S-09", "p1"},
		{&t.Priority2, "PAR-S-09", "p2"},
		{&t.Priority3, "PAR-S-09", "p3"},
		{&t.MinMaxBlockSets, "PAR-S-09", "min_sets"},
		{&t.EntryStep1, "PAR-S-43", "step1"},
		{&t.EntryStep2, "PAR-S-43", "step2"},
		{&t.EntryStep3, "PAR-S-43", "step3"},
		{&t.EntrySpike, "PAR-S-43", "spike"},
		{&t.WindowWeeks, "PAR-B-73", ""},
		{&t.RefWeeks, "PAR-S-14", ""},
		{&t.SpikeDays, "PAR-D-31", "window_days"},
		{&t.NewRungWeeks, "PAR-D-06", ""},
		{&t.SupinatedMinOG, "PAR-A-23", "intermediate_lo"},
		{&t.WarmupWeight, "PAR-S-08", ""},

		{&t.PainGreen, "PAR-D-14", ""},
		{&t.PainAccept, "PAR-D-15", ""},
		{&t.PainDeloadVol, "PAR-D-18", "volume_factor"},
		{&t.PainDeloadDays, "PAR-D-18", "days"},
		{&t.RestNextDay, "PAR-D-28", "rest_days_next_day"},
		{&t.RestWarmup, "PAR-D-28", "rest_days_warmup"},
		{&t.RTTStart, "PAR-D-24", ""},
		{&t.RTTStartReferral, "PAR-D-33", ""},
		{&t.RampStep1, "PAR-D-25", "s1"},
		{&t.RampStep2, "PAR-D-25", "s2"},
		{&t.RampStep3, "PAR-D-25", "s3"},
		{&t.RampStep4, "PAR-D-25", "s4"},
		{&t.RampMinDays, "PAR-D-26", "min_days"},
		{&t.RampSessions, "PAR-D-26", "sessions"},
		{&t.LayoffDays, "PAR-D-29", "days"},
		{&t.ReferralSoft, "PAR-D-32", ""},
		{&t.ReferralDays, "PAR-D-19", ""},
		{&t.BreachCount, "PAR-D-20", "count"},
		{&t.BreachWindow, "PAR-D-20", "days"},
		{&t.RTT5Weeks, "PAR-S-32", ""},
		{&t.PainEntryDays, "PAR-S-42", "entry_days"},
		{&t.PainTrendPoints, "PAR-S-42", "trend_points"},
		{&t.PrehabSessions, "PAR-D-37", ""},
		{&t.PlausibleFrac, "PAR-S-46", ""},

		{&t.BreakShort, "PAR-S-41", "normal_below_days"},
		{&t.BreakMid, "PAR-S-41", "mid_from_days"},
		{&t.BreakMonth, "PAR-S-41", "month_from_days"},
		{&t.BreakQuarter, "PAR-S-41", "quarter_from_days"},
		{&t.BreakHalf, "PAR-S-41", "half_from_days"},
		{&t.BreakFactorMid, "PAR-B-59", "lo"},
		{&t.BreakFactorMonth, "PAR-B-60", "week1_lo"},
		{&t.BreakFactorQuarter, "PAR-B-61", "week1"},
		{&t.BreakFactorHalf, "PAR-B-62", "week1"},
		{&t.BreakGrowth, "PAR-B-60", "growth_lo"},

		{&t.ConfHigh, "PAR-F-30", "high"},
		{&t.ConfLow, "PAR-F-30", "low"},
		{&t.DoseOffsetMid, "PAR-F-31", "mid"},
		{&t.DoseOffsetLow, "PAR-F-31", "low"},
		{&t.RepObsSD, "PAR-F-22", ""},
		{&t.LowRepObsSD, "PAR-F-02", ""},
		{&t.LowRepMax, "PAR-F-02", "max_reps"},
		{&t.RIRMaxUsable, "PAR-F-24", ""},
		{&t.MaxRepsFull, "PAR-F-23", "max_reps"},
		{&t.ContradictionSD, "PAR-F-32", ""},
		{&t.ProcessReps, "PAR-F-28", ""},
		{&t.ProcessHoldFrac, "PAR-S-04", ""},
		{&t.ObsSkillStatic, "PAR-S-03", ""},
		{&t.ObsBalance, "PAR-F-16", "balance"},
		{&t.ObsCore, "PAR-F-16", "core"},
		{&t.ObsEndurance, "PAR-F-16", "endurance"},
		{&t.SelfReportFrac, "PAR-F-20", "frac"},
		{&t.SelfReportMin, "PAR-F-20", "min"},
		{&t.RecallBias, "PAR-F-55", ""},
		{&t.CountedFactor, "PAR-F-21", ""},
		{&t.DerivedFrac, "PAR-F-26", ""},
		{&t.Widening, "PAR-S-44", ""},
		{&t.UnknownFrac, "PAR-S-45", ""},
		{&t.MinHoldSD, "PAR-S-38", "hold_sd"},
		{&t.MinRepSD, "PAR-S-38", "rep_sd"},
		{&t.MinObsSD, "PAR-S-38", "obs_sd"},
		{&t.MinFormEvidence, "PAR-F-41", ""},
		{&t.SIRUsableAbs, "PAR-S-31", "abs_s"},
		{&t.SIRUsableFrac, "PAR-S-31", "frac"},
		{&t.CalibrationWeeks, "PAR-B-29", "calibrate_every_hi"},
		{&t.RIRTrustWeeks, "PAR-B-29", "trust_after"},

		{&t.PlateauMinWeeks, "PAR-S-17", "min_weeks"},
		{&t.PlateauDeloadGap, "PAR-S-17", "deload_gap_weeks"},
		{&t.BalanceReview, "PAR-E-37", ""},
		{&t.AutoregDelta, "PAR-B-31", ""},
		{&t.SleepShort, "PAR-E-41", ""},
		{&t.CheckinFatigue, "PAR-S-28", ""},
		{&t.FormDrop, "PAR-E-15", "drop"},
		{&t.FormMin, "PAR-E-15", "min"},
		{&t.FailStop, "PAR-E-16", ""},
		{&t.PerfDrop, "PAR-E-17", ""},
		{&t.RecommendedMin, "PAR-S-37", "edge_weight"},

		{&t.RealLo4, "PAR-A-45", "le4_lo"},
		{&t.RealHi4, "PAR-A-45", "le4_hi"},
		{&t.RealLo8, "PAR-A-45", "le8_lo"},
		{&t.RealHi8, "PAR-A-45", "le8_hi"},
		{&t.RealLo12, "PAR-A-45", "le12_lo"},
		{&t.RealHi12, "PAR-A-45", "le12_hi"},
		{&t.RealLoTop, "PAR-A-45", "top_lo"},
		{&t.RealHiTop, "PAR-A-45", "top_hi"},

		{&t.SecondsPerRep, "PAR-S-11", "seconds_per_rep"},
		{&t.TransitionS, "PAR-S-11", "transition_s"},
		{&t.MinWarmup, "PAR-B-68", "min_warmup"},
		{&t.GeneralWarmupShort, "PAR-S-37", "general_warmup_short"},
	}
	for _, r := range refs {
		p, ok := k.params[r.id]
		if !ok {
			issues = append(issues, Issue{Check: "KB-03", Where: "tuning", Msg: fmt.Sprintf("parameter %s is used by the planner but missing", r.id)})
			continue
		}
		if r.key == "" {
			if p.Value == nil {
				issues = append(issues, Issue{Check: "KB-03", Where: r.id, Msg: "the planner needs a scalar value"})
				continue
			}
			*r.dst = *p.Value
			continue
		}
		v, ok := p.Values[r.key]
		if !ok {
			issues = append(issues, Issue{Check: "KB-03", Where: r.id, Msg: fmt.Sprintf("the planner needs values.%s", r.key)})
			continue
		}
		*r.dst = v
	}
	return t, issues
}
