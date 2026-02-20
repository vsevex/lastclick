package game

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/lastclick/lastclick/internal/room"
	"github.com/lastclick/lastclick/internal/server"
	"github.com/lastclick/lastclick/internal/volatility"
)

const tickRate = 250 * time.Millisecond

type PulseEvent struct {
	PlayerID int64
	RoomID   string
}

type roomRunner struct {
	cancel context.CancelFunc
	pulses chan PulseEvent
}

// Engine orchestrates all active game rooms.
type Engine struct {
	rooms        *room.Manager
	hub          *server.Hub
	logger       *slog.Logger
	onEnd        EndCallback
	mu           sync.Mutex
	running      map[string]*roomRunner
	pulseLimiter *PulseRateLimiter
}

type EndCallback func(r *room.Room)

func NewEngine(rooms *room.Manager, hub *server.Hub, logger *slog.Logger, onEnd EndCallback) *Engine {
	return &Engine{
		rooms:        rooms,
		hub:          hub,
		logger:       logger,
		onEnd:        onEnd,
		running:      make(map[string]*roomRunner),
		pulseLimiter: NewPulseRateLimiter(500 * time.Millisecond),
	}
}

// SetHub sets the WebSocket hub reference (used to break circular init).
func (e *Engine) SetHub(hub *server.Hub) {
	e.hub = hub
}

func (e *Engine) SubmitPulse(playerID int64, roomID string) {
	if !e.pulseLimiter.AllowPulse(playerID) {
		return
	}
	e.mu.Lock()
	rr, ok := e.running[roomID]
	e.mu.Unlock()
	if !ok {
		return
	}
	select {
	case rr.pulses <- PulseEvent{PlayerID: playerID, RoomID: roomID}:
	default:
		e.logger.Warn("pulse dropped, buffer full", "room", roomID, "player", playerID)
	}
}

func (e *Engine) StartRoom(ctx context.Context, roomID string) {
	r, ok := e.rooms.Get(roomID)
	if !ok || !r.CanStart() {
		return
	}

	e.mu.Lock()
	if _, running := e.running[roomID]; running {
		e.mu.Unlock()
		return
	}
	rCtx, cancel := context.WithCancel(ctx)
	rr := &roomRunner{
		cancel: cancel,
		pulses: make(chan PulseEvent, 256),
	}
	e.running[roomID] = rr
	e.mu.Unlock()

	r.State = room.StateActive
	now := time.Now()
	r.StartedAt = &now
	e.broadcastState(r)

	go e.runLoop(rCtx, r, rr)
}

func (e *Engine) runLoop(ctx context.Context, r *room.Room, rr *roomRunner) {
	defer func() {
		e.mu.Lock()
		delete(e.running, r.ID)
		e.mu.Unlock()
	}()

	// Create volatility feed
	var feed volatility.Feed
	if r.Type == room.RoomAlpha {
		feed = volatility.NewLiveFeed("", "", 0, 0, true, e.logger)
	} else {
		feed = volatility.NewSyntheticFeed(r.Tier.SurvivalTime)
	}

	stopFeed := make(chan struct{})
	defer close(stopFeed)
	volCh := feed.Start(stopFeed)

	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	// Brief ramp-up before survival
	rampTimer := time.NewTimer(5 * time.Second)
	select {
	case <-rampTimer.C:
	case <-ctx.Done():
		return
	}

	r.State = room.StateSurvival
	// Initialize last pulse time for all players to survival start
	survivalStart := time.Now()
	for _, p := range r.AlivePlayers() {
		p.LastPulseAt = survivalStart
	}
	e.broadcastState(r)

	tickCount := 0

	for {
		select {
		case <-ctx.Done():
			return

		case u, ok := <-volCh:
			if !ok {
				e.finishRoom(r)
				return
			}
			r.MarginRatio = u.MarginRatio
			r.VolatilityMul = VolatilityMultiplier(u.MarginRatio)

			if u.MarginRatio >= 1.0 {
				e.finishRoom(r)
				return
			}

		case <-ticker.C:
			tickCount++
			decrement := TickDecrement(tickRate, r.MarginRatio)
			r.GlobalTimer -= decrement
			if r.GlobalTimer < 0 {
				r.GlobalTimer = 0
			}

			graceDur := time.Duration(LatencyGraceTicks) * tickRate
			now := time.Now()
			for _, p := range r.AlivePlayers() {
				if now.Sub(p.LastPulseAt) > r.Tier.PulseWindow+graceDur {
					r.Eliminate(p.ID)
					e.broadcastElimination(r, p.ID)
				}
			}

			alive := r.AliveCount()
			if alive <= 1 || r.GlobalTimer <= 0 {
				e.finishRoom(r)
				return
			}

			// Broadcast tick every 4th tick (~1s) to reduce bandwidth
			if tickCount%4 == 0 {
				e.broadcastTick(r)
			}

		case pulse := <-rr.pulses:
			if r.State != room.StateSurvival {
				continue
			}
			ok, pulseAt := r.RecordPulse(pulse.PlayerID)
			if !ok {
				continue
			}
			ext := PulseExtension(r.Tier.BaseExtension, r.AliveCount())
			r.GlobalTimer += ext
			e.broadcastPulse(r, pulse.PlayerID, ext, pulseAt)
		}
	}
}

