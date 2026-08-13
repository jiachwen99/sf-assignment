package api

import (
	"net/http"
	"strconv"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
)

type dependencyBody struct {
	DependsOnID int64 `json:"dependsOnId"`
}

type dependencyView struct {
	Dependencies []domain.Blocker `json:"dependencies"`
	Dependents   []domain.Blocker `json:"dependents"`
}

// Both directions in one response. The panel draws the chain around the task,
// so asking for them separately would only mean two round trips to render one
// component.
func (s *Server) todoDependencies(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	dependencies, err := s.svc.Dependencies(r.Context(), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	dependents, err := s.svc.Dependents(r.Context(), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, dependencyView{
		Dependencies: orEmpty(dependencies),
		Dependents:   orEmpty(dependents),
	})
}

func (s *Server) addDependency(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	body, err := decode[dependencyBody](r)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if body.DependsOnID <= 0 {
		s.fail(w, r, domain.Invalid("dependsOnId", "Pick a task for this one to wait for"))
		return
	}
	if err := s.svc.AddDependency(r.Context(), id, body.DependsOnID); err != nil {
		s.fail(w, r, err)
		return
	}
	// Answering with the chain saves the client a follow-up request to render
	// the change it just made.
	s.todoDependencies(w, r)
}

func (s *Server) removeDependency(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	dependsOnID, err := strconv.ParseInt(r.PathValue("dependsOnId"), 10, 64)
	if err != nil || dependsOnID <= 0 {
		s.fail(w, r, domain.Invalid("dependsOnId", "Task id must be a positive integer"))
		return
	}
	if err := s.svc.RemoveDependency(r.Context(), id, dependsOnID); err != nil {
		s.fail(w, r, err)
		return
	}
	s.todoDependencies(w, r)
}

func (s *Server) searchTodos(w http.ResponseWriter, r *http.Request) {
	// Optional: a search from outside a task has nothing to exclude.
	exclude, _ := strconv.ParseInt(r.URL.Query().Get("exclude"), 10, 64)

	items, err := s.svc.Search(r.Context(), r.URL.Query().Get("q"), exclude)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// An empty array, not null, so the client can map over it without a guard.
func orEmpty(bs []domain.Blocker) []domain.Blocker {
	if bs == nil {
		return []domain.Blocker{}
	}
	return bs
}
