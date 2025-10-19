package oci

import (
	"fmt"
	"net/http"

	"github.com/mainuli/artifusion/internal/auth"
	"github.com/mainuli/artifusion/internal/config"
	"github.com/mainuli/artifusion/internal/detector"
	"github.com/mainuli/artifusion/internal/errors"
	"github.com/mainuli/artifusion/internal/metrics"
	"github.com/mainuli/artifusion/internal/proxy"
	"github.com/rs/zerolog"
)

// Handler handles OCI/Docker registry protocol requests
type Handler struct {
	config        *config.OCIConfig
	authenticator *auth.ClientAuthenticator
	proxyClient   *proxy.Client
	metrics       *metrics.Metrics
	logger        zerolog.Logger
}

// NewHandler creates a new OCI handler
func NewHandler(
	cfg *config.OCIConfig,
	authenticator *auth.ClientAuthenticator,
	proxyClient *proxy.Client,
	metricsCollector *metrics.Metrics,
	logger zerolog.Logger,
) *Handler {
	return &Handler{
		config:        cfg,
		authenticator: authenticator,
		proxyClient:   proxyClient,
		metrics:       metricsCollector,
		logger:        logger.With().Str("protocol", "oci").Logger(),
	}
}

// ServeHTTP handles OCI/Docker registry requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Msg("OCI request received")

	// Step 1: Authenticate client
	authResult, updatedReq, err := h.authenticateClient(r)
	if err != nil {
		h.handleAuthError(w, r, err)
		return
	}

	// Step 2: Select backend and proxy request
	if err := h.selectBackendAndProxy(w, updatedReq, authResult); err != nil {
		h.logger.Error().Err(err).
			Str("path", updatedReq.URL.Path).
			Str("method", updatedReq.Method).
			Msg("Failed to proxy request")

		errors.ErrorResponse(w, errors.ErrInternal.WithInternal(err))
	}
}

// Name returns the handler name
func (h *Handler) Name() string {
	return "oci"
}

// getEffectiveBaseURL constructs the base URL for this OCI handler based on:
// - Host-based routing: uses configured host + detected scheme
// - Path-based routing: uses request host (proxy-aware) + detected scheme
// - OCI always uses /v2 path (hardcoded by OCI spec, not configurable)
func (h *Handler) getEffectiveBaseURL(r *http.Request) string {
	scheme := detector.GetRequestScheme(r)

	var host string
	if h.config.Host != "" {
		// Host-based routing: use configured host
		host = h.config.Host
	} else {
		// Path-based routing: detect host from request (proxy-aware)
		host = detector.GetRequestHost(r)
	}

	// OCI always uses /v2 path (hardcoded by OCI Distribution Spec)
	return fmt.Sprintf("%s://%s/v2", scheme, host)
}
