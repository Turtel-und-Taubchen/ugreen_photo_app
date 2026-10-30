package thumbs

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// State is what the cache can say about one preview right now.
type State string

const (
	Ready   State = "ready"
	Pending State = "pending"
	Failed  State = "failed"
)

// Entry is the part of an index row a preview depends on.
type Entry struct {
	ID        int64
	RelPath   string
	MediaType string
	Size      int64
	MtimeNs   int64
}

type renderer interface {
	RenderTo(ctx context.Context, absSrc, mediaType, size, dstWebp string) error
}

// Service renders missing previews with a fixed number of workers, previews someone is
// looking at before background work. Every (key, size) is rendered at most once per
// process: a job is never queued twice, a request for a running job joins it, and a failed
// one is remembered.
type Service struct {
	cacheDir, photosDir string
	r                   renderer
	workers             int

	mu      sync.Mutex
	q       queue
	queued  map[string]*job     // waiting jobs by id; len is the true backlog
	running map[string]struct{} // ids a worker is rendering
	// failed remembers failures in memory too, so a job that left no marker (path escape,
	// unwritable cache, a renderer that does not write one) is reported Failed instead of
	// being re-queued by every poll. It resets with the process; then such a job gets
	// exactly one more attempt.
	failed  map[string]struct{}
	started bool
	stopped bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	// wake holds at most one pending signal, so a signal sent while every worker is busy
	// is not lost and enqueuing never blocks.
	wake chan struct{}
}

// NewService renders sources below photosDir into cacheDir with workers parallel renders
// (at least one).
func NewService(cacheDir, photosDir string, r renderer, workers int) *Service {
	return &Service{
		cacheDir:  cacheDir,
		photosDir: photosDir,
		r:         r,
		workers:   max(workers, 1),
		queued:    map[string]*job{},
		running:   map[string]struct{}{},
		failed:    map[string]struct{}{},
		wake:      make(chan struct{}, 1),
	}
}

// Start launches the workers; they stop when ctx ends or Stop is called. Only the first
// call has an effect, and none after Stop.
func (s *Service) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started || s.stopped {
		return
	}
	s.started = true
	ctx, s.cancel = context.WithCancel(ctx)
	s.wg.Add(s.workers)
	for range s.workers {
		go s.work(ctx)
	}
}

// Stop cancels the running renders and returns once every worker has exited. It is safe
// without Start and more than once; jobs still waiting are never rendered.
func (s *Service) Stop() {
	s.mu.Lock()
	s.stopped = true
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.wg.Wait()
}

// Status reports the preview of e in size. A missing preview is queued ahead of all
// background work (or moved there if it already waits) and reported Pending; an unknown
// size is Failed.
func (s *Service) Status(e Entry, size string) State {
	if _, ok := edges[size]; !ok {
		return Failed
	}
	return s.request(e, size, true)
}

// EnqueueBackground queues the missing small previews of entries behind every requested
// one, in the given order. Ready, failed, waiting and running ones are skipped.
func (s *Service) EnqueueBackground(entries []Entry) {
	for _, e := range entries {
		s.request(e, "s", false)
	}
}

func (s *Service) request(e Entry, size string, high bool) State {
	key := Key(e.RelPath, e.Size, e.MtimeNs)
	// The unlocked look answers the common case without contending with the workers.
	if st, ok := s.onDisk(key, size); ok {
		return st
	}
	id := key + "-" + size
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.failed[id]; ok {
		return Failed
	}
	if _, ok := s.running[id]; ok {
		return Pending
	}
	if j, ok := s.queued[id]; ok {
		if high {
			s.q.pushHigh(j)
		}
		return Pending
	}
	// A worker publishes its result before it leaves running, so a job that finished
	// since the unlocked look is on disk now; looking again under the lock is what keeps
	// it from being rendered a second time.
	if st, ok := s.onDisk(key, size); ok {
		return st
	}
	j := &job{id: id, entry: e, size: size, dst: Path(s.cacheDir, key, size)}
	s.queued[id] = j
	if high {
		s.q.pushHigh(j)
	} else {
		s.q.pushLow(j)
	}
	s.signal()
	return Pending
}

// onDisk reports Ready or Failed when the cache already decided; a preview wins over a
// stale marker.
func (s *Service) onDisk(key, size string) (State, bool) {
	if _, err := os.Stat(Path(s.cacheDir, key, size)); err == nil {
		return Ready, true
	}
	if _, err := os.Stat(FailedPath(s.cacheDir, key, size)); err == nil {
		return Failed, true
	}
	return "", false
}

func (s *Service) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *Service) work(ctx context.Context) {
	defer s.wg.Done()
	for {
		j := s.next(ctx)
		if j == nil {
			return
		}
		s.run(ctx, j)
	}
}

// next blocks until a job is available (and marks it running) or ctx ends (nil).
func (s *Service) next(ctx context.Context) *job {
	for {
		if ctx.Err() != nil {
			return nil
		}
		s.mu.Lock()
		j := s.q.pop()
		if j != nil {
			delete(s.queued, j.id)
			s.running[j.id] = struct{}{}
			// One signal wakes one worker; pass it on while work is left for the others.
			if len(s.queued) > 0 {
				s.signal()
			}
		}
		s.mu.Unlock()
		if j != nil {
			return j
		}
		select {
		case <-ctx.Done():
			return nil
		case <-s.wake:
		}
	}
}

// run renders one job. Errors are logged and never end the worker.
func (s *Service) run(ctx context.Context, j *job) {
	err := s.render(ctx, j)
	cancelled := err != nil && ctx.Err() != nil
	if cancelled {
		// Shutdown aborted the render; the file itself is fine, so the marker the
		// renderer leaves on every error must not make the preview permanently failed.
		_ = os.Remove(failedFor(j.dst))
	} else if err != nil {
		slog.Warn("thumbnail job failed", "path", j.entry.RelPath, "size", j.size, "err", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.running, j.id)
	if err != nil && !cancelled {
		s.failed[j.id] = struct{}{}
	}
}

func (s *Service) render(ctx context.Context, j *job) error {
	abs, err := s.source(j.entry.RelPath)
	if err != nil {
		return err
	}
	return s.r.RenderTo(ctx, abs, j.entry.MediaType, j.size, j.dst)
}

var errOutsideLibrary = errors.New("path outside the photo library")

// source resolves a library-relative path and refuses anything that would leave
// photosDir (tech spec §9), so a bad index row can never make a worker read elsewhere.
func (s *Service) source(rel string) (string, error) {
	abs := filepath.Join(s.photosDir, filepath.FromSlash(rel))
	r, err := filepath.Rel(s.photosDir, abs)
	if err != nil || r == "." || r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator)) {
		return "", errOutsideLibrary
	}
	return abs, nil
}
