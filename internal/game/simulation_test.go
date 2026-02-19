package game

import (
	"math"
	"testing"
	"time"

	"github.com/lastclick/lastclick/internal/room"
)

var t1 = room.Tiers[1] // 5★, 3-20 players, 5s pulse window, 3s base ext, 120s survival

// helper: every player pulses at these ticks
func everyonePulses(players []int64, ticks ...int) map[int][]int64 {
	m := make(map[int][]int64)
	for _, t := range ticks {
		m[t] = append([]int64{}, players...)
	}
	return m
}

// helper: merge pulse schedules
func mergePulses(schedules ...map[int][]int64) map[int][]int64 {
	m := make(map[int][]int64)
	for _, s := range schedules {
		for t, pids := range s {
			m[t] = append(m[t], pids...)
		}
	}
	return m
}

// helper: single player pulses at given ticks
func playerPulses(pid int64, ticks ...int) map[int][]int64 {
	m := make(map[int][]int64)
	for _, t := range ticks {
		m[t] = append(m[t], pid)
	}
	return m
}

// ---------------------------------------------------------------------------
// 1. Survival timer decrements correctly at varying margin ratios
// ---------------------------------------------------------------------------

func TestTimerDecrement(t *testing.T) {
	players := []int64{1, 2, 3}
	// No pulses → all eliminated at ~tick 21. Timer should have drained
	// by then. At margin 0.1, drain = 300ms/tick. 21 ticks = 6.3s drained.
	result := RunSimulation(SimConfig{
		Tier:          t1,
		PlayerIDs:     players,
		VolScript:     map[int]float64{1: 0.1},
		PulseSchedule: map[int][]int64{}, // nobody pulses
		MaxTicks:      30,
	})

	if result.FinalTimer >= t1.SurvivalTime {
		t.Fatalf("timer should have decreased from %v, got %v", t1.SurvivalTime, result.FinalTimer)
	}
}

func TestTimerAcceleratesWithMargin(t *testing.T) {
	players := []int64{1, 2, 3}

	// Run 20 ticks at margin 0.1
	low := RunSimulation(SimConfig{
		Tier:          t1,
		PlayerIDs:     players,
		VolScript:     map[int]float64{1: 0.1},
		PulseSchedule: everyonePulses(players, 1, 4, 7, 10, 13, 16, 19),
		MaxTicks:      20,
	})

	// Run 20 ticks at margin 0.9
	high := RunSimulation(SimConfig{
		Tier:          t1,
		PlayerIDs:     players,
		VolScript:     map[int]float64{1: 0.9},
		PulseSchedule: everyonePulses(players, 1, 4, 7, 10, 13, 16, 19),
		MaxTicks:      20,
	})

	if high.FinalTimer >= low.FinalTimer {
		t.Fatalf("higher margin should drain timer faster: low=%v high=%v", low.FinalTimer, high.FinalTimer)
	}
}

// ---------------------------------------------------------------------------
// 2. Pulse window elimination
// ---------------------------------------------------------------------------

func TestPulseWindowElimination(t *testing.T) {
	players := []int64{1, 2, 3}
	// Player 1 never pulses, should be eliminated after pulse window
	// T1 pulse window = 5s = 20 ticks
	result := RunSimulation(SimConfig{
		Tier:      t1,
		PlayerIDs: players,
		VolScript: map[int]float64{1: 0.1},
		PulseSchedule: mergePulses(
			playerPulses(2, 1, 4, 7, 10, 13, 16, 19),
			playerPulses(3, 1, 4, 7, 10, 13, 16, 19),
			// Player 1: no pulses
		),
		MaxTicks: 25,
	})

	st := result.PlayerStats[1]
	if st.Alive {
		t.Fatal("player 1 should be eliminated (no pulses)")
	}
	if st.EliminatedAt == 0 {
		t.Fatal("EliminatedAt should be set")
	}
	// Should be eliminated around tick 21 (5s / 250ms = 20 tick window)
	if st.EliminatedAt < 19 || st.EliminatedAt > 23 {
		t.Fatalf("expected elimination around tick 20, got tick %d", st.EliminatedAt)
	}
}

// ---------------------------------------------------------------------------
// 3. Late pulse — 1 tick after window
// ---------------------------------------------------------------------------

