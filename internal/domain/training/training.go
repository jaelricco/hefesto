package training

import (
	"time"

	"github.com/google/uuid"
)

// Measure is what a set element records.
type Measure string

// Measures.
const (
	MeasureReps        Measure = "reps"
	MeasureHoldSeconds Measure = "hold_seconds"
	MeasureDistanceM   Measure = "distance_m"
	MeasureNone        Measure = "none"
)

// AssistanceClass is the evaluator's view of an element: did help or added
// load change how hard it was? Derived, never supplied by a client.
type AssistanceClass string

// Assistance classes.
const (
	Unassisted AssistanceClass = "unassisted"
	Assisted   AssistanceClass = "assisted"
	Loaded     AssistanceClass = "loaded"
)

// AssistanceTypeBand is the one assistance type that names a band.
const AssistanceTypeBand = "band"

// Block kinds whose interval_s is meaningful.
const (
	BlockKindEMOM  = "emom"
	BlockKindAMRAP = "amrap"
)

// Session statuses.
const (
	StatusDraft     = "draft"
	StatusCompleted = "completed"
	StatusAbandoned = "abandoned"
)

// MaxElementsPerSet bounds a combo. The longest sensible combo is far
// shorter; the bound stops a malformed client from writing thousands.
const MaxElementsPerSet = 20

// Session is a logged workout.
type Session struct {
	ID               uuid.UUID
	StartedAt        time.Time
	EndedAt          *time.Time
	Timezone         string
	LocalDate        time.Time // a civil date; only Y-M-D are meaningful
	Title            string
	Notes            string
	PerceivedFatigue *int
	BodyweightKg     *float64
	Status           string
	IsRestDay        bool
	TemplateID       *uuid.UUID
	CompletedAt      *time.Time
	UpdatedAt        time.Time
	ServerUpdatedAt  time.Time
	Blocks           []Block
}

// Block groups set entries: straight sets, a superset, a circuit, ...
type Block struct {
	ID            uuid.UUID
	SessionID     uuid.UUID
	OrderIndex    int
	Kind          string
	RoundsPlanned *int
	RoundsDone    *int
	IntervalS     *int
	Notes         string
	UpdatedAt     time.Time
	Sets          []SetEntry
}

// SetEntry is one set: always one or more ordered elements.
type SetEntry struct {
	ID                uuid.UUID
	SessionID         uuid.UUID
	BlockID           uuid.UUID
	OrderIndex        int
	RoundIndex        *int
	Kind              string
	IsPlanned         bool
	RestAfterPlannedS *int
	RestAfterActualS  *int
	RPE               *float64
	RIR               *int
	CompletedAt       *time.Time
	Notes             string
	UpdatedAt         time.Time
	Elements          []Element
}

// Element is one exercise performed within a set. A plain set has one; a
// combo has several, in order, with no rest between them.
type Element struct {
	ID              uuid.UUID
	OrderIndex      int
	ExerciseID      uuid.UUID
	Measure         Measure
	Reps            *int
	HoldSeconds     *float64
	DistanceM       *float64
	Tempo           *string
	LoadKg          float64
	IsEccentricOnly bool
	IsPartialROM    bool
	ROMNote         *string
	FormQuality     *int
	Failed          bool
	AssistanceClass AssistanceClass
	Assistance      *Assistance
	// MediaIDs are attached images (a form check), in order.
	MediaIDs []uuid.UUID
}

// Assistance is help received on an element: a band, a partner, a machine,
// a counterweight... Its absence means the element was unassisted.
type Assistance struct {
	ID                uuid.UUID
	Type              string
	BandID            *uuid.UUID
	BandCount         int
	Anchor            *string
	EstimatedAssistKg *float64
	Note              string
}

// ClassOf derives an element's assistance class. Assistance wins over load:
// a band-assisted weighted pull-up is still assisted, and an unlock that
// demands "no assistance" must not count it.
func ClassOf(e Element) AssistanceClass {
	switch {
	case e.Assistance != nil:
		return Assisted
	case e.LoadKg > 0:
		return Loaded
	default:
		return Unassisted
	}
}

// LocalDate is the athlete's calendar day for an instant in a time zone. It
// is stored with the session so a session stays on its day even if the
// athlete later moves time zones (DDL §1.2).
func LocalDate(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// HasValue reports whether the element records a value for its measure.
func (e Element) HasValue() bool {
	switch e.Measure {
	case MeasureReps:
		return e.Reps != nil
	case MeasureHoldSeconds:
		return e.HoldSeconds != nil
	case MeasureDistanceM:
		return e.DistanceM != nil
	default:
		return true
	}
}
