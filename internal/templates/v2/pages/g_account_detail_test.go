package pagesv2

import (
	"context"
	"strings"
	"testing"

	legacy "github.com/withObsrvr/prism/internal/templates/pages"
)

// fabrications are strings the account page may only emit when the gateway
// actually answered. Each one is a measurement claim: a threshold value, a
// liveness status, or a count. Rendering any of them from an empty struct is
// what made a funded mainnet wallet read as an active account holding nothing.
var fabrications = []string{
	"Thresholds 1/2/3",
	"Active · last op",
	"Balances are inspectable",
}

func renderAccountMain(t *testing.T, data legacy.AccountData) string {
	t.Helper()
	var html strings.Builder
	if err := GAccountDetailMain(data, "mainnet").Render(context.Background(), &html); err != nil {
		t.Fatalf("render account main: %v", err)
	}
	return html.String()
}

func TestUnavailableAccountAssertsNothingAboutTheAccount(t *testing.T) {
	out := renderAccountMain(t, legacy.AccountData{
		Address:           "GB32463OFXHQVLFRO3URPEV6RTSLLCBVYGE24JFF4CDMHHP37PQXVY7X",
		ShortAddress:      "GB32...VY7X",
		Unavailable:       true,
		UnavailableReason: "context deadline exceeded",
	})

	for _, bad := range fabrications {
		if strings.Contains(out, bad) {
			t.Errorf("unavailable account page asserts %q; a failed load must not claim measurements", bad)
		}
	}
	// A zero here reads as "this account has none", which is the specific lie.
	if strings.Contains(out, "0 trustlines") {
		t.Error("unavailable account page claims 0 trustlines")
	}
	for _, want := range []string{
		"Account evidence unavailable",
		"Try again",
		// The address is derived locally, so it is one of the few things a
		// failed load may still show.
		"GB32463O",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("unavailable account page missing %q", want)
		}
	}
	// Operator detail belongs in logs, not on the page.
	if strings.Contains(out, "context deadline exceeded") {
		t.Error("raw gateway error leaked into the rendered page")
	}
}

func TestPartialAccountShowsBalancesButNotInventedControls(t *testing.T) {
	// The real shape of the balances-only fallback: holdings are evidence,
	// signers and thresholds were never returned.
	out := renderAccountMain(t, legacy.AccountData{
		Address:      "GB32463OFXHQVLFRO3URPEV6RTSLLCBVYGE24JFF4CDMHHP37PQXVY7X",
		ShortAddress: "GB32...VY7X",
		Partial:      true,
		IsFunded:     true,
		Trustlines:   "3",
		SignerCount:  "—",
		Balances: []legacy.AccountBalance{
			{Code: "ICE", Name: "ICE", Type: "Classic", Balance: "1184369.0275145"},
		},
	})

	for _, bad := range []string{"Thresholds 1/2/3", "Active · last op"} {
		if strings.Contains(out, bad) {
			t.Errorf("partial account page asserts %q from data the gateway never returned", bad)
		}
	}
	// Balances did load, so this evidence must survive.
	if !strings.Contains(out, "ICE") || !strings.Contains(out, "1184369.0275145") {
		t.Error("partial account page dropped the balances that did load")
	}
	if !strings.Contains(out, "Thresholds unknown") {
		t.Error("partial account page should say thresholds are unknown")
	}
}

func TestLoadedAccountStillReportsRealZeros(t *testing.T) {
	// The counterpart risk: having taught the page to say "—" for unknown, a
	// genuine zero from a gateway that did answer must still render as 0.
	out := renderAccountMain(t, legacy.AccountData{
		Address:      "GB32463OFXHQVLFRO3URPEV6RTSLLCBVYGE24JFF4CDMHHP37PQXVY7X",
		ShortAddress: "GB32...VY7X",
		Trustlines:   "0",
		SignerCount:  "1",
		Thresholds: []legacy.AccountThreshold{
			{Label: "Low", Value: "0"}, {Label: "Medium", Value: "0"}, {Label: "High", Value: "0"},
		},
	})
	if strings.Contains(out, "Thresholds unknown") {
		t.Error("a loaded account with real thresholds was reported as unknown")
	}
	if !strings.Contains(out, "Thresholds 0/0/0") {
		t.Error("real threshold values from the gateway were not rendered")
	}
}

func TestFailedSignersCallDoesNotClaimZeroSigners(t *testing.T) {
	// The overview succeeded, so balances and sequence are real evidence. Only
	// the signers call failed. Before per-dependency timeouts this was rare
	// enough to ignore; now that each call fails independently it is a state
	// the page has to render honestly.
	out := renderAccountMain(t, legacy.AccountData{
		Address:        "GB32463OFXHQVLFRO3URPEV6RTSLLCBVYGE24JFF4CDMHHP37PQXVY7X",
		ShortAddress:   "GB32...VY7X",
		SignersUnknown: true,
		SignerCount:    "—",
		Trustlines:     "3",
		SequenceNumber: "245891039232",
	})

	if strings.Contains(out, "Thresholds 1/2/3") {
		t.Error("failed signers call rendered the protocol-default thresholds as this account's")
	}
	if !strings.Contains(out, "Thresholds unknown") {
		t.Error("failed signers call should report thresholds as unknown")
	}
	// Evidence from the calls that did succeed must survive.
	if !strings.Contains(out, "245891039232") {
		t.Error("sequence number from the successful overview was dropped")
	}
}
