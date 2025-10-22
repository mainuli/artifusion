package detector

import (
	"net/http"
	"strings"
)

// getRequestHost extracts the original host considering reverse proxy headers.
// This is critical for host-based routing when Artifusion is deployed behind
// a reverse proxy (nginx, Traefik, Caddy, ALB, etc.).
//
// Checks headers in priority order per RFC 7239 and de facto standards:
//  1. Forwarded (RFC 7239 standard)
//  2. X-Forwarded-Host (de facto standard, widely supported)
//  3. Host (fallback for direct connections)
//
// Examples:
//   - Forwarded: for=client;host=npm.example.com;proto=https
//   - X-Forwarded-Host: npm.example.com
//   - X-Forwarded-Host: npm.example.com, proxy1.internal (takes first)
//   - Host: npm.example.com:443
func getRequestHost(r *http.Request) string {
	// 1. Check RFC 7239 Forwarded header (modern standard)
	// Format: Forwarded: for=192.0.2.60;proto=https;host=example.com
	if fwd := r.Header.Get("Forwarded"); fwd != "" {
		// Parse semicolon-separated parameters
		for _, pair := range strings.Split(fwd, ";") {
			pair = strings.TrimSpace(pair)
			if strings.HasPrefix(pair, "host=") {
				host := strings.TrimPrefix(pair, "host=")
				// Remove quotes if present (RFC 7239 allows quoted values)
				return strings.Trim(host, `"`)
			}
		}
	}

	// 2. Check X-Forwarded-Host (de facto standard)
	// May contain multiple hosts in a chain: "original, proxy1, proxy2"
	// We want the first (original) host
	if xfh := r.Header.Get("X-Forwarded-Host"); xfh != "" {
		hosts := strings.Split(xfh, ",")
		if len(hosts) > 0 {
			return strings.TrimSpace(hosts[0])
		}
	}

	// 3. Fallback to Host header (direct connection or no proxy headers)
	return r.Host
}

// getRequestScheme extracts the original scheme (http/https) considering reverse proxy headers.
// This is critical for constructing absolute URLs when Artifusion is deployed behind
// a reverse proxy that terminates TLS.
//
// Checks in priority order per RFC 7239 and de facto standards:
//  1. Forwarded proto parameter (RFC 7239 standard)
//  2. X-Forwarded-Proto (de facto standard, widely supported)
//  3. TLS connection state (direct HTTPS connection)
//  4. Defaults to https (secure by default)
//
// Examples:
//   - Forwarded: for=client;proto=https;host=example.com
//   - X-Forwarded-Proto: https
//   - X-Forwarded-Proto: https, http (takes first - original client scheme)
//   - TLS connection: https
//   - Fallback: https
func getRequestScheme(r *http.Request) string {
	// 1. Check RFC 7239 Forwarded header (modern standard)
	// Format: Forwarded: for=192.0.2.60;proto=https;host=example.com
	if fwd := r.Header.Get("Forwarded"); fwd != "" {
		// Parse semicolon-separated parameters
		for _, pair := range strings.Split(fwd, ";") {
			pair = strings.TrimSpace(pair)
			if strings.HasPrefix(pair, "proto=") {
				proto := strings.TrimPrefix(pair, "proto=")
				// Remove quotes if present (RFC 7239 allows quoted values)
				proto = strings.Trim(proto, `"`)
				// Validate scheme
				if proto == "http" || proto == "https" {
					return proto
				}
			}
		}
	}

	// 2. Check X-Forwarded-Proto (de facto standard)
	// May contain multiple schemes in a chain: "https, http, http"
	// We want the first (original client) scheme
	if xfp := r.Header.Get("X-Forwarded-Proto"); xfp != "" {
		schemes := strings.Split(xfp, ",")
		if len(schemes) > 0 {
			scheme := strings.TrimSpace(schemes[0])
			// Validate scheme
			if scheme == "http" || scheme == "https" {
				return scheme
			}
		}
	}

	// 3. Check TLS connection state (direct HTTPS connection)
	if r.TLS != nil {
		return "https"
	}

	// 4. Default to https (secure by default)
	// This is safer than defaulting to http
	return "https"
}

// GetRequestHost is the exported version of getRequestHost for use by handlers.
// See getRequestHost documentation for details.
func GetRequestHost(r *http.Request) string {
	return getRequestHost(r)
}

// GetRequestScheme is the exported version of getRequestScheme for use by handlers.
// See getRequestScheme documentation for details.
func GetRequestScheme(r *http.Request) string {
	return getRequestScheme(r)
}
