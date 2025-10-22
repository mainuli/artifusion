package oci

import (
	"testing"

	"github.com/mainuli/artifusion/internal/config"
	"github.com/rs/zerolog"
)

// TestRewritePath_NoRewrite tests that paths are not rewritten when no upstream namespace is configured
func TestRewritePath_NoRewrite(t *testing.T) {
	h := &Handler{
		logger: zerolog.Nop(),
	}

	backend := &config.OCIBackendConfig{
		Name:              "direct-registry",
		UpstreamNamespace: "", // No upstream namespace
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "api version check",
			input:    "/v2/",
			expected: "/v2/",
		},
		{
			name:     "manifest path",
			input:    "/v2/myimage/manifests/latest",
			expected: "/v2/myimage/manifests/latest",
		},
		{
			name:     "blob path",
			input:    "/v2/myorg/image/blobs/sha256:abc123",
			expected: "/v2/myorg/image/blobs/sha256:abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.rewritePath(tt.input, backend)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestRewritePath_WithUpstreamNamespace tests path rewriting with upstream namespace
func TestRewritePath_WithUpstreamNamespace(t *testing.T) {
	h := &Handler{
		logger: zerolog.Nop(),
	}

	backend := &config.OCIBackendConfig{
		Name:              "ghcr",
		UpstreamNamespace: "ghcr.io",
		PathRewrite: config.PathRewriteConfig{
			AddLibraryPrefix: false,
		},
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "api version check unchanged",
			input:    "/v2/",
			expected: "/v2/",
		},
		{
			name:     "simple manifest",
			input:    "/v2/myorg/image/manifests/latest",
			expected: "/v2/ghcr.io/myorg/image/manifests/latest",
		},
		{
			name:     "blob download",
			input:    "/v2/myorg/image/blobs/sha256:abc123",
			expected: "/v2/ghcr.io/myorg/image/blobs/sha256:abc123",
		},
		{
			name:     "tags list",
			input:    "/v2/myorg/image/tags/list",
			expected: "/v2/ghcr.io/myorg/image/tags/list",
		},
		{
			name:     "nested image path",
			input:    "/v2/myorg/subproject/image/manifests/v1.0.0",
			expected: "/v2/ghcr.io/myorg/subproject/image/manifests/v1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.rewritePath(tt.input, backend)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestRewritePath_AddLibraryPrefix tests adding library/ prefix for official images
func TestRewritePath_AddLibraryPrefix(t *testing.T) {
	h := &Handler{
		logger: zerolog.Nop(),
	}

	backend := &config.OCIBackendConfig{
		Name:              "dockerhub",
		UpstreamNamespace: "docker.io",
		PathRewrite: config.PathRewriteConfig{
			AddLibraryPrefix: true,
		},
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "official image gets library prefix",
			input:    "/v2/nginx/manifests/latest",
			expected: "/v2/docker.io/library/nginx/manifests/latest",
		},
		{
			name:     "official image with tag",
			input:    "/v2/alpine/manifests/3.14",
			expected: "/v2/docker.io/library/alpine/manifests/3.14",
		},
		{
			name:     "user image no library prefix",
			input:    "/v2/myuser/image/manifests/latest",
			expected: "/v2/docker.io/myuser/image/manifests/latest",
		},
		{
			name:     "org image no library prefix",
			input:    "/v2/myorg/project/image/manifests/v1.0",
			expected: "/v2/docker.io/myorg/project/image/manifests/v1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.rewritePath(tt.input, backend)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestRewritePath_InvalidPaths tests handling of invalid paths
func TestRewritePath_InvalidPaths(t *testing.T) {
	h := &Handler{
		logger: zerolog.Nop(),
	}

	backend := &config.OCIBackendConfig{
		Name:              "ghcr",
		UpstreamNamespace: "ghcr.io",
		PathRewrite: config.PathRewriteConfig{
			AddLibraryPrefix: false,
		},
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty path after v2",
			input:    "/v2",
			expected: "/v2", // Invalid path, return as-is
		},
		{
			name:     "just v2 slash",
			input:    "/v2/",
			expected: "/v2/", // API version check, return as-is
		},
		{
			name:     "single component no operation",
			input:    "/v2/image",
			expected: "/v2/image", // No operation keyword, return as-is (invalid path)
		},
		{
			name:     "missing v2 prefix",
			input:    "/myimage/manifests/latest",
			expected: "/myimage/manifests/latest", // Missing /v2/ prefix, return as-is
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.rewritePath(tt.input, backend)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestRewritePath_RealWorldScenarios tests production-like scenarios
func TestRewritePath_RealWorldScenarios(t *testing.T) {
	t.Run("Direct Registry No Rewrite", func(t *testing.T) {
		h := &Handler{logger: zerolog.Nop()}
		backend := &config.OCIBackendConfig{
			Name:              "local-registry",
			UpstreamNamespace: "",
			PathRewrite: config.PathRewriteConfig{
				AddLibraryPrefix: false,
			},
		}

		// Direct push to local registry:2, no rewriting
		input := "/v2/myimage/manifests/latest"
		expected := "/v2/myimage/manifests/latest"
		result := h.rewritePath(input, backend)

		if result != expected {
			t.Errorf("Direct registry: expected %s, got %s", expected, result)
		}
	})
}
