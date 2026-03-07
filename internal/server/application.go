package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/withObsrvr/prism/internal/gateway"
)

// Config holds all server configuration, populated by viper.
type Config struct {
	Host       string
	Port       int
	DataSource string // "mock", "gateway", or "auto"
	Gateway    gateway.Config
}

// Application holds all dependencies for the HTTP handlers.
// This is the Alex Edwards pattern from "Let's Go" — all shared
// dependencies live here and are passed to handlers via methods.
type Application struct {
	Logger  *slog.Logger
	Config  Config
	Gateway *gateway.Client
}

// New creates a new Application with all dependencies wired up.
func New(logger *slog.Logger, cfg Config, ctx context.Context) (*Application, error) {
	var gw *gateway.Client

	switch cfg.DataSource {
	case "mock":
		logger.Info("data source set to mock, gateway disabled")
	case "gateway":
		if cfg.Gateway.APIKey == "" {
			return nil, fmt.Errorf("PRISM_DATA_SOURCE=gateway requires PRISM_GATEWAY_API_KEY to be set")
		}
		gw = gateway.New(cfg.Gateway, logger, ctx)
		logger.Info("gateway client initialized (required)",
			"base_url", cfg.Gateway.BaseURL,
		)
	default: // "auto"
		cfg.DataSource = "auto"
		gw = gateway.New(cfg.Gateway, logger, ctx)
		if gw != nil {
			logger.Info("gateway client initialized",
				"base_url", cfg.Gateway.BaseURL,
			)
		} else {
			logger.Info("no gateway API key configured, using mock data")
		}
	}

	app := &Application{
		Logger:  logger,
		Config:  cfg,
		Gateway: gw,
	}

	return app, nil
}

// Shutdown cleans up application resources.
func (app *Application) Shutdown() {
	if app.Gateway != nil {
		app.Gateway.Stop()
	}
}
