package handlers

import (
	"fmt"
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/pages"
)

func (h *Handlers) NetworkHealth(w http.ResponseWriter, r *http.Request) {
	data := pages.NetworkHealthData{
		Status:       "Operational",
		StatusColor:  "emerald",
		LatestLedger: "5,104,938",
		LedgerAge:    "4s ago",
		AvgCloseTime: "5.2s",
		CurrentTPS:   "142",
		PeakTPS:      "312",
		Tx24h:        "1.2M",
		TxChange:     "+8.4%",
		Ops24h:       "3.8M",
		OpsPerTx:     "3.2",
		FailureRate:  "0.12%",
		FeeBase:      "100",
		FeeMedian:    "200",
		FeeP99:       "50,000",
		DailyFees:    "42,100 XLM",
		SorobanInvocations: "284,102",
		ActiveContracts:    "1,847",
		TotalState:         "2.4 GB",
		RentBurned:         "12,400 XLM",
		AvgCPU:             "18.2M insn",
		ProtocolVer:   "22",
		HorizonVer:    "2.30.0",
		SorobanRPCVer: "21.4.0",
		Agreement:     "97.1%",
		ValidatorCount: "35",
		QuorumSets:    "7",
		AvgLatency:    "1.2s",
		Validators: []pages.ValidatorRow{
			{Name: "SDF 1", Org: "SDF", Address: "GCGB2S...ZSTYH", Uptime: "99.95%", LastVote: "4s ago", Status: "Validating", StatusColor: "emerald"},
			{Name: "Blockdaemon 1", Org: "Blockdaemon", Address: "GA7UJ...K29XH", Uptime: "99.98%", LastVote: "4s ago", Status: "Validating", StatusColor: "emerald"},
			{Name: "SatoshiPay DE", Org: "SatoshiPay", Address: "GBFZF...Q72PT", Uptime: "99.92%", LastVote: "6s ago", Status: "Validating", StatusColor: "emerald"},
			{Name: "Lobstr Pool 1", Org: "Lobstr", Address: "GCWJK...H8R4P", Uptime: "99.87%", LastVote: "5s ago", Status: "Validating", StatusColor: "emerald"},
			{Name: "Franklin Temp.", Org: "Franklin", Address: "GCZJM...W2K18", Uptime: "99.99%", LastVote: "4s ago", Status: "Validating", StatusColor: "emerald"},
		},
		RecentLedgers: []pages.NetworkLedger{
			{Sequence: "5,104,938", Age: "4s ago", TxCount: "42", SorobanCalls: "18", Fees: "8,200", CloseTime: "5.1s", IsLatest: true},
			{Sequence: "5,104,937", Age: "9s ago", TxCount: "38", SorobanCalls: "12", Fees: "6,400", CloseTime: "5.3s"},
			{Sequence: "5,104,936", Age: "14s ago", TxCount: "51", SorobanCalls: "24", Fees: "12,100", CloseTime: "4.8s"},
			{Sequence: "5,104,935", Age: "20s ago", TxCount: "29", SorobanCalls: "8", Fees: "4,200", CloseTime: "5.6s"},
			{Sequence: "5,104,934", Age: "25s ago", TxCount: "44", SorobanCalls: "19", Fees: "9,800", CloseTime: "5.0s"},
		},
	}
	pages.NetworkHealth(data).Render(r.Context(), w)
}

func (h *Handlers) ValidatorDetail(w http.ResponseWriter, r *http.Request) {
	data := pages.ValidatorDetailData{
		Name:           "SDF 1",
		NodeType:       "Full Validator",
		NodeID:         "GCGB2S2KGYARPVIA37HYZXVRM2YZUEXA6S33ZU5BUDC6THSB62LZSTYH",
		ShortNodeID:    "GCGB2S...ZSTYH",
		StatusBadge:    "Validating",
		Badges:         []string{"Tier 1", "Archive Publisher"},
		NodeIndex:      "0.83",
		IndexWidth:     "83%",
		Validating24H:  "100%",
		Val24HWidth:    "100%",
		Validating30D:  "99.95%",
		Val30DWidth:    "99.95%",
		CrawlerReject:  "57.3%",
		CrawlerWidth:   "57.3%",
		TrustsCount:    "21",
		TrustsOrgs:     "7",
		TrustedByCount: "91",
		ExtLag:         "0 ms",
		Host:           "core-live-a.stellar.org:11625",
		IP:             "34.227.72.189:11625",
		ISP:            "Amazon.com Inc.",
		Domain:         "www.stellar.org",
		Organization:   "Stellar Dev. Foundation",
		Country:        "United States",
		Version:        "stellar-core 25.2.0.rc2",
		Overlay:        "39 (min 38)",
		LedgerVer:      "25",
		HistoryURL:     "history.stellar.org/prd/.../001/",
		Discovered:     "May 31, 2019",
		OrgName:        "SDF",
		OrgNodes: []pages.OrgNode{
			{Name: "SDF 1", ShortID: "GCGB2S...ZSTYH", Uptime: "99.95%", IsViewing: true},
			{Name: "SDF 2", ShortID: "GCM6QM...QSTK", Uptime: "99.98%"},
			{Name: "SDF 3", ShortID: "GABMKJ...XHYQ", Uptime: "99.97%"},
		},
		OrgUptime:      "99.97%",
		OrgValidators:  "3",
		QuorumThreshold: "Threshold 5 of 7",
		QuorumSets: []pages.QuorumSet{
			{Name: "Blockdaemon", Threshold: "2 of 3", Nodes: []pages.QuorumNode{{Name: "BD 1"}, {Name: "BD 2"}, {Name: "BD 3"}}},
			{Name: "SDF", IsSelf: true, Threshold: "2 of 3", Nodes: []pages.QuorumNode{{Name: "SDF 1", IsSelf: true}, {Name: "SDF 2"}, {Name: "SDF 3"}}},
			{Name: "SatoshiPay", Threshold: "2 of 3", Nodes: []pages.QuorumNode{{Name: "SP DE"}, {Name: "SP SG"}, {Name: "SP US"}}},
			{Name: "Lobstr", Threshold: "2 of 3", Nodes: []pages.QuorumNode{{Name: "Lobstr 1"}, {Name: "Lobstr 2"}, {Name: "Lobstr 3"}}},
			{Name: "PublicNode", Threshold: "2 of 3", Nodes: []pages.QuorumNode{{Name: "PN DE"}, {Name: "PN FI"}, {Name: "PN US"}}},
			{Name: "Coinqvest", Threshold: "2 of 3", Nodes: []pages.QuorumNode{{Name: "CQ DE"}, {Name: "CQ FI"}, {Name: "CQ HK"}}},
			{Name: "Franklin", Threshold: "1 of 1", Nodes: []pages.QuorumNode{{Name: "FT 1"}}},
		},
	}
	pages.ValidatorDetail(data).Render(r.Context(), w)
}

func (h *Handlers) ValidatorPreview(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	fmt.Fprintf(w, `<div class="slideout-content">Preview: %s</div>`, id)
}
