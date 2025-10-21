package detector

import (
	"net/http"
	"strings"
)

// npmEndpoints contains NPM-specific endpoint patterns
// Declared at package level to avoid repeated allocations
var npmEndpoints = []string{
	"/-/ping",             // NPM registry health check
	"/-/whoami",           // NPM user info
	"/-/v1/search",        // NPM package search
	"/-/user/",            // User authentication
	"/-/npm/v1/",          // NPM API v1
	"/-/package/",         // Package metadata
	"/-/all",              // All packages listing
	"/-/v1/login",         // NPM login endpoint
	"/-/npm/v1/security/", // Security advisories
}

// NPMDetector detects NPM registry protocol requests
type NPMDetector struct {
	host       string
	pathPrefix string
}

// NewNPMDetector creates a new NPM detector
// host: optional domain for host-based routing (e.g., "npm.example.com")
// pathPrefix: path prefix for path-based routing - required when host is empty
func NewNPMDetector(host, pathPrefix string) *NPMDetector {
	// Normalize pathPrefix: ensure starts with /, no trailing /
	// SECURITY: No silent defaults - pathPrefix must be explicit from config
	if pathPrefix != "" {
		if !strings.HasPrefix(pathPrefix, "/") {
			pathPrefix = "/" + pathPrefix
		}
		pathPrefix = strings.TrimSuffix(pathPrefix, "/")
	}

	return &NPMDetector{
		host:       host,
		pathPrefix: pathPrefix,
	}
}

// Detect checks if the request is an NPM registry request
func (d *NPMDetector) Detect(r *http.Request) bool {
	// Check 0: Host matching (if configured)
	if d.host != "" {
		requestHost := getRequestHost(r)
		if requestHost != d.host {
			return false
		}
	}

	path := r.URL.Path

	// Check 1: Path prefix matching (if configured)
	if d.pathPrefix != "" {
		if !strings.HasPrefix(path, d.pathPrefix+"/") && path != d.pathPrefix {
			// Path doesn't match prefix
			return false
		}
		// Path matches prefix - route to this protocol handler
		// The handler will validate the specific request and handle auth
		return true
	}

	// No pathPrefix configured - use protocol-specific detection
	// This handles host-only routing mode

	// Check 2: NPM-specific endpoints
	for _, endpoint := range npmEndpoints {
		if strings.Contains(path, endpoint) {
			return true
		}
	}

	// Check 3: Scoped package pattern (@scope/package)
	if strings.Contains(path, "/@") {
		return true
	}

	// Check 4: NPM-specific Accept headers
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "application/vnd.npm.install-v1+json") {
		// NPM-specific Accept header - strong indicator
		return true
	}

	// Check 5: User-Agent header (NPM package managers) with path validation
	userAgent := r.Header.Get("User-Agent")
	if strings.Contains(userAgent, "npm/") ||
		strings.Contains(userAgent, "yarn/") ||
		strings.Contains(userAgent, "pnpm/") {
		// Also check if the path looks like an NPM package request
		// NPM package paths: /package-name or /@scope/package-name or /package/-/tarball.tgz

		// Strip path prefix before analyzing package name
		packagePath := path
		if d.pathPrefix != "" {
			packagePath = strings.TrimPrefix(path, d.pathPrefix)
		}

		parts := strings.Split(strings.Trim(packagePath, "/"), "/")
		if len(parts) >= 1 {
			// Check for package name pattern
			firstPart := parts[0]
			if strings.HasPrefix(firstPart, "@") { // Scoped package
				return true
			}
			// Unscoped package (no file extension and reasonable length)
			if len(parts) == 1 && !strings.Contains(firstPart, ".") && len(firstPart) > 0 && len(firstPart) < 214 {
				return true
			}
			// NPM tarball path: /package/-/package-version.tgz or /@scope/package/-/package-version.tgz
			if len(parts) >= 3 && parts[len(parts)-2] == "-" && strings.HasSuffix(packagePath, ".tgz") {
				return true
			}
		}
	}

	// Check 6: Content-Type header for package publish requests with User-Agent
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") && r.Method == http.MethodPut {
		// Publishing packages uses PUT with JSON content
		// But only if User-Agent suggests NPM client to reduce false positives
		if strings.Contains(userAgent, "npm/") ||
			strings.Contains(userAgent, "yarn/") ||
			strings.Contains(userAgent, "pnpm/") ||
			strings.Contains(userAgent, "node/") {
			return true
		}
	}

	// Check 7: NPM-specific query parameters
	query := r.URL.Query()
	if query.Get("write") == "true" || // NPM publish operation
		query.Has("dist-tags") { // NPM dist-tags operation
		return true
	}

	return false
}

// Protocol returns the protocol name
func (d *NPMDetector) Protocol() Protocol {
	return ProtocolNPM
}

// Priority returns the detection priority (between Maven and OCI)
func (d *NPMDetector) Priority() int {
	return 85 // Between Maven (90) and potential future protocols
}
