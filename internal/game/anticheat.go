package game

import (
	"sync"
	"time"
)

// PulseRateLimiter prevents pulse-spamming by enforcing a minimum interval
// between pulses for each player. Server-authoritative timing.
type PulseRateLimiter struct {
	mu          sync.Mutex
	lastPulse   map[int64]time.Time
	minInterval time.Duration
}

func NewPulseRateLimiter(minInterval time.Duration) *PulseRateLimiter {
	return &PulseRateLimiter{
		lastPulse:   make(map[int64]time.Time),
		minInterval: minInterval,
	}
}

// AllowPulse returns true if enough time has passed since the player's last pulse.
func (pl *PulseRateLimiter) AllowPulse(playerID int64) bool {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	now := time.Now()
	last, ok := pl.lastPulse[playerID]
	if ok && now.Sub(last) < pl.minInterval {
		return false
	}
	pl.lastPulse[playerID] = now
	return true
}

// Reset clears tracking for a player (called when they leave a room).
func (pl *PulseRateLimiter) Reset(playerID int64) {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	delete(pl.lastPulse, playerID)
}

// ResetAll clears all tracking data.
func (pl *PulseRateLimiter) ResetAll() {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	pl.lastPulse = make(map[int64]time.Time)
}
