// Package auth provides GitHub-based authentication for the Artifusion reverse proxy.
//
// The package implements a high-performance authentication system with the following features:
//   - Support for GitHub Personal Access Tokens (classic and fine-grained)
//   - Support for GitHub Actions installation tokens
//   - Organization and team membership validation
//   - LRU caching with TTL to reduce GitHub API load
//   - Singleflight request coalescing to prevent thundering herd
//   - Rate limiting to stay within GitHub API limits
//   - Connection pooling for high concurrency
//
// Design Principles:
//   - DRY: Common code extracted into reusable helper methods
//   - Single Responsibility: Each function has one clear purpose
//   - Security: Error messages sanitized to prevent information disclosure
//   - Performance: Length pre-checks and caching optimize for common paths
//
// Usage Example:
//
//	logger := zerolog.New(os.Stdout)
//	githubClient := NewGitHubClient("https://api.github.com", 5*time.Minute, 0, logger)
//	result, err := githubClient.Validate(ctx, token, "my-org", []string{"my-team"})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Authenticated as: %s\n", result.Username)
package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/google/go-github/v58/github"
	"github.com/mainuli/artifusion/internal/constants"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

// GitHubClient wraps the GitHub API client with connection pooling and rate limiting.
// It provides thread-safe authentication validation with caching and singleflight request coalescing.
//
// Design rationale:
//   - Caching: Reduces GitHub API load and improves response times for repeated validations
//   - Rate limiting: Prevents exhausting GitHub API rate limits (5000 req/hr)
//   - Connection pooling: Optimizes for high concurrency scenarios
//   - Singleflight: Prevents thundering herd during cache misses
//
// Thread safety: All methods are safe for concurrent use.
type GitHubClient struct {
	baseURL         string        // GitHub API base URL (supports enterprise)
	rateLimit       *rate.Limiter // Token bucket rate limiter
	rateLimitBuffer int           // Buffer to stay below GitHub's actual limits
	cache           *AuthCache    // LRU cache with TTL and singleflight
	logger          zerolog.Logger
}

// NewGitHubClient creates a new GitHub client optimized for high concurrency.
//
// Parameters:
//   - apiURL: GitHub API base URL (e.g., "https://api.github.com" or enterprise URL)
//   - cacheTTL: Time-to-live for cached authentication results
//   - rateLimitBuffer: Buffer below GitHub's rate limit (in requests/hour)
//   - logger: Structured logger for debug output and error tracking
//
// The rate limiter is configured at 1.2 req/sec with burst of 50, which translates
// to approximately 4,320 req/hr - well below GitHub's 5,000 req/hr limit.
//
// Returns a fully initialized GitHubClient ready for concurrent use.
func NewGitHubClient(apiURL string, cacheTTL time.Duration, rateLimitBuffer int, logger zerolog.Logger) *GitHubClient {
	// Create auth cache
	cache := NewAuthCache(cacheTTL)

	// Rate limiter: GitHub allows 5000 req/hr = ~1.4 req/sec
	// We use 1.2 req/sec with burst of 50 to better handle traffic spikes
	// while staying well below GitHub's actual limits with the configured buffer
	limiter := rate.NewLimiter(rate.Limit(1.2), 50)

	return &GitHubClient{
		baseURL:         apiURL,
		rateLimit:       limiter,
		rateLimitBuffer: rateLimitBuffer,
		cache:           cache,
		logger:          logger,
	}
}

// Validate authenticates a GitHub token and validates organization/team membership.
// It uses caching with singleflight to optimize for high concurrency.
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - pat: GitHub token (PAT or GitHub Actions token)
//   - requiredOrg: Required organization name (empty string skips org check)
//   - requiredTeams: Required team slugs (empty slice skips team check)
//
// Returns:
//   - *AuthResult: Authentication details including username, org, teams, and token type
//   - error: Non-nil if authentication fails
//
// The validation is cached based on the token, so subsequent calls with the same
// token will return cached results (until TTL expires) without hitting GitHub API.
func (c *GitHubClient) Validate(ctx context.Context, pat string, requiredOrg string, requiredTeams []string) (*AuthResult, error) {
	// Use cache with singleflight
	return c.cache.Get(ctx, pat, func(ctx context.Context) (*AuthResult, error) {
		return c.validateWithGitHub(ctx, pat, requiredOrg, requiredTeams)
	})
}

