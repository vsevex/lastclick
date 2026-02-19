package volatility

import "time"

// ScriptedFeed replays a fixed sequence of margin ratio values at a fixed tick
// rate. Deterministic — same script always produces the same output. Used for
// integration tests with the real Engine.runLoop.
type ScriptedFeed struct {
	Script   []float64     // margin ratio per tick
	TickRate time.Duration // defaults to 250ms if zero
}

func NewScriptedFeed(script []float64) *ScriptedFeed {
	return &ScriptedFeed{Script: script, TickRate: 250 * time.Millisecond}
}

func (f *ScriptedFeed) Start(stop <-chan struct{}) <-chan Update {
	ch := make(chan Update, 32)
	rate := f.TickRate
	if rate == 0 {
		rate = 250 * time.Millisecond
	}
	go func() {
		defer close(ch)
		ticker := time.NewTicker(rate)
		defer ticker.Stop()
		for i, mr := range f.Script {
			select {
			case <-stop:
				return
			case <-ticker.C:
				price := 100.0 * (1.0 - mr*0.5)
				select {
				case ch <- Update{MarginRatio: mr, Price: price}:
				default:
				}
				_ = i
			}
		}
	}()
	return ch
}
