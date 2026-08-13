package api

import (
	"log/slog"
	"net/http"

	"github.com/jiachwen99/sf-assignment/api/internal/service"
)

type Server struct {
	svc *service.Service
	log *slog.Logger
	mux *http.ServeMux
}

// ServeMux carries method and wildcard patterns, so no third-party router.
func NewServer(svc *service.Service, log *slog.Logger) *Server {
	s := &Server{svc: svc, log: log, mux: http.NewServeMux()}

	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s.mux.HandleFunc("GET /api/todos", s.listTodos)
	s.mux.HandleFunc("POST /api/todos", s.createTodo)
	s.mux.HandleFunc("GET /api/todos/{id}", s.getTodo)
	s.mux.HandleFunc("PUT /api/todos/{id}", s.updateTodo)
	s.mux.HandleFunc("DELETE /api/todos/{id}", s.deleteTodo)
	s.mux.HandleFunc("POST /api/todos/{id}/complete", s.completeTodo)

	s.mux.HandleFunc("GET /api/todos/search", s.searchTodos)
	s.mux.HandleFunc("GET /api/todos/{id}/dependencies", s.todoDependencies)
	s.mux.HandleFunc("POST /api/todos/{id}/dependencies", s.addDependency)
	s.mux.HandleFunc("DELETE /api/todos/{id}/dependencies/{dependsOnId}", s.removeDependency)

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
