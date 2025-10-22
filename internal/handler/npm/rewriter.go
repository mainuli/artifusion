package npm

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

const (
	// MaxJSONRewriteSize is the maximum size of JSON body to rewrite (10MB)
	// Larger responses will use fallback text rewriting
	MaxJSONRewriteSize = 10 * 1024 * 1024

	// MaxRecursionDepth limits recursion depth in metadata rewriting
	MaxRecursionDepth = 10
)

// urlFields contains common NPM registry URL fields that should be rewritten
// Declared at package level to avoid repeated allocations during recursion
var urlFields = []string{"tarball", "url", "homepage", "repository", "bugs"}

// determineProxyURL determines the proxy URL for NPM handler
// Constructs URL dynamically from request headers + protocol config
// Returns the full proxy URL including the path prefix (e.g., https://npm.example.com/npm)
func (h *Handler) determineProxyURL(r *http.Request) string {
	return h.getEffectiveBaseURL(r)
}

// rewritePackageJSON rewrites URLs in NPM package JSON metadata
// This handles both individual package metadata and bulk responses
func (h *Handler) rewritePackageJSON(body []byte, backendURL, proxyURL string) ([]byte, error) {
	// Early return for empty body
	if len(body) == 0 {
		return body, nil
	}

	// Check size limit to prevent memory issues with large responses
	if len(body) > MaxJSONRewriteSize {
		h.logger.Warn().
			Int("size", len(body)).
			Int("max_size", MaxJSONRewriteSize).
			Msg("Response body exceeds JSON rewrite size limit, using text fallback")
		return h.rewriteBody(body, backendURL, proxyURL), nil
	}

	// Parse JSON to check structure
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		// Not valid JSON, use text fallback
		h.logger.Debug().Err(err).Msg("Failed to parse JSON, using text fallback")
		return h.rewriteBody(body, backendURL, proxyURL), nil
	}

	// Convert to generic map to handle different response types
	switch data := jsonData.(type) {
	case map[string]interface{}:
		// Single package metadata or error response
		h.rewritePackageMetadata(data, backendURL, proxyURL, 0)

	case []interface{}:
		// Array of packages (search results)
		for _, item := range data {
			if pkgMap, ok := item.(map[string]interface{}); ok {
				h.rewritePackageMetadata(pkgMap, backendURL, proxyURL, 0)
			}
		}

	default:
		// Unknown structure, return as-is
		return body, nil
	}

	// Marshal back to JSON
	rewritten, err := json.Marshal(jsonData)
	if err != nil {
		h.logger.Warn().Err(err).Msg("Failed to marshal rewritten JSON, using text fallback")
		return h.rewriteBody(body, backendURL, proxyURL), nil
	}

	h.logger.Debug().
		Int("original_size", len(body)).
		Int("rewritten_size", len(rewritten)).
		Msg("Package JSON rewritten")

	return rewritten, nil
}

// rewritePackageMetadata recursively rewrites URLs in a package metadata object
func (h *Handler) rewritePackageMetadata(data map[string]interface{}, backendURL, proxyURL string, depth int) {
	// Prevent excessive recursion
	if depth > MaxRecursionDepth {
		h.logger.Warn().
			Int("depth", depth).
			Msg("Max recursion depth reached in package metadata rewriting")
		return
	}

	// Rewrite tarball URL if present
	if tarball, ok := data["tarball"].(string); ok {
		data["tarball"] = h.rewriteURL(tarball, backendURL, proxyURL)
	}

	// Rewrite dist object (contains tarball URL)
	if dist, ok := data["dist"].(map[string]interface{}); ok {
		if tarball, ok := dist["tarball"].(string); ok {
			dist["tarball"] = h.rewriteURL(tarball, backendURL, proxyURL)
		}
	}

	// Rewrite versions object (contains multiple version metadata)
	if versions, ok := data["versions"].(map[string]interface{}); ok {
		for _, versionData := range versions {
			if versionMap, ok := versionData.(map[string]interface{}); ok {
				h.rewritePackageMetadata(versionMap, backendURL, proxyURL, depth+1)
			}
		}
	}

	// Rewrite _attachments object (contains tarball data for publish)
	if attachments, ok := data["_attachments"].(map[string]interface{}); ok {
		for _, attachment := range attachments {
			if attMap, ok := attachment.(map[string]interface{}); ok {
				if url, ok := attMap["url"].(string); ok {
					attMap["url"] = h.rewriteURL(url, backendURL, proxyURL)
				}
			}
		}
	}

	// Targeted rewriting: only rewrite known URL fields to avoid expensive iteration
	// Use scheme-agnostic matching to handle http/https mismatches
	backendHost := extractHostFromURL(backendURL)
	for _, field := range urlFields {
		if strValue, ok := data[field].(string); ok {
			if strings.Contains(strValue, backendHost) {
				data[field] = h.rewriteURL(strValue, backendURL, proxyURL)
			}
		}
	}
}

