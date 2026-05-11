package llm

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu         sync.Mutex
	lastCalled time.Time
	interval   time.Duration
}

func NewRateLimiter(rpm int) *RateLimiter {
	if rpm <= 0 {
		rpm = 30
	}
	return &RateLimiter{
		interval: time.Minute / time.Duration(rpm),
	}
}

func (rl *RateLimiter) Wait() {
	if rl.interval <= 0 {
		return
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	elapsed := time.Since(rl.lastCalled)
	if elapsed < rl.interval {
		time.Sleep(rl.interval - elapsed)
	}
	rl.lastCalled = time.Now()
}
