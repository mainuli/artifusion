package maven

import (
	"net/http"

	"github.com/mainuli/artifusion/internal/auth"
)

// selectBackendAndProxy determines the appropriate backend and proxies the request
func (h *Handler) selectBackendAndProxy(w http.ResponseWriter, r *http.Request, authResult *auth.AuthResult) error {
	method := r.Method

	// Use single backend for both read and write operations
	backend := &h.config.Backend

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
		Msg("Routing to Maven backend")

	// Inject backend auth
	h.injectBackendAuth(r, backend)

	// Proxy with URL rewriting
	return h.proxyWithRewriting(w, r, backend)
}

// isWriteOperation determines if the request is a write operation
func (h *Handler) isWriteOperation(method string) bool {
	// Write operations use PUT or POST
	return method == http.MethodPut || method == http.MethodPost
}
