package handlers

import (
	"fmt"
	"net/http"

	"github.com/withObsrvr/prism/internal/templates/pages"
)

func (h *Handlers) NetworkHealth(w http.ResponseWriter, r *http.Request) {
	data := pages.NetworkHealthData{
		Status:             "Operational",
		StatusColor:        "emerald",
		LatestLedger:       "61,504,113",
		LedgerAge:          "4s ago",
		AvgCloseTime:       "5.2s",
		CurrentTPS:         "142",
		PeakTPS:            "312",
		Tx24h:              "1.2M",
		TxChange:           "+8.4%",
		Ops24h:             "3.8M",
		OpsPerTx:           "3.2",
		FailureRate:        "0.12%",
		FeeBase:            "100",
		FeeMedian:          "200",
		FeeP99:             "50,000",
		DailyFees:          "42,100 XLM",
		SorobanInvocations: "284,102",
		ActiveContracts:    "1,847",
		TotalState:         "2.4 GB",
		RentBurned:         "12,400 XLM",
		AvgCPU:             "18.2M insn",
		ProtocolVer:        "22",
		HorizonVer:         "2.30.0",
		SorobanRPCVer:      "21.4.0",
		Agreement:          "97.1%",
		ValidatorCount:     "35",
		QuorumSets:         "7",
		AvgLatency:         "1.2s",
		Validators: []pages.ValidatorRow{
			{Name: "SDF 1", Org: "SDF", Address: "GCGB2S...ZSTYH", Uptime: "99.95%", LastVote: "4s ago", Status: "Validating", StatusColor: "emerald"},
			{Name: "Blockdaemon 1", Org: "Blockdaemon", Address: "GA7UJ...K29XH", Uptime: "99.98%", LastVote: "4s ago", Status: "Validating", StatusColor: "emerald"},
			{Name: "SatoshiPay DE", Org: "SatoshiPay", Address: "GBFZF...Q72PT", Uptime: "99.92%", LastVote: "6s ago", Status: "Validating", StatusColor: "emerald"},
			{Name: "Lobstr Pool 1", Org: "Lobstr", Address: "GCWJK...H8R4P", Uptime: "99.87%", LastVote: "5s ago", Status: "Validating", StatusColor: "emerald"},
			{Name: "Franklin Temp.", Org: "Franklin", Address: "GCZJM...W2K18", Uptime: "99.99%", LastVote: "4s ago", Status: "Validating", StatusColor: "emerald"},
		},
		RecentLedgers: []pages.NetworkLedger{
			{Sequence: "61,504,113", Age: "4s ago", TxCount: "42", SorobanCalls: "18", Fees: "8,200", CloseTime: "5.1s", IsLatest: true},
			{Sequence: "61,504,112", Age: "9s ago", TxCount: "38", SorobanCalls: "12", Fees: "6,400", CloseTime: "5.3s"},
			{Sequence: "61,504,111", Age: "14s ago", TxCount: "51", SorobanCalls: "24", Fees: "12,100", CloseTime: "4.8s"},
			{Sequence: "61,504,110", Age: "20s ago", TxCount: "29", SorobanCalls: "8", Fees: "4,200", CloseTime: "5.6s"},
			{Sequence: "61,504,109", Age: "25s ago", TxCount: "44", SorobanCalls: "19", Fees: "9,800", CloseTime: "5.0s"},
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
		OrgUptime:       "99.97%",
		OrgValidators:   "3",
		QuorumThreshold: "Threshold 5 of 7",
		QuorumSets: []pages.QuorumSet{
			{Name: "Blockdaemon", Threshold: "2 of 3", Nodes: []pages.QuorumNode{{Name: "BD 1"}, {Name: "BD 2"}, {Name: "BD 3"}}},
			{Name: "SDF", IsSelf: true, Threshold: "2 of 3", Nodes: []pages.QuorumNode{{Name: "SDF 1", IsSelf: true}, {Name: "SDF 2"}, {Name: "SDF 3"}}},
			{Name: "SatoshiPay", Threshold: "2 of 3", Nodes: []pages.QuorumNode{{Name: "Frankfurt"}, {Name: "Singapore"}, {Name: "Iowa"}}},
			{Name: "Franklin T.", Threshold: "2 of 3", Nodes: []pages.QuorumNode{{Name: "FT SCV 1"}, {Name: "FT SCV 2"}, {Name: "FT SCV 3"}}},
			{Name: "LOBSTR", Threshold: "2 of 3", Nodes: []pages.QuorumNode{{Name: "LOBSTR 1"}, {Name: "LOBSTR 2"}, {Name: "LOBSTR 3"}}},
			{Name: "Creit Tech", Threshold: "2 of 3", Nodes: []pages.QuorumNode{{Name: "Hercules"}, {Name: "Lyra"}, {Name: "Boötes"}}},
			{Name: "Public Node", Threshold: "2 of 3", Nodes: []pages.QuorumNode{{Name: "PN 1"}, {Name: "PN 2"}, {Name: "PN 3"}}},
		},
		Trusts: []pages.TrustRow{
			{Name: "Blockdaemon Validator 1", Badge: "Warning", BadgeColor: "amber", Index: "1", TrustWeight: "10", TrustPct: "100%", Valid24H: "100%", Color24H: "gray-900", Valid30D: "99.85%", Color30D: "amber-600", Version: "25.1.3", Country: "Belgium", ISP: "Google LLC"},
			{Name: "Blockdaemon Validator 2", Badge: "Warning", BadgeColor: "amber", Index: "1", TrustWeight: "10", TrustPct: "100%", Valid24H: "100%", Color24H: "gray-900", Valid30D: "99.93%", Color30D: "emerald-600", Version: "25.1.3", Country: "United States", ISP: "Google LLC"},
			{Name: "Blockdaemon Validator 3", Badge: "Warning", BadgeColor: "amber", Index: "1", TrustWeight: "10", TrustPct: "100%", Valid24H: "100%", Color24H: "gray-900", Valid30D: "99.89%", Color30D: "amber-600", Version: "25.1.3", Country: "Taiwan", ISP: "Google LLC"},
			{Name: "SatoshiPay Singapore", Index: "1", TrustWeight: "10", TrustPct: "100%", Valid24H: "100%", Color24H: "gray-900", Valid30D: "99.9%", Color30D: "emerald-600", Version: "25.1.3", Country: "United Kingdom", ISP: "Contabo Asia"},
			{Name: "SatoshiPay Frankfurt", Badge: "Warning", BadgeColor: "amber", Index: "1", TrustWeight: "10", TrustPct: "100%", Valid24H: "99.65%", Color24H: "gray-900", Valid30D: "99.71%", Color30D: "amber-600", Version: "25.1.3", Country: "Germany", ISP: "Contabo Gmbh"},
			{Name: "FT SCV 1", Index: "1", TrustWeight: "10", TrustPct: "100%", Valid24H: "100%", Color24H: "gray-900", Valid30D: "99.92%", Color30D: "emerald-600", Version: "25.1.3", Country: "United States", ISP: "Google LLC"},
		},
		Updates: []pages.NodeUpdate{
			{IconBg: "bg-cyan-100", IconColor: "text-cyan-600", IconSVG: "refresh", Date: "2/27/2026, 11:07 AM", Detail: "Overlay updated to version 39 · Min overlay now 38 · Core updated to 25.2.0.rc2"},
			{IconBg: "bg-violet-100", IconColor: "text-violet-600", IconSVG: "bolt", Date: "2/22/2026, 4:17 AM", Detail: "Stellar core updated to version 25.1.3"},
			{IconBg: "bg-cyan-100", IconColor: "text-cyan-600", IconSVG: "refresh", Date: "2/22/2026, 1:47 AM", Detail: "Overlay updated to version 38 · Min overlay now 35 · Core updated to 25.1.2"},
			{IconBg: "bg-amber-100", IconColor: "text-amber-600", IconSVG: "location", Date: "1/28/2026, 5:11 AM", Detail: "IP changed to 34.227.72.189"},
			{IconBg: "bg-violet-100", IconColor: "text-violet-600", IconSVG: "bolt", Date: "1/23/2026, 7:06 PM", Detail: "Stellar core updated to version 25.1.0"},
		},
	}
	pages.ValidatorDetail(data).Render(r.Context(), w)
}

func (h *Handlers) ValidatorPreview(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<div class="bg-white border-l border-gray-200 shadow-2xl h-full overflow-y-auto slide-panel">
  <div class="sticky top-0 z-10 flex items-center justify-between px-6 py-4 bg-white border-b border-gray-100">
    <button class="flex items-center gap-1.5 text-[13px] font-medium text-gray-500 hover:text-gray-900 transition-colors"
            onclick="document.getElementById('slideout').innerHTML=''; document.getElementById('slideout-backdrop').classList.add('hidden')">
      <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"/></svg>
      Back
    </button>
    <button class="text-gray-400 hover:text-gray-900 transition-colors"
            onclick="document.getElementById('slideout').innerHTML=''; document.getElementById('slideout-backdrop').classList.add('hidden')">
      <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>
    </button>
  </div>
  <div class="px-6 py-5">
    <div class="flex items-center gap-3 mb-5">
      <div class="relative">
        <div class="flex h-11 w-11 items-center justify-center rounded-xl bg-gradient-to-br from-cyan-100 to-blue-50 ring-1 ring-gray-200">
          <svg class="h-5 w-5 text-cyan-600" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"/></svg>
        </div>
        <div class="absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full bg-emerald-500 ring-2 ring-white"></div>
      </div>
      <div>
        <div class="text-lg font-bold text-gray-900">%s</div>
        <div class="flex items-center gap-1.5">
          <span class="font-mono text-[11px] text-gray-400">GAYXZ4...UJPT6EZ4</span>
        </div>
      </div>
    </div>
    <div class="flex items-center gap-1.5 mb-6">
      <span class="rounded-full bg-emerald-50 px-2 py-0.5 ring-1 ring-emerald-200/80 text-[10px] font-semibold text-emerald-700">Validating</span>
      <span class="rounded-full bg-gray-100 px-2 py-0.5 ring-1 ring-gray-200 text-[10px] font-semibold text-gray-600">Tier 1</span>
    </div>
    <div class="grid grid-cols-3 gap-3 mb-6">
      <div class="rounded-lg border border-gray-200 bg-gray-50/50 px-3 py-2.5 text-center">
        <div class="text-[9px] font-semibold uppercase text-gray-400">Node Index</div>
        <div class="text-lg font-bold text-gray-900 tabular">1</div>
      </div>
      <div class="rounded-lg border border-gray-200 bg-gray-50/50 px-3 py-2.5 text-center">
        <div class="text-[9px] font-semibold uppercase text-gray-400">24H Valid.</div>
        <div class="text-lg font-bold text-emerald-600 tabular">100%%</div>
      </div>
      <div class="rounded-lg border border-gray-200 bg-gray-50/50 px-3 py-2.5 text-center">
        <div class="text-[9px] font-semibold uppercase text-gray-400">30D Valid.</div>
        <div class="text-lg font-bold text-emerald-600 tabular">99.85%%</div>
      </div>
    </div>
    <div class="rounded-lg border border-gray-200 overflow-hidden mb-6">
      <div class="detail-row flex items-center justify-between px-4 py-2.5 border-b border-gray-50">
        <span class="text-xs text-gray-500">Organization</span>
        <span class="text-xs font-medium text-emerald-700">Blockdaemon Inc.</span>
      </div>
      <div class="detail-row flex items-center justify-between px-4 py-2.5 border-b border-gray-50">
        <span class="text-xs text-gray-500">Domain</span>
        <span class="text-xs font-medium text-emerald-700">stellar.blockdaemon.com</span>
      </div>
      <div class="detail-row flex items-center justify-between px-4 py-2.5 border-b border-gray-50">
        <span class="text-xs text-gray-500">Host</span>
        <span class="font-mono text-[11px] text-gray-900">stellar-full-validator1.bdnodes.net</span>
      </div>
      <div class="detail-row flex items-center justify-between px-4 py-2.5 border-b border-gray-50">
        <span class="text-xs text-gray-500">Version</span>
        <span class="text-xs text-gray-900">stellar-core 25.1.3</span>
      </div>
      <div class="detail-row flex items-center justify-between px-4 py-2.5 border-b border-gray-50">
        <span class="text-xs text-gray-500">Country</span>
        <span class="text-xs text-gray-900">Belgium</span>
      </div>
      <div class="detail-row flex items-center justify-between px-4 py-2.5">
        <span class="text-xs text-gray-500">ISP</span>
        <span class="text-xs text-gray-900">Google LLC</span>
      </div>
    </div>
    <div class="rounded-lg border border-emerald-200/60 bg-emerald-50/30 p-4 mb-6">
      <div class="flex items-center gap-2 mb-2">
        <svg class="h-4 w-4 text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"/></svg>
        <span class="text-xs font-semibold text-emerald-800">Relationship to SDF 1</span>
      </div>
      <div class="text-xs text-gray-600 leading-relaxed">
        SDF 1 includes this validator in its quorum set. This is a <span class="font-semibold text-gray-900">mutual trust</span> relationship.
      </div>
    </div>
    <div class="mb-6">
      <div class="text-[10px] font-semibold uppercase tracking-wider text-gray-400 mb-2">30D Validating</div>
      <div class="flex items-end gap-[2px] h-10">
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
        <div class="flex-1 rounded-t-[1px] bg-emerald-500" style="height:100%%"></div>
      </div>
    </div>
    <div class="flex gap-2">
      <a href="/network/validators/%s" class="flex-1 flex items-center justify-center gap-2 rounded-lg bg-gray-900 px-4 py-2.5 text-[13px] font-medium text-white hover:bg-gray-800 transition-colors">
        <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/></svg>
        Open Full Page
      </a>
    </div>
  </div>
</div>`, id, id)
}
