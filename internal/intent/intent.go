package intent

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/withObsrvr/prism/internal/gateway"
	prismsearch "github.com/withObsrvr/prism/internal/search"
)

type ID string

const (
	ExpiringContracts  ID = "expiring_contracts"
	ProtocolBusy       ID = "protocol_busy"
	TransactionFailure ID = "transaction_failure"
	ContractActivity   ID = "contract_activity"
	AssetActivity      ID = "asset_activity"
	RecentFailures     ID = "recent_failures"
)

type Match struct {
	ID         ID
	Confidence float64
	Slots      map[string]string
	Reason     string
}

type EvidenceLink struct {
	Label string
	Href  string
}

type ActionLink struct {
	Label string
	Href  string
}

type Metric struct {
	Label string
	Value string
}

type Result struct {
	Title             string
	Answer            string
	Confidence        float64
	EvidenceAvailable bool
	Evidence          []EvidenceLink
	Actions           []ActionLink
	Metrics           []Metric
	Warnings          []string
}

type Env struct {
	Gateway *gateway.Client
	Network string
}

type Handler interface {
	ID() ID
	Match(input string, reg *Registry) (Match, bool)
	Execute(ctx context.Context, env Env, m Match) (Result, error)
}

type Registry struct {
	Protocols  []Protocol
	Handlers   []Handler
	Thresholds BusyThresholds
}

type Protocol struct {
	Key       string
	Name      string
	Aliases   []string
	Contracts map[string][]string
	ExploreQ  string
}

type BusyThresholds struct {
	QuietMaxCalls  int64
	NormalMaxCalls int64
}

func DefaultRegistry() *Registry {
	return &Registry{
		Protocols: []Protocol{
			{
				Key:      "soroswap",
				Name:     "Soroswap",
				Aliases:  []string{"soroswap", "soroswap router", "soroswap factory"},
				ExploreQ: "Soroswap",
				Contracts: map[string][]string{
					"testnet": {
						"CCJUD55AG6W5HAI5LRVNKAE5WDP5XGZBUDS5WNTIVDU7O264UZZE7BRD", // Router
						"CDP3HMUH6SMS3S7NPGNDJLULCOXXEPSHY4JKUKMBNQMATHDHWXRRJTBY", // Factory
					},
				},
			},
			{Key: "phoenix", Name: "Phoenix AMM", Aliases: []string{"phoenix", "phoenix amm"}, ExploreQ: "Phoenix"},
			{Key: "blend", Name: "Blend", Aliases: []string{"blend", "blend pool"}, ExploreQ: "Blend"},
		},
		Handlers: []Handler{
			transactionFailureHandler{},
			contractActivityHandler{},
			assetActivityHandler{},
			recentFailuresHandler{},
			expiringContractsHandler{},
			protocolBusyHandler{},
		},
		Thresholds: BusyThresholds{QuietMaxCalls: 50, NormalMaxCalls: 500},
	}
}

func (r *Registry) Match(input string) (Match, bool) {
	if r == nil {
		r = DefaultRegistry()
	}
	if prismsearch.Classify(input).Known() {
		return Match{}, false
	}
	var best Match
	for _, h := range r.Handlers {
		m, ok := h.Match(input, r)
		if ok && m.Confidence > best.Confidence {
			best = m
		}
	}
	return best, best.ID != ""
}

func (r *Registry) Execute(ctx context.Context, env Env, m Match) (Result, error) {
	if r == nil {
		r = DefaultRegistry()
	}
	for _, h := range r.Handlers {
		if h.ID() == m.ID {
			return h.Execute(ctx, env, m)
		}
	}
	return Result{}, fmt.Errorf("intent: no handler for %s", m.ID)
}

func (r *Registry) Protocol(key string) (Protocol, bool) {
	for _, p := range r.Protocols {
		if p.Key == key {
			return p, true
		}
	}
	return Protocol{}, false
}

func normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer("?", " ", "!", " ", ",", " ", ".", " ", "'", "").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

func timeSlot(q string) string {
	n := normalize(q)
	switch {
	case strings.Contains(n, "this month") || strings.Contains(n, "past month") || strings.Contains(n, "last month"):
		return "30d"
	case strings.Contains(n, "this week") || strings.Contains(n, "past week") || strings.Contains(n, "last week") || strings.Contains(n, "soon"):
		return "7d"
	case strings.Contains(n, "today") || strings.Contains(n, "24h") || strings.Contains(n, "24 hours"):
		return "24h"
	case strings.Contains(n, "hour"):
		return "1h"
	default:
		return ""
	}
}

func findProtocol(q string, protocols []Protocol) (Protocol, bool) {
	n := normalize(q)
	for _, p := range protocols {
		aliases := append([]string{p.Key, p.Name}, p.Aliases...)
		for _, alias := range aliases {
			if strings.Contains(n, normalize(alias)) {
				return p, true
			}
		}
	}
	return Protocol{}, false
}

