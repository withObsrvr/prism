package handlers

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/withObsrvr/prism/internal/gateway"
	"github.com/withObsrvr/prism/internal/intent"
	prismsearch "github.com/withObsrvr/prism/internal/search"
)

func (h *Handlers) SearchSuggest(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	network := networkFromRequest(r)
	resolution, dynamicEntities := h.resolveSearchWithEntities(r.Context(), network, q)
	fmt.Fprint(w, `<div class="v2-suggest-panel">`)
	renderSearchResolutionSuggestion(w, resolution, true)

	seen := map[string]bool{resolution.Destination: true}
	remaining := 6
	for _, entity := range dynamicEntities {
		if remaining == 0 || seen[entity.Href] {
			continue
		}
		renderEntitySuggestion(w, entity)
		seen[entity.Href] = true
		remaining--
	}
	for _, entity := range prismsearch.DefaultRegistry().Search(q, remaining) {
		if remaining == 0 || seen[entity.Href] {
			continue
		}
		renderEntitySuggestion(w, entity)
		seen[entity.Href] = true
		remaining--
	}
	fmt.Fprintf(w, `<div class="v2-suggest-footer">Resolved by <code>%s</code> at %.0f%% confidence</div>`, html.EscapeString(resolution.RuleID), resolution.Confidence*100)
	fmt.Fprint(w, `</div>`)
}

func (h *Handlers) SearchSubmit(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.FormValue("q"))
	if q == "" {
		q = strings.TrimSpace(r.URL.Query().Get("q"))
	}
	if q == "" {
		target := "/v2/explore"
		if qs := exploreFormQuery(r); qs != "" {
			target += "?" + qs
		}
		hxOrRedirect(w, r, target)
		return
	}
	resolution := h.resolveSearch(r.Context(), networkFromRequest(r), q)
	if exploreFormQuery(r) != "" {
		parsed, _ := prismsearch.Parse(q)
		mergeExploreForm(r, &parsed)
		if qs := parsed.QueryString(); qs != "" {
			hxOrRedirect(w, r, "/v2/explore?"+qs)
			return
		}
	}
	hxOrRedirect(w, r, resolution.Destination)
}

func (h *Handlers) resolveSearch(ctx context.Context, network, query string) prismsearch.SearchResolution {
	resolution, _ := h.resolveSearchWithEntities(ctx, network, query)
	return resolution
}

func (h *Handlers) resolveSearchWithEntities(ctx context.Context, network, query string) (prismsearch.SearchResolution, []prismsearch.Entity) {
	ctx, cancel := context.WithTimeout(ctx, 1250*time.Millisecond)
	defer cancel()
	registry := intent.DefaultRegistry()
	resolver := prismsearch.NewResolver(prismsearch.DefaultRegistry())
	resolution := resolver.Resolve(query, prismsearch.ResolveOptions{MatchAnswer: func(input string) (prismsearch.AnswerCandidate, bool) {
		match, ok := registry.Match(input)
		if !ok {
			return prismsearch.AnswerCandidate{}, false
		}
		return prismsearch.AnswerCandidate{
			RuleID:     string(match.ID),
			Label:      searchAnswerLabel(match),
			Summary:    questionIntentSummary(match),
			Confidence: match.Confidence,
			Slots:      match.Slots,
		}, true
	}})
	if resolution.Kind == prismsearch.SearchOpen && resolution.Entity.Known() {
		if destination := h.detectSmartRedirect(ctx, network, resolution.Entity.Value); destination != "" {
			resolution.Destination = destination
			resolution.ActionLabel = searchOpenLabel(resolution.Entity, destination)
		}
	}
	dynamicEntities := []prismsearch.Entity(nil)
	if resolution.Kind == prismsearch.SearchUnsupported && resolution.RuleID != "unsupported.no_answer_rule" {
		dynamicEntities = h.dynamicSearchEntities(ctx, network, query, 4)
		if entity, ok := exactGatewayEntity(query, dynamicEntities); ok {
			destination := entity.Href
			if entity.Type == prismsearch.EntityContract {
				if smartDestination := h.detectSmartRedirect(ctx, network, entity.Value); smartDestination != "" {
					destination = smartDestination
				}
			}
			resolution = prismsearch.SearchResolution{
				Kind:        prismsearch.SearchOpen,
				Query:       strings.TrimSpace(query),
				ActionLabel: gatewayEntityActionLabel(entity, destination),
				Summary:     entity.Subtitle + ". Exact live entity match.",
				RuleID:      "gateway.entity." + string(entity.Type),
				Confidence:  0.9,
				Destination: destination,
				Slots:       map[string]string{"entity_type": string(entity.Type), "entity_value": entity.Value},
			}
		}
	}
	return resolution, dynamicEntities
}

