package backoff

import "time"

// Backoff doubles its delay on each Next() call up to a maximum, and Reset()
// restores it to the minimum. Used to avoid hammering an unreachable server:
// after repeated failures the retry cadence grows 1s, 2s, 4s, ... until it is
// capped at the heartbeat interval.
type Backoff struct {
	min     time.Duration
	max     time.Duration
	current time.Duration
}

// New returns a Backoff starting at min, capped at max. If max is smaller than
// min, max is raised to min so the delay never goes below the start point.
func New(min, max time.Duration) *Backoff {
	if max < min {
		max = min
	}
	return &Backoff{min: min, max: max, current: min}
}

// Next returns the current delay, then doubles it (capped at max) for the next
// call.
func (b *Backoff) Next() time.Duration {
	d := b.current
	if next := b.current * 2; next > b.max {
		b.current = b.max
	} else {
		b.current = next
	}
	return d
}

// Reset restores the delay to the minimum, e.g. after a successful send.
func (b *Backoff) Reset() {
	b.current = b.min
}

// Current returns the delay without advancing it.
func (b *Backoff) Current() time.Duration {
	return b.current
}
