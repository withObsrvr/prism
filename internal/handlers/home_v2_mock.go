package handlers

import (
	"fmt"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	vmv2 "github.com/withObsrvr/prism/internal/templates/v2/viewmodel"
)

func mockHomeV2Data(network string) vmv2.HomeData {
	data := emptyHomeV2Data(network)
	data.MockMode = true
	data.TimelineURL = homeV2FragmentURL("timeline", network, true)
	data.InsightsURL = homeV2FragmentURL("insights", network, true)
	data.TTLURL = homeV2FragmentURL("ttl", network, true)
	data.LeadersURL = homeV2FragmentURL("leaders", network, true)
	data.UtilizationURL = homeV2FragmentURL("utilization", network, true)
	data.Header.LedgerNumber = "Demo"
	data.Header.AgeLabel = "Synthetic fixture"
	return data
}

func mockHomeV2TimelineData(network string, now time.Time) vmv2.HomeTimelineData {
	const count = 60
	base := mockHomeLedgerBase(network)
	ledgers := make([]gateway.RecentLedger, 0, count)
	for offset := 0; offset < count; offset++ {
		sequence := base - int64(offset)
		closedAt := now.Add(-time.Duration(offset*5) * time.Second).UTC()
		payments := 18 + (offset*7)%31
		markets := 9 + (offset*5)%22
		calls := 3 + (offset*3)%13
		deployments := 0
		if offset%11 == 0 {
			deployments = 1
		}
		otherSoroban := offset % 3
		account := 1 + offset%4
		trustlines := offset % 5
		claimable := offset % 2
		sponsorship := offset % 3
		other := 2 + offset%7
		soroban := calls + deployments + otherSoroban
		included := payments + markets + soroban + account + trustlines + claimable + sponsorship + other
		failed := offset % 6
		transactions := 12 + (offset*11)%48

		ledgers = append(ledgers, gateway.RecentLedger{
			LedgerSequence:               sequence,
			ClosedAt:                     closedAt.Format(time.RFC3339),
			TransactionCount:             transactions,
			SuccessfulTxCount:            transactions - min(offset%4, transactions),
			FailedTxCount:                min(offset%4, transactions),
			TransactionSetOperationCount: included,
			OperationCount:               included,
			SuccessfulOperationCount:     included - failed,
			FailedOperationCount:         failed,
			Validator: gateway.LedgerValidator{
				AttributionAvailable: true,
				Status:               "resolved",
				DisplayName:          fmt.Sprintf("Demo validator %d", offset%5+1),
				Source:               "demo_fixture",
			},
			Transactions: gateway.RecentLedgerTransactionStats{
				Total:      transactions,
				Successful: transactions - min(offset%4, transactions),
				Failed:     min(offset%4, transactions),
			},
			Operations: gateway.RecentLedgerOperationStats{
				Included:             included,
				Successful:           included - failed,
				Failed:               failed,
				ClassificationStatus: "materialized",
				Categories: gateway.RecentLedgerOperationCategories{
					AccountCreation:   account,
					Payments:          payments,
					OffersAndAMMs:     markets,
					Trustlines:        trustlines,
					ClaimableBalances: claimable,
					Sponsorship:       sponsorship,
					Soroban:           soroban,
					Other:             other,
				},
				SorobanDetail: gateway.RecentLedgerSorobanDetail{
					ContractCalls:       calls,
					ContractDeployments: deployments,
					Other:               otherSoroban,
				},
			},
		})
	}

	response := &gateway.RecentLedgersResponse{
		LatestSequence: base,
		Count:          count,
		GeneratedAt:    now.UTC().Format(time.RFC3339),
		SourceLedger: gateway.RecentLedgerSource{
			Sequence:   base,
			ClosedAt:   now.UTC().Format(time.RFC3339),
			Freshness:  "fresh",
			AgeSeconds: 0,
		},
		Ledgers: ledgers,
		Provenance: gateway.RecentLedgerProvenance{
			DataSource:            "demo_fixture",
			CompleteThroughLedger: base,
		},
	}
	data := buildHomeTimelineDataAt(response, network, homeV2TimelineURL(network, true), now)
	data.DemoData = true
	data.HeaderState = "Demo data"
	return data
}

func mockHomeLedgerBase(network string) int64 {
	switch network {
	case "testnet":
		return 3_796_000
	case "futurenet":
		return 2_104_552
	default:
		return 63_640_409
	}
}

