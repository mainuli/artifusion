package oci

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mainuli/artifusion/internal/auth"
	"github.com/mainuli/artifusion/internal/config"
)

// selectBackendAndProxy determines the appropriate backend and proxies the request
func (h *Handler) selectBackendAndProxy(w http.ResponseWriter, r *http.Request, authResult *auth.AuthResult) error {
	path := r.URL.Path
	method := r.Method

	// Check if this is a write operation
	if h.isWriteOperation(method, path) {
		// Write operations go directly to push backend (registry:2)
		backend := &h.config.PushBackend

		h.logger.Debug().
			Str("backend", backend.Name).
			Str("url", backend.URL).
			Str("operation", "write").
			Msg("Routing to push backend")

		// Inject backend auth
		h.injectBackendAuth(r, backend)

		// Proxy directly (no path rewriting for push backend)
		return h.proxyTransparent(w, r, backend, path)
	}

	// Read operations: cascade through pull backends with fallback
	// Use array index order for cascade (no explicit priority field)
	backends := h.config.PullBackends

	// Edge case: no backends configured (shouldn't happen due to validation)
	if len(backends) == 0 {
		h.logger.Error().Msg("No pull backends configured")
		w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)

		errResponse := OCIError{
			Errors: []OCIErrorDetail{
				{
					Code:    "UNAVAILABLE",
					Message: "registry service unavailable",
					Detail:  "No pull backends configured",
				},
			},
		}

		if err := encodeJSON(w, errResponse); err != nil {
			h.logger.Error().Err(err).Msg("Failed to encode error response")
			return err
		}
		return nil
	}

	h.logger.Debug().
		Int("backend_count", len(backends)).
		Str("operation", "read").
		Msg("Attempting cascading read backends")

	// Track cascade attempts for better error reporting
	backendsTried := 0
	backendsSkipped := 0

	// Try each backend in order
	for i := range backends {
		backend := &backends[i]

		// Skip GHCR if org doesn't match scope or authenticated user's org
		if backend.UpstreamNamespace == "ghcr.io" && !h.shouldTryGHCR(path, backend, authResult) {
			h.logger.Debug().
				Str("backend", backend.Name).
				Str("path", path).
				Msg("Skipping GHCR backend - org not in scope")
			backendsSkipped++
			continue
		}

		// Count this backend as tried
		backendsTried++

		// Rewrite path for oci-registry namespace routing
		rewrittenPath := h.rewritePath(path, backend)

		h.logger.Debug().
			Str("backend", backend.Name).
			Str("url", backend.URL).
			Int("attempt", i+1).
			Str("original_path", path).
			Str("rewritten_path", rewrittenPath).
			Msg("Trying pull backend")

		// Inject backend auth
		h.injectBackendAuth(r, backend)

		// Execute proxy request WITHOUT streaming the response
		resp, err := h.executeProxyRequest(r, backend, rewrittenPath)

		if err == nil && resp != nil {
			// Ensure response body is always closed (defense in depth)
			// StreamResponse will read the body, but we defer close to ensure cleanup
			bodyCloser := resp.HTTPResp.Body
			bodyClosed := false
			closeBody := func() {
				if !bodyClosed && bodyCloser != nil {
					if closeErr := bodyCloser.Close(); closeErr != nil {
						h.logger.Warn().Err(closeErr).Msg("Failed to close response body")
					}
					bodyClosed = true
				}
			}
			defer closeBody()

			// Check if request was successful
			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				h.logger.Debug().
					Str("backend", backend.Name).
					Int("status", resp.StatusCode).
					Msg("Backend returned success, streaming response")

				// Stream the successful response to client
				_, streamErr := h.proxyClient.StreamResponse(w, resp, true)
				if streamErr != nil {
					h.logger.Error().Err(streamErr).Msg("Failed to stream response")
					return streamErr
				}
				return nil
			}

			// Treat 404, 401, 403, and 5xx errors as "not found" and try next backend
			// 404 = Not Found
			// 401/403 = No access (treat as not found for cascade)
			// 5xx = Backend error (try next backend)
			if resp.StatusCode == http.StatusNotFound ||
				resp.StatusCode == http.StatusUnauthorized ||
				resp.StatusCode == http.StatusForbidden ||
				resp.StatusCode >= 500 {

				h.logger.Debug().
					Str("backend", backend.Name).
					Int("status", resp.StatusCode).
					Str("namespace", backend.UpstreamNamespace).
					Msg("Backend returned error, trying next")
				// Body will be closed by defer
			} else {
				// Other 4xx errors: stream error response to client
				h.logger.Warn().
					Str("backend", backend.Name).
					Int("status", resp.StatusCode).
					Msg("Backend returned client error, streaming error response")

				// Stream the error response to client
				_, streamErr := h.proxyClient.StreamResponse(w, resp, true)
				if streamErr != nil {
					h.logger.Error().Err(streamErr).Msg("Failed to stream error response")
					return streamErr
				}
				return nil
			}
		} else if err != nil {
			// Network error or backend unreachable: try next backend
			h.logger.Warn().Err(err).
				Str("backend", backend.Name).
				Msg("Backend request failed, trying next")
		}
	}

	// All backends failed - provide specific error based on what happened
	var errDetail string
	var statusCode int

	if backendsTried == 0 && backendsSkipped > 0 {
		// All backends were skipped (e.g., all GHCR backends didn't match org scope)
		h.logger.Warn().
			Str("path", path).
			Int("backends_total", len(backends)).
			Int("backends_skipped", backendsSkipped).
			Msg("All backends skipped due to scope filtering")

		errDetail = fmt.Sprintf("Image not accessible: all %d backend(s) filtered by organization scope", backendsSkipped)
		statusCode = http.StatusNotFound
	} else if backendsTried == 0 {
		// No backends tried and none skipped (shouldn't happen, but defensive)
		h.logger.Error().
			Str("path", path).
			Int("backends_total", len(backends)).
			Msg("No backends tried (unexpected)")

		errDetail = "No backends available to serve request"
		statusCode = http.StatusServiceUnavailable
	} else {
		// Some backends were tried but all failed
		h.logger.Warn().
			Str("path", path).
			Int("backends_total", len(backends)).
			Int("backends_tried", backendsTried).
			Int("backends_skipped", backendsSkipped).
			Msg("All attempted backends failed")

		errDetail = fmt.Sprintf("Image not found in any of %d upstream registr%s",
			backendsTried,
			map[bool]string{true: "y", false: "ies"}[backendsTried == 1])
		statusCode = http.StatusNotFound
	}

	// Return error response
	w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errResponse := OCIError{
		Errors: []OCIErrorDetail{
			{
				Code:    "NAME_UNKNOWN",
				Message: "repository name not known to registry",
				Detail:  errDetail,
			},
		},
	}

	if err := encodeJSON(w, errResponse); err != nil {
		h.logger.Error().Err(err).Msg("Failed to encode error response")
		return err
	}
	return nil
}

