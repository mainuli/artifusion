package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mainuli/artifusion/internal/config"
)

// TestRateLimiter_GlobalLimit tests global rate limiting
func TestRateLimiter_GlobalLimit(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:        true,
		RequestsPerSec: 10,
		Burst:          10,
		PerUserEnabled: false,
	}

	rl := NewRateLimiter(cfg)
	defer rl.Stop()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First 10 requests should succeed (burst capacity)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i, rec.Code)
		}
	}

	// 11th request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 (rate limited), got %d", rec.Code)
	}
}

// TestRateLimiter_PerUserLimit tests per-user rate limiting
func TestRateLimiter_PerUserLimit(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:         false,
		PerUserEnabled:  true,
		PerUserRequests: 5,
		PerUserBurst:    5,
	}

	rl := NewRateLimiter(cfg)
	defer rl.Stop()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make 5 requests from user1 (should all succeed)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := SetUsername(req.Context(), "user1")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("user1 request %d: expected 200, got %d", i, rec.Code)
		}
	}

	// 6th request from user1 should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := SetUsername(req.Context(), "user1")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 for user1 (rate limited), got %d", rec.Code)
	}

	// But user2 should still be able to make requests
	req2 := httptest.NewRequest("GET", "/test", nil)
	ctx2 := SetUsername(req2.Context(), "user2")
	req2 = req2.WithContext(ctx2)

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200 for user2, got %d", rec2.Code)
	}
}

// TestRateLimiter_PerUserLimit_MultipleUsers tests isolation between users
func TestRateLimiter_PerUserLimit_MultipleUsers(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:         false,
		PerUserEnabled:  true,
		PerUserRequests: 3,
		PerUserBurst:    3,
	}

	rl := NewRateLimiter(cfg)
	defer rl.Stop()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	users := []string{"alice", "bob", "charlie"}

	// Each user should be able to make their own requests independently
	for _, username := range users {
		// Each user makes 3 requests (their burst limit)
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			ctx := SetUsername(req.Context(), username)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("user %s request %d: expected 200, got %d", username, i, rec.Code)
			}
		}

		// 4th request should be rate limited for each user
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := SetUsername(req.Context(), username)
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("user %s: expected 429 (rate limited), got %d", username, rec.Code)
		}
	}
}

// TestRateLimiter_BothLimits tests that both global and per-user limits work together
func TestRateLimiter_BothLimits(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:         true,
		RequestsPerSec:  100, // High global limit
		Burst:           100,
		PerUserEnabled:  true,
		PerUserRequests: 2, // Low per-user limit
		PerUserBurst:    2,
	}

	rl := NewRateLimiter(cfg)
	defer rl.Stop()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// user1 makes 2 requests (should succeed)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := SetUsername(req.Context(), "user1")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("user1 request %d: expected 200, got %d", i, rec.Code)
		}
	}

	// user1's 3rd request should be rate limited by per-user limit
	// even though global limit hasn't been reached
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := SetUsername(req.Context(), "user1")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 (per-user rate limited), got %d", rec.Code)
	}
}

// TestRateLimiter_NoUsername tests that requests without username work with global limit only
func TestRateLimiter_NoUsername(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:         true,
		RequestsPerSec:  5,
		Burst:           5,
		PerUserEnabled:  true,
		PerUserRequests: 2,
		PerUserBurst:    2,
	}

	rl := NewRateLimiter(cfg)
	defer rl.Stop()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Requests without username should only be subject to global limit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		// No username set in context

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i, rec.Code)
		}
	}

	// 6th request should be rate limited by global limit
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 (global rate limited), got %d", rec.Code)
	}
}

// TestRateLimiter_Disabled tests that disabled rate limiter allows all requests
func TestRateLimiter_Disabled(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:        false,
		PerUserEnabled: false,
	}

	rl := NewRateLimiter(cfg)
	defer rl.Stop()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Should allow many requests without rate limiting
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200 (no rate limiting), got %d", i, rec.Code)
		}
	}
}

// TestRateLimiter_RefillRate tests that rate limiter refills tokens over time
func TestRateLimiter_RefillRate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping refill rate test in short mode")
	}

	cfg := &config.RateLimitConfig{
		Enabled:         false,
		PerUserEnabled:  true,
		PerUserRequests: 2, // 2 requests per second
		PerUserBurst:    2,
	}

	rl := NewRateLimiter(cfg)
	defer rl.Stop()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Use up the burst capacity
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		ctx := SetUsername(req.Context(), "user1")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i, rec.Code)
		}
	}

	// Next request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := SetUsername(req.Context(), "user1")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 (rate limited), got %d", rec.Code)
	}

	// Wait for token refill (at 2 req/sec, wait 1 second for 2 tokens)
	time.Sleep(1100 * time.Millisecond)

	// Should be able to make requests again
	req2 := httptest.NewRequest("GET", "/test", nil)
	ctx2 := SetUsername(req2.Context(), "user1")
	req2 = req2.WithContext(ctx2)

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200 after refill, got %d", rec2.Code)
	}
}

