package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// BackendConfig is an interface that all backend configuration types must implement
// This allows the proxy layer to work with any backend type without coupling to specific implementations
type BackendConfig interface {
	GetName() string
	GetURL() string
	GetMaxIdleConns() int
	GetMaxIdleConnsPerHost() int
	GetIdleConnTimeout() time.Duration
	GetDialTimeout() time.Duration
	GetRequestTimeout() time.Duration
	GetCircuitBreaker() *CircuitBreakerConfig
}

// CircuitBreakerConfig represents circuit breaker configuration
// This is redeclared here to avoid circular dependencies with the config package
type CircuitBreakerConfig struct {
	Enabled          bool
	MaxRequests      uint32
	Interval         time.Duration
	Timeout          time.Duration
	FailureThreshold float64
}

// Client handles backend proxying with connection pooling
type Client struct {
	httpClients       map[string]*http.Client
	mu                sync.RWMutex
	logger            zerolog.Logger
	circuitBreakerMgr *CircuitBreakerManager
}

// NewClient creates a new proxy client
func NewClient(logger zerolog.Logger, cbManager *CircuitBreakerManager) *Client {
	return &Client{
		httpClients:       make(map[string]*http.Client),
		logger:            logger,
		circuitBreakerMgr: cbManager,
	}
}

// Request represents a proxy request
type Request struct {
	Method      string
	Path        string
	Query       string
	Body        io.Reader
	Headers     http.Header
	Backend     BackendConfig
	OriginalReq *http.Request
}

// Response represents a proxy response
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       io.ReadCloser
	HTTPResp   *http.Response
}

// ProxyRequest proxies a request to the backend with connection pooling and circuit breaker protection
func (c *Client) ProxyRequest(req *Request) (*Response, error) {
	// If circuit breaker is enabled for this backend, wrap the request
	if c.circuitBreakerMgr != nil {
		result, err := c.circuitBreakerMgr.Execute(req.Backend, func() (interface{}, error) {
			return c.doProxyRequest(req)
		})

		if err != nil {
			return nil, err
		}

		return result.(*Response), nil
	}

	// Fallback to direct execution if no circuit breaker
	return c.doProxyRequest(req)
}

// doProxyRequest performs the actual proxy request without circuit breaker
func (c *Client) doProxyRequest(req *Request) (*Response, error) {
	// Build backend URL
	backendURL := c.buildBackendURL(req.Backend.GetURL(), req.Path, req.Query)

	c.logger.Debug().
		Str("backend_url", backendURL).
		Str("method", req.Method).
		Msg("Proxying to backend")

	// Create backend request
	backendReq, err := http.NewRequestWithContext(req.OriginalReq.Context(), req.Method, backendURL, req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to create backend request: %w", err)
	}

	// Copy headers (excluding Authorization - backend auth already injected)
	for key, values := range req.Headers {
		if key == "Authorization" {
			continue
		}
		for _, value := range values {
			backendReq.Header.Add(key, value)
		}
	}

	// Get or create HTTP client for this backend
	client := c.getOrCreateClient(req.Backend)

	// Execute request
	startTime := time.Now()
	resp, err := client.Do(backendReq)
	duration := time.Since(startTime)

	if err != nil {
		c.logger.Error().Err(err).
			Str("backend", req.Backend.GetName()).
			Str("url", backendURL).
			Dur("duration", duration).
			Msg("Backend request failed")
		return nil, err
	}

	c.logger.Debug().
		Str("backend", req.Backend.GetName()).
		Int("status", resp.StatusCode).
		Dur("duration", duration).
		Msg("Backend response received")

	return &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       resp.Body,
		HTTPResp:   resp,
	}, nil
}

// StreamResponse streams the response to the client with zero-copy
func (c *Client) StreamResponse(w http.ResponseWriter, resp *Response, copyHeaders bool) (int64, error) {
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.logger.Warn().Err(err).Msg("Failed to close response body after streaming")
		}
	}()

	// Copy response headers if requested
	if copyHeaders {
		for key, values := range resp.Headers {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
	}

	// Write status code
	w.WriteHeader(resp.StatusCode)

	// Stream response body (zero-copy, no buffering)
	// CRITICAL: For multi-GB files, streaming prevents memory exhaustion
	bytesWritten, err := io.Copy(w, resp.Body)
	if err != nil {
		c.logger.Error().Err(err).
			Int64("bytes_written", bytesWritten).
			Msg("Error streaming response body")
		return bytesWritten, err
	}

	c.logger.Debug().
		Int64("bytes", bytesWritten).
		Msg("Response streamed successfully")

	return bytesWritten, nil
}

// ReadResponseBody reads the full response body into memory
// Use only for small responses that need to be modified (e.g., XML rewriting)
func (c *Client) ReadResponseBody(resp *Response) ([]byte, error) {
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.logger.Warn().Err(err).Msg("Failed to close response body after reading")
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to read response body")
		return nil, err
	}

	c.logger.Debug().
		Int("bytes", len(body)).
		Msg("Response body read into memory")

	return body, nil
}

// WriteResponse writes a modified response body to the client
func (c *Client) WriteResponse(w http.ResponseWriter, resp *Response, body []byte, copyHeaders bool) error {
	// Copy response headers if requested
	if copyHeaders {
		for key, values := range resp.Headers {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
	}

	// Update Content-Length
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))

	// Write status code
	w.WriteHeader(resp.StatusCode)

	// Write modified body
	bytesWritten, err := w.Write(body)
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to write response")
		return err
	}

	c.logger.Debug().
		Int("bytes", bytesWritten).
		Bool("modified", true).
		Msg("Response written with modifications")

	return nil
}

// buildBackendURL constructs the backend URL with path and query
func (c *Client) buildBackendURL(baseURL, path, query string) string {
	backendURL := baseURL + path
	if query != "" {
		backendURL += "?" + query
	}
	return backendURL
}

// getOrCreateClient gets or creates an HTTP client for a backend with connection pooling
func (c *Client) getOrCreateClient(backend BackendConfig) *http.Client {
	// Try read lock first (fast path)
	c.mu.RLock()
	client, exists := c.httpClients[backend.GetName()]
	c.mu.RUnlock()

	if exists {
		return client
	}

	// Need to create client, acquire write lock
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := c.httpClients[backend.GetName()]; exists {
		return client
	}

	// Create HTTP transport with aggressive connection pooling for high concurrency
	transport := &http.Transport{
		// Connection pooling
		MaxIdleConns:        backend.GetMaxIdleConns(),
		MaxIdleConnsPerHost: backend.GetMaxIdleConnsPerHost(),
		IdleConnTimeout:     backend.GetIdleConnTimeout(),

		// Connection establishment
		DialContext: (&net.Dialer{
			Timeout:   backend.GetDialTimeout(),
			KeepAlive: 30 * time.Second,
		}).DialContext,

		// TLS optimization
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,

		// Reuse connections
		DisableKeepAlives: false,
	}

	// Create HTTP client
	client = &http.Client{
		Transport: transport,
		Timeout:   backend.GetRequestTimeout(),
		// Don't follow redirects by default - let caller decide
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	c.httpClients[backend.GetName()] = client

	c.logger.Debug().
		Str("backend", backend.GetName()).
		Int("max_idle_conns", backend.GetMaxIdleConns()).
		Int("max_idle_conns_per_host", backend.GetMaxIdleConnsPerHost()).
		Dur("timeout", backend.GetRequestTimeout()).
		Msg("Created HTTP client for backend")

	return client
}
