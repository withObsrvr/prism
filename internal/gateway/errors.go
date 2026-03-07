package gateway

import "fmt"

// APIError represents an HTTP error response from the gateway.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("gateway: HTTP %d: %s", e.StatusCode, e.Message)
}