func mockHomeSummaryResponse(network string) *gateway.HomeSummaryResponse {
	now := time.Now().UTC().Truncate(time.Hour)
	asOf := mockHomeLedgerBase(network)
	minimumObserved := 3.0
	caveats := []gateway.HomeInsightCaveat{}
	failed := int64(4)
	succeeded := int64(87)
	successRate := 87.0 / 91.0
	failureRate := 4.0 / 91.0
	instructionUsed := int64(64_000_000)
	instructionLimit := int64(100_000_000)
	instructionRatio := 0.64
	readWriteUsed := int64(12_800_000)
	readWriteLimit := int64(20_000_000)
	readWriteRatio := 0.64
	avgTxSize := 2048.0
	txSizeLimit := int64(7168)
	txSizeRatio := avgTxSize / float64(txSizeLimit)

	return &gateway.HomeSummaryResponse{
		Network:     network,
		GeneratedAt: now.Add(5 * time.Second).Format(time.RFC3339),
		Freshness: gateway.HomeSummaryFreshness{
			SourceLedger: asOf, SourceClosedAt: now.Format(time.RFC3339), Status: "fresh",
		},
		Components: gateway.HomeSummaryComponents{
			Insights:     gateway.HomeSummaryComponent{Status: "ready", Source: "demo_fixture", AsOfLedger: &asOf, CompleteThroughLedger: &asOf},
			TTLAttention: gateway.HomeSummaryComponent{Status: "ready", Source: "demo_fixture", AsOfLedger: &asOf, CompleteThroughLedger: &asOf},
			Leaders:      gateway.HomeSummaryComponent{Status: "ready", Source: "demo_fixture", AsOfLedger: &asOf, CompleteThroughLedger: &asOf},
			Utilization:  gateway.HomeSummaryComponent{Status: "ready", Source: "demo_fixture", AsOfLedger: &asOf, CompleteThroughLedger: &asOf},
		},
		Insights: []gateway.HomeSummaryInsight{{
			InsightID:       "hiev1_wwnNOzs1woMnG3W4aRHIShks_8_e0A70A62b6lJm7IE",
			Network:         network,
			Type:            "failure_spike",
			EvidenceVersion: "home_insight_evidence_v1",
			Definition: &gateway.HomeInsightDefinition{
				RuleID: "contract_failure_spike", RuleVersion: "1", ComparisonMethod: "rolling_7d_median_prior_complete_hour", MinimumObserved: &minimumObserved, MinimumRatio: 3,
			},
			Subject: gateway.HomeSummaryInsightSubject{
				Kind: "contract", ID: "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD2KM",
				Identity: &gateway.HomeInsightIdentity{DisplayName: "Demo router", Kind: "protocol_contract", VerificationStatus: "inferred", Source: "demo_fixture"},
			},
			Observed: &gateway.HomeInsightObserved{Value: 42, WindowStart: now.Add(-time.Hour).Format(time.RFC3339), WindowEnd: now.Format(time.RFC3339), FirstLedger: asOf - 720, LastLedger: asOf, SourceLedger: asOf},
			Baseline: &gateway.HomeInsightBaseline{Value: 7, WindowStart: now.Add(-169 * time.Hour).Format(time.RFC3339), WindowEnd: now.Add(-time.Hour).Format(time.RFC3339), CompleteHourCount: 168, ZeroBaselinePolicy: "omit_ratio_insight"},
			Ratio:    6,
			Facts: &gateway.HomeInsightFacts{Failure: &gateway.HomeInsightFailureFacts{
				Kind: "failure_spike", AttemptCount: 110, SuccessCount: 68, FailureCount: 42, DistinctTransactionCount: 42, DistinctCallerCount: 19, NetworkFailureCount: 77, SubjectFailureShare: 42.0 / 77.0,
			}},
			PrimaryContributor: &gateway.HomeInsightContribution{Dimension: "function", Kind: "function", Key: "swap", Count: 38, DenominatorName: "subject_failure_count", DenominatorValue: 42, Share: 38.0 / 42.0, FirstLedger: asOf - 720, LastLedger: asOf - 14},
			EvidenceLocator:    &gateway.HomeInsightEvidenceLocator{Kind: "contract_invocations", ContractID: "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD2KM", LedgerStart: asOf - 720, LedgerEnd: asOf, Status: "failed"},
			EvidenceCount:      42,
			Status:             "ready",
			Caveats:            &caveats,
			EvidenceProvenance: &gateway.HomeInsightEvidenceProvenance{Sources: []string{"demo_fixture"}, CompleteThroughLedger: asOf, UpdatedAt: now.Add(5 * time.Second).Format(time.RFC3339)},
		}, {
			// A second insight so the fixture exercises the multi-card layout.
			// .ph-insight-list gets a has-N class from the card count, and the
			// horizontal grid only engages at has-2 or has-3; with a single
			// fixture card the section always rendered as one stacked column,
			// which read as the row treatment having been lost.
			InsightID:       "hiev1_Xq4mB2pTn7RkLs0WvYcHdGj9FaZ1eNuI3oQrSt5UvWx",
			Network:         network,
			Type:            "transaction_activity_spike",
			EvidenceVersion: "home_insight_evidence_v1",
			Definition: &gateway.HomeInsightDefinition{
				RuleID: "network_transaction_activity_spike", RuleVersion: "1", ComparisonMethod: "rolling_7d_median_prior_complete_hour", MinimumRatio: 2,
			},
			Subject:  gateway.HomeSummaryInsightSubject{Kind: "network", ID: network},
			Observed: &gateway.HomeInsightObserved{Value: 5347, WindowStart: now.Add(-time.Hour).Format(time.RFC3339), WindowEnd: now.Format(time.RFC3339), FirstLedger: asOf - 720, LastLedger: asOf, SourceLedger: asOf},
			Baseline: &gateway.HomeInsightBaseline{Value: 2410, WindowStart: now.Add(-169 * time.Hour).Format(time.RFC3339), WindowEnd: now.Add(-time.Hour).Format(time.RFC3339), CompleteHourCount: 168, ZeroBaselinePolicy: "omit_ratio_insight"},
			Ratio:    5347.0 / 2410.0,
			Facts: &gateway.HomeInsightFacts{Activity: &gateway.HomeInsightActivityFacts{
				Kind: "transaction_activity_spike", IncludedTransactionCount: 5347,
				SuccessfulTransactionCount: 5306, FailedTransactionCount: 41,
				IncludedOperationCount: 11208, SorobanTransactionCount: 2353, ClassicOnlyTransactionCount: 2994,
			}},
			EvidenceLocator:    &gateway.HomeInsightEvidenceLocator{Kind: "ledger_activity", LedgerStart: asOf - 720, LedgerEnd: asOf},
			EvidenceCount:      5347,
			Status:             "ready",
			Caveats:            &caveats,
			EvidenceProvenance: &gateway.HomeInsightEvidenceProvenance{Sources: []string{"demo_fixture"}, CompleteThroughLedger: asOf, UpdatedAt: now.Add(5 * time.Second).Format(time.RFC3339)},
		}},
		ContractsNeedingAttention: []gateway.HomeSummaryAttentionContract{
			{ContractID: "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD2KM", ProtocolName: "Demo protocol", ContractName: "Router", Severity: "critical", RemainingLedgers: 9800, RemainingHuman: "about 14 hours", NearestLiveUntilLedger: asOf + 9800, TrackedEntryCount: 12, ExpiringEntryCount: 3, DurabilityClasses: []string{"persistent"}},
			{ContractID: "CBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBVQG", ContractName: "Rewards pool", Severity: "warning", RemainingLedgers: 45000, RemainingHuman: "about 2 days", NearestLiveUntilLedger: asOf + 45000, TrackedEntryCount: 8, ExpiringEntryCount: 1, DurabilityClasses: []string{"contract instance"}},
		},
		Leaders: []gateway.HomeSummaryLeader{
			{ContractID: "CBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBVQG", DisplayName: "Demo liquidity router", Identity: gateway.HomeSummaryContractIdentity{Kind: "protocol_contract", VerificationStatus: "verified", Source: "demo_fixture"}, CallCount24h: 91, UniqueCallers24h: 24, SuccessCount: &succeeded, FailureCount: &failed, SuccessRate: &successRate, FailureRate: &failureRate, TopFunction: "swap", Window: "Last completed 24 hours", AsOfLedger: asOf, UpdatedAt: now.Format(time.RFC3339)},
		},
		Utilization: gateway.HomeSummaryUtilization{
			SourceLedger:            asOf,
			Instructions:            &gateway.HomeSummaryUtilizationMetric{Status: "ready", Used: &instructionUsed, Limit: &instructionLimit, Ratio: &instructionRatio, SourceLedger: asOf, LimitSource: "demo_ledger_configuration"},
			ReadWriteBytes:          &gateway.HomeSummaryUtilizationMetric{Status: "ready", Used: &readWriteUsed, Limit: &readWriteLimit, Ratio: &readWriteRatio, SourceLedger: asOf, LimitSource: "demo_ledger_configuration"},
			TransactionEnvelopeSize: &gateway.HomeSummaryTxSizeMetric{Status: "ready", AvgTxSizeBytes: &avgTxSize, TxSizeLimitBytes: &txSizeLimit, AvgRatio: &txSizeRatio, SourceLedger: asOf, LimitSource: "demo_ledger_configuration"},
		},
		Provenance: gateway.HomeSummaryProvenance{Route: "/home/summary", DataSource: "demo_fixture", GeneratedFrom: []string{"demo_fixture"}},
	}
}
