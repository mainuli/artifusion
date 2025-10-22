package oci

import (
	"strings"

	"github.com/mainuli/artifusion/internal/config"
)

// rewritePath rewrites the request path for oci-registry namespace routing
// Example: /v2/myorg.io/myorg/image/manifests/latest → /v2/ghcr.io/myorg/image/manifests/latest
func (h *Handler) rewritePath(originalPath string, backend *config.OCIBackendConfig) string {
	// For write operations directly to registry:2, no rewriting needed
	if backend.UpstreamNamespace == "" {
		return originalPath
	}

	// Extract components from path
	// Pattern: /v2/<registry>/<image-path>/<operation>/<reference>
	// Example: /v2/myorg.io/myorg/image/manifests/latest

	// Validate path starts with /v2/
	if !strings.HasPrefix(originalPath, "/v2/") {
		// Invalid path, return as-is
		return originalPath
	}

	// Strip /v2/ prefix
	path := strings.TrimPrefix(originalPath, "/v2/")
	if path == "" || path == "/" {
		// API version check endpoint
		return originalPath
	}

	// Split path into parts
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		// Invalid path, return as-is
		return originalPath
	}

	// Find the operation keyword (manifests, blobs, tags, _catalog)
	var imageParts []string
	var suffixStart int

	for i, part := range parts {
		if part == "manifests" || part == "blobs" || part == "tags" || part == "_catalog" {
			suffixStart = i
			break
		}
		imageParts = append(imageParts, part)
	}

	// Build suffix (everything from operation onwards)
	var suffix string
	if suffixStart > 0 && suffixStart < len(parts) {
		suffix = "/" + strings.Join(parts[suffixStart:], "/")
	}

	// Get image name
	imageName := strings.Join(imageParts, "/")
	if imageName == "" {
		return originalPath
	}

	// Apply path rewrite rule for Docker Hub official images
	if backend.PathRewrite.AddLibraryPrefix && !strings.Contains(imageName, "/") {
		// Add library/ prefix for official images (nginx → library/nginx)
		imageName = "library/" + imageName
	}

	// Inject upstream namespace for oci-registry routing
	if backend.UpstreamNamespace != "" {
		imageName = backend.UpstreamNamespace + "/" + imageName
	}

	// Rebuild path
	rewritten := "/v2/" + imageName + suffix

	h.logger.Debug().
		Str("original", originalPath).
		Str("rewritten", rewritten).
		Str("backend", backend.Name).
		Str("namespace", backend.UpstreamNamespace).
		Msg("Path rewritten")

	return rewritten
}
