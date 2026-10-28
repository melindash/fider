package mageos

import (
	"time"

	"github.com/patrickmn/go-cache"
)

// Limiter counts events in fixed windows held in memory. A restart resets the
// counters; that is acceptable because the database unique keys, not this limiter,
// are what prevent duplicate votes. It runs in a single Fider process.
type Limiter struct {
	store *cache.Cache
	now   func() time.Time
}

// NewLimiter creates an empty limiter.
func NewLimiter() *Limiter {
	return &Limiter{store: cache.New(time.Hour, 10*time.Minute), now: time.Now}
}

// Allow records one event for key and reports whether it stays within limit per window.
func (l *Limiter) Allow(key string, limit int, window time.Duration) bool {
	bucket := key + ":" + l.now().Truncate(window).Format(time.RFC3339)
	if err := l.store.Add(bucket, 1, window); err == nil {
		return limit >= 1
	}
	count, err := l.store.IncrementInt(bucket, 1)
	if err != nil {
		return true
	}
	return count <= limit
}

// Votes is the process-wide limiter for anonymous voting.
var Votes = NewLimiter()
