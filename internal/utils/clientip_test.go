package utils

import (
	"net/http"
	"testing"
)

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name         string
		setupRequest func(*http.Request)
		expectedIP   string
		remoteAddr   string
	}{
		{
			name: "extracts IP from X-Forwarded-For (single IP)",
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "203.0.113.1")
			},
			expectedIP: "203.0.113.1",
			remoteAddr: "192.0.2.1:54321",
		},
		{
			name: "extracts leftmost IP from X-Forwarded-For chain (original client)",
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1, 192.0.2.1")
			},
			expectedIP: "203.0.113.1",
			remoteAddr: "192.0.2.1:54321",
		},
		{
			name: "handles X-Forwarded-For with spaces",
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "  203.0.113.1  ,  198.51.100.1  ")
			},
			expectedIP: "203.0.113.1",
			remoteAddr: "192.0.2.1:54321",
		},
		{
			name: "extracts IP from X-Real-IP",
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-Real-IP", "203.0.113.1")
			},
			expectedIP: "203.0.113.1",
			remoteAddr: "192.0.2.1:54321",
		},
		{
			name: "extracts IP from Forwarded header (RFC 7239)",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Forwarded", "for=203.0.113.1;proto=https;host=example.com")
			},
			expectedIP: "203.0.113.1",
			remoteAddr: "192.0.2.1:54321",
		},
		{
			name: "extracts IP from Forwarded header with quoted value",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Forwarded", `for="203.0.113.1";proto=https`)
			},
			expectedIP: "203.0.113.1",
			remoteAddr: "192.0.2.1:54321",
		},
		{
			name: "extracts IP from Forwarded header with mixed parameter order",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Forwarded", "proto=https;host=example.com;for=203.0.113.1")
			},
			expectedIP: "203.0.113.1",
			remoteAddr: "192.0.2.1:54321",
		},
		{
			name: "handles IPv6 in X-Forwarded-For",
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "2001:db8::1")
			},
			expectedIP: "2001:db8::1",
			remoteAddr: "[2001:db8::2]:54321",
		},
		{
			name: "handles IPv6 with brackets in Forwarded header",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Forwarded", "for=\"[2001:db8::1]\"")
			},
			expectedIP: "2001:db8::1",
			remoteAddr: "[2001:db8::2]:54321",
		},
		{
			name: "X-Forwarded-For takes precedence over X-Real-IP",
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "203.0.113.1")
				r.Header.Set("X-Real-IP", "203.0.113.99")
			},
			expectedIP: "203.0.113.1",
			remoteAddr: "192.0.2.1:54321",
		},
		{
			name: "X-Forwarded-For takes precedence over Forwarded",
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "203.0.113.1")
				r.Header.Set("Forwarded", "for=203.0.113.99")
			},
			expectedIP: "203.0.113.1",
			remoteAddr: "192.0.2.1:54321",
		},
		{
			name: "X-Real-IP takes precedence over Forwarded",
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-Real-IP", "203.0.113.1")
				r.Header.Set("Forwarded", "for=203.0.113.99")
			},
			expectedIP: "203.0.113.1",
			remoteAddr: "192.0.2.1:54321",
		},
		{
			name: "falls back to RemoteAddr when no proxy headers",
			setupRequest: func(r *http.Request) {
				// No headers set
			},
			expectedIP: "192.0.2.1",
			remoteAddr: "192.0.2.1:54321",
		},
		{
			name: "strips port from RemoteAddr",
			setupRequest: func(r *http.Request) {
				// No headers set
			},
			expectedIP: "203.0.113.1",
			remoteAddr: "203.0.113.1:12345",
		},
		{
			name: "handles RemoteAddr without port",
			setupRequest: func(r *http.Request) {
				// No headers set
			},
			expectedIP: "203.0.113.1",
			remoteAddr: "203.0.113.1",
		},
		{
			name: "handles IPv6 RemoteAddr with port",
			setupRequest: func(r *http.Request) {
				// No headers set
			},
			expectedIP: "2001:db8::1",
			remoteAddr: "[2001:db8::1]:54321",
		},
		{
			name: "ignores empty X-Forwarded-For and falls back",
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "")
				r.Header.Set("X-Real-IP", "203.0.113.1")
			},
			expectedIP: "203.0.113.1",
			remoteAddr: "192.0.2.1:54321",
		},
		{
			name: "ignores invalid IP in X-Forwarded-For and falls back",
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "invalid-ip")
				r.Header.Set("X-Real-IP", "203.0.113.1")
			},
			expectedIP: "203.0.113.1",
			remoteAddr: "192.0.2.1:54321",
		},
		{
			name: "ignores invalid IP in X-Real-IP and falls back",
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-Real-IP", "not-an-ip")
				r.Header.Set("Forwarded", "for=203.0.113.1")
			},
			expectedIP: "203.0.113.1",
			remoteAddr: "192.0.2.1:54321",
		},
		{
			name: "handles Forwarded without for parameter",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Forwarded", "proto=https;host=example.com")
			},
			expectedIP: "192.0.2.1",
			remoteAddr: "192.0.2.1:54321",
		},
		{
			name: "handles Forwarded with port in for parameter",
			setupRequest: func(r *http.Request) {
				r.Header.Set("Forwarded", "for=203.0.113.1:8080")
			},
			expectedIP: "203.0.113.1",
			remoteAddr: "192.0.2.1:54321",
		},
		{
			name: "handles X-Forwarded-For chain with port numbers",
			setupRequest: func(r *http.Request) {
				// Some proxies might include ports in the chain
				r.Header.Set("X-Forwarded-For", "203.0.113.1:8080, 198.51.100.1")
			},
			expectedIP: "203.0.113.1",
			remoteAddr: "192.0.2.1:54321",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			req.RemoteAddr = tt.remoteAddr
			tt.setupRequest(req)

			got := GetClientIP(req)
			if got != tt.expectedIP {
				t.Errorf("GetClientIP() = %v, want %v", got, tt.expectedIP)
			}
		})
	}
}