func TestLatePulse(t *testing.T) {
	players := []int64{1, 2, 3}
	pulseWindowTicks := int(t1.PulseWindow / simTickRate) // 20

	result := RunSimulation(SimConfig{
		Tier:      t1,
		PlayerIDs: players,
		VolScript: map[int]float64{1: 0.1},
		PulseSchedule: mergePulses(
			playerPulses(2, 1, 4, 7, 10, 13, 16, 19),
			playerPulses(3, 1, 4, 7, 10, 13, 16, 19),
			playerPulses(1, pulseWindowTicks+2), // 1 tick too late
		),
		MaxTicks: pulseWindowTicks + 5,
	})

	st := result.PlayerStats[1]
	if st.Alive {
		t.Fatal("player 1 should be eliminated — pulsed after window")
	}
	// Pulse at tick 22 should be rejected as dead
	found := false
	for _, e := range result.Events {
		if e.Player == 1 && e.Type == "pulse_rejected" && e.Detail == "dead" {
			found = true
		}
	}
	if !found {
		// The elimination happens before the pulse is processed in the same tick,
		// OR the pulse was at tick 22 and elimination at tick 21. Either way, player is dead.
		if st.PulseCount > 0 {
			t.Fatal("late pulse should not have counted")
		}
	}
}

// ---------------------------------------------------------------------------
// 4. Double pulse spam — rate limiter blocks rapid pulses
// ---------------------------------------------------------------------------

func TestDoublePulseSpam(t *testing.T) {
	players := []int64{1, 2, 3}

	result := RunSimulation(SimConfig{
		Tier:      t1,
		PlayerIDs: players,
		VolScript: map[int]float64{1: 0.1},
		PulseSchedule: mergePulses(
			// Player 1 spams: tick 1, 2, 3, 4, 5 (min gap = 2 ticks ≈ 500ms)
			playerPulses(1, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10),
			playerPulses(2, 1, 4, 7, 10),
			playerPulses(3, 1, 4, 7, 10),
		),
		MaxTicks: 10,
	})

	st := result.PlayerStats[1]
	// With 500ms min interval at 250ms ticks, min gap = 2 ticks.
	// Ticks 1,2,3,4,5,6,7,8,9,10 → accepted at 1,3,5,7,9 = 5 pulses
	if st.PulseCount >= 10 {
		t.Fatalf("rate limiter should have blocked spam: got %d pulses from 10 attempts", st.PulseCount)
	}
	if st.PulseCount > 6 {
		t.Fatalf("expected ~5 accepted pulses, got %d", st.PulseCount)
	}
}

// ---------------------------------------------------------------------------
// 5. Simultaneous eliminations — multiple players expire same tick
// ---------------------------------------------------------------------------

func TestSimultaneousElimination(t *testing.T) {
	players := []int64{1, 2, 3, 4, 5}
	// Only player 5 pulses; players 1-4 never pulse → all eliminated same tick
	result := RunSimulation(SimConfig{
		Tier:          t1,
		PlayerIDs:     players,
		VolScript:     map[int]float64{1: 0.1},
		PulseSchedule: playerPulses(5, 1, 4, 7, 10, 13, 16, 19, 22, 25),
		MaxTicks:      30,
	})

	// All 4 should be eliminated at the same tick
	elimTick := result.PlayerStats[1].EliminatedAt
	for _, pid := range []int64{2, 3, 4} {
		if result.PlayerStats[pid].EliminatedAt != elimTick {
			t.Fatalf("players should be eliminated simultaneously: p1@%d p%d@%d",
				elimTick, pid, result.PlayerStats[pid].EliminatedAt)
		}
	}

	if result.WinnerID != 5 {
		t.Fatalf("player 5 should win, got %d", result.WinnerID)
	}
	if result.FinishReason != "last_alive" {
		t.Fatalf("expected last_alive, got %s", result.FinishReason)
	}
}

// ---------------------------------------------------------------------------
// 6. Last-click race — two players left, one pulses, other doesn't
// ---------------------------------------------------------------------------

