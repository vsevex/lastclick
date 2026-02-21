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

// AddPlayer adds a player only when join is allowed: WAITING or COUNTDOWN (active).
// SURVIVAL and FINISHED are locked so no midgame or post-round join.
func (r *Room) AddPlayer(id int64, username string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.State == StateSurvival || r.State == StateFinished {
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

// RemovePlayer removes a player from the room. Refund=true deducts entry from pool (use when leaving during WAITING).
// No refund once countdown started (volatility scouting prevention). Returns true if player was in room.
func (r *Room) RemovePlayer(id int64, refund bool) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.Players[id]
	if !ok {
		return false
	}
	delete(r.Players, id)
	if refund {
		cost := r.Tier.EntryCost
		if r.Pool >= cost {
			r.Pool -= cost
		}
	}
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

// MarkDisconnected marks a player as temporarily disconnected (app closed / connection lost). Not exit; pulse window still applies.
func (r *Room) MarkDisconnected(id int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.Players[id]; ok {
		p.Disconnected = true
	}
}

// ClearDisconnected clears the disconnected flag (player reconnected). Call only when player is still alive.
func (r *Room) ClearDisconnected(id int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.Players[id]; ok {
		p.Disconnected = false
	}
}

// ReconnectCheck decides restore vs eliminate for a reconnecting player. Server-authoritative; no mercy.
// restore=true: clear disconnected and keep in game. eliminate=true: eliminate now (missed pulse window).
func (r *Room) ReconnectCheck(id int64) (restore bool, eliminate bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.Players[id]
	if !ok || !p.Disconnected || !p.Alive {
		return false, false
	}
	switch r.State {
	case StateWaiting, StateActive:
		return true, false
	case StateSurvival:
		if time.Since(p.LastPulseAt) > r.Tier.PulseWindow {
			return false, true
		}
		return true, false
	default:
		return false, false
	}
}

// RecordPulse records a pulse at server time. Caller must be in survival phase.
// Returns (ok, pulseTimestamp). Only updates state if ok.
func (r *Room) RecordPulse(id int64) (bool, time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.Players[id]
	if !ok || !p.Alive {
		return false, time.Time{}
	}
	now := time.Now()
	p.PulseCount++
	p.LastPulseAt = now
	return true, now
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

// ResetRound puts the room back to StateWaiting for the next round. Room is not destroyed.
// Clears all players so everyone must re-enter (and pay entry). Fast re-entry loop.
func (r *Room) ResetRound() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.State != StateFinished {
		return false
	}
	r.State = StateWaiting
	r.Pool = 0
	r.WinnerID = 0
	r.StartedAt = nil
	r.EndedAt = nil
	r.EliminationOrder = nil
	r.GlobalTimer = r.Tier.SurvivalTime
	r.MarginRatio = 0
	r.VolatilityMul = 1.0
	r.Players = make(map[int64]*PlayerState)
	return true
}
