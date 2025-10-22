package oci

import (
	"net/http"
	"time"

	"github.com/mainuli/artifusion/internal/config"
	"github.com/mainuli/artifusion/internal/proxy"
	"github.com/mainuli/artifusion/internal/proxy/rewriter"
)

// proxyTransparent proxies the request to the backend transparently
// This streams the request and response without modification (zero-copy)
func (h *Handler) proxyTransparent(w http.ResponseWriter, r *http.Request, backend *config.OCIBackendConfig, path string) error {
	_, err := h.proxyTransparentWithResponse(w, r, backend, path)
	return err
}

// proxyTransparentWithResponse proxies the request and returns the response
// This allows callers to inspect the response status for fallback logic
func (h *Handler) proxyTransparentWithResponse(w http.ResponseWriter, r *http.Request, backend *config.OCIBackendConfig, path string) (*http.Response, error) {
	// Create proxy request
	proxyReq := &proxy.Request{
		Method:      r.Method,
		Path:        path,
		Query:       r.URL.RawQuery,
		Body:        r.Body,
		Headers:     r.Header,
		Backend:     backend,
		OriginalReq: r,
	}

	// Track backend request timing
	start := time.Now()

	// Execute proxy request
	resp, err := h.proxyClient.ProxyRequest(proxyReq)

	// Record metrics regardless of success/failure
	duration := time.Since(start)

	if err != nil {
		// Record backend error metrics
		h.metrics.RecordBackendError(h.Name(), backend.Name, "network_error")
		h.metrics.RecordBackendLatency(backend.Name, r.Method, duration)
		h.metrics.SetBackendHealth(backend.Name, false)

		h.logger.Error().Err(err).
			Str("backend", backend.Name).
			Dur("duration", duration).
			Msg("Backend request failed")

		return nil, err
	}

	// Record backend latency for all requests
	h.metrics.RecordBackendLatency(backend.Name, r.Method, duration)

	// Record backend health based on status code
	if resp.StatusCode >= 500 {
		// Server error - backend is unhealthy
		h.metrics.RecordBackendErrorByStatus(backend.Name, resp.StatusCode)
		h.metrics.SetBackendHealth(backend.Name, false)
	} else if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		// Success - backend is healthy
		h.metrics.SetBackendHealth(backend.Name, true)
	}
	// 4xx errors don't affect backend health (client errors)

	// Prepare response headers
	h.prepareOCIHeaders(r, resp, backend)

	// Stream response to client
	_, streamErr := h.proxyClient.StreamResponse(w, resp, true)
	if streamErr != nil {
		return resp.HTTPResp, streamErr
	}

	return resp.HTTPResp, nil
}

// executeProxyRequest executes a proxy request and returns the response WITHOUT streaming it
// This is used for cascade logic where we need to inspect the response before deciding
// whether to stream it or try another backend
func (h *Handler) executeProxyRequest(r *http.Request, backend *config.OCIBackendConfig, path string) (*proxy.Response, error) {
	// Create proxy request
	proxyReq := &proxy.Request{
		Method:      r.Method,
		Path:        path,
		Query:       r.URL.RawQuery,
		Body:        r.Body,
		Headers:     r.Header,
		Backend:     backend,
		OriginalReq: r,
	}

	// Track backend request timing
	start := time.Now()

	// Execute proxy request
	resp, err := h.proxyClient.ProxyRequest(proxyReq)

	// Record metrics regardless of success/failure
	duration := time.Since(start)

	if err != nil {
		// Record backend error metrics
		h.metrics.RecordBackendError(h.Name(), backend.Name, "network_error")
		h.metrics.RecordBackendLatency(backend.Name, r.Method, duration)
		h.metrics.SetBackendHealth(backend.Name, false)

		h.logger.Error().Err(err).
			Str("backend", backend.Name).
			Dur("duration", duration).
			Msg("Backend request failed")

		return nil, err
	}

	// Record backend latency for all requests
	h.metrics.RecordBackendLatency(backend.Name, r.Method, duration)

	// Record backend health based on status code
	if resp.StatusCode >= 500 {
		// Server error - backend is unhealthy
		h.metrics.RecordBackendErrorByStatus(backend.Name, resp.StatusCode)
		h.metrics.SetBackendHealth(backend.Name, false)
	} else if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		// Success - backend is healthy
		h.metrics.SetBackendHealth(backend.Name, true)
	}
	// 4xx errors don't affect backend health (client errors)

	// Prepare response headers
	h.prepareOCIHeaders(r, resp, backend)

	// Return response WITHOUT streaming
	return resp, nil
}

// prepareOCIHeaders modifies response headers for OCI/Docker compatibility
func (h *Handler) prepareOCIHeaders(r *http.Request, resp *proxy.Response, backend *config.OCIBackendConfig) {
	// Ensure Docker API version header is present
	if resp.Headers.Get("Docker-Distribution-Api-Version") == "" {
		resp.Headers.Set("Docker-Distribution-Api-Version", "registry/2.0")
	}

	// For HEAD requests, keep Content-Length header (required by Docker client)
	// For other requests with bodies, remove Content-Length to use chunked encoding
	if r.Method != http.MethodHead {
		resp.Headers.Del("Content-Length")
	}

	// Determine public URL for URL rewriting
	// Constructs base URL dynamically from request headers + protocol config
	publicURL := h.getEffectiveBaseURL(r)

	// Use URL rewriter to rewrite response headers (Location, WWW-Authenticate, etc.)
	h.getURLRewriter(publicURL).RewriteResponseHeaders(resp, backend)
}

// getURLRewriter returns a URL rewriter configured with the given public URL
func (h *Handler) getURLRewriter(publicURL string) *rewriter.URLRewriter {
	return rewriter.New(publicURL, h.logger)
}
