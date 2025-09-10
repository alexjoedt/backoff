# backoff

[![Go Reference](https://pkg.go.dev/badge/github.com/alexjoedt/backoff.svg)](https://pkg.go.dev/github.com/alexjoedt/backoff)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexjoedt/backoff)](https://goreportcard.com/report/github.com/alexjoedt/backoff)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Coverage](https://img.shields.io/badge/coverage-93.8%25-brightgreen)](https://github.com/alexjoedt/backoff)

Yet another Go backoff library, because sometimes you need to retry things and the existing ones didn't quite fit what I needed.

## What's in the box?

- Three different backoff strategies (constant, exponential, decorrelated jitter)
- Configurable retry limits and timeouts
- Built-in jitter to avoid the thundering herd problem
- Zero dependencies (just stdlib)

## Installation

```bash
go get github.com/alexjoedt/backoff
```

## Basic usage

```go
package main

import (
    "fmt"
    "time"
    "github.com/alexjoedt/backoff"
)

func main() {
    // Start with 100ms, double each time, give up after 5 tries
    b := backoff.NewExponential(100*time.Millisecond, 2.0,
        backoff.WithMaxRetries(5),
        backoff.WithJitter(), // adds some randomness
    )

    for attempt := 1; ; attempt++ {
        err := doSomethingThatMightFail()
        if err == nil {
            fmt.Println("Success!")
            break
        }

        delay, ok := b.Next()
        if !ok {
            fmt.Println("Giving up after", attempt-1, "attempts")
            break
        }

        fmt.Printf("Try %d failed, waiting %v before retry...\n", attempt, delay)
        time.Sleep(delay)
    }
}

func doSomethingThatMightFail() error {
    // Your flaky operation here
    return nil
}
```

## The different strategies

### Constant - when you just want to wait the same time each retry

```go
// Wait 500ms between each attempt, try 3 times max
b := backoff.NewConstant(500*time.Millisecond, backoff.WithMaxRetries(3))

for {
    err := doThing()
    if err == nil {
        break // Success!
    }

    delay, ok := b.Next()
    if !ok {
        return fmt.Errorf("still broken after 3 tries: %w", err)
    }
    time.Sleep(delay)
}
```

### Exponential - the classic approach

Gets progressively longer waits. Good for most retry scenarios.

```go
// Start at 100ms, double each time, but don't wait longer than 10s total
b := backoff.NewExponential(100*time.Millisecond, 2.0,
    backoff.WithMaxInterval(10*time.Second),
    backoff.WithMaxElapsed(30*time.Second),
)

// This will try: 100ms, 200ms, 400ms, 800ms, etc.
for {
    err := callAPI()
    if err == nil {
        break
    }

    delay, ok := b.Next()
    if !ok {
        return fmt.Errorf("API still down after 30s: %w", err)
    }
    time.Sleep(delay)
}
```

### Decorrelated Jitter - the fancy one

This one's more random and helps avoid the "thundering herd" problem when lots of clients are retrying at the same time.

```go
// Random waits that grow over time but stay unpredictable
b := backoff.NewDecorrelated(100*time.Millisecond, 3.0,
    backoff.WithMinInterval(50*time.Millisecond),
    backoff.WithMaxInterval(10*time.Second),
    backoff.WithMaxRetries(10),
)

for {
    err := connectToDatabase()
    if err == nil {
        break
    }

    delay, ok := b.Next()
    if !ok {
        return fmt.Errorf("database still unreachable after 10 tries: %w", err)
    }
    time.Sleep(delay)
}
```

## Configuration

You can customize the behavior with these options:

```go
// When to give up
backoff.WithMaxRetries(5)                 // Stop after 5 attempts  
backoff.WithMaxElapsed(30*time.Second)    // Or stop after 30 seconds total

// Control the timing
backoff.WithMinInterval(100*time.Millisecond)  // Never wait less than this
backoff.WithMaxInterval(10*time.Second)        // Never wait more than this

// Add some randomness
backoff.WithJitter()                           // Adds equal jitter
backoff.WithJitterStrategy(&backoff.FullJitter{})  // More random
backoff.WithJitterStrategy(&backoff.NoneJitter{})  // No randomness

// For testing with predictable randomness
source := rand.NewPCG(42, 1024)
backoff.WithRandSource(source)
```

## Jitter explained

**No Jitter** - Predictable delays
```
100ms --> 200ms --> 400ms --> 800ms...
```

**Equal Jitter** - Half predictable, half random  
```
50-100ms --> 100-200ms --> 200-400ms --> 400-800ms...
```

**Full Jitter** - Completely random within bounds
```
1-100ms --> 1-200ms --> 1-400ms --> 1-800ms...
```

**Decorrelated Jitter** - Random but still grows over time
```
100ms --> random(min, prev*3) --> random(min, prev*3)...
```

The randomness helps when you have multiple clients hitting the same service, they won't all retry at exactly the same time.

## Thread Safety

**Heads up**: This library isn't thread-safe. Each goroutine should get its own backoff instance.

```go
// Good: Each worker gets its own backoff
func worker(id int) {
    b := backoff.NewExponential(100*time.Millisecond, 2.0)
    // use b in this goroutine...
}

// Bad: Sharing one instance across goroutines  
b := backoff.NewExponential(100*time.Millisecond, 2.0)
for i := 0; i < 10; i++ {
    go func() {
        b.Next() // Race condition!
    }()
}
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

