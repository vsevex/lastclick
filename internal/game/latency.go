package game

import (
	"sync"
	"time"
)

// LatencyNormalizer adjusts pulse timestamps to compensate for device-to-server latency.
// Each client reports its RTT; the server applies a fairness window so high-latency
// players aren't systematically disadvantaged.
type LatencyNormalizer struct {
	mu     sync.RWMutex
	rtts   map[int64]*rttTracker
	window time.Duration // max compensation window
}

type rttTracker struct {
	samples []time.Duration
	avg     time.Duration
}

const maxRTTSamples = 20

func NewLatencyNormalizer(window time.Duration) *LatencyNormalizer {
	return &LatencyNormalizer{
		rtts:   make(map[int64]*rttTracker),
		window: window,
	}
}

// RecordRTT records a round-trip time sample from a client ping.
func (ln *LatencyNormalizer) RecordRTT(playerID int64, rtt time.Duration) {
	ln.mu.Lock()
	defer ln.mu.Unlock()

	t, ok := ln.rtts[playerID]
	if !ok {
		t = &rttTracker{}
		ln.rtts[playerID] = t
	}

	t.samples = append(t.samples, rtt)
	if len(t.samples) > maxRTTSamples {
		t.samples = t.samples[1:]
	}

	var total time.Duration
	for _, s := range t.samples {
		total += s
	}
	t.avg = total / time.Duration(len(t.samples))
}

// AdjustedPulseTime returns the server time adjusted backward by the player's
// average one-way latency, capped by the normalization window.
func (ln *LatencyNormalizer) AdjustedPulseTime(playerID int64, serverTime time.Time) time.Time {
	ln.mu.RLock()
	defer ln.mu.RUnlock()

	t, ok := ln.rtts[playerID]
	if !ok {
		return serverTime
	}

	oneWay := t.avg / 2
	if oneWay > ln.window {
		oneWay = ln.window
	}

	return serverTime.Add(-oneWay)
}

// Cleanup removes tracking data for a player.
func (ln *LatencyNormalizer) Cleanup(playerID int64) {
	ln.mu.Lock()
	defer ln.mu.Unlock()
	delete(ln.rtts, playerID)
}
