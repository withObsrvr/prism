package handlers

import (
	"log/slog"
	"net/http"
)

// Handlers holds shared dependencies for all HTTP handlers.
// Each handler is a method on this struct, receiving dependencies
// via the receiver rather than closures or globals.
type Handlers struct {
	Logger  *slog.Logger
	Network string // "mainnet" or "testnet"
}

// New creates a Handlers instance with all shared dependencies.
func New(logger *slog.Logger, network string) *Handlers {
	return &Handlers{
		Logger:  logger,
		Network: network,
	}
}

// isHTMX checks whether a request was triggered by htmx.
// When true, we return a partial HTML fragment instead of a full page.
func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// Healthz is a simple health check for Nomad/load balancers.
func (h *Handlers) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
