package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

const (
	// RequestIDKey is the context key for request IDs
	RequestIDKey ContextKey = "request_id"
	// UsernameKey is the context key for authenticated usernames
	UsernameKey ContextKey = "username"
)

// RequestID injects a unique request ID into each request
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request already has X-Request-ID header
		requestID := r.Header.Get("X-Request-ID")

		// Generate new UUID if not present
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set response header
		w.Header().Set("X-Request-ID", requestID)

		// Add to request context
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID extracts the request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// GetUsername extracts the authenticated username from context
func GetUsername(ctx context.Context) string {
	if username, ok := ctx.Value(UsernameKey).(string); ok {
		return username
	}
	return ""
}

// SetUsername adds the authenticated username to the context
func SetUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, UsernameKey, username)
}
