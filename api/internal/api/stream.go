package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Server-sent events rather than WebSockets. Changes travel one way, and SSE
// reconnects on its own without a library on either end.
func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// A proxy that buffers holds the stream until it fills, which looks exactly
	// like the feature not working rather than like a proxy setting.
	w.Header().Set("X-Accel-Buffering", "no")

	rc := http.NewResponseController(w)
	changes, release := s.hub.Subscribe()
	defer release()

	// Sent immediately so the client knows it is connected, and so anything
	// buffering in front of this is flushed once rather than at the first change.
	if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
		return
	}
	if err := rc.Flush(); err != nil {
		return
	}

	// The heartbeat keeps an intermediary from closing an idle connection, and
	// gives a client that has gone away something to fail on, which is what
	// releases its subscription.
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return

		case change, ok := <-changes:
			if !ok {
				return
			}
			body, err := json.Marshal(change)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(w, "event: change\ndata: %s\n\n", body); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}

		case <-ticker.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
		}
	}
}
