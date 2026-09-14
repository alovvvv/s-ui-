package service

import (
	"sync"
	"time"
)

// A panel is normally reachable from the whole internet, and the only thing
// between an attacker and the admin account is one password. bcrypt makes each
// attempt cost milliseconds, which is not a rate limit: a single host can still
// run thousands of guesses an hour, and nothing in the log distinguishes that
// from ordinary traffic.
//
// Attempts are counted per source address. They are not counted per username,
// because that would let anyone lock the operator out of their own panel by
// guessing at their username from somewhere else.
const (
	maxLoginFailures = 10
	loginLockout     = 10 * time.Minute
	loginFailureTTL  = 30 * time.Minute
)

type loginFailures struct {
	count    int
	last     time.Time
	lockedAt time.Time
}

var (
	loginMu       sync.Mutex
	loginAttempts = map[string]*loginFailures{}
	lastSweep     time.Time
)

// LoginLockedOut reports whether this address has failed often enough recently
// to be held off, and for how much longer.
func LoginLockedOut(remoteIP string) (bool, time.Duration) {
	loginMu.Lock()
	defer loginMu.Unlock()
	return lockedOutLocked(remoteIP, time.Now())
}

func lockedOutLocked(remoteIP string, now time.Time) (bool, time.Duration) {
	f := loginAttempts[remoteIP]
	if f == nil || f.lockedAt.IsZero() {
		return false, 0
	}
	if remaining := loginLockout - now.Sub(f.lockedAt); remaining > 0 {
		return true, remaining
	}
	// The lockout has run out. Clearing the count as well means the next
	// attempt starts from zero rather than re-locking on a single failure.
	delete(loginAttempts, remoteIP)
	return false, 0
}

// NoteLoginFailure records a failed attempt and reports whether it tripped the
// lockout.
func NoteLoginFailure(remoteIP string) bool {
	now := time.Now()

	loginMu.Lock()
	defer loginMu.Unlock()
	sweepLocked(now)

	f := loginAttempts[remoteIP]
	if f == nil {
		f = &loginFailures{}
		loginAttempts[remoteIP] = f
	}
	// A slow trickle of failures is not an attack. Counting only what arrives
	// inside the window stops a legitimate operator who mistypes once a week
	// from eventually being locked out.
	if !f.last.IsZero() && now.Sub(f.last) > loginFailureTTL {
		f.count = 0
	}
	f.count++
	f.last = now
	if f.count >= maxLoginFailures && f.lockedAt.IsZero() {
		f.lockedAt = now
		return true
	}
	return false
}

// NoteLoginSuccess forgets an address's failures.
func NoteLoginSuccess(remoteIP string) {
	loginMu.Lock()
	delete(loginAttempts, remoteIP)
	loginMu.Unlock()
}

// sweepLocked drops entries nothing is counting any more, so a flood from
// changing source addresses cannot grow the map without bound.
func sweepLocked(now time.Time) {
	if now.Sub(lastSweep) < loginFailureTTL {
		return
	}
	lastSweep = now
	for ip, f := range loginAttempts {
		if now.Sub(f.last) > loginFailureTTL && (f.lockedAt.IsZero() || now.Sub(f.lockedAt) > loginLockout) {
			delete(loginAttempts, ip)
		}
	}
}
