package handlers

import (
	"log/slog"
	"net/http"

	"github.com/withObsrvr/prism/internal/gateway"
)

// Handlers holds shared dependencies for all HTTP handlers.
// Each handler is a method on this struct, receiving dependencies
// via the receiver rather than closures or globals.
type Handlers struct {
	Logger  *slog.Logger
	Gateway *gateway.Client
}

// New creates a Handlers instance with all shared dependencies.
func New(logger *slog.Logger, gw *gateway.Client) *Handlers {
	return &Handlers{
		Logger:  logger,
		Gateway: gw,
	}
}

// networkFromRequest reads the prism_network cookie and returns a validated network name.
// Defaults to "mainnet" if the cookie is missing or invalid.
func networkFromRequest(r *http.Request) string {
	cookie, err := r.Cookie("prism_network")
	if err != nil {
		return "mainnet"
	}
	switch cookie.Value {
	case "mainnet", "testnet", "futurenet":
		return cookie.Value
	default:
		return "mainnet"
	}
}

// SetNetwork sets the prism_network cookie.
func (h *Handlers) SetNetwork(w http.ResponseWriter, r *http.Request) {
	network := r.FormValue("network")
	switch network {
	case "mainnet", "testnet", "futurenet":
	default:
		http.Error(w, "invalid network", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "prism_network",
		Value:    network,
		Path:     "/",
		MaxAge:   365 * 24 * 60 * 60,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusOK)
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
