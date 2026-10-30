package geo

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestACancelledWaitConsumesNoSlot(t *testing.T) {
	clock := newFakeClock()
	cancelWhileSleeping := false
	var cancelSleeper context.CancelFunc
	l := newLimiter(time.Second, clock.Now, func(ctx context.Context, d time.Duration) error {
		if cancelWhileSleeping {
			cancelSleeper()
		}
		return clock.Sleep(ctx, d)
	})
	started := clock.Now()

	first, err := l.Wait(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	first.Start()

	// Cancelled while another caller holds the gate.
	gateCtx, cancelGate := context.WithCancel(context.Background())
	waited := make(chan error)
	go func() {
		_, err := l.Wait(gateCtx)
		waited <- err
	}()
	cancelGate()
	if err := <-waited; !errors.Is(err, context.Canceled) {
		t.Fatalf("wait behind the gate returned %v, want context.Canceled", err)
	}
	first.Done()

	// Cancelled during the pause before its start.
	sleepCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cancelSleeper = cancel
	cancelWhileSleeping = true
	if _, err := l.Wait(sleepCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("wait during the pause returned %v, want context.Canceled", err)
	}
	cancelWhileSleeping = false

	next, err := l.Wait(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	next.Start()
	next.Done()
	if got := clock.Now().Sub(started); got != time.Second {
		t.Errorf("the next request started %v after the first, want exactly 1s", got)
	}
}

// failingOnce lets the first request fail at the transport, as an unreachable Nominatim
// would, and records the fake time at which every request was attempted.
type failingOnce struct {
	clock *fakeClock
	next  http.RoundTripper
	calls atomic.Int32
	at    []time.Time
}

func (f *failingOnce) RoundTrip(r *http.Request) (*http.Response, error) {
	f.at = append(f.at, f.clock.Now())
	if f.calls.Add(1) == 1 {
		return nil, errors.New("connection refused")
	}
	return f.next.RoundTrip(r)
}

func TestAFailedRequestStillHoldsOffTheNextOneForASecond(t *testing.T) {
	t.Run("error status", func(t *testing.T) {
		clock := newFakeClock()
		var calls atomic.Int32
		ok := serveFixtures(t)
		up := newUpstream(t, clock, func(w http.ResponseWriter, r *http.Request) {
			if calls.Add(1) == 1 {
				http.Error(w, "overloaded", http.StatusServiceUnavailable)
				return
			}
			ok(w, r)
		})
		g := newTestGeocoder(t, clock, up.URL, openDB(t))

		if _, err := g.Search(context.Background(), "Saalfelden"); !errors.Is(err, ErrUpstream) {
			t.Fatalf("first search error = %v, want ErrUpstream", err)
		}
		if _, err := g.Search(context.Background(), "Saalfelden"); err != nil {
			t.Fatal(err)
		}
		reqs := up.Requests()
		if len(reqs) != 2 {
			t.Fatalf("upstream saw %d requests, want 2", len(reqs))
		}
		if gap := reqs[1].At.Sub(reqs[0].At); gap != time.Second {
			t.Errorf("retry started %v after the failed request, want 1s", gap)
		}
	})

	t.Run("unreachable", func(t *testing.T) {
		clock := newFakeClock()
		up := newUpstream(t, clock, serveFixtures(t))
		g := newTestGeocoder(t, clock, up.URL, openDB(t))
		rt := &failingOnce{clock: clock, next: http.DefaultTransport}
		g.http = &http.Client{Transport: rt}

		if _, err := g.Search(context.Background(), "Saalfelden"); !errors.Is(err, ErrUpstream) {
			t.Fatalf("first search error = %v, want ErrUpstream", err)
		}
		if _, err := g.Reverse(context.Background(), 47.4259, 12.8467); err != nil {
			t.Fatal(err)
		}
		if len(rt.at) != 2 {
			t.Fatalf("%d attempts, want 2", len(rt.at))
		}
		if gap := rt.at[1].Sub(rt.at[0]); gap != time.Second {
			t.Errorf("next request started %v after the failed one, want 1s", gap)
		}
	})
}
