package detector

import (
	"net/http"
	"strings"
)

// OCIDetector detects OCI/Docker registry protocol requests
type OCIDetector struct {
	host string
}

// NewOCIDetector creates a new OCI detector
// host: optional domain for host-based routing (e.g., "docker.example.com")
func NewOCIDetector(host string) *OCIDetector {
	return &OCIDetector{host: host}
}

// Detect checks if the request is an OCI/Docker registry request
func (d *OCIDetector) Detect(r *http.Request) bool {
	// Check 0: Host matching (if configured)
	if d.host != "" {
		requestHost := getRequestHost(r)
		if requestHost != d.host {
			return false
		}
	}

	// Check 1: OCI spec - path must start with /v2 (includes /v2/ and /v2 exactly)
	if strings.HasPrefix(r.URL.Path, "/v2/") || r.URL.Path == "/v2" {
		return true
	}

	// Check 2: Docker-Distribution-Api-Version header
	if r.Header.Get("Docker-Distribution-Api-Version") != "" {
		return true
	}

	// Check 3: Accept header contains Docker manifest content types
	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "application/vnd.docker.distribution") ||
		strings.Contains(accept, "application/vnd.oci.image") {
		return true
	}

	// Check 4: Content-Type header contains Docker manifest content types
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/vnd.docker.distribution") ||
		strings.Contains(contentType, "application/vnd.oci.image") {
		return true
	}

	return false
}

// Protocol returns the protocol name
func (d *OCIDetector) Protocol() Protocol {
	return ProtocolOCI
}

// Priority returns the detection priority (higher = checked first)
func (d *OCIDetector) Priority() int {
	return 100 // High priority - check OCI first
}
