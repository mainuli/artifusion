package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync/atomic"
	"time"

	"github.com/mainuli/artifusion/internal/constants"
	"github.com/patrickmn/go-cache"
	"golang.org/x/sync/singleflight"
)

// AuthResult represents the result of successful authentication
type AuthResult struct {
	Username   string
	Org        string
	Teams      []string
	TokenType  string // "pat" or "github_actions"
	Repository string // For GitHub Actions: "owner/repo" (empty for PATs)
}

// AuthCache provides thread-safe caching of authentication results
// with singleflight to prevent thundering herd on cache miss
type AuthCache struct {
	cache        *cache.Cache
	ttl          time.Duration
	singleflight singleflight.Group

	// Metrics (atomic for thread-safety)
	hits   atomic.Int64
	misses atomic.Int64
	size   atomic.Int32
}

// NewAuthCache creates a new authentication cache
func NewAuthCache(ttl time.Duration) *AuthCache {
	// Create cache with TTL and cleanup interval
	// Cleanup interval is TTL * CacheCleanupMultiplier
	cleanupInterval := ttl * constants.CacheCleanupMultiplier
	c := cache.New(ttl, cleanupInterval)

	return &AuthCache{
		cache: c,
		ttl:   ttl,
	}
}

// Get retrieves cached auth result or validates with GitHub
// Uses singleflight to prevent multiple concurrent validations for same PAT
func (c *AuthCache) Get(ctx context.Context, pat string, validator func(context.Context) (*AuthResult, error)) (*AuthResult, error) {
	key := c.hashPAT(pat)

	// Try cache first (fast path - no lock contention)
	if result, found := c.cache.Get(key); found {
		c.hits.Add(1)
		return result.(*AuthResult), nil
	}

	c.misses.Add(1)

	// Use singleflight to ensure only one validation per PAT
	// This prevents thundering herd when cache expires
	result, err, _ := c.singleflight.Do(key, func() (interface{}, error) {
		// Double-check cache (might have been populated while waiting)
		if result, found := c.cache.Get(key); found {
			return result.(*AuthResult), nil
		}

		// Validate with GitHub API
		authResult, err := validator(ctx)
		if err != nil {
			return nil, err
		}

		// Cache the result
		c.cache.Set(key, authResult, c.ttl)
		c.size.Add(1)

		return authResult, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*AuthResult), nil
}

// Invalidate removes a PAT from the cache
func (c *AuthCache) Invalidate(pat string) {
	key := c.hashPAT(pat)
	c.cache.Delete(key)
	c.size.Add(-1)
}

// Clear removes all entries from the cache
func (c *AuthCache) Clear() {
	c.cache.Flush()
	c.size.Store(0)
}

// Stats returns cache statistics
func (c *AuthCache) Stats() CacheStats {
	return CacheStats{
		Hits:   c.hits.Load(),
		Misses: c.misses.Load(),
		Size:   int(c.size.Load()),
		HitRate: func() float64 {
			hits := c.hits.Load()
			misses := c.misses.Load()
			total := hits + misses
			if total == 0 {
				return 0
			}
			return float64(hits) / float64(total)
		}(),
	}
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits    int64
	Misses  int64
	Size    int
	HitRate float64
}

// hashPAT creates a SHA256 hash of the PAT for cache key
// This prevents storing actual PATs in memory
func (c *AuthCache) hashPAT(pat string) string {
	hash := sha256.Sum256([]byte(pat))
	return hex.EncodeToString(hash[:])
}
