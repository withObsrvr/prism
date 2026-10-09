package server

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/withObsrvr/prism/internal/jev"
)

func TestNewRequiresKeyWhenJevShadowIsEnabled(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	_, err := New(logger, Config{DataSource: "mock", Jev: jev.Config{Enabled: true}}, context.Background())
	if err == nil || !strings.Contains(err.Error(), "PRISM_JEV_API_KEY") {
		t.Fatalf("error = %v", err)
	}
}

func TestNewInitializesOptionalShadowRecorder(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	path := filepath.Join(t.TempDir(), "shadow.jsonl")
	app, err := New(logger, Config{DataSource: "mock", Jev: jev.Config{ShadowLogPath: path}}, context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if app.LedgerShadowRecorder == nil {
		t.Fatal("shadow recorder was not initialized")
	}
	app.Shutdown()
}

func TestNewLeavesJevDisabledWithoutCredentials(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	app, err := New(logger, Config{DataSource: "mock"}, context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if app.LedgerSelector != nil {
		t.Fatal("disabled Jev configured a ledger selector")
	}
}
