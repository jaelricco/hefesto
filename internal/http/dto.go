package http

import (
	"time"

	"github.com/google/uuid"

	"github.com/jaelricco/hefesto/internal/auth"
	"github.com/jaelricco/hefesto/internal/domain/training"
	"github.com/jaelricco/hefesto/internal/store"
)

// The types in this file mirror api/openapi.yaml, which is the source of
// truth. Response fields the spec marks required-but-nullable carry no
// omitempty: they are always present, as null when empty.

const dateLayout = "2006-01-02"

func utc(t time.Time) time.Time { return t.UTC() }

func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}

// ------------------------------------------------------------------- auth

type deviceIn struct {
	ID         uuid.UUID `json:"id"`
	Platform   string    `json:"platform"`
	Model      *string   `json:"model"`
	OSVersion  *string   `json:"os_version"`
	AppVersion *string   `json:"app_version"`
}

func (d *deviceIn) toStore() *store.Device {
	if d == nil {
		return nil
	}
	return &store.Device{ID: d.ID, Platform: d.Platform, Model: d.Model, OSVersion: d.OSVersion, AppVersion: d.AppVersion}
}

type registerIn struct {
	Email       string    `json:"email"`
	Password    string    `json:"password"`
	DisplayName string    `json:"display_name"`
	Locale      string    `json:"locale"`
	Timezone    string    `json:"timezone"`
	Device      *deviceIn `json:"device"`
}

type loginIn struct {
	Email    string    `json:"email"`
	Password string    `json:"password"`
	Device   *deviceIn `json:"device"`
}

type appleIn struct {
	IdentityToken string    `json:"identity_token"`
	Nonce         string    `json:"nonce"`
	DisplayName   string    `json:"display_name"`
	Device        *deviceIn `json:"device"`
}

type refreshIn struct {
	RefreshToken string `json:"refresh_token"`
}

type authOut struct {
	AccessToken       string    `json:"access_token"`
	TokenType         string    `json:"token_type"`
	ExpiresIn         int       `json:"expires_in"`
	RefreshToken      string    `json:"refresh_token"`
	RefreshExpiresAt  time.Time `json:"refresh_expires_at"`
	User              userOut   `json:"user"`
	DeletionCancelled bool      `json:"deletion_cancelled,omitempty"`
}

func authFrom(s auth.Session) authOut {
	return authOut{
		AccessToken: s.AccessToken, TokenType: "Bearer", ExpiresIn: int(s.ExpiresIn.Seconds()),
		RefreshToken: s.RefreshToken, RefreshExpiresAt: utc(s.RefreshExpiresAt), User: userFrom(s.User),
		DeletionCancelled: s.DeletionCancelled,
	}
}

// --------------------------------------------------------------------- me

type userOut struct {
	ID                  uuid.UUID  `json:"id"`
	Email               *string    `json:"email"`
	EmailVerified       bool       `json:"email_verified"`
	DisplayName         string     `json:"display_name"`
	Locale              string     `json:"locale"`
	UnitSystem          string     `json:"unit_system"`
	WeekStart           int        `json:"week_start"`
	Timezone            string     `json:"timezone"`
	Status              string     `json:"status"`
	DeletionRequestedAt *time.Time `json:"deletion_requested_at"`
	CreatedAt           time.Time  `json:"created_at"`
}

func userFrom(u store.User) userOut {
	return userOut{
		ID: u.ID, Email: u.Email, EmailVerified: u.EmailVerified, DisplayName: u.DisplayName,
		Locale: u.Locale, UnitSystem: u.UnitSystem, WeekStart: u.WeekStart, Timezone: u.Timezone,
		Status: u.Status, DeletionRequestedAt: utcPtr(u.DeletionRequestedAt), CreatedAt: utc(u.CreatedAt),
	}
}

