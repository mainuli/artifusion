package npm

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mainuli/artifusion/internal/config"
	"github.com/mainuli/artifusion/internal/proxy"
)

// proxyWithRewriting proxies the request to the backend with URL rewriting
func (h *Handler) proxyWithRewriting(w http.ResponseWriter, r *http.Request, backend *config.NPMBackendConfig) error {
	// Validate inputs
	if r == nil {
		return fmt.Errorf("request is nil")
	}
	if backend == nil {
		return fmt.Errorf("backend config is nil")
	}
	if r.URL == nil {
		return fmt.Errorf("request URL is nil")
	}

	// Strip path prefix before sending to backend
	path := r.URL.Path
	if h.config.PathPrefix != "" {
		path = strings.TrimPrefix(path, h.config.PathPrefix)
		// Ensure path starts with /
		if path == "" || !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
	}

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

		return err
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

	// Determine proxy URL for rewriting (base URL + path prefix)
	proxyURL := h.determineProxyURL(r)

	// Rewrite Location header (for redirects)
	if location := resp.Headers.Get("Location"); location != "" {
		rewritten := h.rewriteURL(
			location,
			h.config.Backend.URL,
			proxyURL,
		)
		resp.Headers.Set("Location", rewritten)
	}

	// Get content type
	contentType := resp.Headers.Get("Content-Type")

	// Check if we should rewrite the body
	if h.shouldRewriteBody(contentType) {
		// Buffer and rewrite JSON content (package metadata)
		body, err := h.proxyClient.ReadResponseBody(resp)
		if err != nil {
			// Close response body before returning to prevent resource leak
			if closeErr := resp.Body.Close(); closeErr != nil {
				h.logger.Warn().Err(closeErr).Msg("Failed to close response body after read error")
			}
			w.WriteHeader(resp.StatusCode)
			return err
		}

		// Decompress gzip content if needed for URL rewriting
		contentEncoding := resp.Headers.Get("Content-Encoding")
		if decompressed, wasDecompressed := h.decompressIfNeeded(body, contentEncoding); wasDecompressed {
			body = decompressed
			// Remove Content-Encoding header since we decompressed
			resp.Headers.Del("Content-Encoding")
			// Also remove Content-Length since it will change after rewriting
			resp.Headers.Del("Content-Length")
		}

		// Rewrite URLs in body
		rewritten, err := h.rewritePackageJSON(
			body,
			h.config.Backend.URL,
			proxyURL,
		)
		if err != nil {
			// If rewriting fails, log warning but still return original content
			h.logger.Warn().Err(err).
				Str("content_type", contentType).
				Msg("Failed to rewrite response body, returning original")
			rewritten = body
		}

		// Write modified response (WriteResponse handles body close)
		return h.proxyClient.WriteResponse(w, resp, rewritten, true)
	}

	// Stream binary content (tarballs) without modification
	// StreamResponse handles body close
	_, err = h.proxyClient.StreamResponse(w, resp, true)
	return err
}

// decompressIfNeeded decompresses gzip-encoded content if needed
// Returns the decompressed body and true if decompression occurred, or original body and false otherwise
func (h *Handler) decompressIfNeeded(body []byte, contentEncoding string) ([]byte, bool) {
	if contentEncoding != "gzip" {
		return body, false
	}

	gzReader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		h.logger.Warn().Err(err).Msg("Failed to create gzip reader, using raw body")
		return body, false
	}

	decompressed, err := io.ReadAll(gzReader)
	if closeErr := gzReader.Close(); closeErr != nil {
		h.logger.Warn().Err(closeErr).Msg("Failed to close gzip reader")
	}

	if err != nil {
		h.logger.Warn().Err(err).Msg("Failed to decompress gzip body, using raw body")
		return body, false
	}

	h.logger.Debug().
		Int("compressed_size", len(body)).
		Int("decompressed_size", len(decompressed)).
		Msg("Decompressed gzip response for rewriting")

	return decompressed, true
}
