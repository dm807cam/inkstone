package app

import (
	"sync"
	"time"
)

// ipRateLimiter is a fixed-window, per-source request limiter with a hard cap on
// the number of tracked sources. It bounds unauthenticated abuse (e.g. brute
// force of the device-pairing endpoint, #34) without itself becoming a
// memory-exhaustion vector: once maxKeys distinct sources are seen within a
// window it fails open for new sources rather than growing without bound, and
// the whole table is dropped at each window boundary.
type ipRateLimiter struct {
	mu      sync.Mutex
	limit   int           // max requests per source per window
	window  time.Duration // length of a counting window
	maxKeys int           // hard cap on tracked sources per window

	windowStart time.Time
	counts      map[string]int
}

func newIPRateLimiter(limit int, window time.Duration, maxKeys int) *ipRateLimiter {
	return &ipRateLimiter{
		limit:   limit,
		window:  window,
		maxKeys: maxKeys,
		counts:  make(map[string]int),
	}
}

// allow reports whether a request from key may proceed. A nil limiter allows
// everything, so an App constructed without one (e.g. in tests) is unaffected.
func (r *ipRateLimiter) allow(key string) bool {
	if r == nil {
		return true
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if now.Sub(r.windowStart) >= r.window {
		r.windowStart = now
		r.counts = make(map[string]int)
	}
	n, tracked := r.counts[key]
	if !tracked {
		if len(r.counts) >= r.maxKeys {
			// Table full for this window: fail open rather than allocate
			// unbounded entries for (possibly spoofed) new sources.
			return true
		}
		r.counts[key] = 1
		return true
	}
	if n >= r.limit {
		return false
	}
	r.counts[key] = n + 1
	return true
}
