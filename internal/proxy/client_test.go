package proxy

import (
	"net/http"
	"reflect"
	"testing"
)

func TestRemoveHopByHopHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    http.Header
		expected http.Header
	}{
		{
			name: "removes Connection header",
			input: http.Header{
				"Connection":   []string{"close"},
				"Content-Type": []string{"application/json"},
				"X-Custom":     []string{"value"},
			},
			expected: http.Header{
				"Content-Type": []string{"application/json"},
				"X-Custom":     []string{"value"},
			},
		},
		{
			name: "removes Transfer-Encoding header",
			input: http.Header{
				"Transfer-Encoding": []string{"chunked"},
				"Content-Type":      []string{"text/html"},
				"Accept":            []string{"*/*"},
			},
			expected: http.Header{
				"Content-Type": []string{"text/html"},
				"Accept":       []string{"*/*"},
			},
		},
		{
			name: "removes Upgrade header",
			input: http.Header{
				"Upgrade":      []string{"websocket"},
				"Connection":   []string{"Upgrade"},
				"Content-Type": []string{"application/json"},
			},
			expected: http.Header{
				"Content-Type": []string{"application/json"},
			},
		},
		{
			name: "removes TE header",
			input: http.Header{
				"TE":           []string{"trailers"},
				"User-Agent":   []string{"test-client"},
				"Content-Type": []string{"text/plain"},
			},
			expected: http.Header{
				"User-Agent":   []string{"test-client"},
				"Content-Type": []string{"text/plain"},
			},
		},
		{
			name: "removes Trailer header",
			input: http.Header{
				"Trailer":      []string{"X-Checksum"},
				"Content-Type": []string{"application/octet-stream"},
			},
			expected: http.Header{
				"Content-Type": []string{"application/octet-stream"},
			},
		},
		{
			name: "removes Proxy-Connection header (non-standard but common)",
			input: http.Header{
				"Proxy-Connection": []string{"keep-alive"},
				"Host":             []string{"example.com"},
			},
			expected: http.Header{
				"Host": []string{"example.com"},
			},
		},
		{
			name: "removes Keep-Alive header",
			input: http.Header{
				"Keep-Alive":   []string{"timeout=5, max=100"},
				"Content-Type": []string{"application/json"},
			},
			expected: http.Header{
				"Content-Type": []string{"application/json"},
			},
		},
		{
			name: "removes Proxy-Authenticate header",
			input: http.Header{
				"Proxy-Authenticate": []string{"Basic realm=\"proxy\""},
				"Content-Type":       []string{"text/html"},
			},
			expected: http.Header{
				"Content-Type": []string{"text/html"},
			},
		},
		{
			name: "removes Proxy-Authorization header",
			input: http.Header{
				"Proxy-Authorization": []string{"Basic abc123"},
				"Authorization":       []string{"Bearer token"},
				"Content-Type":        []string{"application/json"},
			},
			expected: http.Header{
				"Authorization": []string{"Bearer token"},
				"Content-Type":  []string{"application/json"},
			},
		},
		{
			name: "removes headers specified in Connection (single header)",
			input: http.Header{
				"Connection":   []string{"X-Custom-Hop"},
				"X-Custom-Hop": []string{"value1"},
				"Content-Type": []string{"text/html"},
				"Accept":       []string{"*/*"},
			},
			expected: http.Header{
				"Content-Type": []string{"text/html"},
				"Accept":       []string{"*/*"},
			},
		},
		{
			name: "removes headers specified in Connection (multiple headers comma-separated)",
			input: http.Header{
				"Connection":   []string{"X-Custom-Hop, X-Another, X-Third"},
				"X-Custom-Hop": []string{"value1"},
				"X-Another":    []string{"value2"},
				"X-Third":      []string{"value3"},
				"Content-Type": []string{"application/json"},
			},
			expected: http.Header{
				"Content-Type": []string{"application/json"},
			},
		},
		{
			name: "removes headers specified in Connection (multiple Connection headers)",
			input: http.Header{
				"Connection":   []string{"X-First", "X-Second"},
				"X-First":      []string{"val1"},
				"X-Second":     []string{"val2"},
				"Content-Type": []string{"text/plain"},
			},
			expected: http.Header{
				"Content-Type": []string{"text/plain"},
			},
		},
		{
			name: "case-insensitive matching for standard hop-by-hop headers",
			input: http.Header{
				"CONNECTION":        []string{"close"},
				"transfer-encoding": []string{"chunked"},
				"UPGRADE":           []string{"websocket"},
				"Content-Type":      []string{"text/plain"},
			},
			expected: http.Header{
				"Content-Type": []string{"text/plain"},
			},
		},
		{
			name: "case-insensitive matching for Connection-specified headers",
			input: http.Header{
				"Connection":   []string{"x-custom-hop"},
				"X-Custom-Hop": []string{"value"},
				"Content-Type": []string{"application/json"},
			},
			expected: http.Header{
				"Content-Type": []string{"application/json"},
			},
		},
		{
			name: "preserves safe headers",
			input: http.Header{
				"Content-Type":      []string{"application/json"},
				"Content-Length":    []string{"1234"},
				"Accept":            []string{"application/json"},
				"Accept-Encoding":   []string{"gzip, deflate"},
				"Accept-Language":   []string{"en-US,en;q=0.9"},
				"User-Agent":        []string{"test-agent/1.0"},
				"Host":              []string{"example.com"},
				"Referer":           []string{"https://example.com"},
				"Cache-Control":     []string{"no-cache"},
				"Authorization":     []string{"Bearer token123"},
				"X-Request-ID":      []string{"req-123"},
				"X-Forwarded-For":   []string{"192.168.1.1"},
				"X-Forwarded-Host":  []string{"proxy.com"},
				"X-Forwarded-Proto": []string{"https"},
			},
			expected: http.Header{
				"Content-Type":      []string{"application/json"},
				"Content-Length":    []string{"1234"},
				"Accept":            []string{"application/json"},
				"Accept-Encoding":   []string{"gzip, deflate"},
				"Accept-Language":   []string{"en-US,en;q=0.9"},
				"User-Agent":        []string{"test-agent/1.0"},
				"Host":              []string{"example.com"},
				"Referer":           []string{"https://example.com"},
				"Cache-Control":     []string{"no-cache"},
				"Authorization":     []string{"Bearer token123"},
				"X-Request-ID":      []string{"req-123"},
				"X-Forwarded-For":   []string{"192.168.1.1"},
				"X-Forwarded-Host":  []string{"proxy.com"},
				"X-Forwarded-Proto": []string{"https"},
			},
		},
		{
			name: "removes all standard hop-by-hop headers in one request",
			input: http.Header{
				"Connection":          []string{"close"},
				"Proxy-Connection":    []string{"keep-alive"},
				"Keep-Alive":          []string{"timeout=5"},
				"Proxy-Authenticate":  []string{"Basic"},
				"Proxy-Authorization": []string{"Basic abc"},
				"TE":                  []string{"trailers"},
				"Trailer":             []string{"X-Checksum"},
				"Transfer-Encoding":   []string{"chunked"},
				"Upgrade":             []string{"websocket"},
				"Content-Type":        []string{"application/json"},
				"X-Safe-Header":       []string{"safe-value"},
			},
			expected: http.Header{
				"Content-Type":  []string{"application/json"},
				"X-Safe-Header": []string{"safe-value"},
			},
		},
		{
			name:     "handles empty headers",
			input:    http.Header{},
			expected: http.Header{},
		},
		{
			name: "handles headers with multiple values",
			input: http.Header{
				"Accept":       []string{"application/json", "text/html"},
				"Connection":   []string{"close"},
				"Content-Type": []string{"multipart/form-data; boundary=abc123"},
			},
			expected: http.Header{
				"Accept":       []string{"application/json", "text/html"},
				"Content-Type": []string{"multipart/form-data; boundary=abc123"},
			},
		},
		{
			name: "handles Connection header with extra whitespace",
			input: http.Header{
				"Connection":   []string{" X-First , X-Second  ,  X-Third "},
				"X-First":      []string{"val1"},
				"X-Second":     []string{"val2"},
				"X-Third":      []string{"val3"},
				"Content-Type": []string{"text/plain"},
			},
			expected: http.Header{
				"Content-Type": []string{"text/plain"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeHopByHopHeaders(tt.input)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("removeHopByHopHeaders() mismatch:\nGot:  %v\nWant: %v", result, tt.expected)
			}
		})
	}
}

