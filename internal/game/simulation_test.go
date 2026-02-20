package game

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/lastclick/lastclick/internal/room"
)

var t1 = room.Tiers[1] // 5★, 3-20 players, 5s pulse window, 3s base ext, 90s survival

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
	result := RunSimulation(SimConfig{
		Tier:          t1,
		PlayerIDs:     players,
		VolScript:     map[int]float64{1: 0.1},
		PulseSchedule: map[int][]int64{},
		MaxTicks:      30,
	})

	if result.FinalTimer >= t1.SurvivalTime {
		t.Fatalf("timer should have decreased from %v, got %v", t1.SurvivalTime, result.FinalTimer)
	}
}

func TestTimerAcceleratesWithMargin(t *testing.T) {
	players := []int64{1, 2, 3}

	low := RunSimulation(SimConfig{
		Tier:          t1,
		PlayerIDs:     players,
		VolScript:     map[int]float64{1: 0.1},
		PulseSchedule: everyonePulses(players, 1, 4, 7, 10, 13, 16, 19),
		MaxTicks:      20,
	})

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
	result := RunSimulation(SimConfig{
		Tier:      t1,
		PlayerIDs: players,
		VolScript: map[int]float64{1: 0.1},
		PulseSchedule: mergePulses(
			playerPulses(2, 1, 4, 7, 10, 13, 16, 19),
			playerPulses(3, 1, 4, 7, 10, 13, 16, 19),
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
	// With LatencyGraceTicks=1, window expands by 1 tick → elimination ~tick 22
	if st.EliminatedAt < 20 || st.EliminatedAt > 24 {
		t.Fatalf("expected elimination around tick 21-22, got tick %d", st.EliminatedAt)
	}
}

// ---------------------------------------------------------------------------
// 3. Late pulse — 1 tick after window
// ---------------------------------------------------------------------------

func TestLatePulse(t *testing.T) {
	players := []int64{1, 2, 3}
	pulseWindowTicks := int(t1.PulseWindow / simTickRate) // 20

	// Pulse at pwTicks + grace + 2 → well past both window and grace buffer
	lateTick := pulseWindowTicks + LatencyGraceTicks + 2
	result := RunSimulation(SimConfig{
		Tier:      t1,
		PlayerIDs: players,
		VolScript: map[int]float64{1: 0.1},
		PulseSchedule: mergePulses(
			playerPulses(2, 1, 4, 7, 10, 13, 16, 19, 22, 25),
			playerPulses(3, 1, 4, 7, 10, 13, 16, 19, 22, 25),
			playerPulses(1, lateTick),
		),
		MaxTicks: lateTick + 5,
	})

	st := result.PlayerStats[1]
	if st.Alive {
		t.Fatal("player 1 should be eliminated — pulsed after window + grace")
	}
	if st.PulseCount > 0 {
		t.Fatal("late pulse should not have counted")
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
			playerPulses(1, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10),
			playerPulses(2, 1, 4, 7, 10),
			playerPulses(3, 1, 4, 7, 10),
		),
		MaxTicks: 10,
	})

	st := result.PlayerStats[1]
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
	result := RunSimulation(SimConfig{
		Tier:          t1,
		PlayerIDs:     players,
		VolScript:     map[int]float64{1: 0.1},
		PulseSchedule: playerPulses(5, 1, 4, 7, 10, 13, 16, 19, 22, 25),
		MaxTicks:      30,
	})

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
	players := []int64{1, 2, 3}
	pw := int(t1.PulseWindow / simTickRate)

	result := RunSimulation(SimConfig{
		Tier:      t1,
		PlayerIDs: players,
		VolScript: map[int]float64{1: 0.1},
		PulseSchedule: mergePulses(
			playerPulses(1, 1, 4, 7, 10),
			playerPulses(2, 1, 4, 7, 10, 13, 16, 19, 22, 25, 28, 31),
			playerPulses(3, 1, 4, 7, 10, 13, 16, 19, 22, 25, 28, 31, 34, 37, 40, 43, 46, 49, 52, 55),
		),
		MaxTicks: pw*3 + 10,
	})

	if result.PlayerStats[1].Alive {
		t.Fatal("player 1 should be eliminated")
	}
	if result.PlayerStats[2].Alive {
		t.Fatal("player 2 should be eliminated")
	}
	if result.WinnerID != 3 {
		t.Fatalf("player 3 should win, got %d", result.WinnerID)
	}
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
			15: 1.0,
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
	tests := []struct {
		survived time.Duration
		volMul   float64
		spent    int64
		want     float64
	}{
		{10 * time.Second, 1.0, 5, 2.0},
		{60 * time.Second, 2.0, 10, 12.0},
		{30 * time.Second, 5.0, 1, 150.0},
		{10 * time.Second, 1.0, 0, 0.0},
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
// 10. Shard conversion for losers (entry-cost based)
// ---------------------------------------------------------------------------

func TestShardsForLoser(t *testing.T) {
	tests := []struct {
		entry     int64
		volMul    float64
		placement int
		want      int64
	}{
		{5, 1.0, 6, 2},     // 5*0.4=2 base
		{5, 1.0, 4, 4},     // 2*2=4 (4th place bonus)
		{5, 1.0, 5, 3},     // 2*3/2=3 (5th place bonus)
		{100, 5.0, 10, 60}, // 100*0.6=60 base
		{100, 5.0, 4, 120}, // 60*2=120
		{100, 5.0, 5, 90},  // 60*3/2=90
		{20, 3.0, 8, 10},   // 20*0.5=10 base
	}
	for _, tt := range tests {
		got := ShardsForLoser(tt.entry, tt.volMul, tt.placement)
		if got != tt.want {
			t.Errorf("ShardsForLoser(%d, %.1f, %d) = %d, want %d",
				tt.entry, tt.volMul, tt.placement, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// 11. Rake and placement payouts (12% rake, top-3 split)
// ---------------------------------------------------------------------------

func TestRakeAndPayouts(t *testing.T) {
	// 12% rake
	tests := []struct {
		pool     int64
		wantRake int64
	}{
		{100, 12},
		{1000, 120},
		{50, 6},
	}
	for _, tt := range tests {
		rake := RakeAmount(tt.pool)
		if rake != tt.wantRake {
			t.Errorf("RakeAmount(%d) = %d, want %d", tt.pool, rake, tt.wantRake)
		}
	}

	// Top-3 payouts from pool of 100 (12 rake, 88 post-rake)
	payouts := PlacementPayouts(100, 10)
	if len(payouts) != 3 {
		t.Fatalf("expected 3 payouts, got %d", len(payouts))
	}
	// 88 * 60/100 = 52, 88 * 25/100 = 22, 88 * 15/100 = 13 → sum = 87 (1 lost to truncation)
	if payouts[0].Amount != 52 {
		t.Errorf("1st place: got %d, want 52", payouts[0].Amount)
	}
	if payouts[1].Amount != 22 {
		t.Errorf("2nd place: got %d, want 22", payouts[1].Amount)
	}
	if payouts[2].Amount != 13 {
		t.Errorf("3rd place: got %d, want 13", payouts[2].Amount)
	}

	// 2-player room: 75/25 split
	payouts2 := PlacementPayouts(100, 2)
	if len(payouts2) != 2 {
		t.Fatalf("expected 2 payouts for 2-player room, got %d", len(payouts2))
	}
	if payouts2[0].Amount != 66 { // 88*75/100
		t.Errorf("1st place (2p): got %d, want 66", payouts2[0].Amount)
	}
	if payouts2[1].Amount != 22 { // 88*25/100
		t.Errorf("2nd place (2p): got %d, want 22", payouts2[1].Amount)
	}
}

// ---------------------------------------------------------------------------
// 12. VolatilityMultiplier curve
// ---------------------------------------------------------------------------

func TestVolatilityMultiplier(t *testing.T) {
	tests := []struct {
		mr   float64
		want float64
	}{
		{0.0, 1.0},
		{0.5, 1.5},
		{1.0, 5.0},
		{0.75, 2.6875},
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
		{0.0, 250 * time.Millisecond},
		{0.5, time.Duration(float64(tick) * 2.0)},
		{1.0, time.Duration(float64(tick) * 3.0)},
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
	if len(a.Placements) != len(b.Placements) {
		t.Fatalf("determinism broken: placements %d vs %d", len(a.Placements), len(b.Placements))
	}
	for i := range a.Placements {
		if a.Placements[i] != b.Placements[i] {
			t.Fatalf("determinism broken at placement %d: %d vs %d", i, a.Placements[i], b.Placements[i])
		}
	}
}

// ---------------------------------------------------------------------------
// 16. Room finish state
// ---------------------------------------------------------------------------

func TestRoomFinishState(t *testing.T) {
	players := []int64{1, 2, 3}
	result := RunSimulation(SimConfig{
		Tier:      t1,
		PlayerIDs: players,
		VolScript: map[int]float64{1: 0.1},
		PulseSchedule: mergePulses(
			playerPulses(3, 1, 4, 7, 10, 13, 16, 19, 22, 25, 28, 31, 34),
		),
		MaxTicks: 40,
	})

	if result.WinnerID != 3 {
		t.Fatalf("expected winner 3, got %d", result.WinnerID)
	}
	if result.FinishReason != "last_alive" {
		t.Fatalf("expected last_alive, got %s", result.FinishReason)
	}

	for _, pid := range []int64{1, 2} {
		if result.PlayerStats[pid].Alive {
			t.Fatalf("player %d should be dead", pid)
		}
	}
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
		{10, 0},
		{33, 0},
		{34, 1},
	}
	for _, tt := range tests {
		got := WarChestContribution(tt.rake)
		if got != tt.want {
			t.Errorf("WarChestContribution(%d) = %d, want %d", tt.rake, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// 18. Placements are computed correctly
// ---------------------------------------------------------------------------

func TestPlacementsOrder(t *testing.T) {
	players := []int64{1, 2, 3, 4, 5}
	result := RunSimulation(SimConfig{
		Tier:      t1,
		PlayerIDs: players,
		VolScript: map[int]float64{1: 0.1},
		PulseSchedule: mergePulses(
			playerPulses(5, 1, 4, 7, 10, 13, 16, 19, 22, 25, 28, 31, 34, 37, 40, 43, 46, 49, 52, 55),
			playerPulses(4, 1, 4, 7, 10, 13, 16, 19, 22, 25, 28, 31, 34, 37, 40),
			playerPulses(3, 1, 4, 7, 10, 13, 16, 19, 22, 25),
			playerPulses(2, 1, 4, 7, 10),
			// player 1: no pulses → eliminated first
		),
		MaxTicks: 80,
	})

	if result.WinnerID != 5 {
		t.Fatalf("expected winner 5, got %d", result.WinnerID)
	}
	if len(result.Placements) != 5 {
		t.Fatalf("expected 5 placements, got %d", len(result.Placements))
	}
	// 5 is 1st (last alive), 4 is 2nd (last eliminated), etc.
	if result.Placements[0] != 5 {
		t.Errorf("1st place should be 5, got %d", result.Placements[0])
	}

	// Verify all placement stats are set
	for _, pid := range players {
		st := result.PlayerStats[pid]
		if st.Placement == 0 {
			t.Errorf("player %d placement not set", pid)
		}
	}

	// Winner gets payout, losers get shards
	winner := result.PlayerStats[5]
	if winner.Payout == 0 {
		t.Error("winner should have payout")
	}
	if winner.ShardsEarned != 0 {
		t.Error("winner should not earn shards")
	}

	// Last place should get base shards
	last := result.PlayerStats[result.Placements[4]]
	if last.ShardsEarned == 0 {
		t.Error("last place should earn shards")
	}
}

// ---------------------------------------------------------------------------
// 19. Top-3 payouts sum correctly
// ---------------------------------------------------------------------------

func TestTop3PayoutDistribution(t *testing.T) {
	players := []int64{1, 2, 3, 4, 5}
	result := RunSimulation(SimConfig{
		Tier:      t1,
		PlayerIDs: players,
		VolScript: map[int]float64{1: 0.1},
		PulseSchedule: mergePulses(
			playerPulses(5, 1, 4, 7, 10, 13, 16, 19, 22, 25, 28, 31, 34, 37, 40, 43, 46, 49, 52, 55),
			playerPulses(4, 1, 4, 7, 10, 13, 16, 19, 22, 25, 28, 31, 34, 37, 40),
			playerPulses(3, 1, 4, 7, 10, 13, 16, 19, 22, 25),
			playerPulses(2, 1, 4, 7, 10),
		),
		MaxTicks: 80,
	})

	pool := int64(5) * t1.EntryCost // 25
	rake := RakeAmount(pool)        // 3
	postRake := pool - rake         // 22

	totalPayout := int64(0)
	for _, pid := range result.Placements[:3] {
		totalPayout += result.PlayerStats[pid].Payout
	}

	// Allow 1 star truncation loss per payout
	if totalPayout > postRake {
		t.Fatalf("total payouts %d exceed post-rake pool %d", totalPayout, postRake)
	}
	if totalPayout < postRake-3 {
		t.Fatalf("total payouts %d too far below post-rake pool %d", totalPayout, postRake)
	}

	// 4th and 5th should have 0 payout but positive shards
	for _, pid := range result.Placements[3:] {
		st := result.PlayerStats[pid]
		if st.Payout != 0 {
			t.Errorf("player %d (place %d) should have 0 payout, got %d", pid, st.Placement, st.Payout)
		}
		if st.ShardsEarned == 0 {
			t.Errorf("player %d (place %d) should earn shards", pid, st.Placement)
		}
	}
}

// ---------------------------------------------------------------------------
// 20. Free pulses — StarsSpent equals entry cost
// ---------------------------------------------------------------------------

func TestFreePulses(t *testing.T) {
	players := []int64{1, 2, 3}
	result := RunSimulation(SimConfig{
		Tier:      t1,
		PlayerIDs: players,
		VolScript: map[int]float64{1: 0.1},
		PulseSchedule: mergePulses(
			playerPulses(1, 1, 4, 7, 10, 13, 16, 19),
			playerPulses(2, 1, 4, 7),
			playerPulses(3, 1),
		),
		MaxTicks: 30,
	})

	for _, pid := range players {
		st := result.PlayerStats[pid]
		if st.StarsSpent != t1.EntryCost {
			t.Errorf("player %d StarsSpent=%d, want %d (entry cost only, pulses are free)",
				pid, st.StarsSpent, t1.EntryCost)
		}
	}
}

// ---------------------------------------------------------------------------
// 21. Latency normalization — 50ms vs 200ms vs 400ms sniping advantage
//
// Models latency as server-side arrival delay (ticks) plus per-pulse jitter.
// Uses correct compensation: decision interval = pwTicks, first decision
// adjusted for delay so all groups have the SAME arrival frequency.
// Jitter simulates real-world network variance that can push arrivals past
// the pulse window boundary.
//
// Scenarios:
//   A) No jitter (baseline): proves tick system normalizes static latency
//   B) Realistic jitter: 400ms has 3% chance per pulse of +2 tick spike
// ---------------------------------------------------------------------------

func TestLatencyNormalization(t *testing.T) {
	type latSpec struct {
		name       string
		delay      int     // server-side base delay in ticks
		jitterMax  int     // max additional ticks when jitter fires
		jitterProb float64 // per-pulse probability of jitter
	}

	const perGroup = 5
	const rounds = 1000

	scenarios := []struct {
		name string
		lats []latSpec
	}{
		{
			"no_jitter",
			[]latSpec{
				{"50ms", 0, 0, 0},
				{"200ms", 1, 0, 0},
				{"400ms", 2, 0, 0},
			},
		},
		{
			"realistic_jitter",
			[]latSpec{
				{"50ms", 0, 1, 0.02},  // 2% chance of +1 tick
				{"200ms", 1, 1, 0.05}, // 5% chance of +1 tick
				{"400ms", 2, 2, 0.03}, // 3% chance of +1 or +2 ticks
			},
		},
	}

	tier := room.Tiers[3] // T3 (3s window = 12 ticks) — most latency-sensitive
	pwTicks := int(tier.PulseWindow / simTickRate)

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			lats := sc.lats
			type gstat struct {
				wins, top3, games, totalPlace, survTicks, elims int
			}
			gs := make([]gstat, len(lats))

			rng := rand.New(rand.NewSource(99991))

			for round := 0; round < rounds; round++ {
				var pids []int64
				pgroup := make(map[int64]int)

				// Interleave IDs across groups: {1,2,3}, {4,5,6}, ... → group = (id-1) % 3
				for p := 0; p < perGroup; p++ {
					for gi := range lats {
						pid := int64(p*len(lats) + gi + 1)
						pids = append(pids, pid)
						pgroup[pid] = gi
					}
				}

				schedule := make(map[int][]int64)
				for _, pid := range pids {
					gi := pgroup[pid]
					delay := lats[gi].delay
					jMax := lats[gi].jitterMax
					jProb := lats[gi].jitterProb

					// Decision interval = pwTicks for all (same strategy).
					// First decision = pwTicks - delay (so first arrival lands at pwTicks).
					// Subsequent decisions every pwTicks ticks.
					firstDec := pwTicks - delay
					if firstDec < 1 {
						firstDec = 1
					}
					for d := firstDec; d <= 2400; d += pwTicks {
						arrival := d + delay
						if jMax > 0 && rng.Float64() < jProb {
							arrival += 1 + rng.Intn(jMax) // +1 to +jMax
						}
						if arrival > 2400 {
							break
						}
						schedule[arrival] = append(schedule[arrival], pid)
					}
				}

				volScript := make(map[int]float64)
				base := 0.05 + float64(round%50)*0.005
				for tick := 4; tick <= 2400; tick += 4 {
					mr := base + float64(tick)/1400.0
					if mr > 1.0 {
						mr = 1.0
					}
					volScript[tick] = mr
				}

				res := RunSimulation(SimConfig{
					Tier:          tier,
					PlayerIDs:     pids,
					VolScript:     volScript,
					PulseSchedule: schedule,
					MaxTicks:      2400,
					SilentMode:    true,
				})

				for _, pid := range pids {
					gi := pgroup[pid]
					st := res.PlayerStats[pid]
					gs[gi].games++
					gs[gi].totalPlace += st.Placement
					if st.Placement == 1 {
						gs[gi].wins++
					}
					if st.Placement <= 3 {
						gs[gi].top3++
					}
					if st.EliminatedAt > 0 {
						gs[gi].elims++
						gs[gi].survTicks += st.EliminatedAt
					} else {
						gs[gi].survTicks += res.TotalTicks
					}
				}
			}

			t.Logf("\n  %-10s %8s %7s %7s %9s %7s",
				"Latency", "AvgPlace", "Win%", "Top3%", "AvgSurv", "Elim%")
			for i, l := range lats {
				s := gs[i]
				t.Logf("  %-10s %8.2f %6.2f%% %6.2f%% %8.1fs %6.2f%%",
					l.name,
					float64(s.totalPlace)/float64(s.games),
					float64(s.wins)/float64(s.games)*100,
					float64(s.top3)/float64(s.games)*100,
					float64(s.survTicks)/float64(s.games)*0.25,
					float64(s.elims)/float64(s.games)*100)
			}

			// Top-3 fairness (payout-relevant metric, statistically robust)
			top3Rates := make([]float64, len(lats))
			for i := range lats {
				top3Rates[i] = float64(gs[i].top3) / float64(gs[i].games) * 100
			}
			bestT3, worstT3 := top3Rates[0], top3Rates[0]
			for _, r := range top3Rates {
				if r > bestT3 {
					bestT3 = r
				}
				if r < worstT3 {
					worstT3 = r
				}
			}
			var t3Adv float64
			if bestT3 > 0 {
				t3Adv = (bestT3 - worstT3) / bestT3 * 100
			}
			t.Logf("  Top-3 rate spread: best=%.2f%% worst=%.2f%% relative=%.1f%%",
				bestT3, worstT3, t3Adv)
			if t3Adv > 15 {
				t.Errorf("LATENCY UNFAIR: top-3 rate advantage %.1f%% exceeds 15%%", t3Adv)
			}

			// Placement fairness
			avgPlaces := make([]float64, len(lats))
			for i := range lats {
				avgPlaces[i] = float64(gs[i].totalPlace) / float64(gs[i].games)
			}
			bP, wP := avgPlaces[0], avgPlaces[0]
			for _, ap := range avgPlaces {
				if ap < bP {
					bP = ap
				}
				if ap > wP {
					wP = ap
				}
			}
			var placeAdv float64
			if wP > 0 {
				placeAdv = (wP - bP) / wP * 100
			}
			t.Logf("  Placement spread: best=%.2f worst=%.2f relative=%.1f%%",
				bP, wP, placeAdv)
			if placeAdv > 15 {
				t.Errorf("LATENCY UNFAIR: placement advantage %.1f%% exceeds 15%%", placeAdv)
			}
		})
	}
}
