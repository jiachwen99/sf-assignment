package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
)

// The only place a status code is chosen. Everything below returns one of the
// domain errors and this decides what it means over HTTP.
type errorBody struct {
	Error    string            `json:"error"`
	Fields   map[string]string `json:"fields,omitempty"`
	Current  *domain.Todo      `json:"current,omitempty"`
	Blockers []domain.Blocker  `json:"blockers,omitempty"`
}

func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	var (
		invalid  *domain.ValidationError
		conflict *domain.ConflictError
		blocked  *domain.BlockedError
	)

	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorBody{Error: "not found"})

	case errors.As(err, &invalid):
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "invalid input", Fields: invalid.Fields})

	case errors.As(err, &conflict):
		current := conflict.Current
		writeJSON(w, http.StatusConflict, errorBody{
			Error:   "This task was changed by someone else",
			Current: &current,
		})

	// 409 rather than 422: the request is well formed and would be legal once
	// the blockers are done. It conflicts with the current state, not the rules.
	case errors.As(err, &blocked):
		writeJSON(w, http.StatusConflict, errorBody{
			Error:    "This task is blocked by unfinished work",
			Blockers: blocked.Blockers,
		})

	default:
		s.log.Error("unhandled", "err", err, "path", r.URL.Path)
		writeJSON(w, http.StatusInternalServerError, errorBody{Error: "internal error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Default().Error("encode response", "err", err)
	}
}
