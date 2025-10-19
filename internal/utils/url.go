package utils

import (
	"fmt"
	"net/http"
	"strings"
)

// GetExternalURL extracts the external URL from the request
// Priority:
// 1. Forwarded header (RFC 7239)
// 2. X-Forwarded-Proto + X-Forwarded-Host
// 3. X-Forwarded-Host (assumes https)
// 4. Host header (uses request scheme)
func GetExternalURL(r *http.Request) string {
	// Check Forwarded header (RFC 7239) - most standard
	if forwarded := r.Header.Get("Forwarded"); forwarded != "" {
		proto, host := parseForwardedHeader(forwarded)
		if host != "" {
			if proto == "" {
				proto = "https" // Default to https for forwarded requests
			}
			return fmt.Sprintf("%s://%s", proto, host)
		}
	}

	// Check X-Forwarded-Proto and X-Forwarded-Host
	xForwardedProto := r.Header.Get("X-Forwarded-Proto")
	xForwardedHost := r.Header.Get("X-Forwarded-Host")

	if xForwardedHost != "" {
		proto := xForwardedProto
		if proto == "" {
			proto = "https" // Default to https for proxied requests
		}
		return fmt.Sprintf("%s://%s", proto, xForwardedHost)
	}

	// Fallback to Host header
	host := r.Host
	if host != "" {
		// Determine scheme from TLS
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		// Override with X-Forwarded-Proto if available
		if xForwardedProto != "" {
			scheme = xForwardedProto
		}
		return fmt.Sprintf("%s://%s", scheme, host)
	}

	return ""
}

// parseForwardedHeader parses the Forwarded header (RFC 7239)
// Example: Forwarded: for=192.0.2.60;proto=https;host=example.com
func parseForwardedHeader(forwarded string) (proto, host string) {
	// Split by semicolon to get individual parameters
	parts := strings.Split(forwarded, ";")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(kv[0]))
		value := strings.Trim(strings.TrimSpace(kv[1]), "\"")

		switch key {
		case "proto":
			proto = value
		case "host":
			host = value
		}
	}

	return proto, host
}

// GetExternalURLOrDefault returns the external URL from config or auto-detects from request
func GetExternalURLOrDefault(configuredURL string, r *http.Request) string {
	if configuredURL != "" {
		return configuredURL
	}
	return GetExternalURL(r)
}
