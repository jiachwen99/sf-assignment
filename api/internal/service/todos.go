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
	RecurUnit   *domain.RecurUnit
	RecurEvery  *int
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

	if in.RecurUnit != nil {
		if !in.RecurUnit.Valid() {
			fields["recurUnit"] = "Repeat must be by day, week or month"
		}
		if in.RecurEvery == nil || *in.RecurEvery < 1 {
			fields["recurInterval"] = "Repeat interval must be at least 1"
		}
	} else if in.RecurEvery != nil {
		fields["recurUnit"] = "Repeat needs a unit when an interval is given"
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
		RecurUnit:   in.RecurUnit,
		RecurEvery:  in.RecurEvery,
		RecurAnchor: anchorFor(in),
	})
}

func (s *Service) Todo(ctx context.Context, id int64) (domain.Todo, error) {
	return s.store.Todo(ctx, id)
}

func (s *Service) List(ctx context.Context, f store.ListFilter) (store.Page, error) {
	return s.store.List(ctx, f)
}

func (s *Service) Update(ctx context.Context, id int64, version int, in TodoInput) (domain.Todo, error) {
	if err := in.normaliseAndValidate(); err != nil {
		return domain.Todo{}, err
	}
	existing, err := s.store.Todo(ctx, id)
	if err != nil {
		return domain.Todo{}, err
	}

	// Completion is not a field you can set. It opens the next occurrence of a
	// repeating task and moves the schedule onto it, and an ordinary update
	// does neither, so allowing it here would leave a finished row still
	// carrying a schedule that nothing will ever act on.
	if in.Status == domain.Completed && existing.Status != domain.Completed {
		return domain.Todo{}, domain.Invalid("status",
			"Use the complete action to finish a task, so a repeating one opens its next occurrence")
	}

	// Set once, when the task first becomes recurring, then carried. Deriving
	// it from the current due date on every edit would undo the clamp: a task
	// showing 28 February would re-anchor to the 28th and never see the 31st
	// again.
	anchor := existing.RecurAnchor
	if in.RecurUnit == nil {
		anchor = nil
	} else if anchor == nil {
		anchor = anchorFor(in)
	}

	return s.store.UpdateTodo(ctx, store.TodoUpdate{
		ID:          id,
		Version:     version,
		Name:        in.Name,
		Description: in.Description,
		DueDate:     in.DueDate,
		Status:      in.Status,
		Priority:    in.Priority,
		RecurUnit:   in.RecurUnit,
		RecurEvery:  in.RecurEvery,
		RecurAnchor: anchor,
	})
}

// A schedule with no date has nothing to count from. The task still repeats in
// the sense that the field is set; it just cannot produce an occurrence until
// somebody gives it a due date.
func anchorFor(in TodoInput) *time.Time {
	if in.RecurUnit == nil {
		return nil
	}
	return in.DueDate
}

func (s *Service) Complete(ctx context.Context, id int64, version int) (store.CompleteResult, error) {
	return s.store.Complete(ctx, id, version, time.Now().UTC())
}

func (s *Service) Dependencies(ctx context.Context, id int64) ([]domain.Blocker, error) {
	return s.store.Dependencies(ctx, id)
}

func (s *Service) Dependents(ctx context.Context, id int64) ([]domain.Blocker, error) {
	return s.store.Dependents(ctx, id)
}

func (s *Service) AddDependency(ctx context.Context, id, dependsOnID int64) error {
	return s.store.AddDependency(ctx, id, dependsOnID)
}

func (s *Service) RemoveDependency(ctx context.Context, id, dependsOnID int64) error {
	return s.store.RemoveDependency(ctx, id, dependsOnID)
}

// The floor and the ceiling both belong here rather than at the HTTP edge: the
// short-query rule is about how selective a trigram scan is, and the cap is
// about how many results a picker can usefully show.
const (
	minSearch     = 3
	searchResults = 10
)

func (s *Service) Search(ctx context.Context, term string, excludeID int64) ([]domain.Todo, error) {
	term = strings.TrimSpace(term)
	if len(term) < minSearch {
		return []domain.Todo{}, nil
	}
	return s.store.SearchTodos(ctx, term, excludeID, searchResults)
}

func (s *Service) Delete(ctx context.Context, id int64, version int) error {
	return s.store.DeleteTodo(ctx, id, version)
}

func (s *Service) Restore(ctx context.Context, id int64) (domain.Todo, error) {
	return s.store.RestoreTodo(ctx, id)
}

func (s *Service) Counts(ctx context.Context) (store.Counts, error) {
	return s.store.Counts(ctx)
}

func (s *Service) Trash(ctx context.Context) ([]domain.Todo, error) {
	return s.store.Trash(ctx)
}
