package library

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Status is a snapshot of the scheduler for the UI.
type Status struct {
	State      string // "idle" or "running"
	Trigger    string // "start", "periodic" or "manual"; of the running or last scan
	Done       int
	Total      int
	StartedAt  time.Time
	FinishedAt time.Time
	LastError  string
}

// Scheduler runs scans from one loop, so two scans can never overlap.
type Scheduler struct {
	run      func(ctx context.Context, progress func(done, total int)) (Result, error)
	interval time.Duration
	tick     func(time.Duration) <-chan time.Time
	trigger  chan struct{}

	mu      sync.Mutex
	pending bool // a manual trigger is queued but the loop has not started it yet
	status  Status
}

// NewScheduler: tick is re-armed after every scan, so ticks that fire during a scan are dropped.
func NewScheduler(run func(ctx context.Context, progress func(done, total int)) (Result, error), interval time.Duration, tick func(time.Duration) <-chan time.Time) *Scheduler {
	return &Scheduler{run: run, interval: interval, tick: tick, trigger: make(chan struct{}, 1), status: Status{State: "idle"}}
}

// Trigger requests a manual scan. It reports false, without queueing, when a scan is
// running or an earlier trigger is still waiting to start.
func (s *Scheduler) Trigger() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending || s.status.State == "running" {
		return false
	}
	s.pending = true
	s.trigger <- struct{}{} // cannot block: pending guards the single buffer slot
	return true
}

// Status returns a consistent snapshot.
func (s *Scheduler) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// Run scans immediately, then on every tick and trigger, until ctx is done.
func (s *Scheduler) Run(ctx context.Context) {
	s.scan(ctx, "start")
	for ctx.Err() == nil {
		tick := s.tick(s.interval)
		select {
		case <-ctx.Done():
			return
		case <-tick:
			s.scan(ctx, "periodic")
		case <-s.trigger:
			s.scan(ctx, "manual")
		}
	}
}

func (s *Scheduler) scan(ctx context.Context, trigger string) {
	s.mu.Lock()
	s.pending = false
	s.status = Status{State: "running", Trigger: trigger, StartedAt: time.Now(), LastError: s.status.LastError}
	s.mu.Unlock()

	_, err := s.run(ctx, func(done, total int) {
		s.mu.Lock()
		s.status.Done, s.status.Total = done, total
		s.mu.Unlock()
	})

	s.mu.Lock()
	s.status.State = "idle"
	s.status.FinishedAt = time.Now()
	s.status.LastError = ""
	if err != nil {
		s.status.LastError = err.Error()
	}
	s.mu.Unlock()
	if err != nil {
		slog.Error("library scan failed", "trigger", trigger, "err", err)
	}
}
