package library

import (
	"context"
	"errors"
	"testing"
	"time"
)

func recv[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting")
		panic("unreachable")
	}
}

func TestScansRunAtStartOnEveryIntervalTickAndOnDemandNeverTwoAtOnce(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	armed := make(chan chan time.Time, 8)
	var running, calls, maxRunning int // touched only by the scheduler goroutine until Run returns
	run := func(ctx context.Context, progress func(done, total int)) (Result, error) {
		calls++
		running++
		maxRunning = max(maxRunning, running)
		defer func() { running-- }()
		progress(1, 2)
		started <- struct{}{}
		<-release
		if calls == 4 {
			return Result{}, errors.New("boom")
		}
		return Result{}, nil
	}
	// Each call models time.After: a fresh channel that the scheduler must not reuse.
	tick := func(time.Duration) <-chan time.Time {
		c := make(chan time.Time, 1)
		armed <- c
		return c
	}
	s := NewScheduler(run, time.Minute, tick)
	if st := s.Status(); st.State != "idle" {
		t.Fatalf("initial state %q", st.State)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()

	// start
	recv(t, started)
	if st := s.Status(); st.State != "running" || st.Trigger != "start" || st.Done != 1 || st.Total != 2 || st.StartedAt.IsZero() {
		t.Fatalf("status %+v", st)
	}
	if s.Trigger() {
		t.Fatal("Trigger during a scan must be a no-op")
	}
	release <- struct{}{}
	first := recv(t, armed)
	if st := s.Status(); st.State != "idle" || st.FinishedAt.IsZero() {
		t.Fatalf("status %+v", st)
	}

	// periodic; a tick landing on the old channel during the scan must not cause another scan
	first <- time.Time{}
	recv(t, started)
	if st := s.Status(); st.Trigger != "periodic" {
		t.Fatalf("status %+v", st)
	}
	first <- time.Time{}
	release <- struct{}{}
	recv(t, armed)

	// manual: a second immediate Trigger is rejected
	if !s.Trigger() {
		t.Fatal("Trigger while idle must be accepted")
	}
	if s.Trigger() {
		t.Fatal("second Trigger before the scan started must be rejected")
	}
	recv(t, started)
	if st := s.Status(); st.Trigger != "manual" {
		t.Fatalf("status %+v", st)
	}
	release <- struct{}{}
	next := recv(t, armed)

	// a failing scan is recorded and the loop keeps going
	next <- time.Time{}
	recv(t, started)
	release <- struct{}{}
	last := recv(t, armed)
	if st := s.Status(); st.LastError != "boom" || st.State != "idle" {
		t.Fatalf("status %+v", st)
	}
	last <- time.Time{}
	recv(t, started)
	release <- struct{}{}
	recv(t, armed)
	if st := s.Status(); st.LastError != "" {
		t.Fatalf("error not cleared: %+v", st)
	}

	cancel()
	recv(t, done)
	if calls != 5 || maxRunning != 1 {
		t.Fatalf("calls=%d maxRunning=%d", calls, maxRunning)
	}
}
