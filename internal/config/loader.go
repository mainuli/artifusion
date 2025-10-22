package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Load loads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		// Default config locations
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("/etc/artifusion")
		v.AddConfigPath("$HOME/.artifusion")
		v.AddConfigPath("./config")
		v.AddConfigPath(".")
	}

	// Environment variables
	v.SetEnvPrefix("ARTIFUSION")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is OK if we have env vars
	}

	// Unmarshal into config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Expand environment variables in string fields
	cfg.expandEnvVars()

	// Set defaults
	cfg.SetDefaults()

	return &cfg, nil
}

// expandEnvVars expands environment variables in configuration strings
func (c *Config) expandEnvVars() {
	// Expand OCI backend auth credentials
	for i := range c.Protocols.OCI.PullBackends {
		c.expandOCIBackendAuthEnvVars(&c.Protocols.OCI.PullBackends[i])
	}
	c.expandOCIBackendAuthEnvVars(&c.Protocols.OCI.PushBackend)

	// Expand Maven backend auth credentials
	c.expandMavenBackendAuthEnvVars(&c.Protocols.Maven.Backend)

	// Expand NPM backend auth credentials
	c.expandNPMBackendAuthEnvVars(&c.Protocols.NPM.Backend)
}

func (c *Config) expandOCIBackendAuthEnvVars(backend *OCIBackendConfig) {
	if backend.Auth == nil {
		return
	}

	backend.Auth.Username = os.ExpandEnv(backend.Auth.Username)
	backend.Auth.Password = os.ExpandEnv(backend.Auth.Password)
	backend.Auth.Token = os.ExpandEnv(backend.Auth.Token)
	backend.Auth.HeaderValue = os.ExpandEnv(backend.Auth.HeaderValue)
}

func (c *Config) expandMavenBackendAuthEnvVars(backend *MavenBackendConfig) {
	if backend.Auth == nil {
		return
	}

	backend.Auth.Username = os.ExpandEnv(backend.Auth.Username)
	backend.Auth.Password = os.ExpandEnv(backend.Auth.Password)
	backend.Auth.Token = os.ExpandEnv(backend.Auth.Token)
	backend.Auth.HeaderValue = os.ExpandEnv(backend.Auth.HeaderValue)
}

func (c *Config) expandNPMBackendAuthEnvVars(backend *NPMBackendConfig) {
	if backend.Auth == nil {
		return
	}

	backend.Auth.Username = os.ExpandEnv(backend.Auth.Username)
	backend.Auth.Password = os.ExpandEnv(backend.Auth.Password)
	backend.Auth.Token = os.ExpandEnv(backend.Auth.Token)
	backend.Auth.HeaderValue = os.ExpandEnv(backend.Auth.HeaderValue)
}
