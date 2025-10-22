package detector

import (
	"net/http"
)

// Protocol represents the detected protocol type
type Protocol string

const (
	ProtocolOCI     Protocol = "oci"
	ProtocolMaven   Protocol = "maven"
	ProtocolNPM     Protocol = "npm"
	ProtocolUnknown Protocol = "unknown"
)

// Detector is an interface for protocol detection
type Detector interface {
	// Detect checks if the request matches this protocol
	// Returns true if the protocol is detected
	Detect(r *http.Request) bool

	// Protocol returns the protocol name
	Protocol() Protocol

	// Priority returns the detection priority (higher = checked first)
	Priority() int
}

// Chain manages a chain of protocol detectors
type Chain struct {
	detectors []Detector
}

// NewChain creates a new detector chain
func NewChain(detectors ...Detector) *Chain {
	return &Chain{
		detectors: detectors,
	}
}

// Detect runs all detectors in priority order and returns the first match
func (c *Chain) Detect(r *http.Request) Protocol {
	// Sort by priority (already sorted when added if using Register)
	for _, detector := range c.detectors {
		if detector.Detect(r) {
			return detector.Protocol()
		}
	}

	return ProtocolUnknown
}

// Register adds a detector to the chain in priority order
func (c *Chain) Register(detector Detector) {
	// Insert detector in priority order (highest first)
	inserted := false
	for i, existing := range c.detectors {
		if detector.Priority() > existing.Priority() {
			// Insert before this detector
			c.detectors = append(c.detectors[:i], append([]Detector{detector}, c.detectors[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		// Append at the end
		c.detectors = append(c.detectors, detector)
	}
}

// Detectors returns all registered detectors
func (c *Chain) Detectors() []Detector {
	return c.detectors
}
