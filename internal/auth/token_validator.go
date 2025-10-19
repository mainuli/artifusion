package auth

import (
	"fmt"
	"regexp"
)

// Token type constants
const (
	TokenTypePAT           = "pat"
	TokenTypeGitHubActions = "github_actions"
	TokenTypeUnknown       = "unknown"
)

// Token length constants based on GitHub's official token format specifications
// Reference: https://github.blog/engineering/platform-security/behind-githubs-new-authentication-token-formats/
const (
	// ClassicPATLength is the total length of classic PAT tokens (ghp_ + 36 chars)
	ClassicPATLength = 40

	// FineGrainedPATLength is the total length of fine-grained PAT tokens
	// (github_pat_ + 22 chars + _ + 59 chars)
	FineGrainedPATLength = 93

	// GitHubActionsTokenLength is the total length of GitHub Actions tokens (ghs_ + 36 chars)
	GitHubActionsTokenLength = 40
)

// Token format regex patterns based on GitHub's official token format specifications
// Reference: https://github.blog/engineering/platform-security/behind-githubs-new-authentication-token-formats/
var (
	// Classic PAT: ghp_ + 36 alphanumeric characters
	// Format: ghp_[a-zA-Z0-9]{36}
	// Total length: 40 characters
	regexClassicPAT = regexp.MustCompile(`^ghp_[a-zA-Z0-9]{36}$`)

	// Fine-grained PAT: github_pat_ + 22 chars + _ + 59 chars
	// Format: github_pat_[a-zA-Z0-9]{22}_[a-zA-Z0-9]{59}
	// Total length: 93 characters
	regexFineGrainedPAT = regexp.MustCompile(`^github_pat_[a-zA-Z0-9]{22}_[a-zA-Z0-9]{59}$`)

	// GitHub Actions / Installation token: ghs_ + 36 alphanumeric characters
	// Format: ghs_[a-zA-Z0-9]{36}
	// Total length: 40 characters
	regexGitHubActions = regexp.MustCompile(`^ghs_[a-zA-Z0-9]{36}$`)
)

// ValidateTokenFormat validates GitHub token format without making API calls.
// This preemptive validation prevents GitHub API abuse and rate limit exhaustion
// from attackers sending random/invalid tokens.
//
// Performance optimization: Length checks are performed before regex matching,
// providing a fast-path rejection for tokens with incorrect lengths.
//
// Returns:
//   - tokenType: One of TokenTypePAT, TokenTypeGitHubActions, or TokenTypeUnknown
//   - error: Non-nil if the token format is invalid
//
// Supported formats:
//   - Classic PAT: ghp_[a-zA-Z0-9]{36} (40 chars total)
//   - Fine-grained PAT: github_pat_[a-zA-Z0-9]{22}_[a-zA-Z0-9]{59} (93 chars total)
//   - GitHub Actions: ghs_[a-zA-Z0-9]{36} (40 chars total)
//
// Examples:
//   - Classic PAT: ghp_1234567890abcdefghijABCDEFGHIJ123456
//   - Fine-grained PAT: github_pat_1234567890123456789012_12345678901234567890123456789012345678901234567890123456789
//   - GitHub Actions: ghs_1234567890abcdefghijABCDEFGHIJ123456
func ValidateTokenFormat(token string) (string, error) {
	if token == "" {
		return TokenTypeUnknown, fmt.Errorf("empty token")
	}

	tokenLen := len(token)

	// PERFORMANCE OPTIMIZATION: Fast-path length checks before expensive regex matching
	// This provides significant performance improvement for invalid tokens

	// Check classic PAT format (length check before regex)
	if tokenLen == ClassicPATLength {
		if regexClassicPAT.MatchString(token) {
			return TokenTypePAT, nil
		}
	}

	// Check fine-grained PAT format (length check before regex)
	if tokenLen == FineGrainedPATLength {
		if regexFineGrainedPAT.MatchString(token) {
			return TokenTypePAT, nil
		}
	}

	// Check GitHub Actions token format (length check before regex)
	if tokenLen == GitHubActionsTokenLength {
		if regexGitHubActions.MatchString(token) {
			return TokenTypeGitHubActions, nil
		}
	}

	// Token doesn't match any known GitHub token format
	return TokenTypeUnknown, fmt.Errorf("invalid GitHub token format")
}

// GetTokenType returns the token type without error information.
// This is a convenience wrapper around ValidateTokenFormat for cases
// where only the type is needed and error handling is done elsewhere.
func GetTokenType(token string) string {
	tokenType, _ := ValidateTokenFormat(token)
	return tokenType
}
