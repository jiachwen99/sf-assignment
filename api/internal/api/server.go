package api

import (
	"log/slog"
	"net/http"
	"sort"

	"github.com/jiachwen99/sf-assignment/api/internal/events"
	"github.com/jiachwen99/sf-assignment/api/internal/service"
)

type Server struct {
	svc    *service.Service
	hub    *events.Hub
	log    *slog.Logger
	mux    *http.ServeMux
	routes []string
}

// ServeMux carries method and wildcard patterns, so no third-party router.
//
// The routes are collected as they are registered rather than listed a second
// time beside the registrations, because two lists is exactly how they start to
// disagree. The parity test compares this against the specification.
func NewServer(svc *service.Service, hub *events.Hub, log *slog.Logger) *Server {
	s := &Server{svc: svc, hub: hub, log: log, mux: http.NewServeMux()}

	// The subscriber count is here so a test can assert that connections are
	// released rather than waiting to see whether memory grows.
	s.handle("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"subscribers": hub.Subscribers()})
	})

	s.handle("GET /api/todos", s.listTodos)
	s.handle("POST /api/todos", s.createTodo)
	s.handle("GET /api/todos/{id}", s.getTodo)
	s.handle("PUT /api/todos/{id}", s.updateTodo)
	s.handle("DELETE /api/todos/{id}", s.deleteTodo)
	s.handle("POST /api/todos/{id}/complete", s.completeTodo)
	s.handle("POST /api/todos/bulk/complete", s.bulkComplete)
	s.handle("POST /api/todos/bulk/archive", s.bulkArchive)

	s.handle("POST /api/auth/register", s.register)
	s.handle("POST /api/auth/login", s.login)
	s.handle("POST /api/auth/logout", s.logout)
	s.handle("GET /api/auth/me", s.currentUser)

	s.handle("GET /api/events", s.stream)

	s.handle("GET /api/todos/search", s.searchTodos)
	s.handle("GET /api/todos/counts", s.counts)
	s.handle("GET /api/todos/trash", s.listTrash)
	s.handle("POST /api/todos/{id}/restore", s.restoreTodo)
	s.handle("GET /api/todos/{id}/events", s.todoEvents)
	s.handle("GET /api/todos/{id}/dependencies", s.todoDependencies)
	s.handle("POST /api/todos/{id}/dependencies", s.addDependency)
	s.handle("DELETE /api/todos/{id}/dependencies/{dependsOnId}", s.removeDependency)

	// The specification and the page that renders it describe the API rather
	// than being part of it, so the parity test skips them by prefix.
	s.handle("GET /openapi.yaml", s.openAPI)
	s.handle("GET /docs", s.docs)

	return s
}

func (s *Server) handle(pattern string, handler http.HandlerFunc) {
	s.routes = append(s.routes, pattern)
	s.mux.HandleFunc(pattern, handler)
}

// Every request passes through session resolution, so any write that records
// an event can name who made it without each handler remembering to look.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.withSession(s.mux).ServeHTTP(w, r)
}

// Routes returns every pattern the server serves, sorted.
func (s *Server) Routes() []string {
	out := append([]string(nil), s.routes...)
	sort.Strings(out)
	return out
}
