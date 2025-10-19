package detector

import (
	"net/http"
	"strings"
)

// OCIDetector detects OCI/Docker registry protocol requests
type OCIDetector struct{}

// NewOCIDetector creates a new OCI detector
func NewOCIDetector() *OCIDetector {
	return &OCIDetector{}
}

// Detect checks if the request is an OCI/Docker registry request
func (d *OCIDetector) Detect(r *http.Request) bool {
	// Check 1: Path starts with /v2/
	if strings.HasPrefix(r.URL.Path, "/v2/") {
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
