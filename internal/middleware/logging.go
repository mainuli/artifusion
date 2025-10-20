package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/mainuli/artifusion/internal/utils"
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

// sanitizeHeaders redacts sensitive headers to prevent leaking secrets into logs.
// Returns a sanitized copy safe for logging.
func sanitizeHeaders(headers http.Header) map[string]interface{} {
	// Sensitive headers that must be redacted to prevent credential leakage
	sensitiveHeaders := map[string]bool{
		"authorization":       true,
		"cookie":              true,
		"set-cookie":          true,
		"x-auth-token":        true,
		"x-api-key":           true,
		"proxy-authorization": true,
		"x-csrf-token":        true,
		"x-session-token":     true,
	}

	sanitized := make(map[string]interface{})
	for key, values := range headers {
		lowerKey := strings.ToLower(key)
		if sensitiveHeaders[lowerKey] {
			sanitized[key] = "[REDACTED]"
		} else {
			// Safe headers are logged verbatim
			sanitized[key] = values
		}
	}
	return sanitized
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
				Str("remote_addr", utils.GetClientIP(r)).
				Str("user_agent", r.UserAgent())

			if includeHeaders {
				// SECURITY: Use sanitizeHeaders to prevent leaking Authorization, Cookie, etc.
				event = event.Interface("headers", sanitizeHeaders(r.Header))
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
				Str("remote_addr", utils.GetClientIP(r))

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
