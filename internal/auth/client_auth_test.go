package auth

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

// TestExtractBearerToken tests the extractBearerToken helper function
func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name        string
		authHeader  string
		wantToken   string
		wantError   bool
		errorString string
	}{
		{
			name:       "valid bearer token",
			authHeader: "Bearer ghp_1234567890abcdefghijABCDEFGHIJ123456",
			wantToken:  "ghp_1234567890abcdefghijABCDEFGHIJ123456",
			wantError:  false,
		},
		{
			name:       "valid bearer token with extra whitespace",
			authHeader: "Bearer   ghp_1234567890abcdefghijABCDEFGHIJ123456  ",
			wantToken:  "ghp_1234567890abcdefghijABCDEFGHIJ123456",
			wantError:  false,
		},
		{
			name:        "empty bearer token",
			authHeader:  "Bearer ",
			wantToken:   "",
			wantError:   true,
			errorString: "empty bearer token",
		},
		{
			name:        "bearer with only whitespace",
			authHeader:  "Bearer    ",
			wantToken:   "",
			wantError:   true,
			errorString: "empty bearer token",
		},
		{
			name:       "bearer token with GitHub Actions token",
			authHeader: "Bearer ghs_1234567890abcdefghijABCDEFGHIJ123456",
			wantToken:  "ghs_1234567890abcdefghijABCDEFGHIJ123456",
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToken, err := extractBearerToken(tt.authHeader)

			if (err != nil) != tt.wantError {
				t.Errorf("extractBearerToken() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError && err != nil && !strings.Contains(err.Error(), tt.errorString) {
				t.Errorf("extractBearerToken() error = %v, want error containing %v", err, tt.errorString)
			}

			if gotToken != tt.wantToken {
				t.Errorf("extractBearerToken() = %v, want %v", gotToken, tt.wantToken)
			}
		})
	}
}

// TestExtractBasicAuthToken tests the extractBasicAuthToken helper function
func TestExtractBasicAuthToken(t *testing.T) {
	validPAT := "ghp_1234567890abcdefghijABCDEFGHIJ123456"
	validGHS := "ghs_1234567890abcdefghijABCDEFGHIJ123456"

	tests := []struct {
		name        string
		username    string
		password    string
		wantToken   string
		wantError   bool
		errorString string
	}{
		{
			name:      "token in password field (common pattern)",
			username:  "user",
			password:  validPAT,
			wantToken: validPAT,
			wantError: false,
		},
		{
			name:      "token in username field (alternative pattern)",
			username:  validPAT,
			password:  "somepassword",
			wantToken: validPAT,
			wantError: false,
		},
		{
			name:      "GitHub Actions token in password",
			username:  "user",
			password:  validGHS,
			wantToken: validGHS,
			wantError: false,
		},
		{
			name:        "no valid token in either field",
			username:    "user",
			password:    "password",
			wantToken:   "",
			wantError:   true,
			errorString: "no valid GitHub token",
		},
		{
			name:      "token in both fields (password takes precedence)",
			username:  "ghp_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			password:  validPAT,
			wantToken: validPAT,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create Basic auth header
			credentials := tt.username + ":" + tt.password
			encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
			authHeader := "Basic " + encoded

			gotToken, err := extractBasicAuthToken(authHeader)

			if (err != nil) != tt.wantError {
				t.Errorf("extractBasicAuthToken() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError && err != nil && !strings.Contains(err.Error(), tt.errorString) {
				t.Errorf("extractBasicAuthToken() error = %v, want error containing %v", err, tt.errorString)
			}

			if gotToken != tt.wantToken {
				t.Errorf("extractBasicAuthToken() = %v, want %v", gotToken, tt.wantToken)
			}
		})
	}
}

