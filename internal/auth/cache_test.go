package auth

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestAuthCache_Get_CacheHit tests that cached results are returned without validation
func TestAuthCache_Get_CacheHit(t *testing.T) {
	cache := NewAuthCache(5 * time.Minute)
	validatorCalls := atomic.Int32{}

	validator := func(ctx context.Context) (*AuthResult, error) {
		validatorCalls.Add(1)
		return &AuthResult{
			Username: "testuser",
			Org:      "testorg",
			Teams:    []string{"team1"},
		}, nil
	}

	// First call - should call validator
	result, err := cache.Get(context.Background(), "test-pat", validator)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", result.Username)
	}

	if validatorCalls.Load() != 1 {
		t.Errorf("expected 1 validator call, got %d", validatorCalls.Load())
	}

	// Second call - should use cache
	result2, err := cache.Get(context.Background(), "test-pat", validator)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result2.Username != "testuser" {
		t.Errorf("expected username 'testuser', got '%s'", result2.Username)
	}

	if validatorCalls.Load() != 1 {
		t.Errorf("expected 1 validator call (cached), got %d", validatorCalls.Load())
	}

	// Verify cache stats
	stats := cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("expected 1 cache hit, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("expected 1 cache miss, got %d", stats.Misses)
	}
}

// TestAuthCache_Get_CacheMiss tests that validator is called on cache miss
func TestAuthCache_Get_CacheMiss(t *testing.T) {
	cache := NewAuthCache(5 * time.Minute)
	validatorCalls := atomic.Int32{}

	validator := func(ctx context.Context) (*AuthResult, error) {
		validatorCalls.Add(1)
		return &AuthResult{
			Username: "testuser",
			Org:      "testorg",
		}, nil
	}

	// First call - cache miss
	_, err := cache.Get(context.Background(), "test-pat", validator)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if validatorCalls.Load() != 1 {
		t.Errorf("expected 1 validator call, got %d", validatorCalls.Load())
	}

	stats := cache.Stats()
	if stats.Misses != 1 {
		t.Errorf("expected 1 cache miss, got %d", stats.Misses)
	}
}

// TestAuthCache_Singleflight tests that concurrent requests for same PAT only call validator once
func TestAuthCache_Singleflight(t *testing.T) {
	cache := NewAuthCache(5 * time.Minute)
	validatorCalls := atomic.Int32{}

	validator := func(ctx context.Context) (*AuthResult, error) {
		validatorCalls.Add(1)
		// Simulate slow validation
		time.Sleep(100 * time.Millisecond)
		return &AuthResult{
			Username: "testuser",
			Org:      "testorg",
			Teams:    []string{"team1", "team2"},
		}, nil
	}

	// Launch 100 concurrent requests for the same PAT
	const numGoroutines = 100
	var wg sync.WaitGroup
	results := make([]*AuthResult, numGoroutines)
	errs := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			result, err := cache.Get(context.Background(), "same-pat", validator)
			results[index] = result
			errs[index] = err
		}(i)
	}

	wg.Wait()

	// Verify validator was only called ONCE due to singleflight
	if validatorCalls.Load() != 1 {
		t.Errorf("expected 1 validator call due to singleflight, got %d", validatorCalls.Load())
	}

	// Verify all goroutines got the same result
	for i := 0; i < numGoroutines; i++ {
		if errs[i] != nil {
			t.Errorf("goroutine %d got error: %v", i, errs[i])
		}
		if results[i] == nil {
			t.Errorf("goroutine %d got nil result", i)
			continue
		}
		if results[i].Username != "testuser" {
			t.Errorf("goroutine %d got wrong username: %s", i, results[i].Username)
		}
	}

	// Verify cache stats - all 100 goroutines will increment misses counter before singleflight coalesces
	// Only the first one triggers actual validation, but all register as cache misses
	stats := cache.Stats()
	if stats.Misses < 1 {
		t.Errorf("expected at least 1 cache miss, got %d", stats.Misses)
	}
}

// TestAuthCache_Singleflight_DifferentPATs tests that different PATs are not coalesced
func TestAuthCache_Singleflight_DifferentPATs(t *testing.T) {
	cache := NewAuthCache(5 * time.Minute)
	validatorCalls := atomic.Int32{}

	validator := func(ctx context.Context) (*AuthResult, error) {
		validatorCalls.Add(1)
		time.Sleep(50 * time.Millisecond)
		return &AuthResult{
			Username: "testuser",
			Org:      "testorg",
		}, nil
	}

	// Launch concurrent requests for different PATs
	const numGoroutines = 10
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			pat := "different-pat-" + string(rune('0'+index))
			_, err := cache.Get(context.Background(), pat, validator)
			if err != nil {
				t.Errorf("goroutine %d got error: %v", index, err)
			}
		}(i)
	}

	wg.Wait()

	// Each different PAT should call validator separately
	if validatorCalls.Load() != numGoroutines {
		t.Errorf("expected %d validator calls (one per PAT), got %d", numGoroutines, validatorCalls.Load())
	}
}

