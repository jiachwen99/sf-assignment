// Package events fans changes out to connected clients.
//
// Subscribers are held in memory, so this works for a single instance only. Two
// processes behind a load balancer would not see each other's changes, and the
// answer at that point is a shared broker rather than a bigger map. Saying so is
// cheaper than pretending otherwise.
package events

import "sync"

// What changed, not what it changed to. The client refetches, so a payload
// carrying the new row would only be a second source of truth that can disagree
// with the first.
type Change struct {
	TodoID int64  `json:"todoId"`
	Kind   string `json:"kind"`
}

type Hub struct {
	mu   sync.RWMutex
	next int64
	subs map[int64]chan Change
}

func NewHub() *Hub {
	return &Hub{subs: make(map[int64]chan Change)}
}

// Subscribe returns a channel and the function that releases it. The caller
// must call it, or the subscription leaks for as long as the process lives.
func (h *Hub) Subscribe() (<-chan Change, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	id := h.next
	h.next++

	// Buffered, so one slow reader cannot stall the publisher. A full buffer
	// drops changes for that subscriber alone, which is survivable precisely
	// because the client refetches rather than applying the payload.
	ch := make(chan Change, 16)
	h.subs[id] = ch

	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if existing, ok := h.subs[id]; ok {
			delete(h.subs, id)
			close(existing)
		}
	}
}

// Publish is called after the transaction commits, never inside it. A change
// fanned out from inside one can reach a client, which refetches and reads the
// old data, before the change it describes is visible to anybody else.
func (h *Hub) Publish(c Change) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, ch := range h.subs {
		select {
		case ch <- c:
		default: // Slow consumer. Drop rather than block the writer.
		}
	}
}

// Exposed so a test can assert that subscriptions are released rather than
// guessing, and so the health check can report it.
func (h *Hub) Subscribers() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs)
}
