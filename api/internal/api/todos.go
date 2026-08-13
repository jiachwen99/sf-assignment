package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
	"github.com/jiachwen99/sf-assignment/api/internal/service"
)

type todoBody struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	DueDate     *time.Time `json:"dueDate"`
	Status      string     `json:"status"`
	Priority    string     `json:"priority"`

	// The version the client last read. Ignored on create, required on update.
	Version int `json:"version"`
}

func (b todoBody) input() service.TodoInput {
	return service.TodoInput{
		Name:        b.Name,
		Description: b.Description,
		DueDate:     b.DueDate,
		Status:      domain.Status(b.Status),
		Priority:    domain.Priority(b.Priority),
	}
}

func (s *Server) listTodos(w http.ResponseWriter, r *http.Request) {
	items, err := s.svc.Todos(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if items == nil {
		items = []domain.Todo{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) createTodo(w http.ResponseWriter, r *http.Request) {
	body, err := decode[todoBody](r)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	todo, err := s.svc.Create(r.Context(), body.input())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, todo)
}

func (s *Server) getTodo(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	todo, err := s.svc.Todo(r.Context(), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, todo)
}

func (s *Server) updateTodo(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	body, err := decode[todoBody](r)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if body.Version <= 0 {
		s.fail(w, r, domain.Invalid("version", "Version must be the version you last read"))
		return
	}
	todo, err := s.svc.Update(r.Context(), id, body.Version, body.input())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, todo)
}

func (s *Server) deleteTodo(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	// A DELETE has no body by convention, so the version rides in the query.
	version, err := strconv.Atoi(r.URL.Query().Get("version"))
	if err != nil || version <= 0 {
		s.fail(w, r, domain.Invalid("version", "Version must be the version you last read"))
		return
	}
	if err := s.svc.Delete(r.Context(), id, version); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
