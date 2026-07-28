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
	data.TimelineURL = homeV2TimelineURL(network, true)
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
