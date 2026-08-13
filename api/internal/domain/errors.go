package domain

import (
	"errors"
	"strings"
)

// The whole error vocabulary. These are mapped to status codes in exactly one
// place, at the HTTP edge, so nothing in between needs to wrap or re-wrap.
var ErrNotFound = errors.New("not found")

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Fields))
	for k, v := range e.Fields {
		parts = append(parts, k+": "+v)
	}
	return "invalid input (" + strings.Join(parts, ", ") + ")"
}

func Invalid(field, reason string) *ValidationError {
	return &ValidationError{Fields: map[string]string{field: reason}}
}
