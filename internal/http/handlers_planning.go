package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	core "github.com/jaelricco/hefesto/internal/domain/planning"
	"github.com/jaelricco/hefesto/internal/planning"
)

// The planning tag (spec §10.3). Every call goes through the planner
// service; without one, or with an invalid knowledge base, the endpoints
// answer planning-unavailable.

var regionRE = regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)

func (h *handlers) planner() (*planning.Service, error) {
	if h.Planner == nil {
		return nil, planning.ErrUnavailable
	}
	return h.Planner, nil
}

// knowledge returns the service and its current knowledge base.
func (h *handlers) knowledge(r *http.Request) (*planning.Service, *core.Knowledge, error) {
	svc, err := h.planner()
	if err != nil {
		return nil, nil, err
	}
	k, err := svc.Knowledge.Current(r.Context())
	if err != nil {
		return nil, nil, fmt.Errorf("planner knowledge: %w", err)
	}
	return svc, k, nil
}

func userOf(r *http.Request) uuid.UUID { return principalFrom(r.Context()).UserID }

// ------------------------------------------------------------- onboarding

func (h *handlers) submitOnboarding(w http.ResponseWriter, r *http.Request) error {
	svc, k, err := h.knowledge(r)
	if err != nil {
		return err
	}
	var in onboardingIn
	raw, err := h.bodyRaw(w, r, "OnboardingAnswers", &in)
	if err != nil {
		return err
	}
	answers, err := in.toCore()
	if err != nil {
		return err
	}
	run := func() (int, []byte, error) {
		res, p, err := svc.Onboard(r.Context(), userOf(r), answers)
		if err != nil {
			return 0, nil, err
		}
		body, err := json.Marshal(onboardingResultFrom(k, res, p))
		if err != nil {
			return 0, nil, fmt.Errorf("encoding onboarding result: %w", err)
		}
		return http.StatusOK, body, nil
	}
	key := r.Header.Get("Idempotency-Key")
	if key == "" {
		status, body, err := run()
		if err != nil {
			return err
		}
		writeRaw(w, status, body)
		return nil
	}
	if !idempotencyKeyRE.MatchString(key) {
		return errBadRequest{"Idempotency-Key must be 8 to 255 printable ASCII characters"}
	}
	return h.withIdempotencyKey(w, r, key, raw, run)
}

// ---------------------------------------------------------- profile, goals

func (h *handlers) getTrainingProfile(w http.ResponseWriter, r *http.Request) error {
	svc, err := h.planner()
	if err != nil {
		return err
	}
	k, snap, err := svc.View(r.Context(), userOf(r))
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, profileFrom(snap.Profile, k.IsMinor(snap.Profile.BirthYear, svc.Clock.Now())))
	return nil
}

func (h *handlers) updateTrainingProfile(w http.ResponseWriter, r *http.Request) error {
	svc, k, err := h.knowledge(r)
	if err != nil {
		return err
	}
	var in profileIn
	if err := h.body(w, r, "TrainingProfileUpdate", &in); err != nil {
		return err
	}
	p, err := svc.UpdateProfile(r.Context(), userOf(r), in.toCore())
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, profileFrom(p, k.IsMinor(p.BirthYear, svc.Clock.Now())))
	return nil
}

func (h *handlers) getTrainingGoals(w http.ResponseWriter, r *http.Request) error {
	svc, err := h.planner()
	if err != nil {
		return err
	}
	k, snap, err := svc.View(r.Context(), userOf(r))
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, goalListFrom(snap.Goals, k.Realism(snap, svc.Clock.Now())))
	return nil
}

func (h *handlers) updateTrainingGoals(w http.ResponseWriter, r *http.Request) error {
	svc, err := h.planner()
	if err != nil {
		return err
	}
	var in goalsIn
	if err := h.body(w, r, "TrainingGoalsUpdate", &in); err != nil {
		return err
	}
	goals, err := goalsToCore(in.Goals)
	if err != nil {
		return err
	}
	out, realism, err := svc.SetGoals(r.Context(), userOf(r), goals)
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, goalListFrom(out, realism))
	return nil
}

// ------------------------------------------------------------------- plan

