package handler

import (
	"net/http"
)

// Handler is an interface for protocol-specific handlers
type Handler interface {
	// ServeHTTP handles HTTP requests for this protocol
	ServeHTTP(w http.ResponseWriter, r *http.Request)

	// Name returns the handler name
	Name() string
}
