package detector

import (
	"net/http"
	"strings"
)

// MavenDetector detects Maven repository protocol requests
type MavenDetector struct {
	host       string
	pathPrefix string
}

// NewMavenDetector creates a new Maven detector
// host: optional domain for host-based routing (e.g., "maven.example.com")
// pathPrefix: path prefix for path-based routing - required when host is empty
func NewMavenDetector(host, pathPrefix string) *MavenDetector {
	// Normalize pathPrefix: ensure starts with /, no trailing /
	// SECURITY: No silent defaults - pathPrefix must be explicit from config
	if pathPrefix != "" {
		if !strings.HasPrefix(pathPrefix, "/") {
			pathPrefix = "/" + pathPrefix
		}
		pathPrefix = strings.TrimSuffix(pathPrefix, "/")
	}

	return &MavenDetector{
		host:       host,
		pathPrefix: pathPrefix,
	}
}

// Detect checks if the request is a Maven repository request
func (d *MavenDetector) Detect(r *http.Request) bool {
	// Check 0: Host matching (if configured)
	if d.host != "" {
		requestHost := getRequestHost(r)
		if requestHost != d.host {
			return false
		}
	}

	path := r.URL.Path

	// Check 1: Path prefix matching (if configured)
	if d.pathPrefix != "" {
		if !strings.HasPrefix(path, d.pathPrefix+"/") && path != d.pathPrefix {
			// Path doesn't match prefix
			return false
		}
		// Path matches prefix - route to this protocol handler
		// The handler will validate the specific request and handle auth
		return true
	}

	// No pathPrefix configured - use protocol-specific detection
	// This handles host-only routing mode

	// Check 2: Maven file extensions
	mavenExtensions := []string{
		".pom",    // Maven POM files
		".jar",    // JAR files
		".war",    // WAR files
		".ear",    // EAR files
		".aar",    // Android Archive
		".zip",    // ZIP archives
		".tar.gz", // Tar archives
		".md5",    // Checksums
		".sha1",   // Checksums
		".sha256", // Checksums
		".sha512", // Checksums
		".asc",    // GPG signatures
	}

	for _, ext := range mavenExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}

	// Check 2: Maven metadata files
	if strings.Contains(path, "maven-metadata.xml") {
		return true
	}

	// Check 3: Maven repository path structure
	// Maven uses: /<group-path>/<artifact>/<version>/<artifact-version>.<ext>
	// Example: /com/example/myapp/1.0.0/myapp-1.0.0.jar
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 4 {
		// Check if it looks like Maven structure
		// Last part should be artifact-version.ext
		// Second to last should be version
		if len(parts) >= 4 {
			version := parts[len(parts)-2]
			filename := parts[len(parts)-1]

			// Check if filename contains version
			if strings.Contains(filename, version) {
				return true
			}
		}
	}

	// Check 4: Content-Type header
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/x-maven-pom+xml") ||
		strings.Contains(contentType, "application/java-archive") {
		return true
	}

	// Check 5: User-Agent header (Maven tools)
	userAgent := r.Header.Get("User-Agent")
	if strings.Contains(userAgent, "Apache-Maven") ||
		strings.Contains(userAgent, "Gradle") ||
		strings.Contains(userAgent, "sbt") {
		return true
	}

	return false
}

// Protocol returns the protocol name
func (d *MavenDetector) Protocol() Protocol {
	return ProtocolMaven
}

// Priority returns the detection priority (lower than OCI)
func (d *MavenDetector) Priority() int {
	return 90 // Slightly lower priority than OCI
}
