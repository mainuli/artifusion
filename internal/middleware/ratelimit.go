package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/mainuli/artifusion/internal/config"
	"github.com/mainuli/artifusion/internal/constants"
	"github.com/mainuli/artifusion/internal/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

// userLimiter wraps a rate limiter with last access time for cleanup
type userLimiter struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// RateLimiter implements global and per-user rate limiting using token bucket algorithm
type RateLimiter struct {
	config        *config.RateLimitConfig
	global        *rate.Limiter
	perUser       map[string]*userLimiter
	mu            sync.RWMutex
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
	stopOnce      sync.Once // Ensures Stop() is called only once
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(cfg *config.RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		config:      cfg,
		perUser:     make(map[string]*userLimiter),
		stopCleanup: make(chan struct{}),
	}

	// Create global rate limiter if enabled
	if cfg.Enabled {
		rl.global = rate.NewLimiter(rate.Limit(cfg.RequestsPerSec), cfg.Burst)
	}

	// Start cleanup goroutine to remove stale per-user limiters
	if cfg.PerUserEnabled {
		rl.cleanupTicker = time.NewTicker(constants.RateLimiterCleanupInterval)
		go rl.cleanupStaleUserLimiters()
	}

	return rl
}

// Middleware returns a middleware handler that enforces rate limits
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check global rate limit first
		if rl.config.Enabled && !rl.global.Allow() {
			errors.ErrorResponse(w, errors.ErrGlobalRateLimitExceeded)
			return
		}

		// Check per-user rate limit
		if rl.config.PerUserEnabled {
			// Extract username from context (set by auth middleware)
			username := getUsernameFromContext(r.Context())
			if username != "" {
				limiter := rl.getUserLimiter(username)
				if !limiter.Allow() {
					errors.ErrorResponse(w, errors.ErrUserRateLimitExceeded)
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// getUserLimiter gets or creates a rate limiter for a specific user
func (rl *RateLimiter) getUserLimiter(username string) *rate.Limiter {
	now := time.Now()

	// Try read lock first (fast path)
	rl.mu.RLock()
	ul, exists := rl.perUser[username]
	if exists {
		// Important: Copy the limiter reference while holding read lock
		limiterRef := ul.limiter
		rl.mu.RUnlock()

		// Update last access time with write lock
		rl.mu.Lock()
		// Double-check the limiter still exists (could be deleted by cleanup)
		if ul, stillExists := rl.perUser[username]; stillExists {
			ul.lastAccess = now
		}
		rl.mu.Unlock()

		return limiterRef
	}
	rl.mu.RUnlock()

	// Create new limiter (slow path)
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock to prevent duplicate creation
	if ul, exists := rl.perUser[username]; exists {
		ul.lastAccess = now
		return ul.limiter
	}

	// Create new per-user limiter with current timestamp
	newLimiter := &userLimiter{
		limiter:    rate.NewLimiter(rate.Limit(rl.config.PerUserRequests), rl.config.PerUserBurst),
		lastAccess: now,
	}
	rl.perUser[username] = newLimiter

	return newLimiter.limiter
}

// cleanupStaleUserLimiters periodically removes limiters that haven't been used recently
// This prevents unbounded memory growth with many unique users
func (rl *RateLimiter) cleanupStaleUserLimiters() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.mu.Lock()
			now := time.Now()
			removedCount := 0

			// Only remove limiters that have been inactive beyond threshold
			for username, ul := range rl.perUser {
				if now.Sub(ul.lastAccess) > constants.RateLimiterInactivityThreshold {
					delete(rl.perUser, username)
					removedCount++
				}
			}

			rl.mu.Unlock()

			// Log cleanup activity if any limiters were removed
			if removedCount > 0 {
				log.Debug().
					Int("removed_count", removedCount).
					Int("remaining_count", len(rl.perUser)).
					Msg("Rate limiter cleanup: removed stale user limiters")
			}

		case <-rl.stopCleanup:
			return
		}
	}
}

// Stop stops the rate limiter cleanup goroutine
// Uses sync.Once to prevent double-close panic and ensure proper cleanup order
func (rl *RateLimiter) Stop() {
	if rl.config.PerUserEnabled {
		rl.stopOnce.Do(func() {
			// Stop ticker first to prevent new cleanup cycles
			rl.cleanupTicker.Stop()
			// Close channel to signal goroutine to exit
			close(rl.stopCleanup)
		})
	}
}

// getUsernameFromContext extracts the authenticated username from the request context
// This should be set by the authentication middleware
func getUsernameFromContext(ctx context.Context) string {
	return GetUsername(ctx)
}
