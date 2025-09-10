// Package backoff provides various backoff strategies for retry mechanisms.
package backoff

import (
	"math"
	"math/rand/v2"
	"time"
)

// Sequence defines the interface for backoff strategies.
// Implementations should provide methods to get the next delay duration
// and reset the internal state.
type Sequence interface {
	// Next returns the next delay duration and a boolean indicating
	// whether more retries are allowed. Returns (0, false) when
	// maximum retries or elapsed time limits are reached.
	Next() (time.Duration, bool)

	// Reset resets the backoff sequence to its initial state,
	// clearing retry count and elapsed time.
	Reset()
}

// options holds configuration for backoff strategies.
type options struct {
	maxRetries  int           // -1 = infinite retries
	maxElapsed  time.Duration // 0 = no time limit
	rand        *rand.Rand    // random number generator for jitter
	maxInterval time.Duration // maximum delay interval
	minInterval time.Duration // minimum delay interval
	jitter      Jitter        // jitter strategy to apply
}

// Constant implements a constant backoff strategy with fixed delay intervals.
// This strategy returns the same delay duration for each retry attempt.
//
// Use Constant for scenarios where you want predictable, uniform delays
// between retry attempts. This is useful when you need consistent timing
// or when working with systems that have specific rate limiting requirements.
type Constant struct {
	options *options

	interval time.Duration // fixed delay interval
	retries  int           // current retry count
	elapsed  time.Duration // total elapsed time
}

// NewConstant creates a new constant backoff strategy with the specified interval.
//
// Parameters:
//   - d: The fixed delay duration between retry attempts
//   - opts: Optional configuration functions (WithMaxRetries, WithMaxElapsed, etc.)
//
// Example:
//
//	// 500ms delay with max 3 retries
//	constant := NewConstant(500*time.Millisecond, WithMaxRetries(3))
//
//	// 1 second delay with 30 second total timeout
//	constant := NewConstant(time.Second, WithMaxElapsed(30*time.Second))
func NewConstant(d time.Duration, opts ...Option) *Constant {
	return &Constant{
		interval: d,
		options:  applyOptions(opts),
	}
}

// Next returns the next delay duration and whether more retries are allowed.
// For constant backoff, this always returns the same interval duration
// until maximum retry or elapsed time limits are reached.
//
// Returns:
//   - time.Duration: The delay duration (always the configured interval)
//   - bool: true if more retries are allowed, false if limits are reached
func (c *Constant) Next() (time.Duration, bool) {
	if c.options.maxRetries >= 0 && c.retries >= c.options.maxRetries {
		return 0, false
	}

	if c.options.maxElapsed > 0 && c.elapsed >= c.options.maxElapsed {
		return 0, false
	}

	c.retries++
	c.elapsed += c.interval
	return c.interval, true
}

// Reset resets the constant backoff to its initial state.
// This clears the retry count and elapsed time, allowing the sequence
// to be reused for a new set of retry attempts.
func (c *Constant) Reset() {
	c.retries = 0
	c.elapsed = 0
}

// Exponential implements an exponential backoff strategy where delays
// increase exponentially with each retry attempt.
//
// This strategy is effective for handling temporary failures and avoiding
// overwhelming systems during recovery periods.
type Exponential struct {
	options *options
	base    time.Duration // initial delay duration
	factor  float64       // multiplier for each retry

	retries int           // current retry count
	elapsed time.Duration // total elapsed time
	current time.Duration // current calculated delay
}

// NewExponential creates a new exponential backoff strategy.
//
// Parameters:
//   - base: The initial delay duration for the first retry
//   - factor: The multiplier applied to increase delays (must be > 1.0)
//   - opts: Optional configuration functions
//
// If factor <= 1.0, it defaults to 2.0 for proper exponential growth.
//
// Example:
//
//	// Start at 100ms, double each time, max 5 seconds
//	exp := NewExponential(100*time.Millisecond, 2.0,
//		WithMaxInterval(5*time.Second))
//
//	// With jitter to prevent thundering herd
//	exp := NewExponential(50*time.Millisecond, 1.5,
//		WithJitter(),
//		WithMaxRetries(10))
func NewExponential(base time.Duration, factor float64, opts ...Option) *Exponential {
	if factor <= 1.0 {
		factor = 2.0
	}

	return &Exponential{
		options: applyOptions(opts),
		base:    base,
		factor:  factor,
	}
}

// Next returns the next exponentially increased delay duration.
// The delay grows exponentially: base, base*factor, base*factor^2, etc.
//
// The calculated delay is subject to:
//   - Jitter application (if configured)
//   - Min/max interval bounds
//   - Overflow protection (capped at math.MaxInt64)
//
// Returns:
//   - time.Duration: The calculated delay duration
//   - bool: true if more retries are allowed, false if limits are reached
func (e *Exponential) Next() (time.Duration, bool) {
	if e.options.maxRetries >= 0 && e.retries >= e.options.maxRetries {
		return 0, false
	}

	d := e.base
	if e.retries > 0 {
		d = e.current * time.Duration(e.factor)
	}

	d = e.options.jitter.Apply(d, e.options.rand)

	if float64(d) > float64(math.MaxInt64) {
		d = time.Duration(math.MaxInt64)
	}

	d = applyBounds(d, e.options.minInterval, e.options.maxInterval)
	if e.options.maxElapsed > 0 && e.elapsed+d >= e.options.maxElapsed {
		return 0, false
	}

	e.current = d
	e.retries++
	e.elapsed += d
	return d, true
}

