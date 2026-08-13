package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
)

// Every handler needs body decoding, and the alternative is the same six lines
// repeated per request type.
func decode[T any](r *http.Request) (T, error) {
	var v T
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return v, domain.Invalid("body", "Request body must be valid JSON matching the documented shape")
	}
	return v, nil
}

func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, domain.Invalid("id", "Task id must be a positive integer")
	}
	return id, nil
}