func renderSearchResolutionSuggestion(w http.ResponseWriter, resolution prismsearch.SearchResolution, primary bool) {
	className := "v2-suggest-row"
	if primary {
		className += " primary"
	}
	if resolution.Kind == prismsearch.SearchUnsupported {
		className += " unsupported"
	}
	fmt.Fprintf(w, `<a class="%s" href="%s"><span><b>%s</b><small>%s</small></span><strong>%s</strong></a>`,
		html.EscapeString(className), html.EscapeString(resolution.Destination), html.EscapeString(resolution.ActionLabel), html.EscapeString(resolution.Summary), html.EscapeString(searchKindLabel(resolution.Kind)))
}

func renderEntitySuggestion(w http.ResponseWriter, entity prismsearch.Entity) {
	verb := "Open"
	if strings.Contains(entity.Href, "/explore") {
		verb = "Explore"
	}
	fmt.Fprintf(w, `<a class="v2-suggest-row" href="%s"><span><b>%s %s</b><small>%s</small></span><em>%s</em></a>`,
		html.EscapeString(entity.Href), verb, html.EscapeString(entity.Name), html.EscapeString(entity.Subtitle), verb)
}

func (h *Handlers) dynamicSearchEntities(ctx context.Context, network, query string, limit int) []prismsearch.Entity {
	query = strings.TrimSpace(query)
	if h.Gateway == nil || len([]rune(query)) < 2 || limit <= 0 || h.DataSource == "mock" {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 750*time.Millisecond)
	defer cancel()
	results, err := h.Gateway.Search(ctx, network, query)
	if err != nil {
		if h.Logger != nil {
			h.Logger.Warn("dynamic search suggestions unavailable", "network", network, "error", err)
		}
		return nil
	}
	entities := make([]prismsearch.Entity, 0, min(limit, len(results.Results)))
	for _, result := range results.Results {
		entity, ok := searchEntityFromGateway(result)
		if !ok {
			continue
		}
		entities = append(entities, entity)
		if len(entities) == limit {
			break
		}
	}
	return entities
}

func searchEntityFromGateway(result gateway.SearchResult) (prismsearch.Entity, bool) {
	id := strings.TrimSpace(result.ID)
	if id == "" {
		return prismsearch.Entity{}, false
	}
	label := strings.TrimSpace(result.Label)
	if label == "" {
		label = id
	}
	entity := prismsearch.Entity{Name: label, Value: id, Subtitle: "Gateway entity"}
	switch strings.ToLower(strings.TrimSpace(result.Type)) {
	case "account":
		entity.Type, entity.Href, entity.Subtitle = prismsearch.EntityAccount, "/v2/account/"+url.PathEscape(id), "Account from live search"
	case "smart_wallet":
		entity.Type, entity.Href, entity.Subtitle = prismsearch.EntityAccount, "/v2/account/"+url.PathEscape(id)+"/smart", "Smart account from live search"
	case "contract":
		entity.Type, entity.Href, entity.Subtitle = prismsearch.EntityContract, "/v2/contract/"+url.PathEscape(id), "Contract from live search"
	case "transaction", "tx":
		entity.Type, entity.Href, entity.Subtitle = prismsearch.EntityExplore, "/v2/tx/"+url.PathEscape(id), "Transaction from live search"
	case "ledger":
		entity.Type, entity.Href, entity.Subtitle = prismsearch.EntityExplore, "/v2/ledger/"+url.PathEscape(id), "Ledger from live search"
	case "asset":
		entity.Type, entity.Href, entity.Subtitle = prismsearch.EntityAsset, "/v2/assets/"+url.PathEscape(id), "Asset from live search"
	case "token":
		slug := assetSearchSlug(result)
		if slug == "" {
			slug = id
		}
		entity.Type, entity.Href, entity.Subtitle = prismsearch.EntityAsset, "/v2/assets/"+url.PathEscape(slug), "Token from live search"
	default:
		return prismsearch.Entity{}, false
	}
	return entity, true
}

