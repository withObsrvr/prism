package handlers

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	vmv2 "github.com/withObsrvr/prism/internal/templates/v2/viewmodel"
)

func TestBuildHomeTimelineUsesEnrichedRecentFacts(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 10, 0, time.UTC)
	response := &gateway.RecentLedgersResponse{
		LatestSequence: 101,
		Count:          2,
		SourceLedger: gateway.RecentLedgerSource{
			Sequence:   101,
			ClosedAt:   "2026-07-28T12:00:05Z",
			AgeSeconds: 5,
			Freshness:  "fresh",
		},
		Ledgers: []gateway.RecentLedger{
			timelineFixtureLedger(101, "2026-07-28T12:00:05Z", 19, 4),
			timelineFixtureLedger(100, "2026-07-28T12:00:00Z", 10, 1),
		},
		Provenance: gateway.RecentLedgerProvenance{
			DataSource:            "serving.sv_ledger_stats_recent",
			CompleteThroughLedger: 101,
		},
	}

	data := buildHomeTimelineDataAt(response, "testnet", "/v2/home/timeline?network=testnet", now)

	if data.Status.State != vmv2.HomeSectionReady || data.HeaderState != "Live" {
		t.Fatalf("unexpected timeline state: %+v", data.Status)
	}
	if data.WindowLabel != "Last 2 ledgers · 5 seconds" || data.StartSequence != "100" || data.EndSequence != "101" {
		t.Fatalf("window was not derived from returned rows: %+v", data)
	}
	if len(data.Columns) != 2 || data.Columns[0].Sequence != 100 || data.Columns[1].Sequence != 101 || !data.Columns[1].Latest {
		t.Fatalf("columns are not ordered oldest to newest: %+v", data.Columns)
	}
	latest := data.Columns[1]
	if latest.TransactionCount != 14 || latest.IncludedOperations != 19 || latest.SuccessfulOperations != 15 || latest.FailedOperations != 4 {
		t.Fatalf("explicit ledger evidence was not preserved: %+v", latest)
	}
	if latest.Introducer != "SDF Testnet 3" || !strings.Contains(latest.AccessibleLabel, "introduced by SDF Testnet 3") {
		t.Fatalf("validator introduction evidence was not surfaced: %+v", latest)
	}
	if got := segmentCount(latest.Segments, "payments"); got != 2 {
		t.Fatalf("payments = %d, want 2", got)
	}
	if got := segmentCount(latest.Segments, "calls"); got != 5 {
		t.Fatalf("contract calls = %d, want 5", got)
	}
	if got := segmentCount(latest.Segments, "deployments"); got != 1 {
		t.Fatalf("deployments = %d, want 1", got)
	}
	if data.FailureCount != 5 || data.FailurePercent != "17%" {
		t.Fatalf("failure summary does not use returned operations: count=%d percent=%s", data.FailureCount, data.FailurePercent)
	}
}

func TestBuildHomeTimelineFallsBackToOneUnclassifiedSegment(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 0, 10, 0, time.UTC)
	response := &gateway.RecentLedgersResponse{Ledgers: []gateway.RecentLedger{{
		LedgerSequence:    42,
		ClosedAt:          "2026-07-28T12:00:05Z",
		SuccessfulTxCount: 3,
		FailedTxCount:     1,
		OperationCount:    7,
	}}}

	data := buildHomeTimelineDataAt(response, "mainnet", "/v2/home/timeline?network=mainnet", now)
	if len(data.Columns) != 1 {
		t.Fatalf("columns = %d, want 1", len(data.Columns))
	}
	column := data.Columns[0]
	if column.TransactionCount != 4 || column.IncludedOperations != 7 || column.SuccessfulOperations != 7 || column.FailedOperations != 0 {
		t.Fatalf("legacy counts were not preserved: %+v", column)
	}
	if len(column.Segments) != 1 || column.Segments[0].Kind != "other" || column.Segments[0].Count != 7 {
		t.Fatalf("legacy activity should remain explicitly unclassified: %+v", column.Segments)
	}
}

func TestBuildHomeTimelineDoesNotCallPartialEmptyAuthoritative(t *testing.T) {
	partial := buildHomeTimelineDataAt(&gateway.RecentLedgersResponse{
		Provenance: gateway.RecentLedgerProvenance{Partial: true},
	}, "mainnet", "/v2/home/timeline?network=mainnet", time.Now())
	if partial.Status.State != vmv2.HomeSectionUnavailable {
		t.Fatalf("partial empty state = %q, want unavailable", partial.Status.State)
	}

	empty := buildHomeTimelineDataAt(&gateway.RecentLedgersResponse{}, "mainnet", "/v2/home/timeline?network=mainnet", time.Now())
	if empty.Status.State != vmv2.HomeSectionEmpty {
		t.Fatalf("complete empty state = %q, want empty", empty.Status.State)
	}
}

