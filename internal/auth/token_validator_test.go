package auth

import (
	"strings"
	"testing"
)

func TestValidateTokenFormat(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		wantType  string
		wantError bool
	}{
		// Valid classic PAT tests
		{
			name:      "valid classic PAT with lowercase",
			token:     "ghp_" + strings.Repeat("a", 36),
			wantType:  TokenTypePAT,
			wantError: false,
		},
		{
			name:      "valid classic PAT with uppercase",
			token:     "ghp_" + strings.Repeat("A", 36),
			wantType:  TokenTypePAT,
			wantError: false,
		},
		{
			name:      "valid classic PAT with numbers",
			token:     "ghp_" + strings.Repeat("0", 36),
			wantType:  TokenTypePAT,
			wantError: false,
		},
		{
			name:      "valid classic PAT with mixed",
			token:     "ghp_1234567890abcdefghijABCDEFGHIJ123456",
			wantType:  TokenTypePAT,
			wantError: false,
		},

		// Valid fine-grained PAT tests
		{
			name:      "valid fine-grained PAT",
			token:     "github_pat_" + strings.Repeat("a", 22) + "_" + strings.Repeat("b", 59),
			wantType:  TokenTypePAT,
			wantError: false,
		},
		{
			name:      "valid fine-grained PAT with mixed case",
			token:     "github_pat_" + strings.Repeat("A", 22) + "_" + strings.Repeat("B", 59),
			wantType:  TokenTypePAT,
			wantError: false,
		},
		{
			name:      "valid fine-grained PAT with numbers",
			token:     "github_pat_1234567890123456789012_12345678901234567890123456789012345678901234567890123456789",
			wantType:  TokenTypePAT,
			wantError: false,
		},

		// Valid GitHub Actions token tests
		{
			name:      "valid GitHub Actions token with lowercase",
			token:     "ghs_" + strings.Repeat("a", 36),
			wantType:  TokenTypeGitHubActions,
			wantError: false,
		},
		{
			name:      "valid GitHub Actions token with uppercase",
			token:     "ghs_" + strings.Repeat("A", 36),
			wantType:  TokenTypeGitHubActions,
			wantError: false,
		},
		{
			name:      "valid GitHub Actions token with numbers",
			token:     "ghs_" + strings.Repeat("0", 36),
			wantType:  TokenTypeGitHubActions,
			wantError: false,
		},
		{
			name:      "valid GitHub Actions token with mixed",
			token:     "ghs_1234567890abcdefghijABCDEFGHIJ123456",
			wantType:  TokenTypeGitHubActions,
			wantError: false,
		},

		// Invalid token tests - empty
		{
			name:      "empty token",
			token:     "",
			wantType:  TokenTypeUnknown,
			wantError: true,
		},

		// Invalid token tests - wrong length for classic PAT
		{
			name:      "ghp_ with 35 chars (too short)",
			token:     "ghp_" + strings.Repeat("a", 35),
			wantType:  TokenTypeUnknown,
			wantError: true,
		},
		{
			name:      "ghp_ with 37 chars (too long)",
			token:     "ghp_" + strings.Repeat("a", 37),
			wantType:  TokenTypeUnknown,
			wantError: true,
		},
		{
			name:      "ghp_ with 0 chars after prefix",
			token:     "ghp_",
			wantType:  TokenTypeUnknown,
			wantError: true,
		},

		// Invalid token tests - wrong length for GitHub Actions
		{
			name:      "ghs_ with 35 chars (too short)",
			token:     "ghs_" + strings.Repeat("a", 35),
			wantType:  TokenTypeUnknown,
			wantError: true,
		},
		{
			name:      "ghs_ with 37 chars (too long)",
			token:     "ghs_" + strings.Repeat("a", 37),
			wantType:  TokenTypeUnknown,
			wantError: true,
		},

		// Invalid token tests - wrong length for fine-grained PAT
		{
			name:      "github_pat_ with wrong first part length",
			token:     "github_pat_" + strings.Repeat("a", 21) + "_" + strings.Repeat("b", 59),
			wantType:  TokenTypeUnknown,
			wantError: true,
		},
		{
			name:      "github_pat_ with wrong second part length",
			token:     "github_pat_" + strings.Repeat("a", 22) + "_" + strings.Repeat("b", 58),
			wantType:  TokenTypeUnknown,
			wantError: true,
		},
		{
			name:      "github_pat_ missing separator",
			token:     "github_pat_" + strings.Repeat("a", 81),
			wantType:  TokenTypeUnknown,
			wantError: true,
		},

		// Invalid token tests - special characters
		{
			name:      "ghp_ with special char",
			token:     "ghp_" + strings.Repeat("a", 35) + "!",
			wantType:  TokenTypeUnknown,
			wantError: true,
		},
		{
			name:      "ghs_ with hyphen",
			token:     "ghs_" + strings.Repeat("a", 35) + "-",
			wantType:  TokenTypeUnknown,
			wantError: true,
		},
		{
			name:      "ghp_ with space",
			token:     "ghp_" + strings.Repeat("a", 35) + " ",
			wantType:  TokenTypeUnknown,
			wantError: true,
		},

		// Invalid token tests - wrong prefix
		{
			name:      "wrong prefix gho_",
			token:     "gho_" + strings.Repeat("a", 36),
			wantType:  TokenTypeUnknown,
			wantError: true,
		},
		{
			name:      "wrong prefix ghu_",
			token:     "ghu_" + strings.Repeat("a", 36),
			wantType:  TokenTypeUnknown,
			wantError: true,
		},
		{
			name:      "wrong prefix ghr_",
			token:     "ghr_" + strings.Repeat("a", 36),
			wantType:  TokenTypeUnknown,
			wantError: true,
		},
		{
			name:      "no prefix",
			token:     strings.Repeat("a", 40),
			wantType:  TokenTypeUnknown,
			wantError: true,
		},

		// Invalid token tests - random strings
		{
			name:      "random string",
			token:     "not_a_github_token",
			wantType:  TokenTypeUnknown,
			wantError: true,
		},
		{
			name:      "jwt token (wrong format)",
			token:     "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			wantType:  TokenTypeUnknown,
			wantError: true,
		},
		{
			name:      "base64 string",
			token:     "dGhpcyBpcyBhIHRlc3Q=",
			wantType:  TokenTypeUnknown,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, err := ValidateTokenFormat(tt.token)

			// Check error expectation
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateTokenFormat() error = %v, wantError %v", err, tt.wantError)
				return
			}

			// Check token type
			if gotType != tt.wantType {
				t.Errorf("ValidateTokenFormat() = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}

func TestGetTokenType(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		wantType string
	}{
		{
			name:     "classic PAT",
			token:    "ghp_" + strings.Repeat("a", 36),
			wantType: TokenTypePAT,
		},
		{
			name:     "fine-grained PAT",
			token:    "github_pat_" + strings.Repeat("a", 22) + "_" + strings.Repeat("b", 59),
			wantType: TokenTypePAT,
		},
		{
			name:     "GitHub Actions token",
			token:    "ghs_" + strings.Repeat("a", 36),
			wantType: TokenTypeGitHubActions,
		},
		{
			name:     "invalid token",
			token:    "invalid",
			wantType: TokenTypeUnknown,
		},
		{
			name:     "empty token",
			token:    "",
			wantType: TokenTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType := GetTokenType(tt.token)
			if gotType != tt.wantType {
				t.Errorf("GetTokenType() = %v, want %v", gotType, tt.wantType)
			}
		})
	}
}

// Benchmark token format validation performance
func BenchmarkValidateTokenFormat(b *testing.B) {
	validPAT := "ghp_" + strings.Repeat("a", 36)
	validGHS := "ghs_" + strings.Repeat("a", 36)
	invalid := "invalid_token"

	b.Run("valid PAT", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = ValidateTokenFormat(validPAT)
		}
	})

	b.Run("valid GitHub Actions", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = ValidateTokenFormat(validGHS)
		}
	})

	b.Run("invalid token", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = ValidateTokenFormat(invalid)
		}
	})
}
