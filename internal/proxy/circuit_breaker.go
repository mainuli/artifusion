package proxy

import (
	"fmt"
	"sync"

	"github.com/mainuli/artifusion/internal/metrics"
	"github.com/rs/zerolog"
	"github.com/sony/gobreaker"
)

// CircuitBreakerManager manages circuit breakers for multiple backends
type CircuitBreakerManager struct {
	breakers map[string]*gobreaker.CircuitBreaker
	mu       sync.RWMutex
	logger   zerolog.Logger
	metrics  *metrics.Metrics
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(logger zerolog.Logger, metrics *metrics.Metrics) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*gobreaker.CircuitBreaker),
		logger:   logger.With().Str("component", "circuit_breaker").Logger(),
		metrics:  metrics,
	}
}

// GetOrCreate gets or creates a circuit breaker for a backend
func (cbm *CircuitBreakerManager) GetOrCreate(backend BackendConfig) *gobreaker.CircuitBreaker {
	cbConfig := backend.GetCircuitBreaker()
	if cbConfig == nil || !cbConfig.Enabled {
		return nil
	}

	backendName := backend.GetName()

	// Fast path with read lock
	cbm.mu.RLock()
	cb, exists := cbm.breakers[backendName]
	cbm.mu.RUnlock()

	if exists {
		return cb
	}

	// Slow path with write lock
	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, exists := cbm.breakers[backendName]; exists {
		return cb
	}

	// Create new circuit breaker with backend-specific settings
	settings := gobreaker.Settings{
		Name:        backendName,
		MaxRequests: cbConfig.MaxRequests,
		Interval:    cbConfig.Interval,
		Timeout:     cbConfig.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Open circuit if failure rate exceeds threshold
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && failureRatio >= cbConfig.FailureThreshold
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			// Log circuit breaker state changes for observability
			cbm.logger.Warn().
				Str("backend", name).
				Str("from_state", from.String()).
				Str("to_state", to.String()).
				Msg("Circuit breaker state changed")

			// Emit metrics for monitoring and alerting
			if cbm.metrics != nil {
				cbm.metrics.SetCircuitBreakerState(name, StateToInt(to))
			}
		},
	}

	cb = gobreaker.NewCircuitBreaker(settings)
	cbm.breakers[backendName] = cb

	return cb
}

// Execute executes a function with circuit breaker protection
func (cbm *CircuitBreakerManager) Execute(backend BackendConfig, fn func() (interface{}, error)) (interface{}, error) {
	cb := cbm.GetOrCreate(backend)

	// If circuit breaker is disabled or doesn't exist, execute directly
	if cb == nil {
		return fn()
	}

	// Execute with circuit breaker protection
	result, err := cb.Execute(func() (interface{}, error) {
		return fn()
	})

	// Handle circuit breaker specific errors
	if err == gobreaker.ErrOpenState {
		return nil, fmt.Errorf("circuit breaker open for backend %s: %w", backend.GetName(), err)
	}

	if err == gobreaker.ErrTooManyRequests {
		return nil, fmt.Errorf("too many requests to backend %s (half-open state): %w", backend.GetName(), err)
	}

	return result, err
}

// GetState returns the current state of a circuit breaker
func (cbm *CircuitBreakerManager) GetState(backendName string) gobreaker.State {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()

	if cb, exists := cbm.breakers[backendName]; exists {
		return cb.State()
	}

	return gobreaker.StateClosed
}

// GetCounts returns the current counts of a circuit breaker
func (cbm *CircuitBreakerManager) GetCounts(backendName string) gobreaker.Counts {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()

	if cb, exists := cbm.breakers[backendName]; exists {
		return cb.Counts()
	}

	return gobreaker.Counts{}
}

// Reset resets a circuit breaker to closed state by recreating it
// This is useful for manual recovery or testing scenarios
func (cbm *CircuitBreakerManager) Reset(backendName string) {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	// Delete existing breaker if it exists
	if _, exists := cbm.breakers[backendName]; exists {
		delete(cbm.breakers, backendName)
		cbm.logger.Info().
			Str("backend", backendName).
			Msg("Circuit breaker reset - removed existing breaker")
	}

	// The breaker will be recreated on next GetOrCreate call with fresh state
	// Note: gobreaker doesn't expose a public Reset method, so we delete and recreate
}

// StateToInt converts circuit breaker state to integer for metrics
// 0 = closed, 1 = open, 2 = half-open
func StateToInt(state gobreaker.State) int {
	switch state {
	case gobreaker.StateClosed:
		return 0
	case gobreaker.StateOpen:
		return 1
	case gobreaker.StateHalfOpen:
		return 2
	default:
		return -1
	}
}
