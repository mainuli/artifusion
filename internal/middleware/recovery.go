package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/rs/zerolog"
)

// Recovery is a middleware that recovers from panics and logs the error
func Recovery(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Get request ID for correlation
					requestID := GetRequestID(r.Context())

					// Get stack trace
					stack := debug.Stack()

					// Log panic with full details
					logger.Error().
						Str("request_id", requestID).
						Str("method", r.Method).
						Str("path", r.URL.Path).
						Str("remote_addr", r.RemoteAddr).
						Interface("panic", err).
						Bytes("stack_trace", stack).
						Msg("Panic recovered")

					// Return 500 Internal Server Error
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
