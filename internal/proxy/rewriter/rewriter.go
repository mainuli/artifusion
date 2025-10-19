package rewriter

import (
	"strings"

	"github.com/mainuli/artifusion/internal/proxy"
	"github.com/rs/zerolog"
)

// URLRewriter handles rewriting backend URLs to public URLs in response headers
type URLRewriter struct {
	publicURL string
	logger    zerolog.Logger
}

// New creates a new URLRewriter with the given public URL
func New(publicURL string, logger zerolog.Logger) *URLRewriter {
	return &URLRewriter{
		publicURL: publicURL,
		logger:    logger,
	}
}

// RewriteResponseHeaders rewrites Location and WWW-Authenticate headers in the response
func (r *URLRewriter) RewriteResponseHeaders(resp *proxy.Response, backend proxy.BackendConfig) {
	r.RewriteLocation(resp, backend)
	r.RewriteWWWAuthenticate(resp)
}

// RewriteLocation rewrites Location headers from backend URLs to public URL
func (r *URLRewriter) RewriteLocation(resp *proxy.Response, backend proxy.BackendConfig) {
	location := resp.Headers.Get("Location")
	if location == "" {
		return
	}

	var rewritten string

	// Handle relative paths
	if len(location) > 0 && location[0] == '/' {
		rewritten = r.publicURL + location
		resp.Headers.Set("Location", rewritten)
		r.logger.Debug().
			Str("original", location).
			Str("rewritten", rewritten).
			Msg("Rewrote relative Location header")
		return
	}

	// Handle absolute URLs - replace backend URL with public URL
	backendURL := backend.GetURL()
	if backendURL != "" && strings.HasPrefix(location, backendURL) {
		rewritten = r.publicURL + strings.TrimPrefix(location, backendURL)
		resp.Headers.Set("Location", rewritten)
		r.logger.Debug().
			Str("original", location).
			Str("rewritten", rewritten).
			Str("backend_url", backendURL).
			Str("public_url", r.publicURL).
			Msg("Rewrote absolute Location header")
	}
}

// RewriteWWWAuthenticate rewrites WWW-Authenticate realm to point to public URL
func (r *URLRewriter) RewriteWWWAuthenticate(resp *proxy.Response) {
	authHeader := resp.Headers.Get("WWW-Authenticate")
	if authHeader == "" {
		return
	}

	// Rewrite realm if present in WWW-Authenticate header
	// Example: Bearer realm="http://backend:5000/v2/token"
	// Becomes: Bearer realm="https://example.org/v2/token"
	if strings.Contains(authHeader, "realm=") {
		// Extract and replace realm URL
		rewritten := r.rewriteRealmInAuthHeader(authHeader)
		if rewritten != authHeader {
			resp.Headers.Set("WWW-Authenticate", rewritten)
			r.logger.Debug().
				Str("original", authHeader).
				Str("rewritten", rewritten).
				Str("public_url", r.publicURL).
				Msg("Rewrote WWW-Authenticate header")
		}
	}
}

// rewriteRealmInAuthHeader rewrites the realm parameter in WWW-Authenticate header
func (r *URLRewriter) rewriteRealmInAuthHeader(authHeader string) string {
	// Find realm="..." in the header
	realmStart := strings.Index(authHeader, "realm=\"")
	if realmStart == -1 {
		return authHeader
	}

	// Find the closing quote
	realmValueStart := realmStart + len("realm=\"")
	realmEnd := strings.Index(authHeader[realmValueStart:], "\"")
	if realmEnd == -1 {
		return authHeader
	}

	// Extract original realm URL
	originalRealm := authHeader[realmValueStart : realmValueStart+realmEnd]

	// Extract path from original realm (everything after the domain/port)
	// Example: http://registry:5000/v2/token → /v2/token
	var path string
	if idx := strings.Index(originalRealm, "://"); idx != -1 {
		// Skip scheme
		remaining := originalRealm[idx+3:]
		// Find path start
		if slashIdx := strings.Index(remaining, "/"); slashIdx != -1 {
			path = remaining[slashIdx:]
		}
	} else if len(originalRealm) > 0 && originalRealm[0] == '/' {
		// Already a path
		path = originalRealm
	}

	// If we couldn't extract a path, return original
	if path == "" {
		return authHeader
	}

	// Build new realm with public URL
	newRealm := r.publicURL + path

	// Replace in the header
	return authHeader[:realmValueStart] + newRealm + authHeader[realmValueStart+realmEnd:]
}
