// Author: Caden Lund
package handlers

import (
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
)

// TestALimiterRefusesTheCallPastTheLimitUntilTheWindowEnds: the limit is what stops a script from
// making accounts by the thousand, and the window ending is what lets a real person back in.
func TestALimiterRefusesTheCallPastTheLimitUntilTheWindowEnds(t *testing.T) {
	t.Parallel()
	clock := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	l := NewLimiter(2, time.Minute)
	l.now = func() time.Time { return clock }

	first, second, third := l.Allow("10.0.0.1"), l.Allow("10.0.0.1"), l.Allow("10.0.0.1")
	clock = clock.Add(59 * time.Second)
	stillInWindow := l.Allow("10.0.0.1")
	clock = clock.Add(time.Second)
	nextWindow := l.Allow("10.0.0.1")

	if first != nil || second != nil {
		t.Errorf("calls within the limit: %v, %v, want both allowed", first, second)
	}
	if !errors.Is(third, ErrTooManyRequests) || !errors.Is(stillInWindow, ErrTooManyRequests) {
		t.Errorf("calls past the limit: %v, %v, want ErrTooManyRequests", third, stillInWindow)
	}
	if nextWindow != nil {
		t.Errorf("first call of the next window: %v, want allowed", nextWindow)
	}
}

// TestALimiterCountsEachKeyOnItsOwn: one address spending its calls must not lock out another.
func TestALimiterCountsEachKeyOnItsOwn(t *testing.T) {
	t.Parallel()
	l := NewLimiter(1, time.Minute)

	mine, again, theirs := l.Allow("10.0.0.1"), l.Allow("10.0.0.1"), l.Allow("10.0.0.2")

	if mine != nil || theirs != nil {
		t.Errorf("first calls: %v, %v, want both allowed", mine, theirs)
	}
	if !errors.Is(again, ErrTooManyRequests) {
		t.Errorf("second call from one address: %v, want ErrTooManyRequests", again)
	}
}

// TestALimiterForgetsAddressesWhoseWindowEnded: without the sweep, every address that ever called
// would stay in memory for the life of the process.
func TestALimiterForgetsAddressesWhoseWindowEnded(t *testing.T) {
	t.Parallel()
	clock := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	l := NewLimiter(1, time.Minute)
	l.now = func() time.Time { return clock }
	for _, ip := range []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"} {
		err := l.Allow(ip)
		if err != nil {
			t.Fatalf("allow %s: %v", ip, err)
		}
	}

	clock = clock.Add(time.Minute)
	err := l.Allow("10.0.0.4")

	if err != nil {
		t.Fatalf("allow 10.0.0.4: %v", err)
	}
	if len(l.keys) != 1 {
		t.Errorf("keys held = %d, want 1 (the three ended windows forgotten)", len(l.keys))
	}
}

// TestANilLimiterAllowsEverything: test fixtures turn limits off by leaving them out.
func TestANilLimiterAllowsEverything(t *testing.T) {
	t.Parallel()
	var l *Limiter

	for range 100 {
		err := l.Allow("10.0.0.1")
		if err != nil {
			t.Fatalf("a nil limiter refused: %v", err)
		}
	}
}

// TestClientIPDropsThePort: every request comes from a new port, so keying by host and port would
// never limit anyone.
func TestClientIPDropsThePort(t *testing.T) {
	t.Parallel()
	cases := []struct{ addr, want string }{
		{addr: "203.0.113.7:51234", want: "203.0.113.7"},
		{addr: "[2001:db8::1]:51234", want: "2001:db8::1"},
		{addr: "203.0.113.7", want: "203.0.113.7"},
	}
	for _, c := range cases {
		got := ClientIP(connect.Peer{Addr: c.addr})
		if got != c.want {
			t.Errorf("ClientIP(%q) = %q, want %q", c.addr, got, c.want)
		}
	}
}
