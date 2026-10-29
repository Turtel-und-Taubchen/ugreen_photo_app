package auth

import (
	"sync"
	"time"
)

const (
	loginMaxFailures = 5
	loginWindow      = 60 * time.Second
	loginLockout     = 60 * time.Second
)

type ipState struct {
	failures    []time.Time
	lockedUntil time.Time
}

// LoginLimiter counts failed logins per IP in memory; a restart clears it, which is
// acceptable for a home-network service. Only failures count.
type LoginLimiter struct {
	mu  sync.Mutex
	now func() time.Time
	ips map[string]*ipState
}

func NewLoginLimiter(now func() time.Time) *LoginLimiter {
	if now == nil {
		now = time.Now
	}
	return &LoginLimiter{now: now, ips: map[string]*ipState{}}
}

// prune drops expired failures and forgets idle IPs. Caller holds mu.
func (l *LoginLimiter) prune(ip string, now time.Time) *ipState {
	st := l.ips[ip]
	if st == nil {
		return nil
	}
	kept := st.failures[:0]
	for _, t := range st.failures {
		if now.Sub(t) < loginWindow {
			kept = append(kept, t)
		}
	}
	st.failures = kept
	if len(kept) == 0 && !now.Before(st.lockedUntil) {
		delete(l.ips, ip)
		return nil
	}
	return st
}

// Allowed reports whether a login attempt may be evaluated; during a lockout it also
// returns how long until the lockout ends, even for correct credentials.
func (l *LoginLimiter) Allowed(ip string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if st := l.prune(ip, now); st != nil && now.Before(st.lockedUntil) {
		return false, st.lockedUntil.Sub(now)
	}
	return true, 0
}

// Fail records a failed attempt; the fifth within the window starts the lockout.
func (l *LoginLimiter) Fail(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	st := l.prune(ip, now)
	if st == nil {
		st = &ipState{}
		l.ips[ip] = st
	}
	st.failures = append(st.failures, now)
	if len(st.failures) >= loginMaxFailures {
		st.lockedUntil = now.Add(loginLockout)
		st.failures = nil
	}
}

// Success clears the failure count.
func (l *LoginLimiter) Success(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if st := l.ips[ip]; st != nil && !l.now().Before(st.lockedUntil) {
		delete(l.ips, ip)
	}
}
