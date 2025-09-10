package backoff

import (
	"math/rand/v2"
	"time"
)

// Option is a function type used to configure backoff strategies.
// Options are applied during the creation of backoff instances to
// customize behavior such as retry limits, jitter, and timing bounds.
type Option func(*options)

// WithMaxInterval sets the maximum delay interval for backoff strategies.
// Delays will be capped at this duration regardless of the backoff algorithm.
// A value of 0 means no maximum limit.
//
// Example:
//
//	backoff := NewExponential(100*time.Millisecond, 2.0,
//		WithMaxInterval(5*time.Second))
func WithMaxInterval(d time.Duration) Option {
	return func(o *options) {
		o.maxInterval = d
	}
}

// WithMinInterval sets the minimum delay interval for backoff strategies.
// Delays will never be shorter than this duration.
// A value of 0 means no minimum limit.
//
// Example:
//
//	backoff := NewExponential(10*time.Millisecond, 2.0,
//		WithMinInterval(50*time.Millisecond))
func WithMinInterval(d time.Duration) Option {
	return func(o *options) {
		o.minInterval = d
	}
}

// WithMaxRetries sets the maximum number of retry attempts.
// After this many retries, Next() will return (0, false).
// A value of -1 means unlimited retries.
//
// Example:
//
//	backoff := NewConstant(100*time.Millisecond,
//		WithMaxRetries(5)) // Stop after 5 attempts
func WithMaxRetries(v int) Option {
	return func(o *options) {
		o.maxRetries = v
	}
}

// WithMaxElapsed sets the maximum total elapsed time for all retry attempts.
// Once this duration has passed, Next() will return (0, false).
// A value of 0 means no time limit.
//
// Example:
//
//	backoff := NewExponential(100*time.Millisecond, 2.0,
//		WithMaxElapsed(30*time.Second)) // Stop after 30 seconds total
func WithMaxElapsed(d time.Duration) Option {
	return func(o *options) {
		o.maxElapsed = d
	}
}

// WithRandSource sets a custom random source for jitter calculations.
// This allows for deterministic testing or custom randomization behavior.
// If not specified, a default PCG source with fixed seed is used.
//
// Example:
//
//	source := rand.NewPCG(42, 1024)
//	backoff := NewExponential(100*time.Millisecond, 2.0,
//		WithRandSource(source),
//		WithJitter())
func WithRandSource(s rand.Source) Option {
	return func(o *options) {
		o.rand = rand.New(s)
	}
}

// WithJitter enables equal jitter for the backoff strategy.
// Equal jitter adds randomness to delay intervals by using half the
// calculated delay plus a random amount up to the other half.
// This helps prevent thundering herd problems.
//
// Example:
//
//	backoff := NewExponential(100*time.Millisecond, 2.0,
//		WithJitter()) // Uses EqualJitter strategy
func WithJitter() Option {
	return func(o *options) {
		o.jitter = &EqualJitter{}
	}
}

// WithJitterStrategy sets a custom jitter strategy for the backoff.
// This allows you to use FullJitter, EqualJitter, NoneJitter, or
// implement your own custom jitter algorithm.
//
// Example:
//
//	backoff := NewExponential(100*time.Millisecond, 2.0,
//		WithJitterStrategy(&FullJitter{}))
func WithJitterStrategy(j Jitter) Option {
	return func(o *options) {
		o.jitter = j
	}
}

// applyOptions creates a new options struct with default values and
// applies all provided option functions to configure the backoff behavior.
//
// Default values:
//   - maxRetries: -1 (unlimited)
//   - maxElapsed: 0 (no time limit)
//   - rand: PCG source with seed (42, 1024)
//   - maxInterval: 0 (no maximum)
//   - minInterval: 0 (no minimum)
//   - jitter: NoneJitter (no jitter)
func applyOptions(opts []Option) *options {
	o := &options{
		maxRetries:  -1,
		maxElapsed:  0,
		rand:        rand.New(rand.NewPCG(42, 1024)),
		maxInterval: 0,
		minInterval: 0,
		jitter:      &NoneJitter{},
	}

	for _, opt := range opts {
		opt(o)
	}

	return o
}
