package pagesv2

import (
	"context"
	"strings"
	"testing"
	"time"

	componentsv2 "github.com/withObsrvr/prism/internal/templates/v2/components"
	vmv2 "github.com/withObsrvr/prism/internal/templates/v2/viewmodel"
)

func TestHomeRendersTruthfulShellAndInformationArchitecture(t *testing.T) {
	data := vmv2.HomeData{
		Header:         componentsv2.HeaderData{Network: "testnet", LedgerNumber: "Unavailable", AgeLabel: "Waiting for ledger data"},
		TimelineURL:    "/v2/home/timeline?network=testnet",
		InsightsURL:    "/v2/home/insights?network=testnet",
		TTLURL:         "/v2/home/ttl?network=testnet",
		LeadersURL:     "/v2/home/leaders?network=testnet",
		UtilizationURL: "/v2/home/utilization?network=testnet",
		Prompt:         vmv2.PromptData{Placeholder: "Transaction, account, contract, asset, or ledger"},
	}

	var html strings.Builder
	if err := Home(data).Render(context.Background(), &html); err != nil {
		t.Fatalf("render home: %v", err)
	}
	output := html.String()
	for _, want := range []string{
		"Search Stellar",
		"Paste a hash or address, or ask a question about the network.",
		`placeholder="Transaction, account, contract, asset, or ledger"`,
		"Reading the latest ledgers",
		"What changed",
		"Contract data expiring soon",
		"Busiest contracts, 24h",
		"Smart contract capacity",
		`class="ph-home-evidence-grid"`,
		`hx-get="/v2/home/timeline?network=testnet"`,
		`hx-get="/v2/home/insights?network=testnet"`,
		`hx-get="/v2/home/ttl?network=testnet"`,
		`hx-get="/v2/home/leaders?network=testnet"`,
		`hx-get="/v2/home/utilization?network=testnet"`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("home shell missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"52,844,201",
		"Three contracts are running out of room",
		"94 ms",
		"412 protocols",
		"Evidence section",
		"Lookup and explore",
		"Live ledger signal",
		"Persistent and contract-instance storage only",
		"Ranked by call count, not TVL",
		"Instructions, read and write bytes, and transaction size",
		"Find anything on Stellar",
		"Nearing archival",
		"Most called, 24 hours",
		"Network utilization",
		"Network load",
	} {
		if strings.Contains(output, forbidden) {
			t.Errorf("home shell leaked factual fallback %q", forbidden)
		}
	}
}

func TestHomeTimelineRendersAccessibleEvidenceAndPolling(t *testing.T) {
	data := vmv2.HomeTimelineData{
		Status: vmv2.HomeSectionStatus{
			State:      vmv2.HomeSectionReady,
			AsOfLedger: 101,
			AsOfTime:   time.Date(2026, 7, 28, 12, 0, 5, 0, time.UTC),
		},
		PollURL:         "/v2/home/timeline?network=testnet",
		HeaderState:     "Live",
		HeaderLedger:    "101",
		HeaderAge:       "5s ago",
		HeaderTxCount:   "14",
		WindowLabel:     "Last 2 ledgers · 5 seconds",
		DetailLabel:     "Column height shows included operations.",
		StartSequence:   "100",
		EndSequence:     "101",
		ColumnGridStyle: "--ph-column-count:1",
		Columns: []vmv2.HomeSpectrogramColumn{{
			Sequence:             101,
			SequenceLabel:        "101",
			Href:                 "/v2/ledger/101",
			ClosedAt:             "12:00:05 UTC",
			AgeLabel:             "5s ago",
			TransactionCount:     14,
			IncludedOperations:   19,
			SuccessfulOperations: 15,
			FailedOperations:     4,
			Introducer:           "SDF Testnet 3",
			HeightStyle:          "--ph-column-height:100%",
			FailureStyle:         "--ph-failure-height:100%",
			AccessibleLabel:      "Ledger 101, 14 transactions, 19 included operations, 15 successful operations, 4 failed operations, introduced by SDF Testnet 3",
			Latest:               true,
			Segments:             []vmv2.HomeSpectrogramSegment{{Kind: "calls", Label: "Contract calls", Count: 8, Style: "--ph-segment-weight:8"}},
		}},
		Legend:          []vmv2.HomeSpectrogramLegendItem{{Kind: "calls", Label: "Contract calls", Count: 8, Percentage: "42%"}},
		FailureCount:    4,
		FailurePercent:  "21%",
		AsOfLedgerLabel: "101",
	}

	var html strings.Builder
	if err := HomeTimeline(data).Render(context.Background(), &html); err != nil {
		t.Fatalf("render home timeline: %v", err)
	}
	output := html.String()
	for _, want := range []string{
		`href="/v2/ledger/101"`,
		`aria-label="Ledger 101, 14 transactions, 19 included operations, 15 successful operations, 4 failed operations, introduced by SDF Testnet 3"`,
		"Contract calls",
		"SDF Testnet 3",
		`hx-trigger="every 5s"`,
		`hx-swap-oob="true"`,
		`aria-live="off"`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("timeline missing %q", want)
		}
	}
	if strings.Contains(output, "closed by SDF Testnet 3") {
		t.Fatal("timeline describes one validator as the ledger closer")
	}
}

