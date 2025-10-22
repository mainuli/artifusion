package config

import (
	"testing"
)

// TestSetDefaults_RateLimitBurst tests that burst defaults are applied independently
// of requests_per_sec configuration (fixes bug where burst stayed at 0 when only
// requests_per_sec was configured)
func TestSetDefaults_RateLimitBurst(t *testing.T) {
	tests := []struct {
		name             string
		cfg              Config
		wantBurst        int
		wantPerUserBurst int
	}{
		{
			name: "burst gets default when requests_per_sec is set but burst is not",
			cfg: Config{
				RateLimit: RateLimitConfig{
					Enabled:        true,
					RequestsPerSec: 1000.0, // Set custom value
					Burst:          0,      // Not set - should get default
				},
			},
			wantBurst: DefaultRateLimitBurst,
		},
		{
			name: "per_user_burst gets default when per_user_requests is set but burst is not",
			cfg: Config{
				RateLimit: RateLimitConfig{
					PerUserEnabled:  true,
					PerUserRequests: 100.0, // Set custom value
					PerUserBurst:    0,     // Not set - should get default
				},
			},
			wantPerUserBurst: DefaultPerUserBurst,
		},
		{
			name: "both burst fields get defaults when only rates are set",
			cfg: Config{
				RateLimit: RateLimitConfig{
					Enabled:         true,
					RequestsPerSec:  500.0, // Custom
					Burst:           0,     // Not set
					PerUserEnabled:  true,
					PerUserRequests: 50.0, // Custom
					PerUserBurst:    0,    // Not set
				},
			},
			wantBurst:        DefaultRateLimitBurst,
			wantPerUserBurst: DefaultPerUserBurst,
		},
		{
			name: "explicit burst values are preserved",
			cfg: Config{
				RateLimit: RateLimitConfig{
					Enabled:         true,
					RequestsPerSec:  1000.0,
					Burst:           5000, // Explicit value
					PerUserEnabled:  true,
					PerUserRequests: 100.0,
					PerUserBurst:    500, // Explicit value
				},
			},
			wantBurst:        5000,
			wantPerUserBurst: 500,
		},
		{
			name: "all fields get defaults when none are set",
			cfg: Config{
				RateLimit: RateLimitConfig{
					Enabled:        true,
					PerUserEnabled: true,
				},
			},
			wantBurst:        DefaultRateLimitBurst,
			wantPerUserBurst: DefaultPerUserBurst,
		},
		{
			name: "burst not set when rate limiting disabled",
			cfg: Config{
				RateLimit: RateLimitConfig{
					Enabled: false,
					Burst:   0,
				},
			},
			wantBurst: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cfg.SetDefaults()

			if tt.wantBurst > 0 && tt.cfg.RateLimit.Burst != tt.wantBurst {
				t.Errorf("Burst = %v, want %v", tt.cfg.RateLimit.Burst, tt.wantBurst)
			}

			if tt.wantPerUserBurst > 0 && tt.cfg.RateLimit.PerUserBurst != tt.wantPerUserBurst {
				t.Errorf("PerUserBurst = %v, want %v", tt.cfg.RateLimit.PerUserBurst, tt.wantPerUserBurst)
			}
		})
	}
}

// TestSetDefaults_RateLimitRequestsPerSec tests that requests_per_sec defaults are still applied
func TestSetDefaults_RateLimitRequestsPerSec(t *testing.T) {
	tests := []struct {
		name                string
		cfg                 Config
		wantRequestsPerSec  float64
		wantPerUserRequests float64
	}{
		{
			name: "requests_per_sec gets default when not set",
			cfg: Config{
				RateLimit: RateLimitConfig{
					Enabled:        true,
					RequestsPerSec: 0,
				},
			},
			wantRequestsPerSec: DefaultRateLimitRequestsPerSec,
		},
		{
			name: "per_user_requests gets default when not set",
			cfg: Config{
				RateLimit: RateLimitConfig{
					PerUserEnabled:  true,
					PerUserRequests: 0,
				},
			},
			wantPerUserRequests: DefaultPerUserRequests,
		},
		{
			name: "explicit values are preserved",
			cfg: Config{
				RateLimit: RateLimitConfig{
					Enabled:         true,
					RequestsPerSec:  500.0,
					PerUserEnabled:  true,
					PerUserRequests: 50.0,
				},
			},
			wantRequestsPerSec:  500.0,
			wantPerUserRequests: 50.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cfg.SetDefaults()

			if tt.wantRequestsPerSec > 0 && tt.cfg.RateLimit.RequestsPerSec != tt.wantRequestsPerSec {
				t.Errorf("RequestsPerSec = %v, want %v", tt.cfg.RateLimit.RequestsPerSec, tt.wantRequestsPerSec)
			}

			if tt.wantPerUserRequests > 0 && tt.cfg.RateLimit.PerUserRequests != tt.wantPerUserRequests {
				t.Errorf("PerUserRequests = %v, want %v", tt.cfg.RateLimit.PerUserRequests, tt.wantPerUserRequests)
			}
		})
	}
}
