package middleware

import (
	"net/http"
	"reflect"
	"testing"
)

func TestSanitizeHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    http.Header
		expected map[string]interface{}
	}{
		{
			name: "redacts Authorization header",
			input: http.Header{
				"Authorization": []string{"Bearer ghp_1234567890abcdef"},
				"Content-Type":  []string{"application/json"},
			},
			expected: map[string]interface{}{
				"Authorization": "[REDACTED]",
				"Content-Type":  []string{"application/json"},
			},
		},
		{
			name: "redacts Cookie header",
			input: http.Header{
				"Cookie":       []string{"session=abc123; auth=xyz789"},
				"User-Agent":   []string{"Mozilla/5.0"},
				"Content-Type": []string{"text/html"},
			},
			expected: map[string]interface{}{
				"Cookie":       "[REDACTED]",
				"User-Agent":   []string{"Mozilla/5.0"},
				"Content-Type": []string{"text/html"},
			},
		},
		{
			name: "redacts Set-Cookie header",
			input: http.Header{
				"Set-Cookie":   []string{"session=abc123; HttpOnly"},
				"Content-Type": []string{"application/json"},
			},
			expected: map[string]interface{}{
				"Set-Cookie":   "[REDACTED]",
				"Content-Type": []string{"application/json"},
			},
		},
		{
			name: "redacts X-Auth-Token header",
			input: http.Header{
				"X-Auth-Token": []string{"secret-token-12345"},
				"Accept":       []string{"application/json"},
			},
			expected: map[string]interface{}{
				"X-Auth-Token": "[REDACTED]",
				"Accept":       []string{"application/json"},
			},
		},
		{
			name: "redacts X-Api-Key header",
			input: http.Header{
				"X-Api-Key":  []string{"api-key-secret"},
				"User-Agent": []string{"test-client"},
			},
			expected: map[string]interface{}{
				"X-Api-Key":  "[REDACTED]",
				"User-Agent": []string{"test-client"},
			},
		},
		{
			name: "redacts Proxy-Authorization header",
			input: http.Header{
				"Proxy-Authorization": []string{"Basic abc123"},
				"Host":                []string{"example.com"},
			},
			expected: map[string]interface{}{
				"Proxy-Authorization": "[REDACTED]",
				"Host":                []string{"example.com"},
			},
		},
		{
			name: "case insensitive matching for sensitive headers",
			input: http.Header{
				"AUTHORIZATION": []string{"Bearer token"},
				"cookie":        []string{"session=123"},
				"X-CSRF-Token":  []string{"csrf-token"},
				"Content-Type":  []string{"text/plain"},
			},
			expected: map[string]interface{}{
				"AUTHORIZATION": "[REDACTED]",
				"cookie":        "[REDACTED]",
				"X-CSRF-Token":  "[REDACTED]",
				"Content-Type":  []string{"text/plain"},
			},
		},
		{
			name: "preserves safe headers",
			input: http.Header{
				"Content-Type":    []string{"application/json"},
				"Content-Length":  []string{"1234"},
				"Accept":          []string{"application/json"},
				"User-Agent":      []string{"test-agent"},
				"Host":            []string{"example.com"},
				"Referer":         []string{"https://example.com"},
				"Accept-Encoding": []string{"gzip, deflate"},
				"Accept-Language": []string{"en-US,en;q=0.9"},
				"Cache-Control":   []string{"no-cache"},
			},
			expected: map[string]interface{}{
				"Content-Type":    []string{"application/json"},
				"Content-Length":  []string{"1234"},
				"Accept":          []string{"application/json"},
				"User-Agent":      []string{"test-agent"},
				"Host":            []string{"example.com"},
				"Referer":         []string{"https://example.com"},
				"Accept-Encoding": []string{"gzip, deflate"},
				"Accept-Language": []string{"en-US,en;q=0.9"},
				"Cache-Control":   []string{"no-cache"},
			},
		},
		{
			name: "redacts multiple sensitive headers in single request",
			input: http.Header{
				"Authorization":   []string{"Bearer token123"},
				"Cookie":          []string{"session=abc"},
				"X-Auth-Token":    []string{"auth-token"},
				"X-Session-Token": []string{"session-token"},
				"Content-Type":    []string{"application/json"},
			},
			expected: map[string]interface{}{
				"Authorization":   "[REDACTED]",
				"Cookie":          "[REDACTED]",
				"X-Auth-Token":    "[REDACTED]",
				"X-Session-Token": "[REDACTED]",
				"Content-Type":    []string{"application/json"},
			},
		},
		{
			name:     "handles empty headers",
			input:    http.Header{},
			expected: map[string]interface{}{},
		},
		{
			name: "handles nil values in header",
			input: http.Header{
				"Authorization": nil,
				"Content-Type":  []string{"text/plain"},
			},
			expected: map[string]interface{}{
				"Authorization": "[REDACTED]",
				"Content-Type":  []string{"text/plain"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeHeaders(tt.input)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("sanitizeHeaders() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestSanitizeHeaders_SecurityVerification ensures critical security headers are always redacted
func TestSanitizeHeaders_SecurityVerification(t *testing.T) {
	// This test explicitly verifies that common authentication/session headers
	// are NEVER leaked into logs, even with creative casing
	criticalHeaders := []string{
		"Authorization",
		"AUTHORIZATION",
		"authorization",
		"Cookie",
		"COOKIE",
		"cookie",
		"Set-Cookie",
		"X-Auth-Token",
		"X-API-Key",
		"x-api-key",
		"Proxy-Authorization",
		"X-CSRF-Token",
		"X-Session-Token",
	}

	for _, headerName := range criticalHeaders {
		t.Run("redacts "+headerName, func(t *testing.T) {
			input := http.Header{
				headerName: []string{"super-secret-value-that-must-not-be-logged"},
			}

			result := sanitizeHeaders(input)

			if value, ok := result[headerName]; ok {
				if value != "[REDACTED]" {
					t.Errorf("SECURITY FAILURE: Header %s was not redacted, got: %v", headerName, value)
				}
			} else {
				t.Errorf("Header %s not found in result", headerName)
			}
		})
	}
}
