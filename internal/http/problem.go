// Package http contains the HTTP transport: routing, middleware and the
// RFC 9457 problem+json error representation.
//
// This package may import internal/domain. internal/domain must never import
// this package.
package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// ContentTypeProblem is the media type mandated by RFC 9457.
const ContentTypeProblem = "application/problem+json"

// Problem is an RFC 9457 problem detail.
//
// Type is a stable, documented URI reference that clients may switch on.
// Detail is human-readable and may change; clients must not parse it.
type Problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
	// TraceID lets a user quote one token in a bug report that we can grep for.
	TraceID string `json:"trace_id,omitempty"`
	// Errors carries per-field validation failures, keyed by JSON pointer.
	Errors map[string]string `json:"errors,omitempty"`
}

// WriteProblem serialises p to w with the correct media type and status.
func WriteProblem(w http.ResponseWriter, r *http.Request, p Problem) {
	if p.Status == 0 {
		p.Status = http.StatusInternalServerError
	}
	if p.Type == "" {
		p.Type = "about:blank"
	}
	if p.Instance == "" {
		p.Instance = r.URL.Path
	}

	w.Header().Set("Content-Type", ContentTypeProblem)
	w.WriteHeader(p.Status)
	if err := json.NewEncoder(w).Encode(p); err != nil {
		slog.ErrorContext(r.Context(), "encoding problem response failed", "error", err)
	}
}

// WriteJSON serialises v as an ordinary JSON response.
func WriteJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.ErrorContext(r.Context(), "encoding response failed", "error", err)
	}
}