// validateWithGitHub performs actual GitHub API validation and routes to appropriate validator
func (c *GitHubClient) validateWithGitHub(ctx context.Context, token string, requiredOrg string, requiredTeams []string) (*AuthResult, error) {
	// Wait for rate limit slot
	if err := c.rateLimit.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit: %w", err)
	}

	// Determine token type (already validated by caller via ValidateTokenFormat)
	tokenType, _ := ValidateTokenFormat(token)

	// Route to appropriate validation method based on token type
	switch tokenType {
	case TokenTypeGitHubActions:
		return c.validateGitHubActionsToken(ctx, token, requiredOrg)
	case TokenTypePAT:
		return c.validatePATToken(ctx, token, requiredOrg, requiredTeams)
	default:
		// Should never reach here due to preemptive validation
		return nil, fmt.Errorf("unsupported token type: %s", tokenType)
	}
}

// createGitHubClient creates a GitHub API client with OAuth2 authentication and enterprise URL support.
// This helper eliminates code duplication between PAT and GitHub Actions token validation flows.
//
// Parameters:
//   - token: The GitHub token (PAT or GitHub Actions token) to authenticate with
//
// Returns:
//   - *github.Client: Configured GitHub API client ready for use
//   - error: Non-nil if enterprise URL configuration fails
//
// The client is created with an optimized HTTP transport for high concurrency (see createHTTPClient).
func (c *GitHubClient) createGitHubClient(token string) (*github.Client, error) {
	// Create HTTP client with connection pooling optimized for concurrency
	httpClient := c.createHTTPClient(token)

	// Create GitHub client
	client := github.NewClient(httpClient)

	// Configure enterprise URLs if not using GitHub.com
	if c.baseURL != "https://api.github.com" {
		var err error
		client, err = client.WithEnterpriseURLs(c.baseURL, c.baseURL)
		if err != nil {
			return nil, fmt.Errorf("invalid GitHub API URL: %w", err)
		}
	}

	return client, nil
}

// validatePATToken validates a Personal Access Token (classic or fine-grained).
//
// Validation steps:
//  1. Authenticate with GitHub API using the PAT
//  2. Retrieve the authenticated user's username
//  3. If requiredOrg is set, verify organization membership
//  4. If requiredTeams is set, verify membership in at least one required team
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - token: GitHub Personal Access Token (ghp_* or github_pat_*)
//   - requiredOrg: Organization to check membership (empty to skip)
//   - requiredTeams: Teams to check membership (empty to skip)
//
// Returns AuthResult with username, org, and teams on success.
// Returns error if token is invalid or membership checks fail.
func (c *GitHubClient) validatePATToken(ctx context.Context, token string, requiredOrg string, requiredTeams []string) (*AuthResult, error) {
	// Create GitHub client with enterprise URL support
	client, err := c.createGitHubClient(token)
	if err != nil {
		return nil, err
	}

	// Get authenticated user
	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	username := user.GetLogin()
	if username == "" {
		return nil, fmt.Errorf("failed to get username")
	}

	// Check organization membership (only if requiredOrg is specified)
	var userTeams []string
	orgToReturn := requiredOrg

	if requiredOrg != "" {
		isMember, _, err := client.Organizations.IsMember(ctx, requiredOrg, username)
		if err != nil {
			// SECURITY: Sanitize error to avoid exposing internal details
			// Log the actual error internally, but return a generic message to the client
			c.logger.Debug().
				Err(err).
				Str("org", requiredOrg).
				Str("username", username).
				Msg("GitHub API error during organization membership check")
			return nil, fmt.Errorf("authentication failed: unable to verify organization membership")
		}

		if !isMember {
			// SECURITY: Generic error message that doesn't reveal the organization name
			// This prevents enumeration attacks
			return nil, fmt.Errorf("authentication failed: insufficient permissions")
		}

		// Check team membership if required
		if len(requiredTeams) > 0 {
			found := false
			for _, team := range requiredTeams {
				membership, _, err := client.Teams.GetTeamMembershipBySlug(ctx, requiredOrg, team, username)
				if err == nil && membership.GetState() == "active" {
					userTeams = append(userTeams, team)
					found = true
				}
			}

			if !found {
				// SECURITY: Generic error message that doesn't reveal team names
				// This prevents enumeration attacks
				return nil, fmt.Errorf("authentication failed: insufficient permissions")
			}
		}
	}
	// If requiredOrg is empty, skip org/team checks - PAT validation via Users.Get is sufficient

	return &AuthResult{
		Username:   username,
		Org:        orgToReturn,
		Teams:      userTeams,
		TokenType:  TokenTypePAT,
		Repository: "", // Not applicable for PATs
	}, nil
}

