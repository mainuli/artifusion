package config

import (
	"time"

	"github.com/mainuli/artifusion/internal/proxy"
)

// Config represents the complete application configuration
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	GitHub    GitHubConfig    `mapstructure:"github"`
	Protocols ProtocolsConfig `mapstructure:"protocols"`
	Logging   LoggingConfig   `mapstructure:"logging"`
	Metrics   MetricsConfig   `mapstructure:"metrics"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Port              int           `mapstructure:"port"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout   time.Duration `mapstructure:"shutdown_timeout"`
	MaxHeaderBytes    int           `mapstructure:"max_header_bytes"`
	ReadBufferSize    int           `mapstructure:"read_buffer_size"`
	WriteBufferSize   int           `mapstructure:"write_buffer_size"`
	MaxConcurrentReqs int           `mapstructure:"max_concurrent_requests"`
}

// GitHubConfig contains GitHub authentication configuration
type GitHubConfig struct {
	APIURL          string        `mapstructure:"api_url"`
	RequiredOrg     string        `mapstructure:"required_org"`
	RequiredTeams   []string      `mapstructure:"required_teams"`
	AuthCacheTTL    time.Duration `mapstructure:"auth_cache_ttl"`
	RateLimitBuffer int           `mapstructure:"rate_limit_buffer"`
}

// ProtocolsConfig contains configuration for all protocol handlers
type ProtocolsConfig struct {
	OCI   OCIConfig   `mapstructure:"oci"`
	Maven MavenConfig `mapstructure:"maven"`
	NPM   NPMConfig   `mapstructure:"npm"`
}

// OCIConfig contains OCI/Docker registry configuration
type OCIConfig struct {
	Enabled      bool               `mapstructure:"enabled"`
	Host         string             `mapstructure:"host"` // Optional: domain for host-based routing (e.g., "docker.example.com")
	ClientAuth   ClientAuthConfig   `mapstructure:"client_auth"`
	PullBackends []OCIBackendConfig `mapstructure:"pull_backends"`
	PushBackend  OCIBackendConfig   `mapstructure:"push_backend"`
}

// MavenConfig contains Maven repository configuration
type MavenConfig struct {
	Enabled    bool               `mapstructure:"enabled"`
	Host       string             `mapstructure:"host"`        // Optional: domain for host-based routing (e.g., "maven.example.com")
	PathPrefix string             `mapstructure:"path_prefix"` // URL path prefix - required when host is empty
	ClientAuth ClientAuthConfig   `mapstructure:"client_auth"`
	Backend    MavenBackendConfig `mapstructure:"backend"`
}

// NPMConfig contains NPM registry configuration
type NPMConfig struct {
	Enabled    bool             `mapstructure:"enabled"`
	Host       string           `mapstructure:"host"`        // Optional: domain for host-based routing (e.g., "npm.example.com")
	PathPrefix string           `mapstructure:"path_prefix"` // URL path prefix - required when host is empty
	ClientAuth ClientAuthConfig `mapstructure:"client_auth"`
	Backend    NPMBackendConfig `mapstructure:"backend"`
}

// ClientAuthConfig contains client authentication configuration
type ClientAuthConfig struct {
	SupportedSchemes []string `mapstructure:"supported_schemes"`
	Realm            string   `mapstructure:"realm"`
	Service          string   `mapstructure:"service"`
}

// OCIBackendConfig contains OCI/Docker registry backend configuration
type OCIBackendConfig struct {
	// Common fields
	Name string      `mapstructure:"name"`
	URL  string      `mapstructure:"url"`
	Auth *AuthConfig `mapstructure:"auth"`

	// OCI-specific fields
	UpstreamNamespace string            `mapstructure:"upstream_namespace"` // e.g., "ghcr.io", "docker.io"
	PathRewrite       PathRewriteConfig `mapstructure:"path_rewrite"`

	// Scope defines which organizations should use this backend (for org-based routing)
	// If empty, falls back to requiredOrg from GitHub auth config
	// Use "*" as wildcard to allow all organizations
	// Examples: ["myorg", "anotherorg"], ["*"]
	Scope []string `mapstructure:"scope"`

	// HTTP client pool settings
	MaxIdleConns        int           `mapstructure:"max_idle_conns"`
	MaxIdleConnsPerHost int           `mapstructure:"max_idle_conns_per_host"`
	IdleConnTimeout     time.Duration `mapstructure:"idle_conn_timeout"`
	DialTimeout         time.Duration `mapstructure:"dial_timeout"`
	RequestTimeout      time.Duration `mapstructure:"request_timeout"`

	// Circuit breaker settings
	CircuitBreaker CircuitBreakerConfig `mapstructure:"circuit_breaker"`
}

