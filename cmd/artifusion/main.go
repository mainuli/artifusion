package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mainuli/artifusion/internal/auth"
	"github.com/mainuli/artifusion/internal/config"
	"github.com/mainuli/artifusion/internal/constants"
	"github.com/mainuli/artifusion/internal/detector"
	"github.com/mainuli/artifusion/internal/errors"
	"github.com/mainuli/artifusion/internal/handler/maven"
	"github.com/mainuli/artifusion/internal/handler/npm"
	"github.com/mainuli/artifusion/internal/handler/oci"
	"github.com/mainuli/artifusion/internal/health"
	"github.com/mainuli/artifusion/internal/logging"
	"github.com/mainuli/artifusion/internal/metrics"
	"github.com/mainuli/artifusion/internal/middleware"
	"github.com/mainuli/artifusion/internal/proxy"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

const (
	version = "1.0.0"
)

func main() {
	// Setup initial logging for early startup (before config is loaded)
	// This ensures config loading/validation logs are also formatted nicely
	initialFormat := getEnvOrDefault("ARTIFUSION_LOGGING_FORMAT", "console")
	initialLevel := getEnvOrDefault("ARTIFUSION_LOGGING_LEVEL", "info")

	initialLogger := logging.NewLogger(
		logging.Config{
			Level:  initialLevel,
			Format: initialFormat,
		},
		"artifusion",
		version,
	)
	log.Logger = initialLogger

	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatal().Err(err).Msg("Invalid configuration")
	}

	// Reconfigure logging with settings from config file
	// This allows config file to override environment variables
	logger := logging.NewLogger(
		logging.Config{
			Level:  cfg.Logging.Level,
			Format: cfg.Logging.Format,
		},
		"artifusion",
		version,
	)
	log.Logger = logger

	logger.Info().
		Str("version", version).
		Int("port", cfg.Server.Port).
		Str("log_level", cfg.Logging.Level).
		Msg("Starting Artifusion")

	// Create metrics collector
	metricsCollector := metrics.NewMetrics("artifusion") // Initialize metrics (automatically registered with Prometheus)

	// Create circuit breaker manager with logger and metrics
	circuitBreakerManager := proxy.NewCircuitBreakerManager(logger, metricsCollector)

	logger.Info().Msg("Circuit breaker manager initialized")

	// Create GitHub authentication client
	githubClient := auth.NewGitHubClient(
		cfg.GitHub.APIURL,
		cfg.GitHub.AuthCacheTTL,
		cfg.GitHub.RateLimitBuffer,
		logger,
	)

	// Create shared client authenticator
	clientAuthenticator := auth.NewClientAuthenticator(
		githubClient,
		cfg.GitHub.RequiredOrg,
		cfg.GitHub.RequiredTeams,
		logger,
	)

	// Create shared proxy client with circuit breaker support
	proxyClient := proxy.NewClient(logger, circuitBreakerManager)

	// Create health check handler
	healthHandler := health.NewHandler(version)

	// Register health checkers
	healthHandler.RegisterChecker("github_api", func(ctx context.Context) error {
		// Simple check - could validate GitHub API connectivity
		return nil
	})

	// Setup router
	router := chi.NewRouter()

	// Apply middleware stack (order matters!)

	// 1. Request ID - must be first to ensure all logs have request ID
	router.Use(middleware.RequestID)

	// 2. Security Headers - set security headers early
	router.Use(middleware.SecurityHeaders)

	// 3. Recovery - catch panics early
	router.Use(middleware.Recovery(logger))

	// 4. Logging - log all requests
	// SECURITY NOTICE: Warn if header logging is enabled
	if cfg.Logging.IncludeHeaders {
		logger.Warn().
			Msg("Header logging is ENABLED. Sensitive headers (Authorization, Cookie, etc.) are redacted, " +
				"but this may expose PII in other headers (User-Agent, X-Forwarded-For, Referer) and " +
				"significantly increase log volume/storage costs. Only enable for debugging.")
	}
	router.Use(middleware.Logger(logger, cfg.Logging.IncludeHeaders, cfg.Logging.IncludeBody))

	// 5. Request timeout - enforce maximum request duration
	requestTimeout := constants.DefaultRequestTimeout
	if cfg.Server.WriteTimeout > 0 && cfg.Server.WriteTimeout < requestTimeout {
		// Use server write timeout if it's lower (more restrictive)
		requestTimeout = cfg.Server.WriteTimeout
	}
	router.Use(middleware.Timeout(requestTimeout))

	logger.Info().
		Dur("timeout", requestTimeout).
		Msg("Request timeout middleware enabled")

	// 6. Concurrency limiting - limit total concurrent requests
	if cfg.Server.MaxConcurrentReqs > 0 {
		concurrencyLimiter := middleware.NewConcurrencyLimiter(cfg.Server.MaxConcurrentReqs)
		router.Use(concurrencyLimiter.Middleware)

		logger.Info().
			Int("max_concurrent_requests", cfg.Server.MaxConcurrentReqs).
			Msg("Concurrency limiting enabled")
	}

	// 7. Rate limiting - global and per-user rate limiting
	if cfg.RateLimit.Enabled || cfg.RateLimit.PerUserEnabled {
		rateLimiter := middleware.NewRateLimiter(&cfg.RateLimit)
		router.Use(rateLimiter.Middleware)
		defer rateLimiter.Stop()

		logger.Info().
			Bool("global_enabled", cfg.RateLimit.Enabled).
			Float64("global_rps", cfg.RateLimit.RequestsPerSec).
			Bool("per_user_enabled", cfg.RateLimit.PerUserEnabled).
			Float64("per_user_rps", cfg.RateLimit.PerUserRequests).
			Msg("Rate limiting enabled")
	}

	// Health endpoints
	router.Get("/health", healthHandler.LivenessHandler())
	router.Get("/ready", healthHandler.ReadinessHandler())

	// Metrics endpoint (if enabled)
	if cfg.Metrics.Enabled {
		router.Handle(cfg.Metrics.Path, promhttp.Handler())

		logger.Info().
			Str("path", cfg.Metrics.Path).
			Msg("Prometheus metrics endpoint enabled")
	}

	// Setup protocol detection chain
	detectorChain := detector.NewChain()

	// Initialize protocol handlers
	var ociHandler *oci.Handler
	var mavenHandler *maven.Handler
	var npmHandler *npm.Handler

	// Register OCI handler if enabled
	if cfg.Protocols.OCI.Enabled {
		ociHandler = oci.NewHandler(
			&cfg.Protocols.OCI,
			clientAuthenticator,
			proxyClient,
			metricsCollector,
			logger,
		)

		// Register OCI detector with host
		detectorChain.Register(detector.NewOCIDetector(cfg.Protocols.OCI.Host))

		logger.Info().
			Str("host", cfg.Protocols.OCI.Host).
			Int("pull_backends", len(cfg.Protocols.OCI.PullBackends)).
			Str("push_backend", cfg.Protocols.OCI.PushBackend.URL).
			Msg("OCI/Docker protocol handler enabled")
	}

	// Register Maven handler if enabled
	if cfg.Protocols.Maven.Enabled {
		mavenHandler = maven.NewHandler(
			&cfg.Protocols.Maven,
			clientAuthenticator,
			proxyClient,
			metricsCollector,
			logger,
		)

		// Register Maven detector with host and path prefix
		detectorChain.Register(detector.NewMavenDetector(
			cfg.Protocols.Maven.Host,
			cfg.Protocols.Maven.PathPrefix,
		))

		logger.Info().
			Str("host", cfg.Protocols.Maven.Host).
			Str("path_prefix", cfg.Protocols.Maven.PathPrefix).
			Str("backend", cfg.Protocols.Maven.Backend.URL).
			Msg("Maven protocol handler enabled")
	}

	// Register NPM handler if enabled
	if cfg.Protocols.NPM.Enabled {
		npmHandler = npm.NewHandler(
			&cfg.Protocols.NPM,
			clientAuthenticator,
			proxyClient,
			metricsCollector,
			logger,
		)

		// Register NPM detector with host and path prefix
		detectorChain.Register(detector.NewNPMDetector(
			cfg.Protocols.NPM.Host,
			cfg.Protocols.NPM.PathPrefix,
		))

		logger.Info().
			Str("host", cfg.Protocols.NPM.Host).
			Str("path_prefix", cfg.Protocols.NPM.PathPrefix).
			Str("backend", cfg.Protocols.NPM.Backend.URL).
			Msg("NPM protocol handler enabled")
	}

	// Main request handler with protocol detection
	router.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		// Detect protocol
		protocol := detectorChain.Detect(r)

		logger.Debug().
			Str("protocol", string(protocol)).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Msg("Protocol detected")

		// Route to appropriate handler
		switch protocol {
		case detector.ProtocolOCI:
			if ociHandler != nil {
				ociHandler.ServeHTTP(w, r)
				return
			}

		case detector.ProtocolMaven:
			if mavenHandler != nil {
				mavenHandler.ServeHTTP(w, r)
				return
			}

		case detector.ProtocolNPM:
			if npmHandler != nil {
				npmHandler.ServeHTTP(w, r)
				return
			}

		case detector.ProtocolUnknown:
			fallthrough
		default:
			// Unknown protocol
			errors.ErrorResponse(w, errors.ErrProtocolNotSupported)
			return
		}

		// Fallback (shouldn't reach here)
		errors.ErrorResponse(w, errors.ErrInternal.WithMessage("Internal routing error"))
	})

	// Create HTTP server with optimized settings
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:           router,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
		MaxHeaderBytes:    cfg.Server.MaxHeaderBytes,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Log server configuration
	logger.Info().
		Int("port", cfg.Server.Port).
		Dur("read_timeout", cfg.Server.ReadTimeout).
		Dur("write_timeout", cfg.Server.WriteTimeout).
		Dur("idle_timeout", cfg.Server.IdleTimeout).
		Int("max_header_bytes", cfg.Server.MaxHeaderBytes).
		Msg("HTTP server configuration")

	// Setup graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info().
			Str("address", server.Addr).
			Msg("HTTP server starting")

		serverErrors <- server.ListenAndServe()
	}()

	// Block until shutdown signal or server error
	select {
	case err := <-serverErrors:
		logger.Fatal().Err(err).Msg("Server failed to start")

	case sig := <-shutdown:
		logger.Info().
			Str("signal", sig.String()).
			Msg("Shutdown signal received, starting graceful shutdown")

		// Create shutdown context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()

		// Attempt graceful shutdown
		if err := server.Shutdown(ctx); err != nil {
			logger.Error().Err(err).Msg("Server forced to shutdown")

			// Force close after timeout
			if err := server.Close(); err != nil {
				logger.Error().Err(err).Msg("Failed to close server")
			}
		}

		logger.Info().Msg("Server shutdown complete")
	}

	// Log GitHub auth cache statistics
	stats := githubClient.CacheStats()
	logger.Info().
		Int64("cache_hits", stats.Hits).
		Int64("cache_misses", stats.Misses).
		Int("cache_size", stats.Size).
		Float64("hit_rate", stats.HitRate).
		Msg("GitHub auth cache statistics")
}

// getEnvOrDefault returns the value of an environment variable or a default value if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
