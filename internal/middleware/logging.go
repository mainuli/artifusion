package middleware

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// responseWriter wraps http.ResponseWriter to capture status code and bytes written
type responseWriter struct {
	http.ResponseWriter
	status       int
	bytesWritten int64
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

// Logger creates a structured logging middleware
func Logger(logger zerolog.Logger, includeHeaders bool, _ bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status and bytes
			wrapped := &responseWriter{
				ResponseWriter: w,
				status:         http.StatusOK, // Default status
			}

			// Get request ID from context
			requestID := GetRequestID(r.Context())

			// Log request start
			event := logger.Info().
				Str("request_id", requestID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Str("user_agent", r.UserAgent())

			if includeHeaders {
				event = event.Interface("headers", r.Header)
			}

			event.Msg("Request started")

			// Process request
			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := time.Since(start)

			// Get username from context if authenticated
			username := GetUsername(r.Context())

			// Log request completion
			completionEvent := logger.Info().
				Str("request_id", requestID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", wrapped.status).
				Int64("bytes_written", wrapped.bytesWritten).
				Dur("duration_ms", duration).
				Str("remote_addr", r.RemoteAddr)

			if username != "" {
				completionEvent = completionEvent.Str("username", username)
			}

			// Add status-based level
			if wrapped.status >= 500 {
				completionEvent.Msg("Request completed with server error")
			} else if wrapped.status >= 400 {
				completionEvent.Msg("Request completed with client error")
			} else {
				completionEvent.Msg("Request completed successfully")
			}
		})
	}
}
