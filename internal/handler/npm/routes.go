package npm

import (
	"fmt"
	"net/http"

	"github.com/mainuli/artifusion/internal/auth"
)

// selectBackendAndProxy determines the appropriate backend and proxies the request
func (h *Handler) selectBackendAndProxy(w http.ResponseWriter, r *http.Request, authResult *auth.AuthResult) error {
	// Validate inputs
	if r == nil {
		return fmt.Errorf("request is nil")
	}
	if authResult == nil {
		return fmt.Errorf("auth result is nil")
	}

	method := r.Method

	// Use single backend for both read and write operations (like Maven pattern)
	backend := &h.config.Backend

	// Validate backend configuration
	if backend.URL == "" {
		h.logger.Error().Msg("Backend URL is not configured")
		return fmt.Errorf("backend URL is not configured")
	}

	// Log operation type for debugging
	operationType := "read"
	if h.isWriteOperation(method) {
		operationType = "write"
	}

	h.logger.Debug().
		Str("backend", backend.Name).
		Str("url", backend.URL).
		Str("operation", operationType).
		Str("username", authResult.Username).
		Msg("Routing to NPM backend")

	// Note: Backend authentication is handled by proxy client
	// Proxy with URL rewriting
	return h.proxyWithRewriting(w, r, backend)
}

// isWriteOperation determines if the request is a write operation
func (h *Handler) isWriteOperation(method string) bool {
	// Write operations use PUT or POST
	return method == http.MethodPut || method == http.MethodPost
}