// TestExtractBasicAuthTokenInvalidFormat tests extractBasicAuthToken with invalid formats
func TestExtractBasicAuthTokenInvalidFormat(t *testing.T) {
	tests := []struct {
		name        string
		authHeader  string
		wantError   bool
		errorString string
	}{
		{
			name:        "invalid base64",
			authHeader:  "Basic not-valid-base64!!!",
			wantError:   true,
			errorString: "invalid basic auth",
		},
		{
			name:        "missing colon separator",
			authHeader:  "Basic " + base64.StdEncoding.EncodeToString([]byte("usernameonly")),
			wantError:   true,
			errorString: "invalid basic auth",
		},
		{
			name:        "empty credentials",
			authHeader:  "Basic " + base64.StdEncoding.EncodeToString([]byte("")),
			wantError:   true,
			errorString: "invalid basic auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := extractBasicAuthToken(tt.authHeader)

			if (err != nil) != tt.wantError {
				t.Errorf("extractBasicAuthToken() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError && err != nil && !strings.Contains(err.Error(), tt.errorString) {
				t.Errorf("extractBasicAuthToken() error = %v, want error containing %v", err, tt.errorString)
			}
		})
	}
}

// TestParseBasicAuth tests the ParseBasicAuth function
func TestParseBasicAuth(t *testing.T) {
	tests := []struct {
		name         string
		username     string
		password     string
		wantUsername string
		wantPassword string
		wantError    bool
	}{
		{
			name:         "valid basic auth",
			username:     "user",
			password:     "password",
			wantUsername: "user",
			wantPassword: "password",
			wantError:    false,
		},
		{
			name:         "username with special chars",
			username:     "user@example.com",
			password:     "password123",
			wantUsername: "user@example.com",
			wantPassword: "password123",
			wantError:    false,
		},
		{
			name:         "password with colon",
			username:     "user",
			password:     "pass:word:123",
			wantUsername: "user",
			wantPassword: "pass:word:123",
			wantError:    false,
		},
		{
			name:         "empty username",
			username:     "",
			password:     "password",
			wantUsername: "",
			wantPassword: "password",
			wantError:    false,
		},
		{
			name:         "empty password",
			username:     "user",
			password:     "",
			wantUsername: "user",
			wantPassword: "",
			wantError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create Basic auth header
			credentials := tt.username + ":" + tt.password
			encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
			authHeader := "Basic " + encoded

			gotUsername, gotPassword, err := ParseBasicAuth(authHeader)

			if (err != nil) != tt.wantError {
				t.Errorf("ParseBasicAuth() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if gotUsername != tt.wantUsername {
				t.Errorf("ParseBasicAuth() username = %v, want %v", gotUsername, tt.wantUsername)
			}

			if gotPassword != tt.wantPassword {
				t.Errorf("ParseBasicAuth() password = %v, want %v", gotPassword, tt.wantPassword)
			}
		})
	}
}

// TestAuthenticateAndInjectContext tests the AuthenticateAndInjectContext method
func TestAuthenticateAndInjectContext(t *testing.T) {
	// Note: We can't fully test actual GitHub API calls without mocking,
	// so we only test the error paths that don't require real API calls
	logger := zerolog.Nop() // No-op logger for tests

	// For these tests, we don't need a fully initialized client
	// since we're testing error conditions before GitHub API calls
	authenticator := &ClientAuthenticator{
		githubClient:  nil, // Will fail at GitHub API call, which is fine for these tests
		requiredOrg:   "test-org",
		requiredTeams: []string{"test-team"},
		logger:        logger,
	}

	tests := []struct {
		name          string
		setupRequest  func() *http.Request
		wantError     bool
		errorContains string
	}{
		{
			name: "missing authorization header",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				return req
			},
			wantError:     true,
			errorContains: "no authorization header",
		},
		{
			name: "invalid token format in bearer",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", "Bearer invalid_token_format")
				return req
			},
			wantError:     true,
			errorContains: "invalid token format",
		},
		{
			name: "unsupported auth scheme",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", "Digest username=test")
				return req
			},
			wantError:     true,
			errorContains: "unsupported auth scheme",
		},
		{
			name: "invalid basic auth format",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", "Basic not-base64!!!")
				return req
			},
			wantError:     true,
			errorContains: "invalid basic auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()

			_, newReq, err := authenticator.AuthenticateAndInjectContext(req)

			if (err != nil) != tt.wantError {
				t.Errorf("AuthenticateAndInjectContext() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError && err != nil && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("AuthenticateAndInjectContext() error = %v, want error containing %v", err, tt.errorContains)
			}

			// For error cases, request should be unchanged
			if tt.wantError {
				if newReq.Context() != req.Context() {
					t.Errorf("AuthenticateAndInjectContext() modified context on error")
				}
			}
		})
	}
}