func TestHomeInsightsRendersACompactEvidenceRow(t *testing.T) {
	data := vmv2.HomeInsightsData{
		Status:  vmv2.HomeSectionStatus{State: vmv2.HomeSectionPartial, Warnings: []string{"Insight evidence is incomplete."}},
		Network: "testnet",
		PollURL: "/v2/home/insights?network=testnet",
		Cards: []vmv2.HomeInsightCard{{
			Title:        "Contract deployments increased",
			Detail:       "The most active new contract received 14 calls after deployment.",
			Tone:         "signal",
			SubjectID:    "testnet",
			SubjectLabel: "testnet",
			Metrics: []vmv2.HomeInsightMetric{
				{Label: "Last hour", Value: "80"},
				{Label: "Typical hour", Value: "20"},
				{Label: "Change", Value: "4×"},
			},
			DetailHref: "/v2/insight/hiev1_test?network=testnet",
			Evidence:   []vmv2.HomeInsightEvidenceLink{{Label: "View deployment ledgers", Href: "/v2/explore?time=coverage"}},
			Caveats:    []string{"The contributor list is bounded."},
		}},
	}

	var html strings.Builder
	if err := HomeInsights(data).Render(context.Background(), &html); err != nil {
		t.Fatalf("render home insights: %v", err)
	}
	output := html.String()
	for _, want := range []string{
		`class="ph-insight-list has-1"`,
		`class="ph-insight-proof"`,
		`class="ph-insight-facts"`,
		"Last hour",
		"Typical hour",
		"Change",
		"Why Prism flagged this",
		`class="ph-insight-coverage"`,
		"Coverage note",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("home insight missing %q", want)
		}
	}
	if strings.Contains(output, "View deployment ledgers") {
		t.Fatal("homepage repeated the raw evidence link when the insight explanation is available")
	}
	if strings.Contains(output, "Subject</span><strong>testnet") {
		t.Fatal("network-wide insight rendered a redundant testnet subject")
	}

	data.Cards = append(data.Cards, data.Cards[0], data.Cards[0])
	html.Reset()
	if err := HomeInsights(data).Render(context.Background(), &html); err != nil {
		t.Fatalf("render three home insights: %v", err)
	}
	if !strings.Contains(html.String(), `class="ph-insight-list has-3"`) {
		t.Fatal("three insights did not receive the wide-screen comparison layout")
	}
}

func TestHomeInsightsMakesEmptyAndUnavailableComparisonsUsefulWithoutInventingChange(t *testing.T) {
	tests := []struct {
		name      string
		status    vmv2.HomeSectionStatus
		want      []string
		forbidden []string
	}{
		{
			name:   "authoritative empty",
			status: vmv2.HomeSectionStatus{State: vmv2.HomeSectionEmpty, Message: "No significant changes in the last completed hour."},
			want: []string{
				"What changed",
				"Last completed hour",
				"No unusual changes crossed Prism’s thresholds",
				"Contract failures, deployments, and transaction activity",
			},
			forbidden: []string{"Unavailable", "Retry"},
		},
		{
			name:   "comparison unavailable",
			status: vmv2.HomeSectionStatus{State: vmv2.HomeSectionUnavailable, Message: "The delayed component did not provide retained evidence.", Retryable: true},
			want: []string{
				"Hourly comparison",
				"Current evidence continues below",
				"Comparison delayed",
				"Prism cannot compare the last completed hour right now.",
				"Retry",
			},
			forbidden: []string{"What changed", "Compared with a typical hour"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := vmv2.HomeInsightsData{Status: test.status, Network: "testnet", PollURL: "/v2/home/insights?network=testnet"}
			var html strings.Builder
			if err := HomeInsights(data).Render(context.Background(), &html); err != nil {
				t.Fatalf("render home insights: %v", err)
			}
			output := html.String()
			for _, want := range test.want {
				if !strings.Contains(output, want) {
					t.Errorf("home insight state missing %q", want)
				}
			}
			for _, forbidden := range test.forbidden {
				if strings.Contains(output, forbidden) {
					t.Errorf("home insight state contains %q", forbidden)
				}
			}
		})
	}
}

func TestHomeInsightsRendersQuietHourChecksAndRecentHistoryCompactly(t *testing.T) {
	data := vmv2.HomeInsightsData{
		Status: vmv2.HomeSectionStatus{State: vmv2.HomeSectionEmpty, Message: "No significant changes in the last completed hour."},
		Network: "testnet", PollURL: "/v2/home/insights?network=testnet", WindowLabel: "Jul 31, 21:00 to 22:00 UTC",
		Checks: []vmv2.HomeInsightCheck{{Label: "Contract failures", Value: "0.7× typical", Detail: "8 now, 11 typical", State: "ready"}, {Label: "Successful activity", Value: "1.1× typical", Detail: "6,400 now, 6,200 typical", State: "ready"}},
		RecentLabel: "Contract deployments increased", RecentDetailHref: "/v2/insight/hiev1_example?network=testnet", RecentTimeLabel: "22:00 UTC",
	}
	var html strings.Builder
	if err := HomeInsights(data).Render(context.Background(), &html); err != nil {
		t.Fatalf("render home insights: %v", err)
	}
	output := html.String()
	for _, want := range []string{"ph-insight-checks", "Contract failures", "0.7× typical", "8 now, 11 typical", "Last flagged", "Contract deployments increased"} {
		if !strings.Contains(output, want) {
			t.Errorf("quiet-hour comparison missing %q", want)
		}
	}
}
