# M RPS Limiter

Rate limiting library for Go with support for the next algorithms:
- **Token Bucket Algorithm**: Efficient rate limiting with burst capacity

## Installation

```bash
go get github.com/Murolando/m_rps_limiter
```

## Quick Start

```go
package main

import (
    "log"
    "net/http"
    "time"

    "github.com/Murolando/m_rps_limiter/pkg/limiter"
)

func main() {
    // Create a rate limiter with 10 requests per refill interval
    rateLimiter, err := limiter.NewLimiter(
        10,                                    // limit
        limiter.TokenBucket,                   // algorithm
        time.NewTicker(time.Millisecond*100),  // refill every 100ms
    )
    if err != nil {
        log.Fatal("Failed to create limiter:", err)
    }
    defer rateLimiter.Stop()

    // Apply rate limiting to HTTP handlers
    http.Handle("/api", rateLimiter.Limit(http.HandlerFunc(apiHandler)))
    
    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("API Response"))
}
```

## Algorithm Details

### Token Bucket

The Token Bucket algorithm maintains a bucket of tokens that are consumed by incoming requests:

1. **Initialization**: Bucket starts full with `limit` tokens
2. **Request Processing**: Each request consumes 1 token
3. **Token Refill**: Tokens are added back at regular intervals
4. **Rate Limiting**: Requests are rejected when no tokens are available

## Performance

The library uses efficient design patterns:

- **Atomic Operations**: Thread-safe token management without mutexes
- **Simple Logic**: Minimal operations in the hot path  
- **Background Refill**: Separate goroutine handles token replenishment

**Note**: While optimized for correctness and simplicity, this implementation prioritizes readability over maximum performance. For extremely high-throughput scenarios, consider benchmarking against your specific requirements.

Run benchmarks:
```bash
go test -bench=. ./pkg/limiter/
```

## Testing

Run the test suite:
```bash
go test ./...
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Roadmap

- [ ] Sliding Window algorithm
- [ ] Fixed Window algorithm  
- [ ] Leaky Bucket algorithm
- [ ] Concurrency rate limiting
- [ ] Metrics and monitoring