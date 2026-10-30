package thumbs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

// fakeRenderer stands in for vipsthumbnail/ffmpeg: it records which sources it rendered,
// how many renders overlapped, can hold renders until released and can fail by name.
type fakeRenderer struct {
	hold        chan struct{}   // when set, a render waits until it is closed (or ctx ends)
	fail        map[string]bool // source base names whose render fails
	markOnError bool            // write the .failed marker on error, like Renderer does

	started chan string // receives each source base name as its render begins

	mu        sync.Mutex
	calls     []string
	active    int
	maxActive int
	returned  int
}

func newFake() *fakeRenderer {
	return &fakeRenderer{fail: map[string]bool{}, started: make(chan string, 1024)}
}

func (f *fakeRenderer) RenderTo(ctx context.Context, absSrc, _, _, dstWebp string) error {
	name := filepath.Base(absSrc)
	f.mu.Lock()
	f.calls = append(f.calls, name)
	f.active++
	f.maxActive = max(f.maxActive, f.active)
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		f.active--
		f.returned++
		f.mu.Unlock()
	}()
	f.started <- name

	var err error
	if f.hold != nil {
		select {
		case <-f.hold:
		case <-ctx.Done():
			err = ctx.Err()
		}
	}
	if err == nil && f.fail[name] {
		err = errors.New("render failed")
	}
	if err != nil {
		if f.markOnError {
			if werr := writeMarker(failedFor(dstWebp)); werr != nil {
				return werr
			}
		}
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dstWebp), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dstWebp, []byte("webp"), 0o644)
}

func (f *fakeRenderer) snapshot() (calls []string, maxActive, returned int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...), f.maxActive, f.returned
}

func newService(t *testing.T, f *fakeRenderer, workers int) (*Service, string) {
	t.Helper()
	cache := t.TempDir()
	s := NewService(cache, t.TempDir(), f, workers)
	t.Cleanup(s.Stop)
	return s, cache
}

func entry(rel string) Entry {
	return Entry{RelPath: rel, MediaType: "image", Size: 1, MtimeNs: 1}
}

func entries(rels ...string) []Entry {
	out := make([]Entry, len(rels))
	for i, rel := range rels {
		out[i] = entry(rel)
	}
	return out
}