// Interface implementation for proxy.BackendConfig
func (o *OCIBackendConfig) GetName() string                   { return o.Name }
func (o *OCIBackendConfig) GetURL() string                    { return o.URL }
func (o *OCIBackendConfig) GetMaxIdleConns() int              { return o.MaxIdleConns }
func (o *OCIBackendConfig) GetMaxIdleConnsPerHost() int       { return o.MaxIdleConnsPerHost }
func (o *OCIBackendConfig) GetIdleConnTimeout() time.Duration { return o.IdleConnTimeout }
func (o *OCIBackendConfig) GetDialTimeout() time.Duration     { return o.DialTimeout }
func (o *OCIBackendConfig) GetRequestTimeout() time.Duration  { return o.RequestTimeout }
func (o *OCIBackendConfig) GetCircuitBreaker() *proxy.CircuitBreakerConfig {
	return &proxy.CircuitBreakerConfig{
		Enabled:          o.CircuitBreaker.Enabled,
		MaxRequests:      o.CircuitBreaker.MaxRequests,
		Interval:         o.CircuitBreaker.Interval,
		Timeout:          o.CircuitBreaker.Timeout,
		FailureThreshold: o.CircuitBreaker.FailureThreshold,
	}
}

// MavenBackendConfig contains Maven repository backend configuration
type MavenBackendConfig struct {
	// Common fields
	Name string      `mapstructure:"name"`
	URL  string      `mapstructure:"url"`
	Auth *AuthConfig `mapstructure:"auth"`

	// HTTP client pool settings
	MaxIdleConns        int           `mapstructure:"max_idle_conns"`
	MaxIdleConnsPerHost int           `mapstructure:"max_idle_conns_per_host"`
	IdleConnTimeout     time.Duration `mapstructure:"idle_conn_timeout"`
	DialTimeout         time.Duration `mapstructure:"dial_timeout"`
	RequestTimeout      time.Duration `mapstructure:"request_timeout"`

	// Circuit breaker settings
	CircuitBreaker CircuitBreakerConfig `mapstructure:"circuit_breaker"`
}

// Interface implementation for proxy.BackendConfig
func (m *MavenBackendConfig) GetName() string                   { return m.Name }
func (m *MavenBackendConfig) GetURL() string                    { return m.URL }
func (m *MavenBackendConfig) GetMaxIdleConns() int              { return m.MaxIdleConns }
func (m *MavenBackendConfig) GetMaxIdleConnsPerHost() int       { return m.MaxIdleConnsPerHost }
func (m *MavenBackendConfig) GetIdleConnTimeout() time.Duration { return m.IdleConnTimeout }
func (m *MavenBackendConfig) GetDialTimeout() time.Duration     { return m.DialTimeout }
func (m *MavenBackendConfig) GetRequestTimeout() time.Duration  { return m.RequestTimeout }
func (m *MavenBackendConfig) GetCircuitBreaker() *proxy.CircuitBreakerConfig {
	return &proxy.CircuitBreakerConfig{
		Enabled:          m.CircuitBreaker.Enabled,
		MaxRequests:      m.CircuitBreaker.MaxRequests,
		Interval:         m.CircuitBreaker.Interval,
		Timeout:          m.CircuitBreaker.Timeout,
		FailureThreshold: m.CircuitBreaker.FailureThreshold,
	}
}

// NPMBackendConfig contains NPM registry backend configuration
type NPMBackendConfig struct {
	// Common fields
	Name string      `mapstructure:"name"`
	URL  string      `mapstructure:"url"`
	Auth *AuthConfig `mapstructure:"auth"` // Supports bearer tokens (preemptive)

	// HTTP client pool settings
	MaxIdleConns        int           `mapstructure:"max_idle_conns"`
	MaxIdleConnsPerHost int           `mapstructure:"max_idle_conns_per_host"`
	IdleConnTimeout     time.Duration `mapstructure:"idle_conn_timeout"`
	DialTimeout         time.Duration `mapstructure:"dial_timeout"`
	RequestTimeout      time.Duration `mapstructure:"request_timeout"`

	// Circuit breaker settings
	CircuitBreaker CircuitBreakerConfig `mapstructure:"circuit_breaker"`
}

