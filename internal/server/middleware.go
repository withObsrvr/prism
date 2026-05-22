package server

import (
	"fmt"
	"net/http"
	"time"
)

// logRequest logs every incoming request with method, URI, and duration.
// From "Let's Go" Chapter 6.
func (app *Application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the response writer to capture the status code.
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		app.Logger.Info("request",
			"method", r.Method,
			"uri", r.URL.RequestURI(),
			"status", sw.status,
			"duration", time.Since(start),
			"remote", r.RemoteAddr,
		)
	})
}

// recoverPanic catches any panics in downstream handlers,
// logs the error, and returns a 500 to the client.
// From "Let's Go" Chapter 6.
func (app *Application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.Logger.Error("panic recovered",
					"error", fmt.Sprintf("%s", err),
					"uri", r.URL.RequestURI(),
				)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (sw *statusWriter) WriteHeader(code int) {
	if sw.wrote {
		return
	}
	sw.wrote = true
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *statusWriter) Write(b []byte) (int, error) {
	if !sw.wrote {
		sw.WriteHeader(sw.status)
	}
	return sw.ResponseWriter.Write(b)
}
