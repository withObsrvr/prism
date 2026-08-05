package pagesv2

import (
	"context"
	"strconv"
	"strings"
	"testing"

	legacy "github.com/withObsrvr/prism/internal/templates/pages"
)

func TestLedgerFailedCardCountAndPresentationUseSameClassification(t *testing.T) {
	tx := legacy.LedgerTx{
		Hash:      "failed-hash",
		ShortHash: "fail...hash",
		Kind:      "failed",
		OpType:    "invoke",
	}
	data := legacy.LedgerDetailData{Transactions: []legacy.LedgerTx{tx}}

	if got, want := failedTxCount(data), "1"; got != want {
		t.Fatalf("failedTxCount() = %q, want %q", got, want)
	}
	if got := classicTxCount(data); got != "0" {
		t.Fatalf("classicTxCount() = %q, want 0", got)
	}
	if got := sorobanTxCount(data); got != "0" {
		t.Fatalf("sorobanTxCount() = %q, want 0", got)
	}

	var html strings.Builder
	if err := txCard(tx, "testnet").Render(context.Background(), &html); err != nil {
		t.Fatalf("render failed transaction card: %v", err)
	}
	output := html.String()
	for _, want := range []string{
		`class="px-lg-txc fail"`,
		`data-px-ledger-tx-kind="failed"`,
		`class="px-lg-txc-status fail"`,
		// The network has to travel with the link: the transaction page loads
		// its contents through fragment requests that inherit nothing from the
		// page, so without it they query the default network and the page sits
		// on its loading skeleton.
		`?network=testnet`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("failed transaction card missing %q: %s", want, output)
		}
	}
	if strings.Contains(output, `class="px-lg-txc-status ok"`) {
		t.Errorf("failed transaction card rendered a success status: %s", output)
	}
}

// The operations fetch is capped, so a busy ledger classifies only a sample.
// The subhead has to say which, or it claims the ledger's full operation count
// above a bar whose segments sum to the sample.
func TestCompositionSubhead(t *testing.T) {
	tests := []struct {
		name       string
		opCount    string
		classified string
		want       string
	}{
		{
			name:       "complete ledger states the count plainly",
			opCount:    "108",
			classified: "108",
			want:       "108 operations · classified by Prism's interpretation layer",
		},
		{
			name:       "capped ledger discloses the sample",
			opCount:    "847",
			classified: "200",
			want:       "first 200 of 847 operations · classified by Prism's interpretation layer",
		},
		{
			name:       "no operations fetched says so rather than implying an empty ledger",
			opCount:    "847",
			classified: "0",
			want:       "847 operations · breakdown unavailable",
		},
		{
			name:       "unset (mock data) falls back to the plain form",
			opCount:    "108",
			classified: "",
			want:       "108 operations · classified by Prism's interpretation layer",
		},
		{
			// Ledger 3886649: OperationCount counts only operations in successful
			// transactions, so a ledger with a failure returns more operations
			// than it reports. "first 15 of 11" must never render.
			name:       "more classified than reported never claims a sample",
			opCount:    "11",
			classified: "15",
			want:       "15 operations · classified by Prism's interpretation layer",
		},
		{
			name:       "thousands separators do not break the comparison",
			opCount:    "1,204",
			classified: "200",
			want:       "first 200 of 1,204 operations · classified by Prism's interpretation layer",
		},
		{
			name:       "unparseable counts fall back rather than guessing",
			opCount:    "—",
			classified: "200",
			want:       "200 operations · classified by Prism's interpretation layer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := legacy.LedgerDetailData{OpCount: tt.opCount, OpsClassified: tt.classified}
			if got := compositionSubhead(data); got != tt.want {
				t.Errorf("compositionSubhead() = %q, want %q", got, tt.want)
			}
		})
	}
}

// The legend decodes the bar, so its totals must equal the bar's segments.
func TestLegendEntriesTotalsMatchSegments(t *testing.T) {
	data := legacy.LedgerDetailData{
		OpCount: "108",
		OpBreakdown: []legacy.OpBreakdownItem{
			{Name: "Payment", Count: "24", Family: "transfer", Tier: 1},
			{Name: "Path Payment", Count: "12", Family: "transfer", Tier: 2},
			{Name: "Create Account", Count: "9", Family: "transfer", Tier: 3},
			{Name: "Contract Call", Count: "30", Family: "contract", Tier: 1},
			{Name: "Extend TTL", Count: "6", Family: "contract", Tier: 2},
			{Name: "Clawback", Count: "8", Family: "revoke", Tier: 1},
		},
	}

	entries := legendEntries(data)
	want := []legendEntry{
		{Family: "transfer", Label: "value transfer", Count: "45"},
		{Family: "contract", Label: "contract call", Count: "36"},
		{Family: "revoke", Label: "revocation", Count: "8"},
	}
	if len(entries) != len(want) {
		t.Fatalf("got %d legend entries, want %d", len(entries), len(want))
	}
	total := 0
	for i, e := range entries {
		if e != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, e, want[i])
		}
		n, err := strconv.Atoi(e.Count)
		if err != nil {
			t.Fatalf("entry %d count %q is not a number: %v", i, e.Count, err)
		}
		total += n
	}
	if total != 89 {
		t.Errorf("legend totals sum to %d, want 89 (the sum of the segments)", total)
	}
}

// Families must appear once each, in first-seen order, so the legend lines up
// with the contiguous family blocks in the bar.
func TestLegendEntriesOneRowPerFamily(t *testing.T) {
	data := legacy.LedgerDetailData{
		OpBreakdown: []legacy.OpBreakdownItem{
			{Count: "5", Family: "market", Tier: 1},
			{Count: "3", Family: "market", Tier: 2},
			{Count: "9", Family: "controls", Tier: 1},
		},
	}
	entries := legendEntries(data)
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Family != "market" || entries[0].Count != "8" {
		t.Errorf("first entry = %+v, want market/8", entries[0])
	}
	if entries[1].Family != "controls" || entries[1].Count != "9" {
		t.Errorf("second entry = %+v, want controls/9", entries[1])
	}
}