// rewriteURL rewrites a single URL from backend to proxy
func (h *Handler) rewriteURL(url, backendURL, proxyURL string) string {
	// Extract host:port from backend URL (scheme-agnostic)
	// Backend might be http://artifusion-verdaccio:4873
	// But Verdaccio might return https://artifusion-verdaccio:4873
	backendHost := extractHostFromURL(backendURL)

	// Check if URL contains the backend host (scheme-agnostic)
	if strings.Contains(url, backendHost) {
		// Replace with proxy URL, preserving everything after the host:port
		// Check which scheme is present to avoid redundant replacements
		var rewritten string
		if strings.Contains(url, "http://"+backendHost) {
			rewritten = strings.Replace(url, "http://"+backendHost, proxyURL, 1)
		} else if strings.Contains(url, "https://"+backendHost) {
			rewritten = strings.Replace(url, "https://"+backendHost, proxyURL, 1)
		} else {
			// Host found without scheme prefix, return as-is
			return url
		}

		h.logger.Debug().
			Str("original", url).
			Str("rewritten", rewritten).
			Msg("URL rewritten")

		return rewritten
	}

	// URL doesn't point to our backend, return unchanged
	return url
}

// rewriteBody is a simpler fallback method for text-based rewriting
// Used when JSON parsing fails but content is still text
func (h *Handler) rewriteBody(body []byte, backendURL, proxyURL string) []byte {
	// Extract host:port from backend URL (scheme-agnostic)
	backendHost := extractHostFromURL(backendURL)

	// Replace all occurrences of backend host with proxy URL (both http and https)
	rewritten := bytes.ReplaceAll(body, []byte("http://"+backendHost), []byte(proxyURL))
	rewritten = bytes.ReplaceAll(rewritten, []byte("https://"+backendHost), []byte(proxyURL))

	if !bytes.Equal(body, rewritten) {
		h.logger.Debug().
			Int("original_size", len(body)).
			Int("rewritten_size", len(rewritten)).
			Msg("Body rewritten (text mode)")
	}

	return rewritten
}

// shouldRewriteBody determines if response body should be rewritten
func (h *Handler) shouldRewriteBody(contentType string) bool {
	// Rewrite JSON package metadata
	contentType = strings.ToLower(contentType)

	return strings.Contains(contentType, "application/json") ||
		strings.Contains(contentType, "application/vnd.npm.install-v1+json") ||
		strings.Contains(contentType, "text/plain") ||
		strings.Contains(contentType, "text/html")
}

// extractHostFromURL extracts host:port from a URL, stripping the scheme, path, and query
// This allows scheme-agnostic URL matching (http vs https)
// Examples:
//   - "http://host:8080/path?query" -> "host:8080"
//   - "https://host" -> "host"
func extractHostFromURL(url string) string {
	// Strip scheme
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	// Strip path and query if present
	if idx := strings.IndexAny(host, "/?#"); idx != -1 {
		host = host[:idx]
	}

	return host
}