// TestRemoveHopByHopHeaders_SecurityVerification ensures critical hop-by-hop headers
// are NEVER forwarded, as this would enable HTTP request smuggling attacks
func TestRemoveHopByHopHeaders_SecurityVerification(t *testing.T) {
	criticalHopByHopHeaders := []string{
		"Connection",
		"Proxy-Connection",
		"Keep-Alive",
		"Transfer-Encoding",
		"TE",
		"Trailer",
		"Upgrade",
		"Proxy-Authenticate",
		"Proxy-Authorization",
	}

	for _, headerName := range criticalHopByHopHeaders {
		t.Run("blocks_"+headerName, func(t *testing.T) {
			input := http.Header{
				headerName:     []string{"malicious-value"},
				"Content-Type": []string{"application/json"},
			}

			result := removeHopByHopHeaders(input)

			// Verify the hop-by-hop header was removed
			if _, exists := result[headerName]; exists {
				t.Errorf("SECURITY FAILURE: Hop-by-hop header %s was not removed!", headerName)
			}

			// Verify safe headers were preserved
			if result.Get("Content-Type") != "application/json" {
				t.Error("Safe header Content-Type was not preserved")
			}
		})
	}
}

// TestRemoveHopByHopHeaders_RequestSmugglingPrevention tests specific attack scenarios
func TestRemoveHopByHopHeaders_RequestSmugglingPrevention(t *testing.T) {
	t.Run("blocks Transfer-Encoding smuggling attack", func(t *testing.T) {
		// Attacker tries to smuggle a request using Transfer-Encoding
		input := http.Header{
			"Transfer-Encoding": []string{"chunked"},
			"Content-Length":    []string{"10"},
			"Host":              []string{"backend.com"},
		}

		result := removeHopByHopHeaders(input)

		// Transfer-Encoding must be removed
		if _, exists := result["Transfer-Encoding"]; exists {
			t.Error("Transfer-Encoding was not removed - request smuggling possible!")
		}

		// Content-Length and Host should be preserved
		if result.Get("Content-Length") != "10" {
			t.Error("Content-Length was incorrectly removed")
		}
		if result.Get("Host") != "backend.com" {
			t.Error("Host was incorrectly removed")
		}
	})

	t.Run("blocks Connection header poisoning", func(t *testing.T) {
		// Attacker tries to poison the connection with malicious Connection header
		input := http.Header{
			"Connection":        []string{"transfer-encoding, close"},
			"Transfer-Encoding": []string{"chunked"},
			"Content-Type":      []string{"application/json"},
		}

		result := removeHopByHopHeaders(input)

		// Both Connection and Transfer-Encoding must be removed
		if _, exists := result["Connection"]; exists {
			t.Error("Connection header was not removed")
		}
		if _, exists := result["Transfer-Encoding"]; exists {
			t.Error("Transfer-Encoding was not removed")
		}
	})

	t.Run("blocks TE header desync attack", func(t *testing.T) {
		// Attacker tries to use TE header for request smuggling
		input := http.Header{
			"TE":                []string{"trailers, chunked"},
			"Transfer-Encoding": []string{"chunked"},
			"Host":              []string{"victim.com"},
		}

		result := removeHopByHopHeaders(input)

		// TE and Transfer-Encoding must be removed
		if _, exists := result["TE"]; exists {
			t.Error("TE header was not removed - desync attack possible!")
		}
		if _, exists := result["Transfer-Encoding"]; exists {
			t.Error("Transfer-Encoding was not removed")
		}
	})

	t.Run("blocks Upgrade header smuggling", func(t *testing.T) {
		// Attacker tries to use Upgrade for protocol smuggling
		input := http.Header{
			"Upgrade":      []string{"h2c"},
			"Connection":   []string{"Upgrade, HTTP2-Settings"},
			"Content-Type": []string{"text/html"},
		}

		result := removeHopByHopHeaders(input)

		// Upgrade and Connection must be removed
		if _, exists := result["Upgrade"]; exists {
			t.Error("Upgrade header was not removed - protocol smuggling possible!")
		}
		if _, exists := result["Connection"]; exists {
			t.Error("Connection header was not removed")
		}
	})
}
