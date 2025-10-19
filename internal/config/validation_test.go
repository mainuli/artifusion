package config

import (
	"strings"
	"testing"
	"time"
)

// TestConfig_Validate_Valid tests validation of a valid configuration
func TestConfig_Validate_Valid(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Port:              8080,
			ReadTimeout:       60 * time.Second,
			WriteTimeout:      300 * time.Second,
			MaxConcurrentReqs: 1000,
		},
		GitHub: GitHubConfig{
			APIURL:       "https://api.github.com",
			RequiredOrg:  "myorg",
			AuthCacheTTL: 30 * time.Minute,
		},
		Protocols: ProtocolsConfig{
			OCI: OCIConfig{
				Enabled: true,
				PullBackends: []OCIBackendConfig{
					{
						Name:                "ghcr",
						URL:                 "https://ghcr.io",
						MaxIdleConns:        200,
						MaxIdleConnsPerHost: 100,
						DialTimeout:         10 * time.Second,
						RequestTimeout:      300 * time.Second,
					},
				},
				PushBackend: OCIBackendConfig{
					Name:                "local",
					URL:                 "http://registry:5000",
					MaxIdleConns:        200,
					MaxIdleConnsPerHost: 100,
					DialTimeout:         10 * time.Second,
					RequestTimeout:      300 * time.Second,
				},
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("valid config should not return error: %v", err)
	}
}

// TestServerConfig_Validate tests server configuration validation
func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  ServerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: ServerConfig{
				Port:              8080,
				ReadTimeout:       60 * time.Second,
				WriteTimeout:      300 * time.Second,
				MaxConcurrentReqs: 1000,
			},
			wantErr: false,
		},
		{
			name: "invalid port too low",
			config: ServerConfig{
				Port:              0,
				ReadTimeout:       60 * time.Second,
				WriteTimeout:      300 * time.Second,
				MaxConcurrentReqs: 1000,
			},
			wantErr: true,
			errMsg:  "invalid port",
		},
		{
			name: "invalid port too high",
			config: ServerConfig{
				Port:              70000,
				ReadTimeout:       60 * time.Second,
				WriteTimeout:      300 * time.Second,
				MaxConcurrentReqs: 1000,
			},
			wantErr: true,
			errMsg:  "invalid port",
		},
		{
			name: "invalid read timeout",
			config: ServerConfig{
				Port:              8080,
				ReadTimeout:       0,
				WriteTimeout:      300 * time.Second,
				MaxConcurrentReqs: 1000,
			},
			wantErr: true,
			errMsg:  "invalid read timeout",
		},
		{
			name: "invalid write timeout",
			config: ServerConfig{
				Port:              8080,
				ReadTimeout:       60 * time.Second,
				WriteTimeout:      0,
				MaxConcurrentReqs: 1000,
			},
			wantErr: true,
			errMsg:  "invalid write timeout",
		},
		{
			name: "invalid max concurrent requests",
			config: ServerConfig{
				Port:              8080,
				ReadTimeout:       60 * time.Second,
				WriteTimeout:      300 * time.Second,
				MaxConcurrentReqs: 0,
			},
			wantErr: true,
			errMsg:  "maxConcurrentRequests must be at least 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
			}
		})
	}
}

// TestGitHubConfig_Validate tests GitHub configuration validation
func TestGitHubConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  GitHubConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: GitHubConfig{
				APIURL:       "https://api.github.com",
				RequiredOrg:  "myorg",
				AuthCacheTTL: 30 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "valid config without org",
			config: GitHubConfig{
				APIURL:       "https://api.github.com",
				RequiredOrg:  "",
				AuthCacheTTL: 30 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "missing API URL",
			config: GitHubConfig{
				APIURL:       "",
				RequiredOrg:  "myorg",
				AuthCacheTTL: 30 * time.Minute,
			},
			wantErr: true,
			errMsg:  "apiURL is required",
		},
		{
			name: "invalid API URL",
			config: GitHubConfig{
				APIURL:       "://invalid",
				RequiredOrg:  "myorg",
				AuthCacheTTL: 30 * time.Minute,
			},
			wantErr: true,
			errMsg:  "invalid apiURL",
		},
		{
			name: "invalid auth cache TTL",
			config: GitHubConfig{
				APIURL:       "https://api.github.com",
				RequiredOrg:  "myorg",
				AuthCacheTTL: 0,
			},
			wantErr: true,
			errMsg:  "invalid authCacheTTL",
		},
		{
			name: "required_teams without required_org (security bypass)",
			config: GitHubConfig{
				APIURL:        "https://api.github.com",
				RequiredOrg:   "",
				RequiredTeams: []string{"team1", "team2"},
				AuthCacheTTL:  30 * time.Minute,
			},
			wantErr: true,
			errMsg:  "required_org must be specified when required_teams is configured",
		},
		{
			name: "required_teams with required_org (valid)",
			config: GitHubConfig{
				APIURL:        "https://api.github.com",
				RequiredOrg:   "myorg",
				RequiredTeams: []string{"team1", "team2"},
				AuthCacheTTL:  30 * time.Minute,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
			}
		})
	}
}