// TestAuthCache_Get_ValidatorError tests that validator errors are returned
func TestAuthCache_Get_ValidatorError(t *testing.T) {
	cache := NewAuthCache(5 * time.Minute)
	expectedErr := errors.New("validation failed")

	validator := func(ctx context.Context) (*AuthResult, error) {
		return nil, expectedErr
	}

	result, err := cache.Get(context.Background(), "test-pat", validator)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error '%v', got '%v'", expectedErr, err)
	}

	if result != nil {
		t.Error("expected nil result on error")
	}

	// Verify errors are not cached
	validatorCalls := atomic.Int32{}
	validator2 := func(ctx context.Context) (*AuthResult, error) {
		validatorCalls.Add(1)
		return nil, expectedErr
	}

	// Second call should also call validator (errors not cached)
	_, _ = cache.Get(context.Background(), "test-pat", validator2)
	if validatorCalls.Load() != 1 {
		t.Error("expected validator to be called again after error")
	}
}

// TestAuthCache_Invalidate tests cache invalidation
func TestAuthCache_Invalidate(t *testing.T) {
	cache := NewAuthCache(5 * time.Minute)
	validatorCalls := atomic.Int32{}

	validator := func(ctx context.Context) (*AuthResult, error) {
		validatorCalls.Add(1)
		return &AuthResult{Username: "testuser"}, nil
	}

	// Cache the result
	_, err := cache.Get(context.Background(), "test-pat", validator)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if validatorCalls.Load() != 1 {
		t.Errorf("expected 1 validator call, got %d", validatorCalls.Load())
	}

	// Invalidate
	cache.Invalidate("test-pat")

	// Next call should call validator again
	_, err = cache.Get(context.Background(), "test-pat", validator)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if validatorCalls.Load() != 2 {
		t.Errorf("expected 2 validator calls after invalidation, got %d", validatorCalls.Load())
	}
}

// TestAuthCache_Clear tests clearing the entire cache
func TestAuthCache_Clear(t *testing.T) {
	cache := NewAuthCache(5 * time.Minute)

	validator := func(ctx context.Context) (*AuthResult, error) {
		return &AuthResult{Username: "testuser"}, nil
	}

	// Cache multiple PATs
	_, _ = cache.Get(context.Background(), "pat1", validator)
	_, _ = cache.Get(context.Background(), "pat2", validator)
	_, _ = cache.Get(context.Background(), "pat3", validator)

	stats := cache.Stats()
	if stats.Size != 3 {
		t.Errorf("expected cache size 3, got %d", stats.Size)
	}

	// Clear cache
	cache.Clear()

	stats = cache.Stats()
	if stats.Size != 0 {
		t.Errorf("expected cache size 0 after clear, got %d", stats.Size)
	}
}

// TestAuthCache_Stats tests cache statistics tracking
func TestAuthCache_Stats(t *testing.T) {
	cache := NewAuthCache(5 * time.Minute)

	validator := func(ctx context.Context) (*AuthResult, error) {
		return &AuthResult{Username: "testuser"}, nil
	}

	// First call - miss
	_, _ = cache.Get(context.Background(), "pat1", validator)

	// Second call - hit
	_, _ = cache.Get(context.Background(), "pat1", validator)

	// Third call different PAT - miss
	_, _ = cache.Get(context.Background(), "pat2", validator)

	stats := cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("expected 1 hit, got %d", stats.Hits)
	}
	if stats.Misses != 2 {
		t.Errorf("expected 2 misses, got %d", stats.Misses)
	}
	if stats.Size != 2 {
		t.Errorf("expected size 2, got %d", stats.Size)
	}

	expectedHitRate := 1.0 / 3.0
	if stats.HitRate < expectedHitRate-0.01 || stats.HitRate > expectedHitRate+0.01 {
		t.Errorf("expected hit rate ~%.2f, got %.2f", expectedHitRate, stats.HitRate)
	}
}

// TestAuthCache_Expiration tests that entries expire after TTL
func TestAuthCache_Expiration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping expiration test in short mode")
	}

	// Use very short TTL for testing
	cache := NewAuthCache(100 * time.Millisecond)
	validatorCalls := atomic.Int32{}

	validator := func(ctx context.Context) (*AuthResult, error) {
		validatorCalls.Add(1)
		return &AuthResult{Username: "testuser"}, nil
	}

	// Cache the result
	_, err := cache.Get(context.Background(), "test-pat", validator)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if validatorCalls.Load() != 1 {
		t.Errorf("expected 1 validator call, got %d", validatorCalls.Load())
	}

	// Wait for TTL to expire
	time.Sleep(200 * time.Millisecond)

	// Next call should call validator again (cache expired)
	_, err = cache.Get(context.Background(), "test-pat", validator)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if validatorCalls.Load() != 2 {
		t.Errorf("expected 2 validator calls after expiration, got %d", validatorCalls.Load())
	}
}

// TestAuthCache_PATHashing tests that PATs are hashed, not stored directly
func TestAuthCache_PATHashing(t *testing.T) {
	cache := NewAuthCache(5 * time.Minute)

	pat := "ghp_secretPAT123"
	hash1 := cache.hashPAT(pat)
	hash2 := cache.hashPAT(pat)

	// Same PAT should produce same hash
	if hash1 != hash2 {
		t.Error("same PAT produced different hashes")
	}

	// Hash should not contain the original PAT
	if hash1 == pat {
		t.Error("PAT was not hashed")
	}

	// Different PATs should produce different hashes
	differentPAT := "ghp_differentPAT456"
	hash3 := cache.hashPAT(differentPAT)
	if hash1 == hash3 {
		t.Error("different PATs produced same hash")
	}

	// Hash should be hex-encoded SHA256 (64 characters)
	if len(hash1) != 64 {
		t.Errorf("expected hash length 64, got %d", len(hash1))
	}
}