type expiringContractsHandler struct{}

func (expiringContractsHandler) ID() ID { return ExpiringContracts }

var expiringRE = regexp.MustCompile(`\b(contract|contracts|ttl|storage)\b.*\b(expir|archiv|attention|runway|risk)|\b(expir|archiv|attention|runway|risk)\b.*\b(contract|contracts|ttl|storage)\b`)

func (expiringContractsHandler) Match(input string, _ *Registry) (Match, bool) {
	n := normalize(input)
	if !expiringRE.MatchString(n) {
		return Match{}, false
	}
	return Match{ID: ExpiringContracts, Confidence: 0.88, Slots: map[string]string{"requested_time": timeSlot(n), "window": "current_ttl_snapshot"}, Reason: "contract TTL/expiration wording"}, true
}

func (expiringContractsHandler) Execute(ctx context.Context, env Env, m Match) (Result, error) {
	res := Result{Title: "Contracts near expiration", Confidence: m.Confidence}
	if env.Gateway == nil {
		res.Answer = "I can answer this when live gateway data is available."
		res.Warnings = []string{"Gateway client is not configured."}
		res.Actions = []ActionLink{{Label: "Explore contracts", Href: "/v2/explore?scope=soroban"}}
		return res, nil
	}
	summary, err := env.Gateway.GetHomeSummary(ctx, env.Network)
	if err != nil {
		return res, err
	}
	componentStatus := strings.ToLower(strings.TrimSpace(summary.Components.TTLAttention.Status))
	if componentStatus == "unavailable" || componentStatus == "error" {
		res.Answer = "The current contract TTL evidence is unavailable, so Prism cannot identify contracts with the least runway."
		res.Warnings = append(res.Warnings, "The Gateway marked the TTL attention component unavailable.")
		res.Actions = []ActionLink{{Label: "Explore contracts", Href: "/v2/explore?scope=soroban"}}
		return res, nil
	}
	res.EvidenceAvailable = true
	count := summary.Hero.TTL.ExpiringContractCount
	if count == 0 {
		count = int64(len(summary.ContractsNeedingAttention))
	}
	if count == 0 {
		res.Answer = "The current TTL attention snapshot does not flag any contracts as needing attention."
	} else {
		res.Metrics = append(res.Metrics, Metric{Label: "Contracts", Value: fmt.Sprintf("%d", count)})
		res.Answer = fmt.Sprintf("The current TTL attention snapshot flags %d contracts as having limited runway", count)
		if summary.Hero.TTL.WorstRemainingHours > 0 {
			res.Answer += ". The shortest runway is about " + formatHours(summary.Hero.TTL.WorstRemainingHours)
			res.Metrics = append(res.Metrics, Metric{Label: "Shortest runway", Value: fmt.Sprintf("%dh", summary.Hero.TTL.WorstRemainingHours)})
		}
		res.Answer += "."
	}
	if requested := m.Slots["requested_time"]; requested != "" {
		res.Warnings = append(res.Warnings, "This answer uses the current TTL attention snapshot. It does not prove which contracts will archive within the requested calendar window.")
	}
	if componentStatus == "partial" || componentStatus == "stale" {
		res.Warnings = append(res.Warnings, "The Gateway marked TTL attention evidence "+componentStatus+".")
	}
	for _, c := range summary.ContractsNeedingAttention {
		label := strings.TrimSpace(c.ProtocolName + " " + c.ContractName)
		if label == "" {
			label = short(c.ContractID)
		}
		if c.RemainingHuman != "" {
			label += " · " + c.RemainingHuman
		}
		res.Evidence = append(res.Evidence, EvidenceLink{Label: label, Href: "/v2/contract/" + url.PathEscape(c.ContractID)})
		if len(res.Evidence) >= 5 {
			break
		}
	}
	res.Actions = []ActionLink{{Label: "Explore Soroban activity", Href: "/v2/explore?scope=soroban"}, {Label: "Home summary", Href: "/v2/home"}}
	return res, nil
}

type protocolBusyHandler struct{}

func (protocolBusyHandler) ID() ID { return ProtocolBusy }

func (protocolBusyHandler) Match(input string, reg *Registry) (Match, bool) {
	n := normalize(input)
	p, ok := findProtocol(n, reg.Protocols)
	if !ok {
		return Match{}, false
	}
	busyWords := []string{"busy", "active", "activity", "quiet", "used", "usage", "calls", "swaps"}
	matched := false
	for _, w := range busyWords {
		if strings.Contains(n, w) {
			matched = true
			break
		}
	}
	if !matched {
		return Match{}, false
	}
	t := timeSlot(n)
	if t == "" {
		t = "24h"
	}
	return Match{ID: ProtocolBusy, Confidence: 0.82, Slots: map[string]string{"protocol": p.Key, "time": t}, Reason: "known protocol activity wording"}, true
}