// TestAuthenticateRequestAuthSchemes tests different authentication schemes
func TestAuthenticateRequestAuthSchemes(t *testing.T) {
	// Note: We can't fully test actual GitHub API calls without mocking,
	// so we only test the error paths that don't require real API calls
	logger := zerolog.Nop()

	// For these tests, we don't need a fully initialized client
	// since we're testing error conditions before GitHub API calls
	authenticator := &ClientAuthenticator{
		githubClient:  nil, // Will fail at GitHub API call, which is fine for these tests
		requiredOrg:   "test-org",
		requiredTeams: []string{},
		logger:        logger,
	}

	tests := []struct {
		name          string
		authHeader    string
		wantError     bool
		errorContains string
	}{
		{
			name:          "bearer with invalid format",
			authHeader:    "Bearer invalid_token",
			wantError:     true,
			errorContains: "invalid token format",
		},
		{
			name:          "empty authorization",
			authHeader:    "",
			wantError:     true,
			errorContains: "no authorization header",
		},
		{
			name:          "unsupported auth scheme",
			authHeader:    "Digest username=test",
			wantError:     true,
			errorContains: "unsupported auth scheme",
		},
		{
			name:          "basic auth with no valid token",
			authHeader:    "Basic " + base64.StdEncoding.EncodeToString([]byte("user:pass")),
			wantError:     true,
			errorContains: "no valid GitHub token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			_, err := authenticator.AuthenticateRequest(req)

			if (err != nil) != tt.wantError {
				t.Errorf("AuthenticateRequest() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.errorContains != "" && err != nil && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("AuthenticateRequest() error = %v, want error containing %v", err, tt.errorContains)
			}
		})
	}
}

// TestNewClientAuthenticator tests the constructor
func TestNewClientAuthenticator(t *testing.T) {
	mockGitHubClient := &GitHubClient{}
	logger := zerolog.Nop()
	requiredOrg := "test-org"
	requiredTeams := []string{"team1", "team2"}

	authenticator := NewClientAuthenticator(
		mockGitHubClient,
		requiredOrg,
		requiredTeams,
		logger,
	)

	if authenticator == nil {
		t.Fatal("NewClientAuthenticator() returned nil")
	}

	if authenticator.githubClient != mockGitHubClient {
		t.Error("NewClientAuthenticator() did not set githubClient correctly")
	}

	if authenticator.requiredOrg != requiredOrg {
		t.Errorf("NewClientAuthenticator() requiredOrg = %v, want %v", authenticator.requiredOrg, requiredOrg)
	}

	if len(authenticator.requiredTeams) != len(requiredTeams) {
		t.Errorf("NewClientAuthenticator() requiredTeams length = %v, want %v", len(authenticator.requiredTeams), len(requiredTeams))
	}
}

// BenchmarkExtractBearerToken benchmarks the bearer token extraction
func BenchmarkExtractBearerToken(b *testing.B) {
	authHeader := "Bearer ghp_1234567890abcdefghijABCDEFGHIJ123456"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = extractBearerToken(authHeader)
	}
}

// BenchmarkExtractBasicAuthToken benchmarks the basic auth token extraction
func BenchmarkExtractBasicAuthToken(b *testing.B) {
	validPAT := "ghp_1234567890abcdefghijABCDEFGHIJ123456"
	credentials := "user:" + validPAT
	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
	authHeader := "Basic " + encoded

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = extractBasicAuthToken(authHeader)
	}
}

// BenchmarkParseBasicAuth benchmarks the basic auth parsing
func BenchmarkParseBasicAuth(b *testing.B) {
	credentials := "username:password"
	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))
	authHeader := "Basic " + encoded

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = ParseBasicAuth(authHeader)
	}
}
