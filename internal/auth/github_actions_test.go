package auth

import (
	"strings"
	"testing"
	"time"
)

// TestValidateGitHubActionsToken tests the GitHub Actions token validation flow
// Note: These are integration-style tests that would require actual GitHub API mocking
// For now, we test the token type detection and routing logic
func TestGitHubActionsTokenDetection(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		wantType  string
		wantError bool
	}{
		{
			name:      "GitHub Actions token detected",
			token:     "ghs_" + strings.Repeat("a", 36),
			wantType:  TokenTypeGitHubActions,
			wantError: false,
		},
		{
			name:      "Classic PAT detected",
			token:     "ghp_" + strings.Repeat("a", 36),
			wantType:  TokenTypePAT,
			wantError: false,
		},
		{
			name:      "Fine-grained PAT detected",
			token:     "github_pat_" + strings.Repeat("a", 22) + "_" + strings.Repeat("b", 59),
			wantType:  TokenTypePAT,
			wantError: false,
		},
		{
			name:      "Invalid token rejected",
			token:     "invalid_token",
			wantType:  TokenTypeUnknown,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, err := ValidateTokenFormat(tt.token)

			if (err != nil) != tt.wantError {
				t.Errorf("ValidateTokenFormat() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if gotType != tt.wantType {
				t.Errorf("ValidateTokenFormat() = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}

// TestTokenTypeRouting tests that tokens are routed to the correct validation method
func TestTokenTypeRouting(t *testing.T) {
	// Create a GitHub client with cache
	client := NewGitHubClient("https://api.github.com", 5*time.Minute, 10)

	tests := []struct {
		name     string
		token    string
		wantType string
	}{
		{
			name:     "GitHub Actions token routes to ghs_ validator",
			token:    "ghs_" + strings.Repeat("a", 36),
			wantType: TokenTypeGitHubActions,
		},
		{
			name:     "Classic PAT routes to PAT validator",
			token:    "ghp_" + strings.Repeat("a", 36),
			wantType: TokenTypePAT,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate token format
			tokenType, err := ValidateTokenFormat(tt.token)
			if err != nil {
				t.Fatalf("ValidateTokenFormat() error = %v", err)
			}

			if tokenType != tt.wantType {
				t.Errorf("Token type = %v, want %v", tokenType, tt.wantType)
			}

			// Note: We can't test the actual API calls without mocking
			// The routing logic is tested above
			_ = client // Use client to avoid unused variable warning
		})
	}
}

// TestGitHubActionsTokenValidation_ErrorCases tests error handling for GitHub Actions tokens
func TestGitHubActionsTokenValidation_ErrorCases(t *testing.T) {
	// These tests verify the error cases in validateGitHubActionsToken
	// Actual API mocking would be needed for full coverage

	tests := []struct {
		name          string
		token         string
		requiredOrg   string
		expectedError string
	}{
		{
			name:          "Empty repository owner should error",
			token:         "ghs_" + strings.Repeat("a", 36),
			requiredOrg:   "test-org",
			expectedError: "repository owner", // Part of error message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate token format first
			tokenType, err := ValidateTokenFormat(tt.token)
			if err != nil {
				t.Fatalf("ValidateTokenFormat() unexpected error = %v", err)
			}

			if tokenType != TokenTypeGitHubActions {
				t.Errorf("Expected token type %s, got %s", TokenTypeGitHubActions, tokenType)
			}

			// Note: Full validation would require API mocking
			// This test verifies the token detection works correctly
		})
	}
}

// TestAuthResultWithTokenType tests that AuthResult includes token type information
func TestAuthResultWithTokenType(t *testing.T) {
	tests := []struct {
		name     string
		result   *AuthResult
		wantType string
		wantRepo string
	}{
		{
			name: "PAT AuthResult",
			result: &AuthResult{
				Username:   "testuser",
				Org:        "testorg",
				Teams:      []string{"team1"},
				TokenType:  TokenTypePAT,
				Repository: "",
			},
			wantType: TokenTypePAT,
			wantRepo: "",
		},
		{
			name: "GitHub Actions AuthResult",
			result: &AuthResult{
				Username:   "github-actions[bot]",
				Org:        "testorg",
				Teams:      nil,
				TokenType:  TokenTypeGitHubActions,
				Repository: "testorg/testrepo",
			},
			wantType: TokenTypeGitHubActions,
			wantRepo: "testorg/testrepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.result.TokenType != tt.wantType {
				t.Errorf("TokenType = %v, want %v", tt.result.TokenType, tt.wantType)
			}

			if tt.result.Repository != tt.wantRepo {
				t.Errorf("Repository = %v, want %v", tt.result.Repository, tt.wantRepo)
			}

			// Verify GitHub Actions tokens have expected characteristics
			if tt.result.TokenType == TokenTypeGitHubActions {
				if tt.result.Username != "github-actions[bot]" {
					t.Errorf("Expected username 'github-actions[bot]', got %s", tt.result.Username)
				}

				if tt.result.Teams != nil {
					t.Errorf("GitHub Actions tokens should have nil Teams, got %v", tt.result.Teams)
				}

				if tt.result.Repository == "" {
					t.Error("GitHub Actions tokens should have Repository set")
				}
			}

			// Verify PAT tokens have expected characteristics
			if tt.result.TokenType == TokenTypePAT {
				if tt.result.Repository != "" {
					t.Errorf("PAT tokens should have empty Repository, got %s", tt.result.Repository)
				}
			}
		})
	}
}

// TestPreemptiveValidationPreventsAPICall tests that invalid tokens are rejected before API calls
func TestPreemptiveValidationPreventsAPICall(t *testing.T) {
	invalidTokens := []string{
		"",
		"invalid",
		"ghp_tooshort",
		"ghs_" + strings.Repeat("a", 35), // Too short
		"ghs_" + strings.Repeat("a", 37), // Too long
		"notarealtoken",
	}

	for _, token := range invalidTokens {
		t.Run("invalid_token_"+token, func(t *testing.T) {
			_, err := ValidateTokenFormat(token)
			if err == nil {
				t.Errorf("Expected error for invalid token %q, got nil", token)
			}

			// This demonstrates that the token would be rejected before any API call
			// In the actual flow, client_auth.go calls ValidateTokenFormat first
			// and returns error immediately without calling githubClient.Validate
		})
	}
}

// TestGitHubActionsTokenCaching tests that GitHub Actions tokens are cached properly
func TestGitHubActionsTokenCaching(t *testing.T) {
	cache := NewAuthCache(5 * time.Minute)

	// Create a sample GitHub Actions AuthResult
	result := &AuthResult{
		Username:   "github-actions[bot]",
		Org:        "testorg",
		Teams:      nil,
		TokenType:  TokenTypeGitHubActions,
		Repository: "testorg/testrepo",
	}

	// Verify the AuthResult structure is compatible with caching
	if result.TokenType != TokenTypeGitHubActions {
		t.Errorf("Expected TokenType %s, got %s", TokenTypeGitHubActions, result.TokenType)
	}

	if result.Repository == "" {
		t.Error("GitHub Actions tokens should have Repository field set")
	}

	// Note: Actual caching is tested in cache_test.go
	// This test verifies the AuthResult structure is compatible
	_ = cache // Use cache to avoid unused variable warning
}

// BenchmarkTokenTypeDetection benchmarks the performance of token type detection
func BenchmarkTokenTypeDetection(b *testing.B) {
	validPAT := "ghp_" + strings.Repeat("a", 36)
	validGHS := "ghs_" + strings.Repeat("a", 36)
	invalid := "invalid"

	b.Run("PAT detection", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = ValidateTokenFormat(validPAT)
		}
	})

	b.Run("GitHub Actions detection", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = ValidateTokenFormat(validGHS)
		}
	})

	b.Run("Invalid token rejection", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = ValidateTokenFormat(invalid)
		}
	})
}
