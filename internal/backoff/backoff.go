package backoff

import (
	"math/rand"
	"sync"
	"time"
)

type Strategy interface {
	Backoff(retries int) time.Duration
}

const (
	baseDelay = 1.0 * time.Second
	factor    = 1.6
	jitter    = 0.2
)

type exponential struct {
	maxDelay time.Duration
	r        *rand.Rand
	mu       sync.Mutex
}

func New(maxDelay time.Duration) Strategy {
	return &exponential{
		maxDelay: maxDelay,
		r:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (bc *exponential) Backoff(retries int) time.Duration {
	if retries == 0 {
		return baseDelay
	}
	backoff, max := float64(baseDelay), float64(bc.maxDelay)
	for backoff < max && retries > 0 {
		backoff *= factor
		retries--
	}
	if backoff > max {
		backoff = max
	}

	backoff *= 1 + jitter*(bc.float64()*2-1)
	if backoff < 0 {
		return 0
	}
	return time.Duration(backoff)
}

func (bc *exponential) float64() (f float64) {
	bc.mu.Lock()
	f = bc.r.Float64()
	bc.mu.Unlock()
	return
}
