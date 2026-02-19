package game

import (
	"fmt"
	"sort"
	"time"

	"github.com/lastclick/lastclick/internal/room"
)

const simTickRate = 250 * time.Millisecond

// SimConfig fully describes a deterministic game simulation.
type SimConfig struct {
	Tier      room.TierConfig
	PlayerIDs []int64

	// VolScript maps tick number → margin ratio. Ticks not in the map keep the
	// previous value. The simulation ends with "liquidation" if any value >= 1.0.
	VolScript map[int]float64

	// PulseSchedule maps tick number → list of player IDs that pulse at that tick.
	PulseSchedule map[int][]int64

	MaxTicks   int  // safety cap; 0 defaults to 2400 (10 min at 250ms)
	SilentMode bool // skip event recording for Monte Carlo perf
}

type SimEvent struct {
	Tick   int
	Type   string // "elimination", "pulse", "pulse_rejected", "timer_zero", "liquidation", "last_alive", "finish"
	Player int64
	Detail string
}

type SimPlayerStat struct {
	Alive        bool
	PulseCount   int
	StarsSpent   int64 // = entry cost (pulses are free)
	EliminatedAt int
	Efficiency   float64
	ShardsEarned int64
	Placement    int   // 1-based ranking
	Payout       int64 // star payout for top finishers
}

type SimResult struct {
	Events       []SimEvent
	WinnerID     int64
	FinishReason string // "last_alive", "timer_zero", "liquidation", "max_ticks"
	TotalTicks   int
	FinalTimer   time.Duration
	FinalMargin  float64
	FinalVolMul  float64
	PlayerStats  map[int64]*SimPlayerStat
	Placements   []int64
}

