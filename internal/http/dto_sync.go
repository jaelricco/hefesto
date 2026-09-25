package http

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/jaelricco/hefesto/internal/store"
)

type syncSessionOut struct {
	ID               uuid.UUID  `json:"id"`
	StartedAt        time.Time  `json:"started_at"`
	EndedAt          *time.Time `json:"ended_at"`
	Timezone         string     `json:"timezone"`
	LocalDate        string     `json:"local_date"`
	Title            string     `json:"title"`
	Notes            string     `json:"notes"`
	PerceivedFatigue *int       `json:"perceived_fatigue"`
	BodyweightKg     *float64   `json:"bodyweight_kg"`
	Status           string     `json:"status"`
	IsRestDay        bool       `json:"is_rest_day"`
	TemplateID       *uuid.UUID `json:"template_id"`
	CompletedAt      *time.Time `json:"completed_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	ServerUpdatedAt  time.Time  `json:"server_updated_at"`
	ServerSeq        int64      `json:"server_seq"`
	DeletedAt        *time.Time `json:"deleted_at"`
}

type syncBlockOut struct {
	ID            uuid.UUID  `json:"id"`
	SessionID     uuid.UUID  `json:"session_id"`
	OrderIndex    int        `json:"order_index"`
	Kind          string     `json:"kind"`
	RoundsPlanned *int       `json:"rounds_planned"`
	RoundsDone    *int       `json:"rounds_done"`
	IntervalS     *int       `json:"interval_s"`
	Notes         string     `json:"notes"`
	UpdatedAt     time.Time  `json:"updated_at"`
	ServerSeq     int64      `json:"server_seq"`
	DeletedAt     *time.Time `json:"deleted_at"`
}

// syncSetOut is setOut with its session, version and tombstone.
type syncSetOut struct {
	setOut
	SessionID uuid.UUID  `json:"session_id"`
	ServerSeq int64      `json:"server_seq"`
	DeletedAt *time.Time `json:"deleted_at"`
}

type syncBodyweightOut struct {
	ID           uuid.UUID  `json:"id"`
	MeasuredAt   time.Time  `json:"measured_at"`
	LocalDate    string     `json:"local_date"`
	BodyweightKg float64    `json:"bodyweight_kg"`
	Note         string     `json:"note"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ServerSeq    int64      `json:"server_seq"`
	DeletedAt    *time.Time `json:"deleted_at"`
}

type syncPageOut struct {
	Cursor     int64               `json:"cursor"`
	HasMore    bool                `json:"has_more"`
	Sessions   []syncSessionOut    `json:"sessions"`
	Blocks     []syncBlockOut      `json:"blocks"`
	Sets       []syncSetOut        `json:"sets"`
	Bodyweight []syncBodyweightOut `json:"bodyweight"`
}

func syncPageFrom(p store.SyncPage) syncPageOut {
	out := syncPageOut{
		Cursor: p.Cursor, HasMore: p.HasMore,
		Sessions: make([]syncSessionOut, len(p.Sessions)), Blocks: make([]syncBlockOut, len(p.Blocks)),
		Sets: make([]syncSetOut, len(p.Sets)), Bodyweight: make([]syncBodyweightOut, len(p.Bodyweight)),
	}
	for i, s := range p.Sessions {
		out.Sessions[i] = syncSessionOut{
			ID: s.ID, StartedAt: utc(s.StartedAt), EndedAt: utcPtr(s.EndedAt), Timezone: s.Timezone,
			LocalDate: s.LocalDate.Format(dateLayout), Title: s.Title, Notes: s.Notes,
			PerceivedFatigue: s.PerceivedFatigue, BodyweightKg: s.BodyweightKg, Status: s.Status,
			IsRestDay: s.IsRestDay, TemplateID: s.TemplateID, CompletedAt: utcPtr(s.CompletedAt),
			UpdatedAt: utc(s.UpdatedAt), ServerUpdatedAt: utc(s.ServerUpdatedAt), ServerSeq: s.ServerSeq,
			DeletedAt: utcPtr(s.DeletedAt),
		}
	}
	for i, b := range p.Blocks {
		out.Blocks[i] = syncBlockOut{
			ID: b.ID, SessionID: b.SessionID, OrderIndex: b.OrderIndex, Kind: b.Kind, RoundsPlanned: b.RoundsPlanned,
			RoundsDone: b.RoundsDone, IntervalS: b.IntervalS, Notes: b.Notes, UpdatedAt: utc(b.UpdatedAt),
			ServerSeq: b.ServerSeq, DeletedAt: utcPtr(b.DeletedAt),
		}
	}
	for i, s := range p.Sets {
		out.Sets[i] = syncSetOut{setOut: setFrom(s.SetEntry), SessionID: s.SessionID, ServerSeq: s.ServerSeq, DeletedAt: utcPtr(s.DeletedAt)}
	}
	for i, b := range p.Bodyweight {
		out.Bodyweight[i] = syncBodyweightOut{
			ID: b.ID, MeasuredAt: utc(b.MeasuredAt), LocalDate: b.LocalDate.Format(dateLayout),
			BodyweightKg: b.BodyweightKg, Note: b.Note, UpdatedAt: utc(b.UpdatedAt), ServerSeq: b.ServerSeq,
			DeletedAt: utcPtr(b.DeletedAt),
		}
	}
	return out
}

type syncOpIn struct {
	Op        string          `json:"op"`
	Entity    string          `json:"entity"`
	ID        uuid.UUID       `json:"id"`
	SessionID *uuid.UUID      `json:"session_id"`
	Data      json.RawMessage `json:"data"`
}

// sessionPutIn is a session's full client-side state. Optional fields tell
// an update which ones the client sent.
type sessionPutIn struct {
	StartedAt        time.Time           `json:"started_at"`
	EndedAt          Optional[time.Time] `json:"ended_at"`
	Timezone         string              `json:"timezone"`
	Title            Optional[string]    `json:"title"`
	Notes            Optional[string]    `json:"notes"`
	PerceivedFatigue Optional[int]       `json:"perceived_fatigue"`
	BodyweightKg     Optional[float64]   `json:"bodyweight_kg"`
	IsRestDay        Optional[bool]      `json:"is_rest_day"`
	TemplateID       *uuid.UUID          `json:"template_id"`
	Status           Optional[string]    `json:"status"`
	UpdatedAt        *time.Time          `json:"updated_at"`
}

type bodyweightIn struct {
	MeasuredAt   time.Time  `json:"measured_at"`
	Timezone     string     `json:"timezone"`
	BodyweightKg float64    `json:"bodyweight_kg"`
	Note         string     `json:"note"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

type syncOpResultOut struct {
	Index      int            `json:"index"`
	Status     string         `json:"status"`
	Problem    *Problem       `json:"problem,omitempty"`
	Completion *completionOut `json:"completion,omitempty"`
}

type syncPushOut struct {
	Results []syncOpResultOut `json:"results"`
}