func TestBuildHomeTimelineKeepsFactsVisibleWhenPartialOrStale(t *testing.T) {
	now := time.Date(2026, 7, 28, 12, 5, 0, 0, time.UTC)
	ledger := timelineFixtureLedger(101, "2026-07-28T12:00:00Z", 19, 4)

	partial := buildHomeTimelineDataAt(&gateway.RecentLedgersResponse{
		SourceLedger: gateway.RecentLedgerSource{Sequence: 101, ClosedAt: ledger.ClosedAt, Freshness: "fresh"},
		Ledgers:      []gateway.RecentLedger{ledger},
		Provenance:   gateway.RecentLedgerProvenance{Partial: true, Warnings: []string{"identity coverage incomplete"}},
	}, "mainnet", "/v2/home/timeline?network=mainnet", now)
	if partial.Status.State != vmv2.HomeSectionPartial || len(partial.Columns) != 1 || partial.HeaderState != "Partial data" {
		t.Fatalf("partial facts were not preserved with a caveat: %+v", partial)
	}
	if len(partial.Status.Warnings) != 1 {
		t.Fatalf("partial warnings were not preserved: %+v", partial.Status)
	}

	stale := buildHomeTimelineDataAt(&gateway.RecentLedgersResponse{
		SourceLedger: gateway.RecentLedgerSource{Sequence: 101, ClosedAt: ledger.ClosedAt, Freshness: "stale", AgeSeconds: 300},
		Ledgers:      []gateway.RecentLedger{ledger},
	}, "mainnet", "/v2/home/timeline?network=mainnet", now)
	if stale.Status.State != vmv2.HomeSectionStale || len(stale.Columns) != 1 || stale.HeaderState != "Data delayed" {
		t.Fatalf("stale facts were not preserved with a caveat: %+v", stale)
	}
}

func TestHomeV2LiveModeNeverSeedsMockFacts(t *testing.T) {
	h := &Handlers{Logger: testHomeLogger(), DataSource: "auto"}
	request := httptest.NewRequest("GET", "/v2/home?network=mainnet", nil)
	recorder := httptest.NewRecorder()

	h.HomeV2(recorder, request)
	output := recorder.Body.String()
	for _, forbidden := range []string{"52,844,201", "Soroswap · Router", "84,201", "Synthetic fixture"} {
		if strings.Contains(output, forbidden) {
			t.Errorf("live shell leaked mock fact %q", forbidden)
		}
	}
	for _, required := range []string{"Find anything on Stellar", "Reading the latest ledgers", "/v2/home/timeline?network=mainnet"} {
		if !strings.Contains(output, required) {
			t.Errorf("live shell missing %q", required)
		}
	}
}

func TestHomeV2MockModeIsExplicitlyLabeled(t *testing.T) {
	h := &Handlers{Logger: testHomeLogger(), DataSource: "auto"}
	request := httptest.NewRequest("GET", "/v2/home?network=testnet&mock=true", nil)
	recorder := httptest.NewRecorder()

	h.HomeV2(recorder, request)
	output := recorder.Body.String()
	if !strings.Contains(output, "Demo data") || !strings.Contains(output, "mock=true") {
		t.Fatalf("explicit mock mode is not visibly labeled or preserved in fragment URL: %s", output)
	}
}

func TestHomeV2TimelineWithoutGatewayRendersUnavailableNotMock(t *testing.T) {
	h := &Handlers{Logger: testHomeLogger(), DataSource: "auto"}
	request := httptest.NewRequest("GET", "/v2/home/timeline?network=mainnet", nil)
	recorder := httptest.NewRecorder()

	h.HomeV2Timeline(recorder, request)
	output := recorder.Body.String()
	if !strings.Contains(output, "Ledger activity is temporarily unavailable") || strings.Contains(output, "Demo validator") {
		t.Fatalf("unavailable live timeline did not fail truthfully: %s", output)
	}
}

func timelineFixtureLedger(sequence int64, closedAt string, included, failed int) gateway.RecentLedger {
	return gateway.RecentLedger{
		LedgerSequence:               sequence,
		ClosedAt:                     closedAt,
		TransactionCount:             14,
		TransactionSetOperationCount: included,
		SuccessfulOperationCount:     included - failed,
		FailedOperationCount:         failed,
		Validator: gateway.LedgerValidator{
			AttributionAvailable: true,
			Status:               "resolved",
			DisplayName:          "SDF Testnet 3",
		},
		Transactions: gateway.RecentLedgerTransactionStats{Total: 14, Successful: 13, Failed: 1},
		Operations: gateway.RecentLedgerOperationStats{
			Included:             included,
			Successful:           included - failed,
			Failed:               failed,
			ClassificationStatus: "materialized",
			Categories: gateway.RecentLedgerOperationCategories{
				AccountCreation:   1,
				Payments:          2,
				OffersAndAMMs:     3,
				Trustlines:        1,
				ClaimableBalances: 1,
				Sponsorship:       1,
				Soroban:           8,
				Other:             max(0, included-17),
			},
			SorobanDetail: gateway.RecentLedgerSorobanDetail{ContractCalls: 5, ContractDeployments: 1, Other: 2},
		},
	}
}

func segmentCount(segments []vmv2.HomeSpectrogramSegment, kind string) int {
	for _, segment := range segments {
		if segment.Kind == kind {
			return segment.Count
		}
	}
	return 0
}

func testHomeLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