// TestCircuitBreakerConfig_Validate tests circuit breaker configuration validation
func TestCircuitBreakerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  CircuitBreakerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: CircuitBreakerConfig{
				MaxRequests:      10,
				Interval:         60 * time.Second,
				Timeout:          30 * time.Second,
				FailureThreshold: 0.5,
			},
			wantErr: false,
		},
		{
			name: "invalid max requests",
			config: CircuitBreakerConfig{
				MaxRequests:      0,
				Interval:         60 * time.Second,
				Timeout:          30 * time.Second,
				FailureThreshold: 0.5,
			},
			wantErr: true,
			errMsg:  "maxRequests must be at least 1",
		},
		{
			name: "invalid interval",
			config: CircuitBreakerConfig{
				MaxRequests:      10,
				Interval:         0,
				Timeout:          30 * time.Second,
				FailureThreshold: 0.5,
			},
			wantErr: true,
			errMsg:  "invalid interval",
		},
		{
			name: "invalid timeout",
			config: CircuitBreakerConfig{
				MaxRequests:      10,
				Interval:         60 * time.Second,
				Timeout:          0,
				FailureThreshold: 0.5,
			},
			wantErr: true,
			errMsg:  "invalid timeout",
		},
		{
			name: "invalid failure threshold too low",
			config: CircuitBreakerConfig{
				MaxRequests:      10,
				Interval:         60 * time.Second,
				Timeout:          30 * time.Second,
				FailureThreshold: 0,
			},
			wantErr: true,
			errMsg:  "failureThreshold must be between 0 and 1",
		},
		{
			name: "invalid failure threshold too high",
			config: CircuitBreakerConfig{
				MaxRequests:      10,
				Interval:         60 * time.Second,
				Timeout:          30 * time.Second,
				FailureThreshold: 1.5,
			},
			wantErr: true,
			errMsg:  "failureThreshold must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
			}
		})
	}
}

// TestLoggingConfig_Validate tests logging configuration validation
func TestLoggingConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  LoggingConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid json",
			config: LoggingConfig{
				Level:  "info",
				Format: "json",
			},
			wantErr: false,
		},
		{
			name: "valid console",
			config: LoggingConfig{
				Level:  "debug",
				Format: "console",
			},
			wantErr: false,
		},
		{
			name: "invalid level",
			config: LoggingConfig{
				Level:  "invalid",
				Format: "json",
			},
			wantErr: true,
			errMsg:  "invalid level",
		},
		{
			name: "invalid format",
			config: LoggingConfig{
				Level:  "info",
				Format: "xml",
			},
			wantErr: true,
			errMsg:  "invalid format",
		},
		{
			name: "include_headers enabled (allowed with warning)",
			config: LoggingConfig{
				Level:          "info",
				Format:         "json",
				IncludeHeaders: true,
			},
			wantErr: false, // Should be allowed, operator gets warning at startup
		},
		{
			name: "include_body enabled (allowed with warning)",
			config: LoggingConfig{
				Level:       "debug",
				Format:      "json",
				IncludeBody: true,
			},
			wantErr: false, // Should be allowed
		},
		{
			name: "both include_headers and include_body enabled",
			config: LoggingConfig{
				Level:          "debug",
				Format:         "console",
				IncludeHeaders: true,
				IncludeBody:    true,
			},
			wantErr: false, // Should be allowed, operator gets warnings
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
			}
		})
	}
}

