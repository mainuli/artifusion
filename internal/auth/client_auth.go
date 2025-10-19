package auth

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/mainuli/artifusion/internal/config"
	"github.com/mainuli/artifusion/internal/middleware"
	"github.com/rs/zerolog"
)

// ClientAuthenticator handles client authentication for all protocols
type ClientAuthenticator struct {
	githubClient  *GitHubClient
	requiredOrg   string
	requiredTeams []string
	logger        zerolog.Logger
}

// NewClientAuthenticator creates a new client authenticator
func NewClientAuthenticator(
	githubClient *GitHubClient,
	requiredOrg string,
	requiredTeams []string,
	logger zerolog.Logger,
) *ClientAuthenticator {
	return &ClientAuthenticator{
		githubClient:  githubClient,
		requiredOrg:   requiredOrg,
		requiredTeams: requiredTeams,
		logger:        logger,
	}
}

// AuthenticateRequest extracts credentials from request and validates with GitHub.
// It supports both Bearer and Basic authentication schemes.
//
// Supported authentication schemes:
//   - Bearer: Authorization: Bearer <github-token>
//   - Basic: Authorization: Basic <base64(username:password)>
//
// For Basic auth, the GitHub token can be in either username or password field.
// This is common with Docker and Maven clients that send: username=<anything>, password=<github-token>
func (a *ClientAuthenticator) AuthenticateRequest(r *http.Request) (*AuthResult, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("no authorization header")
	}

	// Extract token based on authentication scheme
	var githubToken string
	var err error

	switch {
	case strings.HasPrefix(authHeader, "Bearer "):
		githubToken, err = extractBearerToken(authHeader)
		if err != nil {
			return nil, err
		}

	case strings.HasPrefix(authHeader, "Basic "):
		githubToken, err = extractBasicAuthToken(authHeader)
		if err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("unsupported auth scheme")
	}

	// PREEMPTIVE VALIDATION: Check token format BEFORE making GitHub API call
	// This prevents API abuse and rate limit exhaustion from invalid tokens
	tokenType, err := ValidateTokenFormat(githubToken)
	if err != nil {
		a.logger.Warn().
			Str("error", err.Error()).
			Int("token_length", len(githubToken)).
			Msg("Invalid token format rejected")
		return nil, fmt.Errorf("invalid token format")
	}

	a.logger.Debug().
		Str("token_type", tokenType).
		Msg("Token format validated")

	// Validate token with GitHub API (with caching)
	authResult, err := a.githubClient.Validate(r.Context(), githubToken, a.requiredOrg, a.requiredTeams)
	if err != nil {
		return nil, fmt.Errorf("github validation failed: %w", err)
	}

	a.logger.Debug().
		Str("username", authResult.Username).
		Str("org", authResult.Org).
		Strs("teams", authResult.Teams).
		Str("token_type", authResult.TokenType).
		Msg("Client authenticated successfully")

	return authResult, nil
}

// AuthenticateAndInjectContext authenticates the request and injects AuthResult into context
func (a *ClientAuthenticator) AuthenticateAndInjectContext(r *http.Request) (*AuthResult, *http.Request, error) {
	authResult, err := a.AuthenticateRequest(r)
	if err != nil {
		return nil, r, err
	}

	// Add username to request context for logging/rate limiting
	ctx := middleware.SetUsername(r.Context(), authResult.Username)
	newReq := r.WithContext(ctx)

	return authResult, newReq, nil
}

// extractBearerToken extracts the token from a Bearer authentication header.
//
// Expected format: "Bearer <token>"
//
// Example:
//
//	Authorization: Bearer ghp_1234567890abcdefghijABCDEFGHIJ123456
//
// Returns:
//   - token: The extracted GitHub token
//   - error: Non-nil if the header format is invalid
func extractBearerToken(authHeader string) (string, error) {
	token := strings.TrimPrefix(authHeader, "Bearer ")
	token = strings.TrimSpace(token)

	if token == "" {
		return "", fmt.Errorf("empty bearer token")
	}

	return token, nil
}

// extractBasicAuthToken extracts the GitHub token from a Basic authentication header.
// The token can be in either the username or password field.
//
// Expected format: "Basic <base64(username:password)>"
//
// Common patterns:
//   - Docker: username=<any>, password=<github-token>
//   - Maven: username=<github-token>, password=<any>
//
// Examples:
//
//	Authorization: Basic dXNlcjpnaHBfMTIzNDU2Nzg5MGFiY2RlZmdoaWpBQkNERUZHSElKMTIzNDU2
//	(decodes to: user:ghp_1234567890abcdefghijABCDEFGHIJ123456)
//
// Returns:
//   - token: The extracted GitHub token (from password or username field)
//   - error: Non-nil if Basic auth parsing fails or no valid token found
func extractBasicAuthToken(authHeader string) (string, error) {
	username, password, err := ParseBasicAuth(authHeader)
	if err != nil {
		return "", fmt.Errorf("invalid basic auth: %w", err)
	}

	// Try password first (most common pattern for Docker/Maven clients)
	tokenType, _ := ValidateTokenFormat(password)
	if tokenType != TokenTypeUnknown {
		return password, nil
	}

	// Fallback to username
	tokenType, _ = ValidateTokenFormat(username)
	if tokenType != TokenTypeUnknown {
		return username, nil
	}

	return "", fmt.Errorf("no valid GitHub token found in basic auth credentials")
}

// ParseBasicAuth parses a Basic auth header
// Returns username and password from "Basic base64(username:password)"
func ParseBasicAuth(authHeader string) (username, password string, err error) {
	encoded := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", err
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid basic auth format")
	}

	return parts[0], parts[1], nil
}

// InjectAuthCredentials injects authentication credentials into request
// This replaces the client's auth header with backend credentials
func InjectAuthCredentials(r *http.Request, authConfig *config.AuthConfig) {
	// Defensive: If no auth configured, return early
	if authConfig == nil {
		return
	}

	// Always remove client auth header to prevent it from being sent to backend
	r.Header.Del("Authorization")

	switch authConfig.Type {
	case "basic":
		// Defensive: Only inject if both username and password are provided
		if authConfig.Username != "" || authConfig.Password != "" {
			r.SetBasicAuth(authConfig.Username, authConfig.Password)
		}

	case "bearer":
		// Defensive: Only inject if token is provided
		if authConfig.Token != "" {
			r.Header.Set("Authorization", "Bearer "+authConfig.Token)
		}

	case "custom":
		// Defensive: Only inject if both header name and value are provided
		if authConfig.HeaderName != "" && authConfig.HeaderValue != "" {
			r.Header.Set(authConfig.HeaderName, authConfig.HeaderValue)
		}
	}
}

// GetRequiredOrg returns the required GitHub organization
func (a *ClientAuthenticator) GetRequiredOrg() string {
	return a.requiredOrg
}