func (e *Engine) finishRoom(r *room.Room) {
	r.State = room.StateFinished
	now := time.Now()
	r.EndedAt = &now

	placements := r.Placements()
	if len(placements) > 0 {
		r.WinnerID = placements[0]
	}

	e.broadcastState(r)

	if e.onEnd != nil {
		e.onEnd(r)
	}

	go func() {
		time.Sleep(30 * time.Second)
		e.rooms.Remove(r.ID)
		e.EnsureRooms()
	}()
}

// systemSlots defines the rooms the system keeps available at all times.
var systemSlots = []struct {
	Type room.RoomType
	Tier int
}{
	{room.RoomBlitz, 1},
	{room.RoomBlitz, 2},
	{room.RoomAlpha, 3},
}

// EnsureRooms guarantees at least one waiting room per system slot.
func (e *Engine) EnsureRooms() {
	waiting := e.rooms.ListByState(room.StateWaiting)
	for _, slot := range systemSlots {
		found := false
		for _, r := range waiting {
			if r.Type == slot.Type && r.Tier.Tier == slot.Tier {
				found = true
				break
			}
		}
		if !found {
			r, err := e.rooms.Create(slot.Type, slot.Tier)
			if err == nil {
				e.logger.Info("system room created",
					"type", string(r.Type), "tier", r.Tier.Tier, "id", r.ID)
			}
		}
	}
}

// HandleMessage implements server.MessageHandler.
func (e *Engine) HandleMessage(ctx context.Context, client *server.Client, msg server.WSMessage) {
	switch msg.Type {
	case "join_room":
		var payload struct {
			RoomID string `json:"room_id"`
		}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return
		}
		r, ok := e.rooms.Get(payload.RoomID)
		if !ok {
			return
		}
		if r.AddPlayer(client.ID, "") {
			e.hub.JoinRoom(client.ID, payload.RoomID)
			e.broadcastState(r)
			if r.CanStart() {
				e.StartRoom(ctx, r.ID)
				e.EnsureRooms()
			}
		}

	case "pulse":
		if client.RoomID == "" {
			return
		}
		e.SubmitPulse(client.ID, client.RoomID)

	case "list_rooms":
		waiting := e.rooms.ListByState(room.StateWaiting)
		active := e.rooms.ListByState(room.StateActive)
		survival := e.rooms.ListByState(room.StateSurvival)
		all := append(append(waiting, active...), survival...)
		type roomInfo struct {
			ID      string `json:"id"`
			Type    string `json:"type"`
			Tier    int    `json:"tier"`
			State   string `json:"state"`
			Players int    `json:"players"`
			Pool    int64  `json:"pool"`
		}
		var list []roomInfo
		for _, rm := range all {
			list = append(list, roomInfo{
				ID:      rm.ID,
				Type:    string(rm.Type),
				Tier:    rm.Tier.Tier,
				State:   rm.State.String(),
				Players: rm.PlayerCount(),
				Pool:    rm.Pool,
			})
		}
		payload, _ := json.Marshal(list)
		e.hub.SendTo(client.ID, server.WSMessage{Type: "room_list", Payload: payload})

	}
}

func (e *Engine) broadcastState(r *room.Room) {
	payload, _ := json.Marshal(map[string]any{
		"room_id":        r.ID,
		"state":          r.State.String(),
		"type":           string(r.Type),
		"tier":           r.Tier.Tier,
		"pool":           r.Pool,
		"alive":          r.AliveCount(),
		"total":          r.PlayerCount(),
		"timer_ms":       r.GlobalTimer.Milliseconds(),
		"margin_ratio":   r.MarginRatio,
		"volatility_mul": r.VolatilityMul,
		"winner_id":      r.WinnerID,
	})
	e.hub.BroadcastRoom(r.ID, server.WSMessage{Type: "room_state", Payload: payload})
}

func (e *Engine) broadcastTick(r *room.Room) {
	payload, _ := json.Marshal(map[string]any{
		"timer_ms":       r.GlobalTimer.Milliseconds(),
		"margin_ratio":   r.MarginRatio,
		"volatility_mul": r.VolatilityMul,
		"alive":          r.AliveCount(),
	})
	e.hub.BroadcastRoom(r.ID, server.WSMessage{Type: "tick", Payload: payload})
}

func (e *Engine) broadcastElimination(r *room.Room, playerID int64) {
	payload, _ := json.Marshal(map[string]any{
		"player_id": playerID,
		"alive":     r.AliveCount(),
	})
	e.hub.BroadcastRoom(r.ID, server.WSMessage{Type: "elimination", Payload: payload})
}

func (e *Engine) broadcastPulse(r *room.Room, playerID int64, ext time.Duration, pulseAt time.Time) {
	payload, _ := json.Marshal(map[string]any{
		"player_id":      playerID,
		"extension_ms":   ext.Milliseconds(),
		"timer_ms":       r.GlobalTimer.Milliseconds(),
		"server_time_ms": pulseAt.UnixMilli(),
	})
	e.hub.BroadcastRoom(r.ID, server.WSMessage{Type: "pulse_ack", Payload: payload})
}
