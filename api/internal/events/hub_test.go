package events

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func receive(t *testing.T, ch <-chan Change) Change {
	t.Helper()
	select {
	case c := <-ch:
		return c
	case <-time.After(time.Second):
		t.Fatal("no change arrived")
		return Change{}
	}
}

func TestEverySubscriberGetsTheChange(t *testing.T) {
	hub := NewHub()

	first, releaseFirst := hub.Subscribe()
	second, releaseSecond := hub.Subscribe()
	defer releaseFirst()
	defer releaseSecond()

	hub.Publish(Change{TodoID: 7, Kind: "updated"})

	require.Equal(t, Change{TodoID: 7, Kind: "updated"}, receive(t, first))
	require.Equal(t, Change{TodoID: 7, Kind: "updated"}, receive(t, second))
}

// The leak this exists to catch: a handler that returns without releasing
// leaves the subscription in the map for the life of the process.
func TestReleasingRemovesTheSubscriber(t *testing.T) {
	hub := NewHub()
	require.Zero(t, hub.Subscribers())

	_, release := hub.Subscribe()
	require.Equal(t, 1, hub.Subscribers())

	release()
	require.Zero(t, hub.Subscribers())

	// Releasing twice is what a deferred release plus an explicit one looks
	// like, and closing a closed channel would panic.
	require.NotPanics(t, release)
}

func TestAReleasedChannelIsClosed(t *testing.T) {
	hub := NewHub()
	ch, release := hub.Subscribe()
	release()

	_, open := <-ch
	require.False(t, open, "a released subscriber can tell it is finished")
}

// One reader that never reads must not stop everybody else, which is what the
// buffer and the default case are for.
func TestASlowSubscriberDoesNotStallThePublisher(t *testing.T) {
	hub := NewHub()

	_, releaseSlow := hub.Subscribe() // never read from
	defer releaseSlow()
	fast, releaseFast := hub.Subscribe()
	defer releaseFast()

	done := make(chan struct{})
	go func() {
		for i := range 100 {
			hub.Publish(Change{TodoID: int64(i), Kind: "updated"})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("publishing blocked on a subscriber that never reads")
	}

	require.NotEmpty(t, fast, "the fast subscriber still received changes")
	require.Equal(t, int64(0), receive(t, fast).TodoID)
}

// Subscribing and releasing from several goroutines at once, because the hub is
// reached from every request handler.
func TestConcurrentSubscribeAndPublish(t *testing.T) {
	hub := NewHub()

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, release := hub.Subscribe()
			hub.Publish(Change{TodoID: 1, Kind: "updated"})
			release()
		}()
	}
	wg.Wait()

	require.Zero(t, hub.Subscribers(), "every subscription was released")
}
