package http

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/jaelricco/hefesto/internal/domain/training"
	"github.com/jaelricco/hefesto/internal/store"
)

var slugRE = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func optionalQuery(r *http.Request, name string, maxLen int) (*string, error) {
	v := strings.TrimSpace(r.URL.Query().Get(name))
	if v == "" {
		return nil, nil
	}
	if len(v) > maxLen {
		return nil, errBadRequest{name + " is too long"}
	}
	return &v, nil
}

func (h *handlers) listExercises(w http.ResponseWriter, r *http.Request) error {
	var f store.ExerciseFilter
	var err error
	if f.Q, err = optionalQuery(r, "q", 100); err != nil {
		return err
	}
	if f.Family, err = optionalQuery(r, "family", 80); err != nil {
		return err
	}
	if f.Family != nil && !slugRE.MatchString(*f.Family) {
		return errBadRequest{"family is not a slug"}
	}
	if f.Equipment, err = optionalQuery(r, "equipment", 40); err != nil {
		return err
	}

	version, err := h.Store.ContentVersion(r.Context())
	if err != nil {
		return err
	}
	etag := `"` + version + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, no-cache")
	if version != "" && r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return nil
	}

	items, err := h.Store.ListExercises(r.Context(), f)
	if err != nil {
		return err
	}
	out := exerciseListOut{Items: make([]exerciseOut, len(items)), ContentVersion: version}
	for i, e := range items {
		out.Items[i] = exerciseFrom(e)
	}
	WriteJSON(w, r, http.StatusOK, out)
	return nil
}

func (h *handlers) getExercise(w http.ResponseWriter, r *http.Request) error {
	slug := chi.URLParam(r, "slug")
	if !slugRE.MatchString(slug) {
		return store.ErrNotFound
	}
	e, err := h.Store.GetExercise(r.Context(), slug)
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, exerciseFrom(e))
	return nil
}

func (h *handlers) listBands(w http.ResponseWriter, r *http.Request) error {
	bands, err := h.Store.ListBands(r.Context(), principalFrom(r.Context()).UserID)
	if err != nil {
		return err
	}
	out := bandListOut{Items: make([]bandOut, len(bands))}
	for i, b := range bands {
		out.Items[i] = bandFrom(b)
	}
	WriteJSON(w, r, http.StatusOK, out)
	return nil
}

func (h *handlers) createBand(w http.ResponseWriter, r *http.Request) error {
	var in bandIn
	if err := h.body(w, r, "BandCreate", &in); err != nil {
		return err
	}
	if in.ResistanceMinKg > in.ResistanceMaxKg {
		return training.FieldErrors{"/resistance_min_kg": "greater than resistance_max_kg"}.Err()
	}
	b, err := h.Store.CreateBand(r.Context(), principalFrom(r.Context()).UserID, store.Band{
		ID: in.ID, Brand: in.Brand, ColourLabel: in.ColourLabel, ResistanceMinKg: in.ResistanceMinKg,
		ResistanceMaxKg: in.ResistanceMaxKg, LengthCm: in.LengthCm, ThicknessMm: in.ThicknessMm,
	})
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusCreated, bandFrom(b))
	return nil
}

func (h *handlers) deleteBand(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "bandId")
	if err != nil {
		return err
	}
	if err := h.Store.DeleteBand(r.Context(), principalFrom(r.Context()).UserID, id); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
