package service

import (
	"context"
	"errors"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
)

/*
 * Per item, not all or nothing.
 *
 * A batch of fifty where one carries a stale version should apply the other
 * forty-nine and say which one did not. One transaction around the lot would
 * discard everything for a conflict on a single row, which is a worse outcome
 * than the problem it prevents: these are independent edits that happen to have
 * been asked for together, not one change spread across fifty rows.
 *
 * Each item therefore runs through the ordinary single-task path, with the same
 * rules and the same refusals.
 */

type BulkResult struct {
	ID    int64  `json:"id"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	// Present when the refusal was a blocked transition, so the caller can say
	// what to go and finish rather than only that it failed.
	Blockers []domain.Blocker `json:"blockers,omitempty"`
}

type BulkItem struct {
	ID      int64 `json:"id"`
	Version int   `json:"version"`
}

func (s *Service) BulkComplete(ctx context.Context, items []BulkItem) []BulkResult {
	return s.each(items, func(item BulkItem) error {
		_, err := s.Complete(ctx, item.ID, item.Version)
		return err
	})
}

func (s *Service) BulkArchive(ctx context.Context, items []BulkItem) []BulkResult {
	return s.each(items, func(item BulkItem) error {
		current, err := s.store.Todo(ctx, item.ID)
		if err != nil {
			return err
		}
		_, err = s.Update(ctx, item.ID, item.Version, TodoInput{
			Name:        current.Name,
			Description: current.Description,
			DueDate:     current.DueDate,
			Status:      domain.Archived,
			Priority:    current.Priority,
			RecurUnit:   current.RecurUnit,
			RecurEvery:  current.RecurEvery,
		})
		return err
	})
}

// In the order given. Completing a blocker before its dependent releases the
// dependent on the way past, so a selection holding both succeeds where the
// reverse order would refuse the second.
func (s *Service) each(items []BulkItem, apply func(BulkItem) error) []BulkResult {
	out := make([]BulkResult, len(items))

	for i, item := range items {
		err := apply(item)
		out[i] = BulkResult{ID: item.ID, OK: err == nil}
		if err == nil {
			continue
		}

		out[i].Error = readable(err)

		var blocked *domain.BlockedError
		if errors.As(err, &blocked) {
			out[i].Blockers = blocked.Blockers
		}
	}
	return out
}

// The same sentences the single-task paths return, so a batch reports refusals
// in the words the rest of the application already uses.
func readable(err error) string {
	var (
		invalid  *domain.ValidationError
		conflict *domain.ConflictError
		blocked  *domain.BlockedError
	)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return "This task no longer exists"
	case errors.As(err, &conflict):
		return "This task was changed by someone else"
	case errors.As(err, &blocked):
		return "This task is blocked by unfinished work"
	case errors.As(err, &invalid):
		for _, reason := range invalid.Fields {
			return reason
		}
	}
	return "This task could not be updated"
}
