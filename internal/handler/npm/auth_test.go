package npm

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mainuli/artifusion/internal/config"
)

func TestInjectBackendAuth_StripsClientAuthWhenNoBackendAuth(t *testing.T) {
	tests := []struct {
		name            string
		clientAuthType  string
		clientAuthValue string
	}{
		{
			name:            "strips Bearer token (GitHub PAT)",
			clientAuthType:  "Bearer",
			clientAuthValue: "Bearer ghp_1234567890abcdefghijklmnopqrstuvwxyz",
		},
		{
			name:            "strips NPM token",
			clientAuthType:  "Bearer",
			clientAuthValue: "Bearer npm_abc123defXYZ789",
		},
		{
			name:            "strips fine-grained PAT",
			clientAuthType:  "Bearer",
			clientAuthValue: "Bearer github_pat_11AAAAAA_aaaaaaaaaaaaaaaaa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/npm/@scope/package", nil)
			req.Header.Set("Authorization", tt.clientAuthValue)

			backend := &config.NPMBackendConfig{
				Auth: nil, // No backend auth configured
			}

			handler := &Handler{}
			handler.injectBackendAuth(req, backend)

			// SECURITY: Client's Authorization header must be stripped
			authHeader := req.Header.Get("Authorization")
			if authHeader != "" {
				t.Errorf("SECURITY FAILURE: Client Authorization header was not stripped! Got: %s", authHeader)
			}
		})
	}
}

func TestInjectBackendAuth_ReplacesWithBackendBasicAuth(t *testing.T) {
	req := httptest.NewRequest("GET", "/npm/@scope/package", nil)
	req.Header.Set("Authorization", "Bearer ghp_client_github_pat_token")

	backend := &config.NPMBackendConfig{
		Auth: &config.AuthConfig{
			Type:     "basic",
			Username: "backend-user",
			Password: "backend-pass",
		},
	}

	handler := &Handler{}
	handler.injectBackendAuth(req, backend)

	// Client auth should be replaced with backend auth
	authHeader := req.Header.Get("Authorization")

	// Should not contain client token
	if strings.Contains(authHeader, "ghp_client") {
		t.Error("Client GitHub PAT was not replaced!")
	}

	// Should have Basic auth
	if !strings.HasPrefix(authHeader, "Basic ") {
		t.Errorf("Backend Basic auth was not injected! Got: %s", authHeader)
	}
}

func TestInjectBackendAuth_ReplacesWithBackendBearerToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/npm/@scope/package", nil)
	req.Header.Set("Authorization", "Bearer ghp_client_token")

	backend := &config.NPMBackendConfig{
		Auth: &config.AuthConfig{
			Type:  "bearer",
			Token: "backend-npm-token-12345",
		},
	}

	handler := &Handler{}
	handler.injectBackendAuth(req, backend)

	// Client auth should be replaced with backend auth
	authHeader := req.Header.Get("Authorization")

	// Should not contain client token
	if strings.Contains(authHeader, "ghp_client") {
		t.Error("Client GitHub PAT was not replaced!")
	}

	// Should have backend bearer token
	expectedAuth := "Bearer backend-npm-token-12345"
	if authHeader != expectedAuth {
		t.Errorf("Backend Bearer token was not injected! Expected: %s, Got: %s", expectedAuth, authHeader)
	}
}

func TestInjectBackendAuth_HandlesCustomHeaderAuth(t *testing.T) {
	req := httptest.NewRequest("GET", "/npm/@scope/package", nil)
	req.Header.Set("Authorization", "Bearer ghp_client_token")

	backend := &config.NPMBackendConfig{
		Auth: &config.AuthConfig{
			Type:        "custom",
			HeaderName:  "X-NPM-Auth",
			HeaderValue: "custom-npm-auth-value",
		},
	}

	handler := &Handler{}
	handler.injectBackendAuth(req, backend)

	// Client Authorization header should be removed
	if req.Header.Get("Authorization") != "" {
		t.Error("Client Authorization header was not removed!")
	}

	// Custom header should be set
	customHeader := req.Header.Get("X-NPM-Auth")
	if customHeader != "custom-npm-auth-value" {
		t.Errorf("Custom auth header was not set! Expected: custom-npm-auth-value, Got: %s", customHeader)
	}
}

func TestInjectBackendAuth_SecurityVerification(t *testing.T) {
	// This test explicitly verifies that GitHub PATs are NEVER leaked to NPM registries
	githubPATs := []string{
		"ghp_1234567890abcdefghijABCDEFGHIJ123456",
		"github_pat_11AAAAAA_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"ghs_1234567890abcdefghijABCDEFGHIJ123456",
	}

	for _, pat := range githubPATs {
		t.Run("never_leaks_"+pat[:10], func(t *testing.T) {
			req := httptest.NewRequest("GET", "/npm/@scope/package", nil)
			req.Header.Set("Authorization", "Bearer "+pat)

			backend := &config.NPMBackendConfig{
				Auth: nil, // No backend auth - public registry
			}

			handler := &Handler{}
			handler.injectBackendAuth(req, backend)

			// Verify PAT is completely removed
			authHeader := req.Header.Get("Authorization")
			if authHeader != "" {
				t.Errorf("CRITICAL SECURITY FAILURE: GitHub PAT leaked to NPM registry! Header: %s", authHeader)
			}
		})
	}
}

func TestInjectBackendAuth_PreservesOtherHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/npm/@scope/package", nil)
	req.Header.Set("Authorization", "Bearer ghp_client_token")
	req.Header.Set("User-Agent", "npm/8.19.2 node/v18.12.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("X-NPM-Version", "8.19.2")

	backend := &config.NPMBackendConfig{
		Auth: nil,
	}

	handler := &Handler{}
	handler.injectBackendAuth(req, backend)

	// Authorization should be removed
	if req.Header.Get("Authorization") != "" {
		t.Error("Authorization header was not removed")
	}

	// Other headers should be preserved
	if req.Header.Get("User-Agent") != "npm/8.19.2 node/v18.12.0" {
		t.Error("User-Agent header was modified")
	}
	if req.Header.Get("Accept") != "application/json" {
		t.Error("Accept header was modified")
	}
	if req.Header.Get("Accept-Encoding") != "gzip, deflate" {
		t.Error("Accept-Encoding header was modified")
	}
	if req.Header.Get("X-NPM-Version") != "8.19.2" {
		t.Error("X-NPM-Version header was modified")
	}
}

func TestInjectBackendAuth_NPMPublicRegistry(t *testing.T) {
	// Common scenario: NPM public registry with no backend auth
	req := httptest.NewRequest("GET", "/npm/express", nil)
	req.Header.Set("Authorization", "Bearer ghp_very_sensitive_github_pat")

	backend := &config.NPMBackendConfig{
		Name: "npmjs-public",
		URL:  "https://registry.npmjs.org",
		Auth: nil, // Public registry - no auth
	}

	handler := &Handler{}
	handler.injectBackendAuth(req, backend)

	// CRITICAL: GitHub PAT must not leak to public NPM registry
	authHeader := req.Header.Get("Authorization")
	if authHeader != "" {
		t.Errorf("SECURITY FAILURE: GitHub PAT would be leaked to public NPM registry! Header: %s", authHeader)
	}
}
