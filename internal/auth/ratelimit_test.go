package auth

import (
	"testing"
	"time"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newFakeClock() *fakeClock { return &fakeClock{t: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)} }

func TestTheSixthAttemptAfterFiveFailuresWithinAMinuteIsRefusedUntilTheLockoutEnds(t *testing.T) {
	c := newFakeClock()
	l := NewLoginLimiter(c.now)
	for i := 0; i < 5; i++ {
		if ok, _ := l.Allowed("1.2.3.4"); !ok {
			t.Fatalf("attempt %d refused too early", i+1)
		}
		l.Fail("1.2.3.4")
		c.advance(5 * time.Second)
	}
	ok, retry := l.Allowed("1.2.3.4")
	if ok || retry <= 0 || retry > time.Minute {
		t.Fatalf("sixth attempt: ok=%v retry=%v", ok, retry)
	}
	if ok, _ := l.Allowed("5.6.7.8"); !ok {
		t.Fatal("another IP must not be affected")
	}
	c.advance(retry - time.Second)
	if ok, _ := l.Allowed("1.2.3.4"); ok {
		t.Fatal("lockout ended early")
	}
	c.advance(time.Second)
	if ok, _ := l.Allowed("1.2.3.4"); !ok {
		t.Fatal("lockout did not end")
	}
}

func TestFailuresOlderThanAMinuteDoNotAddUp(t *testing.T) {
	c := newFakeClock()
	l := NewLoginLimiter(c.now)
	for i := 0; i < 4; i++ {
		l.Fail("ip")
	}
	c.advance(61 * time.Second)
	l.Fail("ip")
	if ok, _ := l.Allowed("ip"); !ok {
		t.Fatal("stale failures were counted")
	}
}

func TestASuccessfulLoginClearsTheFailureCount(t *testing.T) {
	c := newFakeClock()
	l := NewLoginLimiter(c.now)
	for i := 0; i < 4; i++ {
		l.Fail("ip")
	}
	l.Success("ip")
	l.Fail("ip")
	if ok, _ := l.Allowed("ip"); !ok {
		t.Fatal("count was not reset by the success")
	}
}

func TestASuccessDuringALockoutDoesNotLiftIt(t *testing.T) {
	c := newFakeClock()
	l := NewLoginLimiter(c.now)
	for i := 0; i < 5; i++ {
		l.Fail("ip")
	}
	l.Success("ip")
	if ok, _ := l.Allowed("ip"); ok {
		t.Fatal("lockout was lifted")
	}
}
