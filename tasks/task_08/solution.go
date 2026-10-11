package main

import (
	"math"
	"sync"
	"time"
)

type Clock interface{ Now() time.Time }
type Limiter struct {
	mu     sync.Mutex
	clock  Clock
	rate   float64
	burst  int
	tokens float64
	last   time.Time
}

func NewLimiter(clock Clock, ratePerSec float64, burst int) *Limiter {
	if ratePerSec <= 0 {
		ratePerSec = 0
	}
	return &Limiter{
		mu:     sync.Mutex{},
		clock:  clock,
		rate:   ratePerSec,
		burst:  burst,
		tokens: float64(burst),
		last:   time.Time{},
	}
}
func (l *Limiter) AllowN(n int) bool {
	if l.clock == nil {
		return false
	}
	if n <= 0 || n > l.burst {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.clock.Now()
	delta := now.Sub(l.last).Seconds()
	rateTokens := l.rate * delta
	l.tokens += rateTokens
	l.tokens = math.Min(l.tokens, float64(l.burst))
	l.last = now

	if float64(n) <= l.tokens {
		l.tokens -= float64(n)
		return true
	}

	return false
}