// Reset resets the exponential backoff to its initial state.
// This clears the retry count, elapsed time, and current delay calculation.
func (e *Exponential) Reset() {
	e.retries = 0
	e.elapsed = 0
	e.current = 0
}

// Decorrelated implements a decorrelated jitter backoff strategy.
// This strategy uses randomized delays to prevent synchronized retry attempts
// across multiple clients, effectively preventing thundering herd problems.
//
// The algorithm picks a random delay between the minimum interval and
// (previous_delay * factor), providing both exponential growth characteristics
// and randomization to spread out retry attempts.
type Decorrelated struct {
	initial time.Duration // initial delay duration
	factor  float64       // growth factor for delay calculation
	options *options      // configuration options

	retries int           // current retry count
	elapsed time.Duration // total elapsed time
	prev    time.Duration // previous delay duration
}

// NewDecorrelated creates a new decorrelated jitter backoff strategy.
//
// Parameters:
//   - initial: The initial delay duration for the first retry
//   - factor: The growth factor for delay calculation (must be > 1.0)
//   - opts: Optional configuration functions
//
// If factor <= 1.0, it defaults to 3.0 for effective jitter spread.
// If no maxInterval is specified, it defaults to 30 seconds.
//
// Example:
//
//	// Start at 100ms with 3x growth factor
//	dcr := NewDecorrelated(100*time.Millisecond, 3.0,
//		WithMaxInterval(10*time.Second),
//		WithMaxRetries(5))
//
//	// With custom min/max bounds
//	dcr := NewDecorrelated(50*time.Millisecond, 2.5,
//		WithMinInterval(10*time.Millisecond),
//		WithMaxInterval(5*time.Second))
func NewDecorrelated(initial time.Duration, factor float64, opts ...Option) *Decorrelated {
	if factor <= 1.0 {
		factor = 3.0
	}

	o := applyOptions(opts)

	if o.maxInterval <= 0 {
		o.maxInterval = 30 * time.Second
	}

	return &Decorrelated{
		initial: initial,
		factor:  factor,
		options: o,
	}
}

// Next returns the next decorrelated delay duration.
// For the first retry, returns the initial duration.
// For subsequent retries, picks a random duration between minInterval
// and (previous_delay * factor), bounded by maxInterval.
//
// This randomization helps prevent multiple clients from retrying
// simultaneously, reducing load spikes on recovering systems.
//
// Returns:
//   - time.Duration: The calculated random delay duration
//   - bool: true if more retries are allowed, false if limits are reached
func (dcr *Decorrelated) Next() (time.Duration, bool) {
	if dcr.options.maxRetries >= 0 && dcr.retries >= dcr.options.maxRetries {
		return 0, false
	}

	var base time.Duration
	if dcr.retries == 0 || dcr.prev <= 0 {
		base = dcr.initial
	} else {
		low := dcr.options.minInterval
		high := time.Duration(float64(dcr.prev) * dcr.factor)
		high = max(high, low)
		if high > dcr.options.maxInterval && dcr.options.maxInterval > 0 {
			high = dcr.options.maxInterval
		}
		base = randBetween(dcr.options.rand, low, high)
	}

	base = applyBounds(base, dcr.options.minInterval, dcr.options.maxInterval)
	delay := dcr.options.jitter.Apply(base, dcr.options.rand)

	if dcr.options.maxElapsed > 0 && dcr.elapsed+delay > dcr.options.maxElapsed {
		return 0, false
	}

	dcr.retries++
	dcr.elapsed += delay
	dcr.prev = base
	return delay, true
}

// Reset resets the decorrelated backoff to its initial state.
// This clears the retry count, elapsed time, and previous delay history.
func (dcr *Decorrelated) Reset() {
	dcr.retries = 0
	dcr.elapsed = 0
	dcr.prev = 0
}

// applyBounds ensures the duration falls within the specified min/max bounds.
// Returns the bounded duration, with negative durations converted to 0.
func applyBounds(d, min, max time.Duration) time.Duration {
	if min > 0 && d < min {
		d = min
	}
	if max > 0 && d > max {
		d = max
	}
	if d < 0 {
		return 0
	}
	return d
}

// randBetween generates a random duration between low and high (inclusive).
// If high <= low, returns low. Used for decorrelated jitter calculations.
func randBetween(r *rand.Rand, low, high time.Duration) time.Duration {
	if high <= low {
		return low
	}
	span := high - low
	return low + time.Duration(r.Int64N(int64(span)+1))
}
