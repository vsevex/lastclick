package server

import (
	"encoding/json"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"
)

// Metrics collects basic application metrics (no Prometheus dep needed for MVP).
type Metrics struct {
	wsConnections    atomic.Int64
	activeRooms      atomic.Int64
	totalPulses      atomic.Int64
	totalRoomsPlayed atomic.Int64
	startTime        time.Time
}

func NewMetrics() *Metrics {
	return &Metrics{startTime: time.Now()}
}

func (m *Metrics) IncrWSConn()      { m.wsConnections.Add(1) }
func (m *Metrics) DecrWSConn()      { m.wsConnections.Add(-1) }
func (m *Metrics) IncrRooms()       { m.activeRooms.Add(1) }
func (m *Metrics) DecrRooms()       { m.activeRooms.Add(-1) }
func (m *Metrics) IncrPulse()       { m.totalPulses.Add(1) }
func (m *Metrics) IncrRoomsPlayed() { m.totalRoomsPlayed.Add(1) }

// ServeHTTP exposes metrics as JSON at /metrics.
func (m *Metrics) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	data := map[string]any{
		"uptime_seconds": int(time.Since(m.startTime).Seconds()),
		"ws_connections": m.wsConnections.Load(),
		"active_rooms":   m.activeRooms.Load(),
		"total_pulses":   m.totalPulses.Load(),
		"total_rooms":    m.totalRoomsPlayed.Load(),
		"goroutines":     runtime.NumGoroutine(),
		"heap_alloc_mb":  mem.HeapAlloc / 1024 / 1024,
		"sys_mb":         mem.Sys / 1024 / 1024,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
