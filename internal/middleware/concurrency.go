package middleware

import (
	"net/http"
	"sync/atomic"

	"github.com/mainuli/artifusion/internal/errors"
)

// ConcurrencyLimiter limits the number of concurrent requests using a semaphore pattern
type ConcurrencyLimiter struct {
	semaphore     chan struct{}
	maxConcurrent int
	active        atomic.Int32 // Track active requests for metrics
}

// NewConcurrencyLimiter creates a new concurrency limiter
func NewConcurrencyLimiter(maxConcurrent int) *ConcurrencyLimiter {
	return &ConcurrencyLimiter{
		semaphore:     make(chan struct{}, maxConcurrent),
		maxConcurrent: maxConcurrent,
	}
}

// Middleware returns a middleware handler that limits concurrent requests
func (cl *ConcurrencyLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case cl.semaphore <- struct{}{}:
			// Successfully acquired semaphore slot
			cl.active.Add(1)
			defer func() {
				<-cl.semaphore // Release semaphore slot
				cl.active.Add(-1)
			}()

			next.ServeHTTP(w, r)

		default:
			// Semaphore full, reject request
			errors.ErrorResponse(w, errors.ErrTooManyConcurrentRequests)
		}
	})
}

// ActiveRequests returns the current number of active requests
func (cl *ConcurrencyLimiter) ActiveRequests() int32 {
	return cl.active.Load()
}

// MaxConcurrent returns the maximum allowed concurrent requests
func (cl *ConcurrencyLimiter) MaxConcurrent() int {
	return cl.maxConcurrent
}
