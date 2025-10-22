package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mainuli/artifusion/internal/utils"
	"github.com/rs/zerolog"
)

// responseWriter wraps http.ResponseWriter to capture status and bytes written
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

// isHealthEndpoint checks if the request is for a health check endpoint
func isHealthEndpoint(path string) bool {
	return path == "/health" || path == "/ready"
}

// getLogEvent returns the appropriate log event based on the request path
// Health endpoints use debug level, all others use info level
func getLogEvent(logger zerolog.Logger, path string) *zerolog.Event {
	if isHealthEndpoint(path) {
		return logger.Debug()
	}
	return logger.Info()
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
			clientIP := utils.GetClientIP(r)

			// Log request start - format: IP "METHOD /path" request_id=... user_agent=...
			requestLine := fmt.Sprintf("%s \"%s %s\"", clientIP, r.Method, r.URL.Path)

			// Use debug level for health endpoints, info level for others
			event := getLogEvent(logger, r.URL.Path).
				Str("request_id", requestID).
				Str("user_agent", r.UserAgent())

			if includeHeaders {
				// SECURITY: Use sanitizeHeaders to prevent leaking Authorization, Cookie, etc.
				event = event.Interface("headers", sanitizeHeaders(r.Header))
			}

			event.Msg(requestLine)

			// Process request
			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := time.Since(start)

			// Get username from context if authenticated
			username := GetUsername(r.Context())

			// Log request completion - format: IP "METHOD /path" status=200 duration=0.16ms bytes=107
			completionLine := fmt.Sprintf("%s \"%s %s\"", clientIP, r.Method, r.URL.Path)

			// Use debug level for health endpoints, info level for others
			completionEvent := getLogEvent(logger, r.URL.Path).
				Str("request_id", requestID).
				Int("status", wrapped.status).
				Dur("duration", duration).
				Int64("bytes", wrapped.bytesWritten).
				Str("user_agent", r.UserAgent())

			if username != "" {
				completionEvent = completionEvent.Str("username", username)
			}

			completionEvent.Msg(completionLine)
		})
	}
}
