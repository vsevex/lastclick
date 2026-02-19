package game

import (
	"math"
	"time"
)

// VolatilityMultiplier increases as the margin ratio approaches liquidation (1.0).
// Uses exponential curve for dramatic escalation near the end.
func VolatilityMultiplier(marginRatio float64) float64 {
	clamped := math.Max(0, math.Min(1, marginRatio))
	return 1.0 + 4.0*math.Pow(clamped, 3)
}

// SurvivalEfficiency computes the primary competitive metric:
// Efficiency = (TimeSurvivedSec * VolatilityMultiplier) / StarsSpent
func SurvivalEfficiency(timeSurvived time.Duration, volMul float64, starsSpent int64) float64 {
	if starsSpent <= 0 {
		return 0
	}
	return (timeSurvived.Seconds() * volMul) / float64(starsSpent)
}

// RakeAmount computes the 10% pool rake.
func RakeAmount(pool int64) int64 {
	return pool / 10
}

// WinnerPayout computes the winner's share after rake.
func WinnerPayout(pool int64) int64 {
	return pool - RakeAmount(pool)
}

// WarChestContribution is 3% of total rake.
func WarChestContribution(rake int64) int64 {
	return rake * 3 / 100
}

// ShardsFromBurn converts 40-60% of burned Stars into Blitz Shards.
// The ratio scales with volatility multiplier (higher vol = more generous).
func ShardsFromBurn(starsSpent int64, volMul float64) int64 {
	ratio := 0.4 + 0.2*math.Min(1, (volMul-1)/4)
	return int64(float64(starsSpent) * ratio)
}