// isWriteOperation determines if the request is a write operation
func (h *Handler) isWriteOperation(method, path string) bool {
	// 1. Create upload session
	if method == http.MethodPost && strings.Contains(path, "/blobs/uploads") {
		return true
	}

	// 2. Upload chunks / commit (sticky by UUID)
	if strings.Contains(path, "/blobs/uploads/") {
		return true
	}

	// 3. Push manifest
	if method == http.MethodPut && strings.Contains(path, "/manifests/") {
		return true
	}

	// 4. Delete operations
	if method == http.MethodDelete {
		return true
	}

	return false
}

// encodeJSON writes JSON response with proper error handling
func encodeJSON(w http.ResponseWriter, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("failed to encode JSON response: %w", err)
	}
	return nil
}

// shouldTryGHCR determines if we should try the GHCR backend for this image
func (h *Handler) shouldTryGHCR(path string, backend *config.OCIBackendConfig, _ *auth.AuthResult) bool {
	imageOrg := extractOrgFromPath(path)
	if imageOrg == "" {
		return false // Can't determine org, skip GHCR
	}

	// If scope is configured, check if image org is in scope
	if len(backend.Scope) > 0 {
		for _, scopeOrg := range backend.Scope {
			if scopeOrg == "*" || imageOrg == scopeOrg {
				h.logger.Debug().
					Str("image_org", imageOrg).
					Strs("scope", backend.Scope).
					Msg("Image org matches backend scope")
				return true
			}
		}
		h.logger.Debug().
			Str("image_org", imageOrg).
			Strs("scope", backend.Scope).
			Msg("Image org not in backend scope")
		return false
	}

	// No scope configured - fall back to requiredOrg from auth
	requiredOrg := h.authenticator.GetRequiredOrg()
	if requiredOrg == "" {
		return true // No org requirement, allow all
	}

	return imageOrg == requiredOrg
}

// extractOrgFromPath extracts the organization/user from the image path
// /v2/myorg/myimage/manifests/latest -> myorg
// /v2/myuser/myrepo/blobs/sha256:abc -> myuser
func extractOrgFromPath(path string) string {
	// Remove /v2/ prefix
	path = strings.TrimPrefix(path, "/v2/")
	if path == "" || path == "/" {
		return ""
	}

	// Split and get first component
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return ""
	}

	// First part is the org/user
	return parts[0]
}
