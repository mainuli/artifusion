package utils

import (
	"net"
	"net/http"
	"strings"
)

// GetClientIP extracts the real client IP from HTTP request headers when behind a proxy.
// Priority order:
//  1. X-Forwarded-For (leftmost IP = original client)
//  2. X-Real-IP
//  3. Forwarded header (RFC 7239, for= parameter)
//  4. RemoteAddr (direct connection, fallback)
//
// Returns the client IP address as a string. For X-Forwarded-For chains,
// returns the leftmost (original client) IP, not the proxy chain IPs.
//
// Examples:
//   - X-Forwarded-For: "203.0.113.1, 198.51.100.1, 192.0.2.1" → "203.0.113.1"
//   - X-Real-IP: "203.0.113.1" → "203.0.113.1"
//   - Forwarded: "for=203.0.113.1;proto=https" → "203.0.113.1"
//   - RemoteAddr: "203.0.113.1:54321" → "203.0.113.1"
func GetClientIP(r *http.Request) string {
	// 1. Check X-Forwarded-For (most common proxy header)
	// Format: X-Forwarded-For: client, proxy1, proxy2
	// Take leftmost IP (original client)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Split by comma and take first IP
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			clientIP := strings.TrimSpace(ips[0])
			if ip := parseIP(clientIP); ip != "" {
				return ip
			}
		}
	}

	// 2. Check X-Real-IP (used by some proxies like nginx)
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if ip := parseIP(xri); ip != "" {
			return ip
		}
	}

	// 3. Check RFC 7239 Forwarded header (modern standard)
	// Format: Forwarded: for=192.0.2.60;proto=https;host=example.com
	if fwd := r.Header.Get("Forwarded"); fwd != "" {
		if forIP := parseForwardedForIP(fwd); forIP != "" {
			return forIP
		}
	}

	// 4. Fallback to RemoteAddr (direct connection IP)
	// RemoteAddr includes port, so strip it
	return stripPort(r.RemoteAddr)
}

// parseIP validates and normalizes an IP address
// Handles both IPv4 and IPv6, strips port if present
// Returns empty string if invalid
func parseIP(ipStr string) string {
	ipStr = strings.TrimSpace(ipStr)
	if ipStr == "" {
		return ""
	}

	// Strip port if present (handles both IPv4:port and [IPv6]:port)
	ip := stripPort(ipStr)

	// Validate it's a valid IP
	if net.ParseIP(ip) != nil {
		return ip
	}

	return ""
}

// stripPort removes the port from an IP address string
// Handles both IPv4 (192.0.2.1:8080) and IPv6 ([2001:db8::1]:8080)
func stripPort(hostPort string) string {
	// net.SplitHostPort handles both IPv4:port and [IPv6]:port
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		// No port present, return as-is
		return hostPort
	}
	return host
}

// parseForwardedForIP extracts the IP from the "for=" parameter in Forwarded header
// Example: Forwarded: for=192.0.2.60;proto=https;host=example.com
// Returns the IP address or empty string if not found/invalid
func parseForwardedForIP(forwarded string) string {
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

		if key == "for" {
			// Value might be IP:port or just IP
			// Can also be quoted or contain IPv6 brackets
			// Examples: for=192.0.2.60, for="[2001:db8::1]", for="192.0.2.60:8080"
			value = strings.Trim(value, "[]") // Remove IPv6 brackets if present
			return parseIP(value)
		}
	}

	return ""
}
