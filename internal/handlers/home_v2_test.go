package handlers

import (
	"strings"
	"testing"

	"github.com/withObsrvr/prism/internal/gateway"
)

func TestBuildHomeV2FeedLedgerUsesEnrichedRecentFacts(t *testing.T) {
	recent := gateway.RecentLedger{
		LedgerSequence:               3707457,
		ClosedAt:                     "2026-07-20T12:00:05Z",
		SuccessfulTxCount:            13,
		FailedTxCount:                1,
		OperationCount:               15,
		TransactionCount:             14,
		TransactionSetOperationCount: 19,
		SuccessfulOperationCount:     15,
		FailedOperationCount:         4,
		Validator: gateway.LedgerValidator{
			PublicKey:            "GC2V2EFSXN6SQTWVYA5EPJPBWWIMSD2XQNKUOHGEKB535AQE2I6IXV2Z",
			AttributionAvailable: true,
			Status:               "resolved",
			DisplayName:          "SDF Testnet 3",
		},
		Operations: gateway.RecentLedgerOperationStats{
			Included:             19,
			Successful:           15,
			Failed:               4,
			ClassificationStatus: "materialized",
			Categories: gateway.RecentLedgerOperationCategories{
				AccountCreation:   1,
				Payments:          2,
				OffersAndAMMs:     3,
				Trustlines:        1,
				ClaimableBalances: 1,
				Sponsorship:       1,
				Soroban:           8,
				Other:             2,
			},
		},
	}
	next := gateway.RecentLedger{ClosedAt: "2026-07-20T12:00:00Z"}

	row, feed, _ := buildHomeV2FeedLedger("testnet", recent, &next, nil)

	if row.TransactionCount != "14" || row.IncludedOperationCount != "19" || row.SuccessfulOperationCount != "15" || row.FailedOperationCount != "4" {
		t.Fatalf("explicit ledger facts were not mapped: %+v", row)
	}
	if !strings.Contains(row.Meta, "19 operations (4 failed)") || row.Introducer != "SDF Testnet 3" {
		t.Fatalf("ledger meaning was not surfaced: %q", row.Meta)
	}
	if strings.Contains(row.Meta, "closed by") || strings.Contains(row.Meta, "introduced by") {
		t.Fatalf("introducer should remain structured presentation data: %q", row.Meta)
	}
	if len(row.Chips) != 8 || row.Chips[1].Label != "2 payments" || row.Chips[6].Label != "8 Soroban ops" {
		t.Fatalf("operation categories were not mapped: %+v", row.Chips)
	}
	if feed.IncludedOperationCount != row.IncludedOperationCount || feed.FailedOperationCount != row.FailedOperationCount || len(feed.Chips) != len(row.Chips) {
		t.Fatalf("polling feed diverged from initial render: feed=%+v row=%+v", feed, row)
	}
	if feed.Introducer != row.Introducer {
		t.Fatalf("polling feed lost introducer attribution: feed=%+v row=%+v", feed, row)
	}
}

func TestBuildHomeV2FeedLedgerPreservesLegacyRecentFallback(t *testing.T) {
	recent := gateway.RecentLedger{
		LedgerSequence:    42,
		ClosedAt:          "2026-07-20T12:00:05Z",
		SuccessfulTxCount: 3,
		FailedTxCount:     1,
		OperationCount:    7,
	}

	row, _, _ := buildHomeV2FeedLedger("mainnet", recent, nil, nil)

	if row.TransactionCount != "4" || row.IncludedOperationCount != "7" || row.SuccessfulOperationCount != "7" || row.FailedOperationCount != "0" {
		t.Fatalf("legacy count fallback changed: %+v", row)
	}
	if len(row.Chips) != 1 || row.Chips[0].Label != "3 successful txs" {
		t.Fatalf("legacy classification fallback changed: %+v", row.Chips)
	}
}