// validateGitHubActionsToken validates a GitHub Actions installation token (ghs_).
//
// GitHub Actions tokens are scoped to repositories and have different permissions
// than PATs. This method validates by fetching accessible repositories and extracting
// the owner information.
//
// Validation steps:
//  1. Call /installation/repositories endpoint (optimized to fetch only 1 repo)
//  2. Extract repository owner from the response
//  3. If requiredOrg is set, verify the owner matches
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - token: GitHub Actions installation token (ghs_*)
//   - requiredOrg: Organization to match against repo owner (empty to skip)
//
// Returns AuthResult with "github-actions[bot]" as username and repository info.
// Team membership checks are not applicable for installation tokens.
func (c *GitHubClient) validateGitHubActionsToken(ctx context.Context, token string, requiredOrg string) (*AuthResult, error) {
	// Create GitHub client with enterprise URL support
	client, err := c.createGitHubClient(token)
	if err != nil {
		return nil, err
	}

	// Call /installation/repositories with limit of 1 for fast response
	// We only need one repository to extract the owner information
	repos, _, err := client.Apps.ListRepos(ctx, &github.ListOptions{
		PerPage: 1, // OPTIMIZATION: Only fetch one repo to get owner
	})
	if err != nil {
		c.logger.Debug().
			Err(err).
			Msg("GitHub API error during installation repositories fetch")
		return nil, fmt.Errorf("failed to fetch installation repositories: %w", err)
	}

	if repos.TotalCount == nil || *repos.TotalCount == 0 {
		return nil, fmt.Errorf("no repositories found for GitHub Actions token")
	}

	if len(repos.Repositories) == 0 {
		return nil, fmt.Errorf("repositories array is empty")
	}

	// Extract repository owner from first (and only) repository
	repo := repos.Repositories[0]
	repoOwner := repo.Owner.GetLogin()
	fullRepoName := repo.GetFullName()

	if repoOwner == "" {
		return nil, fmt.Errorf("failed to get repository owner")
	}

	// Validate org only if requiredOrg is configured
	if requiredOrg != "" {
		if repoOwner != requiredOrg {
			// SECURITY: Generic error message that doesn't reveal the organization name
			// This prevents enumeration attacks
			return nil, fmt.Errorf("authentication failed: insufficient permissions")
		}
	}

	return &AuthResult{
		Username:   "github-actions[bot]",
		Org:        repoOwner,
		Repository: fullRepoName,
		TokenType:  TokenTypeGitHubActions,
		Teams:      nil, // Not applicable for installation tokens
	}, nil
}

// createHTTPClient creates an HTTP client optimized for high concurrency.
// All timeout and connection pool values are defined as constants for easy tuning.
//
// Configuration details:
//   - Connection pooling: Maintains up to 100 idle connections total, 10 per host
//   - Timeouts: 30s overall, 10s for dial and TLS handshake
//   - Keep-alive: 30s interval, 90s idle timeout
//
// These values are optimized for GitHub API's characteristics and rate limits.
func (c *GitHubClient) createHTTPClient(pat string) *http.Client {
	// Create transport with aggressive connection pooling
	transport := &http.Transport{
		// Connection pooling - constants from internal/constants/timeouts.go
		MaxIdleConns:        constants.GitHubMaxIdleConns,
		MaxIdleConnsPerHost: constants.GitHubMaxIdleConnsPerHost,
		IdleConnTimeout:     constants.GitHubIdleConnTimeout,

		// Connection establishment - constants from internal/constants/timeouts.go
		DialContext: (&net.Dialer{
			Timeout:   constants.GitHubDialTimeout,
			KeepAlive: constants.GitHubKeepAlive,
		}).DialContext,

		// TLS optimization - constants from internal/constants/timeouts.go
		TLSHandshakeTimeout:   constants.GitHubTLSHandshakeTimeout,
		ExpectContinueTimeout: constants.GitHubExpectContinueTimeout,

		// Reuse connections
		DisableKeepAlives: false,
	}

	// Create OAuth2 token source
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: pat},
	)

	// Create OAuth2 HTTP client with our transport
	// Note: We use context.Background() here intentionally as this is a configuration context
	// for the oauth2 client, not a request-scoped context. The actual request contexts are
	// passed to the GitHub API calls (e.g., client.Users.Get(ctx, "")).
	// This context is only used to configure the base HTTP client, which persists across requests.
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &http.Client{
		Transport: transport,
		Timeout:   constants.GitHubHTTPTimeout,
	})

	return oauth2.NewClient(ctx, ts)
}

// InvalidateCache removes a PAT from the cache
func (c *GitHubClient) InvalidateCache(pat string) {
	c.cache.Invalidate(pat)
}

// ClearCache removes all cached entries
func (c *GitHubClient) ClearCache() {
	c.cache.Clear()
}

// CacheStats returns cache statistics
func (c *GitHubClient) CacheStats() CacheStats {
	return c.cache.Stats()
}
