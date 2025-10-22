package metrics

import (
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// Request metrics
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	RequestSize     *prometheus.HistogramVec
	ResponseSize    *prometheus.HistogramVec
	ActiveRequests  prometheus.Gauge

	// Auth metrics
	AuthCacheHits   prometheus.Counter
	AuthCacheMisses prometheus.Counter
	AuthCacheSize   prometheus.Gauge
	GitHubAPICalls  *prometheus.CounterVec
	AuthDuration    *prometheus.HistogramVec

	// Backend metrics
	BackendRequests    *prometheus.CounterVec
	BackendDuration    *prometheus.HistogramVec
	BackendErrors      *prometheus.CounterVec
	BackendHealthGauge *prometheus.GaugeVec
	BackendLatency     *prometheus.HistogramVec
	BackendErrorRate   *prometheus.CounterVec
	ConnectionPoolSize *prometheus.GaugeVec

	// Rate limiting metrics
	RateLimitExceeded *prometheus.CounterVec

	// Circuit breaker metrics
	CircuitBreakerState *prometheus.GaugeVec

	// Internal tracking
	activeRequests atomic.Int32
}

// NewMetrics creates a new metrics collector
func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		// Request metrics
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "requests_total",
				Help:      "Total number of HTTP requests",
			},
			[]string{"protocol", "method", "status"},
		),

		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "request_duration_seconds",
				Help:      "Request duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"protocol", "method"},
		),

		RequestSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "request_size_bytes",
				Help:      "Request size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"protocol"},
		),

		ResponseSize: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "response_size_bytes",
				Help:      "Response size in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"protocol"},
		),

		ActiveRequests: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "active_requests",
				Help:      "Number of active requests",
			},
		),

		// Auth metrics
		AuthCacheHits: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "auth_cache_hits_total",
				Help:      "Total number of auth cache hits",
			},
		),

		AuthCacheMisses: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "auth_cache_misses_total",
				Help:      "Total number of auth cache misses",
			},
		),

		AuthCacheSize: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "auth_cache_size",
				Help:      "Number of entries in auth cache",
			},
		),

		GitHubAPICalls: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "github_api_calls_total",
				Help:      "Total number of GitHub API calls",
			},
			[]string{"endpoint", "status"},
		),

		AuthDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "auth_duration_seconds",
				Help:      "Authentication duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"cache_hit"},
		),

		// Backend metrics
		BackendRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "backend_requests_total",
				Help:      "Total number of backend requests",
			},
			[]string{"protocol", "backend", "status"},
		),

		BackendDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "backend_duration_seconds",
				Help:      "Backend request duration in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"protocol", "backend"},
		),

		BackendErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "backend_errors_total",
				Help:      "Total number of backend errors",
			},
			[]string{"protocol", "backend", "error_type"},
		),

		BackendHealthGauge: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "backend_health",
				Help:      "Backend health status (1=healthy, 0=unhealthy)",
			},
			[]string{"backend"},
		),

		BackendLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "backend_latency_seconds",
				Help:      "Backend request latency in seconds",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30},
			},
			[]string{"backend", "method"},
		),

		BackendErrorRate: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "backend_error_rate_total",
				Help:      "Backend error rate by status code",
			},
			[]string{"backend", "status_code"},
		),

		ConnectionPoolSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "connection_pool_size",
				Help:      "Number of connections in the pool",
			},
			[]string{"backend", "state"}, // state: idle, active
		),

		// Rate limiting metrics
		RateLimitExceeded: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "rate_limit_exceeded_total",
				Help:      "Total number of rate limit rejections",
			},
			[]string{"limit_type"}, // "global" or "per_user"
		),

		// Circuit breaker metrics
		CircuitBreakerState: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "circuit_breaker_state",
				Help:      "Circuit breaker state (0=closed, 1=open, 2=half-open)",
			},
			[]string{"backend"},
		),
	}

	return m
}

// RequestStarted increments active requests counter
func (m *Metrics) RequestStarted() {
	m.activeRequests.Add(1)
	m.ActiveRequests.Set(float64(m.activeRequests.Load()))
}

