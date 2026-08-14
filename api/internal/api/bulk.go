package api

import (
	"context"
	"net/http"

	"github.com/jiachwen99/sf-assignment/api/internal/domain"
	"github.com/jiachwen99/sf-assignment/api/internal/service"
)

// Capped, because the response holds a result per item and the work is done in
// one request. Fifty is a screen of selected rows, which is what this is for.
const maxBulkItems = 50

type bulkBody struct {
	Items []service.BulkItem `json:"items"`
}

func (s *Server) bulkComplete(w http.ResponseWriter, r *http.Request) {
	s.bulk(w, r, s.svc.BulkComplete)
}

func (s *Server) bulkArchive(w http.ResponseWriter, r *http.Request) {
	s.bulk(w, r, s.svc.BulkArchive)
}

/*
 * 200 whatever the individual results are.
 *
 * The request succeeded: every item was attempted and every outcome is in the
 * body. A status code has one value and this has fifty answers, so putting a
 * failure code on a batch where forty-nine worked would be less true than 200
 * and would push callers towards discarding the whole response.
 */
func (s *Server) bulk(
	w http.ResponseWriter,
	r *http.Request,
	run func(ctx context.Context, items []service.BulkItem) []service.BulkResult,
) {
	body, err := decode[bulkBody](r)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	if len(body.Items) == 0 {
		s.fail(w, r, domain.Invalid("items", "Select at least one task"))
		return
	}
	if len(body.Items) > maxBulkItems {
		s.fail(w, r, domain.Invalid("items", "A batch can hold at most 50 tasks"))
		return
	}
	for _, item := range body.Items {
		if item.ID <= 0 || item.Version <= 0 {
			s.fail(w, r, domain.Invalid("items", "Every task needs an id and the version you last read"))
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"results": run(r.Context(), body.Items)})
}
