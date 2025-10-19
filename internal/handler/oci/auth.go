package oci

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mainuli/artifusion/internal/auth"
	"github.com/mainuli/artifusion/internal/config"
)

// OCIError represents an OCI registry error response
type OCIError struct {
	Errors []OCIErrorDetail `json:"errors"`
}

// OCIErrorDetail represents a single error in an OCI error response
type OCIErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// authenticateClient validates the client's GitHub PAT using shared authenticator
func (h *Handler) authenticateClient(r *http.Request) (*auth.AuthResult, *http.Request, error) {
	authResult, newReq, err := h.authenticator.AuthenticateAndInjectContext(r)
	if err != nil {
		return nil, r, err
	}

	return authResult, newReq, nil
}

// handleAuthError returns an OCI-compliant error response
func (h *Handler) handleAuthError(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.Warn().Err(err).
		Str("path", r.URL.Path).
		Str("remote_addr", r.RemoteAddr).
		Msg("Authentication failed")

	// Set WWW-Authenticate challenge header
	// If realm is empty, use Basic auth (direct authentication without token exchange)
	// Otherwise, use Bearer auth with token endpoint
	realm := h.config.ClientAuth.Realm
	service := h.config.ClientAuth.Service
	if service == "" {
		service = "artifusion"
	}

	var authHeader string
	if realm == "" {
		// Use Basic auth for direct GitHub PAT authentication
		authHeader = fmt.Sprintf(`Basic realm="%s"`, service)
	} else {
		// Use Bearer auth with token endpoint
		authHeader = fmt.Sprintf(`Bearer realm="%s",service="%s"`, realm, service)
	}

	w.Header().Set("WWW-Authenticate", authHeader)
	w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)

	// Return OCI error response
	errResponse := OCIError{
		Errors: []OCIErrorDetail{
			{
				Code:    "UNAUTHORIZED",
				Message: "authentication required",
				Detail:  "GitHub PAT required via Bearer or Basic auth",
			},
		},
	}

	if err := json.NewEncoder(w).Encode(errResponse); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode auth error response")
	}
}

// injectBackendAuth injects backend authentication credentials
func (h *Handler) injectBackendAuth(r *http.Request, backend *config.OCIBackendConfig) {
	if backend.Auth == nil {
		r.Header.Del("Authorization")
		return
	}

	auth.InjectAuthCredentials(r, backend.Auth)
}