// RequestCompleted decrements active requests counter
func (m *Metrics) RequestCompleted() {
	m.activeRequests.Add(-1)
	m.ActiveRequests.Set(float64(m.activeRequests.Load()))
}

// RecordRequest records a completed request
func (m *Metrics) RecordRequest(protocol, method string, statusCode int, duration time.Duration) {
	m.RequestsTotal.WithLabelValues(protocol, method, statusCodeToString(statusCode)).Inc()
	m.RequestDuration.WithLabelValues(protocol, method).Observe(duration.Seconds())
}

// RecordAuthCacheHit records an auth cache hit
func (m *Metrics) RecordAuthCacheHit() {
	m.AuthCacheHits.Inc()
}

// RecordAuthCacheMiss records an auth cache miss
func (m *Metrics) RecordAuthCacheMiss() {
	m.AuthCacheMisses.Inc()
}

// SetAuthCacheSize sets the auth cache size
func (m *Metrics) SetAuthCacheSize(size int) {
	m.AuthCacheSize.Set(float64(size))
}

// RecordGitHubAPICall records a GitHub API call
func (m *Metrics) RecordGitHubAPICall(endpoint string, statusCode int) {
	m.GitHubAPICalls.WithLabelValues(endpoint, statusCodeToString(statusCode)).Inc()
}

// RecordAuthDuration records authentication duration
func (m *Metrics) RecordAuthDuration(duration time.Duration, cacheHit bool) {
	cacheHitStr := "false"
	if cacheHit {
		cacheHitStr = "true"
	}
	m.AuthDuration.WithLabelValues(cacheHitStr).Observe(duration.Seconds())
}

// RecordBackendRequest records a backend request
func (m *Metrics) RecordBackendRequest(protocol, backend string, statusCode int, duration time.Duration) {
	m.BackendRequests.WithLabelValues(protocol, backend, statusCodeToString(statusCode)).Inc()
	m.BackendDuration.WithLabelValues(protocol, backend).Observe(duration.Seconds())
}

// RecordBackendError records a backend error
func (m *Metrics) RecordBackendError(protocol, backend, errorType string) {
	m.BackendErrors.WithLabelValues(protocol, backend, errorType).Inc()
}

// RecordRateLimitExceeded records a rate limit rejection
func (m *Metrics) RecordRateLimitExceeded(limitType string) {
	m.RateLimitExceeded.WithLabelValues(limitType).Inc()
}

// SetCircuitBreakerState sets the circuit breaker state
func (m *Metrics) SetCircuitBreakerState(backend string, state int) {
	m.CircuitBreakerState.WithLabelValues(backend).Set(float64(state))
}

// SetBackendHealth sets the backend health status
func (m *Metrics) SetBackendHealth(backend string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	m.BackendHealthGauge.WithLabelValues(backend).Set(value)
}

// RecordBackendLatency records backend request latency
func (m *Metrics) RecordBackendLatency(backend, method string, duration time.Duration) {
	m.BackendLatency.WithLabelValues(backend, method).Observe(duration.Seconds())
}

// RecordBackendErrorByStatus records backend errors by status code
func (m *Metrics) RecordBackendErrorByStatus(backend string, statusCode int) {
	m.BackendErrorRate.WithLabelValues(backend, statusCodeToString(statusCode)).Inc()
}

// SetConnectionPoolSize sets the connection pool size
func (m *Metrics) SetConnectionPoolSize(backend, state string, size int) {
	m.ConnectionPoolSize.WithLabelValues(backend, state).Set(float64(size))
}

// statusCodeToString converts status code to string
func statusCodeToString(code int) string {
	if code >= 200 && code < 300 {
		return "2xx"
	} else if code >= 300 && code < 400 {
		return "3xx"
	} else if code >= 400 && code < 500 {
		return "4xx"
	} else if code >= 500 && code < 600 {
		return "5xx"
	}
	return "unknown"
}
