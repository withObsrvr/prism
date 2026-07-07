package handlers

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildAccountDataRejectsInvalidAccountIDAsFormatError(t *testing.T) {
	h := &Handlers{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	req := httptest.NewRequest("GET", "/account/not-an-account", nil)

	_, err := h.buildAccountData(req, "mainnet", "not-an-account")
	if err == nil {
		t.Fatal("buildAccountData error = nil, want invalid account id format")
	}
	if !strings.Contains(err.Error(), "invalid account id format") {
		t.Fatalf("buildAccountData error = %q, want format wording", err)
	}
}
