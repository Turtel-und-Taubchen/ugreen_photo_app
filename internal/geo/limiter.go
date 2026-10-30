package geo

import (
	"context"
	"time"
)

// limiter spaces request starts at least interval apart, across every caller of one Geocoder.
//
// The gate is a mutex that a waiting caller can give up on (ctx). The holder keeps it from
// before its wait until after its request and cache write, so a caller queued behind an
// identical query re-checks the cache only once that answer is stored, and Nominatim never
// sees two of our requests in flight at once.
type limiter struct {
	gate     chan struct{}
	interval time.Duration
	now      func() time.Time
	sleep    func(context.Context, time.Duration) error
	last     time.Time // start of the latest request; guarded by gate
}

func newLimiter(interval time.Duration, now func() time.Time, sleep func(context.Context, time.Duration) error) *limiter {
	return &limiter{gate: make(chan struct{}, 1), interval: interval, now: now, sleep: sleep}
}

// turn is the right to start one request. Exactly one Done must follow a successful Wait.
type turn struct {
	l    *limiter
	done bool
}

// Wait returns once the caller holds the gate and interval has passed since the last start.
// On error (ctx ended) the caller holds nothing and no slot was used.
func (l *limiter) Wait(ctx context.Context) (*turn, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	select {
	case l.gate <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if !l.last.IsZero() {
		for {
			d := l.last.Add(l.interval).Sub(l.now())
			if d <= 0 {
				break
			}
			if err := l.sleep(ctx, d); err != nil {
				<-l.gate
				return nil, err
			}
		}
	}
	if err := ctx.Err(); err != nil {
		<-l.gate
		return nil, err
	}
	return &turn{l: l}, nil
}

// Start records the request start. It must be called before the request is sent, so the
// spacing holds from start to start and a request that fails still counts.
func (t *turn) Start() {
	t.l.last = t.l.now()
}

// Done releases the gate. Without a Start the slot stays unused for the next caller.
func (t *turn) Done() {
	if t.done {
		return
	}
	t.done = true
	<-t.l.gate
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
