package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *handlers) completeSession(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "sessionId")
	if err != nil {
		return err
	}
	var in completeIn
	if err := h.body(w, r, "SessionComplete", &in); err != nil {
		return err
	}
	c, err := h.Store.CompleteSession(r.Context(), writer(r, in.UpdatedAt), id, in.toStore())
	if err != nil {
		return err
	}
	out := completionFrom(c)
	WriteJSON(w, r, http.StatusOK, out)
	return nil
}

// notModified sets the content-version ETag and reports whether the client
// already has this version, in which case it has answered 304.
func notModified(w http.ResponseWriter, r *http.Request, version string) bool {
	etag := `"` + version + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, no-cache")
	if version != "" && r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return true
	}
	return false
}

func (h *handlers) getSkillGraph(w http.ResponseWriter, r *http.Request) error {
	version, err := h.Store.ContentVersion(r.Context())
	if err != nil {
		return err
	}
	if notModified(w, r, version) {
		return nil
	}
	g, err := h.Store.SkillGraph(r.Context())
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, graphFrom(g))
	return nil
}

func (h *handlers) getSkill(w http.ResponseWriter, r *http.Request) error {
	slug := chi.URLParam(r, "slug")
	if !slugRE.MatchString(slug) {
		return errBadRequest{"slug is not a slug"}
	}
	s, injuries, err := h.Store.SkillDetail(r.Context(), slug)
	if err != nil {
		return err
	}
	out := skillDetailOut{skillOut: skillFrom(s), Injuries: make([]injuryOut, len(injuries)), InjuryDisclaimer: injuryDisclaimer}
	for i, in := range injuries {
		io := injuryOut{
			Region: in.Region, Name: in.Name, Description: in.Description, RiskFactors: nonNil(in.RiskFactors),
			EarlySigns: nonNil(in.EarlySigns), PrehabExercises: make([]prehabOut, len(in.Prehab)), Disclaimer: in.Disclaimer,
		}
		for j, p := range in.Prehab {
			io.PrehabExercises[j] = prehabOut{ExerciseID: p.ExerciseID, Slug: p.Slug}
		}
		out.Injuries[i] = io
	}
	WriteJSON(w, r, http.StatusOK, out)
	return nil
}

func (h *handlers) getMySkillMap(w http.ResponseWriter, r *http.Request) error {
	g, states, err := h.Store.SkillMap(r.Context(), principalFrom(r.Context()).UserID)
	if err != nil {
		return err
	}
	out := skillMapOut{skillGraphOut: graphFrom(g), States: make([]levelStateOut, len(states))}
	for i, s := range states {
		out.States[i] = levelStateOut{
			LevelID: s.LevelID, State: string(s.State), BestValue: s.BestValue, BestUnit: s.BestUnit,
			BestAt: utcPtr(s.BestAt), FirstAchievedAt: utcPtr(s.FirstAchievedAt), Verification: s.Verification,
			StaleSince: utcPtr(s.StaleSince), AttemptsCount: s.Attempts,
		}
	}
	WriteJSON(w, r, http.StatusOK, out)
	return nil
}

func (h *handlers) attestLevel(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "levelId")
	if err != nil {
		return err
	}
	var in attestIn
	if err := h.body(w, r, "AttestRequest", &in); err != nil {
		return err
	}
	u, newly, err := h.Store.AttestLevel(r.Context(), principalFrom(r.Context()).UserID, id)
	if err != nil {
		return err
	}
	if newly == nil {
		newly = []uuid.UUID{}
	}
	WriteJSON(w, r, http.StatusOK, attestOut{Unlock: unlockFrom(u), NewlyAvailable: newly})
	return nil
}

func (h *handlers) getMyProgress(w http.ResponseWriter, r *http.Request) error {
	p, err := h.Store.UserProgress(r.Context(), principalFrom(r.Context()).UserID)
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, progressOut{XPTotal: p.XPTotal, Streak: streakFrom(p.Streak)})
	return nil
}
