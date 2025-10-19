package middleware

import (
	"net/http"
)

// SecurityHeaders adds security-related HTTP headers to all responses
// This middleware implements defense-in-depth security practices
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		// Instructs browsers to respect the Content-Type header
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking attacks
		// Prevents the page from being embedded in frames/iframes
		w.Header().Set("X-Frame-Options", "DENY")

		// Enable XSS protection in older browsers
		// Modern browsers have this enabled by default, but this ensures compatibility
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Enforce HTTPS for all future requests (HSTS)
		// Only set when request is over HTTPS or behind HTTPS proxy
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			// max-age=31536000: Enforce HTTPS for 1 year
			// includeSubDomains: Apply to all subdomains
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Content Security Policy
		// Restrict resource loading to prevent XSS and data injection attacks
		// default-src 'none': Block all content by default (appropriate for API server)
		// For a more permissive policy, you could use:
		// default-src 'self'; script-src 'self'; style-src 'self'
		w.Header().Set("Content-Security-Policy", "default-src 'none'")

		// Referrer policy
		// Control how much referrer information is sent with requests
		// strict-origin-when-cross-origin: Send full URL for same-origin, origin only for cross-origin HTTPS
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions Policy (formerly Feature Policy)
		// Disable potentially dangerous browser features
		// This prevents the page from accessing camera, microphone, geolocation, etc.
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

		// X-Permitted-Cross-Domain-Policies
		// Prevent Adobe Flash and PDF from loading content from this domain
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}
