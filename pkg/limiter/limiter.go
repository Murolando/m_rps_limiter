// Package limiter provides rate limiting functionality with support for various algorithms.
package limiter

import (
	"errors"
	"net/http"
	"time"
)

// TokenBucket is the algorithm name constant for the Token Bucket rate limiting algorithm.
// The Token Bucket algorithm maintains a bucket of tokens that are consumed by requests
// and refilled at regular intervals, allowing for burst traffic up to the bucket capacity.
const (
	TokenBucket AlgorithmName = "token_bucket"
)

// AlgorithmName represents the name of a rate limiting algorithm.
type AlgorithmName string

// LimitAlgorithm defines the interface that all rate limiting algorithms must implement.
type LimitAlgorithm interface {
	// tryPass attempts to consume a token/permit for a request.
	// Returns true if the request should be allowed, false if it should be rate limited.
	// This method must be thread-safe.
	tryPass() bool

	// Stop gracefully shuts down the algorithm and cleans up any resources.
	// This should stop any background goroutines and prevent resource leaks.
	Stop()
}

// Limiter wraps a rate limiting algorithm and provides HTTP middleware functionality.
type Limiter struct {
	LimitAlgorithm
}

var _ LimitAlgorithm = &tokenBucket{}

// NewLimiter creates a new rate limiter with the specified configuration.
//
// Parameters:
//   - limit: Maximum number of tokens in the bucket (must be > 0)
//   - algoName: Algorithm to use for rate limiting (currently supports TokenBucket)
//   - timer: Ticker for token refill intervals (nil for default 100ms interval)
//
// Returns:
//   - *Limiter: Configured limiter instance
//   - error: Error if invalid parameters are provided
//
// Example:
//
//	limiter, err := NewLimiter(10, TokenBucket, time.NewTicker(time.Second))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer limiter.Stop()
func NewLimiter(limit int32, algoName AlgorithmName, timer *time.Ticker) (*Limiter, error) {
	if limit < 1 {
		return nil, errors.New("limit must be greater than 0")
	}
	// default timer
	if timer == nil {
		timer = time.NewTicker(time.Millisecond * 100)
	}

	var algo LimitAlgorithm
	switch algoName {
	case TokenBucket:
		algo = newTokenBucket(limit, timer)
	default:
		return nil, errors.New("unknown algorithm")
	}

	return &Limiter{
		LimitAlgorithm: algo,
	}, nil
}

// Limit returns an HTTP middleware that applies rate limiting to incoming requests.
// If the rate limit is exceeded, it returns HTTP 429 (Too Many Requests).
//
// Parameters:
//   - next: The HTTP handler to wrap with rate limiting
//
// Returns:
//   - http.Handler: Wrapped handler with rate limiting applied
//
// Example:
//
//	handler := limiter.Limit(http.HandlerFunc(myHandler))
//	http.Handle("/api", handler)
func (l *Limiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if l.tryPass() {
			next.ServeHTTP(w, r)
		} else {
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
		}
	})
}
