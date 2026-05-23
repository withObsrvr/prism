package handlers

import (
	"log/slog"
	"net/http"

	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/templates/fragments"
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

// networkFromRequest reads the network query param first, then the prism_network cookie.
// Defaults to "mainnet" if neither value is present or valid.
func networkFromRequest(r *http.Request) string {
	switch r.URL.Query().Get("network") {
	case "mainnet", "testnet", "futurenet":
		return r.URL.Query().Get("network")
	}
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

// useLiveData returns true if this request should attempt live gateway data.
// Returns false (use mock) when:
//   - Gateway client is nil (no API key configured)
//   - Request has ?mock=true query param (manual override for testing)
//
// Usage in fragment handlers:
//
//	if h.useLiveData(r) {
//	    data, err := h.buildXxxData(r, network)
//	    if err == nil { /* render live */ return }
//	    h.Logger.Warn("live data failed, falling back to mock", "error", err)
//	}
//	/* render mock */
func (h *Handlers) useLiveData(r *http.Request) bool {
	if h.Gateway == nil {
		return false
	}
	if r.URL.Query().Get("mock") == "true" {
		return false
	}
	return true
}

// renderFragmentError logs the primary error, sets a 500 status, and renders
// an inline error component with a retry button. If the error template itself
// fails to render, that is also logged.
func (h *Handlers) renderFragmentError(w http.ResponseWriter, r *http.Request, msg string, err error) {
	h.Logger.Error("render fragment", "error", err, "path", r.URL.Path)
	w.WriteHeader(http.StatusInternalServerError)
	if err2 := fragments.FragmentError(msg, r.URL.Path).Render(r.Context(), w); err2 != nil {
		h.Logger.Error("render fragment error fallback", "error", err2)
	}
}

// Healthz is a simple health check for Nomad/load balancers.
func (h *Handlers) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
