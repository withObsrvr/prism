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
	"github.com/withObsrvr/prism/internal/server"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Prism web server",
	Long: `Start the Prism HTTP server. By default it listens on :3000.

  prism serve
  prism serve --port 8080
  prism serve --network testnet`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().Int("port", 3000, "port to listen on")
	serveCmd.Flags().String("host", "0.0.0.0", "host to bind to")
	serveCmd.Flags().String("network", "mainnet", "stellar network (mainnet, testnet)")

	viper.BindPFlag("port", serveCmd.Flags().Lookup("port"))
	viper.BindPFlag("host", serveCmd.Flags().Lookup("host"))
	viper.BindPFlag("network", serveCmd.Flags().Lookup("network"))

	// Defaults.
	viper.SetDefault("port", 3000)
	viper.SetDefault("host", "0.0.0.0")
	viper.SetDefault("network", "mainnet")
}

func runServe(cmd *cobra.Command, args []string) error {
	// Set up structured logging (Alex Edwards pattern).
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(viper.GetString("log_level")),
	}))

	// Build the application (dependency injection via struct).
	app, err := server.New(logger, server.Config{
		Host:    viper.GetString("host"),
		Port:    viper.GetInt("port"),
		Network: viper.GetString("network"),
	})
	if err != nil {
		return fmt.Errorf("initializing server: %w", err)
	}

	// Configure the HTTP server with sensible timeouts
	// (from Let's Go Further, Chapter 3).
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", app.Config.Host, app.Config.Port),
		Handler:      app.Routes(),
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Graceful shutdown (Let's Go Further, Chapter 12).
	shutdownErr := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		logger.Info("shutting down server", "signal", s.String())

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		shutdownErr <- srv.Shutdown(ctx)
	}()

	logger.Info("starting server",
		"addr", srv.Addr,
		"network", app.Config.Network,
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
