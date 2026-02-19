package room

import (
	"sort"
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

	EliminationOrder []int64

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
		r.EliminationOrder = append(r.EliminationOrder, id)
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
	p.LastPulseAt = time.Now()
	return true
}

// Placements returns player IDs ordered by finishing position.
// Alive players first (co-survivors ranked by efficiency desc, then ID asc —
// latency-neutral since efficiency is identical for co-survivors in the same room),
// then eliminated players in reverse elimination order (last eliminated = best).
func (r *Room) Placements() []int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var alive []*PlayerState
	for _, p := range r.Players {
		if p.Alive {
			alive = append(alive, p)
		}
	}
	// Hash-based deterministic shuffle — breaks ID correlation so co-survivors
	// are ranked fairly regardless of ID assignment.
	roomSeed := int64(0)
	for _, c := range r.ID {
		roomSeed = roomSeed*31 + int64(c)
	}
	mix := func(id int64) int64 {
		h := id ^ (roomSeed * 2654435761)
		h ^= h >> 16
		h *= 0x45d9f3b
		h ^= h >> 16
		return h
	}
	sort.Slice(alive, func(i, j int) bool {
		return mix(alive[i].ID) < mix(alive[j].ID)
	})
	result := make([]int64, 0, len(r.Players))
	for _, p := range alive {
		result = append(result, p.ID)
	}
	for i := len(r.EliminationOrder) - 1; i >= 0; i-- {
		result = append(result, r.EliminationOrder[i])
	}
	return result
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