type userUpdateIn struct {
	DisplayName *string `json:"display_name"`
	Locale      *string `json:"locale"`
	UnitSystem  *string `json:"unit_system"`
	WeekStart   *int    `json:"week_start"`
	Timezone    *string `json:"timezone"`
}

type deletionOut struct {
	DeletionRequestedAt time.Time `json:"deletion_requested_at"`
	HardDeleteAfter     time.Time `json:"hard_delete_after"`
}

// -------------------------------------------------------------- catalogue

type exerciseOut struct {
	ID              uuid.UUID `json:"id"`
	Slug            string    `json:"slug"`
	Name            string    `json:"name"`
	Aka             []string  `json:"aka"`
	Family          string    `json:"family"`
	DefaultMeasure  string    `json:"default_measure"`
	LoadSemantics   string    `json:"load_semantics"`
	IsBodyweight    bool      `json:"is_bodyweight"`
	Unilateral      bool      `json:"unilateral"`
	TempoApplicable bool      `json:"tempo_applicable"`
	Equipment       []string  `json:"equipment"`
	Summary         string    `json:"summary"`
	Cues            []string  `json:"cues"`
	CommonFaults    []string  `json:"common_faults"`
	Status          string    `json:"status"`
}

func exerciseFrom(e store.Exercise) exerciseOut {
	return exerciseOut{
		ID: e.ID, Slug: e.Slug, Name: e.Name, Aka: e.Aka, Family: e.Family, DefaultMeasure: e.DefaultMeasure,
		LoadSemantics: e.LoadSemantics, IsBodyweight: e.IsBodyweight, Unilateral: e.Unilateral,
		TempoApplicable: e.TempoApplicable, Equipment: e.Equipment, Summary: e.Summary, Cues: e.Cues,
		CommonFaults: e.CommonFaults, Status: e.Status,
	}
}

type exerciseListOut struct {
	Items          []exerciseOut `json:"items"`
	ContentVersion string        `json:"content_version"`
}

type bandOut struct {
	ID              uuid.UUID `json:"id"`
	Owner           string    `json:"owner"`
	Brand           string    `json:"brand"`
	ColourLabel     string    `json:"colour_label"`
	ResistanceMinKg float64   `json:"resistance_min_kg"`
	ResistanceMaxKg float64   `json:"resistance_max_kg"`
	LengthCm        *float64  `json:"length_cm"`
	ThicknessMm     *float64  `json:"thickness_mm"`
}

func bandFrom(b store.Band) bandOut {
	owner := "catalogue"
	if b.Mine {
		owner = "mine"
	}
	return bandOut{
		ID: b.ID, Owner: owner, Brand: b.Brand, ColourLabel: b.ColourLabel, ResistanceMinKg: b.ResistanceMinKg,
		ResistanceMaxKg: b.ResistanceMaxKg, LengthCm: b.LengthCm, ThicknessMm: b.ThicknessMm,
	}
}

type bandListOut struct {
	Items []bandOut `json:"items"`
}

type bandIn struct {
	ID              uuid.UUID `json:"id"`
	Brand           string    `json:"brand"`
	ColourLabel     string    `json:"colour_label"`
	ResistanceMinKg float64   `json:"resistance_min_kg"`
	ResistanceMaxKg float64   `json:"resistance_max_kg"`
	LengthCm        *float64  `json:"length_cm"`
	ThicknessMm     *float64  `json:"thickness_mm"`
}

// --------------------------------------------------------------- sessions

type sessionCreateIn struct {
	ID           uuid.UUID  `json:"id"`
	StartedAt    time.Time  `json:"started_at"`
	Timezone     string     `json:"timezone"`
	Title        string     `json:"title"`
	Notes        string     `json:"notes"`
	BodyweightKg *float64   `json:"bodyweight_kg"`
	IsRestDay    bool       `json:"is_rest_day"`
	TemplateID   *uuid.UUID `json:"template_id"`
	UpdatedAt    *time.Time `json:"updated_at"`
}

