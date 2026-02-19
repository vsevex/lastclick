package game

import "time"

// PulseExtension calculates how much time a pulse adds to the global timer.
// Diminishing returns: baseExtension / alivePlayers.
func PulseExtension(baseExtension time.Duration, alivePlayers int) time.Duration {
	if alivePlayers <= 0 {
		return 0
	}
	return baseExtension / time.Duration(alivePlayers)
}

// TickDecrement is the amount the global timer decreases each tick.
// Accelerates as margin ratio approaches 1.0 (liquidation).
func TickDecrement(tickInterval time.Duration, marginRatio float64) time.Duration {
	accel := 1.0 + marginRatio*2.0
	return time.Duration(float64(tickInterval) * accel)
}
