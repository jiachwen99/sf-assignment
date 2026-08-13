package service

import (
	"context"
	"strings"
	"time"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
	"github.com/jiachwen99/sf-assignment/api/internal/store"
)

type Service struct {
	store *store.Store
}

func New(s *store.Store) *Service {
	return &Service{store: s}
}

type TodoInput struct {
	Name        string
	Description string
	DueDate     *time.Time
	Status      domain.Status
	Priority    domain.Priority
}

// Validation lives here rather than at the HTTP edge, so the rules hold for
// any caller and the handler stays about decoding and status codes.
func (in *TodoInput) normaliseAndValidate() error {
	in.Name = strings.TrimSpace(in.Name)

	// Messages are whole sentences naming their field, because the client shows
	// them verbatim next to the control and "must not be empty" alone says
	// nothing on its own.
	fields := map[string]string{}
	if in.Name == "" {
		fields["name"] = "Name must not be empty"
	}
	if len(in.Name) > 500 {
		fields["name"] = "Name must be 500 characters or fewer"
	}
	if in.Status == "" {
		in.Status = domain.NotStarted
	} else if !in.Status.Valid() {
		fields["status"] = "Status must be one of not_started, in_progress, completed, archived"
	}
	if in.Priority == "" {
		in.Priority = domain.Medium
	} else if !in.Priority.Valid() {
		fields["priority"] = "Priority must be one of low, medium, high"
	}

	if len(fields) > 0 {
		return &domain.ValidationError{Fields: fields}
	}
	return nil
}

func (s *Service) Create(ctx context.Context, in TodoInput) (domain.Todo, error) {
	if err := in.normaliseAndValidate(); err != nil {
		return domain.Todo{}, err
	}
	return s.store.CreateTodo(ctx, store.NewTodo{
		Name:        in.Name,
		Description: in.Description,
		DueDate:     in.DueDate,
		Status:      in.Status,
		Priority:    in.Priority,
	})
}

func (s *Service) Todo(ctx context.Context, id int64) (domain.Todo, error) {
	return s.store.Todo(ctx, id)
}

func (s *Service) Todos(ctx context.Context) ([]domain.Todo, error) {
	return s.store.Todos(ctx)
}

func (s *Service) Update(ctx context.Context, id int64, version int, in TodoInput) (domain.Todo, error) {
	if err := in.normaliseAndValidate(); err != nil {
		return domain.Todo{}, err
	}
	return s.store.UpdateTodo(ctx, store.TodoUpdate{
		ID:          id,
		Version:     version,
		Name:        in.Name,
		Description: in.Description,
		DueDate:     in.DueDate,
		Status:      in.Status,
		Priority:    in.Priority,
	})
}

func (s *Service) Delete(ctx context.Context, id int64, version int) error {
	return s.store.DeleteTodo(ctx, id, version)
}