type sessionUpdateIn struct {
	StartedAt        Optional[time.Time] `json:"started_at"`
	EndedAt          Optional[time.Time] `json:"ended_at"`
	Timezone         Optional[string]    `json:"timezone"`
	Title            Optional[string]    `json:"title"`
	Notes            Optional[string]    `json:"notes"`
	PerceivedFatigue Optional[int]       `json:"perceived_fatigue"`
	BodyweightKg     Optional[float64]   `json:"bodyweight_kg"`
	IsRestDay        Optional[bool]      `json:"is_rest_day"`
	Status           Optional[string]    `json:"status"`
	UpdatedAt        *time.Time          `json:"updated_at"`
}

type sessionOut struct {
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
	Blocks           []blockOut `json:"blocks"`
}

func sessionFrom(s training.Session) sessionOut {
	out := sessionOut{
		ID: s.ID, StartedAt: utc(s.StartedAt), EndedAt: utcPtr(s.EndedAt), Timezone: s.Timezone,
		LocalDate: s.LocalDate.Format(dateLayout), Title: s.Title, Notes: s.Notes,
		PerceivedFatigue: s.PerceivedFatigue, BodyweightKg: s.BodyweightKg, Status: s.Status,
		IsRestDay: s.IsRestDay, TemplateID: s.TemplateID, CompletedAt: utcPtr(s.CompletedAt),
		UpdatedAt: utc(s.UpdatedAt), ServerUpdatedAt: utc(s.ServerUpdatedAt), Blocks: make([]blockOut, len(s.Blocks)),
	}
	for i, b := range s.Blocks {
		out.Blocks[i] = blockFrom(b)
	}
	return out
}

type sessionSummaryOut struct {
	ID         uuid.UUID  `json:"id"`
	StartedAt  time.Time  `json:"started_at"`
	EndedAt    *time.Time `json:"ended_at"`
	Timezone   string     `json:"timezone"`
	LocalDate  string     `json:"local_date"`
	Title      string     `json:"title"`
	Status     string     `json:"status"`
	IsRestDay  bool       `json:"is_rest_day"`
	BlockCount int        `json:"block_count"`
	SetCount   int        `json:"set_count"`
}

type sessionPageOut struct {
	Items      []sessionSummaryOut `json:"items"`
	NextCursor *string             `json:"next_cursor"`
}

