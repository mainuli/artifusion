package npm

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mainuli/artifusion/internal/auth"
	"github.com/mainuli/artifusion/internal/config"
)

// npmErrorResponse represents an NPM-compatible error response
type npmErrorResponse struct {
	Error string `json:"error"`
}

// authenticateClient validates the client's GitHub PAT using shared authenticator
func (h *Handler) authenticateClient(r *http.Request) (*auth.AuthResult, *http.Request, error) {
	authResult, newReq, err := h.authenticator.AuthenticateAndInjectContext(r)
	if err != nil {
		return nil, r, err
	}

	return authResult, newReq, nil
}

// handleAuthError returns an NPM-compliant error response
func (h *Handler) handleAuthError(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.Warn().Err(err).
		Str("path", r.URL.Path).
		Str("remote_addr", r.RemoteAddr).
		Msg("Authentication failed")

	// Set WWW-Authenticate challenge header with Bearer scheme (NPM standard)
	realm := h.config.ClientAuth.Realm
	if realm == "" {
		realm = "Artifusion NPM Registry"
	}

	// NPM uses Bearer token authentication
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="%s"`, realm))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)

	// Return NPM-compatible error response
	errResp := npmErrorResponse{
		Error: "Authentication required. Please provide a valid GitHub Personal Access Token.",
	}

	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode error response")
	}
}

// injectBackendAuth injects backend authentication credentials
func (h *Handler) injectBackendAuth(r *http.Request, backend *config.NPMBackendConfig) {
	if backend.Auth == nil {
		return
	}

	auth.InjectAuthCredentials(r, backend.Auth)
}
