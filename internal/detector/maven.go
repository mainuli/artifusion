package detector

import (
	"net/http"
	"strings"
)

// MavenDetector detects Maven repository protocol requests
type MavenDetector struct {
	pathPrefix string
}

// NewMavenDetector creates a new Maven detector
func NewMavenDetector(pathPrefix string) *MavenDetector {
	// Ensure pathPrefix starts with / and doesn't end with /
	if pathPrefix == "" {
		pathPrefix = "/m2"
	}
	if !strings.HasPrefix(pathPrefix, "/") {
		pathPrefix = "/" + pathPrefix
	}
	pathPrefix = strings.TrimSuffix(pathPrefix, "/")

	return &MavenDetector{
		pathPrefix: pathPrefix,
	}
}

// Detect checks if the request is a Maven repository request
func (d *MavenDetector) Detect(r *http.Request) bool {
	path := r.URL.Path

	// Check 0: Path prefix matching (e.g., /m2/*)
	if strings.HasPrefix(path, d.pathPrefix+"/") || path == d.pathPrefix {
		return true
	}

	// Check 1: Maven file extensions
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
