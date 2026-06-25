package prism

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/server"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Prism web server",
	Long: `Start the Prism HTTP server. By default it listens on :3000.

  prism serve
  prism serve --port 8080
  PRISM_DATA_SOURCE=mock prism serve`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().Int("port", 3002, "port to listen on")
	serveCmd.Flags().String("host", "0.0.0.0", "host to bind to")

	viper.BindPFlag("port", serveCmd.Flags().Lookup("port"))
	viper.BindPFlag("host", serveCmd.Flags().Lookup("host"))

	// Defaults.
	viper.SetDefault("port", 3002)
	viper.SetDefault("host", "0.0.0.0")
	viper.SetDefault("data_source", "auto")

	// Gateway defaults. Keep this below the HTTP write timeout so slow gateway
	// endpoints fail fast and pages can render their mock/stale fallbacks. Explorer
	// event queries can legitimately take 6s+ on remote deployments, so leave
	// enough headroom while still staying below the 10s WriteTimeout.
	viper.SetDefault("gateway.base_url", "https://gateway.withobsrvr.com")
	viper.SetDefault("gateway.timeout", "8s")
}

func runServe(cmd *cobra.Command, args []string) error {
	// Set up structured logging (Alex Edwards pattern).
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(viper.GetString("log_level")),
	}))

	// Parse gateway timeout.
	gwTimeout, err := time.ParseDuration(viper.GetString("gateway.timeout"))
	if err != nil {
		gwTimeout = 8 * time.Second
	}

	// Application-wide context for background goroutines.
	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	// Build the application (dependency injection via struct).
	app, err := server.New(logger, server.Config{
		Host:       viper.GetString("host"),
		Port:       viper.GetInt("port"),
		DataSource: viper.GetString("data_source"),
		Gateway: gateway.Config{
			BaseURL: viper.GetString("gateway.base_url"),
			APIKey:  viper.GetString("gateway.api_key"),
			Timeout: gwTimeout,
		},
	}, appCtx)
	if err != nil {
		return fmt.Errorf("initializing server: %w", err)
	}
	defer app.Shutdown()

	// Configure the HTTP server with sensible timeouts
	// (from Let's Go Further, Chapter 3).
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", app.Config.Host, app.Config.Port),
		Handler:      app.Routes(),
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
		IdleTimeout: time.Minute,
		ReadTimeout: 5 * time.Second,
		// Bumped from 10s: a ledger-detail render fans out to several gateway calls
		// that are slow during catch-up (cold reads 10-12s); 10s cut off the write
		// mid-response ("i/o timeout"). Keep under the DO App Platform LB ceiling.
		WriteTimeout: 35 * time.Second,
	}

	// Graceful shutdown (Let's Go Further, Chapter 12).
	shutdownErr := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		logger.Info("shutting down server", "signal", s.String())
		appCancel()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		shutdownErr <- srv.Shutdown(ctx)
	}()

	logger.Info("starting server",
		"addr", srv.Addr,
		"data_source", app.Config.DataSource,
		"gateway_timeout", gwTimeout,
	)

	err = srv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	err = <-shutdownErr
	if err != nil {
		return err
	}

	logger.Info("server stopped")
	return nil
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
