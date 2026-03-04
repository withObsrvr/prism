package server

import (
	"log/slog"
)

// Config holds all server configuration, populated by viper.
type Config struct {
	Host    string
	Port    int
	Network string // "mainnet" or "testnet"
}

// Application holds all dependencies for the HTTP handlers.
// This is the Alex Edwards pattern from "Let's Go" — all shared
// dependencies live here and are passed to handlers via methods.
type Application struct {
	Logger *slog.Logger
	Config Config
}

// New creates a new Application with all dependencies wired up.
func New(logger *slog.Logger, cfg Config) (*Application, error) {
	app := &Application{
		Logger: logger,
		Config: cfg,
	}

	// Future: initialize database connections, cache, etc.
	// app.DB, err = database.Open(cfg.DatabaseURL)
	// app.Cache = cache.New(cfg.RedisURL)

	return app, nil
}
