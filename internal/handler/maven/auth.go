package maven

import (
	"fmt"
	"net/http"

	"github.com/mainuli/artifusion/internal/auth"
	"github.com/mainuli/artifusion/internal/config"
)

// authenticateClient validates the client's GitHub PAT using shared authenticator
func (h *Handler) authenticateClient(r *http.Request) (*auth.AuthResult, *http.Request, error) {
	authResult, newReq, err := h.authenticator.AuthenticateAndInjectContext(r)
	if err != nil {
		return nil, r, err
	}

	return authResult, newReq, nil
}

// handleAuthError returns a Maven-compliant error response
func (h *Handler) handleAuthError(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.Warn().Err(err).
		Str("path", r.URL.Path).
		Str("remote_addr", r.RemoteAddr).
		Msg("Authentication failed")

	// Set WWW-Authenticate challenge header
	realm := h.config.ClientAuth.Realm
	if realm == "" {
		realm = "Artifusion Maven Repository"
	}

	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
	w.WriteHeader(http.StatusUnauthorized)
	if _, writeErr := w.Write([]byte("Authentication required\n")); writeErr != nil {
		h.logger.Error().Err(writeErr).Msg("Failed to write authentication error response")
	}
}

// injectBackendAuth injects backend authentication credentials
func (h *Handler) injectBackendAuth(r *http.Request, backend *config.MavenBackendConfig) {
	if backend.Auth == nil {
		return
	}

	auth.InjectAuthCredentials(r, backend.Auth)
}
