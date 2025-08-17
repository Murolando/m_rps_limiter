package limiter

import (
	"context"
	"sync/atomic"
	"time"
)

// tokenBucket implements the Token Bucket rate limiting algorithm.
// It maintains a bucket of tokens that are consumed by requests and refilled at regular intervals.
type tokenBucket struct {
	tokens *atomic.Int32      // Current number of tokens in the bucket
	limit  int32              // Maximum number of tokens the bucket can hold
	ctx    context.Context    // Context for graceful shutdown
	cancel context.CancelFunc // Cancel function to stop the token refill goroutine
}

// newTokenBucket creates a new token bucket with the specified limit and timer.
// The bucket starts full with 'limit' tokens and begins the refill process immediately.
func newTokenBucket(limit int32, timer *time.Ticker) *tokenBucket {
	l := &atomic.Int32{}
	l.Store(limit)
	ctx, cancel := context.WithCancel(context.Background())
	t := &tokenBucket{
		ctx:    ctx,
		cancel: cancel,
		limit:  limit,
		tokens: l,
	}
	go t.Repeater(timer)
	return t
}
func (t *tokenBucket) tryPass() bool {
	if t.tokens.Load() >= 1 {
		t.tokens.Add(-1)
		return true
	}
	return false
}

// Repeater runs in a separate goroutine and refills tokens at regular intervals.
// It adds one token to the bucket on each timer tick, up to the maximum limit.
// The goroutine stops when the context is cancelled.
func (t *tokenBucket) Repeater(timer *time.Ticker) {
	for {
		select {
		case <-t.ctx.Done():
			return
		case <-timer.C:
			if t.tokens.Load() < t.limit {
				t.tokens.Add(1)
			}
		}
	}
}

func (t *tokenBucket) Stop() {
	t.cancel()
}
