package room

import (
	"sync"
	"time"
)

// Room holds the full mutable state for a single game room.
type Room struct {
	mu sync.RWMutex

	ID        string
	Type      RoomType
	Tier      TierConfig
	State     RoomState
	Pool      int64
	Players   map[int64]*PlayerState
	WinnerID  int64
	CreatedAt time.Time
	StartedAt *time.Time
	EndedAt   *time.Time

	// Survival phase fields
	GlobalTimer   time.Duration
	MarginRatio   float64 // 0..1 — 1 means liquidation
	VolatilityMul float64
}

func NewRoom(id string, roomType RoomType, tier TierConfig) *Room {
	return &Room{
		ID:            id,
		Type:          roomType,
		Tier:          tier,
		State:         StateWaiting,
		Players:       make(map[int64]*PlayerState),
		CreatedAt:     time.Now(),
		GlobalTimer:   tier.SurvivalTime,
		MarginRatio:   0,
		VolatilityMul: 1.0,
	}
}

func (r *Room) AddPlayer(id int64, username string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.State != StateWaiting {
		return false
	}
	if len(r.Players) >= r.Tier.MaxPlayers {
		return false
	}
	if _, exists := r.Players[id]; exists {
		return false
	}
	r.Players[id] = &PlayerState{
		ID:       id,
		Username: username,
		Alive:    true,
		JoinedAt: time.Now(),
	}
	r.Pool += r.Tier.EntryCost
	return true
}

func (r *Room) AliveCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, p := range r.Players {
		if p.Alive {
			count++
		}
	}
	return count
}

func (r *Room) AlivePlayers() []*PlayerState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*PlayerState
	for _, p := range r.Players {
		if p.Alive {
			out = append(out, p)
		}
	}
	return out
}

func (r *Room) Eliminate(id int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.Players[id]; ok && p.Alive {
		p.Alive = false
		now := time.Now()
		p.EliminatedAt = &now
	}
}

func (r *Room) RecordPulse(id int64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.Players[id]
	if !ok || !p.Alive {
		return false
	}
	p.PulseCount++
	p.StarsSpent++
	p.LastPulseAt = time.Now()
	return true
}

func (r *Room) PlayerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.Players)
}

func (r *Room) CanStart() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.State == StateWaiting && len(r.Players) >= r.Tier.MinPlayers
}