// Interface implementation for proxy.BackendConfig
func (n *NPMBackendConfig) GetName() string                   { return n.Name }
func (n *NPMBackendConfig) GetURL() string                    { return n.URL }
func (n *NPMBackendConfig) GetMaxIdleConns() int              { return n.MaxIdleConns }
func (n *NPMBackendConfig) GetMaxIdleConnsPerHost() int       { return n.MaxIdleConnsPerHost }
func (n *NPMBackendConfig) GetIdleConnTimeout() time.Duration { return n.IdleConnTimeout }
func (n *NPMBackendConfig) GetDialTimeout() time.Duration     { return n.DialTimeout }
func (n *NPMBackendConfig) GetRequestTimeout() time.Duration  { return n.RequestTimeout }
func (n *NPMBackendConfig) GetCircuitBreaker() *proxy.CircuitBreakerConfig {
	return &proxy.CircuitBreakerConfig{
		Enabled:          n.CircuitBreaker.Enabled,
		MaxRequests:      n.CircuitBreaker.MaxRequests,
		Interval:         n.CircuitBreaker.Interval,
		Timeout:          n.CircuitBreaker.Timeout,
		FailureThreshold: n.CircuitBreaker.FailureThreshold,
	}
}

// PathRewriteConfig contains path rewriting rules
type PathRewriteConfig struct {
	AddLibraryPrefix bool `mapstructure:"add_library_prefix"`
}

// AuthConfig contains backend authentication configuration
type AuthConfig struct {
	Type        string `mapstructure:"type"`
	Username    string `mapstructure:"username"`
	Password    string `mapstructure:"password"`
	Token       string `mapstructure:"token"`
	HeaderName  string `mapstructure:"header_name"`
	HeaderValue string `mapstructure:"header_value"`
}

