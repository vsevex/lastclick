package game

import (
	"fmt"
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
	StarsSpent   int64
	EliminatedAt int
	Efficiency   float64
	ShardsEarned int64
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
		p.LastPulseAt = time.Time{} // zero; we track via tick numbers
	}

	stats := make(map[int64]*SimPlayerStat, len(cfg.PlayerIDs))
	for _, pid := range cfg.PlayerIDs {
		stats[pid] = &SimPlayerStat{Alive: true}
	}

	lastPulseTick := make(map[int64]int)
	lastPulseTickForWindow := make(map[int64]int)
	for _, pid := range cfg.PlayerIDs {
		lastPulseTickForWindow[pid] = 0 // survival starts at tick 0
	}

	var events []SimEvent
	silent := cfg.SilentMode
	marginRatio := 0.0
	volMul := 1.0
	minPulseGap := int(500*time.Millisecond/simTickRate) + 1 // 500ms / 250ms = 2 ticks, need gap > 2 → ≥ 3 actually ceil(500/250)=2

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

		// 2. Process pulses
		if pulses, ok := cfg.PulseSchedule[tick]; ok {
			for _, pid := range pulses {
				st := stats[pid]
				if !st.Alive {
					continue
				}
				if last, ok := lastPulseTick[pid]; ok && (tick-last) < minPulseGap {
					continue
				}

				// Pulse accepted
				st.PulseCount++
				st.StarsSpent++
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

		// 4. Pulse window check
		pulseWindowTicks := int(cfg.Tier.PulseWindow / simTickRate)
		for _, pid := range cfg.PlayerIDs {
			st := stats[pid]
			if !st.Alive {
				continue
			}
			ticksSincePulse := tick - lastPulseTickForWindow[pid]
			if ticksSincePulse > pulseWindowTicks {
				st.Alive = false
				st.EliminatedAt = tick
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

	// Compute final stats
	result.Events = events
	result.FinalTimer = r.GlobalTimer
	result.FinalMargin = marginRatio
	result.FinalVolMul = volMul

	pool := int64(len(cfg.PlayerIDs)) * cfg.Tier.EntryCost

	for _, pid := range cfg.PlayerIDs {
		st := stats[pid]
		survivalTicks := result.TotalTicks
		if st.EliminatedAt > 0 {
			survivalTicks = st.EliminatedAt
		}
		survivalDur := time.Duration(survivalTicks) * simTickRate
		st.Efficiency = SurvivalEfficiency(survivalDur, volMul, st.StarsSpent)

		if pid == result.WinnerID {
			st.ShardsEarned = 0 // winner gets payout, not shards
		} else {
			st.ShardsEarned = ShardsFromBurn(st.StarsSpent, volMul)
		}
	}

	// Verify payout
	if result.WinnerID != 0 && !silent {
		events = append(events, SimEvent{
			Type:   "finish",
			Detail: fmt.Sprintf("winner=%d payout=%d rake=%d pool=%d", result.WinnerID, WinnerPayout(pool), RakeAmount(pool), pool),
		})
		result.Events = events
	}

	return result
}