func TestParseIP(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid IPv4", "203.0.113.1", "203.0.113.1"},
		{"valid IPv4 with spaces", "  203.0.113.1  ", "203.0.113.1"},
		{"valid IPv6", "2001:db8::1", "2001:db8::1"},
		{"IPv4 with port", "203.0.113.1:8080", "203.0.113.1"},
		{"IPv6 with port", "[2001:db8::1]:8080", "2001:db8::1"},
		{"invalid IP", "not-an-ip", ""},
		{"empty string", "", ""},
		{"localhost", "localhost", ""}, // Invalid - we want IPs only
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseIP(tt.input)
			if got != tt.expected {
				t.Errorf("parseIP(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestStripPort(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"IPv4 with port", "203.0.113.1:8080", "203.0.113.1"},
		{"IPv4 without port", "203.0.113.1", "203.0.113.1"},
		{"IPv6 with port", "[2001:db8::1]:8080", "2001:db8::1"},
		{"IPv6 without port", "2001:db8::1", "2001:db8::1"},
		{"hostname with port", "example.com:8080", "example.com"},
		{"hostname without port", "example.com", "example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripPort(tt.input)
			if got != tt.expected {
				t.Errorf("stripPort(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseForwardedForIP(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple for parameter",
			input:    "for=203.0.113.1",
			expected: "203.0.113.1",
		},
		{
			name:     "for with other parameters",
			input:    "for=203.0.113.1;proto=https;host=example.com",
			expected: "203.0.113.1",
		},
		{
			name:     "quoted for value",
			input:    `for="203.0.113.1"`,
			expected: "203.0.113.1",
		},
		{
			name:     "IPv6 with brackets",
			input:    `for="[2001:db8::1]"`,
			expected: "2001:db8::1",
		},
		{
			name:     "for with port",
			input:    "for=203.0.113.1:8080",
			expected: "203.0.113.1",
		},
		{
			name:     "mixed parameter order",
			input:    "proto=https;for=203.0.113.1;host=example.com",
			expected: "203.0.113.1",
		},
		{
			name:     "no for parameter",
			input:    "proto=https;host=example.com",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "invalid IP in for",
			input:    "for=not-an-ip",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseForwardedForIP(tt.input)
			if got != tt.expected {
				t.Errorf("parseForwardedForIP(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