// CircuitBreakerConfig contains circuit breaker settings
type CircuitBreakerConfig struct {
	Enabled          bool          `mapstructure:"enabled"`
	MaxRequests      uint32        `mapstructure:"max_requests"`
	Interval         time.Duration `mapstructure:"interval"`
	Timeout          time.Duration `mapstructure:"timeout"`
	FailureThreshold float64       `mapstructure:"failure_threshold"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level          string `mapstructure:"level"`
	Format         string `mapstructure:"format"`
	IncludeHeaders bool   `mapstructure:"include_headers"`
	IncludeBody    bool   `mapstructure:"include_body"`
}

// MetricsConfig contains Prometheus metrics configuration
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

// RateLimitConfig contains rate limiting configuration
type RateLimitConfig struct {
	Enabled         bool    `mapstructure:"enabled"`
	RequestsPerSec  float64 `mapstructure:"requests_per_sec"`
	Burst           int     `mapstructure:"burst"`
	PerUserEnabled  bool    `mapstructure:"per_user_enabled"`
	PerUserRequests float64 `mapstructure:"per_user_requests"`
	PerUserBurst    int     `mapstructure:"per_user_burst"`
}

// Default values
const (
	DefaultServerPort        = 8080
	DefaultReadTimeout       = 60 * time.Second
	DefaultWriteTimeout      = 300 * time.Second
	DefaultIdleTimeout       = 120 * time.Second
	DefaultShutdownTimeout   = 30 * time.Second
	DefaultMaxHeaderBytes    = 1 << 20   // 1 MB
	DefaultReadBufferSize    = 32 * 1024 // 32 KB
	DefaultWriteBufferSize   = 32 * 1024 // 32 KB
	DefaultMaxConcurrentReqs = 10000

	DefaultAuthCacheTTL    = 30 * time.Minute
	DefaultRateLimitBuffer = 100

	DefaultMaxIdleConns        = 200
	DefaultMaxIdleConnsPerHost = 100
	DefaultIdleConnTimeout     = 90 * time.Second
	DefaultDialTimeout         = 10 * time.Second
	DefaultRequestTimeout      = 300 * time.Second

	DefaultCircuitBreakerMaxRequests      = 10
	DefaultCircuitBreakerInterval         = 60 * time.Second
	DefaultCircuitBreakerTimeout          = 30 * time.Second
	DefaultCircuitBreakerFailureThreshold = 0.5

	DefaultRateLimitRequestsPerSec = 1000.0
	DefaultRateLimitBurst          = 2000
	DefaultPerUserRequests         = 100.0
	DefaultPerUserBurst            = 200
)

// SetDefaults sets default values for missing configuration
func (c *Config) SetDefaults() {
	// Server defaults
	if c.Server.Port == 0 {
		c.Server.Port = DefaultServerPort
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = DefaultReadTimeout
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = DefaultWriteTimeout
	}
	if c.Server.IdleTimeout == 0 {
		c.Server.IdleTimeout = DefaultIdleTimeout
	}
	if c.Server.ShutdownTimeout == 0 {
		c.Server.ShutdownTimeout = DefaultShutdownTimeout
	}
	if c.Server.MaxHeaderBytes == 0 {
		c.Server.MaxHeaderBytes = DefaultMaxHeaderBytes
	}
	if c.Server.ReadBufferSize == 0 {
		c.Server.ReadBufferSize = DefaultReadBufferSize
	}
	if c.Server.WriteBufferSize == 0 {
		c.Server.WriteBufferSize = DefaultWriteBufferSize
	}
	if c.Server.MaxConcurrentReqs == 0 {
		c.Server.MaxConcurrentReqs = DefaultMaxConcurrentReqs
	}

	// GitHub defaults
	if c.GitHub.APIURL == "" {
		c.GitHub.APIURL = "https://api.github.com"
	}
	if c.GitHub.AuthCacheTTL == 0 {
		c.GitHub.AuthCacheTTL = DefaultAuthCacheTTL
	}
	if c.GitHub.RateLimitBuffer == 0 {
		c.GitHub.RateLimitBuffer = DefaultRateLimitBuffer
	}

	// Rate limit defaults
	if c.RateLimit.Enabled && c.RateLimit.RequestsPerSec == 0 {
		c.RateLimit.RequestsPerSec = DefaultRateLimitRequestsPerSec
		c.RateLimit.Burst = DefaultRateLimitBurst
	}
	if c.RateLimit.PerUserEnabled && c.RateLimit.PerUserRequests == 0 {
		c.RateLimit.PerUserRequests = DefaultPerUserRequests
		c.RateLimit.PerUserBurst = DefaultPerUserBurst
	}

	// Protocol-specific backend defaults
	for i := range c.Protocols.OCI.PullBackends {
		c.setOCIBackendDefaults(&c.Protocols.OCI.PullBackends[i])
	}
	c.setOCIBackendDefaults(&c.Protocols.OCI.PushBackend)
	c.setMavenBackendDefaults(&c.Protocols.Maven.Backend)
	c.setNPMBackendDefaults(&c.Protocols.NPM.Backend)

	// Maven path prefix default
	if c.Protocols.Maven.PathPrefix == "" {
		c.Protocols.Maven.PathPrefix = "/maven"
	}

	// NPM path prefix default
	if c.Protocols.NPM.PathPrefix == "" {
		c.Protocols.NPM.PathPrefix = "/npm"
	}

	// Logging defaults
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "console"
	}

	// Metrics defaults
	if c.Metrics.Path == "" {
		c.Metrics.Path = "/metrics"
	}
}

// backendDefaults is an interface for backend configs that need default values
type backendDefaults interface {
	getConnectionSettings() *backendConnectionSettings
	getCircuitBreaker() *CircuitBreakerConfig
}

// backendConnectionSettings holds pointers to connection-related fields
// This allows us to set defaults without knowing the concrete backend type
type backendConnectionSettings struct {
	MaxIdleConns        *int
	MaxIdleConnsPerHost *int
	IdleConnTimeout     *time.Duration
	DialTimeout         *time.Duration
	RequestTimeout      *time.Duration
}

// getConnectionSettings returns pointers to OCIBackendConfig connection fields
func (o *OCIBackendConfig) getConnectionSettings() *backendConnectionSettings {
	return &backendConnectionSettings{
		MaxIdleConns:        &o.MaxIdleConns,
		MaxIdleConnsPerHost: &o.MaxIdleConnsPerHost,
		IdleConnTimeout:     &o.IdleConnTimeout,
		DialTimeout:         &o.DialTimeout,
		RequestTimeout:      &o.RequestTimeout,
	}
}

// getCircuitBreaker returns pointer to OCIBackendConfig circuit breaker
func (o *OCIBackendConfig) getCircuitBreaker() *CircuitBreakerConfig {
	return &o.CircuitBreaker
}

// getConnectionSettings returns pointers to MavenBackendConfig connection fields
func (m *MavenBackendConfig) getConnectionSettings() *backendConnectionSettings {
	return &backendConnectionSettings{
		MaxIdleConns:        &m.MaxIdleConns,
		MaxIdleConnsPerHost: &m.MaxIdleConnsPerHost,
		IdleConnTimeout:     &m.IdleConnTimeout,
		DialTimeout:         &m.DialTimeout,
		RequestTimeout:      &m.RequestTimeout,
	}
}

// getCircuitBreaker returns pointer to MavenBackendConfig circuit breaker
func (m *MavenBackendConfig) getCircuitBreaker() *CircuitBreakerConfig {
	return &m.CircuitBreaker
}

// getConnectionSettings returns pointers to NPMBackendConfig connection fields
func (n *NPMBackendConfig) getConnectionSettings() *backendConnectionSettings {
	return &backendConnectionSettings{
		MaxIdleConns:        &n.MaxIdleConns,
		MaxIdleConnsPerHost: &n.MaxIdleConnsPerHost,
		IdleConnTimeout:     &n.IdleConnTimeout,
		DialTimeout:         &n.DialTimeout,
		RequestTimeout:      &n.RequestTimeout,
	}
}

// getCircuitBreaker returns pointer to NPMBackendConfig circuit breaker
func (n *NPMBackendConfig) getCircuitBreaker() *CircuitBreakerConfig {
	return &n.CircuitBreaker
}

// setBackendDefaultsCommon sets default values for any backend configuration
// This eliminates code duplication across protocol-specific backend defaults
func (c *Config) setBackendDefaultsCommon(backend backendDefaults) {
	settings := backend.getConnectionSettings()

	// Connection pool defaults
	if *settings.MaxIdleConns == 0 {
		*settings.MaxIdleConns = DefaultMaxIdleConns
	}
	if *settings.MaxIdleConnsPerHost == 0 {
		*settings.MaxIdleConnsPerHost = DefaultMaxIdleConnsPerHost
	}
	if *settings.IdleConnTimeout == 0 {
		*settings.IdleConnTimeout = DefaultIdleConnTimeout
	}
	if *settings.DialTimeout == 0 {
		*settings.DialTimeout = DefaultDialTimeout
	}
	if *settings.RequestTimeout == 0 {
		*settings.RequestTimeout = DefaultRequestTimeout
	}

	// Circuit breaker defaults
	cb := backend.getCircuitBreaker()
	if cb.Enabled {
		if cb.MaxRequests == 0 {
			cb.MaxRequests = DefaultCircuitBreakerMaxRequests
		}
		if cb.Interval == 0 {
			cb.Interval = DefaultCircuitBreakerInterval
		}
		if cb.Timeout == 0 {
			cb.Timeout = DefaultCircuitBreakerTimeout
		}
		if cb.FailureThreshold == 0 {
			cb.FailureThreshold = DefaultCircuitBreakerFailureThreshold
		}
	}
}

// setOCIBackendDefaults sets default values for OCI backend configuration
func (c *Config) setOCIBackendDefaults(backend *OCIBackendConfig) {
	c.setBackendDefaultsCommon(backend)
}

// setMavenBackendDefaults sets default values for Maven backend configuration
func (c *Config) setMavenBackendDefaults(backend *MavenBackendConfig) {
	c.setBackendDefaultsCommon(backend)
}

// setNPMBackendDefaults sets default values for NPM backend configuration
func (c *Config) setNPMBackendDefaults(backend *NPMBackendConfig) {
	c.setBackendDefaultsCommon(backend)
}
