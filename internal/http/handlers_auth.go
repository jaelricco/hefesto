package http

import (
	"net/http"
	"time"

	"github.com/jaelricco/hefesto/internal/auth"
	"github.com/jaelricco/hefesto/internal/domain/training"
	"github.com/jaelricco/hefesto/internal/store"
)

func (h *handlers) register(w http.ResponseWriter, r *http.Request) error {
	var in registerIn
	if err := h.body(w, r, "RegisterRequest", &in); err != nil {
		return err
	}
	if in.Timezone != "" {
		if err := checkTimezone("/timezone", in.Timezone); err != nil {
			return err
		}
	}
	sess, err := h.Auth.Register(r.Context(), auth.Registration{
		Email: in.Email, Password: in.Password, DisplayName: in.DisplayName, Locale: in.Locale, Timezone: in.Timezone,
	}, h.client(r, in.Device))
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusCreated, authFrom(sess))
	return nil
}

func (h *handlers) login(w http.ResponseWriter, r *http.Request) error {
	var in loginIn
	if err := h.body(w, r, "LoginRequest", &in); err != nil {
		return err
	}
	sess, err := h.Auth.Login(r.Context(), in.Email, in.Password, h.client(r, in.Device))
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, authFrom(sess))
	return nil
}

func (h *handlers) apple(w http.ResponseWriter, r *http.Request) error {
	var in appleIn
	if err := h.body(w, r, "AppleSignInRequest", &in); err != nil {
		return err
	}
	sess, created, err := h.Auth.SignInWithApple(r.Context(), auth.AppleSignIn{
		IdentityToken: in.IdentityToken, Nonce: in.Nonce, DisplayName: in.DisplayName,
	}, h.client(r, in.Device))
	if err != nil {
		return err
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	WriteJSON(w, r, status, authFrom(sess))
	return nil
}

func (h *handlers) refresh(w http.ResponseWriter, r *http.Request) error {
	var in refreshIn
	if err := h.body(w, r, "RefreshRequest", &in); err != nil {
		return err
	}
	sess, err := h.Auth.Refresh(r.Context(), in.RefreshToken, h.client(r, nil))
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, authFrom(sess))
	return nil
}

func (h *handlers) logout(w http.ResponseWriter, r *http.Request) error {
	var in refreshIn
	if err := h.body(w, r, "RefreshRequest", &in); err != nil {
		return err
	}
	if err := h.Auth.Logout(r.Context(), in.RefreshToken); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// ---------------------------------------------------------------------- me

func (h *handlers) getMe(w http.ResponseWriter, r *http.Request) error {
	u, err := h.Store.UserByID(r.Context(), principalFrom(r.Context()).UserID)
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, userFrom(u))
	return nil
}

func (h *handlers) updateMe(w http.ResponseWriter, r *http.Request) error {
	var in userUpdateIn
	if err := h.body(w, r, "UserUpdate", &in); err != nil {
		return err
	}
	if in.Timezone != nil {
		if err := checkTimezone("/timezone", *in.Timezone); err != nil {
			return err
		}
	}
	u, err := h.Store.UpdateProfile(r.Context(), principalFrom(r.Context()).UserID, store.ProfileUpdate{
		DisplayName: in.DisplayName, Locale: in.Locale, UnitSystem: in.UnitSystem,
		WeekStart: in.WeekStart, Timezone: in.Timezone,
	})
	if err != nil {
		return err
	}
	WriteJSON(w, r, http.StatusOK, userFrom(u))
	return nil
}

func (h *handlers) deleteMe(w http.ResponseWriter, r *http.Request) error {
	u, err := h.Store.RequestDeletion(r.Context(), principalFrom(r.Context()).UserID)
	if err != nil {
		return err
	}
	at := u.DeletionRequestedAt.UTC()
	WriteJSON(w, r, http.StatusAccepted, deletionOut{
		DeletionRequestedAt: at, HardDeleteAfter: at.Add(h.DeletionGrace),
	})
	return nil
}

// checkTimezone requires an IANA zone name the server knows.
func checkTimezone(field, name string) error {
	if _, err := time.LoadLocation(name); err != nil || name == "Local" {
		return training.FieldErrors{field: "not a known IANA time zone"}.Err()
	}
	return nil
}
