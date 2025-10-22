package detector

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetRequestHost(t *testing.T) {
	tests := []struct {
		name         string
		setupRequest func() *http.Request
		expectedHost string
	}{
		{
			name: "uses Forwarded header (RFC 7239)",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Forwarded", "for=192.0.2.60;proto=https;host=npm.example.com")
				req.Host = "localhost:8080"
				return req
			},
			expectedHost: "npm.example.com",
		},
		{
			name: "uses Forwarded header with quoted host",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Forwarded", `for=192.0.2.60;host="maven.example.com";proto=https`)
				req.Host = "localhost:8080"
				return req
			},
			expectedHost: "maven.example.com",
		},
		{
			name: "uses Forwarded header with mixed order",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Forwarded", "proto=https;host=docker.example.com;for=192.0.2.60")
				req.Host = "localhost:8080"
				return req
			},
			expectedHost: "docker.example.com",
		},
		{
			name: "uses X-Forwarded-Host header",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Forwarded-Host", "npm.example.com")
				req.Host = "localhost:8080"
				return req
			},
			expectedHost: "npm.example.com",
		},
		{
			name: "uses first host from X-Forwarded-Host chain",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Forwarded-Host", "npm.example.com, proxy1.internal, proxy2.internal")
				req.Host = "localhost:8080"
				return req
			},
			expectedHost: "npm.example.com",
		},
		{
			name: "uses X-Forwarded-Host with spaces",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Forwarded-Host", "  maven.example.com  ,  proxy.internal  ")
				req.Host = "localhost:8080"
				return req
			},
			expectedHost: "maven.example.com",
		},
		{
			name: "falls back to Host header when no proxy headers",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Host = "direct.example.com:8080"
				return req
			},
			expectedHost: "direct.example.com:8080",
		},
		{
			name: "Forwarded takes precedence over X-Forwarded-Host",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Forwarded", "host=forwarded.example.com")
				req.Header.Set("X-Forwarded-Host", "x-forwarded.example.com")
				req.Host = "localhost:8080"
				return req
			},
			expectedHost: "forwarded.example.com",
		},
		{
			name: "X-Forwarded-Host takes precedence over Host",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Forwarded-Host", "x-forwarded.example.com")
				req.Host = "localhost:8080"
				return req
			},
			expectedHost: "x-forwarded.example.com",
		},
		{
			name: "handles empty Forwarded header gracefully",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Forwarded", "")
				req.Header.Set("X-Forwarded-Host", "backup.example.com")
				req.Host = "localhost:8080"
				return req
			},
			expectedHost: "backup.example.com",
		},
		{
			name: "handles Forwarded without host parameter",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Forwarded", "for=192.0.2.60;proto=https")
				req.Header.Set("X-Forwarded-Host", "backup.example.com")
				req.Host = "localhost:8080"
				return req
			},
			expectedHost: "backup.example.com",
		},
		{
			name: "handles empty X-Forwarded-Host gracefully",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Forwarded-Host", "")
				req.Host = "fallback.example.com"
				return req
			},
			expectedHost: "fallback.example.com",
		},
		{
			name: "handles host with port in Forwarded",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Forwarded", "host=npm.example.com:443")
				req.Host = "localhost:8080"
				return req
			},
			expectedHost: "npm.example.com:443",
		},
		{
			name: "handles host with port in X-Forwarded-Host",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Forwarded-Host", "maven.example.com:8443")
				req.Host = "localhost:8080"
				return req
			},
			expectedHost: "maven.example.com:8443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			actualHost := getRequestHost(req)

			if actualHost != tt.expectedHost {
				t.Errorf("getRequestHost() = %q, want %q", actualHost, tt.expectedHost)
			}
		})
	}
}

func TestGetRequestScheme(t *testing.T) {
	tests := []struct {
		name           string
		setupRequest   func() *http.Request
		expectedScheme string
	}{
		{
			name: "uses Forwarded proto parameter (RFC 7239)",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Forwarded", "for=192.0.2.60;proto=https;host=example.com")
				return req
			},
			expectedScheme: "https",
		},
		{
			name: "uses Forwarded proto=http",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Forwarded", "proto=http;host=example.com")
				return req
			},
			expectedScheme: "http",
		},
		{
			name: "uses Forwarded proto with quotes",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Forwarded", `proto="https";host=example.com`)
				return req
			},
			expectedScheme: "https",
		},
		{
			name: "uses Forwarded proto with mixed order",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Forwarded", "host=example.com;proto=https;for=client")
				return req
			},
			expectedScheme: "https",
		},
		{
			name: "uses X-Forwarded-Proto header",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Forwarded-Proto", "https")
				return req
			},
			expectedScheme: "https",
		},
		{
			name: "uses X-Forwarded-Proto=http",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Forwarded-Proto", "http")
				return req
			},
			expectedScheme: "http",
		},
		{
			name: "uses first scheme from X-Forwarded-Proto chain",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Forwarded-Proto", "https, http, http")
				return req
			},
			expectedScheme: "https",
		},
		{
			name: "uses X-Forwarded-Proto with spaces",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Forwarded-Proto", "  http  ,  https  ")
				return req
			},
			expectedScheme: "http",
		},
		{
			name: "defaults to https when no headers (secure by default)",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				return req
			},
			expectedScheme: "https",
		},
		{
			name: "Forwarded takes precedence over X-Forwarded-Proto",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Forwarded", "proto=https")
				req.Header.Set("X-Forwarded-Proto", "http")
				return req
			},
			expectedScheme: "https",
		},
		{
			name: "X-Forwarded-Proto takes precedence over default",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Forwarded-Proto", "http")
				return req
			},
			expectedScheme: "http",
		},
		{
			name: "handles empty Forwarded header gracefully",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Forwarded", "")
				req.Header.Set("X-Forwarded-Proto", "http")
				return req
			},
			expectedScheme: "http",
		},
		{
			name: "handles Forwarded without proto parameter",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Forwarded", "for=192.0.2.60;host=example.com")
				req.Header.Set("X-Forwarded-Proto", "http")
				return req
			},
			expectedScheme: "http",
		},
		{
			name: "handles empty X-Forwarded-Proto gracefully",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Forwarded-Proto", "")
				return req
			},
			expectedScheme: "https",
		},
		{
			name: "ignores invalid scheme in Forwarded",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Forwarded", "proto=ftp")
				req.Header.Set("X-Forwarded-Proto", "http")
				return req
			},
			expectedScheme: "http",
		},
		{
			name: "ignores invalid scheme in X-Forwarded-Proto",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Forwarded-Proto", "wss")
				return req
			},
			expectedScheme: "https",
		},
		{
			name: "case sensitive scheme validation (lowercase https)",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Forwarded-Proto", "https")
				return req
			},
			expectedScheme: "https",
		},
		{
			name: "rejects uppercase HTTPS (scheme must be lowercase)",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("X-Forwarded-Proto", "HTTPS")
				return req
			},
			expectedScheme: "https", // Falls back to default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			actualScheme := getRequestScheme(req)

			if actualScheme != tt.expectedScheme {
				t.Errorf("getRequestScheme() = %q, want %q", actualScheme, tt.expectedScheme)
			}
		})
	}
}
