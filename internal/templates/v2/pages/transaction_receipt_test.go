package pagesv2

import (
	"context"
	"strings"
	"testing"

	legacy "github.com/withObsrvr/prism/internal/templates/pages"
)

func TestClassicMultiOperationHeroRendersForFullPageAndHTMX(t *testing.T) {
	data := classicMultiOperationReceiptFixture()
	components := map[string]struct {
		render func(*strings.Builder) error
	}{
		"full page": {
			render: func(out *strings.Builder) error {
				return TransactionReceipt(data, "mainnet").Render(context.Background(), out)
			},
		},
		"hero fragment": {
			render: func(out *strings.Builder) error {
				return TxReceiptHeroFragment(data).Render(context.Background(), out)
			},
		},
		"detail fragment": {
			render: func(out *strings.Builder) error {
				return TxReceiptDetailFragment(data).Render(context.Background(), out)
			},
		},
	}

	for name, component := range components {
		t.Run(name, func(t *testing.T) {
			var out strings.Builder
			if err := component.render(&out); err != nil {
				t.Fatalf("render: %v", err)
			}
			html := out.String()
			if strings.Contains(html, "Called contract") || strings.Contains(html, "What was called") {
				t.Fatalf("classic transaction rendered contract-call copy: %s", html)
			}
			if name != "detail fragment" && !strings.Contains(html, "14 operations") {
				t.Fatalf("render missing operation count: %s", html)
			}
			if name == "detail fragment" {
				if !strings.Contains(html, "Manage Buy Offer") || !strings.Contains(html, "Manage Sell Offer") {
					t.Fatalf("detail fragment lost operation evidence: %s", html)
				}
			}
		})
	}
}

func classicMultiOperationReceiptFixture() legacy.TxReceiptData {
	operations := make([]legacy.TxOperation, 0, 14)
	for i := 0; i < 7; i++ {
		operations = append(operations,
			legacy.TxOperation{Index: "buy", Type: "Manage Buy Offer", Status: "Success"},
			legacy.TxOperation{Index: "sell", Type: "Manage Sell Offer", Status: "Success"},
		)
	}
	return legacy.TxReceiptData{
		Hash:                "abc",
		ShortHash:           "abc",
		Status:              "success",
		SemanticTxType:      "multi_op",
		EffectiveActorShort: "GABC...WXYZ",
		EffectiveActorAddr:  "GABC",
		SourceAddr:          "GABC...WXYZ",
		SourceAddrFull:      "GABC",
		Ledger:              "63,000,000",
		LedgerRaw:           "63000000",
		OpsCount:            "14",
		EventsCount:         "0",
		FeePaidXLM:          "0.00014 XLM",
		Operations:          operations,
	}
}
