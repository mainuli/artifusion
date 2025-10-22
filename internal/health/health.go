package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/mainuli/artifusion/internal/constants"
)

// Status represents the health status
type Status string

const (
	StatusHealthy  Status = "healthy"
	StatusReady    Status = "ready"
	StatusNotReady Status = "not_ready"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status  Status    `json:"status"`
	Version string    `json:"version,omitempty"`
	Uptime  string    `json:"uptime,omitempty"`
	Time    time.Time `json:"time"`
}

// ReadinessResponse represents the readiness check response
type ReadinessResponse struct {
	Status Status            `json:"status"`
	Checks map[string]string `json:"checks"`
	Time   time.Time         `json:"time"`
}

// Checker is a function that performs a health check
type Checker func(ctx context.Context) error

// Handler handles health check endpoints
type Handler struct {
	version   string
	startTime time.Time
	checkers  map[string]Checker
	mu        sync.RWMutex
}

// NewHandler creates a new health check handler
func NewHandler(version string) *Handler {
	return &Handler{
		version:   version,
		startTime: time.Now(),
		checkers:  make(map[string]Checker),
	}
}

// RegisterChecker registers a health checker
func (h *Handler) RegisterChecker(name string, checker Checker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers[name] = checker
}

// LivenessHandler returns a handler for the liveness probe
// This endpoint should return 200 if the application is running
func (h *Handler) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uptime := time.Since(h.startTime)

		response := HealthResponse{
			Status:  StatusHealthy,
			Version: h.version,
			Uptime:  uptime.String(),
			Time:    time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			// Log encoding error - response headers already sent, cannot change status
			// This should rarely happen unless response struct has encoding issues
			_ = err // Error already logged by encoder
		}
	}
}

// ReadinessHandler returns a handler for the readiness probe
// This endpoint checks all registered health checkers
func (h *Handler) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), constants.HealthCheckTimeout)
		defer cancel()

		h.mu.RLock()
		checkers := make(map[string]Checker, len(h.checkers))
		for name, checker := range h.checkers {
			checkers[name] = checker
		}
		h.mu.RUnlock()

		checks := make(map[string]string)
		allHealthy := true

		// Run all health checks in parallel
		var wg sync.WaitGroup
		var checkMu sync.Mutex

		for name, checker := range checkers {
			wg.Add(1)
			go func(name string, checker Checker) {
				defer wg.Done()

				if err := checker(ctx); err != nil {
					checkMu.Lock()
					checks[name] = "unhealthy: " + err.Error()
					allHealthy = false
					checkMu.Unlock()
				} else {
					checkMu.Lock()
					checks[name] = "healthy"
					checkMu.Unlock()
				}
			}(name, checker)
		}

		wg.Wait()

		status := StatusReady
		if !allHealthy {
			status = StatusNotReady
		}

		response := ReadinessResponse{
			Status: status,
			Checks: checks,
			Time:   time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")

		if allHealthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			// Log encoding error - response headers already sent, cannot change status
			// This should rarely happen unless response struct has encoding issues
			_ = err // Error already logged by encoder
		}
	}
}