func (h *handlers) getTrainingPlan(w http.ResponseWriter, r *http.Request) error {
	svc, k, err := h.knowledge(r)
	if err != nil {
		return err
	}
	var p core.Plan
	if v := r.URL.Query().Get("week"); v != "" {
		day, perr := time.Parse(dateLayout, v)
		if perr != nil {
			return errBadRequest{"week is not a date (YYYY-MM-DD)"}
		}
		p, err = svc.WeekPlan(r.Context(), userOf(r), day)
	} else {
		p, err = svc.Plan(r.Context(), userOf(r))
	}
	if err != nil {
		return err
	}
	// A plan never changes; a new one has a new ID.
	if notModified(w, r, p.ID) {
		return nil
	}
	WriteJSON(w, r, http.StatusOK, planFrom(k, p))
	return nil
}

func (h *handlers) regenerateTrainingPlan(w http.ResponseWriter, r *http.Request) error {
	svc, k, err := h.knowledge(r)
	if err != nil {
		return err
	}
	p, err := svc.Regenerate(r.Context(), userOf(r))
	if err != nil {
		return err
	}
	w.Header().Set("ETag", `"`+p.ID+`"`)
	WriteJSON(w, r, http.StatusOK, planFrom(k, p))
	return nil
}

func (h *handlers) getPlannedSession(w http.ResponseWriter, r *http.Request) error {
	svc, k, err := h.knowledge(r)
	if err != nil {
		return err
	}
	id, err := pathID(r, "plannedSessionId")
	if err != nil {
		return err
	}
	p, s, err := svc.PlannedSession(r.Context(), userOf(r), id)
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, plannedSessionDetailOut{PlanID: p.ID, WeekStart: dateOut(p.WeekStart),
		RulesetVersion: p.RulesetVersion, Session: plannedSessionFrom(k, s), Disclaimer: p.Disclaimer})
	return nil
}

// page reads the cursor and limit of a list, newest first.
func page(r *http.Request) (*planning.Cursor, int, error) {
	q := r.URL.Query()
	limit := 20
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			return nil, 0, errBadRequest{"limit must be between 1 and 100"}
		}
		limit = n
	}
	if v := q.Get("cursor"); v != "" {
		at, id, err := decodeCursor(v)
		if err != nil {
			return nil, 0, errBadRequest{"cursor is not one this server issued"}
		}
		return &planning.Cursor{At: at, ID: id}, limit, nil
	}
	return nil, limit, nil
}

func (h *handlers) listPlanDecisions(w http.ResponseWriter, r *http.Request) error {
	svc, err := h.planner()
	if err != nil {
		return err
	}
	after, limit, err := page(r)
	if err != nil {
		return err
	}
	ds, err := svc.Decisions(r.Context(), userOf(r), after, limit+1) // one extra says whether there is more
	if err != nil {
		return err
	}
	out := decisionPageOut{Items: []decisionOut{}}
	if len(ds) > limit {
		last := ds[limit-1]
		c := encodeCursor(last.At, last.ID)
		out.NextCursor, ds = &c, ds[:limit]
	}
	for _, d := range ds {
		out.Items = append(out.Items, decisionOut{ID: d.ID, Trigger: d.Trigger, SourceID: d.SourceID,
			OccurredAt: d.At.UTC(), Changes: changesFrom(d.Changes)})
	}
	WriteJSON(w, r, http.StatusOK, out)
	return nil
}

// ----------------------------------------------------------------- events

func (h *handlers) reportPain(w http.ResponseWriter, r *http.Request) error {
	svc, err := h.planner()
	if err != nil {
		return err
	}
	var in painIn
	if err := h.body(w, r, "PainReportCreate", &in); err != nil {
		return err
	}
	out, err := svc.ReportPain(r.Context(), userOf(r), in.ID.String(), in.toCore())
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, eventResultFrom(out))
	return nil
}

func (h *handlers) listPainReports(w http.ResponseWriter, r *http.Request) error {
	svc, err := h.planner()
	if err != nil {
		return err
	}
	after, limit, err := page(r)
	if err != nil {
		return err
	}
	rs, err := svc.PainReports(r.Context(), userOf(r), after, limit+1)
	if err != nil {
		return err
	}
	out := painPageOut{Items: []painReportOut{}}
	if len(rs) > limit {
		last := rs[limit-1]
		id, err := uuid.Parse(last.ID)
		if err != nil {
			return fmt.Errorf("pain report %q: %w", last.ID, err)
		}
		c := encodeCursor(last.At, id)
		out.NextCursor, rs = &c, rs[:limit]
	}
	for _, p := range rs {
		out.Items = append(out.Items, painReportFrom(p))
	}
	WriteJSON(w, r, http.StatusOK, out)
	return nil
}