func exactGatewayEntity(query string, entities []prismsearch.Entity) (prismsearch.Entity, bool) {
	query = strings.TrimSpace(query)
	for _, entity := range entities {
		if strings.EqualFold(query, entity.Name) || strings.EqualFold(query, entity.Value) {
			return entity, true
		}
	}
	return prismsearch.Entity{}, false
}

func gatewayEntityActionLabel(entity prismsearch.Entity, destination string) string {
	switch {
	case strings.Contains(destination, "/smart"):
		return "Open smart account"
	case strings.Contains(destination, "/tx/"):
		return "Open transaction"
	case strings.Contains(destination, "/ledger/"):
		return "Open ledger"
	case entity.Type == prismsearch.EntityContract:
		return "Open contract"
	case entity.Type == prismsearch.EntityAccount:
		return "Open account"
	case entity.Type == prismsearch.EntityAsset:
		return "Open asset"
	default:
		return "Open entity"
	}
}

func exploreFormQuery(r *http.Request) string {
	v := url.Values{}
	for _, key := range []string{"scope", "topic", "fn", "asset", "time", "status"} {
		value := strings.TrimSpace(r.FormValue(key))
		if value == "" {
			continue
		}
		if key == "scope" && value == "all" {
			continue
		}
		if key == "time" && value == "1h" {
			continue
		}
		v.Set(key, value)
	}
	return v.Encode()
}

func mergeExploreForm(r *http.Request, q *prismsearch.Query) {
	if v := strings.TrimSpace(r.FormValue("scope")); v != "" {
		q.Scope = v
	}
	if v := strings.TrimSpace(r.FormValue("topic")); v != "" {
		q.Topic = v
	}
	if v := strings.TrimSpace(r.FormValue("fn")); v != "" {
		q.Fn = v
	}
	if v := strings.TrimSpace(r.FormValue("asset")); v != "" {
		q.Asset = strings.ToUpper(v)
	}
	if v := strings.TrimSpace(r.FormValue("time")); v != "" {
		q.Time = v
	}
	if v := strings.TrimSpace(r.FormValue("status")); v != "" {
		q.Status = v
	}
}

func hxOrRedirect(w http.ResponseWriter, r *http.Request, target string) {
	if isHTMX(r) {
		w.Header().Set("HX-Redirect", target)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func questionIntentSummary(m intent.Match) string {
	switch m.ID {
	case intent.ExpiringContracts:
		return "Current contract TTL attention snapshot with evidence."
	case intent.ProtocolBusy:
		return "Known protocol activity using an explicit evidence window."
	case intent.TransactionFailure:
		return "Decoded receipt and semantic failure evidence for the supplied transaction."
	case intent.ContractActivity:
		return "Contract analytics with the returned evidence window disclosed."
	case intent.AssetActivity:
		return "Asset transfers, accounts, holders, and volume from the 24-hour record."
	case intent.RecentFailures:
		return "Recent decoded failures with collection coverage disclosed."
	default:
		return "Deterministic answer with linked evidence."
	}
}

func searchAnswerLabel(match intent.Match) string {
	switch match.ID {
	case intent.TransactionFailure:
		return "Answer transaction failure"
	case intent.ContractActivity:
		return "Answer contract activity"
	case intent.AssetActivity:
		return "Answer asset activity"
	case intent.RecentFailures:
		return "Answer recent failures"
	case intent.ExpiringContracts:
		return "Answer contract TTL question"
	case intent.ProtocolBusy:
		return "Answer protocol activity"
	default:
		return "Answer supported question"
	}
}

func searchOpenLabel(entity prismsearch.Classification, destination string) string {
	switch {
	case strings.Contains(destination, "/smart"):
		return "Open smart account"
	case strings.Contains(destination, "/assets/"):
		return "Open asset"
	case entity.Type == prismsearch.ClassTxHash:
		return "Open transaction"
	case entity.Type == prismsearch.ClassAccount:
		return "Open account"
	case entity.Type == prismsearch.ClassContract:
		return "Open contract"
	case entity.Type == prismsearch.ClassLedger:
		return "Open ledger"
	default:
		return "Open entity"
	}
}

func searchKindLabel(kind prismsearch.SearchResolutionKind) string {
	switch kind {
	case prismsearch.SearchOpen:
		return "Open"
	case prismsearch.SearchExplore:
		return "Explore"
	case prismsearch.SearchAnswer:
		return "Answer"
	default:
		return "Unsupported"
	}
}
