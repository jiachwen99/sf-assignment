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
	RecurUnit   *string    `json:"recurUnit"`
	RecurEvery  *int       `json:"recurInterval"`

	// The version the client last read. Ignored on create, required on update.
	Version int `json:"version"`
}

func (b todoBody) input() service.TodoInput {
	in := service.TodoInput{
		Name:        b.Name,
		Description: b.Description,
		DueDate:     b.DueDate,
		Status:      domain.Status(b.Status),
		Priority:    domain.Priority(b.Priority),
		RecurEvery:  b.RecurEvery,
	}
	if b.RecurUnit != nil {
		unit := domain.RecurUnit(*b.RecurUnit)
		in.RecurUnit = &unit
	}
	return in
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
	version, err := queryVersion(r)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if err := s.svc.Delete(r.Context(), id, version); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listTrash(w http.ResponseWriter, r *http.Request) {
	items, err := s.svc.Trash(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if items == nil {
		items = []domain.Todo{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// No version: nothing can edit a task while it is in the trash, so the copy
// being restored is the only copy there has been.
func (s *Server) restoreTodo(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	todo, err := s.svc.Restore(r.Context(), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, todo)
}

// Completion is its own route rather than a status in the body, because it is
// not only a status change: a recurring task also gets its next occurrence, and
// the response has to say which one so the client can point at it.
func (s *Server) completeTodo(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	version, err := queryVersion(r)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	res, err := s.svc.Complete(r.Context(), id, version)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"completed": res.Completed,
		"spawned":   res.Spawned,
	})
}

// Neither DELETE nor a completion carries a body by convention, so the version
// rides in the query string for both.
func queryVersion(r *http.Request) (int, error) {
	version, err := strconv.Atoi(r.URL.Query().Get("version"))
	if err != nil || version <= 0 {
		return 0, domain.Invalid("version", "Version must be the version you last read")
	}
	return version, nil
}