func (h *handlers) reportExertionSymptoms(w http.ResponseWriter, r *http.Request) error {
	svc, err := h.planner()
	if err != nil {
		return err
	}
	var in eventIn
	if err := h.body(w, r, "PlanEventCreate", &in); err != nil {
		return err
	}
	out, err := svc.ReportSymptoms(r.Context(), userOf(r), in.ID.String())
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, eventResultFrom(out))
	return nil
}

// region returns the region of the path, which must be one of the knowledge
// base's.
func region(r *http.Request, k *core.Knowledge) (string, error) {
	id := chi.URLParam(r, "region")
	if !regionRE.MatchString(id) {
		return "", errBadRequest{"region is not a region ID"}
	}
	if _, ok := k.Region(id); !ok {
		return "", fmt.Errorf("region %s: %w", id, planning.ErrNotFound)
	}
	return id, nil
}

func (h *handlers) listMyRegions(w http.ResponseWriter, r *http.Request) error {
	svc, err := h.planner()
	if err != nil {
		return err
	}
	k, snap, err := svc.View(r.Context(), userOf(r))
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, regionOverviewFrom(k, k.RegionStatuses(snap, svc.Clock.Now())))
	return nil
}

func (h *handlers) answerRedFlags(w http.ResponseWriter, r *http.Request) error {
	svc, k, err := h.knowledge(r)
	if err != nil {
		return err
	}
	id, err := region(r, k)
	if err != nil {
		return err
	}
	var in redFlagsIn
	if err := h.body(w, r, "RedFlagAnswers", &in); err != nil {
		return err
	}
	out, err := svc.AnswerRedFlags(r.Context(), userOf(r), in.ID.String(), id, in.Answers)
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, eventResultFrom(out))
	return nil
}

func (h *handlers) confirmRegionClearance(w http.ResponseWriter, r *http.Request) error {
	svc, k, err := h.knowledge(r)
	if err != nil {
		return err
	}
	id, err := region(r, k)
	if err != nil {
		return err
	}
	return h.clearance(w, r, svc, id)
}

func (h *handlers) confirmScreeningClearance(w http.ResponseWriter, r *http.Request) error {
	svc, err := h.planner()
	if err != nil {
		return err
	}
	return h.clearance(w, r, svc, "")
}

func (h *handlers) clearance(w http.ResponseWriter, r *http.Request, svc *planning.Service, region string) error {
	var in eventIn
	if err := h.body(w, r, "PlanEventCreate", &in); err != nil {
		return err
	}
	out, err := svc.ConfirmClearance(r.Context(), userOf(r), in.ID.String(), region)
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, eventResultFrom(out))
	return nil
}

func (h *handlers) changeHealthConsent(w http.ResponseWriter, r *http.Request) error {
	svc, err := h.planner()
	if err != nil {
		return err
	}
	var in consentIn
	if err := h.body(w, r, "HealthConsentChange", &in); err != nil {
		return err
	}
	out, err := svc.SetConsent(r.Context(), userOf(r), in.ID.String(),
		core.ConsentChange{Granted: in.Granted, Screening: in.Screening, PastInjuries: in.PastInjuries})
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, eventResultFrom(out))
	return nil
}

func (h *handlers) listMyCapacities(w http.ResponseWriter, r *http.Request) error {
	svc, err := h.planner()
	if err != nil {
		return err
	}
	k, snap, err := svc.View(r.Context(), userOf(r))
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, capacitiesFrom(k, snap.Capacities))
	return nil
}

// -------------------------------------------------------------- catalogue

// catalogue answers a catalogue endpoint, keyed on the ruleset version.
func (h *handlers) catalogue(view func(*core.Knowledge) any) func(http.ResponseWriter, *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		_, k, err := h.knowledge(r)
		if err != nil {
			return err
		}
		if notModified(w, r, k.Version) {
			return nil
		}
		WriteJSON(w, r, http.StatusOK, view(k))
		return nil
	}
}

func (h *handlers) listPlannerRules() func(http.ResponseWriter, *http.Request) error {
	return h.catalogue(func(k *core.Knowledge) any { return rulesFrom(k) })
}

func (h *handlers) listPlannerSources() func(http.ResponseWriter, *http.Request) error {
	return h.catalogue(func(k *core.Knowledge) any { return sourcesFrom(k) })
}

func (h *handlers) listPlannerParameters() func(http.ResponseWriter, *http.Request) error {
	return h.catalogue(func(k *core.Knowledge) any { return parametersFrom(k) })
}

func (h *handlers) getPlannerCatalogue() func(http.ResponseWriter, *http.Request) error {
	return h.catalogue(func(k *core.Knowledge) any { return catalogueFrom(k) })
}