func TestLastClickRace(t *testing.T) {
	// 3 players. Player 1 stops pulsing early; players 2 & 3 survive.
	// Then player 2 stops → player 3 wins.
	players := []int64{1, 2, 3}
	pw := int(t1.PulseWindow / simTickRate) // 20 ticks

	result := RunSimulation(SimConfig{
		Tier:      t1,
		PlayerIDs: players,
		VolScript: map[int]float64{1: 0.1},
		PulseSchedule: mergePulses(
			playerPulses(1, 1, 4, 7, 10),                             // stops at tick 10 → eliminated ~tick 31
			playerPulses(2, 1, 4, 7, 10, 13, 16, 19, 22, 25, 28, 31), // stops at tick 31 → eliminated ~tick 52
			playerPulses(3, 1, 4, 7, 10, 13, 16, 19, 22, 25, 28, 31, 34, 37, 40, 43, 46, 49, 52, 55),
		),
		MaxTicks: pw*3 + 10,
	})

	// Player 1 should be eliminated first
	if result.PlayerStats[1].Alive {
		t.Fatal("player 1 should be eliminated")
	}
	// Player 2 should be eliminated second
	if result.PlayerStats[2].Alive {
		t.Fatal("player 2 should be eliminated")
	}
	// Player 3 wins
	if result.WinnerID != 3 {
		t.Fatalf("player 3 should win, got %d", result.WinnerID)
	}
	// Elimination order
	if result.PlayerStats[1].EliminatedAt >= result.PlayerStats[2].EliminatedAt {
		t.Fatal("player 1 should be eliminated before player 2")
	}
}

// ---------------------------------------------------------------------------
// 7. Liquidation trigger — margin ratio hits 1.0
// ---------------------------------------------------------------------------

func TestLiquidationTrigger(t *testing.T) {
	players := []int64{1, 2, 3}

	result := RunSimulation(SimConfig{
		Tier:      t1,
		PlayerIDs: players,
		VolScript: map[int]float64{
			1:  0.2,
			5:  0.5,
			10: 0.8,
			15: 1.0, // liquidation
		},
		PulseSchedule: everyonePulses(players, 1, 4, 7, 10, 13),
		MaxTicks:      20,
	})

	if result.FinishReason != "liquidation" {
		t.Fatalf("expected liquidation finish, got %s", result.FinishReason)
	}
	if result.TotalTicks != 15 {
		t.Fatalf("expected finish at tick 15, got %d", result.TotalTicks)
	}
}

// ---------------------------------------------------------------------------
// 8. Timer reaches zero
// ---------------------------------------------------------------------------

func TestTimerZero(t *testing.T) {
	players := []int64{1, 2, 3}
	// Players pulse as late as possible (every 19 ticks, just within the
	// 20-tick pulse window). Between pulse rounds the timer drains heavily
	// while only getting baseExtension back once.
	// At margin 0.95: drain = 725ms/tick. Over 19 ticks = 13.775s drained.
	// Extension from 3 players: base 3s total. Net = -10.775s/round.
	// 120s / 10.775s ≈ 12 rounds ≈ 228 ticks to drain.
	pulses := make(map[int][]int64)
	for tick := 1; tick <= 300; tick += 19 {
		pulses[tick] = players
	}

	result := RunSimulation(SimConfig{
		Tier:          t1,
		PlayerIDs:     players,
		VolScript:     map[int]float64{1: 0.95},
		PulseSchedule: pulses,
		MaxTicks:      400,
	})

	if result.FinishReason != "timer_zero" {
		t.Fatalf("expected timer_zero, got %s (ticks=%d timer=%v)", result.FinishReason, result.TotalTicks, result.FinalTimer)
	}
	if result.FinalTimer > 0 {
		t.Fatalf("final timer should be 0, got %v", result.FinalTimer)
	}
}

// ---------------------------------------------------------------------------
// 9. Efficiency formula
// ---------------------------------------------------------------------------

