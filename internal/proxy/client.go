package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/mainuli/artifusion/internal/config"
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
	GetCircuitBreaker() *config.CircuitBreakerConfig
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

// hopByHopHeaders lists HTTP/1.1 hop-by-hop headers per RFC 7230 Section 6.1.
// These headers are meaningful only for a single transport-level connection
// and must not be forwarded by proxies to prevent request smuggling and
// connection poisoning attacks.
var hopByHopHeaders = map[string]bool{
	"connection":          true,
	"proxy-connection":    true, // Non-standard but common
	"keep-alive":          true,
	"proxy-authenticate":  true,
	"proxy-authorization": true,
	"te":                  true,
	"trailer":             true,
	"transfer-encoding":   true,
	"upgrade":             true,
}

// removeHopByHopHeaders filters headers that should not be forwarded to upstream.
// Mirrors httputil.ReverseProxy behavior per RFC 7230.
//
// This prevents HTTP request smuggling attacks by ensuring hop-by-hop headers
// (which are meant for a single connection) don't reach the backend.
//
// Additionally, any headers named in the Connection header are also removed,
// as they are connection-specific per the HTTP spec.
func removeHopByHopHeaders(headers http.Header) http.Header {
	// Create a new header map for the filtered result
	filtered := make(http.Header)

	// Build set of headers to remove from Connection header values
	// The Connection header can specify additional hop-by-hop headers
	// e.g., "Connection: close, X-Custom-Header"
	removeHeaders := make(map[string]bool)
	for _, v := range headers["Connection"] {
		for _, field := range strings.Split(v, ",") {
			field = strings.TrimSpace(field)
			if field != "" {
				removeHeaders[strings.ToLower(field)] = true
			}
		}
	}

	// Copy headers except hop-by-hop ones
	for key, values := range headers {
		lowerKey := strings.ToLower(key)

		// Skip if it's a standard hop-by-hop header
		if hopByHopHeaders[lowerKey] {
			continue
		}

		// Skip if it's named in Connection header
		if removeHeaders[lowerKey] {
			continue
		}

		// Safe to forward - copy all values
		filtered[key] = values
	}

	return filtered
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

	// SECURITY: Filter hop-by-hop headers before forwarding (RFC 7230 Section 6.1)
	// This prevents HTTP request smuggling and connection poisoning attacks
	filteredHeaders := removeHopByHopHeaders(req.Headers)

	// Copy safe headers (excluding Authorization - will be set separately for backend auth)
	for key, values := range filteredHeaders {
		if key == "Authorization" {
			continue
		}
		for _, value := range values {
			backendReq.Header.Add(key, value)
		}
	}

	// Inject backend authentication if configured
	if err := c.injectBackendAuth(backendReq, req.Backend); err != nil {
		return nil, fmt.Errorf("failed to inject backend auth: %w", err)
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

// authProvider is an interface for backends that support authentication
type authProvider interface {
	GetAuth() *config.AuthConfig
}

// validateAuthCredentials validates authentication credentials for security
func validateAuthCredentials(auth *config.AuthConfig) error {
	switch strings.ToLower(auth.Type) {
	case "basic":
		if auth.Username == "" || auth.Password == "" {
			return fmt.Errorf("basic auth requires both username and password")
		}
		if strings.ContainsAny(auth.Username, "\r\n") || strings.ContainsAny(auth.Password, "\r\n") {
			return fmt.Errorf("username/password cannot contain newlines")
		}
	case "bearer":
		if auth.Token == "" {
			return fmt.Errorf("bearer auth requires a token")
		}
		if strings.ContainsAny(auth.Token, "\r\n") {
			return fmt.Errorf("token cannot contain newlines")
		}
	case "header":
		if auth.HeaderName == "" || auth.HeaderValue == "" {
			return fmt.Errorf("header auth requires both header_name and header_value")
		}
		if strings.ContainsAny(auth.HeaderValue, "\r\n") {
			return fmt.Errorf("header value cannot contain newlines")
		}
		// Validate header name is not a forbidden header
		forbidden := []string{"host", "content-length", "transfer-encoding", "connection", "upgrade"}
		for _, f := range forbidden {
			if strings.EqualFold(auth.HeaderName, f) {
				return fmt.Errorf("cannot set forbidden header: %s", auth.HeaderName)
			}
		}
	}
	return nil
}

// injectBackendAuth adds authentication headers to the backend request if configured
func (c *Client) injectBackendAuth(req *http.Request, backend BackendConfig) error {
	// Check if backend has authentication configured
	authBackend, ok := backend.(authProvider)
	if !ok {
		return nil // Backend doesn't support authentication
	}

	auth := authBackend.GetAuth()
	if auth == nil {
		return nil // No auth configured
	}

	// Empty auth type means no authentication
	if auth.Type == "" {
		return nil
	}

	// Validate credentials
	if err := validateAuthCredentials(auth); err != nil {
		return fmt.Errorf("invalid backend auth configuration for %s: %w", backend.GetName(), err)
	}

	var injectedAuthType string

	switch strings.ToLower(auth.Type) {
	case "basic":
		// Basic authentication with username and password
		req.SetBasicAuth(auth.Username, auth.Password)
		injectedAuthType = "basic"
	case "bearer":
		// Bearer token authentication
		req.Header.Set("Authorization", "Bearer "+auth.Token)
		injectedAuthType = "bearer"
	case "header":
		// Custom header authentication
		req.Header.Set(auth.HeaderName, auth.HeaderValue)
		injectedAuthType = "header"
	default:
		return fmt.Errorf("unsupported auth type: %s", auth.Type)
	}

	// Log once after successful injection
	c.logger.Debug().
		Str("backend", backend.GetName()).
		Str("auth_type", injectedAuthType).
		Msg("Injected backend authentication")

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