type blockIn struct {
	OrderIndex    int        `json:"order_index"`
	Kind          string     `json:"kind"`
	RoundsPlanned *int       `json:"rounds_planned"`
	RoundsDone    *int       `json:"rounds_done"`
	IntervalS     *int       `json:"interval_s"`
	Notes         string     `json:"notes"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

type blockOut struct {
	ID            uuid.UUID `json:"id"`
	OrderIndex    int       `json:"order_index"`
	Kind          string    `json:"kind"`
	RoundsPlanned *int      `json:"rounds_planned"`
	RoundsDone    *int      `json:"rounds_done"`
	IntervalS     *int      `json:"interval_s"`
	Notes         string    `json:"notes"`
	UpdatedAt     time.Time `json:"updated_at"`
	Sets          []setOut  `json:"sets"`
}

func blockFrom(b training.Block) blockOut {
	out := blockOut{
		ID: b.ID, OrderIndex: b.OrderIndex, Kind: b.Kind, RoundsPlanned: b.RoundsPlanned,
		RoundsDone: b.RoundsDone, IntervalS: b.IntervalS, Notes: b.Notes, UpdatedAt: utc(b.UpdatedAt),
		Sets: make([]setOut, len(b.Sets)),
	}
	for i, s := range b.Sets {
		out.Sets[i] = setFrom(s)
	}
	return out
}

type setIn struct {
	BlockID           uuid.UUID   `json:"block_id"`
	OrderIndex        int         `json:"order_index"`
	RoundIndex        *int        `json:"round_index"`
	Kind              string      `json:"kind"`
	IsPlanned         bool        `json:"is_planned"`
	RestAfterPlannedS *int        `json:"rest_after_planned_s"`
	RestAfterActualS  *int        `json:"rest_after_actual_s"`
	RPE               *float64    `json:"rpe"`
	RIR               *int        `json:"rir"`
	CompletedAt       *time.Time  `json:"completed_at"`
	Notes             string      `json:"notes"`
	UpdatedAt         *time.Time  `json:"updated_at"`
	Elements          []elementIn `json:"elements"`
}

type elementIn struct {
	ID              uuid.UUID     `json:"id"`
	ExerciseID      uuid.UUID     `json:"exercise_id"`
	Measure         string        `json:"measure"`
	Reps            *int          `json:"reps"`
	HoldSeconds     *float64      `json:"hold_seconds"`
	DistanceM       *float64      `json:"distance_m"`
	Tempo           *string       `json:"tempo"`
	LoadKg          float64       `json:"load_kg"`
	IsEccentricOnly bool          `json:"is_eccentric_only"`
	IsPartialROM    bool          `json:"is_partial_rom"`
	ROMNote         *string       `json:"rom_note"`
	FormQuality     *int          `json:"form_quality"`
	Failed          bool          `json:"failed"`
	Assistance      *assistanceIn `json:"assistance"`
	MediaIDs        []uuid.UUID   `json:"media_ids"`
}

type assistanceIn struct {
	ID                uuid.UUID  `json:"id"`
	Type              string     `json:"type"`
	BandID            *uuid.UUID `json:"band_id"`
	BandCount         int        `json:"band_count"`
	Anchor            *string    `json:"anchor"`
	EstimatedAssistKg *float64   `json:"estimated_assist_kg"`
	Note              string     `json:"note"`
}

func (s setIn) toDomain(id, sessionID uuid.UUID) training.SetEntry {
	out := training.SetEntry{
		ID: id, SessionID: sessionID, BlockID: s.BlockID, OrderIndex: s.OrderIndex, RoundIndex: s.RoundIndex,
		Kind: orDefault(s.Kind, "working"), IsPlanned: s.IsPlanned, RestAfterPlannedS: s.RestAfterPlannedS,
		RestAfterActualS: s.RestAfterActualS, RPE: s.RPE, RIR: s.RIR, CompletedAt: s.CompletedAt, Notes: s.Notes,
		Elements: make([]training.Element, len(s.Elements)),
	}
	for i, e := range s.Elements {
		el := training.Element{
			ID: e.ID, ExerciseID: e.ExerciseID, Measure: training.Measure(e.Measure), Reps: e.Reps,
			HoldSeconds: e.HoldSeconds, DistanceM: e.DistanceM, Tempo: e.Tempo, LoadKg: e.LoadKg,
			IsEccentricOnly: e.IsEccentricOnly, IsPartialROM: e.IsPartialROM, ROMNote: e.ROMNote,
			FormQuality: e.FormQuality, Failed: e.Failed, MediaIDs: e.MediaIDs,
		}
		if a := e.Assistance; a != nil {
			el.Assistance = &training.Assistance{
				ID: a.ID, Type: a.Type, BandID: a.BandID, BandCount: a.BandCount, Anchor: a.Anchor,
				EstimatedAssistKg: a.EstimatedAssistKg, Note: a.Note,
			}
		}
		out.Elements[i] = el
	}
	return out
}

type setOut struct {
	ID                uuid.UUID    `json:"id"`
	BlockID           uuid.UUID    `json:"block_id"`
	OrderIndex        int          `json:"order_index"`
	RoundIndex        *int         `json:"round_index"`
	Kind              string       `json:"kind"`
	IsPlanned         bool         `json:"is_planned"`
	RestAfterPlannedS *int         `json:"rest_after_planned_s"`
	RestAfterActualS  *int         `json:"rest_after_actual_s"`
	RPE               *float64     `json:"rpe"`
	RIR               *int         `json:"rir"`
	CompletedAt       *time.Time   `json:"completed_at"`
	Notes             string       `json:"notes"`
	UpdatedAt         time.Time    `json:"updated_at"`
	Elements          []elementOut `json:"elements"`
}

func setFrom(s training.SetEntry) setOut {
	out := setOut{
		ID: s.ID, BlockID: s.BlockID, OrderIndex: s.OrderIndex, RoundIndex: s.RoundIndex, Kind: s.Kind,
		IsPlanned: s.IsPlanned, RestAfterPlannedS: s.RestAfterPlannedS, RestAfterActualS: s.RestAfterActualS,
		RPE: s.RPE, RIR: s.RIR, CompletedAt: utcPtr(s.CompletedAt), Notes: s.Notes, UpdatedAt: utc(s.UpdatedAt),
		Elements: make([]elementOut, len(s.Elements)),
	}
	for i, e := range s.Elements {
		out.Elements[i] = elementFrom(e)
	}
	return out
}

type elementOut struct {
	ID              uuid.UUID      `json:"id"`
	OrderIndex      int            `json:"order_index"`
	ExerciseID      uuid.UUID      `json:"exercise_id"`
	Measure         string         `json:"measure"`
	Reps            *int           `json:"reps"`
	HoldSeconds     *float64       `json:"hold_seconds"`
	DistanceM       *float64       `json:"distance_m"`
	Tempo           *string        `json:"tempo"`
	LoadKg          float64        `json:"load_kg"`
	IsEccentricOnly bool           `json:"is_eccentric_only"`
	IsPartialROM    bool           `json:"is_partial_rom"`
	ROMNote         *string        `json:"rom_note"`
	FormQuality     *int           `json:"form_quality"`
	Failed          bool           `json:"failed"`
	AssistanceClass string         `json:"assistance_class"`
	Assistance      *assistanceOut `json:"assistance"`
	MediaIDs        []uuid.UUID    `json:"media_ids"`
}

type assistanceOut struct {
	ID                uuid.UUID  `json:"id"`
	Type              string     `json:"type"`
	BandID            *uuid.UUID `json:"band_id"`
	BandCount         int        `json:"band_count"`
	Anchor            *string    `json:"anchor"`
	EstimatedAssistKg *float64   `json:"estimated_assist_kg"`
	Note              string     `json:"note"`
}

func elementFrom(e training.Element) elementOut {
	out := elementOut{
		ID: e.ID, OrderIndex: e.OrderIndex, ExerciseID: e.ExerciseID, Measure: string(e.Measure), Reps: e.Reps,
		HoldSeconds: e.HoldSeconds, DistanceM: e.DistanceM, Tempo: e.Tempo, LoadKg: e.LoadKg,
		IsEccentricOnly: e.IsEccentricOnly, IsPartialROM: e.IsPartialROM, ROMNote: e.ROMNote,
		FormQuality: e.FormQuality, Failed: e.Failed, AssistanceClass: string(e.AssistanceClass),
		MediaIDs: nonNil(e.MediaIDs),
	}
	if a := e.Assistance; a != nil {
		out.Assistance = &assistanceOut{
			ID: a.ID, Type: a.Type, BandID: a.BandID, BandCount: a.BandCount, Anchor: a.Anchor,
			EstimatedAssistKg: a.EstimatedAssistKg, Note: a.Note,
		}
	}
	return out
}

type reorderIn struct {
	Blocks []uuid.UUID `json:"blocks"`
	Sets   []struct {
		BlockID uuid.UUID   `json:"block_id"`
		SetIDs  []uuid.UUID `json:"set_ids"`
	} `json:"sets"`
}

type lastSetOut struct {
	SessionID uuid.UUID `json:"session_id"`
	LocalDate string    `json:"local_date"`
	Set       setOut    `json:"set"`
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
