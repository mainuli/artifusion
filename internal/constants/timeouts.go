package constants

import "time"

// Timeout and interval constants used throughout the application
// Centralized here for easy tuning and consistency

const (
	// Rate Limiter Configuration
	// RateLimiterCleanupInterval defines how often we remove stale per-user rate limiters
	RateLimiterCleanupInterval = 5 * time.Minute

	// RateLimiterInactivityThreshold defines when a per-user limiter is considered stale
	// Limiters inactive for longer than this duration will be removed during cleanup
	RateLimiterInactivityThreshold = 1 * time.Hour

	// Health Check Configuration
	// HealthCheckTimeout defines maximum time allowed for health check operations
	// This prevents health checks from blocking indefinitely
	HealthCheckTimeout = 10 * time.Second

	// Cache Configuration
	// CacheCleanupMultiplier determines cleanup interval based on TTL
	// Cleanup interval = TTL * CacheCleanupMultiplier
	// A multiplier of 2 means cleanup runs twice as often as TTL expiration
	CacheCleanupMultiplier = 2

	// Request Timeout Configuration
	// DefaultRequestTimeout is the default timeout for all HTTP requests
	// This provides a reasonable upper bound for most requests
	DefaultRequestTimeout = 30 * time.Second

	// GitHub HTTP Client Configuration
	// These timeouts are optimized for GitHub API calls with high concurrency

	// GitHubHTTPTimeout is the overall timeout for GitHub HTTP requests
	// This includes time for connection, TLS handshake, request, and response
	GitHubHTTPTimeout = 30 * time.Second

	// GitHubDialTimeout is the maximum time to wait for a TCP connection to be established
	GitHubDialTimeout = 10 * time.Second

	// GitHubKeepAlive is the interval for sending keep-alive probes on idle connections
	// This prevents connections from being closed by intermediate proxies
	GitHubKeepAlive = 30 * time.Second

	// GitHubTLSHandshakeTimeout is the maximum time to wait for TLS handshake
	GitHubTLSHandshakeTimeout = 10 * time.Second

	// GitHubExpectContinueTimeout is the time to wait for a server's first response headers
	// after fully writing the request headers if the request has an "Expect: 100-continue" header
	GitHubExpectContinueTimeout = 1 * time.Second

	// GitHubIdleConnTimeout is the maximum time an idle connection will remain in the pool
	GitHubIdleConnTimeout = 90 * time.Second
)

// GitHub HTTP Client Connection Pool Configuration
// These values are optimized for high concurrency with GitHub API
const (
	// GitHubMaxIdleConns is the maximum number of idle connections across all hosts
	GitHubMaxIdleConns = 100

	// GitHubMaxIdleConnsPerHost is the maximum number of idle connections per host
	GitHubMaxIdleConnsPerHost = 10
)
