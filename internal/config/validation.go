package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server config
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server config: %w", err)
	}

	// Validate GitHub config
	if err := c.GitHub.Validate(); err != nil {
		return fmt.Errorf("github config: %w", err)
	}

	// Validate protocols
	if err := c.Protocols.Validate(); err != nil {
		return fmt.Errorf("protocols config: %w", err)
	}

	// Validate logging
	if err := c.Logging.Validate(); err != nil {
		return fmt.Errorf("logging config: %w", err)
	}

	// At least one protocol must be enabled
	if !c.Protocols.OCI.Enabled && !c.Protocols.Maven.Enabled && !c.Protocols.NPM.Enabled {
		return fmt.Errorf("at least one protocol must be enabled")
	}

	return nil
}

// Validate validates server configuration
func (s *ServerConfig) Validate() error {
	if s.Port < 1 || s.Port > 65535 {
		return fmt.Errorf("invalid port: %d", s.Port)
	}

	if s.ReadTimeout <= 0 {
		return fmt.Errorf("invalid read timeout: %v", s.ReadTimeout)
	}

	if s.WriteTimeout <= 0 {
		return fmt.Errorf("invalid write timeout: %v", s.WriteTimeout)
	}

	if s.MaxConcurrentReqs < 1 {
		return fmt.Errorf("maxConcurrentRequests must be at least 1")
	}

	return nil
}

// Validate validates GitHub configuration
func (g *GitHubConfig) Validate() error {
	if g.APIURL == "" {
		return fmt.Errorf("apiURL is required")
	}

	if _, err := url.Parse(g.APIURL); err != nil {
		return fmt.Errorf("invalid apiURL: %w", err)
	}

	// RequiredOrg is optional - if empty, only PAT validation is performed
	// If provided, organization membership will be checked

	// SECURITY: Prevent team enforcement bypass
	// If teams are required, org must also be specified since team checks
	// only run inside the org membership validation block
	if len(g.RequiredTeams) > 0 && g.RequiredOrg == "" {
		return fmt.Errorf("required_org must be specified when required_teams is configured")
	}

	if g.AuthCacheTTL <= 0 {
		return fmt.Errorf("invalid authCacheTTL: %v", g.AuthCacheTTL)
	}

	return nil
}

// Validate validates protocols configuration
func (p *ProtocolsConfig) Validate() error {
	if p.OCI.Enabled {
		if err := p.OCI.Validate(); err != nil {
			return fmt.Errorf("oci config: %w", err)
		}
	}

	if p.Maven.Enabled {
		if err := p.Maven.Validate(); err != nil {
			return fmt.Errorf("maven config: %w", err)
		}
	}

	if p.NPM.Enabled {
		if err := p.NPM.Validate(); err != nil {
			return fmt.Errorf("npm config: %w", err)
		}
	}

	// SECURITY: Validate path_prefix uniqueness for protocols with empty host
	// This prevents routing conflicts where multiple protocols could match the same request
	pathPrefixes := make(map[string]string) // map[path_prefix]protocol_name

	if p.Maven.Enabled && p.Maven.Host == "" && p.Maven.PathPrefix != "" {
		pathPrefixes[p.Maven.PathPrefix] = "maven"
	}

	if p.NPM.Enabled && p.NPM.Host == "" && p.NPM.PathPrefix != "" {
		if existing, exists := pathPrefixes[p.NPM.PathPrefix]; exists {
			return fmt.Errorf("path_prefix conflict: both %s and npm use path_prefix '%s' with empty host", existing, p.NPM.PathPrefix)
		}
		pathPrefixes[p.NPM.PathPrefix] = "npm"
	}

	// Note: OCI always uses /v2 path prefix, but this is implicitly unique
	// since it's hardcoded in the detector and not configurable

	return nil
}

// Validate validates OCI configuration
func (o *OCIConfig) Validate() error {
	if len(o.PullBackends) == 0 {
		return fmt.Errorf("at least one pull backend is required")
	}

	for i, backend := range o.PullBackends {
		if err := backend.Validate(); err != nil {
			return fmt.Errorf("pull backend %d: %w", i, err)
		}
	}

	if err := o.PushBackend.Validate(); err != nil {
		return fmt.Errorf("push backend: %w", err)
	}

	return nil
}

// Validate validates Maven configuration
func (m *MavenConfig) Validate() error {
	// SECURITY: Prevent routing conflicts - require explicit path_prefix when host is not set
	if m.Host == "" && m.PathPrefix == "" {
		return fmt.Errorf("path_prefix is required when host is empty (set either host for domain-based routing or path_prefix for path-based routing)")
	}

	// Validate path_prefix format
	if m.PathPrefix != "" {
		if !strings.HasPrefix(m.PathPrefix, "/") {
			return fmt.Errorf("path_prefix must start with '/' (got: %s)", m.PathPrefix)
		}
	}

	if err := m.Backend.Validate(); err != nil {
		return fmt.Errorf("backend: %w", err)
	}

	return nil
}