func waitStarted(t *testing.T, f *fakeRenderer, want string) {
	t.Helper()
	select {
	case got := <-f.started:
		if got != want {
			t.Fatalf("render started for %s, want %s", got, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("render of %s never started", want)
	}
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting until %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

// renderedInOrder waits until n renders have returned and checks their start order.
func renderedInOrder(t *testing.T, f *fakeRenderer, want ...string) {
	t.Helper()
	eventually(t, fmt.Sprintf("%d renders returned", len(want)), func() bool {
		_, _, returned := f.snapshot()
		return returned >= len(want)
	})
	if calls, _, _ := f.snapshot(); !reflect.DeepEqual(calls, want) {
		t.Fatalf("render order %v, want %v", calls, want)
	}
}

func TestNeverMoreRendersAtOnceThanWorkers(t *testing.T) {
	f := newFake()
	f.hold = make(chan struct{})
	s, _ := newService(t, f, 2)
	s.EnqueueBackground(entries("1.jpg", "2.jpg", "3.jpg", "4.jpg", "5.jpg", "6.jpg"))
	s.Start(t.Context())

	eventually(t, "both workers render", func() bool {
		_, maxActive, _ := f.snapshot()
		return maxActive == 2
	})
	time.Sleep(50 * time.Millisecond) // room for a third render that must not start
	if calls, _, _ := f.snapshot(); len(calls) != 2 {
		t.Fatalf("%d renders started while both workers were busy, want 2", len(calls))
	}
	close(f.hold)
	eventually(t, "all renders returned", func() bool {
		_, _, returned := f.snapshot()
		return returned == 6
	})
	if _, maxActive, _ := f.snapshot(); maxActive != 2 {
		t.Fatalf("%d renders ran at once, want at most 2", maxActive)
	}
}

func TestARequestedThumbnailJumpsAheadOfBackgroundJobs(t *testing.T) {
	f := newFake()
	f.hold = make(chan struct{})
	s, _ := newService(t, f, 1)
	s.EnqueueBackground(entries("b1.jpg", "b2.jpg", "b3.jpg"))
	s.Start(t.Context())
	waitStarted(t, f, "b1.jpg")

	if st := s.Status(entry("visible.jpg"), "s"); st != Pending {
		t.Fatalf("Status of a missing preview = %s, want pending", st)
	}
	close(f.hold)
	renderedInOrder(t, f, "b1.jpg", "visible.jpg", "b2.jpg", "b3.jpg")
	if st := s.Status(entry("visible.jpg"), "s"); st != Ready {
		t.Fatalf("Status after the render = %s, want ready", st)
	}
}

func TestAFailedRenderDoesNotStopTheWorker(t *testing.T) {
	f := newFake()
	f.fail["bad.jpg"] = true
	f.markOnError = true
	s, _ := newService(t, f, 1)
	s.EnqueueBackground(entries("bad.jpg", "good.jpg"))
	s.Start(t.Context())

	renderedInOrder(t, f, "bad.jpg", "good.jpg")
	if st := s.Status(entry("good.jpg"), "s"); st != Ready {
		t.Fatalf("Status of the job after the failure = %s, want ready", st)
	}
	if st := s.Status(entry("bad.jpg"), "s"); st != Failed {
		t.Fatalf("Status of the failed job = %s, want failed", st)
	}
}

func TestAFailureWithoutAMarkerIsReportedFailedInsteadOfRenderedAgain(t *testing.T) {
	f := newFake()
	f.fail["bad.jpg"] = true // markOnError stays off: nothing on disk records the failure
	s, _ := newService(t, f, 1)
	s.Start(t.Context())

	s.Status(entry("bad.jpg"), "s")
	renderedInOrder(t, f, "bad.jpg")
	eventually(t, "the failure is reported", func() bool {
		return s.Status(entry("bad.jpg"), "s") == Failed
	})
	s.EnqueueBackground(entries("bad.jpg"))
	time.Sleep(20 * time.Millisecond)
	if calls, _, _ := f.snapshot(); len(calls) != 1 {
		t.Fatalf("failed preview rendered %d times, want once", len(calls))
	}
}

func TestARequestForTheRunningJobJoinsItInsteadOfRenderingAgain(t *testing.T) {
	f := newFake()
	f.hold = make(chan struct{})
	s, _ := newService(t, f, 2)
	s.Start(t.Context())

	s.Status(entry("x.jpg"), "s")
	waitStarted(t, f, "x.jpg")
	for range 3 {
		if st := s.Status(entry("x.jpg"), "s"); st != Pending {
			t.Fatalf("Status of the running job = %s, want pending", st)
		}
	}
	s.EnqueueBackground(entries("x.jpg"))
	close(f.hold)
	renderedInOrder(t, f, "x.jpg")
	time.Sleep(20 * time.Millisecond) // a duplicate would start on the idle second worker
	if st := s.Status(entry("x.jpg"), "s"); st != Ready {
		t.Fatalf("Status after the render = %s, want ready", st)
	}
	if calls, _, _ := f.snapshot(); len(calls) != 1 {
		t.Fatalf("running job rendered %d times, want once", len(calls))
	}
}

func TestAWaitingBackgroundJobIsPromotedWhenRequested(t *testing.T) {
	f := newFake()
	s, _ := newService(t, f, 1)
	s.EnqueueBackground(entries("b1.jpg", "b2.jpg", "x.jpg", "b3.jpg"))
	if st := s.Status(entry("x.jpg"), "s"); st != Pending {
		t.Fatalf("Status of a waiting job = %s, want pending", st)
	}
	s.Start(t.Context())

	renderedInOrder(t, f, "x.jpg", "b1.jpg", "b2.jpg", "b3.jpg")
}

func TestTheLastRequestedThumbnailRendersFirst(t *testing.T) {
	f := newFake()
	s, _ := newService(t, f, 1)
	for _, rel := range []string{"h1.jpg", "h2.jpg", "h3.jpg"} {
		s.Status(entry(rel), "s")
	}
	s.Status(entry("h1.jpg"), "s") // scrolled back: h1 is on screen again
	s.Start(t.Context())

	renderedInOrder(t, f, "h1.jpg", "h3.jpg", "h2.jpg")
}

func TestBackgroundJobsRenderInArrivalOrderOnce(t *testing.T) {
	f := newFake()
	s, _ := newService(t, f, 1)
	s.EnqueueBackground(entries("b1.jpg", "b2.jpg"))
	s.EnqueueBackground(entries("b3.jpg", "b1.jpg", "b4.jpg"))
	s.Start(t.Context())

	renderedInOrder(t, f, "b1.jpg", "b2.jpg", "b3.jpg", "b4.jpg")
}

func TestBackgroundSkipsReadyAndFailedPreviews(t *testing.T) {
	f := newFake()
	s, cache := newService(t, f, 1)
	ready, failed := entry("ready.jpg"), entry("failed.jpg")
	if err := writeMarker(Path(cache, Key(ready.RelPath, 1, 1), "s")); err != nil {
		t.Fatal(err)
	}
	if err := writeMarker(FailedPath(cache, Key(failed.RelPath, 1, 1), "s")); err != nil {
		t.Fatal(err)
	}
	s.EnqueueBackground([]Entry{ready, failed, entry("missing.jpg")})
	s.Start(t.Context())

	renderedInOrder(t, f, "missing.jpg")
	if got := s.Status(ready, "s"); got != Ready {
		t.Fatalf("Status of a cached preview = %s, want ready", got)
	}
	if got := s.Status(failed, "s"); got != Failed {
		t.Fatalf("Status of a marked preview = %s, want failed", got)
	}
}

func TestAnUnknownSizeIsFailedWithoutRendering(t *testing.T) {
	f := newFake()
	s, _ := newService(t, f, 1)
	s.Start(t.Context())

	if st := s.Status(entry("a.jpg"), "xl"); st != Failed {
		t.Fatalf("Status for size xl = %s, want failed", st)
	}
	time.Sleep(20 * time.Millisecond)
	if calls, _, _ := f.snapshot(); len(calls) != 0 {
		t.Fatalf("unknown size rendered: %v", calls)
	}
}

func TestAPathEscapingTheLibraryFailsWithoutRendering(t *testing.T) {
	f := newFake()
	s, _ := newService(t, f, 1)
	s.Start(t.Context())

	for _, rel := range []string{"../outside.jpg", "2024/../../outside.jpg", "..", "", "."} {
		e := entry(rel)
		eventually(t, rel+" is reported failed", func() bool { return s.Status(e, "s") == Failed })
	}
	s.Status(entry("inside.jpg"), "s")
	renderedInOrder(t, f, "inside.jpg")
}

func TestARenderAbortedByStopIsNotRecordedAsFailed(t *testing.T) {
	f := newFake()
	f.hold = make(chan struct{}) // never released: only Stop ends the render
	f.markOnError = true
	s, cache := newService(t, f, 1)
	s.Start(t.Context())
	e := entry("slow.jpg")
	s.Status(e, "s")
	waitStarted(t, f, "slow.jpg")

	stopWithin(t, s)
	if _, err := os.Stat(FailedPath(cache, Key(e.RelPath, 1, 1), "s")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("aborted render left a failure marker (stat err %v)", err)
	}
	if st := s.Status(e, "s"); st != Pending {
		t.Fatalf("Status after an aborted render = %s, want pending", st)
	}
}

func stopWithin(t *testing.T, s *Service) {
	t.Helper()
	done := make(chan struct{})
	go func() { s.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not return")
	}
}

func TestStopDoesNotHang(t *testing.T) {
	t.Run("without Start", func(t *testing.T) {
		s, _ := newService(t, newFake(), 1)
		stopWithin(t, s)
		s.Start(t.Context()) // after Stop: no workers start
		s.Status(entry("a.jpg"), "s")
		stopWithin(t, s)
	})
	t.Run("twice", func(t *testing.T) {
		s, _ := newService(t, newFake(), 3)
		s.Start(t.Context())
		stopWithin(t, s)
		stopWithin(t, s)
	})
	t.Run("while renders run and jobs wait", func(t *testing.T) {
		f := newFake()
		f.hold = make(chan struct{})
		s, _ := newService(t, f, 2)
		s.EnqueueBackground(entries("1.jpg", "2.jpg", "3.jpg", "4.jpg"))
		s.Start(t.Context())
		eventually(t, "both workers render", func() bool {
			_, maxActive, _ := f.snapshot()
			return maxActive == 2
		})
		stopWithin(t, s)
		if calls, _, returned := f.snapshot(); returned != len(calls) || len(calls) != 2 {
			t.Fatalf("after Stop: %d renders started, %d returned; want 2 and 2", len(calls), returned)
		}
	})
	t.Run("after the parent context ended", func(t *testing.T) {
		s, _ := newService(t, newFake(), 2)
		ctx, cancel := context.WithCancel(t.Context())
		s.Start(ctx)
		cancel()
		stopWithin(t, s)
	})
}

func TestConcurrentRequestsRenderEachThumbnailExactlyOnce(t *testing.T) {
	f := newFake()
	s, _ := newService(t, f, 4)
	var all []Entry
	for i := range 40 {
		all = append(all, entry(fmt.Sprintf("%02d.jpg", i)))
	}
	s.EnqueueBackground(all[20:])
	s.Start(t.Context())

	var wg sync.WaitGroup
	for g := range 8 {
		wg.Go(func() {
			for round := range 30 {
				for i := range all {
					e := all[(i+g*5+round)%len(all)]
					s.Status(e, "s")
					if g%2 == 0 {
						s.EnqueueBackground([]Entry{e})
					}
				}
			}
		})
	}
	wg.Wait()
	for _, e := range all {
		eventually(t, e.RelPath+" is ready", func() bool { return s.Status(e, "s") == Ready })
	}
	calls, maxActive, _ := f.snapshot()
	seen := map[string]int{}
	for _, c := range calls {
		seen[c]++
	}
	for name, n := range seen {
		if n != 1 {
			t.Errorf("%s rendered %d times, want once", name, n)
		}
	}
	if len(seen) != len(all) || maxActive > 4 {
		t.Fatalf("%d distinct renders (want %d), %d at once (want <= 4)", len(seen), len(all), maxActive)
	}
}