// TestRateLimiter_GetUserLimiter_DoubleCheckedLocking tests concurrency safety
func TestRateLimiter_GetUserLimiter_DoubleCheckedLocking(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:         false,
		PerUserEnabled:  true,
		PerUserRequests: 10,
		PerUserBurst:    10,
	}

	rl := NewRateLimiter(cfg)
	defer rl.Stop()

	// Launch multiple goroutines trying to get limiter for same user concurrently
	const numGoroutines = 100
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			limiter := rl.getUserLimiter("testuser")
			if limiter == nil {
				t.Error("getUserLimiter returned nil")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Should only have one limiter for the user
	rl.mu.RLock()
	if len(rl.perUser) != 1 {
		t.Errorf("expected 1 limiter, got %d", len(rl.perUser))
	}
	rl.mu.RUnlock()
}

// TestRateLimiter_Stop tests cleanup on stop
func TestRateLimiter_Stop(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:         false,
		PerUserEnabled:  true,
		PerUserRequests: 10,
		PerUserBurst:    10,
	}

	rl := NewRateLimiter(cfg)

	// Stop should not panic
	rl.Stop()

	// Note: Cannot call Stop multiple times as it closes a channel
	// In production, Stop should only be called once during shutdown
}

// TestRateLimiter_InfrastructureEndpoints tests that health/ready/metrics endpoints
// are exempt from rate limiting (critical for Kubernetes probes and monitoring)
func TestRateLimiter_InfrastructureEndpoints(t *testing.T) {
	// Configure very restrictive rate limits to ensure exemption is working
	cfg := &config.RateLimitConfig{
		Enabled:        true,
		RequestsPerSec: 1, // Very low
		Burst:          1, // Very low
	}

	rl := NewRateLimiter(cfg)
	defer rl.Stop()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	infrastructureEndpoints := []string{"/health", "/ready", "/metrics"}

	// Make many requests to each infrastructure endpoint
	// They should all succeed despite very low rate limit
	for _, endpoint := range infrastructureEndpoints {
		for i := 0; i < 100; i++ {
			req := httptest.NewRequest("GET", endpoint, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("%s request %d: expected 200 (exempt from rate limit), got %d",
					endpoint, i, rec.Code)
			}
		}
	}

	// Regular endpoint should still be rate limited
	req := httptest.NewRequest("GET", "/api/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/api/test first request: expected 200, got %d", rec.Code)
	}

	// Second request to regular endpoint should be rate limited
	req2 := httptest.NewRequest("GET", "/api/test", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("/api/test second request: expected 429 (rate limited), got %d", rec2.Code)
	}
}

// TestIsInfrastructureEndpoint tests the helper function
func TestIsInfrastructureEndpoint(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/health", true},
		{"/ready", true},
		{"/metrics", true},
		{"/v2/", false},
		{"/maven/", false},
		{"/npm/", false},
		{"/api/test", false},
		{"/health/extra", false},    // Not exact match
		{"/ready/something", false}, // Not exact match
		{"/healthz", false},         // Similar but not same
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isInfrastructureEndpoint(tt.path)
			if got != tt.want {
				t.Errorf("isInfrastructureEndpoint(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestRateLimiter_InfrastructureEndpoints_WithPerUserLimit tests that infrastructure
// endpoints bypass both global and per-user rate limits
func TestRateLimiter_InfrastructureEndpoints_WithPerUserLimit(t *testing.T) {
	cfg := &config.RateLimitConfig{
		Enabled:         true,
		RequestsPerSec:  1,
		Burst:           1,
		PerUserEnabled:  true,
		PerUserRequests: 1,
		PerUserBurst:    1,
	}

	rl := NewRateLimiter(cfg)
	defer rl.Stop()

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make many authenticated requests to /health
	// Should all succeed despite per-user rate limit
	for i := 0; i < 50; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		ctx := SetUsername(req.Context(), "testuser")
		req = req.WithContext(ctx)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("/health request %d: expected 200 (exempt from per-user rate limit), got %d",
				i, rec.Code)
		}
	}

	// But regular endpoint with same user should still be rate limited
	req := httptest.NewRequest("GET", "/api/test", nil)
	ctx := SetUsername(req.Context(), "testuser")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/api/test first request: expected 200, got %d", rec.Code)
	}

	// Second request should be rate limited
	req2 := httptest.NewRequest("GET", "/api/test", nil)
	ctx2 := SetUsername(req2.Context(), "testuser")
	req2 = req2.WithContext(ctx2)

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusTooManyRequests {
		t.Errorf("/api/test second request: expected 429 (per-user rate limited), got %d", rec2.Code)
	}
}
