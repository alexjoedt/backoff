package backoff

import (
	"math/rand/v2"
	"testing"
	"time"
)

// BenchmarkConstant measures the performance of constant backoff
func BenchmarkConstant(b *testing.B) {
	constant := NewConstant(100 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = constant.Next()
	}
}

// BenchmarkExponential measures the performance of exponential backoff
func BenchmarkExponential(b *testing.B) {
	exponential := NewExponential(100*time.Millisecond, 2.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = exponential.Next()
	}
}

// BenchmarkExponentialWithJitter measures exponential backoff with jitter
func BenchmarkExponentialWithJitter(b *testing.B) {
	exponential := NewExponential(100*time.Millisecond, 2.0,
		WithJitter(),
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = exponential.Next()
	}
}

// BenchmarkDecorrelated measures the performance of decorrelated backoff
func BenchmarkDecorrelated(b *testing.B) {
	decorrelated := NewDecorrelated(100*time.Millisecond, 3.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = decorrelated.Next()
	}
}

// BenchmarkJitterStrategies compares different jitter strategies
func BenchmarkJitterStrategies(b *testing.B) {
	duration := 100 * time.Millisecond
	r := rand.New(rand.NewPCG(42, 1024))

	jitters := map[string]Jitter{
		"None":  &NoneJitter{},
		"Equal": &EqualJitter{},
		"Full":  &FullJitter{},
	}

	for name, jitter := range jitters {
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = jitter.Apply(duration, r)
			}
		})
	}
}

// BenchmarkReset measures the performance of reset operations
func BenchmarkReset(b *testing.B) {
	strategies := map[string]Sequence{
		"Constant":     NewConstant(100 * time.Millisecond),
		"Exponential":  NewExponential(100*time.Millisecond, 2.0),
		"Decorrelated": NewDecorrelated(100*time.Millisecond, 3.0),
	}

	for name, strategy := range strategies {
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				strategy.Reset()
			}
		})
	}
}

// BenchmarkConcurrent measures concurrent access to backoff strategies
func BenchmarkConcurrent(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		// Each goroutine gets its own backoff instance to avoid race conditions
		exp := NewExponential(100*time.Millisecond, 2.0,
			WithMaxInterval(5*time.Second),
			WithJitter(),
		)
		for pb.Next() {
			_, _ = exp.Next()
		}
	})
}

// BenchmarkMemoryAllocation measures memory allocations
func BenchmarkMemoryAllocation(b *testing.B) {
	exponential := NewExponential(100*time.Millisecond, 2.0)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = exponential.Next()
	}
}
