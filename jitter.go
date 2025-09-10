package backoff

import (
	"math/rand/v2"
	"time"
)

// Jitter defines the interface for applying randomization to delay durations.
// Jitter strategies help prevent thundering herd problems by adding randomness
// to retry attempts, spreading them out over time instead of having all clients
// retry simultaneously.
type Jitter interface {
	// Apply takes a calculated delay duration and applies jitter using the
	// provided random number generator, returning the final delay to use.
	Apply(d time.Duration, r *rand.Rand) time.Duration
}

// NoneJitter implements a jitter strategy that applies no randomization.
// The delay duration is returned unchanged. This is the default jitter
// strategy when no jitter options are specified.
type NoneJitter struct{}

// Apply returns the input duration unchanged, providing no jitter.
func (nj *NoneJitter) Apply(d time.Duration, _ *rand.Rand) time.Duration {
	return d
}

// FullJitter implements a jitter strategy that randomizes the entire delay.
// The final delay is a random value between 1 and the calculated delay duration.
// This provides maximum randomization but may result in very short delays.
//
// Formula: random(1, calculated_delay)
type FullJitter struct{}

// Apply returns a random duration between 1 and the input duration (inclusive).
// If the input duration is <= 0, returns 0.
func (FullJitter) Apply(d time.Duration, r *rand.Rand) time.Duration {
	if d <= 0 {
		return 0
	}
	return time.Duration(r.Int64N(int64(d)) + 1)
}

// EqualJitter implements a jitter strategy that uses half the calculated delay
// as a base and adds randomness to the other half. This provides a good balance
// between maintaining reasonable delay lengths and adding randomization.
//
// Formula: (calculated_delay / 2) + random(0, calculated_delay / 2)
type EqualJitter struct{}

// Apply returns half the input duration plus a random amount up to the other half.
// This ensures the result is between 50% and 100% of the original duration.
// If the input duration is <= 0, returns 0.
func (EqualJitter) Apply(d time.Duration, r *rand.Rand) time.Duration {
	if d <= 0 {
		return 0
	}
	half := d / 2
	return half + time.Duration(r.Int64N(int64(d-half)+1))
}
