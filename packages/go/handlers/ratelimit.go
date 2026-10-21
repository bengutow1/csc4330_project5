// Author: Caden Lund
package handlers

import (
	"net"
	"sync"
	"time"

	"connectrpc.com/connect"
)

// Limiter caps how often one key (a client IP, or an account) may call: at most n calls in each
// window. It counts in memory, which holds because one API process serves every request. A nil
// Limiter allows everything, which is how test fixtures turn limits off.
type Limiter struct {
	n      int
	window time.Duration
	now    func() time.Time

	mu    sync.Mutex
	keys  map[string]count
	swept time.Time
}

// count is one key's calls since its window started.
type count struct {
	since time.Time
	calls int
}

// NewLimiter allows n calls per key in each window.
func NewLimiter(n int, window time.Duration) *Limiter {
	return &Limiter{n: n, window: window, now: time.Now, keys: map[string]count{}}
}

// Allow spends one of key's calls, or answers ErrTooManyRequests once the window's are spent.
func (l *Limiter) Allow(key string) error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.sweep(now)

	c := l.keys[key]
	if now.Sub(c.since) >= l.window {
		c = count{since: now}
	}
	if c.calls >= l.n {
		return ErrTooManyRequests
	}
	c.calls++
	l.keys[key] = c
	return nil
}

// sweep forgets keys whose window has ended, at most once a window, so a stream of one-off
// addresses cannot grow the map without bound.
func (l *Limiter) sweep(now time.Time) {
	if now.Sub(l.swept) < l.window {
		return
	}
	for key, c := range l.keys {
		if now.Sub(c.since) >= l.window {
			delete(l.keys, key)
		}
	}
	l.swept = now
}

// ClientIP is the address a request came from, without its port: the key the public rpcs are
// limited by. The app connects to the API directly, with no proxy in front whose forwarding
// header the API would have to trust.
func ClientIP(peer connect.Peer) string {
	host, _, err := net.SplitHostPort(peer.Addr)
	if err != nil {
		return peer.Addr
	}
	return host
}