func (protocolBusyHandler) Execute(ctx context.Context, env Env, m Match) (Result, error) {
	reg := DefaultRegistry()
	p, _ := reg.Protocol(m.Slots["protocol"])
	res := Result{Title: "Protocol activity: " + p.Name, Confidence: m.Confidence}
	exploreHref := "/v2/explore?q=" + url.QueryEscape(p.ExploreQ)
	res.Actions = []ActionLink{{Label: "Explore " + p.Name, Href: exploreHref}}
	if env.Gateway == nil {
		res.Answer = "I can check whether " + p.Name + " is busy when live gateway data is available."
		res.Warnings = []string{"Gateway client is not configured."}
		return res, nil
	}
	// Prefer the explicitly windowed 24-hour leader evidence when available.
	if summary, err := env.Gateway.GetHomeSummary(ctx, env.Network); err == nil && summary != nil {
		for _, l := range summary.Leaders {
			if strings.Contains(normalize(l.ProtocolName+" "+l.ContractName), normalize(p.Name)) || strings.Contains(normalize(l.ProtocolName), p.Key) {
				band := busyBand(l.CallCount24h, reg.Thresholds)
				res.Answer = fmt.Sprintf("%s looks %s: %d calls and %d unique callers in the last 24 hours.", p.Name, band, l.CallCount24h, l.UniqueCallers24h)
				res.Metrics = []Metric{{Label: "Calls", Value: fmt.Sprintf("%d", l.CallCount24h)}, {Label: "Unique callers", Value: fmt.Sprintf("%d", l.UniqueCallers24h)}}
				res.Evidence = []EvidenceLink{{Label: strings.TrimSpace(l.ProtocolName + " " + l.ContractName), Href: "/v2/contract/" + url.PathEscape(l.ContractID)}}
				res.EvidenceAvailable = true
				return res, nil
			}
		}
	}

	contracts := p.Contracts[env.Network]
	if len(contracts) == 0 {
		res.Answer = "I know " + p.Name + ", but I do not have registered contract IDs for " + env.Network + " yet, so I cannot compute a busy score."
		res.Warnings = []string{"Add protocol contract IDs to the intent registry for this network."}
		return res, nil
	}
	var totalCalls, uniqueCallers, recent7d int64
	loadedContracts := 0
	for _, id := range contracts {
		analytics, err := env.Gateway.GetContractAnalytics(ctx, env.Network, id)
		if err != nil {
			res.Warnings = append(res.Warnings, "Could not load analytics for "+short(id)+": "+err.Error())
			continue
		}
		loadedContracts++
		totalCalls += analytics.Stats.TotalCallsAsCallee + analytics.Stats.TotalCallsAsCaller
		uniqueCallers += analytics.Stats.UniqueCallers
		for _, d := range analytics.DailyCalls7D {
			recent7d += d.Count
		}
		res.Evidence = append(res.Evidence, EvidenceLink{Label: p.Name + " contract " + short(id), Href: "/v2/contract/" + url.PathEscape(id)})
	}
	res.EvidenceAvailable = loadedContracts > 0
	res.Answer = fmt.Sprintf("The available contract analytics report %d all-time observed calls for %s", totalCalls, p.Name)
	res.Metrics = []Metric{{Label: "All-time calls", Value: fmt.Sprintf("%d", totalCalls)}, {Label: "Unique callers", Value: fmt.Sprintf("%d", uniqueCallers)}}
	if recent7d > 0 {
		res.Answer += fmt.Sprintf(" and %d calls across the returned seven daily buckets", recent7d)
		res.Metrics = append(res.Metrics, Metric{Label: "Seven daily buckets", Value: fmt.Sprintf("%d", recent7d)})
	}
	if uniqueCallers > 0 {
		res.Answer += fmt.Sprintf(" and %d unique callers", uniqueCallers)
	}
	res.Answer += ". Prism cannot assign the 24-hour busy band without 24-hour protocol evidence."
	res.Warnings = append(res.Warnings, "The fallback analytics are not the requested 24-hour window.")
	if len(res.Evidence) == 0 {
		res.Answer = "I found registered contracts for " + p.Name + ", but could not load enough analytics to compute a busy score."
	}
	return res, nil
}

func busyBand(score int64, t BusyThresholds) string {
	switch {
	case score <= t.QuietMaxCalls:
		return "quiet"
	case score <= t.NormalMaxCalls:
		return "moderately active"
	default:
		return "busy"
	}
}

func humanTime(s string) string {
	switch s {
	case "1h":
		return "last hour"
	case "24h":
		return "last 24 hours"
	case "7d":
		return "the last 7 days"
	case "30d":
		return "the last 30 days"
	default:
		return s
	}
}

func formatHours(hours int64) string {
	if hours == 1 {
		return "1 hour"
	}
	return fmt.Sprintf("%d hours", hours)
}

func short(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:6] + "…" + s[len(s)-6:]
}

func SortMatches(ms []Match) {
	sort.SliceStable(ms, func(i, j int) bool { return ms[i].Confidence > ms[j].Confidence })
}
