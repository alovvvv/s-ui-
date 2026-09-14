package service

import (
	"testing"
	"time"
)

func resetLoginState() {
	loginMu.Lock()
	loginAttempts = map[string]*loginFailures{}
	lastSweep = time.Time{}
	loginMu.Unlock()
}

// bcrypt alone is not a rate limit. A single host could run guesses for as long
// as it liked, and nothing in the log told that apart from ordinary traffic.
func TestLockoutAfterRepeatedFailures(t *testing.T) {
	resetLoginState()
	const ip = "203.0.113.7"

	for i := 1; i < maxLoginFailures; i++ {
		if NoteLoginFailure(ip) {
			t.Fatalf("locked out after %d failures, before the limit of %d", i, maxLoginFailures)
		}
		if locked, _ := LoginLockedOut(ip); locked {
			t.Fatalf("locked out after %d failures", i)
		}
	}

	if !NoteLoginFailure(ip) {
		t.Fatalf("the %dth failure did not trip the lockout", maxLoginFailures)
	}
	locked, remaining := LoginLockedOut(ip)
	if !locked {
		t.Fatal("not locked out after reaching the limit")
	}
	if remaining <= 0 || remaining > loginLockout {
		t.Errorf("remaining = %v, want between 0 and %v", remaining, loginLockout)
	}
}

// The limit counts addresses, not usernames. Counting usernames would let
// anyone lock the operator out of their own panel from somewhere else.
func TestLockoutIsPerAddress(t *testing.T) {
	resetLoginState()
	for i := 0; i < maxLoginFailures; i++ {
		NoteLoginFailure("203.0.113.7")
	}
	if locked, _ := LoginLockedOut("203.0.113.7"); !locked {
		t.Fatal("the offending address was not locked out")
	}
	if locked, _ := LoginLockedOut("198.51.100.4"); locked {
		t.Error("an unrelated address was locked out")
	}
}

// A successful login clears the record, so an operator who mistypes twice and
// then gets it right does not carry those failures around.
func TestSuccessClearsFailures(t *testing.T) {
	resetLoginState()
	const ip = "203.0.113.7"
	NoteLoginFailure(ip)
	NoteLoginFailure(ip)
	NoteLoginSuccess(ip)

	loginMu.Lock()
	_, present := loginAttempts[ip]
	loginMu.Unlock()
	if present {
		t.Error("failures survived a successful login")
	}
}

// The lockout has to end by itself, and end fully: coming back with the counter
// still at the limit would re-lock on the next single failure.
func TestLockoutExpires(t *testing.T) {
	resetLoginState()
	const ip = "203.0.113.7"
	for i := 0; i < maxLoginFailures; i++ {
		NoteLoginFailure(ip)
	}

	loginMu.Lock()
	loginAttempts[ip].lockedAt = time.Now().Add(-loginLockout - time.Minute)
	loginMu.Unlock()

	if locked, _ := LoginLockedOut(ip); locked {
		t.Fatal("still locked out after the window passed")
	}
	if NoteLoginFailure(ip) {
		t.Error("a single failure after the lockout expired locked the address again")
	}
}

// Failures spread over weeks are someone mistyping, not an attack.
func TestOldFailuresDoNotAccumulate(t *testing.T) {
	resetLoginState()
	const ip = "203.0.113.7"

	for i := 0; i < maxLoginFailures-1; i++ {
		NoteLoginFailure(ip)
	}
	loginMu.Lock()
	loginAttempts[ip].last = time.Now().Add(-loginFailureTTL - time.Minute)
	loginMu.Unlock()

	if NoteLoginFailure(ip) {
		t.Error("a failure long after the previous ones tripped the lockout")
	}
}
