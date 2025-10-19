package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/mainuli/artifusion/internal/errors"
)

// Timeout creates middleware that enforces a maximum request duration
// This provides a global timeout for all requests to prevent indefinite hangs
func Timeout(duration time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create context with timeout
			ctx, cancel := context.WithTimeout(r.Context(), duration)
			defer cancel()

			// Channel to signal when request handling is complete
			done := make(chan struct{})
			panicChan := make(chan interface{}, 1)

			// Run the next handler in a goroutine
			go func() {
				defer func() {
					// Recover from panics in handler
					if p := recover(); p != nil {
						panicChan <- p
					}
				}()

				// Execute the handler with timeout context
				next.ServeHTTP(w, r.WithContext(ctx))
				close(done)
			}()

			// Wait for completion, panic, or timeout
			select {
			case <-done:
				// Request completed successfully
				return

			case p := <-panicChan:
				// Panic occurred in handler - re-panic to be caught by recovery middleware
				panic(p)

			case <-ctx.Done():
				// Context cancelled or deadline exceeded
				if ctx.Err() == context.DeadlineExceeded {
					// Request exceeded maximum allowed duration
					errors.ErrorResponse(w, errors.ErrBackendTimeout.WithMessage(
						"Request exceeded maximum allowed duration"))
				}
				// Note: If context was cancelled for other reasons, the error
				// will be handled by the handler itself
			}
		})
	}
}
