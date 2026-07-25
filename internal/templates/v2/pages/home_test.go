package pagesv2

import (
	"context"
	"strings"
	"testing"

	componentsv2 "github.com/withObsrvr/prism/internal/templates/v2/components"
	vmv2 "github.com/withObsrvr/prism/internal/templates/v2/viewmodel"
)

func TestHomeLedgerRowsRenderOperationEvidence(t *testing.T) {
	data := vmv2.HomeData{
		Header: componentsv2.HeaderData{Network: "testnet", LedgerNumber: "3,707,457", AgeLabel: "just now"},
		LedgerFeed: vmv2.LedgerFeedData{Rows: []vmv2.LedgerRowData{{
			LedgerNumber:             "3,707,457",
			TransactionCount:         "14",
			IncludedOperationCount:   "19",
			SuccessfulOperationCount: "15",
			FailedOperationCount:     "4",
			Meta:                     "with 19 operations (4 failed)",
			Introducer:               "SDF Testnet 3",
			Chips:                    []componentsv2.LedgerMetricChip{{Label: "8 Soroban ops", Kind: "soroban"}},
		}}},
		FeedJSON: `{"ledgers":[]}`,
		FeedLive: true,
	}

	var html strings.Builder
	if err := Home(data).Render(context.Background(), &html); err != nil {
		t.Fatalf("render home: %v", err)
	}
	output := html.String()
	for _, want := range []string{
		"included operations", "successful ops", "failed ops", "introduced by", "SDF Testnet 3", "This validator introduced the transaction-set value selected through SCP.", "8 Soroban ops", "ph-ledger-chip soroban", "successfulOperationCount",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("home ledger evidence missing %q", want)
		}
	}
	if strings.Contains(output, "closed by SDF Testnet 3") {
		t.Fatal("home ledger still describes one validator as the ledger closer")
	}
	if strings.Contains(output, `id="ph-focus-instructions"`) || strings.Contains(output, `id="ph-focus-readwrite"`) {
		t.Fatal("home hero still renders zero-valued utilization as ledger evidence")
	}
}
