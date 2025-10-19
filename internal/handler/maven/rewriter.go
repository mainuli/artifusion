package maven

import (
	"bytes"
	"net/http"
	"strings"
)

// determineProxyURL determines the proxy URL for Maven handler
// Constructs URL dynamically from request headers + protocol config
// Returns the full proxy URL including the path prefix (e.g., https://maven.example.com/maven)
func (h *Handler) determineProxyURL(r *http.Request) string {
	return h.getEffectiveBaseURL(r)
}

// rewriteBody rewrites URLs in response body
func (h *Handler) rewriteBody(body []byte, readBackendURL, writeBackendURL, proxyURL string) []byte {
	// Replace all occurrences of backend URLs with proxy public URL
	rewritten := body

	// Rewrite read backend URL
	if readBackendURL != "" {
		rewritten = bytes.ReplaceAll(rewritten,
			[]byte(readBackendURL),
			[]byte(proxyURL))
	}

	// Rewrite write backend URL
	if writeBackendURL != "" {
		rewritten = bytes.ReplaceAll(rewritten,
			[]byte(writeBackendURL),
			[]byte(proxyURL))
	}

	h.logger.Debug().
		Int("original_size", len(body)).
		Int("rewritten_size", len(rewritten)).
		Msg("Body rewritten")

	return rewritten
}

// rewriteURL rewrites a URL from backend to proxy
func (h *Handler) rewriteURL(url, readBackendURL, writeBackendURL, proxyURL string) string {
	// Replace backend URL with proxy URL
	rewritten := url

	if strings.HasPrefix(url, readBackendURL) {
		rewritten = strings.Replace(url, readBackendURL, proxyURL, 1)
	} else if strings.HasPrefix(url, writeBackendURL) {
		rewritten = strings.Replace(url, writeBackendURL, proxyURL, 1)
	}

	if rewritten != url {
		h.logger.Debug().
			Str("original", url).
			Str("rewritten", rewritten).
			Msg("URL rewritten")
	}

	return rewritten
}

// shouldRewriteBody determines if response body should be rewritten
func (h *Handler) shouldRewriteBody(contentType string) bool {
	// Rewrite XML and text metadata files
	contentType = strings.ToLower(contentType)

	return strings.Contains(contentType, "xml") ||
		strings.Contains(contentType, "text/plain") ||
		strings.Contains(contentType, "text/html") ||
		strings.Contains(contentType, "application/x-maven-pom+xml")
}