// RunSimulation executes a fully deterministic game loop. No goroutines, no
// channels, no time.Now(). Everything is driven by discrete tick steps.
//
// Processing order per tick:
//  1. Apply volatility update (if scripted for this tick)
//  2. Process pulses (with rate limiting)
//  3. Decrement global timer
//  4. Check pulse window → eliminate expired players
//  5. Check end conditions
func RunSimulation(cfg SimConfig) SimResult {
	maxTicks := cfg.MaxTicks
	if maxTicks <= 0 {
		maxTicks = 2400
	}

	r := room.NewRoom("sim-room", room.RoomBlitz, cfg.Tier)
	for _, pid := range cfg.PlayerIDs {
		r.AddPlayer(pid, fmt.Sprintf("bot-%d", pid))
	}
	r.State = room.StateSurvival
	for _, p := range r.AlivePlayers() {
		p.LastPulseAt = time.Time{}
	}

	stats := make(map[int64]*SimPlayerStat, len(cfg.PlayerIDs))
	for _, pid := range cfg.PlayerIDs {
		stats[pid] = &SimPlayerStat{Alive: true, StarsSpent: cfg.Tier.EntryCost}
	}

	lastPulseTick := make(map[int64]int)
	lastPulseTickForWindow := make(map[int64]int)
	for _, pid := range cfg.PlayerIDs {
		lastPulseTickForWindow[pid] = 0
	}

	var events []SimEvent
	var eliminationOrder []int64
	silent := cfg.SilentMode
	marginRatio := 0.0
	volMul := 1.0
	minPulseGap := int(500*time.Millisecond/simTickRate) + 1

	result := SimResult{PlayerStats: stats}

	for tick := 1; tick <= maxTicks; tick++ {
		// 1. Volatility update
		if mr, ok := cfg.VolScript[tick]; ok {
			marginRatio = mr
			volMul = VolatilityMultiplier(mr)
			if mr >= 1.0 {
				if !silent {
					events = append(events, SimEvent{Tick: tick, Type: "liquidation", Detail: fmt.Sprintf("margin=%.4f", mr)})
				}
				result.FinishReason = "liquidation"
				result.TotalTicks = tick
				break
			}
		}

		// 2. Process pulses (free — no star cost)
		if pulses, ok := cfg.PulseSchedule[tick]; ok {
			for _, pid := range pulses {
				st := stats[pid]
				if !st.Alive {
					continue
				}
				if last, ok := lastPulseTick[pid]; ok && (tick-last) < minPulseGap {
					continue
				}

				st.PulseCount++
				lastPulseTick[pid] = tick
				lastPulseTickForWindow[pid] = tick

				ext := PulseExtension(cfg.Tier.BaseExtension, r.AliveCount())
				r.GlobalTimer += ext

				if !silent {
					events = append(events, SimEvent{
						Tick:   tick,
						Type:   "pulse",
						Player: pid,
						Detail: fmt.Sprintf("ext=%dms timer=%dms", ext.Milliseconds(), r.GlobalTimer.Milliseconds()),
					})
				}
			}
		}

		// 3. Timer decrement
		dec := TickDecrement(simTickRate, marginRatio)
		r.GlobalTimer -= dec
		if r.GlobalTimer < 0 {
			r.GlobalTimer = 0
		}

		// 4. Pulse window check (with latency grace)
		pulseWindowTicks := int(cfg.Tier.PulseWindow/simTickRate) + LatencyGraceTicks
		for _, pid := range cfg.PlayerIDs {
			st := stats[pid]
			if !st.Alive {
				continue
			}
			ticksSincePulse := tick - lastPulseTickForWindow[pid]
			if ticksSincePulse > pulseWindowTicks {
				st.Alive = false
				st.EliminatedAt = tick
				eliminationOrder = append(eliminationOrder, pid)
				r.Eliminate(pid)
				if !silent {
					events = append(events, SimEvent{
						Tick:   tick,
						Type:   "elimination",
						Player: pid,
						Detail: fmt.Sprintf("no_pulse_for=%d_ticks window=%d_ticks", ticksSincePulse, pulseWindowTicks),
					})
				}
			}
		}

		// 5. End conditions
		alive := r.AliveCount()
		if alive <= 1 {
			result.FinishReason = "last_alive"
			result.TotalTicks = tick
			for _, pid := range cfg.PlayerIDs {
				if stats[pid].Alive {
					result.WinnerID = pid
				}
			}
			if !silent {
				events = append(events, SimEvent{Tick: tick, Type: "last_alive", Player: result.WinnerID})
			}
			break
		}
		if r.GlobalTimer <= 0 {
			result.FinishReason = "timer_zero"
			result.TotalTicks = tick
			if !silent {
				events = append(events, SimEvent{Tick: tick, Type: "timer_zero"})
			}
			break
		}

		if tick == maxTicks {
			result.FinishReason = "max_ticks"
			result.TotalTicks = tick
		}
	}

	// Compute placements: alive sorted by pulse count desc, then eliminated in reverse order
	type pidCount struct {
		pid   int64
		count int
	}
	var aliveSorted []pidCount
	for _, pid := range cfg.PlayerIDs {
		if stats[pid].Alive {
			aliveSorted = append(aliveSorted, pidCount{pid, stats[pid].PulseCount})
		}
	}
	// Hash-based deterministic shuffle — latency-neutral co-survivor ranking
	tickSeed := int64(result.TotalTicks) * 7919
	mix := func(id int64) int64 {
		h := id ^ (tickSeed * 2654435761)
		h ^= h >> 16
		h *= 0x45d9f3b
		h ^= h >> 16
		return h
	}
	sort.Slice(aliveSorted, func(i, j int) bool {
		return mix(aliveSorted[i].pid) < mix(aliveSorted[j].pid)
	})
	placements := make([]int64, 0, len(cfg.PlayerIDs))
	for _, a := range aliveSorted {
		placements = append(placements, a.pid)
	}
	for i := len(eliminationOrder) - 1; i >= 0; i-- {
		placements = append(placements, eliminationOrder[i])
	}
	result.Placements = placements
	if len(placements) > 0 {
		result.WinnerID = placements[0]
	}

	// Compute final stats
	result.Events = events
	result.FinalTimer = r.GlobalTimer
	result.FinalMargin = marginRatio
	result.FinalVolMul = volMul

	pool := int64(len(cfg.PlayerIDs)) * cfg.Tier.EntryCost
	payouts := PlacementPayouts(pool, len(cfg.PlayerIDs))
	topPlaces := len(payouts)

	for i, pid := range placements {
		st := stats[pid]
		place := i + 1
		st.Placement = place

		survivalTicks := result.TotalTicks
		if st.EliminatedAt > 0 {
			survivalTicks = st.EliminatedAt
		}
		survivalDur := time.Duration(survivalTicks) * simTickRate
		st.Efficiency = SurvivalEfficiency(survivalDur, volMul, st.StarsSpent)

		if place <= topPlaces {
			for _, pp := range payouts {
				if pp.Place == place {
					st.Payout = pp.Amount
					break
				}
			}
			st.ShardsEarned = 0
		} else {
			st.ShardsEarned = ShardsForLoser(cfg.Tier.EntryCost, volMul, place)
		}
	}

	if !silent && len(placements) > 0 {
		detail := fmt.Sprintf("pool=%d rake=%d placements:", pool, RakeAmount(pool))
		for _, pp := range payouts {
			if pp.Place-1 < len(placements) {
				detail += fmt.Sprintf(" %d→%d★", pp.Place, pp.Amount)
			}
		}
		events = append(events, SimEvent{Type: "finish", Detail: detail})
		result.Events = events
	}

	return result
}
