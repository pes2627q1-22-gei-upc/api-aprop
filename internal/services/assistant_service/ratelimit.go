package assistant_service

import (
	"sync"
	"time"
)

const (
	rateLimitMessages = 20
	rateLimitWindow   = time.Minute
)

// rateLimiter allows rateLimitMessages messages per user in a sliding rateLimitWindow.
// shortcut: in memory, so it resets on restart and is not shared between instances; use a shared store if the API scales out.
type rateLimiter struct {
	mu   sync.Mutex
	sent map[string][]time.Time
}

// allow records a message of the user and reports whether it is within the limit.
func (l *rateLimiter) allow(userID string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.sent == nil {
		l.sent = map[string][]time.Time{}
	}
	recent := l.sent[userID][:0]
	for _, t := range l.sent[userID] {
		if now.Sub(t) < rateLimitWindow {
			recent = append(recent, t)
		}
	}
	if len(recent) >= rateLimitMessages {
		l.sent[userID] = recent
		return false
	}
	l.sent[userID] = append(recent, now)
	return true
}
