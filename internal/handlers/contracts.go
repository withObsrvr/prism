package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/templates/pages"
)

func (h *Handlers) ContractList(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	var data pages.ContractListData

	if h.Gateway != nil {
		contracts, err := h.Gateway.GetTopContracts(r.Context(), network, 20)
		if err != nil {
			h.Logger.Warn("gateway error, using mock data for contracts", "error", err)
			data = mockContractListData()
		} else {
			items := make([]pages.ContractListItem, 0, len(contracts))
			for i, c := range contracts {
				items = append(items, pages.ContractListItem{
					Rank:        i + 1,
					Name:        gateway.ShortAddress(c.ContractID),
					Address:     gateway.ShortAddress(c.ContractID),
					Tag:         "Contract",
					TagColor:    "violet",
					Invocations: gateway.FormatNumber(c.TotalCalls),
					Change:      "",
					IsPositive:  true,
				})
			}
			data = pages.ContractListData{Contracts: items}
		}
	} else {
		data = mockContractListData()
	}

	pages.ContractList(data).Render(r.Context(), w)
}

func (h *Handlers) ContractDetail(w http.ResponseWriter, r *http.Request) {
	network := networkFromRequest(r)
	contractID := r.PathValue("id")

	var data pages.ContractDetailData

	if h.Gateway != nil {
		d, err := h.buildContractDetailData(r, network, contractID)
		if err != nil {
			h.Logger.Warn("gateway error, using mock data for contract detail", "error", err, "contract", contractID)
			data = mockContractDetailData()
		} else {
			data = d
		}
	} else {
		data = mockContractDetailData()
	}

	pages.ContractDetail(data).Render(r.Context(), w)
}

func (h *Handlers) buildContractDetailData(r *http.Request, network, contractID string) (pages.ContractDetailData, error) {
	ctx := r.Context()

	analytics, err := h.Gateway.GetContractAnalytics(ctx, network, contractID)
	if err != nil {
		return pages.ContractDetailData{}, fmt.Errorf("fetching contract analytics: %w", err)
	}

	// Get recent calls.
	recentCalls, _ := h.Gateway.GetContractRecentCalls(ctx, network, contractID, 10)

	// Build functions list.
	var funcs []pages.ContractFunction
	for _, f := range analytics.TopFunctions {
		funcs = append(funcs, pages.ContractFunction{
			Name:    f.Name,
			Calls24h: gateway.FormatNumber(f.Count),
		})
	}

	// Build daily call activity points.
	var activityPoints []float64
	for _, d := range analytics.DailyCalls7D {
		activityPoints = append(activityPoints, float64(d.Count))
	}

	totalCalls := analytics.Stats.TotalCallsAsCaller + analytics.Stats.TotalCallsAsCallee

	// Build invocations from recent calls.
	var invocations []pages.ContractInvocation
	for _, call := range recentCalls {
		statusColor := "emerald"
		status := "Success"
		if !call.Successful {
			statusColor = "red"
			status = "Failed"
		}
		age := "—"
		if t, err := time.Parse(time.RFC3339, call.ClosedAt); err == nil {
			d := time.Since(t)
			if d.Hours() > 24 {
				age = fmt.Sprintf("%.0fd ago", d.Hours()/24)
			} else if d.Hours() > 1 {
				age = fmt.Sprintf("%.0fh ago", d.Hours())
			} else if d.Minutes() > 1 {
				age = fmt.Sprintf("%.0fm ago", d.Minutes())
			} else {
				age = fmt.Sprintf("%.0fs ago", d.Seconds())
			}
		}
		invocations = append(invocations, pages.ContractInvocation{
			TxHash:      call.TransactionHash,
			ShortHash:   gateway.ShortHash(call.TransactionHash),
			Function:    call.FunctionName,
			Caller:      gateway.ShortAddress(call.SourceAccount),
			Status:      status,
			StatusColor: statusColor,
			Age:         age,
		})
	}

	// Compute last activity.
	lastActivity := "—"
	if analytics.Timeline.LastActivity != "" {
		if t, err := time.Parse(time.RFC3339, analytics.Timeline.LastActivity); err == nil {
			d := time.Since(t)
			if d.Hours() > 24 {
				lastActivity = fmt.Sprintf("%.0fd ago", d.Hours()/24)
			} else if d.Hours() > 1 {
				lastActivity = fmt.Sprintf("%.0fh ago", d.Hours())
			} else {
				lastActivity = fmt.Sprintf("%.0fm ago", d.Minutes())
			}
		}
	}

	deployedAt := "—"
	if analytics.Timeline.FirstSeen != "" {
		if t, err := time.Parse(time.RFC3339, analytics.Timeline.FirstSeen); err == nil {
			deployedAt = t.Format("Jan 2, 2006")
		}
	}

	data := pages.ContractDetailData{
		Name:             gateway.ShortAddress(contractID),
		Address:          contractID,
		TotalInvocations: gateway.FormatNumber(totalCalls),
		LastInvoked:      lastActivity,
		DeployedAt:       deployedAt,
		Functions:        funcs,
		Invocations:      invocations,
		ActivityPoints:   activityPoints,
		AvgInvocations:   gateway.FormatNumber(analytics.Stats.TotalCallsAsCallee),
		FunctionsCount:   fmt.Sprintf("%d", len(analytics.TopFunctions)),
	}

	return data, nil
}

func (h *Handlers) ContractEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if isHTMX(r) {
		fmt.Fprintf(w, `<div id="tab-content">Events for %s</div>`, id)
		return
	}
	fmt.Fprintf(w, "Contract Events: %s", id)
}
