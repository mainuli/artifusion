package config

import (
	"fmt"
	"net/url"
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

	// Validate external_url if provided
	if s.ExternalURL != "" {
		parsedURL, err := url.Parse(s.ExternalURL)
		if err != nil {
			return fmt.Errorf("invalid external_url: %w", err)
		}
		// external_url must have a scheme and host
		if parsedURL.Scheme == "" {
			return fmt.Errorf("external_url must include scheme (http:// or https://)")
		}
		if parsedURL.Host == "" {
			return fmt.Errorf("external_url must include host")
		}
		// external_url should not have a path (to avoid confusion)
		if parsedURL.Path != "" && parsedURL.Path != "/" {
			return fmt.Errorf("external_url should not include path (found: %s)", parsedURL.Path)
		}
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
	if err := m.Backend.Validate(); err != nil {
		return fmt.Errorf("backend: %w", err)
	}

	return nil
}

// Validate validates NPM configuration
func (n *NPMConfig) Validate() error {
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

	return nil
}