// TestProtocolsConfig_Validate tests that at least one protocol must be enabled
func TestProtocolsConfig_Validate(t *testing.T) {
	t.Run("no protocols enabled", func(t *testing.T) {
		cfg := &Config{
			Server: ServerConfig{
				Port:              8080,
				ReadTimeout:       60 * time.Second,
				WriteTimeout:      300 * time.Second,
				MaxConcurrentReqs: 1000,
			},
			GitHub: GitHubConfig{
				APIURL:       "https://api.github.com",
				RequiredOrg:  "myorg",
				AuthCacheTTL: 30 * time.Minute,
			},
			Protocols: ProtocolsConfig{
				OCI: OCIConfig{
					Enabled: false,
				},
				Maven: MavenConfig{
					Enabled: false,
				},
			},
			Logging: LoggingConfig{
				Level:  "info",
				Format: "json",
			},
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("expected error when no protocols enabled")
		}
		if !strings.Contains(err.Error(), "at least one protocol must be enabled") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

// TestOCIConfig_Validate tests OCI protocol validation
func TestOCIConfig_Validate(t *testing.T) {
	t.Run("missing pull backends", func(t *testing.T) {
		cfg := OCIConfig{
			Enabled:      true,
			PullBackends: []OCIBackendConfig{},
			PushBackend: OCIBackendConfig{
				URL:                 "http://registry:5000",
				MaxIdleConns:        200,
				MaxIdleConnsPerHost: 100,
				DialTimeout:         10 * time.Second,
				RequestTimeout:      300 * time.Second,
			},
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for missing pull backends")
		}
		if !strings.Contains(err.Error(), "at least one pull backend is required") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// TestMavenConfig_Validate tests Maven protocol validation
func TestMavenConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  MavenConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with host and path_prefix",
			config: MavenConfig{
				Host:       "maven.example.com",
				PathPrefix: "/maven",
				Backend: MavenBackendConfig{
					URL:                 "https://repo.example.com",
					MaxIdleConns:        200,
					MaxIdleConnsPerHost: 100,
					DialTimeout:         10 * time.Second,
					RequestTimeout:      300 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with host and empty path_prefix",
			config: MavenConfig{
				Host:       "maven.example.com",
				PathPrefix: "",
				Backend: MavenBackendConfig{
					URL:                 "https://repo.example.com",
					MaxIdleConns:        200,
					MaxIdleConnsPerHost: 100,
					DialTimeout:         10 * time.Second,
					RequestTimeout:      300 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with empty host and path_prefix",
			config: MavenConfig{
				Host:       "",
				PathPrefix: "/maven",
				Backend: MavenBackendConfig{
					URL:                 "https://repo.example.com",
					MaxIdleConns:        200,
					MaxIdleConnsPerHost: 100,
					DialTimeout:         10 * time.Second,
					RequestTimeout:      300 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid - empty host requires path_prefix",
			config: MavenConfig{
				Host:       "",
				PathPrefix: "",
				Backend: MavenBackendConfig{
					URL:                 "https://repo.example.com",
					MaxIdleConns:        200,
					MaxIdleConnsPerHost: 100,
					DialTimeout:         10 * time.Second,
					RequestTimeout:      300 * time.Second,
				},
			},
			wantErr: true,
			errMsg:  "path_prefix is required when host is empty",
		},
		{
			name: "invalid - path_prefix must start with /",
			config: MavenConfig{
				Host:       "",
				PathPrefix: "maven",
				Backend: MavenBackendConfig{
					URL:                 "https://repo.example.com",
					MaxIdleConns:        200,
					MaxIdleConnsPerHost: 100,
					DialTimeout:         10 * time.Second,
					RequestTimeout:      300 * time.Second,
				},
			},
			wantErr: true,
			errMsg:  "path_prefix must start with '/'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
			}
		})
	}
}

// TestNPMConfig_Validate tests NPM protocol validation
func TestNPMConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  NPMConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with host and path_prefix",
			config: NPMConfig{
				Host:       "npm.example.com",
				PathPrefix: "/npm",
				Backend: NPMBackendConfig{
					URL:                 "https://registry.npmjs.org",
					MaxIdleConns:        200,
					MaxIdleConnsPerHost: 100,
					DialTimeout:         10 * time.Second,
					RequestTimeout:      300 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with host and empty path_prefix",
			config: NPMConfig{
				Host:       "npm.example.com",
				PathPrefix: "",
				Backend: NPMBackendConfig{
					URL:                 "https://registry.npmjs.org",
					MaxIdleConns:        200,
					MaxIdleConnsPerHost: 100,
					DialTimeout:         10 * time.Second,
					RequestTimeout:      300 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "valid config with empty host and path_prefix",
			config: NPMConfig{
				Host:       "",
				PathPrefix: "/npm",
				Backend: NPMBackendConfig{
					URL:                 "https://registry.npmjs.org",
					MaxIdleConns:        200,
					MaxIdleConnsPerHost: 100,
					DialTimeout:         10 * time.Second,
					RequestTimeout:      300 * time.Second,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid - empty host requires path_prefix",
			config: NPMConfig{
				Host:       "",
				PathPrefix: "",
				Backend: NPMBackendConfig{
					URL:                 "https://registry.npmjs.org",
					MaxIdleConns:        200,
					MaxIdleConnsPerHost: 100,
					DialTimeout:         10 * time.Second,
					RequestTimeout:      300 * time.Second,
				},
			},
			wantErr: true,
			errMsg:  "path_prefix is required when host is empty",
		},
		{
			name: "invalid - path_prefix must start with /",
			config: NPMConfig{
				Host:       "",
				PathPrefix: "npm",
				Backend: NPMBackendConfig{
					URL:                 "https://registry.npmjs.org",
					MaxIdleConns:        200,
					MaxIdleConnsPerHost: 100,
					DialTimeout:         10 * time.Second,
					RequestTimeout:      300 * time.Second,
				},
			},
			wantErr: true,
			errMsg:  "path_prefix must start with '/'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
			}
		})
	}
}

// TestProtocolsConfig_PathPrefixUniqueness tests path_prefix uniqueness validation
func TestProtocolsConfig_PathPrefixUniqueness(t *testing.T) {
	t.Run("path_prefix conflict - both protocols use /registry with empty host", func(t *testing.T) {
		cfg := ProtocolsConfig{
			Maven: MavenConfig{
				Enabled:    true,
				Host:       "",
				PathPrefix: "/registry",
				Backend: MavenBackendConfig{
					URL:                 "https://maven.example.com",
					MaxIdleConns:        200,
					MaxIdleConnsPerHost: 100,
					DialTimeout:         10 * time.Second,
					RequestTimeout:      300 * time.Second,
				},
			},
			NPM: NPMConfig{
				Enabled:    true,
				Host:       "",
				PathPrefix: "/registry",
				Backend: NPMBackendConfig{
					URL:                 "https://npm.example.com",
					MaxIdleConns:        200,
					MaxIdleConnsPerHost: 100,
					DialTimeout:         10 * time.Second,
					RequestTimeout:      300 * time.Second,
				},
			},
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for path_prefix conflict")
		}
		if !strings.Contains(err.Error(), "path_prefix conflict") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("same path_prefix allowed with different hosts", func(t *testing.T) {
		cfg := ProtocolsConfig{
			Maven: MavenConfig{
				Enabled:    true,
				Host:       "maven.example.com",
				PathPrefix: "/api",
				Backend: MavenBackendConfig{
					URL:                 "https://maven.example.com",
					MaxIdleConns:        200,
					MaxIdleConnsPerHost: 100,
					DialTimeout:         10 * time.Second,
					RequestTimeout:      300 * time.Second,
				},
			},
			NPM: NPMConfig{
				Enabled:    true,
				Host:       "npm.example.com",
				PathPrefix: "/api",
				Backend: NPMBackendConfig{
					URL:                 "https://npm.example.com",
					MaxIdleConns:        200,
					MaxIdleConnsPerHost: 100,
					DialTimeout:         10 * time.Second,
					RequestTimeout:      300 * time.Second,
				},
			},
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("should allow same path_prefix with different hosts, got error: %v", err)
		}
	})

	t.Run("unique path_prefix with empty host - valid", func(t *testing.T) {
		cfg := ProtocolsConfig{
			Maven: MavenConfig{
				Enabled:    true,
				Host:       "",
				PathPrefix: "/maven",
				Backend: MavenBackendConfig{
					URL:                 "https://maven.example.com",
					MaxIdleConns:        200,
					MaxIdleConnsPerHost: 100,
					DialTimeout:         10 * time.Second,
					RequestTimeout:      300 * time.Second,
				},
			},
			NPM: NPMConfig{
				Enabled:    true,
				Host:       "",
				PathPrefix: "/npm",
				Backend: NPMBackendConfig{
					URL:                 "https://npm.example.com",
					MaxIdleConns:        200,
					MaxIdleConnsPerHost: 100,
					DialTimeout:         10 * time.Second,
					RequestTimeout:      300 * time.Second,
				},
			},
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("should allow unique path_prefix, got error: %v", err)
		}
	})
}

// TestValidateBackendCommon tests the common backend validation helper
func TestValidateBackendCommon(t *testing.T) {
	// Valid circuit breaker config for tests
	validCB := CircuitBreakerConfig{
		Enabled:          true,
		MaxRequests:      10,
		Interval:         60 * time.Second,
		Timeout:          30 * time.Second,
		FailureThreshold: 0.5,
	}

	// Disabled circuit breaker for simpler tests
	disabledCB := CircuitBreakerConfig{
		Enabled: false,
	}

	tests := []struct {
		name                string
		backendURL          string
		maxIdleConns        int
		maxIdleConnsPerHost int
		dialTimeout         time.Duration
		requestTimeout      time.Duration
		circuitBreaker      CircuitBreakerConfig
		wantErr             bool
		errMsg              string
	}{
		{
			name:                "valid config",
			backendURL:          "https://example.com",
			maxIdleConns:        200,
			maxIdleConnsPerHost: 100,
			dialTimeout:         10 * time.Second,
			requestTimeout:      300 * time.Second,
			circuitBreaker:      disabledCB,
			wantErr:             false,
		},
		{
			name:                "valid config with circuit breaker",
			backendURL:          "https://example.com",
			maxIdleConns:        200,
			maxIdleConnsPerHost: 100,
			dialTimeout:         10 * time.Second,
			requestTimeout:      300 * time.Second,
			circuitBreaker:      validCB,
			wantErr:             false,
		},
		{
			name:                "missing URL",
			backendURL:          "",
			maxIdleConns:        200,
			maxIdleConnsPerHost: 100,
			dialTimeout:         10 * time.Second,
			requestTimeout:      300 * time.Second,
			circuitBreaker:      disabledCB,
			wantErr:             true,
			errMsg:              "url is required",
		},
		{
			name:                "invalid URL",
			backendURL:          "://invalid",
			maxIdleConns:        200,
			maxIdleConnsPerHost: 100,
			dialTimeout:         10 * time.Second,
			requestTimeout:      300 * time.Second,
			circuitBreaker:      disabledCB,
			wantErr:             true,
			errMsg:              "invalid url",
		},
		{
			name:                "maxIdleConns too low",
			backendURL:          "https://example.com",
			maxIdleConns:        0,
			maxIdleConnsPerHost: 100,
			dialTimeout:         10 * time.Second,
			requestTimeout:      300 * time.Second,
			circuitBreaker:      disabledCB,
			wantErr:             true,
			errMsg:              "maxIdleConns must be at least 1",
		},
		{
			name:                "maxIdleConnsPerHost too low",
			backendURL:          "https://example.com",
			maxIdleConns:        200,
			maxIdleConnsPerHost: 0,
			dialTimeout:         10 * time.Second,
			requestTimeout:      300 * time.Second,
			circuitBreaker:      disabledCB,
			wantErr:             true,
			errMsg:              "maxIdleConnsPerHost must be at least 1",
		},
		{
			name:                "maxIdleConnsPerHost exceeds maxIdleConns",
			backendURL:          "https://example.com",
			maxIdleConns:        100,
			maxIdleConnsPerHost: 200,
			dialTimeout:         10 * time.Second,
			requestTimeout:      300 * time.Second,
			circuitBreaker:      disabledCB,
			wantErr:             true,
			errMsg:              "maxIdleConnsPerHost cannot exceed maxIdleConns",
		},
		{
			name:                "invalid dialTimeout",
			backendURL:          "https://example.com",
			maxIdleConns:        200,
			maxIdleConnsPerHost: 100,
			dialTimeout:         0,
			requestTimeout:      300 * time.Second,
			circuitBreaker:      disabledCB,
			wantErr:             true,
			errMsg:              "invalid dialTimeout",
		},
		{
			name:                "negative dialTimeout",
			backendURL:          "https://example.com",
			maxIdleConns:        200,
			maxIdleConnsPerHost: 100,
			dialTimeout:         -5 * time.Second,
			requestTimeout:      300 * time.Second,
			circuitBreaker:      disabledCB,
			wantErr:             true,
			errMsg:              "invalid dialTimeout",
		},
		{
			name:                "invalid requestTimeout",
			backendURL:          "https://example.com",
			maxIdleConns:        200,
			maxIdleConnsPerHost: 100,
			dialTimeout:         10 * time.Second,
			requestTimeout:      0,
			circuitBreaker:      disabledCB,
			wantErr:             true,
			errMsg:              "invalid requestTimeout",
		},
		{
			name:                "negative requestTimeout",
			backendURL:          "https://example.com",
			maxIdleConns:        200,
			maxIdleConnsPerHost: 100,
			dialTimeout:         10 * time.Second,
			requestTimeout:      -300 * time.Second,
			circuitBreaker:      disabledCB,
			wantErr:             true,
			errMsg:              "invalid requestTimeout",
		},
		{
			name:                "invalid circuit breaker config",
			backendURL:          "https://example.com",
			maxIdleConns:        200,
			maxIdleConnsPerHost: 100,
			dialTimeout:         10 * time.Second,
			requestTimeout:      300 * time.Second,
			circuitBreaker: CircuitBreakerConfig{
				Enabled:          true,
				MaxRequests:      0, // Invalid
				Interval:         60 * time.Second,
				Timeout:          30 * time.Second,
				FailureThreshold: 0.5,
			},
			wantErr: true,
			errMsg:  "circuit breaker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBackendCommon(
				tt.backendURL,
				tt.maxIdleConns,
				tt.maxIdleConnsPerHost,
				tt.dialTimeout,
				tt.requestTimeout,
				tt.circuitBreaker,
			)
			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
			}
		})
	}
}

// TestBackendConfig_Validate_Integration tests that all backend types use validateBackendCommon
func TestBackendConfig_Validate_Integration(t *testing.T) {
	// This test ensures that all backend types (OCI, Maven, NPM) properly use validateBackendCommon
	// by testing that they all reject the same invalid configurations

	invalidURL := "://invalid"

	t.Run("OCIBackendConfig rejects invalid URL", func(t *testing.T) {
		backend := OCIBackendConfig{
			URL:                 invalidURL,
			MaxIdleConns:        200,
			MaxIdleConnsPerHost: 100,
			DialTimeout:         10 * time.Second,
			RequestTimeout:      300 * time.Second,
		}
		err := backend.Validate()
		if err == nil {
			t.Error("expected error for invalid URL")
		}
		if !strings.Contains(err.Error(), "invalid url") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("MavenBackendConfig rejects invalid URL", func(t *testing.T) {
		backend := MavenBackendConfig{
			URL:                 invalidURL,
			MaxIdleConns:        200,
			MaxIdleConnsPerHost: 100,
			DialTimeout:         10 * time.Second,
			RequestTimeout:      300 * time.Second,
		}
		err := backend.Validate()
		if err == nil {
			t.Error("expected error for invalid URL")
		}
		if !strings.Contains(err.Error(), "invalid url") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("NPMBackendConfig rejects invalid URL", func(t *testing.T) {
		backend := NPMBackendConfig{
			URL:                 invalidURL,
			MaxIdleConns:        200,
			MaxIdleConnsPerHost: 100,
			DialTimeout:         10 * time.Second,
			RequestTimeout:      300 * time.Second,
		}
		err := backend.Validate()
		if err == nil {
			t.Error("expected error for invalid URL")
		}
		if !strings.Contains(err.Error(), "invalid url") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}