func TestEfficiencyCalculation(t *testing.T) {
	// Efficiency = (timeSurvivedSec * volMul) / starsSpent
	tests := []struct {
		survived time.Duration
		volMul   float64
		spent    int64
		want     float64
	}{
		{10 * time.Second, 1.0, 5, 2.0},
		{60 * time.Second, 2.0, 10, 12.0},
		{30 * time.Second, 5.0, 1, 150.0},
		{10 * time.Second, 1.0, 0, 0.0}, // zero spent → 0
	}
	for _, tt := range tests {
		got := SurvivalEfficiency(tt.survived, tt.volMul, tt.spent)
		if math.Abs(got-tt.want) > 0.001 {
			t.Errorf("SurvivalEfficiency(%v, %.1f, %d) = %.3f, want %.3f",
				tt.survived, tt.volMul, tt.spent, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// 10. Shard conversion
// ---------------------------------------------------------------------------

func TestShardConversion(t *testing.T) {
	// ShardsFromBurn: ratio = 0.4 + 0.2*min(1, (volMul-1)/4)
	tests := []struct {
		spent  int64
		volMul float64
		want   int64
	}{
		{100, 1.0, 40}, // ratio=0.4 → 40
		{100, 3.0, 50}, // ratio=0.4+0.2*0.5=0.5 → 50
		{100, 5.0, 60}, // ratio=0.4+0.2*1.0=0.6 → 60
		{0, 2.0, 0},    // zero spent → 0
		{10, 1.0, 4},   // 10*0.4=4
	}
	for _, tt := range tests {
		got := ShardsFromBurn(tt.spent, tt.volMul)
		if got != tt.want {
			t.Errorf("ShardsFromBurn(%d, %.1f) = %d, want %d",
				tt.spent, tt.volMul, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// 11. Rake and payout
// ---------------------------------------------------------------------------

func TestRakeAndPayout(t *testing.T) {
	tests := []struct {
		pool       int64
		wantRake   int64
		wantPayout int64
	}{
		{100, 10, 90},
		{15, 1, 14}, // 15/10 = 1
		{1000, 100, 900},
	}
	for _, tt := range tests {
		rake := RakeAmount(tt.pool)
		payout := WinnerPayout(tt.pool)
		if rake != tt.wantRake {
			t.Errorf("RakeAmount(%d) = %d, want %d", tt.pool, rake, tt.wantRake)
		}
		if payout != tt.wantPayout {
			t.Errorf("WinnerPayout(%d) = %d, want %d", tt.pool, payout, tt.wantPayout)
		}
	}
}

// ---------------------------------------------------------------------------
// 12. VolatilityMultiplier curve
// ---------------------------------------------------------------------------

func TestVolatilityMultiplier(t *testing.T) {
	// VM = 1 + 4*mr^3
	tests := []struct {
		mr   float64
		want float64
	}{
		{0.0, 1.0},
		{0.5, 1.5},     // 1 + 4*0.125 = 1.5
		{1.0, 5.0},     // 1 + 4*1.0 = 5.0
		{0.75, 2.6875}, // 1 + 4*0.421875
	}
	for _, tt := range tests {
		got := VolatilityMultiplier(tt.mr)
		if math.Abs(got-tt.want) > 0.0001 {
			t.Errorf("VolatilityMultiplier(%.2f) = %.4f, want %.4f", tt.mr, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// 13. Pulse extension diminishing returns
// ---------------------------------------------------------------------------

func TestPulseExtensionDiminishing(t *testing.T) {
	base := 3 * time.Second
	// ext = base / alive
	tests := []struct {
		alive int
		want  time.Duration
	}{
		{10, 300 * time.Millisecond},
		{5, 600 * time.Millisecond},
		{2, 1500 * time.Millisecond},
		{1, 3 * time.Second},
		{0, 0},
	}
	for _, tt := range tests {
		got := PulseExtension(base, tt.alive)
		if got != tt.want {
			t.Errorf("PulseExtension(3s, %d) = %v, want %v", tt.alive, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// 14. TickDecrement acceleration
// ---------------------------------------------------------------------------

func TestTickDecrement(t *testing.T) {
	tick := 250 * time.Millisecond
	tests := []struct {
		mr   float64
		want time.Duration
	}{
		{0.0, 250 * time.Millisecond},             // 250 * (1+0) = 250
		{0.5, time.Duration(float64(tick) * 2.0)}, // 250 * (1+1.0) = 500
		{1.0, time.Duration(float64(tick) * 3.0)}, // 250 * (1+2.0) = 750
	}
	for _, tt := range tests {
		got := TickDecrement(tick, tt.mr)
		if got != tt.want {
			t.Errorf("TickDecrement(250ms, %.1f) = %v, want %v", tt.mr, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// 15. Determinism — same config always produces same result
// ---------------------------------------------------------------------------

func TestDeterminism(t *testing.T) {
	cfg := SimConfig{
		Tier:      t1,
		PlayerIDs: []int64{1, 2, 3, 4, 5},
		VolScript: map[int]float64{1: 0.1, 50: 0.3, 100: 0.6, 150: 0.9},
		PulseSchedule: mergePulses(
			playerPulses(1, 1, 4, 7, 10, 13, 16, 19),
			playerPulses(2, 1, 4, 7, 10, 13, 16, 19, 22, 25, 28, 31, 34, 37, 40),
			playerPulses(3, 1, 4, 7, 10, 13, 16, 19, 22, 25, 28, 31, 34, 37, 40, 43, 46, 49),
			playerPulses(4, 1, 4, 7, 10, 13, 16, 19, 22, 25, 28, 31, 34, 37, 40, 43, 46, 49, 52, 55, 58),
			playerPulses(5, 1, 4, 7, 10, 13, 16, 19, 22, 25, 28, 31, 34, 37, 40, 43, 46, 49, 52, 55, 58, 61, 64, 67, 70),
		),
		MaxTicks: 200,
	}

	a := RunSimulation(cfg)
	b := RunSimulation(cfg)

	if a.WinnerID != b.WinnerID {
		t.Fatalf("determinism broken: winner %d vs %d", a.WinnerID, b.WinnerID)
	}
	if a.TotalTicks != b.TotalTicks {
		t.Fatalf("determinism broken: ticks %d vs %d", a.TotalTicks, b.TotalTicks)
	}
	if a.FinishReason != b.FinishReason {
		t.Fatalf("determinism broken: reason %s vs %s", a.FinishReason, b.FinishReason)
	}
	if a.FinalTimer != b.FinalTimer {
		t.Fatalf("determinism broken: timer %v vs %v", a.FinalTimer, b.FinalTimer)
	}
	if len(a.Events) != len(b.Events) {
		t.Fatalf("determinism broken: events %d vs %d", len(a.Events), len(b.Events))
	}
	for i := range a.Events {
		if a.Events[i].Tick != b.Events[i].Tick || a.Events[i].Type != b.Events[i].Type || a.Events[i].Player != b.Events[i].Player {
			t.Fatalf("determinism broken at event %d: %+v vs %+v", i, a.Events[i], b.Events[i])
		}
	}
}

// ---------------------------------------------------------------------------
// 16. Room reset after finish
// ---------------------------------------------------------------------------

func TestRoomFinishState(t *testing.T) {
	players := []int64{1, 2, 3}
	result := RunSimulation(SimConfig{
		Tier:      t1,
		PlayerIDs: players,
		VolScript: map[int]float64{1: 0.1},
		PulseSchedule: mergePulses(
			playerPulses(3, 1, 4, 7, 10, 13, 16, 19, 22, 25, 28, 31, 34),
			// 1 and 2 don't pulse → eliminated
		),
		MaxTicks: 40,
	})

	if result.WinnerID != 3 {
		t.Fatalf("expected winner 3, got %d", result.WinnerID)
	}
	if result.FinishReason != "last_alive" {
		t.Fatalf("expected last_alive, got %s", result.FinishReason)
	}

	// Verify all non-winners are dead
	for _, pid := range []int64{1, 2} {
		if result.PlayerStats[pid].Alive {
			t.Fatalf("player %d should be dead", pid)
		}
	}
	// Winner is alive
	if !result.PlayerStats[3].Alive {
		t.Fatal("winner should be alive")
	}
}

// ---------------------------------------------------------------------------
// 17. War chest contribution
// ---------------------------------------------------------------------------

func TestWarChestContribution(t *testing.T) {
	tests := []struct {
		rake int64
		want int64
	}{
		{100, 3},
		{1000, 30},
		{10, 0}, // 10*3/100 = 0 (int truncation)
		{33, 0}, // 33*3/100 = 0
		{34, 1}, // 34*3/100 = 1
	}
	for _, tt := range tests {
		got := WarChestContribution(tt.rake)
		if got != tt.want {
			t.Errorf("WarChestContribution(%d) = %d, want %d", tt.rake, got, tt.want)
		}
	}
}
