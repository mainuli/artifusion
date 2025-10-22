package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// AppError represents a structured application error with HTTP status code and error code
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
	Internal   error  `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Internal)
	}
	return e.Message
}

// Unwrap returns the wrapped error (for errors.Is and errors.As)
func (e *AppError) Unwrap() error {
	return e.Internal
}

// WithInternal wraps an internal error with additional context
func (e *AppError) WithInternal(err error) *AppError {
	return &AppError{
		Code:       e.Code,
		Message:    e.Message,
		StatusCode: e.StatusCode,
		Internal:   err,
	}
}

// WithMessage overrides the default message
func (e *AppError) WithMessage(message string) *AppError {
	return &AppError{
		Code:       e.Code,
		Message:    message,
		StatusCode: e.StatusCode,
		Internal:   e.Internal,
	}
}

// WithMessagef formats and overrides the default message
func (e *AppError) WithMessagef(format string, args ...interface{}) *AppError {
	return &AppError{
		Code:       e.Code,
		Message:    fmt.Sprintf(format, args...),
		StatusCode: e.StatusCode,
		Internal:   e.Internal,
	}
}

// Predefined errors for common scenarios
var (
	// Backend errors
	ErrBackendTimeout = &AppError{
		Code:       "BACKEND_TIMEOUT",
		Message:    "Backend request timeout",
		StatusCode: http.StatusGatewayTimeout,
	}

	// Rate limiting errors
	ErrGlobalRateLimitExceeded = &AppError{
		Code:       "GLOBAL_RATE_LIMIT_EXCEEDED",
		Message:    "Global rate limit exceeded, please try again later",
		StatusCode: http.StatusTooManyRequests,
	}

	ErrUserRateLimitExceeded = &AppError{
		Code:       "USER_RATE_LIMIT_EXCEEDED",
		Message:    "User rate limit exceeded, please try again later",
		StatusCode: http.StatusTooManyRequests,
	}

	// Protocol errors
	ErrProtocolNotSupported = &AppError{
		Code:       "PROTOCOL_NOT_SUPPORTED",
		Message:    "Protocol not supported",
		StatusCode: http.StatusNotFound,
	}

	// Server errors
	ErrInternal = &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    "Internal server error",
		StatusCode: http.StatusInternalServerError,
	}

	// Concurrency errors
	ErrTooManyConcurrentRequests = &AppError{
		Code:       "TOO_MANY_CONCURRENT_REQUESTS",
		Message:    "Too many concurrent requests",
		StatusCode: http.StatusServiceUnavailable,
	}
)

// ErrorResponse renders an error as an HTTP JSON response
// It handles both AppError and generic error types
func ErrorResponse(w http.ResponseWriter, err error) {
	appErr, ok := err.(*AppError)
	if !ok {
		// Convert generic error to internal error
		appErr = &AppError{
			Code:       "INTERNAL_ERROR",
			Message:    "Internal server error",
			StatusCode: http.StatusInternalServerError,
			Internal:   err,
		}
	}

	// Set content type to JSON
	w.Header().Set("Content-Type", "application/json")

	// Add error code header for easier client-side handling
	w.Header().Set("X-Error-Code", appErr.Code)

	// Write status code
	w.WriteHeader(appErr.StatusCode)

	// Create error response body
	response := map[string]string{
		"error":   appErr.Code,
		"message": appErr.Message,
	}

	// Encode and write response
	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		// If JSON encoding fails, write plain text as fallback
		http.Error(w, appErr.Message, appErr.StatusCode)
	}
}