// Validate validates NPM configuration
func (n *NPMConfig) Validate() error {
	// SECURITY: Prevent routing conflicts - require explicit path_prefix when host is not set
	if n.Host == "" && n.PathPrefix == "" {
		return fmt.Errorf("path_prefix is required when host is empty (set either host for domain-based routing or path_prefix for path-based routing)")
	}

	// Validate path_prefix format
	if n.PathPrefix != "" {
		if !strings.HasPrefix(n.PathPrefix, "/") {
			return fmt.Errorf("path_prefix must start with '/' (got: %s)", n.PathPrefix)
		}
	}

	if err := n.Backend.Validate(); err != nil {
		return fmt.Errorf("backend: %w", err)
	}

	return nil
}

// validateBackendCommon validates common backend configuration fields
// This is a helper to eliminate code duplication across protocol-specific backend validators
func validateBackendCommon(backendURL string, maxIdleConns, maxIdleConnsPerHost int, dialTimeout, requestTimeout time.Duration, circuitBreaker CircuitBreakerConfig) error {
	// Validate URL
	if backendURL == "" {
		return fmt.Errorf("url is required")
	}

	if _, err := url.Parse(backendURL); err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}

	// Validate connection pool settings
	if maxIdleConns < 1 {
		return fmt.Errorf("maxIdleConns must be at least 1")
	}

	if maxIdleConnsPerHost < 1 {
		return fmt.Errorf("maxIdleConnsPerHost must be at least 1")
	}

	if maxIdleConnsPerHost > maxIdleConns {
		return fmt.Errorf("maxIdleConnsPerHost cannot exceed maxIdleConns")
	}

	// Validate timeouts
	if dialTimeout <= 0 {
		return fmt.Errorf("invalid dialTimeout: %v", dialTimeout)
	}

	if requestTimeout <= 0 {
		return fmt.Errorf("invalid requestTimeout: %v", requestTimeout)
	}

	// Validate circuit breaker settings
	if circuitBreaker.Enabled {
		if err := circuitBreaker.Validate(); err != nil {
			return fmt.Errorf("circuit breaker: %w", err)
		}
	}

	return nil
}

// Validate validates OCI backend configuration
func (b *OCIBackendConfig) Validate() error {
	return validateBackendCommon(
		b.URL,
		b.MaxIdleConns,
		b.MaxIdleConnsPerHost,
		b.DialTimeout,
		b.RequestTimeout,
		b.CircuitBreaker,
	)
}

// Validate validates Maven backend configuration
func (b *MavenBackendConfig) Validate() error {
	return validateBackendCommon(
		b.URL,
		b.MaxIdleConns,
		b.MaxIdleConnsPerHost,
		b.DialTimeout,
		b.RequestTimeout,
		b.CircuitBreaker,
	)
}

// Validate validates NPM backend configuration
func (b *NPMBackendConfig) Validate() error {
	return validateBackendCommon(
		b.URL,
		b.MaxIdleConns,
		b.MaxIdleConnsPerHost,
		b.DialTimeout,
		b.RequestTimeout,
		b.CircuitBreaker,
	)
}

// Validate validates circuit breaker configuration
func (cb *CircuitBreakerConfig) Validate() error {
	if cb.MaxRequests < 1 {
		return fmt.Errorf("maxRequests must be at least 1")
	}

	if cb.Interval <= 0 {
		return fmt.Errorf("invalid interval: %v", cb.Interval)
	}

	if cb.Timeout <= 0 {
		return fmt.Errorf("invalid timeout: %v", cb.Timeout)
	}

	if cb.FailureThreshold <= 0 || cb.FailureThreshold > 1 {
		return fmt.Errorf("failureThreshold must be between 0 and 1")
	}

	return nil
}

// Validate validates logging configuration
func (l *LoggingConfig) Validate() error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLevels[l.Level] {
		return fmt.Errorf("invalid level: %s (must be debug, info, warn, or error)", l.Level)
	}

	validFormats := map[string]bool{
		"json":    true,
		"console": true,
	}

	if !validFormats[l.Format] {
		return fmt.Errorf("invalid format: %s (must be json or console)", l.Format)
	}

	// NOTE: IncludeHeaders should only be used for debugging/troubleshooting
	//
	// While sensitive headers (Authorization, Cookie, etc.) are automatically redacted
	// by the logging middleware, enabling header logging still has implications:
	//
	// 1. Performance: Increases log volume and processing overhead
	// 2. Storage: Significantly larger log files
	// 3. Privacy: Other headers may contain PII (User-Agent, X-Forwarded-For, Referer)
	// 4. Compliance: May require additional data handling considerations
	//
	// A warning will be logged at startup if this is enabled.
	// This is intentionally allowed to support debugging scenarios.

	return nil
}
